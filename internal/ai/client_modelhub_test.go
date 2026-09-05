package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

// MH1（蒸馏 unsloth §三/§七）请求构造回归：modelhub 引擎采样让位 + 思考默认关。
//httptest 服务器扮演 Studio /v1/chat/completions，捕获请求体逐字段断言。

// newModelHubTestClient 构造 engineMgr 就绪的 Client：modelhub（与对照引擎）
// 都指到同一个捕获服务器。返回捕获到的最近请求体 map。
func newModelHubTestClient(t *testing.T, engineID string) (*Client, func() map[string]any) {
	t.Helper()
	var mu sync.Mutex
	got := map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = map[string]any{}
		_ = json.Unmarshal(body, &got)
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"好\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(srv.Close)

	mgr := modelengine.NewManager("", "")
	for _, id := range []string{"modelhub", "herdsman"} {
		if err := mgr.SaveEngine(modelengine.EngineConfig{ID: id, BaseURL: srv.URL + "/v1", Enabled: true}); err != nil {
			t.Fatalf("SaveEngine(%s): %v", id, err)
		}
	}

	c, _ := newTestClient(t, http.NotFoundHandler())
	c.SetEngineManager(mgr)
	return c, func() map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return got
	}
}

// drainChunks 消费完 SSE 通道并确认无流错误。
func drainChunks(t *testing.T, chunks <-chan SSEChunk, cancel context.CancelFunc) {
	t.Helper()
	for ch := range chunks {
		if ch.Error != "" {
			t.Fatalf("stream error: %s", ch.Error)
		}
	}
	cancel()
}

func TestChatStreamChunks_ModelHubOmitsTemperatureAndDisablesThinking(t *testing.T) {
	c, snapshot := newModelHubTestClient(t, "modelhub")
	chunks, cancel, err := c.ChatStreamChunks(context.Background(), "", "sys", "hi", ChatSimpleOptions{EngineID: "modelhub"})
	if err != nil {
		t.Fatalf("ChatStreamChunks: %v", err)
	}
	drainChunks(t, chunks, cancel)

	got := snapshot()
	if _, present := got["temperature"]; present {
		t.Errorf("modelhub 未显式配置时应省略 temperature（让位 Studio 自动调优），实际 %v", got["temperature"])
	}
	kwargs, _ := got["chat_template_kwargs"].(map[string]any)
	if kwargs == nil || kwargs["enable_thinking"] != false {
		t.Errorf("modelhub 默认应显式 chat_template_kwargs.enable_thinking=false，实际 %v", got["chat_template_kwargs"])
	}
	if _, present := got["enable_thinking"]; present {
		t.Error("modelhub 不应发顶层 enable_thinking（实测唯一有效通道是 chat_template_kwargs）")
	}
}

func TestChatStreamChunks_ModelHubExplicitTemperatureAndThinkingPassThrough(t *testing.T) {
	c, snapshot := newModelHubTestClient(t, "modelhub")
	chunks, cancel, err := c.ChatStreamChunks(context.Background(), "", "sys", "hi", ChatSimpleOptions{
		EngineID: "modelhub", Temperature: 0.3, MaxTokens: 1024, EnableThinking: true,
	})
	if err != nil {
		t.Fatalf("ChatStreamChunks: %v", err)
	}
	drainChunks(t, chunks, cancel)

	got := snapshot()
	if v, _ := got["temperature"].(float64); v != 0.3 {
		t.Errorf("显式 temperature 应透传，实际 %v", got["temperature"])
	}
	kwargs, _ := got["chat_template_kwargs"].(map[string]any)
	if kwargs == nil || kwargs["enable_thinking"] != true {
		t.Errorf("显式开思考应传 enable_thinking=true，实际 %v", got["chat_template_kwargs"])
	}
	if v, _ := got["max_tokens"].(float64); v < 4096 {
		t.Errorf("开思考时小预算应抬到 4096，实际 %v", got["max_tokens"])
	}
}

func TestChatStreamChunks_NonModelHubKeepsDefaultTemperature(t *testing.T) {
	c, snapshot := newModelHubTestClient(t, "herdsman")
	chunks, cancel, err := c.ChatStreamChunks(context.Background(), "", "sys", "hi", ChatSimpleOptions{EngineID: "herdsman"})
	if err != nil {
		t.Fatalf("ChatStreamChunks: %v", err)
	}
	drainChunks(t, chunks, cancel)

	got := snapshot()
	if v, _ := got["temperature"].(float64); v != 0.7 {
		t.Errorf("非 modelhub 引擎应维持 0.7 兜底，实际 %v", got["temperature"])
	}
	if _, present := got["chat_template_kwargs"]; present {
		t.Errorf("未开思考时不应携带 chat_template_kwargs，实际 %v", got["chat_template_kwargs"])
	}
}
