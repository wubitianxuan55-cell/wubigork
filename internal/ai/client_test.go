package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/auth"
	"github.com/gaea/gaea/internal/config"
)

// newTestClient 构造指向测试服务器的 Client（white-box：直接注入 httpClient/token）
func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := &config.Config{XaiAPIBaseURL: srv.URL, Model: "grok-4.20"}
	c := &Client{
		cfg:        cfg,
		httpClient: srv.Client(),
		tokenStore: auth.NewTokenStore(t.TempDir() + "/token.json"),
		sem:        make(chan struct{}, 4),
	}
	// 直接注入有效 token，避免走 tokenStore / 刷新流程
	// ExpiresIn 需大于 3600（IsExpired 提前 1 小时刷新），否则偶发"登录已过期"。
	c.token = &auth.Token{AccessToken: "test-token", ObtainedAt: time.Now(), ExpiresIn: 7200}
	return c, srv
}

func reqForTest() *ChatRequest {
	return &ChatRequest{
		Model:    "grok-4.20",
		Messages: []ChatMessage{{Role: "user", Content: "你好"}},
	}
}

// ─── Chat 非流式 ─────────────────────────────────────────────

func TestChat_Success(t *testing.T) {
	var gotModel, gotBody string
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotModel = r.Header.Get("Authorization")
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		gotBody = req.Messages[0].Content
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "chat-1", "model": "grok-4.20",
			"choices": []map[string]any{{
				"index":   0,
				"message": map[string]any{"role": "assistant", "content": "回复内容"},
			}},
		})
	}))
	resp, err := c.Chat(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Choices[0].Message.Content != "回复内容" {
		t.Errorf("回复 = %q", resp.Choices[0].Message.Content)
	}
	if gotModel != "Bearer test-token" {
		t.Errorf("Authorization = %q", gotModel)
	}
	if gotBody != "你好" {
		t.Errorf("请求体消息 = %q", gotBody)
	}
}

func TestChat_Non200(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server exploded"))
	}))
	_, err := c.Chat(context.Background(), reqForTest())
	if err == nil {
		t.Fatal("500 应报错")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("错误信息 = %q, want 含 500", err.Error())
	}
}

func TestChat_APIReturnedError(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "rate limited", "code": "rate_limit"},
		})
	}))
	_, err := c.Chat(context.Background(), reqForTest())
	if err == nil {
		t.Fatal("API 返回 error 字段应报错")
	}
	if !strings.Contains(err.Error(), "rate_limit") {
		t.Errorf("错误信息 = %q, want 含错误码", err.Error())
	}
}

// ─── ChatStream SSE ──────────────────────────────────────────

func TestChatStream_SSEText(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"你好\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"世界\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	chunks, err := c.ChatStream(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	var sb strings.Builder
	done := false
	for ch := range chunks {
		if ch.Error != "" {
			t.Fatalf("stream error: %s", ch.Error)
		}
		if ch.Done {
			done = true
		}
		sb.WriteString(ch.Content)
	}
	if sb.String() != "你好世界" {
		t.Errorf("拼接内容 = %q, want 你好世界", sb.String())
	}
	if !done {
		t.Error("未收到 Done 标记")
	}
}

// TestChatStream_SSEReasoning 验证思考模式：delta.reasoning_content 被保留并透传。
func TestChatStream_SSEReasoning(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(`data: {"choices":[{"index":0,"delta":{"reasoning_content":"先思考"}}]}` + "\n\n"))
		w.Write([]byte(`data: {"choices":[{"index":0,"delta":{"reasoning_content":"再回答"}}]}` + "\n\n"))
		w.Write([]byte(`data: {"choices":[{"index":0,"delta":{"content":"结果"}}]}` + "\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	chunks, err := c.ChatStream(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	var reasoning, content strings.Builder
	done := false
	for ch := range chunks {
		if ch.Error != "" {
			t.Fatalf("stream error: %s", ch.Error)
		}
		if ch.Done {
			done = true
		}
		reasoning.WriteString(ch.Reasoning)
		content.WriteString(ch.Content)
	}
	if reasoning.String() != "先思考再回答" {
		t.Errorf("推理内容 = %q, want 先思考再回答", reasoning.String())
	}
	if content.String() != "结果" {
		t.Errorf("正文 = %q, want 结果", content.String())
	}
	if !done {
		t.Error("未收到 Done 标记")
	}
}

func TestChatStream_SSEToolCall(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// 工具调用分片：name 和 arguments 分开到达
		w.Write([]byte(`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"ls","arguments":""}}]}}]}` + "\n\n"))
		w.Write([]byte(`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\":"}}]}}]}` + "\n\n"))
		w.Write([]byte(`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\".\"}"}}]}}]}` + "\n\n"))
		w.Write([]byte(`data: {"choices":[{"index":0,"delta":{"finish_reason":"tool_calls"}}]}` + "\n\n"))
	}))
	chunks, err := c.ChatStream(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	var calls []ChatToolCall
	for ch := range chunks {
		if len(ch.ToolCalls) > 0 {
			calls = ch.ToolCalls
		}
	}
	if len(calls) != 1 {
		t.Fatalf("工具调用数 = %d, want 1", len(calls))
	}
	if calls[0].ID != "call_1" || calls[0].Function.Name != "ls" {
		t.Errorf("工具调用 = %+v", calls[0])
	}
	if calls[0].Function.Arguments != `{"path":"."}` {
		t.Errorf("参数分片未拼装 = %q", calls[0].Function.Arguments)
	}
}

func TestChatStream_Non200(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request body"))
	}))
	_, err := c.ChatStream(context.Background(), reqForTest())
	if err == nil {
		t.Fatal("400 应报错")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("错误信息 = %q", err.Error())
	}
}

// ─── ChatSimpleStream 完整拼装 ───────────────────────────────

func TestChatSimpleStream_CollectsFullText(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"第一段\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"第二段\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush() // SSE 逐块冲刷，避免 [DONE] 未落盘前连接关闭导致 flaky
		}
	}))
	text, err := c.ChatSimpleStream(context.Background(), "grok-4.20", "sys", "user")
	if err != nil {
		t.Fatalf("ChatSimpleStream: %v", err)
	}
	if text != "第一段第二段" {
		t.Errorf("文本 = %q, want 第一段第二段", text)
	}
}

// ─── 模型解析 ────────────────────────────────────────────────

func TestResolveModelName_ReqModelWins(t *testing.T) {
	cfg := &config.Config{Model: "default-model"}
	c := &Client{cfg: cfg, sem: make(chan struct{}, 4)}
	if got := c.resolveModelName("explicit", ""); got != "explicit" {
		t.Errorf("resolveModelName = %q, want explicit", got)
	}
}

func TestResolveModelName_XAIFallback(t *testing.T) {
	cfg := &config.Config{Model: "grok-4.20"}
	c := &Client{cfg: cfg, sem: make(chan struct{}, 4)}
	if got := c.resolveModelName("", ""); got != "grok-4.20" {
		t.Errorf("resolveModelName = %q, want 默认模型", got)
	}
}

// ─── 引擎状态 ────────────────────────────────────────────────

func TestActiveEngineID_Default(t *testing.T) {
	c := &Client{sem: make(chan struct{}, 4)}
	if got := c.ActiveEngineID(); got != "xai" {
		t.Errorf("ActiveEngineID = %q, want xai（默认）", got)
	}
	c.SetActiveEngine("ollama")
	if got := c.ActiveEngineID(); got != "ollama" {
		t.Errorf("ActiveEngineID = %q, want ollama", got)
	}
	c.SetActiveEngine("")
	if got := c.ActiveEngineID(); got != "xai" {
		t.Errorf("空引擎应回退 xai, got %q", got)
	}
}

// ─── 并发信号量 ──────────────────────────────────────────────

func TestSemaphore_AcquireRelease(t *testing.T) {
	c := &Client{sem: make(chan struct{}, 1)}
	if err := c.acquireSem(context.Background()); err != nil {
		t.Fatalf("acquireSem: %v", err)
	}
	// 第二个 acquire 应阻塞（这里用带超时的 ctx 验证）
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := c.acquireSem(ctx); err == nil {
		t.Fatal("信号量满时 acquire 应阻塞到超时")
	}
	c.releaseSem()
	if err := c.acquireSem(context.Background()); err != nil {
		t.Fatalf("释放后 acquire: %v", err)
	}
	c.releaseSem()
}

func TestSemaphore_CtxCancelled(t *testing.T) {
	c := &Client{sem: make(chan struct{}, 1)}
	c.acquireSem(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.acquireSem(ctx); err == nil {
		t.Error("ctx 取消后 acquire 应报错")
	}
	c.releaseSem()
}
