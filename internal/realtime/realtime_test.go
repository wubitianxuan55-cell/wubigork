// Package realtime — seam 与 OpenAI Realtime 会话测试（S0，全离线）。
//
// 用 httptest.Server + websocket.Upgrader 造假 Realtime 服务：
//   - 校验 Dial 握手头（Authorization / OpenAI-Beta）与 ?model= 查询参数
//   - 校验 session.update 与 input_audio_buffer.append 载荷（base64 PCM）
//   - 校验服务端事件 → Event 常量映射（含 audio.delta PCM 解码、未知事件容错不崩）
package realtime

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// ── 造假 Realtime 服务 ──

// upgradeCapture 握手请求捕获（Authorization/OpenAI-Beta 头与 model 查询参数）。
type upgradeCapture struct {
	model         string
	authorization string
	openAIBeta    string
}

// fakeServer 起一个假 Realtime WebSocket 服务。服务端连接经 conns 交给测试
// 主动下发事件；客户端发来的文本消息经 msgs 回传给测试断言。
func fakeServer(t *testing.T) (srv *httptest.Server, conns <-chan *websocket.Conn, msgs <-chan []byte, capture *upgradeCapture) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	connCh := make(chan *websocket.Conn, 1)
	msgCh := make(chan []byte, 32)
	cap := &upgradeCapture{}
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.model = r.URL.Query().Get("model")
		cap.authorization = r.Header.Get("Authorization")
		cap.openAIBeta = r.Header.Get("OpenAI-Beta")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		connCh <- conn
		for {
			mt, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.TextMessage {
				msgCh <- raw
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv, connCh, msgCh, cap
}

// dialTestSession 经注册表构造并拨号到 fake 服务（测试辅助；Cleanup 自动关闭）。
func dialTestSession(t *testing.T, cfg Config) *OpenAISession {
	t.Helper()
	s, err := New("openai", cfg)
	if err != nil {
		t.Fatalf("New(openai): %v", err)
	}
	sess, ok := s.(*OpenAISession)
	if !ok {
		t.Fatalf("类型应为 *OpenAISession, got %T", s)
	}
	if err := sess.Dial(context.Background()); err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	return sess
}

// readEvent 带超时读取一个会话事件。
func readEvent(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(2 * time.Second):
		t.Fatal("等待会话事件超时")
		return Event{}
	}
}

// ── seam 注册表 ──

func TestRegistry_Kinds(t *testing.T) {
	kinds := Kinds()
	if len(kinds) != 1 || kinds[0] != "openai" {
		t.Fatalf("kinds = %v, want [openai]（当前唯一实现）", kinds)
	}
}

func TestRegistry_ConstructWithDefaults(t *testing.T) {
	s, err := New("openai", Config{APIKey: "sk-test"})
	if err != nil {
		t.Fatalf("New(openai): %v", err)
	}
	o := s.(*OpenAISession)
	if o.cfg.BaseURL != DefaultOpenAIBaseURL || o.cfg.Model != DefaultOpenAIModel || o.cfg.Voice != DefaultOpenAIVoice {
		t.Errorf("默认值未生效: baseURL=%q model=%q voice=%q", o.cfg.BaseURL, o.cfg.Model, o.cfg.Voice)
	}
}

func TestRegistry_MissingAPIKeyFailsClosed(t *testing.T) {
	if _, err := New("openai", Config{}); err == nil {
		t.Fatal("缺 APIKey 应报错（fail-closed）")
	}
}

func TestRegistry_UnknownKindFailsClosed(t *testing.T) {
	if _, err := New("no-such-realtime", Config{}); err == nil {
		t.Fatal("未知 kind 应报错（fail-closed）")
	}
}

func TestRegistry_DuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("重复注册应 panic（互斥注册纪律）")
		}
	}()
	Register("openai", func(cfg Config) (RealtimeSession, error) { return nil, nil }) // 已注册 → panic
}

// TestOpenAISession_ImplementsInterface 编译期断言：OpenAISession 满足 RealtimeSession。
func TestOpenAISession_ImplementsInterface(t *testing.T) {
	var _ RealtimeSession = (*OpenAISession)(nil)
}

// ── endpoint 推导 ──

func TestOpenAISession_EndpointSchemeConversion(t *testing.T) {
	s, err := New("openai", Config{APIKey: "sk-test", BaseURL: "https://example.com/v1/realtime", Model: "m1"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ep, err := s.(*OpenAISession).endpoint()
	if err != nil {
		t.Fatalf("endpoint: %v", err)
	}
	if want := "wss://example.com/v1/realtime?model=m1"; ep != want {
		t.Errorf("endpoint = %q, want %q", ep, want)
	}
}

func TestOpenAISession_EndpointInvalid(t *testing.T) {
	s, err := New("openai", Config{APIKey: "sk-test", BaseURL: "ftp://x/y"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := s.(*OpenAISession).endpoint(); err == nil {
		t.Fatal("不支持 scheme 应报错")
	}
}

// ── Dial 握手 + session.update ──

func TestOpenAISession_DialHandshakeAndSessionUpdate(t *testing.T) {
	srv, _, msgs, cap := fakeServer(t)
	dialTestSession(t, Config{
		BaseURL:      srv.URL,
		APIKey:       "sk-test",
		Instructions: "你是测试助手",
		Voice:        "alloy",
		Model:        "gpt-4o-realtime-preview",
	})

	if cap.authorization != "Bearer sk-test" {
		t.Errorf("Authorization = %q, want %q", cap.authorization, "Bearer sk-test")
	}
	if cap.openAIBeta != "realtime=v1" {
		t.Errorf("OpenAI-Beta = %q, want %q", cap.openAIBeta, "realtime=v1")
	}
	if cap.model != "gpt-4o-realtime-preview" {
		t.Errorf("model query = %q, want %q", cap.model, "gpt-4o-realtime-preview")
	}

	// 服务端应先收到 session.update（instructions/voice/pcm16）
	select {
	case raw := <-msgs:
		var upd struct {
			Type    string `json:"type"`
			Session struct {
				Instructions     string `json:"instructions"`
				Voice            string `json:"voice"`
				InputAudioFormat string `json:"input_audio_format"`
			} `json:"session"`
		}
		if err := json.Unmarshal(raw, &upd); err != nil {
			t.Fatalf("解析 session.update: %v", err)
		}
		if upd.Type != "session.update" {
			t.Errorf("首条消息 type = %q, want session.update", upd.Type)
		}
		if upd.Session.Instructions != "你是测试助手" {
			t.Errorf("instructions = %q, want 测试指令透传", upd.Session.Instructions)
		}
		if upd.Session.Voice != "alloy" {
			t.Errorf("voice = %q, want alloy", upd.Session.Voice)
		}
		if upd.Session.InputAudioFormat != "pcm16" {
			t.Errorf("input_audio_format = %q, want pcm16", upd.Session.InputAudioFormat)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("未收到 session.update")
	}
}

// ── SendAudio ──

func TestOpenAISession_SendAudio(t *testing.T) {
	srv, _, msgs, _ := fakeServer(t)
	sess := dialTestSession(t, Config{BaseURL: srv.URL, APIKey: "sk-test"})

	pcm := []byte{0x01, 0x02, 0x03, 0x04, 0xFF, 0x00}
	if err := sess.SendAudio(pcm); err != nil {
		t.Fatalf("SendAudio: %v", err)
	}
	if err := sess.SendAudio(nil); err != nil {
		t.Errorf("SendAudio(空帧) 应防御性忽略, got %v", err)
	}

	// 首条 = session.update（Dial 内下发），第二条 = input_audio_buffer.append
	expect := []struct {
		typ   string
		audio []byte
	}{{"session.update", nil}, {"input_audio_buffer.append", pcm}}
	for i, w := range expect {
		select {
		case raw := <-msgs:
			var m struct {
				Type  string `json:"type"`
				Audio string `json:"audio"`
			}
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatalf("解析服务端消息[%d]: %v", i, err)
			}
			if m.Type != w.typ {
				t.Errorf("消息[%d] type = %q, want %q", i, m.Type, w.typ)
			}
			if w.audio == nil {
				continue
			}
			got, err := base64.StdEncoding.DecodeString(m.Audio)
			if err != nil {
				t.Fatalf("消息[%d] audio base64 解码: %v", i, err)
			}
			if !bytes.Equal(got, w.audio) {
				t.Errorf("消息[%d] audio = %v, want %v（base64 PCM 往返应一致）", i, got, w.audio)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("未收到服务端消息[%d]（%s）", i, w.typ)
		}
	}
}

func TestOpenAISession_SendAudioBeforeDialFails(t *testing.T) {
	s, err := New("openai", Config{APIKey: "sk-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.SendAudio([]byte{0x01}); err == nil {
		t.Fatal("未 Dial 即 SendAudio 应报错")
	}
}

// ── 事件解析 ──

func TestOpenAISession_EventParsing(t *testing.T) {
	srv, conns, _, _ := fakeServer(t)
	sess := dialTestSession(t, Config{BaseURL: srv.URL, APIKey: "sk-test"})

	audioPCM := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	serverEvents := []string{
		`{"type":"session.created","session":{"id":"s1"}}`,
		`{"type":"input_audio_buffer.speech_started"}`,
		`{"type":"input_audio_buffer.speech_stopped"}`,
		`{"type":"response.text.delta","delta":"你好"}`,
		`{"type":"response.audio.delta","delta":"` + base64.StdEncoding.EncodeToString(audioPCM) + `"}`,
		`{"type":"response.audio.done"}`,
		`{"type":"error","error":{"message":"boom"}}`,
		`{"type":"future.unknown_event","x":1}`, // 未知事件 → Unknown 容错
		`not-json-at-all`,                       // 非 JSON 也不崩
	}
	conn := <-conns
	for _, e := range serverEvents {
		_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, []byte(e)); err != nil {
			t.Fatalf("服务端下发事件失败: %v", err)
		}
	}

	want := []struct {
		typ  string
		pcm  []byte
		raw  string
	}{
		{EventSessionCreated, nil, serverEvents[0]},
		{EventInputAudioBufferSpeechStarted, nil, serverEvents[1]},
		{EventInputAudioBufferSpeechStopped, nil, serverEvents[2]},
		{EventResponseTextDelta, nil, serverEvents[3]},
		{EventResponseAudioDelta, audioPCM, serverEvents[4]},
		{EventResponseAudioDone, nil, serverEvents[5]},
		{EventError, nil, serverEvents[6]},
		{EventUnknown, nil, serverEvents[7]},
		{EventUnknown, nil, serverEvents[8]},
	}
	for i, w := range want {
		ev := readEvent(t, sess.Events())
		if ev.Type != w.typ {
			t.Errorf("事件[%d] type = %q, want %q", i, ev.Type, w.typ)
		}
		if w.pcm != nil && !bytes.Equal(ev.AudioPCM, w.pcm) {
			t.Errorf("事件[%d] AudioPCM = %v, want %v（audio.delta 应解码 PCM）", i, ev.AudioPCM, w.pcm)
		}
		if w.pcm == nil && len(ev.AudioPCM) != 0 {
			t.Errorf("事件[%d] AudioPCM 应为 nil, got %v", i, ev.AudioPCM)
		}
		if ev.DataJSON != w.raw {
			t.Errorf("事件[%d] DataJSON = %q, want 原始 JSON %q", i, ev.DataJSON, w.raw)
		}
	}
}

// ── Close ──

func TestOpenAISession_CloseIdempotent(t *testing.T) {
	srv, _, _, _ := fakeServer(t)
	sess := dialTestSession(t, Config{BaseURL: srv.URL, APIKey: "sk-test"})

	if err := sess.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := sess.Close(); err != nil {
		t.Errorf("Close 应幂等, got %v", err)
	}
	if err := sess.SendAudio([]byte{0x01}); err == nil {
		t.Error("关闭后 SendAudio 应报错")
	}

	// 读循环退出后事件通道应关闭（消费方可干净收尾）
	select {
	case _, ok := <-sess.Events():
		if ok {
			t.Error("Close 后事件通道应关闭且为空")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close 后事件通道未关闭")
	}
}

func TestOpenAISession_CloseWithoutDial(t *testing.T) {
	s, err := New("openai", Config{APIKey: "sk-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("未 Dial 直接 Close 应合法, got %v", err)
	}
}
