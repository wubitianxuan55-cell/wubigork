package app

// 原罪工具循环测试：可编排的多轮假 LLM（第 N 次请求返回预置 SSE），断言
//  1. 请求体形态——首轮带 7 个 tools、次轮把 assistant(tool_calls)+tool 结果接回；
//  2. 落库结果——正文 = 模型输出、extra.tools 轨迹、工具真实副作用；
//  3. 降级——模型/端点不支持 tools 时去掉工具重试一次，写作不中断；
//  4. 轮次上限与未知工具——不空转、如实回错、不落空消息。

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// sinScriptLLM 按轮次编排的假 LLM：记录每次请求体，第 N 次请求回 rounds[N]。
type sinScriptLLM struct {
	mu     sync.Mutex
	bodies []map[string]interface{}
	rounds []string
}

func (s *sinScriptLLM) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		s.mu.Lock()
		n := len(s.bodies)
		s.bodies = append(s.bodies, body)
		script := ""
		if n < len(s.rounds) {
			script = s.rounds[n]
		}
		s.mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(script))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})
}

func (s *sinScriptLLM) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.bodies)
}

func (s *sinScriptLLM) request(i int) map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i >= len(s.bodies) {
		return nil
	}
	return s.bodies[i]
}

// newSinToolLoopApp 装配测试 App：用户配置目录先重定向到临时 HOME（便签/大纲
// 落临时目录，不碰真机数据），再套用统一聊天测试装配。
func newSinToolLoopApp(t *testing.T, llm *sinScriptLLM) *App {
	t.Helper()
	sinTestHome(t)
	a := newChatServiceTestAppWithHandler(t, llm.handler())
	if err := a.core.SetFeatureModel("sin", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel(sin): %v", err)
	}
	return a
}

// sinSSELine 一行 SSE data（payload 走 json.Marshal，避免手写转义出错）。
func sinSSELine(payload interface{}) string {
	b, _ := json.Marshal(payload)
	return "data: " + string(b) + "\n\n"
}

// sinSSEContent 正文帧 + 结束帧。
func sinSSEContent(text string) string {
	return sinSSELine(map[string]interface{}{
		"choices": []interface{}{map[string]interface{}{
			"index": 0, "delta": map[string]interface{}{"content": text},
		}},
	}) + "data: [DONE]\n\n"
}

// sinSSEToolCall 工具调用帧（finish_reason=tool_calls 在 delta 内——解析器读这里）。
func sinSSEToolCall(id, name, args string) string {
	return sinSSELine(map[string]interface{}{
		"choices": []interface{}{map[string]interface{}{
			"index": 0,
			"delta": map[string]interface{}{
				"tool_calls": []interface{}{map[string]interface{}{
					"index": 0, "id": id, "type": "function",
					"function": map[string]interface{}{"name": name, "arguments": args},
				}},
				"finish_reason": "tool_calls",
			},
		}},
	}) + "data: [DONE]\n\n"
}

// sinReqMessages / sinReqTools 请求体取字段（缺字段返回 nil）。
// sinSSEToolMultiCall 单个响应里带多条工具调用（真机形态：模型一轮连发多个搜索）。
// 每项 = [id, name, args]。
func sinSSEToolMultiCall(calls [][3]string) string {
	tcs := make([]interface{}, 0, len(calls))
	for i, c := range calls {
		tcs = append(tcs, map[string]interface{}{
			"index": i, "id": c[0], "type": "function",
			"function": map[string]interface{}{"name": c[1], "arguments": c[2]},
		})
	}
	return sinSSELine(map[string]interface{}{
		"choices": []interface{}{map[string]interface{}{
			"index": 0,
			"delta": map[string]interface{}{
				"tool_calls":    tcs,
				"finish_reason": "tool_calls",
			},
		}},
	}) + "data: [DONE]\n\n"
}

func sinReqMessages(req map[string]interface{}) []interface{} {
	v, _ := req["messages"].([]interface{})
	return v
}

func sinReqTools(req map[string]interface{}) []interface{} {
	v, _ := req["tools"].([]interface{})
	return v
}

// TestSinToolLoopExecutesToolAndPersistsTrace 多轮主路径：模型先查便签、再写正文。
// 工具真实执行（便签落盘）、轨迹落 extra.tools、次轮消息数组接得上、正文原样落库。
func TestSinToolLoopExecutesToolAndPersistsTrace(t *testing.T) {
	llm := &sinScriptLLM{rounds: []string{
		sinSSEToolCall("call_1", sinToolNotes, `{"action":"write","content":"女主叫林晚"}`),
		sinSSEContent("雨落在窗上。"),
	}}
	a := newSinToolLoopApp(t, llm)
	story, err := a.SinTopicCreate("工具轮")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinStream(story.ID, "写开场"); err != nil {
		t.Fatalf("SinStream: %v", err)
	}
	msgs := waitSinMessages(t, a, story.ID, 2, 5*time.Second)
	assistant := msgs[1]
	if assistant.Content != "雨落在窗上。" {
		t.Errorf("正文 = %q, want 模型输出原样", assistant.Content)
	}

	var extra struct {
		Tools []sinToolTrace `json:"tools"`
	}
	if err := json.Unmarshal([]byte(assistant.Extra), &extra); err != nil {
		t.Fatalf("extra 不是合法 JSON: %q", assistant.Extra)
	}
	if len(extra.Tools) != 1 {
		t.Fatalf("extra.tools = %+v, want 1 条", extra.Tools)
	}
	tr := extra.Tools[0]
	if tr.Name != sinToolNotes || tr.ID != "call_1" {
		t.Errorf("轨迹 = %+v", tr)
	}
	if !strings.Contains(tr.Output, "已记录便签 #0") || tr.Error != "" {
		t.Errorf("轨迹输出 = %q, err = %q", tr.Output, tr.Error)
	}
	if tr.ReadOnly {
		t.Error("写类工具不应标只读")
	}
	if !strings.Contains(tr.Args, "林晚") {
		t.Errorf("轨迹应留原始参数: %q", tr.Args)
	}

	// 工具真的执行了（副作用落原罪自有目录）。
	if doc := loadSinNotes(mustSinNotesPath(t, story.ID)); len(doc.Notes) != 1 ||
		!strings.Contains(doc.Notes[0], "林晚") {
		t.Errorf("便签未落盘: %+v", doc)
	}

	// 请求形态：两轮；首轮带 7 个工具定义；次轮接上 assistant(tool_calls)+tool 结果。
	if llm.count() != 2 {
		t.Fatalf("请求轮次 = %d, want 2", llm.count())
	}
	if got := len(sinReqTools(llm.request(0))); got != 7 {
		t.Errorf("首轮 tools = %d, want 7", got)
	}
	req2 := sinReqMessages(llm.request(1))
	if len(req2) != 4 {
		t.Fatalf("次轮消息数 = %d, want 4（system/user/assistant/tool）", len(req2))
	}
	last, _ := req2[3].(map[string]interface{})
	if last["role"] != "tool" || last["tool_call_id"] != "call_1" {
		t.Errorf("次轮末条应为 tool 结果: %+v", last)
	}
	if content, _ := last["content"].(string); !strings.Contains(content, "已记录便签") {
		t.Errorf("tool 结果内容 = %q", content)
	}
	if assistantMsg, _ := req2[2].(map[string]interface{}); assistantMsg["role"] != "assistant" {
		t.Errorf("次轮第 3 条应为 assistant(tool_calls): %+v", assistantMsg)
	}
}

// TestSinToolLoopDegradesWhenToolsUnsupported 首轮带工具被端点 400 拒绝：去掉工具
// 重试一次，写作不中断、不留工具轨迹（降级路径的 a notice 由前端消费）。
func TestSinToolLoopDegradesWhenToolsUnsupported(t *testing.T) {
	var mu sync.Mutex
	var sawTools []bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, hasTools := body["tools"]
		mu.Lock()
		sawTools = append(sawTools, hasTools)
		mu.Unlock()
		if hasTools {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"tools not supported by this model"}}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sinSSEContent("纯文本续写。")))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	})

	sinTestHome(t)
	a := newChatServiceTestAppWithHandler(t, handler)
	if err := a.core.SetFeatureModel("sin", "herdsman", "qwen3-8b"); err != nil {
		t.Fatalf("SetFeatureModel(sin): %v", err)
	}
	story, err := a.SinTopicCreate("降级")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinStream(story.ID, "写开场"); err != nil {
		t.Fatalf("SinStream: %v", err)
	}
	msgs := waitSinMessages(t, a, story.ID, 2, 5*time.Second)
	if msgs[1].Content != "纯文本续写。" {
		t.Errorf("降级后正文 = %q", msgs[1].Content)
	}
	if strings.Contains(msgs[1].Extra, `"tools"`) {
		t.Errorf("降级路径不应留工具轨迹: %q", msgs[1].Extra)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(sawTools) != 2 || !sawTools[0] || sawTools[1] {
		t.Errorf("请求序列 = %v, want [带工具 不带工具]", sawTools)
	}
}

// TestSinToolLoopRoundCapAndUnknownTool 模型拿不存在的工具原地打转：
//   - 未知工具如实回错（可用清单照列）并接回消息数组；
//   - 轮次封顶 4 且最后一轮不带 tools；
//   - 全程没有正文时不落空消息（前端收到 error）。
func TestSinToolLoopRoundCapAndUnknownTool(t *testing.T) {
	call := sinSSEToolCall("call_1", "no_such_tool", "{}")
	llm := &sinScriptLLM{rounds: []string{call, call, call, call}}
	a := newSinToolLoopApp(t, llm)
	story, err := a.SinTopicCreate("空转")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinStream(story.ID, "写开场"); err != nil {
		t.Fatalf("SinStream: %v", err)
	}
	// 消息只有用户那条：没有正文就不落空助手消息。
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if msgs, err := a.SinMessages(story.ID); err == nil && len(msgs) >= 1 && llm.count() >= sinToolRoundsMax {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	if got := llm.count(); got > sinToolRoundsMax+1 {
		t.Fatalf("请求轮次 = %d, 超出上限 %d（工具轮 + 兜底收尾轮）", got, sinToolRoundsMax+1)
	}
	if got := len(sinReqTools(llm.request(sinToolRoundsMax - 1))); got != 0 {
		t.Errorf("收尾轮不应带 tools, got %d", got)
	}
	if got := len(sinReqTools(llm.request(llm.count() - 1))); got != 0 {
		t.Errorf("兜底收尾轮也不应带 tools, got %d", got)
	}
	// 收尾轮注入「工具阶段结束」系统指令（真机实测：不点名时模型会继续要工具）。
	sawNudge := false
	for _, m := range sinReqMessages(llm.request(sinToolRoundsMax - 1)) {
		if mm, ok := m.(map[string]interface{}); ok && mm["role"] == "system" {
			if c, _ := mm["content"].(string); strings.Contains(c, "工具阶段结束") {
				sawNudge = true
			}
		}
	}
	if !sawNudge {
		t.Error("收尾轮应注入「工具阶段结束」系统指令")
	}
	req1 := sinReqMessages(llm.request(1))
	last, _ := req1[len(req1)-1].(map[string]interface{})
	if content, _ := last["content"].(string); !strings.Contains(content, "不存在") ||
		!strings.Contains(content, sinToolNotes) {
		t.Errorf("未知工具应如实回错并列可用清单: %q", content)
	}
	msgs, err := a.SinMessages(story.ID)
	if err != nil {
		t.Fatalf("SinMessages: %v", err)
	}
	// 整轮没有正文 = 失败收尾，不落半截（与既有失败路径同口径：正文 = 模型
	// 输出，没有正文就没有可落的消息；前端收到 error 帧）。
	if len(msgs) != 0 {
		t.Errorf("无正文时不应落任何消息: %+v", msgs)
	}
}

// TestSinToolLoopFinalizeFallbackWritesStory 真机复现（2026-09-12 走查）：模型在收尾轮
// （不带 tools）仍回工具调用 —— 改前这一轮没有正文就整单报错、什么都没落库；
// 现在忽略该调用并用兜底收尾轮把正文逼出来。
func TestSinToolLoopFinalizeFallbackWritesStory(t *testing.T) {
	call := sinSSEToolCall("c1", sinToolNotes, `{"action":"list"}`)
	llm := &sinScriptLLM{rounds: []string{
		call, call, call, // 3 个工具轮（真执行）
		call, // 收尾轮：模型仍要工具 → 忽略
		sinSSEContent("她推开门，雨气扑面。"), // 兜底收尾轮：正文
	}}
	a := newSinToolLoopApp(t, llm)
	story, err := a.SinTopicCreate("收尾兜底")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinStream(story.ID, "写下一幕"); err != nil {
		t.Fatalf("SinStream: %v", err)
	}
	msgs := waitSinMessages(t, a, story.ID, 2, 8*time.Second)
	if got := msgs[1].Content; got != "她推开门，雨气扑面。" {
		t.Errorf("正文 = %q, want 兜底轮输出", got)
	}
	if got := llm.count(); got != sinToolRoundsMax+1 {
		t.Errorf("请求轮次 = %d, want %d", got, sinToolRoundsMax+1)
	}
}

// TestSinToolCallBudgetStopsRunawayCalls 真机走查实测的问题：不设闸门时模型会拿
// 同一个工具一路查到轮次封顶（实测 12 连搜，正文只剩 62 字）。预算按「一整轮用户
// 回合」计：同名 ≤3 次、总量 ≤8 次；超限不执行、如实回绝，模型只能用手上的信息收尾。
func TestSinToolCallBudgetStopsRunawayCalls(t *testing.T) {
	// 真机复现形态：同一个响应里一次要 5 次同名工具（实测模型一轮连搜 5 条）。
	multi := sinSSEToolMultiCall([][3]string{
		{"c1", sinToolNotes, `{"action":"list"}`},
		{"c2", sinToolNotes, `{"action":"list"}`},
		{"c3", sinToolNotes, `{"action":"list"}`},
		{"c4", sinToolNotes, `{"action":"list"}`},
		{"c5", sinToolNotes, `{"action":"list"}`},
	})
	llm := &sinScriptLLM{rounds: []string{
		multi,
		sinSSEContent("雨停了。"),
	}}
	a := newSinToolLoopApp(t, llm)
	story, err := a.SinTopicCreate("预算")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinStream(story.ID, "写下一幕"); err != nil {
		t.Fatalf("SinStream: %v", err)
	}
	msgs := waitSinMessages(t, a, story.ID, 2, 8*time.Second)
	if got := msgs[1].Content; got != "雨停了。" {
		t.Errorf("正文 = %q, want 模型输出（预算拦下后仍能收尾）", got)
	}
	var extra struct {
		Tools []sinToolTrace `json:"tools"`
	}
	if err := json.Unmarshal([]byte(msgs[1].Extra), &extra); err != nil {
		t.Fatalf("extra 不是合法 JSON: %q", msgs[1].Extra)
	}
	if len(extra.Tools) != 5 {
		t.Fatalf("轨迹条数 = %d, want 5（超限也要留痕，用户看得见被拦）", len(extra.Tools))
	}
	for i, tr := range extra.Tools {
		over := i >= sinToolCallMaxPerTool
		if over && tr.Error == "" {
			t.Errorf("第 %d 次同名调用应被预算拦下: %+v", i+1, tr)
		}
		if over && !strings.Contains(tr.Error, "上限") {
			t.Errorf("第 %d 次回绝理由应说明上限: %q", i+1, tr.Error)
		}
		if !over && tr.Error != "" {
			t.Errorf("第 %d 次在预算内，不该被拦: %q", i+1, tr.Error)
		}
	}
	// 到预算上限后模型仍在调工具：最后一轮（不带 tools）必须收尾成正文，
	// 且轮次不会无限增长。
	if got := llm.count(); got > sinToolRoundsMax {
		t.Errorf("请求轮次 = %d, 超出上限 %d", got, sinToolRoundsMax)
	}
}
