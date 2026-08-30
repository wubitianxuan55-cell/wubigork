// Package realtime — S2 轮次控制 / session.update 扩展 / 新增事件常量测试。
package realtime

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// readServerMsg 带超时读取一条服务端收到的客户端消息并解析 type。
func readServerMsg(t *testing.T, msgs <-chan []byte) (msgType string, raw []byte) {
	t.Helper()
	select {
	case raw = <-msgs:
	case <-time.After(2 * time.Second):
		t.Fatal("等待服务端消息超时")
	}
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		t.Fatalf("解析服务端消息 %q: %v", raw, err)
	}
	return head.Type, raw
}

// TestOpenAISession_TurnControlFrames TurnControl 四动作协议帧：
// commit / clear / create / cancel 逐帧下发且载荷精确。
func TestOpenAISession_TurnControlFrames(t *testing.T) {
	srv, _, msgs, _ := fakeServer(t)
	sess := dialTestSession(t, Config{BaseURL: srv.URL, APIKey: "sk-test"})

	// 首条 session.update 吃掉
	if mt, _ := readServerMsg(t, msgs); mt != "session.update" {
		t.Fatalf("首条消息 = %q, want session.update", mt)
	}

	if err := sess.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := sess.ClearBuffer(); err != nil {
		t.Fatalf("ClearBuffer: %v", err)
	}
	if err := sess.CreateResponse(); err != nil {
		t.Fatalf("CreateResponse: %v", err)
	}
	if err := sess.CancelResponse(); err != nil {
		t.Fatalf("CancelResponse: %v", err)
	}

	want := []string{
		"input_audio_buffer.commit",
		"input_audio_buffer.clear",
		"response.create",
		"response.cancel",
	}
	for _, w := range want {
		if mt, _ := readServerMsg(t, msgs); mt != w {
			t.Errorf("协议帧 type = %q, want %q", mt, w)
		}
	}
}

// TestOpenAISession_SessionUpdateTurnDetection session.update 扩展字段：
// turn_detection = {server_vad, create_response, interrupt_response} +
// output_audio_format = pcm16（显式防漂移）。
func TestOpenAISession_SessionUpdateTurnDetection(t *testing.T) {
	srv, _, msgs, _ := fakeServer(t)
	dialTestSession(t, Config{BaseURL: srv.URL, APIKey: "sk-test"})

	_, raw := readServerMsg(t, msgs)
	var upd struct {
		Session struct {
			InputAudioFormat  string `json:"input_audio_format"`
			OutputAudioFormat string `json:"output_audio_format"`
			TurnDetection     *struct {
				Type              string `json:"type"`
				CreateResponse    bool   `json:"create_response"`
				InterruptResponse bool   `json:"interrupt_response"`
			} `json:"turn_detection"`
		} `json:"session"`
	}
	if err := json.Unmarshal(raw, &upd); err != nil {
		t.Fatalf("解析 session.update: %v", err)
	}
	if upd.Session.InputAudioFormat != "pcm16" {
		t.Errorf("input_audio_format = %q, want pcm16（协议值不变）", upd.Session.InputAudioFormat)
	}
	if upd.Session.OutputAudioFormat != "pcm16" {
		t.Errorf("output_audio_format = %q, want pcm16（显式防漂移）", upd.Session.OutputAudioFormat)
	}
	td := upd.Session.TurnDetection
	if td == nil {
		t.Fatal("turn_detection 缺失, want server_vad 配置")
	}
	if td.Type != "server_vad" {
		t.Errorf("turn_detection.type = %q, want server_vad", td.Type)
	}
	if !td.CreateResponse || !td.InterruptResponse {
		t.Errorf("turn_detection = %+v, want create_response/interrupt_response 均 true", td)
	}
}

// TestOpenAISession_S2EventConstants S2 新增 7 个事件常量 → 白名单映射
//（解析骨架零改动：映射后原样透传）。
func TestOpenAISession_S2EventConstants(t *testing.T) {
	srv, conns, _, _ := fakeServer(t)
	sess := dialTestSession(t, Config{BaseURL: srv.URL, APIKey: "sk-test"})

	audioPCM := []byte{0x10, 0x20, 0x30}
	serverEvents := []string{
		`{"type":"response.done","response":{"status":"completed"}}`,
		`{"type":"response.created"}`,
		`{"type":"response.audio_transcript.delta","delta":"你"}`,
		`{"type":"response.audio_transcript.done","transcript":"你好"}`,
		`{"type":"conversation.item.input_audio_transcription.completed","transcript":"用户说的话"}`,
		`{"type":"conversation.item.input_audio_transcription.failed","error":{"message":"x"}}`,
		`{"type":"input_audio_buffer.committed"}`,
	}
	want := []string{
		EventResponseDone,
		EventResponseCreated,
		EventResponseAudioTranscriptDelta,
		EventResponseAudioTranscriptDone,
		EventInputAudioTranscriptionCompleted,
		EventInputAudioTranscriptionFailed,
		EventInputAudioBufferCommitted,
	}

	conn := <-conns
	for _, e := range serverEvents {
		_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, []byte(e)); err != nil {
			t.Fatalf("服务端下发事件失败: %v", err)
		}
	}

	for i, w := range want {
		ev := readEvent(t, sess.Events())
		if ev.Type != w {
			t.Errorf("事件[%d] type = %q, want %q", i, ev.Type, w)
		}
		if ev.DataJSON != serverEvents[i] {
			t.Errorf("事件[%d] DataJSON = %q, want 原始 JSON 透传", i, ev.DataJSON)
		}
	}

	// audio.delta 的 PCM 解码与既有路径一致（S2 白名单不影响解码骨架）
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	delta := `{"type":"response.audio.delta","delta":"` + base64.StdEncoding.EncodeToString(audioPCM) + `"}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(delta)); err != nil {
		t.Fatalf("下发 audio.delta 失败: %v", err)
	}
	ev := readEvent(t, sess.Events())
	if ev.Type != EventResponseAudioDelta || string(ev.AudioPCM) != string(audioPCM) {
		t.Errorf("audio.delta 解码 = (%q, %v), want (%q, %v)", ev.Type, ev.AudioPCM, EventResponseAudioDelta, audioPCM)
	}
}
