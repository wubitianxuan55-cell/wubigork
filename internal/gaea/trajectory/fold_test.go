package trajectory

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/provider"
)

func entry(seq int64, kind string, payload any) session.LogEntry {
	b, _ := json.Marshal(payload)
	return session.LogEntry{Seq: seq, Ts: 1700000000 + seq, Kind: kind, Payload: b}
}

func TestFoldEmpty(t *testing.T) {
	tl := FoldTrajectory(nil)
	if !tl.Ok || len(tl.Turns) != 0 || len(tl.BetweenTurns) != 0 {
		t.Fatalf("empty fold wrong: %+v", tl)
	}
}

// 回归：Turns / Turn.Records / BetweenTurns 必须序列化成 [] 而不是 null——
// Go 的 nil 切片序列化为 JSON null，前端 turns.length / records.filter
// 按数组消费会整页崩（ErrorBoundary 接管）。
func TestFoldEmptySlicesMarshalAsArrays(t *testing.T) {
	cases := map[string]any{
		"fold": FoldTrajectory(nil),
		// turn_started 后立刻 turn_done：产生一个 Records 为空的轮
		"foldEmptyTurn": FoldTrajectory([]session.LogEntry{
			entry(1, "turn_started", map[string]any{}),
			entry(2, "turn_done", map[string]any{}),
		}),
		"binding":  EmptyTrajectory(),
		"agentnet": EmptyAgentNetwork(),
	}
	for name, v := range cases {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		if strings.Contains(string(b), "null") {
			t.Fatalf("%s: serialized JSON contains null: %s", name, b)
		}
	}
}

func TestFoldOneTurnRecords(t *testing.T) {
	sys := strings.Repeat("s", 200)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "调研一下"}),
		entry(3, "request_header", map[string]any{"system": sys, "tools": []any{map[string]any{"name": "grep", "schema": `{"name":"grep"}`}}}),
		entry(4, "reasoning", map[string]any{"text": "先搜索"}),
		entry(5, "tool_dispatch", map[string]any{"id": "t1", "name": "grep", "args": `{"pattern":"x"}`, "partial": false}),
		entry(6, "tool_result", map[string]any{"id": "t1", "name": "grep", "output": "匹配行", "truncated": false}),
		entry(7, "message", map[string]any{"text": "结论如下", "reasoning": ""}),
		entry(8, "usage", map[string]any{"promptTokens": 1200, "completionTokens": 40, "cacheHitTokens": 1000, "cacheMissTokens": 200}),
		entry(9, "turn_done", map[string]any{"err": ""}),
	}
	tl := FoldTrajectory(entries)
	if len(tl.Turns) != 1 {
		t.Fatalf("turns = %d, want 1", len(tl.Turns))
	}
	turn := tl.Turns[0]
	if turn.End == nil || turn.End.Err != "" {
		t.Fatalf("turn end wrong: %+v", turn.End)
	}
	kinds := make([]string, 0, len(turn.Records))
	for _, r := range turn.Records {
		kinds = append(kinds, r.Kind)
	}
	want := []string{"user", "header", "assistant", "tool"}
	if strings.Join(kinds, ",") != strings.Join(want, ",") {
		t.Fatalf("record kinds = %v, want %v", kinds, want)
	}
	header := turn.Records[1].Header
	if header == nil || header.Change != "initial" || header.ToolCount != 1 || header.Tokens == 0 {
		t.Fatalf("header record wrong: %+v", turn.Records[1])
	}
	tool := turn.Records[3].Tool
	if tool == nil || tool.Name != "grep" || tool.Args != `{"pattern":"x"}` || tool.Output != "匹配行" || tool.Status != "ok" {
		t.Fatalf("tool record wrong: %+v", turn.Records[3])
	}
	if turn.Records[3].DurationMs <= 0 {
		t.Fatalf("tool duration missing: %+v", turn.Records[3])
	}
	asst := turn.Records[2].Assistant
	if asst == nil || asst.Reasoning != "先搜索" || asst.Text != "结论如下" || asst.Usage == nil || asst.Usage.PromptTokens != 1200 {
		t.Fatalf("assistant record wrong: %+v", turn.Records[2])
	}
}

// 迁移/投影产物（ToLogEntries：assistant_message 内嵌工具调用 + 回合边界）
// 折叠出的轨迹与运行期事件流同构：user → assistant → tool（含结果）。
func TestFoldLegacyProjectedEntries(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "读一下报价单"},
		{Role: provider.RoleAssistant, Content: "好的", ToolCalls: []provider.ToolCall{{ID: "c1", Name: "read_file", Arguments: `{"path":"x"}`}}},
		{Role: provider.RoleTool, Content: "表格内容", ToolCallID: "c1", Name: "read_file"},
	}
	entries := session.ToLogEntries(msgs)
	tl := FoldTrajectory(entries)
	if len(tl.Turns) != 1 {
		t.Fatalf("turns = %d, want 1（回合边界生效）", len(tl.Turns))
	}
	kinds := make([]string, 0, len(tl.Turns[0].Records))
	for _, r := range tl.Turns[0].Records {
		kinds = append(kinds, r.Kind)
	}
	want := []string{"user", "header", "assistant", "tool"}
	if strings.Join(kinds, ",") != strings.Join(want, ",") {
		t.Fatalf("record kinds = %v, want %v", kinds, want)
	}
	header := tl.Turns[0].Records[1].Header
	if header == nil || header.ToolCount != 1 || header.System == "" {
		t.Fatalf("synthesized header record wrong: %+v", tl.Turns[0].Records[1])
	}
	asst := tl.Turns[0].Records[2].Assistant
	if asst == nil || asst.Text != "好的" {
		t.Fatalf("assistant record wrong: %+v", tl.Turns[0].Records[1])
	}
	tool := tl.Turns[0].Records[3].Tool
	if tool == nil || tool.Name != "read_file" || tool.Args != `{"path":"x"}` || tool.Output != "表格内容" || tool.Status != "ok" {
		t.Fatalf("tool record wrong: %+v", tl.Turns[0].Records[2])
	}
}

func TestFoldHeaderChange(t *testing.T) {
	sys1 := strings.Repeat("a", 200)
	sys2 := strings.Repeat("b", 200)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "hi"}),
		entry(3, "request_header", map[string]any{"system": sys1, "tools": []any{}}),
		entry(4, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1}),
		entry(5, "request_header", map[string]any{"system": sys2, "tools": []any{map[string]any{"name": "bash", "schema": "{}"}}}),
		entry(6, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1}),
	}
	tl := FoldTrajectory(entries)
	records := tl.Turns[0].Records
	if len(records) != 5 {
		t.Fatalf("records = %d, want 5", len(records))
	}
	if records[1].Header.Change != "initial" {
		t.Fatalf("first change = %q", records[1].Header.Change)
	}
	if records[3].Header.Change != "system-and-tools" {
		t.Fatalf("second change = %q, want system-and-tools", records[3].Header.Change)
	}
}

func TestFoldNestedToolParentID(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "run"}),
		entry(3, "tool_dispatch", map[string]any{"id": "task1", "name": "task", "args": "{}", "partial": false}),
		entry(4, "tool_dispatch", map[string]any{"id": "sub1", "name": "bash", "args": "echo hi", "partial": false, "parentId": "task1"}),
		entry(5, "tool_result", map[string]any{"id": "sub1", "name": "bash", "output": "hi"}),
		entry(6, "tool_result", map[string]any{"id": "task1", "name": "task", "output": "done"}),
		entry(7, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1}),
	}
	tl := FoldTrajectory(entries)
	var tools []Record
	for _, r := range tl.Turns[0].Records[1:] {
		if r.Kind == "tool" {
			tools = append(tools, r)
		}
	}
	if len(tools) != 2 {
		t.Fatalf("tool records = %d, want 2", len(tools))
	}
	if tools[1].Tool.ParentID != "task1" {
		t.Fatalf("sub tool parentId = %q, want task1", tools[1].Tool.ParentID)
	}
}

func TestFoldBetweenTurnsCompaction(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "x"}),
		entry(3, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1}),
		entry(4, "turn_done", map[string]any{"err": ""}),
		entry(5, "compaction_done", map[string]any{"trigger": "manual", "summary": "旧内容压缩"}),
		entry(6, "turn_started", map[string]any{}),
		entry(7, "user_message", map[string]any{"content": "y"}),
		entry(8, "compaction_done", map[string]any{"trigger": "ratio", "summary": "轮内压缩"}),
		entry(9, "usage", map[string]any{"promptTokens": 5, "completionTokens": 1}),
	}
	tl := FoldTrajectory(entries)
	if len(tl.Turns) != 2 {
		t.Fatalf("turns = %d, want 2", len(tl.Turns))
	}
	if len(tl.BetweenTurns) != 1 || tl.BetweenTurns[0].Kind != "compact" || tl.BetweenTurns[0].Compact.Trigger != "manual" {
		t.Fatalf("between-turns compaction wrong: %+v", tl.BetweenTurns)
	}
	found := false
	for _, r := range tl.Turns[1].Records {
		if r.Kind == "compact" && r.Compact != nil && r.Compact.Trigger == "ratio" {
			found = true
		}
	}
	if !found {
		t.Fatalf("in-turn compaction record missing: %+v", tl.Turns[1].Records)
	}
}

func TestFoldErrorToolAndRunning(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "run"}),
		entry(3, "tool_dispatch", map[string]any{"id": "t1", "name": "bash", "args": "rm -rf /", "partial": false}),
		entry(4, "tool_result", map[string]any{"id": "t1", "name": "bash", "output": "", "err": "denied by policy"}),
		entry(5, "tool_dispatch", map[string]any{"id": "t2", "name": "sleep", "args": "100", "partial": false}),
		entry(6, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1}),
	}
	tl := FoldTrajectory(entries)
	records := tl.Turns[0].Records
	if records[1].Tool.Status != "error" || records[1].Tool.Err != "denied by policy" {
		t.Fatalf("error call wrong: %+v", records[1])
	}
	if records[2].Tool.Status != "running" {
		t.Fatalf("unfinished call should stay running: %+v", records[2])
	}
}

func TestFoldPreviewCaps(t *testing.T) {
	long := strings.Repeat("a", 5000)
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "x"}),
		entry(3, "tool_result", map[string]any{"id": "t1", "name": "bash", "output": long}),
		entry(4, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1}),
	}
	tl := FoldTrajectory(entries)
	out := tl.Turns[0].Records[1].Tool.Output
	if r := len([]rune(out)); r > maxOutPreview+1 {
		t.Fatalf("output not capped: runes=%d", r)
	}
}

func TestFoldAskAndApprovalRecords(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "开始"}),
		entry(3, "ask_request", map[string]any{
			"id": "a1",
			"questions": []any{
				map[string]any{"id": "q1", "header": "并行改动协调", "prompt": "如何协调并行改动？", "options": []any{}},
			},
		}),
		entry(4, "approval_request", map[string]any{"id": "p1", "tool": "cost_save", "subject": "写入成本库：钢材"}),
		entry(5, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1}),
	}
	tl := FoldTrajectory(entries)
	records := tl.Turns[0].Records
	if len(records) != 4 {
		t.Fatalf("records = %d, want 4", len(records))
	}
	if records[1].Kind != "ask" || records[1].Ask == nil || records[1].Ask.Question != "如何协调并行改动？" {
		t.Fatalf("ask record wrong: %+v", records[1])
	}
	if records[2].Kind != "approval" || records[2].Approval == nil || records[2].Approval.Tool != "cost_save" {
		t.Fatalf("approval record wrong: %+v", records[2])
	}
}

// TestFoldSubagentMessage v4.26 对话流式重造：subagent_message（子代理完成
// 回投）折叠为 kind=subagent 记录（text 展示级截断 + ref + 父 task 调用 ID）。
// 旧日志无此 kind 不受影响；未知 kind 仍被跳过（既有行为不变）。
func TestFoldSubagentMessage(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "调研 Foo"}),
		entry(3, "tool_dispatch", map[string]any{"id": "t1", "name": "task", "args": `{"prompt":"..."}`}),
		entry(4, "subagent_message", map[string]any{"text": "找到 3 处调用", "ref": "sa_abc", "parentId": "t1"}),
		entry(5, "tool_result", map[string]any{"id": "t1", "name": "task", "output": "<task-result>…"}),
		entry(6, "turn_done", map[string]any{"err": ""}),
	}
	tl := FoldTrajectory(entries)
	records := tl.Turns[0].Records
	var sub *Record
	for i := range records {
		if records[i].Kind == "subagent" {
			sub = &records[i]
		}
	}
	if sub == nil || sub.Subagent == nil {
		t.Fatalf("subagent 记录缺失: %+v", records)
	}
	if sub.Subagent.Text != "找到 3 处调用" || sub.Subagent.Ref != "sa_abc" || sub.Subagent.ParentID != "t1" {
		t.Fatalf("subagent 记录字段错误: %+v", sub.Subagent)
	}
}
