package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/chat"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// These tests intentionally run the same path as the desktop chat UI:
// ChatStreamPlain(search=true) plus the real SSE event contract. The older
// ChatSend tests cannot catch the two-authority streaming bug because the
// blocking path reads toolLoopResult.Reply directly.

type streamSearchResult struct {
	events      []map[string]interface{}
	msgs        []chat.Message
	searchCalls int
}

func runStreamSearchTurn(
	t *testing.T,
	handler http.Handler,
	message string,
	searchEnabled, forceSearch bool,
) streamSearchResult {
	t.Helper()
	a := newChatServiceTestAppWithHandler(t, handler)
	topic, err := a.ChatTopicCreate("stream-search", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}

	fixed := "chat_search_stream_" + strings.ReplaceAll(t.Name(), "/", "_")
	origRunID := newChatStreamRunID
	newChatStreamRunID = func() string { return fixed }
	t.Cleanup(func() { newChatStreamRunID = origRunID })

	var searchCalls atomic.Int64
	origSearch := chatWebSearch
	chatWebSearch = func(q string) (chatSearchOutcome, error) {
		searchCalls.Add(1)
		return chatSearchOutcome{
			State:   builtin.SearchStateOK,
			Query:   q,
			Text:    "1. 罐头搜索结果 [probe]",
			Sources: 1,
			References: []chatSearchReference{{
				Title:   "罐头搜索结果",
				URL:     "https://example.com/search-result",
				Source:  "example",
				Snippet: "用于搜索卡 render 的短摘要",
			}},
		}, nil
	}
	t.Cleanup(func() { chatWebSearch = origSearch })

	events := captureStreamEvents(t, a, "chat-stream:"+fixed, func() {
		if _, err := a.ChatStreamPlain(topic.ID, message, searchEnabled, false, forceSearch); err != nil {
			t.Errorf("ChatStreamPlain: %v", err)
		}
	})
	msgs, err := a.ChatMessagesList(topic.ID)
	if err != nil {
		t.Fatalf("ChatMessagesList: %v", err)
	}
	return streamSearchResult{
		events:      events,
		msgs:        msgs,
		searchCalls: int(searchCalls.Load()),
	}
}

func streamReplyAndContents(t *testing.T, events []map[string]interface{}) (string, string) {
	t.Helper()
	var delta strings.Builder
	var doneReply string
	var finalType string
	for _, ev := range events {
		switch ev["type"] {
		case "delta":
			if s, ok := ev["content"].(string); ok {
				delta.WriteString(s)
			}
		case "done", "error", "cancelled":
			finalType = ev["type"].(string)
			if s, ok := ev["reply"].(string); ok {
				doneReply = s
			}
		}
	}
	if finalType != "done" {
		t.Fatalf("终态 = %q, want done; events=%v", finalType, events)
	}
	return delta.String(), doneReply
}

func assertCanonicalStreamReply(t *testing.T, res streamSearchResult, want string) {
	t.Helper()
	delta, doneReply := streamReplyAndContents(t, res.events)
	if delta != want {
		t.Errorf("delta 拼接 = %q, want %q", delta, want)
	}
	if doneReply != want {
		t.Errorf("done.reply = %q, want %q", doneReply, want)
	}
	if len(res.msgs) != 2 {
		t.Fatalf("消息数 = %d, want 2; messages=%+v", len(res.msgs), res.msgs)
	}
	if got := res.msgs[1].Content; got != want {
		t.Errorf("DB assistant = %q, want %q", got, want)
	}
	for _, surface := range []string{delta, doneReply, res.msgs[1].Content} {
		if strings.Contains(surface, "<tool_call>") ||
			strings.Contains(surface, "<function=") ||
			strings.Contains(surface, "<tool") {
			t.Errorf("工具标记泄漏到用户面: %.200q", surface)
		}
	}
}

func terminalStreamEvent(t *testing.T, events []map[string]interface{}, wantType string) map[string]interface{} {
	t.Helper()
	for _, ev := range events {
		if ev["type"] == wantType {
			return ev
		}
	}
	t.Fatalf("缺少 %s 事件: %v", wantType, events)
	return nil
}

func hasStreamPhase(events []map[string]interface{}, phase string) bool {
	for _, ev := range events {
		if ev["type"] == "status" && ev["phase"] == phase {
			return true
		}
	}
	return false
}

func TestChatStreamSearch_AutoModeDirectAnswer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sseReply(w, "这是无需联网的直接回答。")
	})
	res := runStreamSearchTurn(t, handler, "你好", true, false)
	assertCanonicalStreamReply(t, res, "这是无需联网的直接回答。")
	if res.searchCalls != 0 {
		t.Errorf("search seam calls = %d, want 0", res.searchCalls)
	}
	done := terminalStreamEvent(t, res.events, "done")
	if done["search_state"] != "not_requested" {
		t.Errorf("search_state = %v, want not_requested", done["search_state"])
	}
	if done["searches"] != float64(0) {
		t.Errorf("searches = %v, want 0", done["searches"])
	}
	if !hasStreamPhase(res.events, "thinking") || !hasStreamPhase(res.events, "finalizing") {
		t.Errorf("缺少 thinking/finalizing 进度: %v", res.events)
	}
}

func TestChatStreamSearch_OneSearchThenProse(t *testing.T) {
	var round atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if round.Add(1) == 1 {
			sseToolCallReply(w, "call_1", "web_search", `"{\"query\":\"成都 明天 天气\"}"`)
			return
		}
		sseReply(w, "成都明天多云，适合出行。")
	})
	res := runStreamSearchTurn(t, handler, "明天成都天气怎么样", true, false)
	assertCanonicalStreamReply(t, res, "成都明天多云，适合出行。")
	if res.searchCalls != 1 {
		t.Errorf("search seam calls = %d, want 1", res.searchCalls)
	}
	done := terminalStreamEvent(t, res.events, "done")
	if done["search_state"] != "ok" || done["searches"] != float64(1) {
		t.Errorf("done search metadata = %v", done)
	}
	if !hasStreamPhase(res.events, "searching") || !hasStreamPhase(res.events, "search_done") {
		t.Errorf("缺少搜索进度事件: %v", res.events)
	}
	var extra map[string]interface{}
	if err := json.Unmarshal([]byte(res.msgs[1].Extra), &extra); err != nil {
		t.Fatalf("assistant extra JSON: %v", err)
	}
	search, ok := extra["search"].(map[string]interface{})
	if !ok || search["state"] != "ok" || search["sources"] != float64(1) {
		t.Errorf("落库搜索摘要异常: %v", extra)
	}
	refs, ok := search["references"].([]interface{})
	if !ok || len(refs) != 1 {
		t.Fatalf("落库来源引用异常: %v", search["references"])
	}
	ref, ok := refs[0].(map[string]interface{})
	if !ok || ref["url"] != "https://example.com/search-result" ||
		ref["title"] != "罐头搜索结果" || ref["source"] != "example" {
		t.Errorf("来源引用字段异常: %v", refs[0])
	}
}

func TestChatStreamSearch_FinalRoundRawMarkupStripped(t *testing.T) {
	var round atomic.Int64
	raw := "最终说明。\n<tool_call>\n<function=web_search>\n<parameter=query>\n不应泄漏\n</parameter>\n</function>\n</tool_call>"
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		switch round.Add(1) {
		case 1:
			sseToolCallReply(w, "call_1", "web_search", `"{\"query\":\"第一搜\"}"`)
		case 2:
			sseToolCallReply(w, "call_2", "web_search", `"{\"query\":\"第二搜\"}"`)
		default:
			sseReply(w, raw)
		}
	})
	res := runStreamSearchTurn(t, handler, "需要两轮搜索的问题", true, false)
	assertCanonicalStreamReply(t, res, "最终说明。")
	if res.searchCalls != 2 {
		t.Errorf("search seam calls = %d, want 2", res.searchCalls)
	}
	done := terminalStreamEvent(t, res.events, "done")
	if done["searches"] != float64(2) || done["search_state"] != "ok" {
		t.Errorf("done search metadata = %v", done)
	}
}

func TestChatStreamSearch_CrossChunkToolPrefixStripped(t *testing.T) {
	var round atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		switch round.Add(1) {
		case 1:
			sseToolCallReply(w, "call_1", "web_search", `"{\"query\":\"第一搜\"}"`)
		case 2:
			sseToolCallReply(w, "call_2", "web_search", `"{\"query\":\"第二搜\"}"`)
		default:
			sseReply(w, "答案正常，尾部是残缺标记<tool")
		}
	})
	res := runStreamSearchTurn(t, handler, "跨分片标记问题", true, false)
	assertCanonicalStreamReply(t, res, "答案正常，尾部是残缺标记")
	if res.searchCalls != 2 {
		t.Errorf("search seam calls = %d, want 2", res.searchCalls)
	}
}

func TestChatStreamSearch_JSONMarkupExecuted(t *testing.T) {
	var round atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if round.Add(1) == 1 {
			sseReply(w, `<tool_call>{"name":"web_search","arguments":{"query":"JSON 查询"}}</tool_call>`)
			return
		}
		sseReply(w, "JSON 工具调用已执行。")
	})
	res := runStreamSearchTurn(t, handler, "测试 JSON 工具形态", true, false)
	assertCanonicalStreamReply(t, res, "JSON 工具调用已执行。")
	if res.searchCalls != 1 {
		t.Errorf("search seam calls = %d, want 1", res.searchCalls)
	}
}

func TestChatStreamSearch_FencedMarkupExecuted(t *testing.T) {
	var round atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if round.Add(1) == 1 {
			sseReply(w, "```tool_call\n{\"function\":{\"name\":\"web_search\",\"arguments\":{\"query\":\"围栏查询\"}}}\n```")
			return
		}
		sseReply(w, "围栏工具调用已执行。")
	})
	res := runStreamSearchTurn(t, handler, "测试围栏工具形态", true, false)
	assertCanonicalStreamReply(t, res, "围栏工具调用已执行。")
	if res.searchCalls != 1 {
		t.Errorf("search seam calls = %d, want 1", res.searchCalls)
	}
}
