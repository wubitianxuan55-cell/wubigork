// Package weixin — 微信 ClawBot iLink 通道（多助手架构）
// 每个助手实例独立：Token + 人格 + 长轮询
package weixin

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
	"github.com/gaea/gaea/internal/netclient"
)

// ─── 配置 ────────────────────────────────────────────────────

type Config struct {
	ILinkURL      string // iLink API 地址
	BotToken      string // ClawBot Token
	BotID         string // Bot 用户 ID
	AssistantID   string // 绑定的助手 ID
	PersonalityID string // 绑定的人格 ID
}

func DefaultConfig() Config {
	return Config{ILinkURL: "https://ilinkai.weixin.qq.com"}
}

// ─── 入站防线参数（v4.8 子项 d）─────────────────────────────

const (
	wxRateLimit     = 20               // per-peer 滑动窗口内放行条数
	wxRateWindow    = time.Minute      // 滑动窗口长度
	wxMaxTextBytes  = 4096             // 入站文本字节上限（4KB）
	wxMaxMediaItems = 5                // item_list 多媒体条数上限
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
}

func New(cfg Config, chatFn ChatFunc) *Server {
	if cfg.ILinkURL == "" {
		cfg.ILinkURL = "https://ilinkai.weixin.qq.com"
	}
	return &Server{
		cfg:     cfg,
		chatFn:  chatFn,
		client:  netclient.NewSimpleClient(90 * time.Second),
		stopCh:  make(chan struct{}),
		pollTO:  30 * time.Second,
		limiter: newRateLimiter(wxRateLimit, wxRateWindow),
	}
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

func (s *Server) IsRunning() bool  { return s.running.Load() }

// SessionExpired 会话是否过期（getUpdates 返回 sessExp -14，需重新绑定）
func (s *Server) SessionExpired() bool { return s.sessionExpired.Load() }
func (s *Server) HasILink() bool   { return s.cfg.BotToken != "" }
func (s *Server) AssistantID() string { return s.cfg.AssistantID }

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
		if ec == 0 { ec = resp.Ret }
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

// SendFileCard 产物回推 seam（v4.8 子项 c，文本降级版）：iLink 文件上传端点
// 尚未探明，当前以文本卡片告知产物名称与去向，经 Push 发往最近活跃会话。
// 真上传端点探明后仅需替换本方法实现（上传文件卡片 + caption），调用方不变；
// 约定详见 docs/ilink-non-text-protocol.md「产物回推 seam」一节。
func (s *Server) SendFileCard(localPath, caption string) error {
	name := filepath.Base(localPath)
	text := "🖼 产物已生成：" + name + "（微信暂不支持直接收文件卡片，请在桌面端「书房·绘梦」查看）"
	if caption != "" {
		text += "\n" + caption
	}
	return s.Push(text)
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
	req.Header.Set("AuthorizationType", "ilink_bot_token")
	req.Header.Set("Authorization", "Bearer "+s.cfg.BotToken)
	req.Header.Set("X-WECHAT-UIN", randomUIN())
	req.Header.Set("iLink-App-Id", "bot")
	req.Header.Set("iLink-App-ClientVersion", "132099") // buildClientVersion("2.4.3") = 0x020403

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
