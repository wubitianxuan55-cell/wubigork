package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// mockClient 实现 ai.LLMClient，记录请求并返回预置 chunks。
type mockClient struct {
	gotReq *ai.ChatRequest
	chunks []ai.SSEChunk
}

func (m *mockClient) ChatStream(ctx context.Context, req *ai.ChatRequest) (<-chan ai.SSEChunk, error) {
	m.gotReq = req
	ch := make(chan ai.SSEChunk, len(m.chunks))
	for _, c := range m.chunks {
		ch <- c
	}
	close(ch)
	return ch, nil
}
func (m *mockClient) ChatSimpleStream(ctx context.Context, model, systemPrompt, userMsg string) (string, error) {
	return "", nil
}
func (m *mockClient) ChatSimpleStreamWithOptions(ctx context.Context, model, systemPrompt, userMsg string, opts ai.ChatSimpleOptions) (string, error) {
	return "", nil
}
func (m *mockClient) GenerateImage(ctx context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	return nil, nil
}

// TestBridge_Stream_ToolCall 验证工具调用往返：provider.Message → ai.ChatMessage → ai.SSEChunk → provider.Chunk。
func TestBridge_Stream_ToolCall(t *testing.T) {
	mc := &mockClient{chunks: []ai.SSEChunk{{
		Done: true,
		ToolCalls: []ai.ChatToolCall{{
			ID:       "call_1",
			Type:     "function",
			Function: ai.ChatToolFunction{Name: "spec_query", Arguments: `{"question":"砷"}`},
		}},
	}}}
	SetClient(mc)
	p := &Provider{name: "gaea", model: "grok-3", client: mc}

	req := provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "砷超标限值是多少？"},
			{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "call_1", Name: "spec_query", Arguments: `{"question":"砷"}`}}},
			{Role: provider.RoleTool, ToolCallID: "call_1", Name: "spec_query", Content: "GB 36600: 60 mg/kg"},
		},
		Tools:       []provider.ToolSchema{{Name: "spec_query", Description: "规范查询", Parameters: json.RawMessage(`{"type":"object"}`)}},
		MaxTokens:   1024,
		Temperature: 0.3,
	}

	ch, err := p.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	var got []provider.Chunk
	for c := range ch {
		got = append(got, c)
	}

	// 验证发送给 client 的消息转换
	if mc.gotReq == nil {
		t.Fatal("client 未收到请求")
	}
	if mc.gotReq.Model != "grok-3" {
		t.Errorf("model = %q, want grok-3", mc.gotReq.Model)
	}
	if len(mc.gotReq.Messages) != 3 {
		t.Fatalf("messages = %d, want 3", len(mc.gotReq.Messages))
	}
	assistant := mc.gotReq.Messages[1]
	if len(assistant.ToolCalls) != 1 || assistant.ToolCalls[0].ID != "call_1" || assistant.ToolCalls[0].Function.Name != "spec_query" {
		t.Errorf("assistant tool_calls 转换错误: %+v", assistant.ToolCalls)
	}
	toolMsg := mc.gotReq.Messages[2]
	if toolMsg.Role != "tool" || toolMsg.ToolCallID != "call_1" || toolMsg.Name != "spec_query" {
		t.Errorf("tool 消息转换错误: %+v", toolMsg)
	}
	if len(mc.gotReq.Tools) != 1 || mc.gotReq.Tools[0].Name != "spec_query" {
		t.Errorf("tools 转换错误: %+v", mc.gotReq.Tools)
	}

	// 验证返回给 agent 的 chunk 转换：ToolCall + Done
	if len(got) != 2 {
		t.Fatalf("chunks = %d, want 2 (ToolCall + Done)", len(got))
	}
	tc := got[0]
	if tc.Type != provider.ChunkToolCall || tc.ToolCall == nil || tc.ToolCall.Name != "spec_query" || tc.ToolCall.Arguments != `{"question":"砷"}` {
		t.Errorf("工具调用 chunk 转换错误: %+v", tc)
	}
	if got[1].Type != provider.ChunkDone {
		t.Errorf("chunk1 = %+v, want ChunkDone", got[1])
	}
}

// TestBridge_Stream_TextDone 验证文本流与结束标记。

// TestBridge_Stream_TextDone 验证文本流与结束标记。
func TestBridge_Stream_TextDone(t *testing.T) {
	mc := &mockClient{chunks: []ai.SSEChunk{
		{Content: "砷"},
		{Content: "超标"},
		{Done: true},
	}}
	p := &Provider{name: "gaea", model: "m", client: mc}
	ch, err := p.Stream(context.Background(), provider.Request{Messages: []provider.Message{{Role: provider.RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	var got []provider.Chunk
	for c := range ch {
		got = append(got, c)
	}
	if len(got) != 3 {
		t.Fatalf("chunks = %d, want 3", len(got))
	}
	if got[0].Type != provider.ChunkText || got[0].Text != "砷" {
		t.Errorf("chunk0 = %+v", got[0])
	}
	if got[1].Type != provider.ChunkText || got[1].Text != "超标" {
		t.Errorf("chunk1 = %+v", got[1])
	}
	if got[2].Type != provider.ChunkDone {
		t.Errorf("chunk2 = %+v, want ChunkDone", got[2])
	}
}

// TestBridge_Stream_Error 验证错误透传为 ChunkError。
func TestBridge_Stream_Error(t *testing.T) {
	mc := &mockClient{chunks: []ai.SSEChunk{{Error: "boom"}}}
	p := &Provider{name: "gaea", model: "m", client: mc}
	ch, err := p.Stream(context.Background(), provider.Request{})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	var got []provider.Chunk
	for c := range ch {
		got = append(got, c)
	}
	if got[0].Err == nil || got[0].Err.Error() != "boom" {
		t.Errorf("error 未透传: %v", got[0].Err)
	}
}

// TestBridge_Factory 验证注册与工厂（未注入 client 时报错）。

// TestBridge_Factory 验证注册与工厂（未注入 client 时报错）。
func TestBridge_Factory(t *testing.T) {
	SetClient(nil) // 先清空包级 client，验证未注入报错
	if _, err := provider.New("wubigrok", provider.Config{Name: "t", Model: "m"}); err == nil {
		t.Fatal("未注入 client 时 New 应报错")
	}
	SetClient(&mockClient{})
	p, err := provider.New("wubigrok", provider.Config{Name: "t", Model: "m"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.Name() != "t" {
		t.Errorf("Name = %q, want t", p.Name())
	}
}

var errBoom = errors.New("boom")

// TestBridge_Stream_EmptyModel 验证 model 为空时请求透传空模型名，
// 由 ai.Client 按当前活跃引擎动态解析（办公板块跟随模型中心的关键）。
func TestBridge_Stream_EmptyModel(t *testing.T) {
	mc := &mockClient{chunks: []ai.SSEChunk{{Done: true}}}
	p := &Provider{name: "gaea", model: "", client: mc}
	_, err := p.Stream(context.Background(), provider.Request{Messages: []provider.Message{{Role: provider.RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if mc.gotReq == nil {
		t.Fatal("client 未收到请求")
	}
	if mc.gotReq.Model != "" {
		t.Errorf("model = %q, want 空（由 ai.Client 动态解析）", mc.gotReq.Model)
	}
}
