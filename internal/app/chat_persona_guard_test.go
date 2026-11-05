package app

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// 众神殿刀1（进度计划/gaea-chat-persona-optimization-20261105.md §1.1③）：
// 空正文/取消守卫——空流/仅 reasoning 不得伪装成成功回合（显式 error 终态、
// 不落库、不推进记账），取消空轮以 cancelled 终态收尾且不落库，
// AppendExchange 对空 assistant fail-loud。

// emptyStreamHandler 只发 [DONE] 的假 LLM（空流）。
func emptyStreamHandler(calls *atomic.Int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
}

// reasoningOnlyStreamHandler 只发 reasoning delta + [DONE] 的假 LLM。
func reasoningOnlyStreamHandler(calls *atomic.Int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"思考中\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
}

// newPersonaGuardTestApp 带 runID 固定桩与 whisper 会话清理的流式测试装配。
func newPersonaGuardTestApp(t *testing.T, personalityID string, handler http.Handler) (*App, string) {
	t.Helper()
	a := newChatServiceTestAppWithHandler(t, handler)
	cleanupWhisperSession(t, personalityID)
	fixed := "cs_guard_" + personalityID
	orig := newChatStreamRunID
	newChatStreamRunID = func() string { return fixed }
	t.Cleanup(func() {
		cleanupWhisperSession(t, personalityID)
		newChatStreamRunID = orig
	})
	return a, fixed
}

func countMessages(t *testing.T, a *App, topicID string) int {
	t.Helper()
	msgs, err := a.ChatMessagesList(topicID)
	if err != nil {
		t.Fatalf("ChatMessagesList: %v", err)
	}
	return len(msgs)
}

// TestChatStreamPersona_EmptyReplyErrors 非取消空正文：有界重试一次仍空 →
// error 终态（不发 done），chat.db 不落任何行。
func TestChatStreamPersona_EmptyReplyErrors(t *testing.T) {
	var calls atomic.Int64
	a, fixed := newPersonaGuardTestApp(t, "persona_guard_empty", emptyStreamHandler(&calls))
	topic, err := a.ChatTopicCreate("空流守卫", "persona_guard_empty")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	events := captureStreamEvents(t, a, "chat-stream:"+fixed, func() {
		if _, err := a.ChatStreamPersona(topic.ID, "你好", "persona_guard_empty", false, false, false); err != nil {
			t.Errorf("ChatStreamPersona: %v", err)
		}
	})
	if len(events) == 0 || events[len(events)-1]["type"] != "error" {
		t.Fatalf("终态应为 error: %v", events)
	}
	last := events[len(events)-1]
	if msg, _ := last["error"].(string); !strings.Contains(msg, "空回复") {
		t.Errorf("error 文案应说明空回复: %v", msg)
	}
	for _, ev := range events {
		if ev["type"] == "done" {
			t.Fatalf("空回复不得发 done: %v", events)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("空流应有界重试一次（calls=%d, want 2）", got)
	}
	if n := countMessages(t, a, topic.ID); n != 0 {
		t.Fatalf("空回复轮不得落库（rows=%d）", n)
	}
	time.Sleep(100 * time.Millisecond) // 异步落库/状态协程收尾
}

// TestChatStreamPersona_ReasoningOnlyErrors 仅 reasoning 流：不重试（reasoning
// 已外发，重试会重复拼接），error 终态，不落库。
func TestChatStreamPersona_ReasoningOnlyErrors(t *testing.T) {
	var calls atomic.Int64
	a, fixed := newPersonaGuardTestApp(t, "persona_guard_reasoning", reasoningOnlyStreamHandler(&calls))
	topic, err := a.ChatTopicCreate("仅思考流守卫", "persona_guard_reasoning")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	events := captureStreamEvents(t, a, "chat-stream:"+fixed, func() {
		if _, err := a.ChatStreamPersona(topic.ID, "你好", "persona_guard_reasoning", false, false, false); err != nil {
			t.Errorf("ChatStreamPersona: %v", err)
		}
	})
	if len(events) == 0 || events[len(events)-1]["type"] != "error" {
		t.Fatalf("终态应为 error: %v", events)
	}
	for _, ev := range events {
		if ev["type"] == "done" {
			t.Fatalf("仅 reasoning 流不得发 done: %v", events)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("reasoning 已外发时不得重试（calls=%d, want 1）", got)
	}
	if n := countMessages(t, a, topic.ID); n != 0 {
		t.Fatalf("空回复轮不得落库（rows=%d）", n)
	}
	time.Sleep(100 * time.Millisecond)
}

// TestChatStreamPersona_CancelledEmptyNoRows 首 token 前取消的空轮：cancelled
// 终态（非 done 非 error），chat.db 不落任何行（取消轮不得伪造空 assistant 交换）。
func TestChatStreamPersona_CancelledEmptyNoRows(t *testing.T) {
	var calls atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		time.Sleep(400 * time.Millisecond) // 留出取消窗口（首 token 前）
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: [DONE]\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
	a, fixed := newPersonaGuardTestApp(t, "persona_guard_cancel", handler)
	topic, err := a.ChatTopicCreate("取消空轮守卫", "persona_guard_cancel")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	time.AfterFunc(100*time.Millisecond, func() { a.ChatStreamCancel(fixed) })
	events := captureStreamEvents(t, a, "chat-stream:"+fixed, func() {
		if _, err := a.ChatStreamPersona(topic.ID, "你好", "persona_guard_cancel", false, false, false); err != nil {
			t.Errorf("ChatStreamPersona: %v", err)
		}
	})
	if len(events) == 0 || events[len(events)-1]["type"] != "cancelled" {
		t.Fatalf("终态应为 cancelled: %v", events)
	}
	for _, ev := range events {
		if ev["type"] == "done" {
			t.Fatalf("取消空轮不得发 done: %v", events)
		}
	}
	if n := countMessages(t, a, topic.ID); n != 0 {
		t.Fatalf("取消空轮不得落库（rows=%d）", n)
	}
	time.Sleep(100 * time.Millisecond)
}

// TestAppendExchange_RejectsEmptyAssistant 存储层防线：空 assistant fail-loud。
func TestAppendExchange_RejectsEmptyAssistant(t *testing.T) {
	a := newChatServiceTestApp(t)
	topic, err := a.ChatTopicCreate("空回复防线", "gaea")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	if err := a.chatStore.AppendExchange(topic.ID, "你好", "", ""); err == nil {
		t.Fatal("空 assistant 应拒绝落库")
	}
	if n := countMessages(t, a, topic.ID); n != 0 {
		t.Fatalf("拒绝后不得留下半条交换（rows=%d）", n)
	}
}
