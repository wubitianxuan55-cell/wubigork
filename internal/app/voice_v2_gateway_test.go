package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/whisper"
)

func TestVoiceV2GatewayInjectsModelAndStreamsUpstream(t *testing.T) {
	oldUpstream := voiceV2GatewayUpstream
	t.Cleanup(func() { voiceV2GatewayUpstream = oldUpstream })

	var got map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("upstream path = %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read upstream body: %v", err)
			return
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("decode upstream body: %v", err)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"你好\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()
	voiceV2GatewayUpstream = upstream.URL

	a := &mediaState{core: &core{}}
	body := `{"model":"sidecar-model","stream":true,"messages":[` +
		`{"role":"system","content":"你是短句语音助手"},` +
		`{"role":"user","content":"你好"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	rr := httptest.NewRecorder()
	a.handleVoiceV2ChatCompletion(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if got["model"] != voiceV2LLMName {
		t.Fatalf("upstream model = %v, want %s", got["model"], voiceV2LLMName)
	}
	if !strings.Contains(rr.Body.String(), "\"content\":\"你好\"") ||
		!strings.Contains(rr.Body.String(), "data: [DONE]") {
		t.Fatalf("stream body 未完整转发: %q", rr.Body.String())
	}
}

func TestVoiceV2GatewayStripsToolsPayload(t *testing.T) {
	// P1 网关剥 tools：对话/动作平面分离（架构定稿）——sidecar 将来启用工具
	// 透传时，工具定义与选择不得进语音 4B 的 prompt。
	oldUpstream := voiceV2GatewayUpstream
	t.Cleanup(func() { voiceV2GatewayUpstream = oldUpstream })

	var got map[string]interface{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("decode upstream body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"好\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()
	voiceV2GatewayUpstream = upstream.URL

	a := &mediaState{core: &core{}}
	body := `{"model":"m","stream":true,"tools":[{"type":"function","function":{"name":"gen_image","parameters":{}}}],"tool_choice":"auto","messages":[{"role":"user","content":"画一张图"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	rr := httptest.NewRecorder()
	a.handleVoiceV2ChatCompletion(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if _, ok := got["tools"]; ok {
		t.Fatalf("tools 应被网关剥除: %v", got["tools"])
	}
	if _, ok := got["tool_choice"]; ok {
		t.Fatalf("tool_choice 应被网关剥除: %v", got["tool_choice"])
	}
}

func TestInjectVoiceV2SystemContextPreservesTurnOrder(t *testing.T) {
	payload := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": "人格"},
			map[string]interface{}{"role": "user", "content": "第一问"},
			map[string]interface{}{"role": "assistant", "content": "第一答"},
			map[string]interface{}{"role": "user", "content": "第二问"},
		},
	}
	injectVoiceV2SystemContext(payload, "【Gaea 长期记忆】\n- 用户喜欢简短回答")
	messages := payload["messages"].([]interface{})
	if len(messages) != 4 {
		t.Fatalf("messages len = %d, want 4", len(messages))
	}
	system := messages[0].(map[string]interface{})["content"].(string)
	if !strings.Contains(system, "人格") || !strings.Contains(system, "用户喜欢简短回答") {
		t.Fatalf("system 未合并人格与记忆: %q", system)
	}
	for i, role := range []string{"system", "user", "assistant", "user"} {
		if got := messages[i].(map[string]interface{})["role"]; got != role {
			t.Fatalf("messages[%d].role = %v, want %s", i, got, role)
		}
	}
}

func TestInjectVoiceV2SystemContextPrependsWhenMissing(t *testing.T) {
	payload := map[string]interface{}{
		"messages": []interface{}{map[string]interface{}{"role": "user", "content": "你好"}},
	}
	injectVoiceV2SystemContext(payload, "记忆")
	messages := payload["messages"].([]interface{})
	if len(messages) != 2 || messages[0].(map[string]interface{})["role"] != "system" {
		t.Fatalf("缺 system 时应前置: %#v", messages)
	}
}

func TestApplyVoiceV2PreparedPromptInjectsStableAndDynamicContext(t *testing.T) {
	payload := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": "sidecar 基础指令"},
			map[string]interface{}{"role": "user", "content": "今天有点累"},
		},
	}
	prepared := &voiceV2PreparedTurn{pre: whisper.PreLLMResult{
		SystemPrompt:   "PERSONA_STABLE",
		DynamicContext: "EMOTION_CURRENT",
	}}
	applyVoiceV2PreparedPrompt(payload, prepared)

	// P1 缓存命中刀契约：system 合并稳定人格段；动态段独立成消息插在末条
	// user 之前；user 消息本体逐字节不动（下一轮 sidecar 历史回放原文，
	// 公共前缀=[system+全部历史] 才能被引擎前缀缓存复用）。
	messages := payload["messages"].([]interface{})
	if len(messages) != 3 {
		t.Fatalf("动态段应独立成消息（共 3 条），got %d", len(messages))
	}
	system := messages[0].(map[string]interface{})["content"].(string)
	if !strings.Contains(system, "sidecar 基础指令") || !strings.Contains(system, "PERSONA_STABLE") {
		t.Fatalf("稳定人格上下文未合并: %q", system)
	}
	dyn := messages[1].(map[string]interface{})
	if dyn["role"] != "system" || !strings.Contains(dyn["content"].(string), "EMOTION_CURRENT") {
		t.Fatalf("动态上下文应独立 system 消息: %#v", dyn)
	}
	user := messages[2].(map[string]interface{})
	if user["content"] != "今天有点累" {
		t.Fatalf("user 消息本体不得被改写: %q", user["content"])
	}
}

func TestVoiceV2GatewayRejectsNonPost(t *testing.T) {
	a := &mediaState{core: &core{}}
	rr := httptest.NewRecorder()
	a.handleVoiceV2ChatCompletion(rr, httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

func TestVoiceV2LongTermContextUsesActivePersonalityMemory(t *testing.T) {
	fs := whisper.NewFactStore()
	fs.Add(whisper.MemoryFact{
		Summary:       "用户希望语音回答保持简短",
		Weight:        1.8,
		Confidence:    0.95,
		SelfRelevance: 1,
		Status:        "active",
	})
	orch := whisper.NewOrchestrator("whisper_voice_gateway_test", whisper.PersonalityPresets[0])
	orch.FactStore = fs
	sessionID := "whisper_voice_gateway_test"
	whisperSessionsMu.Lock()
	old, hadOld := whisperSessions[sessionID]
	whisperSessions[sessionID] = orch
	whisperSessionsMu.Unlock()
	t.Cleanup(func() {
		whisperSessionsMu.Lock()
		if hadOld {
			whisperSessions[sessionID] = old
		} else {
			delete(whisperSessions, sessionID)
		}
		whisperSessionsMu.Unlock()
	})

	a := &mediaState{
		core: &core{},
		app:  &App{whisperState: &whisperState{core: &core{}}},
	}
	a.setVoicePersonality("voice_gateway_test")
	memory := a.voiceV2LongTermContext("今天想听点轻松的")
	if !strings.Contains(memory, "Gaea 长期记忆") || !strings.Contains(memory, "保持简短") {
		t.Fatalf("未注入活跃人格长期记忆: %q", memory)
	}
	a.setVoicePersonality("plain")
	if got := a.voiceV2LongTermContext("你好"); got != "" {
		t.Fatalf("plain 模式不应注入人格记忆: %q", got)
	}
}

func TestVoiceV2GatewayUpstreamFailureRollsBackPreparedTurn(t *testing.T) {
	a := newChatServiceTestApp(t)
	pid := "voice_gateway_rollback"
	sessionID := "whisper_" + pid
	whisperSessionsMu.Lock()
	delete(whisperSessions, sessionID)
	whisperSessionsMu.Unlock()
	t.Cleanup(func() {
		a.whisperState.rollbackVoiceV2Prepared()
		whisperSessionsMu.Lock()
		delete(whisperSessions, sessionID)
		whisperSessionsMu.Unlock()
	})
	a.mediaState = &mediaState{core: a.core, app: a}
	a.mediaState.setVoicePersonality(pid)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "model unavailable", http.StatusServiceUnavailable)
	}))
	defer upstream.Close()
	oldUpstream := voiceV2GatewayUpstream
	voiceV2GatewayUpstream = upstream.URL
	t.Cleanup(func() { voiceV2GatewayUpstream = oldUpstream })

	body := `{"model":"sidecar","stream":true,"messages":[` +
		`{"role":"system","content":"base"},` +
		`{"role":"user","content":"我今天有点累"}]}`
	rr := httptest.NewRecorder()
	a.mediaState.handleVoiceV2ChatCompletion(
		rr, httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body)),
	)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rr.Code)
	}
	orch := a.whisperState.getOrCreateOrch(pid)
	if got := orch.State.Counters.TotalTurns; got != 0 {
		t.Fatalf("上游失败后预推进未回滚: totalTurns=%d", got)
	}
	a.whisperState.voiceV2PrepareMu.Lock()
	pending := a.whisperState.voiceV2Prepared != nil
	a.whisperState.voiceV2PrepareMu.Unlock()
	if pending {
		t.Fatal("上游失败后仍残留 prepared turn")
	}
}

func TestVoiceV2GatewayBindsProtocolItemIDToPreparedTurn(t *testing.T) {
	a := newChatServiceTestApp(t)
	pid := "voice_gateway_item"
	sessionID := "whisper_" + pid
	whisperSessionsMu.Lock()
	delete(whisperSessions, sessionID)
	whisperSessionsMu.Unlock()
	t.Cleanup(func() {
		a.whisperState.rollbackVoiceV2Prepared()
		whisperSessionsMu.Lock()
		delete(whisperSessions, sessionID)
		whisperSessionsMu.Unlock()
	})
	a.mediaState = &mediaState{core: a.core, app: a}
	a.mediaState.setVoicePersonality(pid)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()
	oldUpstream := voiceV2GatewayUpstream
	voiceV2GatewayUpstream = upstream.URL
	t.Cleanup(func() { voiceV2GatewayUpstream = oldUpstream })

	body := `{"model":"sidecar","stream":true,"messages":[{"role":"user","content":"今天想散散步"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set(voiceV2ItemIDHeader, "msg_v2_17")
	req.Header.Set(voiceV2TurnIDHeader, "turn_v2_17")
	req.Header.Set(voiceV2TurnRevisionHeader, "2")
	rr := httptest.NewRecorder()
	a.mediaState.handleVoiceV2ChatCompletion(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}

	a.whisperState.voiceV2PrepareMu.Lock()
	prepared := a.whisperState.voiceV2Prepared
	a.whisperState.voiceV2PrepareMu.Unlock()
	if prepared == nil {
		t.Fatal("网关未保留待提交预推进回合")
	}
	if prepared.itemID != "msg_v2_17" || prepared.turnID != "turn_v2_17" || prepared.turnRevision != 2 {
		t.Fatalf("协议回合元数据未绑定: %#v", prepared)
	}
}
