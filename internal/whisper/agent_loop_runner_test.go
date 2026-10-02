package whisper

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// ─── T6-5.1: agent_loop_runner.go ───────────────────────────────

func TestParseSingleToolCall_Valid(t *testing.T) {
	raw := "{\"name\":\"use_computer\",\"arguments\":\"{\\\"action\\\":\\\"list_folder\\\"}\"}"
	call := parseSingleToolCall(raw)
	if call == nil {
		t.Fatal("合法工具调用应解析成功")
	}
	if call.Name != "use_computer" {
		t.Errorf("工具名错误: %q", call.Name)
	}
	if call.Args["action"] != "list_folder" {
		t.Errorf("arguments 解析错误: %v", call.Args)
	}
}

func TestParseSingleToolCall_Invalid(t *testing.T) {
	if got := parseSingleToolCall("not json at all"); got != nil {
		t.Errorf("非法 JSON 应返回 nil, got %+v", got)
	}
	if got := parseSingleToolCall(""); got != nil {
		t.Errorf("空串应返回 nil, got %+v", got)
	}
}

func TestParseSingleToolCall_BadArguments(t *testing.T) {
	raw := "{\"name\":\"x\",\"arguments\":\"not-json\"}"
	call := parseSingleToolCall(raw)
	if call == nil || call.Args["raw"] != "not-json" {
		t.Errorf("arguments 非 JSON 应放入 raw 字段, got %+v", call)
	}
}

func TestParseToolCallsFromReply(t *testing.T) {
	block := "{\"name\":\"web_search\",\"arguments\":\"{\\\"query\\\":\\\"测试\\\"}\"}"
	reply := "我先看看目录" + "\n" + "```tool_call\n" + block + "\n```\n" + "再回复用户"
	text, calls := parseToolCallsFromReply(reply)
	if len(calls) != 1 || calls[0].Name != "web_search" {
		t.Fatalf("应解析出 1 个工具调用, got %v", calls)
	}
	if !strings.Contains(text, "我先看看目录") || !strings.Contains(text, "再回复用户") {
		t.Errorf("正文应保留工具块外文本, got %q", text)
	}
	if strings.Contains(text, "tool_call") {
		t.Errorf("正文不应含工具块标记, got %q", text)
	}
}

func TestBuildInitialMessages_NoPlan(t *testing.T) {
	r := &AgentLoopRunner{SystemPrompt: "你是助手"}
	messages := r.buildInitialMessages("用户消息", nil)
	if len(messages) != 2 {
		t.Fatalf("无计划应为 2 条消息, got %d", len(messages))
	}
	if messages[0]["role"] != "system" || messages[1]["role"] != "user" {
		t.Errorf("消息角色错误: %v", messages)
	}
}

func TestBuildInitialMessages_WithPlan(t *testing.T) {
	r := &AgentLoopRunner{SystemPrompt: "你是助手"}
	plan := &TaskPlan{Title: "整理桌面", Steps: []TaskStep{{Index: 1, Description: "列出文件", Status: "pending"}}}
	messages := r.buildInitialMessages("开始", plan)
	if len(messages) != 3 {
		t.Fatalf("有计划应为 3 条消息, got %d", len(messages))
	}
	planMsg, _ := messages[2]["content"].(string)
	if !strings.Contains(planMsg, "整理桌面") || !strings.Contains(planMsg, "当前任务计划") {
		t.Errorf("第 3 条应为计划块, got %q", planMsg)
	}
}

func TestCallLLM_PlainText(t *testing.T) {
	r := &AgentLoopRunner{Llm: llmStub{reply: "任务已完成"}}
	text, calls, err := r.callLLM([]map[string]interface{}{{"role": "user", "content": "hi"}})
	if err != nil || len(calls) != 0 || text != "任务已完成" {
		t.Errorf("纯文本回复解析失败: text=%q calls=%v err=%v", text, calls, err)
	}
}

func TestCallLLM_WithTool(t *testing.T) {
	block := "{\"name\":\"use_computer\",\"arguments\":\"{\\\"action\\\":\\\"stat_file\\\",\\\"path\\\":\\\"C:\\\\\\\\a\\\\\\\\b.txt\\\"}\"}"
	reply := "```tool_call\n" + block + "\n```"
	r := &AgentLoopRunner{Llm: llmStub{reply: reply}}
	text, calls, err := r.callLLM([]map[string]interface{}{{"role": "user", "content": "hi"}})
	if err != nil {
		t.Fatalf("callLLM 错误: %v", err)
	}
	if len(calls) != 1 || calls[0].Name != "use_computer" {
		t.Fatalf("应解析出 use_computer, got %v (text=%q)", calls, text)
	}
	if calls[0].Args["action"] != "stat_file" {
		t.Errorf("args 解析错误: %v", calls[0].Args)
	}
}

func TestCallLLM_Error(t *testing.T) {
	r := &AgentLoopRunner{Llm: llmStub{err: context.Canceled}}
	_, _, err := r.callLLM([]map[string]interface{}{{"role": "user", "content": "hi"}})
	if err == nil {
		t.Fatal("LLM 错误应向上返回")
	}
}

func TestExecuteToolBatch_UnknownTool(t *testing.T) {
	r := &AgentLoopRunner{}
	results := r.executeToolBatch([]AgentAction{{Name: "some_tool"}})
	if len(results) != 1 || results[0].Name != "some_tool" || !strings.Contains(results[0].Content, "未知工具") {
		t.Errorf("未知工具应返回提示, got %v", results)
	}
}

func TestExecuteToolBatch_AppendMemory(t *testing.T) {
	r := &AgentLoopRunner{}
	results := r.executeToolBatch([]AgentAction{{Name: "append_memory"}})
	if len(results) != 1 || results[0].Content != "记忆已记录" {
		t.Errorf("append_memory 应返回已记录, got %v", results)
	}
}

// TestExecuteToolBatch_UseComputerNilRouter Router 未接线时必须产出**诚实回执**
// （线3 IN4-05 现场复核：旧行为「什么都不追加」= 模型拿到空结果集、用户侧看不到
// 动作没执行；旧断言把该静默失败当期望，已随缺口修复一并更新）。
func TestExecuteToolBatch_UseComputerNilRouter(t *testing.T) {
	r := &AgentLoopRunner{}
	results := r.executeToolBatch([]AgentAction{{Name: "use_computer", Args: map[string]string{"action": "list_folder"}}})
	if len(results) != 1 {
		t.Fatalf("Router 为 nil 时应产出 1 条诚实回执, got %v", results)
	}
	if results[0].Name != "use_computer" || !strings.Contains(results[0].Content, "未启用") ||
		!strings.Contains(results[0].Content, "未执行") {
		t.Errorf("回执应经既有 use_computer 通道说明「未启用/未执行」, got %+v", results[0])
	}
}

func TestShouldContinue(t *testing.T) {
	r := &AgentLoopRunner{}
	if !r.shouldContinue([]ToolResultForFollowUp{{Name: "use_computer"}}) {
		t.Error("use_computer 结果应继续循环")
	}
	if r.shouldContinue([]ToolResultForFollowUp{{Name: "web_search"}}) {
		t.Error("web_search 结果不应继续循环")
	}
	if r.shouldContinue(nil) {
		t.Error("空结果不应继续循环")
	}
}

func TestDefaultAgentLoopRunner(t *testing.T) {
	r := DefaultAgentLoopRunner(llmStub{reply: "x"})
	if r.MaxRounds != DesktopAgentMaxToolRounds {
		t.Errorf("默认 MaxRounds 应为 %d, got %d", DesktopAgentMaxToolRounds, r.MaxRounds)
	}
	if r.Llm == nil {
		t.Error("默认运行器应注入 Llm")
	}
}

// recordingLlm 记录送入 LLM 的 user prompt，供「诚实回执是否真的回传给模型」断言。
type recordingLlm struct {
	reply string
	seen  []string
}

func (m *recordingLlm) Chat(systemPrompt, userPrompt string) (string, error) {
	m.seen = append(m.seen, userPrompt)
	return m.reply, nil
}

func TestRunAgentLoop_FinalReplyBranch(t *testing.T) {
	// Router == nil → use_computer 产出诚实回执 → 回执进入下一轮 prompt（旧行为是
	// 空结果集 + 静默 AllPassed 收尾；线3 已按诚实降级口径修正，本用例随之更新）。
	innerArgs, _ := json.Marshal(map[string]string{"action": "list_folder", "path": `C:\x`})
	block, _ := json.Marshal(map[string]interface{}{"name": "use_computer", "arguments": string(innerArgs)})
	reply := "```tool_call\n" + string(block) + "\n```\n"
	llm := &recordingLlm{reply: reply}
	r := &AgentLoopRunner{Llm: llm, MaxRounds: 3}
	result := r.RunAgentLoop(context.Background(), "帮我看看", nil)
	if result.TotalResults == 0 {
		t.Fatalf("诚实回执应计入工具结果, got %+v", result)
	}
	// 第 2 轮及之后的 prompt 必须带上这条回执（用户/模型可见「没执行」）
	found := false
	for i, p := range llm.seen {
		if i == 0 {
			continue
		}
		if strings.Contains(p, "未启用") && strings.Contains(p, "未执行") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("诚实回执应回传给模型, prompts=%v", llm.seen)
	}
}

func TestRunAgentLoop_NoTools(t *testing.T) {
	r := &AgentLoopRunner{Llm: llmStub{reply: "直接完成"}, MaxRounds: 3}
	result := r.RunAgentLoop(context.Background(), "帮我看看", nil)
	if result.FinalReply != "直接完成" {
		t.Errorf("FinalReply 错误: %q", result.FinalReply)
	}
	if !result.AllPassed || result.ToolRounds != 0 {
		t.Errorf("无工具轮应直接通过, got %+v", result)
	}
}

func TestRunAgentLoop_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := &AgentLoopRunner{Llm: llmStub{reply: "x"}, MaxRounds: 3}
	result := r.RunAgentLoop(ctx, "hi", nil)
	if result.FinalReply != "任务已取消" {
		t.Errorf("取消上下文应返回已取消, got %q", result.FinalReply)
	}
}

func TestRunAgentLoop_MaxRounds(t *testing.T) {
	dir := t.TempDir()
	innerArgs, _ := json.Marshal(map[string]string{"action": "list_folder", "path": dir})
	block, _ := json.Marshal(map[string]interface{}{"name": "use_computer", "arguments": string(innerArgs)})
	reply := "```tool_call\n" + string(block) + "\n```\n"
	r := &AgentLoopRunner{
		Llm:       llmStub{reply: reply},
		Router:    &RouterContext{DataRoot: t.TempDir(), CWD: dir, AllowFileWrite: true},
		MaxRounds: 1,
		SessionID: "sess-1",
	}
	result := r.RunAgentLoop(context.Background(), "帮我看看", nil)
	if !strings.Contains(result.FinalReply, "最大执行轮数") {
		t.Errorf("应到达最大轮数, got %q", result.FinalReply)
	}
	if result.ToolRounds != 1 || result.TotalResults != 1 {
		t.Errorf("应执行 1 轮 1 个结果, got %+v", result)
	}
}
