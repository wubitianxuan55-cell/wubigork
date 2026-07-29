package asr

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// StreamASR WebSocket 流式语音识别客户端
//
// 对接 Herdsman /v1/audio/transcriptions/stream?model=... WebSocket 端点
// 对齐 Ackem asr_engine.py 的流式设计：持续推送 PCM16/16k mono 帧，异步接收识别结果
type StreamASR struct {
	baseURL string
	model   string
	conn    *websocket.Conn
	mu      sync.Mutex
	closed  bool

	// 回调
	onResult   func(text string, isFinal bool)
	onError    func(err error)
	onComplete func()

	done chan struct{}
}

// StreamResult WebSocket 返回的识别结果
type StreamResult struct {
	Text    string `json:"text"`
	IsFinal bool   `json:"is_final"`
}

// NewStreamASR 创建流式 ASR 客户端
// model 推荐：sherpa-onnx-streaming-zipformer-zh-14m（实时中英文）、funasr（实时中文）
func NewStreamASR(baseURL, model string) *StreamASR {
	if model == "" {
		model = "sherpa-onnx-streaming-zipformer-zh-14m"
	}
	return &StreamASR{
		baseURL: baseURL,
		model:   model,
		done:    make(chan struct{}),
	}
}

// OnResult 设置识别结果回调
func (s *StreamASR) OnResult(fn func(text string, isFinal bool)) {
	s.onResult = fn
}

// OnError 设置错误回调
func (s *StreamASR) OnError(fn func(err error)) {
	s.onError = fn
}

// OnComplete 设置完成回调
func (s *StreamASR) OnComplete(fn func()) {
	s.onComplete = fn
}

// Connect 建立 WebSocket 连接并开始接收结果
func (s *StreamASR) Connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("asr stream: already closed")
	}

	// 构造 WebSocket URL
	u, err := url.Parse(s.baseURL)
	if err != nil {
		return fmt.Errorf("asr stream: parse base URL: %w", err)
	}
	u = u.JoinPath("/v1/audio/transcriptions/stream")
	u.RawQuery = "model=" + s.model

	wsURL := u.String()
	if u.Scheme == "http" {
		wsURL = "ws" + wsURL[4:]
	} else if u.Scheme == "https" {
		wsURL = "wss" + wsURL[5:]
	}

	slog.Info("ASR WebSocket 连接中", "url", wsURL, "model", s.model)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("asr stream: dial: %w", err)
	}
	s.conn = conn

	// 启动接收 goroutine
	go s.readLoop()

	return nil
}

// SendAudio 发送 PCM16/16k mono 音频帧
func (s *StreamASR) SendAudio(pcmData []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.conn == nil {
		return fmt.Errorf("asr stream: not connected")
	}

	// 设置写超时
	s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return s.conn.WriteMessage(websocket.BinaryMessage, pcmData)
}

// SendEnd 发送结束信号（发送空文本帧）
func (s *StreamASR) SendEnd() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.conn == nil {
		return nil
	}

	s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return s.conn.WriteMessage(websocket.TextMessage, []byte(`{"eof":true}`))
}

// Close 关闭连接
func (s *StreamASR) Close() error {
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
		// 发送关闭帧
		msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(3*time.Second))
		conn.Close()
	}

	select {
	case <-s.done:
	default:
		close(s.done)
	}

	return nil
}

// Wait 等待直到连接关闭
func (s *StreamASR) Wait() {
	<-s.done
}

// readLoop 持续读取 WebSocket 消息
func (s *StreamASR) readLoop() {
	defer func() {
		s.mu.Lock()
		s.closed = true
		if s.conn != nil {
			s.conn.Close()
			s.conn = nil
		}
		s.mu.Unlock()

		select {
		case <-s.done:
		default:
			close(s.done)
		}

		if s.onComplete != nil {
			s.onComplete()
		}
	}()

	for {
		s.mu.Lock()
		if s.closed || s.conn == nil {
			s.mu.Unlock()
			return
		}
		conn := s.conn
		s.mu.Unlock()

		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			if s.onError != nil {
				s.onError(fmt.Errorf("asr stream: read: %w", err))
			}
			return
		}

		// 尝试解析 JSON 结果
		var result StreamResult
		if err := json.Unmarshal(message, &result); err != nil {
			slog.Warn("ASR stream: 无法解析结果", "raw", string(message))
			continue
		}

		if s.onResult != nil && result.Text != "" {
			s.onResult(result.Text, result.IsFinal)
		}
	}
}
