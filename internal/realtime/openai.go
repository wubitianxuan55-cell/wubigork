// openai.go — OpenAI Realtime API 会话实现（kind "openai"）。
//
// 协议（通用实现 + 防御解析为主，宁漏勿误）：
//   - 握手：WSS {BaseURL}?model={Model}，头 Authorization: Bearer + OpenAI-Beta: realtime=v1
//   - 初始化：Dial 内下发 session.update（instructions / voice / 输入音频格式 pcm16 16k）
//   - 输入：SendAudio 构造 input_audio_buffer.append（audio = base64(PCM16/16k mono)）
//   - 输出：readLoop 解析服务端 JSON 事件 → 类型映射 Event 常量 → Events 通道；
//     response.audio.delta 的 base64 音频解码进 Event.AudioPCM；未识别事件一律
//     EventUnknown（携带原始 JSON），单条解析失败仅告警跳过，不崩不中断会话。
package realtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// DefaultOpenAIBaseURL OpenAI Realtime WebSocket 端点（Config.BaseURL 空时使用；
	// 自定义端点需含完整路径，http(s) scheme 自动转 ws(s) 供离线测试注入）。
	DefaultOpenAIBaseURL = "wss://api.openai.com/v1/realtime"
	// DefaultOpenAIModel 默认 Realtime 模型（Config.Model 空时使用）。
	DefaultOpenAIModel = "gpt-4o-realtime-preview"
	// DefaultOpenAIVoice 默认输出音色（Config.Voice 空时使用）。
	DefaultOpenAIVoice = "alloy"

	// 输入音频格式：PCM16 16kHz 单声道（与 voice 管线采集参数一致，见
	// internal/voice 的 SampleRate/BitDepth/Channels 常量）。
	openAIInputAudioFormat = "pcm16"
	// 发送写超时（对齐 internal/asr StreamASR）。
	writeTimeout = 5 * time.Second
	// 事件通道缓冲；满时丢弃新事件并告警（绝不阻塞读循环/造成死锁）。
	eventBuffer = 128
)

func init() {
	Register("openai", func(cfg Config) (RealtimeSession, error) { return newOpenAISession(cfg) })
}

// OpenAISession OpenAI Realtime API 会话（RealtimeSession 实现）。
type OpenAISession struct {
	cfg Config

	conn   *websocket.Conn
	mu     sync.Mutex // 串行化 conn 写与关闭
	closed bool

	events chan Event
}

// newOpenAISession 构造 OpenAI Realtime 会话（仅校验/补默认值，不做网络 I/O）。
func newOpenAISession(cfg Config) (RealtimeSession, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("realtime/openai: api key is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultOpenAIBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultOpenAIModel
	}
	if cfg.Voice == "" {
		cfg.Voice = DefaultOpenAIVoice
	}
	return &OpenAISession{cfg: cfg, events: make(chan Event, eventBuffer)}, nil
}

// Dial 建立 WebSocket 连接，启动事件读循环并下发 session.update。
func (s *OpenAISession) Dial(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("realtime/openai: session closed")
	}
	if s.conn != nil {
		s.mu.Unlock()
		return fmt.Errorf("realtime/openai: already connected")
	}
	s.mu.Unlock()

	endpoint, err := s.endpoint()
	if err != nil {
		return err
	}
	header := http.Header{}
	header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	header.Set("OpenAI-Beta", "realtime=v1")

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, endpoint, header)
	if err != nil {
		return fmt.Errorf("realtime/openai: dial %s: %w", endpoint, err)
	}

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	go s.readLoop()

	if err := s.sendSessionUpdate(); err != nil {
		s.Close()
		return fmt.Errorf("realtime/openai: send session.update: %w", err)
	}
	return nil
}

// endpoint 计算拨号地址：http(s) scheme 自动转 ws(s)（离线测试注入 httptest
// 端点用），ws(s) 原样使用；模型经 query 参数传递。
func (s *OpenAISession) endpoint() (string, error) {
	u, err := url.Parse(s.cfg.BaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("realtime/openai: invalid base URL %q", s.cfg.BaseURL)
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("realtime/openai: unsupported scheme %q", u.Scheme)
	}
	q := u.Query()
	q.Set("model", s.cfg.Model)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// sessionUpdate session.update 下发载荷（instructions/voice/输入音频格式）。
type sessionUpdate struct {
	Type    string             `json:"type"`
	Session sessionUpdateVoice `json:"session"`
}

type sessionUpdateVoice struct {
	Instructions     string `json:"instructions,omitempty"`
	Voice            string `json:"voice"`
	InputAudioFormat string `json:"input_audio_format"`
}

// sendSessionUpdate 下发会话配置（instructions / voice / pcm16 输入音频）。
func (s *OpenAISession) sendSessionUpdate() error {
	return s.writeJSON(sessionUpdate{
		Type: "session.update",
		Session: sessionUpdateVoice{
			Instructions:     s.cfg.Instructions,
			Voice:            s.cfg.Voice,
			InputAudioFormat: openAIInputAudioFormat,
		},
	})
}

// inputAudioAppend input_audio_buffer.append 载荷。
type inputAudioAppend struct {
	Type  string `json:"type"`
	Audio string `json:"audio"` // base64(PCM16/16k mono)
}

// SendAudio 发送一帧输入音频：input_audio_buffer.append（base64 PCM16/16k mono）。
// 空帧防御性忽略（无意义负载）。
func (s *OpenAISession) SendAudio(pcm []byte) error {
	if len(pcm) == 0 {
		return nil
	}
	return s.writeJSON(inputAudioAppend{
		Type:  "input_audio_buffer.append",
		Audio: base64.StdEncoding.EncodeToString(pcm),
	})
}

// writeJSON 加锁写一条 JSON 文本消息（未连接/已关闭报错）。
func (s *OpenAISession) writeJSON(v interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.conn == nil {
		return fmt.Errorf("realtime/openai: not connected")
	}
	s.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return s.conn.WriteJSON(v)
}

// Events 返回服务端事件通道。
func (s *OpenAISession) Events() <-chan Event {
	return s.events
}

// Close 关闭会话（幂等）。未 Dial 直接 Close 合法（探测场景）。
func (s *OpenAISession) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	conn := s.conn
	s.conn = nil
	s.mu.Unlock()

	if conn != nil {
		msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		_ = conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(writeTimeout))
		_ = conn.Close()
	}
	return nil
}

// readLoop 持续读取服务端 JSON 事件，规范化后投递 Events 通道。
// 连接断开（含 Close 触发）时退出并关闭事件通道（读循环只启动一次，
// 所有 emit 都发生在本 goroutine 内，defer close 无 send-on-closed 竞态）。
func (s *OpenAISession) readLoop() {
	defer close(s.events)
	for {
		s.mu.Lock()
		conn := s.conn
		s.mu.Unlock()
		if conn == nil {
			return
		}
		messageType, raw, err := conn.ReadMessage()
		if err != nil {
			if s.isOpen() && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Warn("Realtime 读循环退出", "kind", "openai", "error", err)
			}
			return
		}
		if messageType != websocket.TextMessage {
			slog.Warn("Realtime 收到非文本消息，忽略", "kind", "openai", "messageType", messageType)
			continue
		}
		s.dispatch(raw)
	}
}

// dispatch 解析一条服务端事件并投递（未知事件 → EventUnknown 不丢原始 JSON）。
func (s *OpenAISession) dispatch(raw []byte) {
	var head struct {
		Type  string `json:"type"`
		Delta string `json:"delta"` // response.audio.delta / response.text.delta 增量载荷
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		// 防御：非 JSON 消息按 Unknown 透传（宁漏勿误，不崩）
		s.emit(Event{Type: EventUnknown, DataJSON: string(raw)})
		return
	}
	ev := Event{Type: eventKind(head.Type), DataJSON: string(raw)}
	if head.Type == EventResponseAudioDelta && head.Delta != "" {
		if pcm, err := base64.StdEncoding.DecodeString(head.Delta); err == nil {
			ev.AudioPCM = pcm
		} else {
			slog.Warn("Realtime audio delta base64 解码失败（保留原始 JSON）", "kind", "openai", "error", err)
		}
	}
	s.emit(ev)
}

// eventKind 服务端事件名 → Event.Type（已映射事件名与常量值一致，原样透传；
// 未识别事件 → EventUnknown）。
func eventKind(serverType string) string {
	if _, ok := knownServerEvents[serverType]; ok {
		return serverType
	}
	return EventUnknown
}

// knownServerEvents 已映射的 OpenAI Realtime 服务端事件名。
var knownServerEvents = map[string]struct{}{
	EventSessionCreated:                {},
	EventSessionUpdated:                {},
	EventInputAudioBufferSpeechStarted: {},
	EventInputAudioBufferSpeechStopped: {},
	EventResponseAudioDelta:            {},
	EventResponseAudioDone:             {},
	EventResponseTextDelta:             {},
	EventError:                         {},
}

// emit 投递事件；通道满时丢弃并告警（绝不阻塞读循环/造成死锁）。
func (s *OpenAISession) emit(ev Event) {
	select {
	case s.events <- ev:
	default:
		slog.Warn("Realtime 事件通道已满，丢弃事件", "kind", "openai", "type", ev.Type)
	}
}

// isOpen 会话是否仍处于打开状态（供读循环区分主动关闭与异常断开）。
func (s *OpenAISession) isOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.closed
}
