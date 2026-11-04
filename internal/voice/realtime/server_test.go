// server_test.go — 协议层验收：握手/会话池上限/回声链序/宽容解码/音频上行。
package realtime

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func testServer(t *testing.T, h Handler) *Server {
	t.Helper()
	srv := &Server{Addr: "127.0.0.1:0", Handler: h}
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(srv.Stop)
	return srv
}

func dial(t *testing.T, srv *Server) *websocket.Conn {
	t.Helper()
	u := url.URL{Scheme: "ws", Host: srv.Addr, Path: "/v1/realtime"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// recvUntil 帧收集器：读到 until 返回命中帧，超时/EOF 报错。
func recvUntil(t *testing.T, conn *websocket.Conn, until func(map[string]any) bool, max time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var ev map[string]any
		if err := json.Unmarshal(raw, &ev); err != nil {
			t.Fatalf("bad frame: %v", err)
		}
		if until(ev) {
			return ev
		}
	}
	t.Fatal("recvUntil 超时")
	return nil
}

func collect(t *testing.T, conn *websocket.Conn, n int, max time.Duration) []map[string]any {
	t.Helper()
	var out []map[string]any
	deadline := time.Now().Add(max)
	for len(out) < n {
		_ = conn.SetReadDeadline(deadline)
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read (%d/%d): %v", len(out), n, err)
		}
		var ev map[string]any
		if err := json.Unmarshal(raw, &ev); err != nil {
			t.Fatalf("bad frame: %v", err)
		}
		out = append(out, ev)
	}
	return out
}

func mustType(t *testing.T, ev map[string]any) string {
	t.Helper()
	typ, _ := ev["type"].(string)
	if typ == "" {
		t.Fatalf("帧缺 type: %v", ev)
	}
	return typ
}

// recordingHandler 记录下行事件供断言。
type recordingHandler struct {
	audioFrames int
	audioBytes  int
	configs     int
	creates     int
	cancels     int
	ends        int
}

func (r *recordingHandler) OnAudio(_ *Session, pcm []byte) {
	r.audioFrames++
	r.audioBytes += len(pcm)
}
func (r *recordingHandler) OnSessionConfig(*Session, SessionConfig) { r.configs++ }
func (r *recordingHandler) OnResponseCreate(*Session)               { r.creates++ }
func (r *recordingHandler) OnResponseCancel(*Session)               { r.cancels++ }
func (r *recordingHandler) OnSessionEnd(*Session)                   { r.ends++ }

func TestHandshakeAndSessionUpdate(t *testing.T) {
	h := &recordingHandler{}
	srv := testServer(t, h)
	conn := dial(t, srv)

	// ① 连接即收 session.created（含 session id）
	created := recvUntil(t, conn, func(ev map[string]any) bool {
		return mustType(t, ev) == "session.created"
	}, 2*time.Second)
	sess, _ := created["session"].(map[string]any)
	if sess == nil || !strings.HasPrefix(sess["id"].(string), "session_") {
		t.Fatalf("session.created 形状异常: %v", created)
	}

	// ② session.update → session.updated（合并 threshold/voice/instructions）
	th := 0.75
	tr := true
	upd, _ := json.Marshal(map[string]any{
		"type":         "realtime",
		"instructions": "你是 gaea",
		"audio": map[string]any{
			"input": map[string]any{
				"turn_detection": map[string]any{"type": "server_vad", "threshold": th, "interrupt_response": tr},
			},
			"output": map[string]any{"voice": "char_gaea"},
		},
	})
	_ = conn.WriteJSON(map[string]any{"type": "session.update", "session": json.RawMessage(upd)})
	updated := recvUntil(t, conn, func(ev map[string]any) bool {
		return mustType(t, ev) == "session.updated"
	}, 2*time.Second)
	us, _ := updated["session"].(map[string]any)
	ua, _ := us["audio"].(map[string]any)
	ui, _ := ua["input"].(map[string]any)
	ud, _ := ui["turn_detection"].(map[string]any)
	if ud["threshold"].(float64) != th || ud["interrupt_response"] != true {
		t.Fatalf("session.updated 未合并 VAD 配置: %v", ud)
	}
	if ua["output"].(map[string]any)["voice"] != "char_gaea" {
		t.Fatalf("session.updated 未合并 voice: %v", ua)
	}
	if h.configs != 2 { // 连接初始 1 次 + session.update 1 次
		t.Fatalf("OnSessionConfig 回调次数=%d", h.configs)
	}

	// ③ 未知类型宽容（无 error 帧）；④ 坏 JSON → error 帧不断连
	_ = conn.WriteJSON(map[string]any{"type": "gaea.extension.custom", "x": 1})
	_ = conn.WriteMessage(websocket.TextMessage, []byte("{not json"))
	errEv := recvUntil(t, conn, func(ev map[string]any) bool {
		return mustType(t, ev) == "error"
	}, 2*time.Second)
	msg, _ := errEv["error"].(map[string]any)["message"].(string)
	if !strings.Contains(msg, "invalid") {
		t.Fatalf("error 帧内容异常: %v", errEv)
	}
	// 连接仍活：再发 append 应到 Handler
	payload := make([]byte, 320)
	_ = conn.WriteJSON(map[string]any{"type": "input_audio_buffer.append", "audio": base64.StdEncoding.EncodeToString(payload)})
	deadline := time.Now().Add(2 * time.Second)
	for h.audioFrames == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if h.audioFrames != 1 || h.audioBytes != len(payload) {
		t.Fatalf("音频上行丢失: frames=%d bytes=%d", h.audioFrames, h.audioBytes)
	}
}

func TestSessionLimitReached(t *testing.T) {
	srv := testServer(t, ProtocolEcho{})
	conn1 := dial(t, srv)
	recvUntil(t, conn1, func(ev map[string]any) bool { return mustType(t, ev) == "session.created" }, 2*time.Second)

	conn2, _, err := websocket.DefaultDialer.Dial("ws://"+srv.Addr, nil)
	if err != nil {
		// 升级后立即断开也可能表现为 dial 错误（close 1008）
		return
	}
	defer func() { _ = conn2.Close() }()
	ev := recvUntil(t, conn2, func(e map[string]any) bool { return mustType(t, e) == "error" }, 2*time.Second)
	msg, _ := ev["error"].(map[string]any)["message"].(string)
	if msg != "session_limit_reached" {
		t.Fatalf("第二条连接应收到 session_limit_reached: %v", ev)
	}
}

func TestEchoResponseChainOrder(t *testing.T) {
	srv := testServer(t, ProtocolEcho{})
	conn := dial(t, srv)
	recvUntil(t, conn, func(ev map[string]any) bool { return mustType(t, ev) == "session.created" }, 2*time.Second)

	_ = conn.WriteJSON(map[string]any{"type": "response.create"})
	// 回声链 10 帧
	frames := collect(t, conn, 10, 3*time.Second)
	wantOrder := []string{
		"response.created",
		"response.output_item.added",
		"response.content_part.added",
		"response.output_audio_transcript.delta",
		"response.output_audio_transcript.done",
		"response.output_audio.delta",
		"response.output_audio.done",
		"response.content_part.done",
		"response.output_item.done",
		"response.done",
	}
	var respID, itemID string
	for i, ev := range frames {
		if mustType(t, ev) != wantOrder[i] {
			t.Fatalf("链序断裂 @%d: got %s want %s", i, mustType(t, ev), wantOrder[i])
		}
		switch wantOrder[i] {
		case "response.created":
			resp, _ := ev["response"].(map[string]any)
			respID, _ = resp["id"].(string)
			if !strings.HasPrefix(respID, "resp_") {
				t.Fatalf("response.id 异常: %v", respID)
			}
		case "response.output_audio.delta":
			if ev["response_id"] != respID {
				t.Fatalf("audio delta response_id 不一致: %v", ev["response_id"])
			}
			itemID, _ = ev["item_id"].(string)
			// 100ms 静音 = 3200 字节 PCM16
			b64, _ := ev["delta"].(string)
			raw, err := base64.StdEncoding.DecodeString(b64)
			if err != nil || len(raw) != 3200 {
				t.Fatalf("audio delta 载荷异常: %d bytes err=%v", len(raw), err)
			}
		case "response.done":
			resp, _ := ev["response"].(map[string]any)
			if resp["id"] != respID || resp["status"] != "completed" {
				t.Fatalf("response.done 不一致: %v", resp)
			}
		}
	}
	if !strings.HasPrefix(itemID, "item_") {
		t.Fatalf("item_id 缺失: %q", itemID)
	}
}

func TestAudioAppendRoundtrip(t *testing.T) {
	h := &recordingHandler{}
	srv := testServer(t, h)
	conn := dial(t, srv)
	recvUntil(t, conn, func(ev map[string]any) bool { return mustType(t, ev) == "session.created" }, 2*time.Second)

	raw := make([]byte, 640) // 320 样本 = 20ms @16k
	_ = conn.WriteJSON(map[string]any{"type": "input_audio_buffer.append", "audio": base64.StdEncoding.EncodeToString(raw)})
	_ = conn.WriteJSON(map[string]any{"type": "input_audio_buffer.append", "audio": "###bad###"})
	ev := recvUntil(t, conn, func(e map[string]any) bool { return mustType(t, e) == "error" }, 2*time.Second)
	msg, _ := ev["error"].(map[string]any)["message"].(string)
	if !strings.Contains(msg, "base64") {
		t.Fatalf("坏 base64 应报 error: %v", ev)
	}
	deadline := time.Now().Add(2 * time.Second)
	for h.audioFrames == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if h.audioFrames != 1 || h.audioBytes != len(raw) {
		t.Fatalf("上行音频不符: frames=%d bytes=%d want 1/%d", h.audioFrames, h.audioBytes, len(raw))
	}
}
