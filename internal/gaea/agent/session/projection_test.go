package session

import (
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

func raw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// 三类事件（user/assistant/tool）→ 消息
func TestProjectMessagesThreeKinds(t *testing.T) {
	entries := []LogEntry{
		{Seq: 1, Kind: KindUserMessage, Payload: raw(t, userLogPayload{Content: "你好"})},
		{Seq: 2, Kind: KindAssistantStarted, Payload: raw(t, assistantLogPayload{ID: "m2"})},
		{Seq: 3, Kind: KindAssistantDelta, Payload: raw(t, map[string]string{"text": "你"})},
		{Seq: 4, Kind: KindAssistantDelta, Payload: raw(t, map[string]string{"text": "好"})},
		{Seq: 5, Kind: KindToolCall, Payload: raw(t, toolCallLogPayload{ID: "t1", Name: "read_file", Args: "{}"})},
		{Seq: 6, Kind: KindToolResult, Payload: raw(t, toolResultLogPayload{ID: "t1", Output: "file body"})},
		{Seq: 7, Kind: KindAssistantMessage, Payload: raw(t, assistantLogPayload{ID: "m3", Text: "完成"})},
	}
	msgs := ProjectMessages(entries)
	if len(msgs) != 4 {
		t.Fatalf("messages = %d, want 4", len(msgs))
	}
	// 1 user
	if msgs[0].Role != provider.RoleUser || msgs[0].Content != "你好" {
		t.Errorf("msg0 = %+v", msgs[0])
	}
	// 2 assistant：delta 累积 + tool call 附加
	if msgs[1].Role != provider.RoleAssistant || msgs[1].Content != "你好" {
		t.Errorf("msg1 = %+v", msgs[1])
	}
	if len(msgs[1].ToolCalls) != 1 || msgs[1].ToolCalls[0].ID != "t1" {
		t.Errorf("msg1 toolcalls = %+v", msgs[1].ToolCalls)
	}
	// 3 tool result
	if msgs[2].Role != provider.RoleTool || msgs[2].Content != "file body" || msgs[2].ToolCallID != "t1" {
		t.Errorf("msg2 = %+v", msgs[2])
	}
	// 4 assistant message（完整）
	if msgs[3].Role != provider.RoleAssistant || msgs[3].Content != "完成" {
		t.Errorf("msg3 = %+v", msgs[3])
	}
}

// 事件级 kind 同样投影（text/message/tool_dispatch/tool_result）
func TestProjectMessagesEventKinds(t *testing.T) {
	entries := []LogEntry{
		{Seq: 1, Kind: "turn_started", Payload: raw(t, map[string]string{})},
		{Seq: 2, Kind: "text", Payload: raw(t, map[string]string{"text": "a"})},
		{Seq: 3, Kind: "text", Payload: raw(t, map[string]string{"text": "b"})},
		{Seq: 4, Kind: "message", Payload: raw(t, assistantLogPayload{Text: "ab", Reasoning: "r"})},
		{Seq: 5, Kind: "tool_dispatch", Payload: raw(t, toolCallLogPayload{ID: "t1", Name: "bash", Args: "{}"})},
		{Seq: 6, Kind: "tool_result", Payload: raw(t, toolResultLogPayload{ID: "t1", Output: "o"})},
		{Seq: 7, Kind: "usage", Payload: raw(t, usageLogPayload{PromptTokens: 1})},
		{Seq: 8, Kind: "notice", Payload: raw(t, map[string]any{"level": "info"})},
		{Seq: 9, Kind: "phase", Payload: raw(t, map[string]string{"text": "x"})},
		{Seq: 10, Kind: "turn_done", Payload: raw(t, map[string]string{})},
	}
	msgs := ProjectMessages(entries)
	// message 事件是全文：原地替换流式 delta 累积的同一条 assistant 消息，
	// 后续 tool_dispatch 附加工具调用 → 2 条消息（与运行期 executor 一致）。
	if len(msgs) != 2 {
		t.Fatalf("messages = %d, want 2 (assistant ab+toolcall, tool result)", len(msgs))
	}
	if msgs[0].Role != provider.RoleAssistant || msgs[0].Content != "ab" || msgs[0].ReasoningContent != "r" {
		t.Errorf("msg0 = %+v", msgs[0])
	}
	if len(msgs[0].ToolCalls) != 1 || msgs[0].ToolCalls[0].ID != "t1" {
		t.Errorf("msg0 toolcalls = %+v", msgs[0].ToolCalls)
	}
	if msgs[1].Role != provider.RoleTool || msgs[1].Content != "o" {
		t.Errorf("msg1 = %+v", msgs[1])
	}
}

// 非消息事件不投影
func TestProjectMessagesIgnoresNonMessageKinds(t *testing.T) {
	entries := []LogEntry{
		{Seq: 1, Kind: "usage", Payload: raw(t, usageLogPayload{})},
		{Seq: 2, Kind: "notice", Payload: raw(t, map[string]any{})},
		{Seq: 3, Kind: "approval_request", Payload: raw(t, map[string]any{})},
		{Seq: 4, Kind: "turn_done", Payload: raw(t, map[string]string{})},
	}
	if msgs := ProjectMessages(entries); len(msgs) != 0 {
		t.Fatalf("messages = %d, want 0", len(msgs))
	}
}

// tool_result 终结当前 assistant 消息：其后的 text 开始新 assistant 消息
func TestProjectMessagesToolResultFinalizesPending(t *testing.T) {
	entries := []LogEntry{
		{Seq: 1, Kind: KindToolCall, Payload: raw(t, toolCallLogPayload{ID: "t1", Name: "bash", Args: "{}"})},
		{Seq: 2, Kind: KindToolResult, Payload: raw(t, toolResultLogPayload{ID: "t1", Output: "o"})},
		{Seq: 3, Kind: KindAssistantDelta, Payload: raw(t, map[string]string{"text": "再写"})},
	}
	msgs := ProjectMessages(entries)
	if len(msgs) != 3 {
		t.Fatalf("messages = %d, want 3", len(msgs))
	}
	if msgs[0].Role != provider.RoleAssistant || len(msgs[0].ToolCalls) != 1 {
		t.Errorf("msg0 = %+v", msgs[0])
	}
	if msgs[1].Role != provider.RoleTool {
		t.Errorf("msg1 = %+v", msgs[1])
	}
	if msgs[2].Role != provider.RoleAssistant || msgs[2].Content != "再写" {
		t.Errorf("msg2 = %+v", msgs[2])
	}
}

// system_message 投影为 system 消息（迁移路径）
func TestProjectMessagesSystem(t *testing.T) {
	entries := []LogEntry{
		{Seq: 1, Kind: KindSystemMessage, Payload: raw(t, userLogPayload{Content: "sys"})},
		{Seq: 2, Kind: KindUserMessage, Payload: raw(t, userLogPayload{Content: "u"})},
	}
	msgs := ProjectMessages(entries)
	if len(msgs) != 2 || msgs[0].Role != provider.RoleSystem || msgs[0].Content != "sys" {
		t.Fatalf("messages = %+v", msgs)
	}
}
