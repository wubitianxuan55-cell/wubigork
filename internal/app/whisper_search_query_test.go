package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// B6 拟查询刀全链单测（2026-11-06）：触发走规则门控，查询词由模型从语境拟写
// （「所以，明天的天气怎么样」实机病例——规则清洗修不完连接词/指代）。
// 拟写请求经同一 herdsman 测试服务（system 含「搜索查询拟写器」标记区分），
// chatWebSearch seam 捕获最终查询词。
func sseReply(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"" + text + "\"}}]}\n\n"))
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func TestWhisperChat_LLMFormulatedSearchQuery(t *testing.T) {
	var formulationSeen atomic.Bool
	var capturedQuery atomic.Value
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "搜索查询拟写器") {
			formulationSeen.Store(true)
			sseReply(w, "成都 明天 天气")
			return
		}
		sseReply(w, "好的")
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	a := newChatServiceTestAppWithHandler(t, handler)
	personalityID := "persona_b6_formulate"
	cleanupWhisperSession(t, personalityID)
	t.Cleanup(func() { cleanupWhisperSession(t, personalityID) })
	orig := chatWebSearch
	chatWebSearch = func(q string) (chatSearchOutcome, error) {
		capturedQuery.Store(q)
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: q, Text: "罐头"}, nil
	}
	t.Cleanup(func() { chatWebSearch = orig })

	if _, err := a.whisperChatCore(context.Background(), "所以，明天的天气怎么样", personalityID, "", false, true /* B7 后拟查询归 lowLatency(voice) 面 */, true, false, nil, nil); err != nil {
		t.Fatalf("whisperChatCore: %v", err)
	}
	if !formulationSeen.Load() {
		t.Fatal("拟查询调用未发出（应先于检索）")
	}
	if got, _ := capturedQuery.Load().(string); got != "成都 明天 天气" {
		t.Fatalf("检索应使用模型拟写的查询词，got %q", got)
	}
	// 异步记忆落库协程与 TempDir 清理竞速（house band-aid，dual-channel 同款）
	time.Sleep(300 * time.Millisecond)
}

// 降级路径：拟写请求失败（服务端 500）→ 回退原句检索（行为同 B5 前）。
func TestWhisperChat_FormulateFallback(t *testing.T) {
	var capturedQuery atomic.Value
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "搜索查询拟写器") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		sseReply(w, "好的")
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	a := newChatServiceTestAppWithHandler(t, handler)
	personalityID := "persona_b6_fallback"
	cleanupWhisperSession(t, personalityID)
	t.Cleanup(func() { cleanupWhisperSession(t, personalityID) })
	orig := chatWebSearch
	chatWebSearch = func(q string) (chatSearchOutcome, error) {
		capturedQuery.Store(q)
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: q, Text: "罐头"}, nil
	}
	t.Cleanup(func() { chatWebSearch = orig })

	if _, err := a.whisperChatCore(context.Background(), "所以，明天的天气怎么样", personalityID, "", false, true /* B7 后拟查询归 lowLatency(voice) 面 */, true, false, nil, nil); err != nil {
		t.Fatalf("whisperChatCore: %v", err)
	}
	if got, _ := capturedQuery.Load().(string); got != "所以，明天的天气怎么样" {
		t.Fatalf("拟写失败应回退原句，got %q", got)
	}
	time.Sleep(300 * time.Millisecond)
}
