package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

// ---- action-3 取消族（pan_1794012727813：取消不得丢已生成内容）----

// sseContentChunk 单条正文 delta（不收尾，供分片编排）。
func sseContentChunk(w http.ResponseWriter, text string) {
	payload, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"index": 0, "delta": map[string]any{"content": text}}},
	})
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = w.Write([]byte("data: " + string(payload) + "\n\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// ssePartialThenHang 流出一段正文后挂住等客户端断连（取消把 HTTP ctx 打断）。
// 信号前留出本地回环消费窗：工具轮正文先完整缓冲、轮结束才外发，mid-round
// 取消没有客户端可见同步点（delta 事件不存在），只能以「服务端已流出+消费窗」
// 保证 chunk 已进客户端 sbContent 缓冲——否则取消先到、缓冲为空、抢救无物。
func ssePartialThenHang(w http.ResponseWriter, r *http.Request, text string, signal chan struct{}) {
	sseContentChunk(w, text)
	if signal != nil {
		time.Sleep(500 * time.Millisecond)
		close(signal)
	}
	<-r.Context().Done()
}

// assertCancelTerminal 断言终态为 cancelled、无 error 帧、reply 非空且与
// delta 拼接一致、DB assistant 行同文（取消轮照常落库，action-3）。
func assertCancelTerminal(t *testing.T, res streamSearchResult, want string) {
	t.Helper()
	terminalStreamEvent(t, res.events, "cancelled")
	for _, ev := range res.events {
		if ev["type"] == "error" {
			t.Fatalf("取消轮不得发 error 帧: %v", ev)
		}
	}
	var delta strings.Builder
	var reply string
	for _, ev := range res.events {
		switch ev["type"] {
		case "delta":
			if s, ok := ev["content"].(string); ok {
				delta.WriteString(s)
			}
		case "cancelled":
			if s, ok := ev["reply"].(string); ok {
				reply = s
			}
		}
	}
	if reply != want {
		t.Errorf("cancelled.reply = %q, want %q", reply, want)
	}
	if delta.String() != want {
		t.Errorf("delta 拼接 = %q, want %q", delta.String(), want)
	}
	if len(res.msgs) != 2 {
		t.Fatalf("消息数 = %d, want 2（取消轮保留正文照常落库）", len(res.msgs))
	}
	if got := res.msgs[1].Content; got != want {
		t.Errorf("DB assistant = %q, want %q", got, want)
	}
}

// 首轮取消（非工具路径）：正文已外发、流未结束时取消 → cancelled 帧保留
// partial 且落库同文；不得裸奔 error「请求已取消」。
func TestChatStreamCancel_PlainNoTool_KeepsPartial(t *testing.T) {
	firstDelta := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ssePartialThenHang(w, r, "故事的开头是这样", firstDelta)
	})
	a := newChatServiceTestAppWithHandler(t, handler)
	topic, err := a.ChatTopicCreate("cancel-plain", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	fixed := "chat_cancel_plain_" + strings.ReplaceAll(t.Name(), "/", "_")
	origRunID := newChatStreamRunID
	newChatStreamRunID = func() string { return fixed }
	t.Cleanup(func() { newChatStreamRunID = origRunID })

	// 非工具路径 delta 边流边外发，可在首个 delta 事件确定性取消（reply
	// builder 先写后 emit，事件被消费即后端已含该段正文）。
	events := captureStreamEventsOn(t, a, "chat-stream:"+fixed, func() {
		if _, err := a.ChatStreamPlain(topic.ID, "讲个故事", false, false, false); err != nil {
			t.Errorf("ChatStreamPlain: %v", err)
		}
	}, func(ev map[string]interface{}) {
		if ev["type"] == "delta" {
			if !a.ChatStreamCancel(fixed) {
				t.Error("ChatStreamCancel = false, run 未登记")
			}
		}
	})
	res := streamSearchResult{events: events}
	if msgs, err := a.ChatMessagesList(topic.ID); err == nil {
		res.msgs = msgs
	}
	assertCancelTerminal(t, res, "故事的开头是这样")
}

// 首轮取消（工具轮、零正文）：cancelled 空终态、不落库 assistant——对齐
// persona 空轮语义，不伪造成功也不报「未生成可交付正文」。
func TestChatStreamCancel_PlainToolLoop_EmptyRound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // 一字未发即被取消
	})
	a := newChatServiceTestAppWithHandler(t, handler)
	topic, err := a.ChatTopicCreate("cancel-empty", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	fixed := "chat_cancel_empty_" + strings.ReplaceAll(t.Name(), "/", "_")
	origRunID := newChatStreamRunID
	newChatStreamRunID = func() string { return fixed }
	t.Cleanup(func() { newChatStreamRunID = origRunID })

	events := captureStreamEvents(t, a, "chat-stream:"+fixed, func() {
		if _, err := a.ChatStreamPlain(topic.ID, "讲个故事", true, false, false); err != nil {
			t.Errorf("ChatStreamPlain: %v", err)
		}
		if !a.ChatStreamCancel(fixed) {
			t.Error("ChatStreamCancel = false, run 未登记")
		}
	})
	terminalStreamEvent(t, events, "cancelled")
	for _, ev := range events {
		if ev["type"] == "error" {
			t.Fatalf("取消空轮不得发 error 帧: %v", ev)
		}
	}
	msgs, err := a.ChatMessagesList(topic.ID)
	if err != nil {
		t.Fatalf("ChatMessagesList: %v", err)
	}
	// 空 assistant 不落库（不伪造成功），但用户输入单条落 user 行不随取消湮灭
	if len(msgs) != 1 {
		t.Fatalf("取消空轮应仅落 user 行，消息数 = %d: %+v", len(msgs), msgs)
	}
	if msgs[0].Role != "user" || msgs[0].Content != "讲个故事" {
		t.Fatalf("user 行异常: %+v", msgs[0])
	}
}

// 跨轮取消（工具轮）：第 1 轮交付散文并发起搜索、第 2 轮正文外发中取消 →
// cancelled 帧必须同时保留第 1 轮散文与第 2 轮残段（此前轮 res.Reply +
// 当前轮 sbContent 抢救，action-3 病灶②）。
func TestChatStreamCancel_PlainToolLoop_CrossRoundKeepsProse(t *testing.T) {
	var round atomic.Int64
	firstDelta := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch round.Add(1) {
		case 1:
			// 散文 + 工具调用同轮：正文先外发并累计，随后执行搜索
			sseContentChunk(w, "让我先查一下。")
			sseToolCallReply(w, "call_1", "web_search", `"{\"query\":\"跨轮取消\"}"`)
		default:
			ssePartialThenHang(w, r, "查到了：", firstDelta)
		}
	})
	a := newChatServiceTestAppWithHandler(t, handler)
	topic, err := a.ChatTopicCreate("cancel-cross", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	fixed := "chat_cancel_cross_" + strings.ReplaceAll(t.Name(), "/", "_")
	origRunID := newChatStreamRunID
	newChatStreamRunID = func() string { return fixed }
	t.Cleanup(func() { newChatStreamRunID = origRunID })
	origSearch := chatWebSearch
	chatWebSearch = func(q string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: q, Text: "罐头结果"}, nil
	}
	t.Cleanup(func() { chatWebSearch = origSearch })

	events := captureStreamEvents(t, a, "chat-stream:"+fixed, func() {
		if _, err := a.ChatStreamPlain(topic.ID, "边说边查", true, false, false); err != nil {
			t.Errorf("ChatStreamPlain: %v", err)
		}
		select {
		case <-firstDelta:
		case <-time.After(3 * time.Second):
			t.Error("等待第 2 轮正文超时")
			return
		}
		if !a.ChatStreamCancel(fixed) {
			t.Error("ChatStreamCancel = false, run 未登记")
		}
	})
	res := streamSearchResult{events: events}
	if msgs, err := a.ChatMessagesList(topic.ID); err == nil {
		res.msgs = msgs
	}
	assertCancelTerminal(t, res, "让我先查一下。查到了：")
}

// persona 路径取消（工具轮）：第 2 轮正文外发中取消 ctx → 返回正文须含
// 残段，不因取消静默归零（whisper 工具分支同口径，action-3）。
func TestWhisperChatCancel_ToolLoop_KeepsPartial(t *testing.T) {
	firstDelta := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "搜索查询拟写器") {
			sseReply(w, "测试 查询")
			return
		}
		if strings.Contains(string(body), `"role":"tool"`) {
			ssePartialThenHang(w, r, "查到了：", firstDelta)
			return
		}
		sseToolCallReply(w, "call_1", "web_search", `"{\"query\":\"角色取消\"}"`)
	})
	a := newChatServiceTestAppWithHandler(t, handler)
	personalityID := "persona_action3_cancel"
	cleanupWhisperSession(t, personalityID)
	t.Cleanup(func() { cleanupWhisperSession(t, personalityID) })
	orig := chatWebSearch
	chatWebSearch = func(q string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: q, Text: "罐头结果"}, nil
	}
	t.Cleanup(func() { chatWebSearch = orig })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type coreResult struct {
		reply string
		err   error
	}
	done := make(chan coreResult, 1)
	go func() {
		out, err := a.whisperChatCore(ctx, "帮我查一下", personalityID, "", false, false, true, false,
			func(string) {
				select {
				case <-firstDelta:
				default:
					close(firstDelta)
				}
			}, nil)
		reply, _ := out["reply"].(string)
		done <- coreResult{reply, err}
	}()
	select {
	case <-firstDelta:
	case <-time.After(3 * time.Second):
		t.Fatal("等待角色流首段正文超时")
	}
	cancel()
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("whisperChatCore: %v", got.err)
		}
		if !strings.Contains(got.reply, "查到了：") {
			t.Fatalf("角色取消轮丢正文: %q", got.reply)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("whisperChatCore 取消后未返回")
	}
	time.Sleep(300 * time.Millisecond) // 异步记忆落库与 TempDir 清理竞速（house band-aid）
}
