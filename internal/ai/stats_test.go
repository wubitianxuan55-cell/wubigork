package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/modelengine"
)

// newStatsClient 构造带统计引擎管理器的测试客户端。
func newStatsClient(t *testing.T, handler http.Handler) (*Client, *modelengine.Manager) {
	t.Helper()
	c, _ := newTestClient(t, handler)
	mgr := modelengine.NewManager("", "")
	mgr.SetStatsPath(filepath.Join(t.TempDir(), "model_stats.json"))
	c.SetEngineManager(mgr)
	return c, mgr
}

// TestChat_RecordsUsage 非流式调用应记录次数与 token 用量。
func TestChat_RecordsUsage(t *testing.T) {
	c, mgr := newStatsClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "chat-1", "model": "grok-4.20",
			"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": "hi"}}},
			"usage":   map[string]any{"prompt_tokens": 12, "completion_tokens": 3, "total_tokens": 15},
		})
	}))
	if _, err := c.Chat(context.Background(), reqForTest()); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	sum := mgr.GetModelCallStats()
	if sum.TotalCalls != 1 || sum.SuccessCalls != 1 {
		t.Fatalf("TotalCalls/SuccessCalls = %d/%d, want 1/1", sum.TotalCalls, sum.SuccessCalls)
	}
	if sum.InputTokens != 12 || sum.OutputTokens != 3 {
		t.Errorf("tokens = %d/%d, want 12/3", sum.InputTokens, sum.OutputTokens)
	}
	if sum.PerModel[0].EngineID != "xai" || sum.PerModel[0].Model != "grok-4.20" {
		t.Errorf("PerModel = %+v", sum.PerModel)
	}
}

// TestChatStream_RecordsUsage 流式调用应记录次数、token 用量与成功状态。
func TestChatStream_RecordsUsage(t *testing.T) {
	c, mgr := newStatsClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		time.Sleep(10 * time.Millisecond)
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":20,\"completion_tokens\":7,\"total_tokens\":27}}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	chunks, err := c.ChatStream(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	for ch := range chunks {
		if ch.Error != "" {
			t.Fatalf("stream error: %s", ch.Error)
		}
	}
	sum := mgr.GetModelCallStats()
	if sum.TotalCalls != 1 || sum.SuccessCalls != 1 {
		t.Fatalf("TotalCalls/SuccessCalls = %d/%d, want 1/1", sum.TotalCalls, sum.SuccessCalls)
	}
	if sum.InputTokens != 20 || sum.OutputTokens != 7 {
		t.Errorf("tokens = %d/%d, want 20/7", sum.InputTokens, sum.OutputTokens)
	}
	if sum.AvgDurationMs <= 0 {
		t.Errorf("AvgDurationMs = %d, want > 0", sum.AvgDurationMs)
	}
}

// TestChatStream_ErrorRecordsFail 流式请求失败应记录失败次数。
func TestChatStream_ErrorRecordsFail(t *testing.T) {
	c, mgr := newStatsClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("upstream down"))
	}))
	_, err := c.ChatStream(context.Background(), reqForTest())
	if err == nil {
		t.Fatal("502 应报错")
	}
	sum := mgr.GetModelCallStats()
	if sum.TotalCalls != 1 || sum.FailCalls != 1 {
		t.Fatalf("TotalCalls/FailCalls = %d/%d, want 1/1", sum.TotalCalls, sum.FailCalls)
	}
	if sum.PerModel[0].LastError == "" {
		t.Error("失败调用应记录 LastError")
	}
}

// TestStats_RecordLatency 耗时统计非负。
func TestStats_RecordLatency(t *testing.T) {
	c, mgr := newStatsClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "x", "model": "grok-4.20",
			"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": "ok"}}},
		})
	}))
	start := time.Now()
	if _, err := c.Chat(context.Background(), reqForTest()); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	elapsed := time.Since(start).Milliseconds()
	sum := mgr.GetModelCallStats()
	if sum.AvgDurationMs < 1 || sum.AvgDurationMs > elapsed+50 {
		t.Errorf("AvgDurationMs = %d, 期望在 [1, %d] 内", sum.AvgDurationMs, elapsed+50)
	}
}

// TestChat_401NoRefreshRecordsSingleFail 401 且无 refresh token 时只记一次失败，不重复统计。
func TestChat_401NoRefreshRecordsSingleFail(t *testing.T) {
	c, mgr := newStatsClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	if _, err := c.Chat(context.Background(), reqForTest()); err == nil {
		t.Fatal("401 应报错")
	}
	sum := mgr.GetModelCallStats()
	if sum.TotalCalls != 1 || sum.FailCalls != 1 || sum.SuccessCalls != 0 {
		t.Errorf("TotalCalls/Success/Fail = %d/%d/%d, want 1/0/1", sum.TotalCalls, sum.SuccessCalls, sum.FailCalls)
	}
}

// TestParseStreamEvents_CapturesUsage SSE 流中的 usage 块应透传到 SSEChunk。
func TestParseStreamEvents_CapturesUsage(t *testing.T) {
	c, _ := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":9,\"completion_tokens\":4}}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	req := &ChatRequest{Model: "grok-4.20", Messages: []ChatMessage{{Role: "user", Content: "hi"}}}
	chunks, err := c.ChatStream(context.Background(), req)
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	var last *ChatUsage
	for ch := range chunks {
		if ch.Usage != nil {
			last = ch.Usage
		}
	}
	if last == nil || last.PromptTokens != 9 || last.CompletionTokens != 4 {
		t.Fatalf("Usage = %+v, want 9/4", last)
	}
}

// TestChatStream_SendsIncludeUsage 流式请求应带 stream_options.include_usage，
// 让 OpenAI 兼容服务端在流末尾返回 usage，从而统计到真实 Token。
func TestChatStream_SendsIncludeUsage(t *testing.T) {
	var got map[string]any
	c, mgr := newStatsClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":30,\"completion_tokens\":11,\"total_tokens\":41}}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	chunks, err := c.ChatStream(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	for range chunks {
	}
	so, ok := got["stream_options"].(map[string]any)
	if !ok || so["include_usage"] != true {
		t.Fatalf("stream_options = %v, want include_usage=true", got["stream_options"])
	}
	sum := mgr.GetModelCallStats()
	if sum.InputTokens != 30 || sum.OutputTokens != 11 {
		t.Errorf("tokens = %d/%d, want 30/11", sum.InputTokens, sum.OutputTokens)
	}
}

// TestChatUsage_CacheSplit 验证两种缓存拆分形状都能被解析：
// DeepSeek 顶层 prompt_cache_{hit,miss}_tokens 与
// OpenAI/MiMo prompt_tokens_details.cached_tokens。
func TestChatUsage_CacheSplit(t *testing.T) {
	t.Run("deepseek-style", func(t *testing.T) {
		var u ChatUsage
		if err := json.Unmarshal([]byte(`{
			"prompt_tokens": 1200, "completion_tokens": 200, "total_tokens": 1400,
			"prompt_cache_hit_tokens": 800, "prompt_cache_miss_tokens": 400
		}`), &u); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if u.CacheHitTokens() != 800 || u.CacheMissTokens() != 400 {
			t.Errorf("deepseek cache = hit %d miss %d, want 800/400", u.CacheHitTokens(), u.CacheMissTokens())
		}
	})
	t.Run("openai-style", func(t *testing.T) {
		var u ChatUsage
		if err := json.Unmarshal([]byte(`{
			"prompt_tokens": 1000, "completion_tokens": 50, "total_tokens": 1050,
			"prompt_tokens_details": {"cached_tokens": 700}
		}`), &u); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if u.CacheHitTokens() != 700 || u.CacheMissTokens() != 300 {
			t.Errorf("openai cache = hit %d miss %d, want 700/300", u.CacheHitTokens(), u.CacheMissTokens())
		}
	})
	t.Run("no-cache-fields", func(t *testing.T) {
		var u ChatUsage
		if err := json.Unmarshal([]byte(`{
			"prompt_tokens": 500, "completion_tokens": 30, "total_tokens": 530
		}`), &u); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		// 无缓存信息：命中 0，未命中按 prompt 推算（不出现负数）。
		if u.CacheHitTokens() != 0 || u.CacheMissTokens() != 500 {
			t.Errorf("no-cache = hit %d miss %d, want 0/500", u.CacheHitTokens(), u.CacheMissTokens())
		}
	})
}

// TestChatStream_FallbackWithoutStreamOptions 服务端不支持 stream_options 时，
// 应去掉该字段重试一次，且只统计成功的调用。
func TestChatStream_FallbackWithoutStreamOptions(t *testing.T) {
	calls := 0
	var secondBody map[string]any
	c, mgr := newStatsClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if calls == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"unknown field stream_options"}}`))
			return
		}
		secondBody = body
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"}}]}\n\n"))
		w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":2}}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	chunks, err := c.ChatStream(context.Background(), reqForTest())
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	for range chunks {
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2（首次带 stream_options 失败 + 去掉重试）", calls)
	}
	if _, ok := secondBody["stream_options"]; ok {
		t.Errorf("重试请求不应包含 stream_options: %v", secondBody["stream_options"])
	}
	sum := mgr.GetModelCallStats()
	if sum.TotalCalls != 1 || sum.SuccessCalls != 1 {
		t.Fatalf("TotalCalls/SuccessCalls = %d/%d, want 1/1", sum.TotalCalls, sum.SuccessCalls)
	}
	if sum.InputTokens != 5 || sum.OutputTokens != 2 {
		t.Errorf("tokens = %d/%d, want 5/2", sum.InputTokens, sum.OutputTokens)
	}
}
