// Package weixin — 微信 ClawBot iLink 通道（多助手架构）
// 每个助手实例独立：Token + 人格 + 长轮询
package weixin

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/netclient"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// ─── 配置 ────────────────────────────────────────────────────

type Config struct {
	ILinkURL      string // iLink API 地址
	BotToken      string // ClawBot Token
	BotID         string // Bot 用户 ID
	AssistantID   string // 绑定的助手 ID
	PersonalityID string // 绑定的人格 ID
	// CapturePath 真机抓包文件路径（JSONL，v4.8.2 真机窗口）。空=默认
	// UserCacheDir/gaea/wx_capture.jsonl（UserCacheDir 不可用退 TempDir）。
	CapturePath string
}

func DefaultConfig() Config {
	return Config{ILinkURL: "https://ilinkai.weixin.qq.com"}
}

// ─── 入站防线参数（v4.8 子项 d）─────────────────────────────

const (
	wxRateLimit     = 20           // per-peer 滑动窗口内放行条数
	wxRateWindow    = time.Minute  // 滑动窗口长度
	wxMaxTextBytes  = 4096         // 入站文本字节上限（4KB）
	wxMaxMediaItems = 5            // item_list 多媒体条数上限
	rateLimitedText = "消息太频繁，稍后再说" // 超限固定文案（不触发 LLM）
	truncatedMark   = "（消息过长已截断）"
)

// ─── 回调 ────────────────────────────────────────────────────

type ChatFunc func(userMsg string, fromUser string) (reply string, err error)

// ─── Server ──────────────────────────────────────────────────

type Server struct {
	cfg     Config
	chatFn  ChatFunc
	client  *http.Client
	running atomic.Bool
	stopMu  sync.Mutex
	stopCh  chan struct{}

	syncBuf   string
	syncBufMu sync.Mutex
	pollTO    time.Duration
	pollCount int64

	sessionExpired atomic.Bool

	// 最近活跃会话（v4.4 主动推送）：handle 收到消息时记录发送者与上下文
	// token，Push 据此向「最近一次发来消息的会话」回推。个人小号单用户场景
	// 足够；多联系人场景由上层按 assistant 维度分 Server 隔离。
	lastPeerMu      sync.Mutex
	lastFromUser    string
	lastContextTokn string

	// OnSessionExpired 会话过期回调（getUpdates 返回 errcode=-14 sessExp 时触发一次）；
	// 由上层注入（如 app 层 emit 前端事件并提示重新扫码）。nil 时仅记录日志。
	OnSessionExpired func()

	// getUpdatesFn 可替换的 getUpdates 实现（测试注入，避免真实 HTTP；nil 时用默认实现）。
	getUpdatesFn func(req *pollReq, timeout time.Duration) (*pollResp, error)
	// sendFn 可替换的回复发送实现（测试注入，避免真实 HTTP；nil 时用默认 Send）。
	sendFn func(toUser, contextToken, text string) error
	// notifyStartFn / notifyStopFn 可替换的通知实现（测试注入，避免真实网络调用）。
	notifyStartFn func()
	notifyStopFn  func()

	// MediaRecognizer 可注入的图片识别实现（v4.8 子项 b，对齐 sendFn 注入
	// 模式）：入参 iLink 下发的图片 URL，出参识别文本；下载与临时文件清理
	// 由实现方负责（app 层用 weixin.OCRMediaRecognizer(a.GaeaOCRText) 一行
	// 注入）。nil=关闭，行为与现状完全一致（仅占位提示行）。
	MediaRecognizer func(url string) (string, error)

	// limiter 入站 per-peer 滑动窗口限频（v4.8 子项 d①）；测试经
	// limiter.clock 注入时钟。
	limiter *rateLimiter

	// capturePath 真机抓包文件路径（v4.8.2）：由 Server 持有并注册进包级
	// sink（capture.go），qrlogin 等包级函数据此抓包。
	capturePath string

	// 媒体域线索（v4.8.2 上传探针）：扫码登录响应的 baseurl/redirect_host。
	// SetMediaHosts 显式注入，或探针时从登录钩子的包级缓存懒采纳；空=未登录
	// （SendFileCard 直接走现有文本降级）。
	mediaMu           sync.Mutex
	mediaBaseURL      string
	mediaRedirectHost string
}

func New(cfg Config, chatFn ChatFunc) *Server {
	if cfg.ILinkURL == "" {
		cfg.ILinkURL = "https://ilinkai.weixin.qq.com"
	}
	s := &Server{
		cfg:     cfg,
		chatFn:  chatFn,
		client:  netclient.NewSimpleClient(90 * time.Second),
		stopCh:  make(chan struct{}),
		pollTO:  30 * time.Second,
		limiter: newRateLimiter(wxRateLimit, wxRateWindow),
	}
	// 真机抓包路径（v4.8.2）：Server 持有并注册进包级 sink（capture.go），
	// app 层未配置时取默认路径（UserCacheDir/TempDir 兜底）。
	s.capturePath = cfg.CapturePath
	if s.capturePath == "" {
		s.capturePath = defaultCapturePath()
	}
	setCapturePath(s.capturePath)
	return s
}

// ─── 生命周期 ────────────────────────────────────────────────

func (s *Server) Start() error {
	if s.cfg.BotToken == "" {
		return fmt.Errorf("未配置 BotToken")
	}
	// 已在运行：幂等返回（不重复拉起第二个 pollLoop，也不动 stopCh）
	if s.running.Swap(true) {
		return nil
	}
	// 重启支持：Stop 已 close 并置空 stopCh，这里重建新通道；
	// 同时清掉会话过期标记，保证 Stop→Start 后轮询真正恢复（而非空转）
	s.stopMu.Lock()
	if s.stopCh == nil {
		s.stopCh = make(chan struct{})
	}
	s.stopMu.Unlock()
	s.sessionExpired.Store(false)
	s.notifyStart()
	go s.pollLoop()
	slog.Info("[weixin] 助手通道启动",
		"assistant", s.cfg.AssistantID,
		"personality", s.cfg.PersonalityID,
	)
	return nil
}

// Stop 幂等停止：仅第一次真正 close(stopCh) 并通知，之后调用无副作用（不 panic）。
func (s *Server) Stop() {
	s.running.Store(false)
	s.stopMu.Lock()
	if s.stopCh == nil {
		s.stopMu.Unlock()
		return // 已停止过（或从未启动），幂等返回
	}
	close(s.stopCh)
	s.stopCh = nil
	s.stopMu.Unlock()
	s.notifyStop()
	slog.Info("[weixin] 助手通道关闭", "assistant", s.cfg.AssistantID)
}

func (s *Server) IsRunning() bool { return s.running.Load() }

// SessionExpired 会话是否过期（getUpdates 返回 sessExp -14，需重新绑定）
func (s *Server) SessionExpired() bool { return s.sessionExpired.Load() }
func (s *Server) HasILink() bool       { return s.cfg.BotToken != "" }
func (s *Server) AssistantID() string  { return s.cfg.AssistantID }

// ─── 长轮询 ──────────────────────────────────────────────────

const (
	maxFail = 5
	backoff = 30 * time.Second
	retry   = 3 * time.Second
	sessExp = -14
)

type pollReq struct {
	GetUpdatesBuf string      `json:"get_updates_buf"`
	BaseInfo      interface{} `json:"base_info"`
}

type pollResp struct {
	Ret                  int          `json:"ret"`
	ErrCode              int          `json:"errcode"`
	ErrMsg               string       `json:"errmsg"`
	Msgs                 []inboundMsg `json:"msgs"`
	GetUpdatesBuf        string       `json:"get_updates_buf"`
	SyncBuf              string       `json:"sync_buf"`
	LongPollingTimeoutMs int          `json:"longpolling_timeout_ms"`
}

type inboundMsg struct {
	FromUserID   string `json:"from_user_id"`
	ToUserID     string `json:"to_user_id"`
	ClientID     string `json:"client_id"`
	ContextToken string `json:"context_token"`
	ItemList     []struct {
		Type      int        `json:"type"`
		TextItem  *textItem  `json:"text_item,omitempty"`
		ImageItem *imageItem `json:"image_item,omitempty"`
		FileItem  *fileItem  `json:"file_item,omitempty"`
	} `json:"item_list"`
}

type textItem struct {
	Text string `json:"text"`
}

// imageItem / fileItem 是 iLink 非文本消息项（S4.5「发图即识别」协议探明第一
// 刀）：字段名按 iLink 惯例留位（file_id/url/name/md5/size），真实负载以服务端
// 下发为准——解析是防御性的，未知字段不报错。拿到 URL/file_id 后可接 vision/
// 文件下载管线做字节级识别。协议探明度与多态风险见 docs/ilink-non-text-protocol.md。
type imageItem struct {
	FileID string `json:"file_id,omitempty"`
	URL    string `json:"url,omitempty"`
	Name   string `json:"name,omitempty"`
	MD5    string `json:"md5,omitempty"`
}

type fileItem struct {
	FileID string `json:"file_id,omitempty"`
	URL    string `json:"url,omitempty"`
	Name   string `json:"name,omitempty"`
	Size   int64  `json:"size,omitempty"`
}

// ─── 防御解析（v4.8 子项 a）─────────────────────────────────
//
// iLink 非文本负载未真机定稿，字段可能多态（url=数组/对象、file_id=数字…）。
// 单一字段的怪异形态绝不能让整条消息——乃至整个长轮询批次——解析失败
// （失败会让同一 sync_buf 反复重试）。两个 UnmarshalJSON 把不认识的形态
// 降级为零值：宁漏勿误，消息照常进 handle。

// coerceString 把多态 JSON 标量降级为字符串：string 原样、数字去尾零格式化、
// 布尔转字面量；数组/对象/null 一律丢弃（返回空串，宁漏勿误）。
func coerceString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return "" // array / object / nil：防御性丢弃
	}
}

// coerceInt64 容忍 size 以数字或数字字符串下发；其余形态降级为 0。
func coerceInt64(v interface{}) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		return 0
	}
}

// isJSONObject 报告原始 JSON 是否为对象形态；非对象（null/数组/标量）按空
// 负载降级，不让 item 级多态炸掉整条消息。
func isJSONObject(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) > 0 && trimmed[0] == '{'
}

// UnmarshalJSON image_item 防御解析：url/file_id/md5 允许多态，name 超长、
// emoji/中文不报错；整体非对象时按空负载处理。
func (it *imageItem) UnmarshalJSON(data []byte) error {
	if !isJSONObject(data) {
		return nil
	}
	var raw struct {
		FileID interface{} `json:"file_id"`
		URL    interface{} `json:"url"`
		Name   string      `json:"name"`
		MD5    interface{} `json:"md5"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	it.FileID = coerceString(raw.FileID)
	it.URL = coerceString(raw.URL)
	it.Name = raw.Name
	it.MD5 = coerceString(raw.MD5)
	return nil
}

// UnmarshalJSON file_item 防御解析：file_id/url 多态容忍，size 数字/字符串
// 皆可；整体非对象时按空负载处理。
func (it *fileItem) UnmarshalJSON(data []byte) error {
	if !isJSONObject(data) {
		return nil
	}
	var raw struct {
		FileID interface{} `json:"file_id"`
		URL    interface{} `json:"url"`
		Name   string      `json:"name"`
		Size   interface{} `json:"size"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	it.FileID = coerceString(raw.FileID)
	it.URL = coerceString(raw.URL)
	it.Name = raw.Name
	it.Size = coerceInt64(raw.Size)
	return nil
}

func (s *Server) pollLoop() {
	var fails int
	timeout := s.pollTO

	// 测试注入：getUpdatesFn 非 nil 时替换默认 HTTP 实现（生产保持默认）
	getUpdates := s.getUpdates
	if s.getUpdatesFn != nil {
		getUpdates = s.getUpdatesFn
	}

	for s.running.Load() {
		req := pollReq{BaseInfo: s.baseInfo()}
		s.syncBufMu.Lock()
		req.GetUpdatesBuf = s.syncBuf
		s.syncBufMu.Unlock()

		resp, err := getUpdates(&req, timeout)
		if err != nil {
			fails++
			slog.Error("[weixin] getUpdates 失败", "assistant", s.cfg.AssistantID, "err", err)
			if fails >= maxFail {
				s.sleepOrStop(backoff)
				fails = 0
			} else {
				s.sleepOrStop(retry)
			}
			continue
		}

		if resp.LongPollingTimeoutMs > 0 {
			timeout = time.Duration(resp.LongPollingTimeoutMs) * time.Millisecond
		}

		ec := resp.ErrCode
		if ec == 0 {
			ec = resp.Ret
		}
		if ec != 0 {
			if ec == sessExp {
				s.sessionExpired.Store(true)
				slog.Warn("[weixin] 会话过期（token 无效或需重新绑定）", "assistant", s.cfg.AssistantID)
				// 会话失效自愈：触发上层回调（app 层 emit 前端事件提示重新扫码），
				// 然后退出轮询，等待上层 Stop→Start 重启——不再 5 分钟空转
				if s.OnSessionExpired != nil {
					s.OnSessionExpired()
				}
				return
			}
			fails++
			s.sleepOrStop(retry)
			continue
		}

		s.sessionExpired.Store(false)
		fails = 0

		if buf := resp.GetUpdatesBuf; buf != "" {
			s.syncBufMu.Lock()
			s.syncBuf = buf
			s.syncBufMu.Unlock()
		} else if buf := resp.SyncBuf; buf != "" {
			s.syncBufMu.Lock()
			s.syncBuf = buf
			s.syncBufMu.Unlock()
		}

		// 活跃诊断：每 10 次成功轮询记录一次（确认长轮询在跑）
		s.pollCount++
		if s.pollCount%10 == 0 {
			s.syncBufMu.Lock()
			bufLen := len(s.syncBuf)
			s.syncBufMu.Unlock()
			slog.Info("[weixin] 轮询活跃", "assistant", s.cfg.AssistantID, "round", s.pollCount, "syncBufLen", bufLen)
		}

		for i := range resp.Msgs {
			s.handle(&resp.Msgs[i])
		}
	}
}

func (s *Server) handle(msg *inboundMsg) {
	// v4.8 d① per-peer 滑动窗口限频：超限发固定文案，不触发 LLM（不进聊天
	// 管道，也不更新 lastPeer——正常消息已记录过回推目标）。
	if msg.FromUserID != "" && !s.limiter.Allow(msg.FromUserID) {
		slog.Warn("[weixin] 消息超过频率限制，已按固定文案回复",
			"assistant", s.cfg.AssistantID, "from", msg.FromUserID)
		if s.sendFn != nil {
			_ = s.sendFn(msg.FromUserID, msg.ContextToken, rateLimitedText)
		} else {
			_ = s.Send(msg.FromUserID, msg.ContextToken, rateLimitedText)
		}
		return
	}

	text := ""
	var media []string      // 未识别多媒体的提示行（统一走「内容暂无法读取」包装）
	var recognized []string // 已识别图片的提示行（自带括号包装）
	mediaCount := 0
	mediaOverflow := 0
	for _, item := range msg.ItemList {
		switch {
		case item.Type == 1 && item.TextItem != nil:
			text += item.TextItem.Text
		case item.ImageItem != nil:
			mediaCount++
			if mediaCount > wxMaxMediaItems { // d③ 条数上限：超出不逐个处理
				mediaOverflow++
				continue
			}
			if s.MediaRecognizer != nil && item.ImageItem.URL != "" {
				if desc, ok := s.recognizeImage(*item.ImageItem); ok {
					recognized = append(recognized, recognizedImageLabel(*item.ImageItem, desc))
					continue
				}
			}
			media = append(media, imageItemLabel(*item.ImageItem))
		case item.FileItem != nil:
			mediaCount++
			if mediaCount > wxMaxMediaItems {
				mediaOverflow++
				continue
			}
			media = append(media, fileItemLabel(*item.FileItem))
		default:
			// 未知类型且无已识别负载：宁漏勿误——静默跳过（协议字段待探明，
			// 不把无法理解的项喂给模型）。
		}
	}
	if mediaOverflow > 0 {
		media = append(media, fmt.Sprintf("…等 %d 个文件", mediaOverflow))
	}
	// S4.5 发图即识别 / v4.8 子项 b：非文本消息转成模型可见的提示行。已识别
	// 图片逐条给出「（用户发来图片「name」，识别内容：…）」；未识别项保持
	// 原状（统一包装「内容暂无法读取」）。
	if len(recognized) > 0 || len(media) > 0 {
		var parts []string
		parts = append(parts, recognized...)
		if len(media) > 0 {
			parts = append(parts, "（用户发来一条"+strings.Join(media, "、")+"，内容暂无法读取）")
		}
		hint := strings.Join(parts, "")
		if text == "" {
			text = hint
		} else {
			text = hint + " 附言：" + text
		}
	}
	// v4.8 d② 入站文本 4KB 截断（rune 安全），防止超长消息喂爆模型。
	text = truncateTextBytes(text, wxMaxTextBytes)
	if text == "" || s.chatFn == nil {
		return
	}

	// 记录最近活跃会话（主动推送 Push 的回推目标）。
	s.lastPeerMu.Lock()
	s.lastFromUser = msg.FromUserID
	s.lastContextTokn = msg.ContextToken
	s.lastPeerMu.Unlock()

	slog.Info("[weixin] 收到消息", "assistant", s.cfg.AssistantID, "from", msg.FromUserID, "len", len(text))
	reply, err := s.chatFn(text, msg.FromUserID)
	if err != nil {
		slog.Error("[weixin] AI回复失败", "err", err)
		reply = "思考中…请稍后再试"
	}
	if s.sendFn != nil {
		err = s.sendFn(msg.FromUserID, msg.ContextToken, reply)
	} else {
		err = s.Send(msg.FromUserID, msg.ContextToken, reply)
	}
	if err != nil {
		slog.Error("[weixin] 回复失败", "err", err)
	}
}

// imageItemLabel / fileItemLabel 把非文本消息项转成模型可见的简短描述。
func imageItemLabel(it imageItem) string {
	if it.Name != "" {
		return "图片消息（" + it.Name + "）"
	}
	return "图片消息"
}

func fileItemLabel(it fileItem) string {
	if it.Name != "" {
		return "文件消息（" + it.Name + "）"
	}
	return "文件消息"
}

// ─── 图片识别管线（v4.8 子项 b）─────────────────────────────

// recognizeImage 调用注入的 MediaRecognizer 识别图片。前后各一条 slog 便于
// 真机诊断；失败或空结果一律返回 false（handle 保留原占位提示行），错误
// 不上抛不 panic。
func (s *Server) recognizeImage(it imageItem) (string, bool) {
	slog.Info("[weixin] 图片识别开始",
		"assistant", s.cfg.AssistantID,
		"url", truncateRunes(it.URL, 120),
		"name", it.Name)
	desc, err := s.MediaRecognizer(it.URL)
	if err != nil {
		slog.Warn("[weixin] 图片识别失败，保留占位提示",
			"assistant", s.cfg.AssistantID, "url", truncateRunes(it.URL, 120), "err", err)
		return "", false
	}
	desc = strings.TrimSpace(desc)
	if desc == "" {
		slog.Warn("[weixin] 图片识别结果为空，保留占位提示",
			"assistant", s.cfg.AssistantID, "url", truncateRunes(it.URL, 120))
		return "", false
	}
	slog.Info("[weixin] 图片识别完成",
		"assistant", s.cfg.AssistantID, "runes", len([]rune(desc)))
	return desc, true
}

// recognizedImageLabel 已识别图片的提示行：识别内容截前 300 rune（超长加省略号）。
func recognizedImageLabel(it imageItem, desc string) string {
	content := truncateRunes(desc, 300)
	if it.Name != "" {
		return "（用户发来图片「" + it.Name + "」，识别内容：" + content + "）"
	}
	return "（用户发来图片，识别内容：" + content + "）"
}

// truncateRunes 按 rune 数截断，超长追加省略号。
func truncateRunes(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n]) + "…"
}

// truncateTextBytes 按字节上限截断文本（不撕裂 UTF-8 多字节序列），截断时
// 追加固定标记，让模型与用户都知道内容不完整。
func truncateTextBytes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1] // 回退到 rune 边界（最多回退 3 字节）
	}
	return cut + truncatedMark
}

// LastPeer 返回最近活跃会话（fromUser, contextToken）；无记录时返回空串。
func (s *Server) LastPeer() (string, string) {
	s.lastPeerMu.Lock()
	defer s.lastPeerMu.Unlock()
	return s.lastFromUser, s.lastContextTokn
}

// Push 主动向最近活跃会话发送文本（v4.4 离线代办回推通路）。
// 尚无活跃会话（启动后没人发过消息）时报错，由上层决定重试或标记失败。
func (s *Server) Push(text string) error {
	from, ctx := s.LastPeer()
	if from == "" {
		return fmt.Errorf("无活跃微信会话（尚未收到任何消息），无法主动推送")
	}
	return s.Send(from, ctx, text)
}

// SendFileCard 产物回推 seam（v4.8.2 探针版）：iLink 上传端点仍未探明，本版
// 在保留现有文本降级（逐字节不变）的前提下做「尽力上传探测」——以扫码登录
// 响应的 baseurl/redirect_host（docs/ilink-non-text-protocol.md §2 唯一线索）
// 为基地址，按推断端点序列逐个 multipart 试传（每个失败抓一行 upload_probe）；
// 任一端点返回成功 errcode 且给出 url/file_id 即组 image_item 图片卡片，全部
// 失败/未登录/文件不合规/任何 panic 一律降级文本卡片——产物回推绝不崩主流程。
// 端点序列是「基于惯例的推断」，以真机抓包为准。
func (s *Server) SendFileCard(localPath, caption string) (err error) {
	sent := false
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[weixin] 产物回推 panic，降级文本卡片",
				"assistant", s.cfg.AssistantID, "recover", r)
			if sent {
				err = nil // 图片已发出：不再重复降级推送
				return
			}
			err = s.sendFileCardText(localPath, caption)
		}
	}()
	return s.sendFileCardProbed(localPath, caption, &sent)
}

// ─── 产物上传探针（v4.8.2：端点为推断，以真机抓包为准）────────────────────

// wxProbeTimeout 探针超时：宁可早失败降级文本，不拖垮产物回推。
const wxProbeTimeout = 15 * time.Second

// uploadImageExts 图片产物扩展名白名单（图片产物场景，对齐 media_download 的
// png/jpeg/webp/gif 口径）。
var uploadImageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true,
}

// probeResult 探针响应的防御解析：errcode/ret 用指针区分「缺失」与「0」
// （缺 errcode 的错误响应不得误判成功）；url/file_id 容忍顶层或 data 内嵌。
// 真实响应形态以抓包为准，此处只按最常见惯例留位。
type probeResult struct {
	ErrCode *int   `json:"errcode"`
	Ret     *int   `json:"ret"`
	URL     string `json:"url"`
	FileID  string `json:"file_id"`
	Data    struct {
		URL    string `json:"url"`
		FileID string `json:"file_id"`
	} `json:"data"`
}

func parseProbeResult(body []byte) probeResult {
	var pr probeResult
	_ = json.Unmarshal(body, &pr) // 非 JSON 响应：零值，按失败处理
	return pr
}

func (pr probeResult) ok() bool {
	return (pr.ErrCode != nil && *pr.ErrCode == 0) || (pr.Ret != nil && *pr.Ret == 0)
}

// errCodeVal 抓包展示用：errcode 优先，缺失时回退 ret（可能为 0=缺失，以
// resp 原文与 ok 标志为准）。
func (pr probeResult) errCodeVal() int {
	if pr.ErrCode != nil {
		return *pr.ErrCode
	}
	if pr.Ret != nil {
		return *pr.Ret
	}
	return 0
}

func (pr probeResult) mediaRef() (url, fileID string) {
	url, fileID = pr.URL, pr.FileID
	if url == "" {
		url = pr.Data.URL
	}
	if fileID == "" {
		fileID = pr.Data.FileID
	}
	return url, fileID
}

// SetMediaHosts 显式注入媒体域线索（baseurl/redirect_host）：app 层扫码登录
// 成功后的接线点（一行）；测试亦由此注入。空串=清除（回退文本降级）。
func (s *Server) SetMediaHosts(baseURL, redirectHost string) {
	s.mediaMu.Lock()
	s.mediaBaseURL, s.mediaRedirectHost = baseURL, redirectHost
	s.mediaMu.Unlock()
}

// probeBase 上传探针基地址：优先 Server 字段（SetMediaHosts 注入），为空时
// 懒采纳扫码登录钩子的包级缓存（Server 创建可能早于登录）。结果经
// normalizeMediaBase 补全为可请求的 http(s) 基地址。
func (s *Server) probeBase() string {
	s.mediaMu.Lock()
	base := s.mediaBaseURL
	if base == "" {
		var redirect string
		base, redirect = latestMediaHosts()
		s.mediaBaseURL, s.mediaRedirectHost = base, redirect
	}
	s.mediaMu.Unlock()
	return normalizeMediaBase(base)
}

// normalizeMediaBase 把登录响应里的 baseurl 补全成 http(s) 基地址（字段真实
// 形态待真机抓包：可能是裸域、带 scheme 或带尾斜杠路径）。探针目标来自微信
// 登录响应（与 cfg.ILinkURL 同级信任），不套 media_download 的 SSRF 防线。
func normalizeMediaBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	if !strings.Contains(base, "://") {
		base = "https://" + base
	}
	return strings.TrimRight(base, "/")
}

// sendFileCardText 现有文本降级卡片（v4.8 子项 c 原实现逐字节保留）：无媒体
// 域线索、探针全败、文件不合规、panic 恢复时的最终退路。
func (s *Server) sendFileCardText(localPath, caption string) error {
	name := filepath.Base(localPath)
	text := "🖼 产物已生成：" + name + "（微信暂不支持直接收文件卡片，请在桌面端「书房·绘梦」查看）"
	if caption != "" {
		text += "\n" + caption
	}
	return s.Push(text)
}

// sendFileCardProbed 探针主流程：未登录→抓 skipped 后文本降级；文件不合规→
// 同；否则按推断端点序列逐个试传。任何成功路径经 sent 告知外层（panic 恢复
// 时不再重复推送）。
func (s *Server) sendFileCardProbed(localPath, caption string, sent *bool) error {
	name := filepath.Base(localPath)
	base := s.probeBase()
	if base == "" {
		// 未登录或登录响应未带媒体域：抓一行跳过记录后走现有文本降级。
		capture("upload_probe", map[string]interface{}{"skipped": "no_media_host", "file": name})
		return s.sendFileCardText(localPath, caption)
	}
	data, err := readUploadableFile(localPath)
	if err != nil {
		slog.Warn("[weixin] 产物文件不可上传，降级文本卡片",
			"assistant", s.cfg.AssistantID, "file", name, "err", err)
		capture("upload_probe", map[string]interface{}{"skipped": "file_rejected", "file": name, "err": err.Error()})
		return s.sendFileCardText(localPath, caption)
	}

	// 推断端点序列（**以真机抓包为准**）：a) uploadfile（multipart file +
	// base_info 字段）；b) sendmessage multipart 直发（file + 路由/caption
	// 字段）；c) upload（最简形态）。鉴权头与 apiPost 同形态。
	baseInfoJSON, _ := json.Marshal(s.baseInfo())
	probes := []struct {
		endpoint string
		fields   map[string]string
	}{
		{endpoint: "/ilink/bot/uploadfile", fields: map[string]string{"base_info": string(baseInfoJSON)}},
		{endpoint: "/ilink/bot/sendmessage", fields: s.probeSendFields(caption)},
		{endpoint: "/ilink/bot/upload"},
	}
	for _, p := range probes {
		body, status, perr := s.probeUpload(base+p.endpoint, name, data, p.fields)
		pr := parseProbeResult(body)
		ok := perr == nil && pr.ok()
		rec := map[string]interface{}{
			"endpoint": p.endpoint,
			"status":   status,
			"ok":       ok,
			"errcode":  pr.errCodeVal(),
			"resp":     string(body),
		}
		if perr != nil {
			rec["err"] = perr.Error()
		}
		capture("upload_probe", rec)
		if !ok {
			continue
		}
		// sendmessage multipart 直发成功即视为图片（含 caption）已送达，不再补发。
		if p.endpoint == "/ilink/bot/sendmessage" {
			*sent = true
			return nil
		}
		url, fileID := pr.mediaRef()
		if url == "" && fileID == "" {
			continue // 成功码但未返回媒体句柄：试下一端点
		}
		if err := s.sendImageCard(url, fileID, name, caption); err != nil {
			capture("upload_probe", map[string]interface{}{
				"stage": "send_image_card", "url": url, "file_id": fileID, "err": err.Error(),
			})
			continue
		}
		*sent = true
		return nil
	}
	return s.sendFileCardText(localPath, caption)
}

// probeSendFields sendmessage multipart 直发的路由与文案字段（file 字段由
// probeUpload 统一附加）。
func (s *Server) probeSendFields(caption string) map[string]string {
	from, ctxTokn := s.LastPeer()
	f := map[string]string{"text": caption}
	if from != "" {
		f["to_user_id"] = from
	}
	if ctxTokn != "" {
		f["context_token"] = ctxTokn
	}
	return f
}

// probeUpload 对推断端点发一次 multipart 上传探测（15s 超时；鉴权头同
// apiPost 形态）。返回响应体（≤64KB）与 HTTP 状态码；传输层错误一并返回
// 已读 body（通常为空），由调用方统一抓包后试下一端点。
func (s *Server) probeUpload(rawURL, filename string, data []byte, fields map[string]string) ([]byte, int, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err == nil {
		_, err = fw.Write(data)
	}
	if err == nil {
		for k, v := range fields {
			if err = mw.WriteField(k, v); err != nil {
				break
			}
		}
	}
	if err == nil {
		err = mw.Close()
	}
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequest("POST", rawURL, &buf)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	s.setAuthHeaders(req)

	c := s.client
	if wxProbeTimeout < c.Timeout {
		c = netclient.NewSimpleClient(wxProbeTimeout)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != 200 {
		return b, resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return b, resp.StatusCode, nil
}

// sendImageCard 组 image_item 形态的 sendmessage 发送图片卡片（字段名参照
// 入站 image_item：url/file_id/name），caption 作为后续 text item。失败原样
// 返回错误，由探针循环抓包后继续下一端点。
func (s *Server) sendImageCard(imageURL, fileID, name, caption string) error {
	ii := map[string]string{"name": name}
	if imageURL != "" {
		ii["url"] = imageURL
	}
	if fileID != "" {
		ii["file_id"] = fileID
	}
	items := []map[string]interface{}{{"type": 3, "image_item": ii}}
	if caption != "" {
		items = append(items, map[string]interface{}{"type": 1, "text_item": map[string]string{"text": caption}})
	}
	from, ctxTokn := s.LastPeer()
	msg := map[string]interface{}{
		"from_user_id":  "",
		"to_user_id":    from,
		"client_id":     genClientID(),
		"message_type":  2,
		"message_state": 2,
		"item_list":     items,
	}
	if ctxTokn != "" {
		msg["context_token"] = ctxTokn
	}
	body, _ := json.Marshal(map[string]interface{}{"msg": msg, "base_info": s.baseInfo()})
	_, err := s.apiPost("/ilink/bot/sendmessage", body, 20*time.Second)
	return err
}

// readUploadableFile 读取待上传产物：扩展名白名单 + 20MiB 上限（沿用
// wxImageMaxBytes 口径）。不合规返回错误（探针层抓包后降级文本卡片）。
func readUploadableFile(path string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if !uploadImageExts[ext] {
		return nil, fmt.Errorf("扩展名 %q 不在图片白名单（png/jpeg/webp/gif）", ext)
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Size() > wxImageMaxBytes {
		return nil, fmt.Errorf("文件 %d 字节超过上传上限 %d", st.Size(), wxImageMaxBytes)
	}
	return os.ReadFile(path)
}

// ─── 发送 ────────────────────────────────────────────────────

func (s *Server) Send(toUser, contextToken, text string) error {
	msg := map[string]interface{}{
		"from_user_id":  "",
		"to_user_id":    toUser,
		"client_id":     genClientID(),
		"message_type":  2,
		"message_state": 2,
		"item_list": []map[string]interface{}{
			{"type": 1, "text_item": map[string]string{"text": text}},
		},
	}
	if contextToken != "" {
		msg["context_token"] = contextToken
	}
	body, _ := json.Marshal(map[string]interface{}{
		"msg":       msg,
		"base_info": s.baseInfo(),
	})

	_, err := s.apiPost("/ilink/bot/sendmessage", body, 20*time.Second)
	return err
}

// ─── API ─────────────────────────────────────────────────────

// batchHasMedia 报告批次里是否出现非纯文本 item（image_item/file_item 负载
// 或未知 type）——真机探明第一手数据的触发条件；纯文本批次不抓，避免刷屏。
func batchHasMedia(msgs []inboundMsg) bool {
	for i := range msgs {
		for _, it := range msgs[i].ItemList {
			if it.ImageItem != nil || it.FileItem != nil || it.Type != 1 {
				return true
			}
		}
	}
	return false
}

func (s *Server) getUpdates(req *pollReq, timeout time.Duration) (*pollResp, error) {
	body, _ := json.Marshal(req)
	respBody, err := s.apiPost("/ilink/bot/getupdates", body, timeout)
	if err != nil {
		return nil, err
	}
	var resp pollResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	// v4.8.2 真机抓包窗口：批次含 image_item/file_item/未知 type 时抓整批
	// 原始 JSON（kind=inbound_media）。钩子放 getUpdates 本地——respBody 原始
	// 字节仍在作用域内，apiPost/getUpdates 签名零改动（最小侵入）。
	if batchHasMedia(resp.Msgs) {
		capture("inbound_media", json.RawMessage(respBody))
	}
	return &resp, nil
}

func (s *Server) notifyStart() {
	if s.notifyStartFn != nil {
		s.notifyStartFn()
		return
	}
	body, _ := json.Marshal(map[string]interface{}{"base_info": s.baseInfo()})
	s.apiPost("/ilink/bot/msg/notifystart", body, 10*time.Second)
}

func (s *Server) notifyStop() {
	if s.notifyStopFn != nil {
		s.notifyStopFn()
		return
	}
	body, _ := json.Marshal(map[string]interface{}{"base_info": s.baseInfo()})
	s.apiPost("/ilink/bot/msg/notifystop", body, 10*time.Second)
}

func (s *Server) apiPost(endpoint string, body []byte, timeout time.Duration) ([]byte, error) {
	req, _ := http.NewRequest("POST", s.cfg.ILinkURL+endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.setAuthHeaders(req)

	c := s.client
	if timeout < c.Timeout {
		c = netclient.NewSimpleClient(timeout)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return b, nil
}

// setAuthHeaders 设置与 apiPost 相同的鉴权头形态（Bearer BotToken + iLink
// 通道头）；上传探针 multipart 复用，保证「鉴权形态同 apiPost」。
func (s *Server) setAuthHeaders(req *http.Request) {
	req.Header.Set("AuthorizationType", "ilink_bot_token")
	req.Header.Set("Authorization", "Bearer "+s.cfg.BotToken)
	req.Header.Set("X-WECHAT-UIN", randomUIN())
	req.Header.Set("iLink-App-Id", "bot")
	req.Header.Set("iLink-App-ClientVersion", "132099") // buildClientVersion("2.4.3") = 0x020403
}

func (s *Server) baseInfo() map[string]string {
	return map[string]string{"channel_version": "2.4.3", "bot_agent": "gaea-desktop/1.0.0"}
}

// sleepOrStop 在 stopCh 关闭或超时二者间等待。stopCh 受 stopMu 保护：
// Stop 关闭后置 nil（幂等），重启时重建；这里在锁内取快照，避免读到
// 正在被替换的通道（nil 通道会让 select 走 time.After 分支，同样可被
// 外层 running=false 收尾，不会空转死等）。
func (s *Server) sleepOrStop(d time.Duration) {
	s.stopMu.Lock()
	ch := s.stopCh
	s.stopMu.Unlock()
	select {
	case <-ch:
	case <-time.After(d):
	}
}

// ─── 工具 ────────────────────────────────────────────────────

// genClientID 对齐腾讯官方 openclaw-weixin："{prefix}:{ts}-{hex}" 格式
func genClientID() string {
	ts := time.Now().UnixMilli()
	hex := randomHex(4)
	return fmt.Sprintf("gaea-weixin:%d-%s", ts, hex)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func randomUIN() string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%010d", time.Now().UnixNano()%10000000000)))
}
