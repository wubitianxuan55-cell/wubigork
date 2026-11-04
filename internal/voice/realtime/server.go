// server.go — 8791 /v1/realtime WS 服务器（进程内，会话池=1）。
//
// 行为对齐 sidecar（上游 6feb527 api/openai_realtime）：
//   - 第二条连接 → error{session_limit_reached} + close 1008；
//   - 单写者泵（出站帧有序，response.created 顺序守卫由此免费获得）；
//   - 未知 client 事件宽容忽略；坏 JSON 回 error 帧不断连。
//
// Handler 接口是刀3+ 真管线的接缝；刀2 默认挂 ProtocolEcho（协议回声响应器）。
package realtime

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Handler 管线接缝：session 生命周期事件下沉点。
type Handler interface {
	// OnAudio 追加一帧上行音频（PCM16 16k 原始字节）。
	OnAudio(s *Session, pcm []byte)
	// OnSessionConfig session.update 合并后的配置。
	OnSessionConfig(s *Session, cfg SessionConfig)
	// OnResponseCreate 请求生成响应（server_vad 或客户端发起）。
	OnResponseCreate(s *Session)
	// OnResponseCancel 打断当前响应。
	OnResponseCancel(s *Session)
	// OnSessionEnd 会话断开。
	OnSessionEnd(s *Session)
}

// Server 8791 WS 服务器。Start/Stop 幂等；单会话池。
type Server struct {
	Addr    string // 形如 127.0.0.1:8791
	Handler Handler

	mu      sync.Mutex
	httpSrv *http.Server
	cur     *Session
}

// Session 一条实时连接的会话状态。
type Session struct {
	ID      string
	ConvID  string
	cfg     SessionConfig
	srv     *Server
	conn    *websocket.Conn
	out     chan map[string]any
	done    chan struct{}
	closeMu sync.Mutex
	closed  bool
}

// Config 会话配置快照（读并发安全）。
func (s *Session) Config() SessionConfig {
	return s.cfg
}

// Send 出站帧（顺序由单写者泵保证）。
func (s *Session) Send(ev map[string]any) {
	select {
	case s.out <- ev:
	case <-s.done:
	}
}

// Close 关闭会话（幂等）。
func (s *Session) Close() {
	s.closeMu.Lock()
	if s.closed {
		s.closeMu.Unlock()
		return
	}
	s.closed = true
	s.closeMu.Unlock()
	close(s.done)
	_ = s.conn.Close()
}

// Start 启动监听（阻塞前立刻返回；幂等）。
func (srv *Server) Start() error {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.httpSrv != nil {
		return nil
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("realtime 监听 %s 失败: %w", srv.Addr, err)
	}
	srv.Addr = ln.Addr().String() // :0 实测端口回填（测试用）
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/realtime", srv.handleWS)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv.httpSrv = &http.Server{Handler: mux}
	go func() {
		if err := srv.httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Warn("voice realtime server 退出", "error", err)
		}
	}()
	return nil
}

// Stop 停止监听并断开活动会话（幂等）。
func (srv *Server) Stop() {
	srv.mu.Lock()
	hs := srv.httpSrv
	srv.httpSrv = nil
	srv.mu.Unlock()
	if hs != nil {
		_ = hs.Close()
	}
	srv.mu.Lock()
	cur := srv.cur
	srv.cur = nil
	srv.mu.Unlock()
	if cur != nil {
		cur.Close()
	}
}

// upgrader 仅监听 127.0.0.1，来源校验放行（前端 origin 随 wails/端口变化）。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  64 * 1024,
	WriteBufferSize: 64 * 1024,
	CheckOrigin:     func(*http.Request) bool { return true },
}

func (srv *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	srv.mu.Lock()
	if srv.cur != nil {
		srv.mu.Unlock()
		conn, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			_ = conn.WriteJSON(evError("session_limit_reached"))
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(1008, "session_limit_reached"),
				time.Now().Add(time.Second))
			_ = conn.Close()
		}
		return
	}
	// 抢占会话槽（未升级成功也要清槽）
	sess := &Session{
		ID:     "session_" + randHex(16),
		ConvID: "conv_" + randHex(16),
		srv:    srv,
		out:    make(chan map[string]any, 512),
		done:   make(chan struct{}),
	}
	srv.cur = sess
	srv.mu.Unlock()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		srv.mu.Lock()
		if srv.cur == sess {
			srv.cur = nil
		}
		srv.mu.Unlock()
		sess.Close()
		return
	}
	sess.conn = conn
	slog.Info("voice realtime 会话接入", "session", sess.ID, "remote", r.RemoteAddr)

	// 单写者泵
	go func() {
		for {
			select {
			case ev := <-sess.out:
				if err := conn.WriteJSON(ev); err != nil {
					sess.Close()
					return
				}
			case <-sess.done:
				return
			}
		}
	}()

	sess.Send(evSessionCreated(sess.ID, sess.cfg))
	if srv.Handler != nil {
		srv.Handler.OnSessionConfig(sess, sess.cfg)
	}

	srv.readLoop(sess)

	slog.Info("voice realtime 会话断开", "session", sess.ID)
	sess.Close()
	srv.mu.Lock()
	if srv.cur == sess {
		srv.cur = nil
	}
	srv.mu.Unlock()
	if srv.Handler != nil {
		srv.Handler.OnSessionEnd(sess)
	}
}

func (srv *Server) readLoop(sess *Session) {
	conn := sess.conn
	conn.SetReadLimit(8 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})
	// 写侧心跳
	go func() {
		t := time.NewTicker(25 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second))
			case <-sess.done:
				return
			}
		}
	}()
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		srv.dispatch(sess, raw)
		select {
		case <-sess.done:
			return
		default:
		}
	}
}

func (srv *Server) dispatch(sess *Session, raw []byte) {
	var env clientEnvelope
	if err := json.Unmarshal(raw, &env); err != nil || env.Type == "" {
		sess.Send(evError("invalid event: missing or bad type"))
		return
	}
	switch env.Type {
	case "session.update":
		cfg, err := parseSessionUpdate(env.Session)
		if err != nil {
			sess.Send(evError(err.Error()))
			return
		}
		mergeSessionConfig(&sess.cfg, cfg)
		sess.Send(evSessionUpdated(sess.cfg))
		if srv.Handler != nil {
			srv.Handler.OnSessionConfig(sess, sess.cfg)
		}
	case "input_audio_buffer.append":
		pcm, err := decodeAppendAudio(env.Audio)
		if err != nil {
			sess.Send(evError(err.Error()))
			return
		}
		if srv.Handler != nil && len(pcm) > 0 {
			srv.Handler.OnAudio(sess, pcm)
		}
	case "input_audio_buffer.commit":
		sess.Send(evBufferCommitted("item_" + randHex(8)))
	case "input_audio_buffer.clear":
		// 清空上行缓冲（刀3 管线态生效；协议壳无缓冲可清）
	case "conversation.item.create":
		if len(env.Item) > 0 {
			var item map[string]any
			if err := json.Unmarshal(env.Item, &item); err == nil {
				sess.Send(evItemCreated(item))
			}
		}
	case "response.create":
		if srv.Handler != nil {
			srv.Handler.OnResponseCreate(sess)
		}
	case "response.cancel":
		if srv.Handler != nil {
			srv.Handler.OnResponseCancel(sess)
		}
	default:
		// 未知/扩展事件：宽容忽略（契约要求解码不丢不炸）
	}
}

func mergeSessionConfig(dst *SessionConfig, patch SessionConfig) {
	if patch.Instructions != "" {
		dst.Instructions = patch.Instructions
	}
	if patch.Voice != "" {
		dst.Voice = patch.Voice
	}
	if patch.VAD.Type != "" {
		dst.VAD = patch.VAD
	}
}

func randHex(nBytes int) string {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()[:nBytes*2]
	}
	return hex.EncodeToString(b)
}
