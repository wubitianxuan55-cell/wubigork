package bridge

import (
	"context"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// ── LLM seam（3.0 Step 3b）─────────────────────────────────────

// TestBridge_ImplementsLLMProvider 编译期+运行时断言：bridge Provider 满足
// LLM seam 定义接口 provider.LLMProvider（Stream + Chat）。
func TestBridge_ImplementsLLMProvider(t *testing.T) {
	var _ provider.LLMProvider = (*Provider)(nil) // 编译期断言
	mc := &mockClient{chunks: []ai.SSEChunk{{Done: true}}}
	p := &Provider{name: "gaea", model: "m", client: mc}
	if _, ok := interface{}(p).(provider.LLMProvider); !ok {
		t.Fatal("bridge Provider 必须满足 provider.LLMProvider")
	}
}

// TestBridge_Chat_Aggregates 验证 Chat（一次性补全）与 Stream 同一条转换路径：
// 文本/思考链/工具调用/用量全部聚合进 Completion。
func TestBridge_Chat_Aggregates(t *testing.T) {
	mc := &mockClient{chunks: []ai.SSEChunk{
		{Reasoning: "先查表"},
		{Content: "限值 60"},
		{Done: true, Usage: &ai.ChatUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120}},
	}}
	SetClient(mc)
	p := &Provider{name: "gaea", model: "m", client: mc}

	c, err := p.Chat(context.Background(), provider.Request{
		Messages: []provider.Message{{Role: provider.RoleUser, Content: "砷限值？"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if c.Content != "限值 60" {
		t.Errorf("Content = %q, want 限值 60", c.Content)
	}
	if c.ReasoningContent != "先查表" {
		t.Errorf("ReasoningContent = %q, want 先查表", c.ReasoningContent)
	}
	if c.Usage == nil || c.Usage.TotalTokens != 120 {
		t.Errorf("Usage = %+v, want TotalTokens=120", c.Usage)
	}
	// 工具调用聚合
	mc2 := &mockClient{chunks: []ai.SSEChunk{{
		Done: true,
		ToolCalls: []ai.ChatToolCall{{
			ID: "c1", Type: "function",
			Function: ai.ChatToolFunction{Name: "spec_query", Arguments: "{\"q\":\"砷\"}"},
		}},
	}}}
	SetClient(mc2)
	p2 := &Provider{name: "gaea", model: "m", client: mc2}
	c2, err := p2.Chat(context.Background(), provider.Request{})
	if err != nil {
		t.Fatalf("Chat(tools): %v", err)
	}
	if len(c2.ToolCalls) != 1 || c2.ToolCalls[0].ID != "c1" || c2.ToolCalls[0].Name != "spec_query" {
		t.Errorf("ToolCalls = %+v, want c1/spec_query", c2.ToolCalls)
	}
}

// TestBridge_Chat_StreamError 验证 Chat 透传流内错误（与 Stream 的 ChunkError 一致）。
func TestBridge_Chat_StreamError(t *testing.T) {
	mc := &mockClient{chunks: []ai.SSEChunk{{Error: "boom"}}}
	SetClient(mc)
	p := &Provider{name: "gaea", model: "m", client: mc}
	if _, err := p.Chat(context.Background(), provider.Request{}); err == nil || err.Error() != "boom" {
		t.Fatalf("Chat error = %v, want boom", err)
	}
}

// ── 现有按引擎分支行为以测试固化（seam 验收硬要求） ──────────
// bridge.go Stream 的 "p.engine == herdsman || p.engine == ollama" 分支：
// 本地引擎强制开启 thinking 并守护 max_tokens >= 4096。herdsman 已有测试
// （TestBridge_Stream_Reasoning / TestBridge_Stream_ThinkingMaxTokensGuard），
// 此处补 ollama（同一分支）与空引擎（不触发）以钉死整条分支行为。

// TestBridge_Stream_OllamaEnablesThinking 冻结 ollama 与 herdsman 同一分支：
// EnableThinking=true + chat_template_kwargs + 小预算抬到 4096。
func TestBridge_Stream_OllamaEnablesThinking(t *testing.T) {
	mc := &mockClient{chunks: []ai.SSEChunk{{Done: true}}}
	SetClient(mc)
	p := &Provider{name: "gaea", model: "m", engine: "ollama", client: mc}
	if _, err := p.Stream(context.Background(), provider.Request{
		Messages:  []provider.Message{{Role: provider.RoleUser, Content: "hi"}},
		MaxTokens: 1024,
	}); err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if mc.gotReq == nil {
		t.Fatal("client 未收到请求")
	}
	if mc.gotReq.EnableThinking == nil || !*mc.gotReq.EnableThinking {
		t.Errorf("ollama 应开启 enable_thinking: %+v", mc.gotReq.EnableThinking)
	}
	if v, _ := mc.gotReq.ChatTemplateKwargs["enable_thinking"].(bool); !v {
		t.Errorf("ollama 应携带 chat_template_kwargs.enable_thinking: %+v", mc.gotReq.ChatTemplateKwargs)
	}
	if mc.gotReq.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096（本地引擎思考模式守护）", mc.gotReq.MaxTokens)
	}
}

// TestBridge_Stream_EmptyEngineNoThinking 冻结空引擎分支：无引擎（跟随全局
// 活跃引擎）时不强制 thinking，MaxTokens 原样透传。
func TestBridge_Stream_EmptyEngineNoThinking(t *testing.T) {
	mc := &mockClient{chunks: []ai.SSEChunk{{Done: true}}}
	SetClient(mc)
	p := &Provider{name: "gaea", model: "m", engine: "", client: mc}
	if _, err := p.Stream(context.Background(), provider.Request{
		Messages:  []provider.Message{{Role: provider.RoleUser, Content: "hi"}},
		MaxTokens: 512,
	}); err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if mc.gotReq == nil {
		t.Fatal("client 未收到请求")
	}
	if mc.gotReq.EnableThinking != nil {
		t.Errorf("空引擎不应强制 enable_thinking: %+v", mc.gotReq.EnableThinking)
	}
	if mc.gotReq.MaxTokens != 512 {
		t.Errorf("MaxTokens = %d, want 512（空引擎不守护）", mc.gotReq.MaxTokens)
	}
}
