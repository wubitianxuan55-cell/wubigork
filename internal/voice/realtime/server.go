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
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// voiceSessionIdleTakeover 占位会话空闲接管阈值。占位连接可能是孤儿（客户端已
// 丢弃引用但从未 close——浏览器仍代回 pong，读超时 60s 永远杀不掉），表现为会话
// 槽被永久占用、后续所有连接被 session_limit_reached 弹回=语音台无限重连卡死。
// 判据：活跃会话（开麦）每 20ms 上行音频帧永不空闲；静麦闲置会话在有人新连接
// 时让位=换台语义。2026-11-04 实弹三次重启同一死局后的自愈刀。
const voiceSessionIdleTakeover = 60 * time.Second

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
	// OnPlaybackDrained 客户端播放排空（回声门真实关闭锚点）。
	OnPlaybackDrained(s *Session)
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
	// lastInbound 最近一次客户端帧（UnixNano，dispatch 刷新）。孤儿连接只有
	// 浏览器协议层 pong、零应用帧，据此与服务端读超时（pong 会喂）区分开。
	lastInbound atomic.Int64
	// exit handleWS 全程（含 readLoop 与 OnSessionEnd）完成后关闭。空闲接管
	// 必须等它：OnSessionEnd 是管线全量重置，晚到会抹掉新会话已写入的配置。
	exit chan struct{}
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
	mux.HandleFunc("/v1/diag/playback", srv.handleDiagPlayback)
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
	for {
		srv.mu.Lock()
		if srv.cur == nil {
			break // 空槽在手（保持持锁），落到下方抢占
		}
		idle := time.Since(time.Unix(0, srv.cur.lastInbound.Load()))
		if idle <= voiceSessionIdleTakeover {
			srv.mu.Unlock()
			conn, err := upgrader.Upgrade(w, r, nil)
			if err == nil {
				// 拒绝必须留痕：2026-11-04 卡死排查时服务端对弹连完全静默，
				// 只能靠前端倒推。idle 一并记录，便于区分活会话与僵尸占位。
				slog.Warn("voice realtime 拒绝第二连接（会话池=1）", "incumbent_idle", idle.Round(time.Millisecond))
				_ = conn.WriteJSON(evError("session_limit_reached"))
				_ = conn.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(1008, "session_limit_reached"),
					time.Now().Add(time.Second))
				_ = conn.Close()
			}
			return
		}
		// 僵尸占位自愈：占位会话超过阈值零客户端帧=孤儿连接（浏览器 pong
		// 保活使其对读超时免疫），收走会话槽放行新连接，旧连接异步关闭。
		// 必须等旧会话 epilogue（OnSessionEnd 全量重置）走完再放行，
		// 否则晚到的重置会抹掉新会话已写入的 VAD/voice 配置。
		stale := srv.cur
		srv.cur = nil
		srv.mu.Unlock()
		slog.Warn("voice realtime 接管空闲占位会话", "stale", stale.ID, "idle", idle.Round(time.Millisecond))
		stale.Close()
		select {
		case <-stale.exit:
		case <-time.After(5 * time.Second):
			slog.Warn("voice realtime 等待旧会话退出超时（照常放行）", "stale", stale.ID)
		}
		// 重回循环：等待期间可能有别的连接已抢槽
	}
	// 抢占会话槽（持锁自循环落入；未升级成功也要清槽）
	sess := &Session{
		ID:     "session_" + randHex(16),
		ConvID: "conv_" + randHex(16),
		srv:    srv,
		out:    make(chan map[string]any, 512),
		done:   make(chan struct{}),
		exit:   make(chan struct{}),
	}
	sess.lastInbound.Store(time.Now().UnixNano())
	srv.cur = sess
	srv.mu.Unlock()
	defer close(sess.exit)

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
	// M12 蒸馏（router:290-385 关闭序同位：unit 在 SESSION_END 全链 drain 后才
	// 释放）——槽位释放必须晚于 OnSessionEnd，否则快速重连可在此窗口抢槽写入
	// 配置/音频，被随后的旧会话全量重置抹掉（空闲接管路径本就等 stale.exit，
	// 不受影响）。
	if srv.Handler != nil {
		srv.Handler.OnSessionEnd(sess)
	}
	srv.mu.Lock()
	if srv.cur == sess {
		srv.cur = nil
	}
	srv.mu.Unlock()
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
	sess.lastInbound.Store(time.Now().UnixNano())
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
	case "playback.drained":
		// 播放排空回执（回声门真实关闭锚点；上游 block 模式 playback.is_active
		// 的跨机等价物——播放状态在本侧客户端，服务端无法自知）
		if srv.Handler != nil {
			srv.Handler.OnPlaybackDrained(sess)
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
