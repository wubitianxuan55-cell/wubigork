package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// B7 有界工具循环单测（2026-11-06 用户拍板）：搜索注册为真工具，模型自己判
// 断要不要搜/搜什么；plain/persona 板块按钮语义=允许联网总开关，关键词门控
// 退役。假 LLM 按线格式喂 tool_calls（delta 分片+finish_reason=tool_calls）。
func sseToolCallReply(w http.ResponseWriter, id, name, args string) {
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = w.Write([]byte(`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"` + id + `","type":"function","function":{"name":"` + name + `","arguments":` + args + `}}]}}]}` + "\n\n"))
	_, _ = w.Write([]byte(`data: {"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}` + "\n\n"))
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// TestChatToolLoop_SearchExecutedAndFedBack 第一轮模型发起 web_search → 工具
// 结果回填 → 第二轮交付正文；检索词=模型拟写的 args.query。
func TestChatToolLoop_SearchExecutedAndFedBack(t *testing.T) {
	var round atomic.Int64
	var capturedQuery atomic.Value
	var sawToolResult atomic.Bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		n := round.Add(1)
		_ = n
		if strings.Contains(string(body), `"role":"tool"`) {
			sawToolResult.Store(true)
		}
		if n == 1 {
			sseToolCallReply(w, "call_1", "web_search", `"{\"query\":\"成都 明天 天气\"}"`)
			return
		}
		sseReply(w, "成都明天小雨，14 到 21 度。")
	})
	a := newChatServiceTestAppWithHandler(t, handler)
	orig := chatWebSearch
	chatWebSearch = func(q string) (chatSearchOutcome, error) {
		capturedQuery.Store(q)
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: q, Text: "1. 成都天气 [罐头]"}, nil
	}
	t.Cleanup(func() { chatWebSearch = orig })

	topic, err := a.ChatTopicCreate("B7循环", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	out, err := a.ChatSend(topic.ID, "明天成都的天气怎么样", "plain", true, false, true)
	if err != nil {
		t.Fatalf("ChatSend: %v", err)
	}
	if !sawToolResult.Load() {
		t.Fatal("工具结果应回填进第二轮 messages（role=tool）")
	}
	if got, _ := capturedQuery.Load().(string); got != "成都 明天 天气" {
		t.Fatalf("检索词应为模型拟写的 args.query，got %q", got)
	}
	if out["reply"] != "成都明天小雨，14 到 21 度。" {
		t.Errorf("reply = %v", out["reply"])
	}
}

// TestChatToolLoop_SearchErrorStillAnswers 搜索适配器故障 → tool 消息带如实
// 占位 → 模型仍交付正文（失败可见不阻断对话，对齐旧占位语义）。
func TestChatToolLoop_SearchErrorStillAnswers(t *testing.T) {
	var round atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		n := round.Add(1)
		if strings.Contains(string(body), `"role":"tool"`) && strings.Contains(string(body), "联网搜索暂不可用") {
			sseReply(w, "搜索挂了，我如实说明：暂无实时信息。")
			return
		}
		if n == 1 {
			sseToolCallReply(w, "call_2", "web_search", `"{\"query\":\"今天新闻\"}"`)
			return
		}
		sseReply(w, "兜底回复")
	})
	a := newChatServiceTestAppWithHandler(t, handler)
	orig := chatWebSearch
	chatWebSearch = func(string) (chatSearchOutcome, error) { return chatSearchOutcome{}, errString("网络不可达") }
	t.Cleanup(func() { chatWebSearch = orig })

	topic, err := a.ChatTopicCreate("B7降级", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	out, err := a.ChatSend(topic.ID, "有什么新闻", "plain", true, false, true)
	if err != nil {
		t.Fatalf("搜索失败不应报错: %v", err)
	}
	if out["reply"] == "" {
		t.Errorf("占位后模型仍应交付正文: %v", out)
	}
}

// TestChatToolLoop_SearchOffNoTools 搜索关=不传工具（零风险回滚面：请求与
// 旧版逐字节一致），模型直答。
func TestChatToolLoop_SearchOffNoTools(t *testing.T) {
	var sawTools atomic.Bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), `"tools"`) {
			sawTools.Store(true)
		}
		sseReply(w, "好的")
	})
	a := newChatServiceTestAppWithHandler(t, handler)

	topic, err := a.ChatTopicCreate("B7关闭", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	out, err := a.ChatSend(topic.ID, "普通闲聊", "plain", false, false, false)
	if err != nil {
		t.Fatalf("ChatSend: %v", err)
	}
	if sawTools.Load() {
		t.Fatal("搜索关=不应携带 tools 字段")
	}
	if out["reply"] != "好的" {
		t.Errorf("reply = %v", out["reply"])
	}
}

// TestChatToolLoop_ForceSearchSystemAtTop 实机 500 回归（2026-11-06）：强制
// 搜索的指令必须并入首条 system，不得新增中间 system 消息——qwen 模板拒绝
// 「System message must be at the beginning」（引擎 HTTP 500）。
func TestChatToolLoop_ForceSearchSystemAtTop(t *testing.T) {
	var round atomic.Int64
	var systemsAtNonZero atomic.Bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Messages []struct {
				Role string `json:"role"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &req); err == nil {
			for i, m := range req.Messages {
				if m.Role == "system" && i != 0 {
					systemsAtNonZero.Store(true)
				}
			}
		}
		n := round.Add(1)
		if n == 1 {
			sseToolCallReply(w, "call_fs", "web_search", `"{\"query\":\"成都 天气\"}"`)
			return
		}
		sseReply(w, "晴天")
	})
	a := newChatServiceTestAppWithHandler(t, handler)
	topic, err := a.ChatTopicCreate("B7强搜", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}
	if _, err := a.ChatSend(topic.ID, "明天成都的天气怎么样", "plain", true, false, true); err != nil {
		t.Fatalf("ChatSend: %v", err)
	}
	if systemsAtNonZero.Load() {
		t.Fatal("强制搜索轮出现非首位 system 消息（qwen 模板 500 病根）")
	}
}
