// Package weixin — 微信 ClawBot iLink 通道（多助手架构）
// 每个助手实例独立：Token + 人格 + 长轮询
package weixin

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
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

// InboundFileHandler 入站文件处理器（v4.9 跨线契约，app 线注入，签名逐字锁定）：
// 收到入站文件并下载解密落盘后，以 (本地路径, 文件名, 字节数, 明文 MD5) 同步
// 调用；返回值作为注入文本行原样拼入喂给模型的消息（建议自带括号包装，对齐
// recognizedImageLabel 风格）。约定：
//   - 临时文件在处理器返回后即被删除（gaea-wxfile-*.tmp）——如需留存字节请
//     自行复制（同 OnInboundImage 的「钩子内部复制自持」语义）；
//   - 返回空串（或 panic，由调用方 recover）→ 回退占位提示行（宁漏勿误）；
//   - 注入与文本提取（docx/xlsx/pptx/pdf… → 文本）由 app 线负责，本包只定义
//     与调用。
type InboundFileHandler func(localPath, fileName string, sizeBytes int64, md5sum string) string

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

	// OnInboundImage 入站图片旁路钩子（v4.9 对话式改图，参照 MediaRecognizer
	// 注入模式）：识别管线成功拿到解密落盘路径后以 (发送者, 本地路径) 旁路
	// 调用——app 层注入「复制进改图自持缓存」。仅旁路：OCR 识别主流程一行不
	// 改；明文 URL（无本地落盘文件）不触发；钩子 panic 由 invokeInboundImage
	// Hook recover，绝不打断识别主流程。
	OnInboundImage func(fromUser, localPath string)

	// FileHandler 入站文件处理器（v4.9 跨线契约，InboundFileHandler 类型）：
	// app 线注入「文件 → 文本提取」实现后，入站 file_item 走「下载解密落盘 →
	// 提取 → 注入文本行」全链路；nil=关闭，行为与现状完全一致（占位提示行，
	// 且不发起下载零流量）。panic 由 invokeFileHandler recover，绝不打断消息
	// 主流程。
	FileHandler InboundFileHandler

	// limiter 入站 per-peer 滑动窗口限频（v4.8 子项 d①）；测试经
	// limiter.clock 注入时钟。
	limiter *rateLimiter

	// capturePath 真机抓包文件路径（v4.8.2）：由 Server 持有并注册进包级
	// sink（capture.go），qrlogin 等包级函数据此抓包。
	capturePath string
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
	FromUserID   string     `json:"from_user_id"`
	ToUserID     string     `json:"to_user_id"`
	ClientID     string     `json:"client_id"`
	ContextToken string     `json:"context_token"`
	ItemList     []itemElem `json:"item_list"`
}

// itemElem 消息项（type=1 文本 text_item / 2 图片 image_item / 4 文件
// file_item / 其余未知类型静默跳过）。v4.9 从匿名结构体提升为命名类型：
// 入站文件接入后测试与业务共用同一构造入口，字段漂移编译期即报错。
type itemElem struct {
	Type      int        `json:"type"`
	TextItem  *textItem  `json:"text_item,omitempty"`
	ImageItem *imageItem `json:"image_item,omitempty"`
	FileItem  *fileItem  `json:"file_item,omitempty"`
}

type textItem struct {
	Text string `json:"text"`
}

// imageItem / fileItem 是 iLink 非文本消息项（S4.5「发图即识别」协议探明第一
// 刀）：字段名按 iLink 惯例留位（file_id/url/name/md5/size），真实负载以服务端
// 下发为准——解析是防御性的，未知字段不报错。拿到 URL/file_id 后可接 vision/
// 文件下载管线做字节级识别。协议探明度与多态风险见 docs/ilink-non-text-protocol.md。
//
// v4.8.3 真机定稿（抓包+hermes/SDK 印证）：图片消息 type=2，真实下载地址在
// media.full_url（CDN 密文），密钥在 aeskey(hex)/media.aes_key(base64-of-hex)；
// file_id/url/name/md5 为留位字段（真机图片消息不下发，保留兼容）。
type imageItem struct {
	FileID string    `json:"file_id,omitempty"`
	URL    string    `json:"url,omitempty"`
	Name   string    `json:"name,omitempty"`
	MD5    string    `json:"md5,omitempty"`
	AESKey string    `json:"aeskey,omitempty"` // 32 位 hex（真机实测形态）
	Media  *cdnMedia `json:"media,omitempty"`
}

// cdnMedia 加密媒体下载票据（真机实测：full_url 指向 novac2c CDN，
// aes_key 为 base64(hex 字符串) 形态，encrypt_query_param 为下载票据）。
type cdnMedia struct {
	EncryptQueryParam string `json:"encrypt_query_param,omitempty"`
	AESKey            string `json:"aes_key,omitempty"`
	FullURL           string `json:"full_url,omitempty"`
}

// fileItem iLink 文件消息项（v4.9 真机抓包定稿 2026-09-02 16:58：入站
// file_item 与图片完全同构——下载票据在 media.full_url（novac2c CDN 密文），
// 密钥在 media.aes_key（base64-of-hex，与图片同口径）；元数据 file_name/md5/len，
// 其中 **len 真机为字符串形态**（"433849"）。旧 file_id/url/name/size 为留位
// 兼容字段（真机文件消息不下发，保留防多态）。协议细节见
// docs/ilink-non-text-protocol.md §2。
type fileItem struct {
	FileID   string    `json:"file_id,omitempty"`
	URL      string    `json:"url,omitempty"`
	Name     string    `json:"name,omitempty"`
	Size     int64     `json:"size,omitempty"`
	FileName string    `json:"file_name,omitempty"` // 真机形态：完整文件名
	MD5      string    `json:"md5,omitempty"`       // 真机形态：32 位 hex（明文摘要）
	Len      int64     `json:"len,omitempty"`       // 真机形态为字符串，防御解析转数值
	Media    *cdnMedia `json:"media,omitempty"`
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
// emoji/中文不报错；media（v4.8.3 真机形态）整体非对象时按缺失处理；
// 整体非对象时按空负载处理。
func (it *imageItem) UnmarshalJSON(data []byte) error {
	if !isJSONObject(data) {
		return nil
	}
	var raw struct {
		FileID interface{} `json:"file_id"`
		URL    interface{} `json:"url"`
		Name   string      `json:"name"`
		MD5    interface{} `json:"md5"`
		AESKey interface{} `json:"aeskey"`
		Media  *cdnMedia   `json:"media"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	it.FileID = coerceString(raw.FileID)
	it.URL = coerceString(raw.URL)
	it.Name = raw.Name
	it.MD5 = coerceString(raw.MD5)
	it.AESKey = coerceString(raw.AESKey)
	it.Media = nil
	if raw.Media != nil {
		m := *raw.Media // 防御拷贝：确保 media 内字段已是零值安全形态
		it.Media = &m
	}
	return nil
}

// resolveDownload 解析图片的下载地址与 AES 密钥（v4.8.3 真机形态优先）：
// url = media.full_url（CDN 密文地址）否则留位字段 url；key = aeskey(hex)
// 否则 media.aes_key（base64-of-hex）反解。返回 key 为空表示明文 URL。
func (it imageItem) resolveDownload() (rawURL, aesKeyHex string) {
	rawURL, aesKeyHex = it.URL, it.AESKey
	if it.Media != nil {
		if rawURL == "" {
			rawURL = it.Media.FullURL
		}
		if aesKeyHex == "" && it.Media.AESKey != "" {
			if raw, err := aesKeyFromBase64Hex(it.Media.AESKey); err == nil {
				aesKeyHex = hex.EncodeToString(raw)
			} // 解析失败宁漏勿误：key 留空按明文处理
		}
	}
	return rawURL, aesKeyHex
}

// UnmarshalJSON file_item 防御解析（v4.9 真机形态 + 旧留位兼容）：file_id/url/
// file_name/md5 多态容忍；len/size 数字/字符串皆可（真机 len="433849" 字符串）；
// media（真机同构形态）整体非对象时按缺失处理；整体非对象时按空负载处理。
// 原始负载里个别字段类型怪异（如 name 下发数字）时 json 会报 UnmarshalTypeError
// 但其余字段照常填充——这里吞掉该错误、用已填充部分继续（宁漏勿误：单字段
// 怪异绝不能让整条消息乃至整个长轮询批次解析失败重试）。
func (it *fileItem) UnmarshalJSON(data []byte) error {
	if !isJSONObject(data) {
		return nil
	}
	var raw struct {
		FileID   interface{} `json:"file_id"`
		URL      interface{} `json:"url"`
		Name     string      `json:"name"`
		Size     interface{} `json:"size"`
		FileName interface{} `json:"file_name"`
		MD5      interface{} `json:"md5"`
		Len      interface{} `json:"len"`
		Media    *cdnMedia   `json:"media"`
	}
	_ = json.Unmarshal(data, &raw) // 类型错误不外抛：已填充字段照常消费（宁漏勿误）
	it.FileID = coerceString(raw.FileID)
	it.URL = coerceString(raw.URL)
	it.Name = raw.Name
	it.Size = coerceInt64(raw.Size)
	it.FileName = coerceString(raw.FileName)
	it.MD5 = coerceString(raw.MD5)
	it.Len = coerceInt64(raw.Len)
	it.Media = nil
	if raw.Media != nil {
		// media 存在但字段全空（类型怪异被降级/空对象）等价缺失，规范化为 nil
		if raw.Media.EncryptQueryParam != "" || raw.Media.AESKey != "" || raw.Media.FullURL != "" {
			m := *raw.Media // 防御拷贝：确保 media 内字段已是零值安全形态
			it.Media = &m
		}
	}
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
	var recognized []string // 已处理项的自包含提示行（已识别图片 / 已提取文件，自带括号包装）
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
				if s.MediaRecognizer != nil {
					if u, _ := item.ImageItem.resolveDownload(); u != "" {
						if desc, ok := s.recognizeImage(*item.ImageItem, msg.FromUserID); ok {
						recognized = append(recognized, recognizedImageLabel(*item.ImageItem, desc))
						continue
					}
				}
			}
			media = append(media, imageItemLabel(*item.ImageItem))
		case item.FileItem != nil:
			mediaCount++
			if mediaCount > wxMaxMediaItems { // d③ 条数上限：超出不逐个处理
				mediaOverflow++
				continue
			}
			// v4.9 入站文件：下载解密落盘 → FileHandler 提取文本（app 线注入）。
			// 处理器 nil / 下载失败 / panic / 空结果 → 现有占位提示行（宁漏勿误
			// 不变；处理器 nil 时不发起下载，零流量）。
			if line, ok := s.processInboundFile(*item.FileItem); ok {
				recognized = append(recognized, line) // 与已识别图片同为自包含提示行
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
	// v4.8.3：空回复不推送——产物走 SendFileCard 图片卡片（图+caption 已由
	// seam 内部发出，失败亦有其内部文本降级），回调返回空串表示「已送出，
	// 勿重复推送」。
	if reply == "" {
		return
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

// fileItemDisplayName 文件名取值优先级：file_name（真机形态）→ name（旧留位）。
func fileItemDisplayName(fi fileItem) string {
	if fi.FileName != "" {
		return fi.FileName
	}
	return fi.Name
}

// fileItemLabel 文件消息占位提示（FileHandler 缺席/失败时的「宁漏勿误」退路）：
// 真机形态展示 file_name + 人类可读大小；旧留位字段兼容；全缺退「文件消息」。
func fileItemLabel(it fileItem) string {
	name := fileItemDisplayName(it)
	size := it.Len
	if size == 0 {
		size = it.Size
	}
	switch {
	case name != "" && size > 0:
		return fmt.Sprintf("文件消息（%s，%s）", name, humanBytes(size))
	case name != "":
		return "文件消息（" + name + "）"
	default:
		return "文件消息"
	}
}

// humanBytes 人类可读大小（1024 进制，KB/MB…保留 1 位小数；字节原样）。
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// ─── 图片识别管线（v4.8 子项 b）─────────────────────────────

// recognizeImage 调用注入的 MediaRecognizer 识别图片。v4.8.3 真机形态：
// 加密 CDN 媒体（media.full_url + aeskey）由本包先下载解密落临时文件，再以
// file:// 本地路径交 MediaRecognizer（DownloadImage 支持 file:// 读取本包
// 生成的解密产物，app 层注入的 OCRMediaRecognizer 无需改动即生效）；明文
// URL 走原路（识别器自行下载）。前后各一条 slog 便于真机诊断；失败或空
// 结果一律返回 false（handle 保留原占位提示行），错误不上抛不 panic。
// v4.9：fromUser 仅用于入站图片旁路钩子（OnInboundImage，改图缓存），识别
// 链路本身零改动。
func (s *Server) recognizeImage(it imageItem, fromUser string) (string, bool) {
	rawURL, aesKeyHex := it.resolveDownload()
	if rawURL == "" {
		return "", false
	}
	urlForLog := truncateRunes(rawURL, 120)
	if aesKeyHex != "" {
		// 加密 CDN：下载→AES-128-ECB 解密→临时文件→file:// 交识别器
		path, cleanup, err := DownloadImageEncrypted(rawURL, aesKeyHex)
		if err != nil {
			slog.Warn("[weixin] 加密图片下载解密失败，保留占位提示",
				"assistant", s.cfg.AssistantID, "url", urlForLog, "err", err)
			return "", false
		}
		defer cleanup()
		// v4.9 对话式改图旁路：解密落盘成功即通知缓存（钩子内部复制文件自持
		// 并 recover panic），随后 OCR 识别照旧——主流程零依赖零改动。
		s.invokeInboundImageHook(fromUser, path)
		rawURL = "file://" + path
	}
	slog.Info("[weixin] 图片识别开始",
		"assistant", s.cfg.AssistantID,
		"url", urlForLog,
		"name", it.Name)
	desc, err := s.MediaRecognizer(rawURL)
	if err != nil {
		slog.Warn("[weixin] 图片识别失败，保留占位提示",
			"assistant", s.cfg.AssistantID, "url", urlForLog, "err", err)
		return "", false
	}
	desc = strings.TrimSpace(desc)
	if desc == "" {
		slog.Warn("[weixin] 图片识别结果为空，保留占位提示",
			"assistant", s.cfg.AssistantID, "url", urlForLog)
		return "", false
	}
	slog.Info("[weixin] 图片识别完成",
		"assistant", s.cfg.AssistantID, "runes", len([]rune(desc)))
	return desc, true
}

// invokeInboundImageHook 旁路调用入站图片钩子（nil=无钩子，零开销）。钩子
// panic 一律 recover 吞掉并记日志——旁路消费者（改图缓存）绝不能打断 OCR
// 识别主流程；fromUser/localPath 原样透传。
func (s *Server) invokeInboundImageHook(fromUser, localPath string) {
	if s.OnInboundImage == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("[weixin] 入站图片旁路钩子 panic，已忽略",
				"assistant", s.cfg.AssistantID, "recover", r)
		}
	}()
	s.OnInboundImage(fromUser, localPath)
}

// ─── 入站文件管线（v4.9 跨线契约）─────────────────────────────

// processInboundFile 入站文件处理链：resolveInboundFile（下载解密落盘）→
// FileHandler 同步提取 → 临时文件用后即删。返回 (注入文本行, true) 表示成功；
// 处理器 nil（不发起下载）/ 下载解密失败 / 处理器 panic 或返回空串一律
// ("", false)——上层保留 fileItemLabel 占位提示行，宁漏勿误。
func (s *Server) processInboundFile(fi fileItem) (string, bool) {
	if s.FileHandler == nil {
		return "", false // 关闭态：零流量，行为与现状完全一致
	}
	path, name, size, md5sum, err := resolveInboundFile(fi)
	if err != nil {
		slog.Warn("[weixin] 入站文件下载解密失败，保留占位提示",
			"assistant", s.cfg.AssistantID, "name", fileItemDisplayName(fi), "err", err)
		return "", false
	}
	defer func() { _ = os.Remove(path) }() // 用后即删；处理器如需留存自行复制
	line := s.invokeFileHandler(path, name, size, md5sum)
	if strings.TrimSpace(line) == "" {
		return "", false
	}
	return line, true
}

// invokeFileHandler 调用入站文件处理器（nil=关闭）。处理器 panic 一律 recover
// 吞掉并记日志（同 OnInboundImage 先例）——文件提取消费者绝不能打断消息主流程。
func (s *Server) invokeFileHandler(localPath, fileName string, sizeBytes int64, md5sum string) (result string) {
	if s.FileHandler == nil {
		return ""
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("[weixin] 入站文件处理器 panic，已忽略",
				"assistant", s.cfg.AssistantID, "file", fileName, "recover", r)
			result = ""
		}
	}()
	return s.FileHandler(localPath, fileName, sizeBytes, md5sum)
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

// SendFileCard 产物回推 seam（v4.8.3 真机协议版 + v4.9 文件卡泛化）：按扩展名
// 分流——图片白名单（png/jpeg/webp/gif）走真机定稿图片卡链（getuploadurl
// media_type=1 → CDN 密文上传 → image_item 卡片，逐字节不变）；非图片
//（docx/xlsx/pptx/pdf/zip/txt/md 等）走文件卡链（探针制：media_type=3 +
// file_item，upload_probe 已埋点，任何失败降级图片卡链同款文本路径）；
// 成功后 caption 独立文本补发。协议经三方印证（本机抓包实测解密、hermes-
// agent 生产实现、openilink-sdk-go 导出符号），上传域与扫码 baseurl 无关。
// 无活跃会话/文件不合规/任何失败/panic 一律降级现有文本卡片（逐字节不变）
// ——产物回推绝不崩主流程。
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
	return s.sendFileCardUpload(localPath, caption, &sent)
}

// ─── 产物上传（v4.8.3：真机协议定稿）──────────────────────────────────────

// uploadImageExts 图片产物扩展名白名单（图片产物场景，对齐 media_download 的
// png/jpeg/webp/gif 口径）。
var uploadImageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true,
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

// sendFileCardUpload 上传主流程：无活跃会话→抓 skipped 后文本降级；否则按
// 扩展名分流（图片→uploadImageToCDN 真机定稿链；非图片→uploadFileToCDN
// 探针制文件卡链），成功后 caption 独立补发（顺序与 hermes 相反：卡先文后
// ——卡片失败时整体降级单条文本卡片，避免重复推送）。任何失败经 sent 告知
// 外层（panic 恢复时不再重复推送）。
func (s *Server) sendFileCardUpload(localPath, caption string, sent *bool) error {
	name := filepath.Base(localPath)
	from, ctxTokn := s.LastPeer()
	if from == "" {
		capture("upload_probe", map[string]interface{}{"skipped": "no_peer", "file": name})
		return s.sendFileCardText(localPath, caption)
	}
	isImage := uploadImageExts[strings.ToLower(filepath.Ext(name))]
	var item map[string]interface{}
	var err error
	if isImage {
		item, err = s.uploadImageToCDN(localPath, from)
	} else {
		item, err = s.uploadFileToCDN(localPath, from) // 探针制文件卡链
	}
	if err != nil {
		slog.Warn("[weixin] 产物上传失败，降级文本卡片",
			"assistant", s.cfg.AssistantID, "file", name, "err", err)
		rec := map[string]interface{}{"stage": "upload", "file": name, "err": err.Error()}
		if !isImage {
			rec["card"] = "file" // 文件链失败节点：图片链记录保持原样零新增
		}
		capture("upload_probe", rec)
		return s.sendFileCardText(localPath, caption)
	}
	if isImage {
		err = s.sendImageCardViaUpload(item)
	} else {
		err = s.sendFileCardViaUpload(item, name)
	}
	if err != nil {
		stage := "send_image_card"
		if !isImage {
			stage = "send_file_card"
		}
		slog.Warn("[weixin] 媒体卡片发送失败，降级文本卡片",
			"assistant", s.cfg.AssistantID, "file", name, "card", stage, "err", err)
		capture("upload_probe", map[string]interface{}{"stage": stage, "file": name, "err": err.Error()})
		return s.sendFileCardText(localPath, caption)
	}
	*sent = true
	rec := map[string]interface{}{"stage": "delivered", "file": name}
	if !isImage {
		rec["card"] = "file"
	}
	capture("upload_probe", rec)
	if caption != "" {
		// caption 独立文本消息补发（失败仅记日志：卡片本身已送达）
		if err := s.Send(from, ctxTokn, caption); err != nil {
			slog.Warn("[weixin] 卡片 caption 补发失败", "assistant", s.cfg.AssistantID, "err", err)
		}
	}
	return nil
}

// readUploadableFile 读取待上传图片产物：扩展名白名单 + 20MiB 上限（沿用
// wxImageMaxBytes 口径）。不合规返回错误（探针层抓包后降级文本卡片）。
func readUploadableFile(path string) ([]byte, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if !uploadImageExts[ext] {
		return nil, fmt.Errorf("扩展名 %q 不在图片白名单（png/jpeg/webp/gif）", ext)
	}
	return readUploadableFileAny(path, wxImageMaxBytes)
}

// readUploadableFileAny 读取待上传产物（类型不限形态；v4.9 文件卡链用）：
// 仅尺寸上限防线（入站文件同款 50MiB 口径），扩展名由调用方分流决定。
func readUploadableFileAny(path string, maxBytes int64) ([]byte, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Size() > maxBytes {
		return nil, fmt.Errorf("文件 %d 字节超过上传上限 %d", st.Size(), maxBytes)
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
