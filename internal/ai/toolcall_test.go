package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockSSEServer 返回一个 SSE 服务，按帧推送自定义事件。
func mockSSEServer(frames []string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, f := range frames {
			io.WriteString(w, f)
		}
	}))
}

// TestParseStreamEvents_ToolCalls 验证带 tool_calls 分片的 SSE 流能拼装为完整工具调用。
func TestParseStreamEvents_ToolCalls(t *testing.T) {
	// OpenAI 兼容流式工具调用：id 与 name 在首帧，arguments 分片。
	frames := []string{
		"data: {\"choices\":[{\"delta\":{\"role\":\"assistant\",\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"spec_query\",\"arguments\":\"\"}}]}}]}\n\n",
		"data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"question\\\":\\\"砷超标\"}}]}}]}\n\n",
		"data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"限值\\\"}\"}}]}}]}\n\n",
		"data: {\"choices\":[{\"delta\":{\"finish_reason\":\"tool_calls\"}}]}\n\n",
		"data: [DONE]\n\n",
	}
	srv := mockSSEServer(frames)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	c := &Client{sem: make(chan struct{}, 1)}
	if err := c.acquireSem(context.Background()); err != nil {
		t.Fatalf("acquireSem: %v", err)
	}
	chunks := make(chan SSEChunk, 16)
	go c.parseStreamEvents(context.Background(), resp, chunks)

	var got []SSEChunk
	for ch := range chunks {
		got = append(got, ch)
	}

	if len(got) == 0 {
		t.Fatal("no chunks received")
	}
	last := got[len(got)-1]
	if len(last.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call in final chunk, got %d (chunks: %+v)", len(last.ToolCalls), got)
	}
	tc := last.ToolCalls[0]
	if tc.ID != "call_1" {
		t.Errorf("expected id call_1, got %q", tc.ID)
	}
	if tc.Function.Name != "spec_query" {
		t.Errorf("expected name spec_query, got %q", tc.Function.Name)
	}
	wantArgs := `{"question":"砷超标限值"}`
	if tc.Function.Arguments != wantArgs {
		t.Errorf("expected args %q, got %q", wantArgs, tc.Function.Arguments)
	}
	if !last.Done {
		t.Errorf("expected Done=true on final tool-call chunk")
	}
}

// TestChatMessageMarshal_ToolCalls 验证 assistant 工具调用消息序列化为 OpenAI 兼容格式。
func TestChatMessageMarshal_ToolCalls(t *testing.T) {
	m := ChatMessage{
		Role:    "assistant",
		Content: "",
		ToolCalls: []ChatToolCall{{
			ID:   "call_1",
			Type: "function",
			Function: ChatToolFunction{
				Name:      "spec_query",
				Arguments: `{"question":"砷超标限值"}`,
			},
		}},
	}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, `"tool_calls"`) || !strings.Contains(s, `"function"`) {
		t.Errorf("output missing tool_calls structure: %s", s)
	}
	if !strings.Contains(s, `"spec_query"`) {
		t.Errorf("output missing function name: %s", s)
	}
}

// TestChatRequestMarshal_Tools 验证 ChatRequest 带 tools 序列化正确，
// 且符合 OpenAI 兼容格式（每个工具必须含 "type":"function" 包裹层，
// 缺失会导致 DeepSeek/Grok 等 API 以 400 拒绝：tools[0]: missing field `type`）。
func TestChatRequestMarshal_Tools(t *testing.T) {
	req := ChatRequest{
		Model:    "grok-3",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
		Stream:   true,
		Tools: []ChatToolSchema{{
			Type: "function",
			Function: ChatToolFunctionSpec{
				Name:        "calc_math",
				Description: "数学计算",
				Parameters:  []byte(`{"type":"object"}`),
			},
		}},
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"type":"function"`) {
		t.Fatalf("output missing type:function: %s", s)
	}
	if !strings.Contains(s, `"function":{"name":"calc_math"`) {
		t.Fatalf("output missing function wrapper: %s", s)
	}
	if !strings.Contains(s, `"calc_math"`) {
		t.Errorf("output missing tool name: %s", s)
	}
}
