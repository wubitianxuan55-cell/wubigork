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

// TestFoldAttentionResolvedPairs 锁定「注意力事务」在派生轨迹里的成对性：
// approval_resolved / ask_resolved 按 id 回填请求记录（outcome/等待时长/
// decision），而不是新增一条孤立记录；找不到请求的孤立结果事件被忽略
// （日志原文仍是真相，折叠只是派生视图）。
func TestFoldAttentionResolvedPairs(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "继续"}),
		entry(3, "approval_request", map[string]any{
			"id": "p1", "tool": "cost_save", "subject": "写入成本库：钢材",
			"source": "permission_request", "reason": "跨库写入", "request": true,
		}),
		entry(4, "approval_resolved", map[string]any{
			"id": "p1", "tool": "cost_save", "subject": "写入成本库：钢材",
			"source": "permission_request", "decision": "allow_once",
			"outcome": "allowed", "latency_ms": 1500,
		}),
		entry(5, "ask_request", map[string]any{
			"id": "q1",
			"questions": []any{
				map[string]any{"id": "q1", "prompt": "选哪个方案？", "options": []any{}},
			},
		}),
		entry(6, "ask_resolved", map[string]any{"id": "q1", "outcome": "answered", "answered": 1, "latency_ms": 800}),
		// 孤立的解决事件（请求在别的分片/被重写）：不得新增记录。
		entry(7, "approval_resolved", map[string]any{"id": "ghost", "outcome": "denied"}),
	}
	tl := FoldTrajectory(entries)
	records := tl.Turns[0].Records
	if len(records) != 3 {
		t.Fatalf("records = %d, want 3（孤立的解决事件不建记录）: %+v", len(records), records)
	}
	if records[1].Kind != "approval" || records[1].Approval == nil {
		t.Fatalf("approval record missing: %+v", records[1])
	}
	ap := records[1].Approval
	if ap.Source != "permission_request" || ap.Reason != "跨库写入" {
		t.Errorf("approval 判别字段未落轨迹: %+v", ap)
	}
	if ap.Outcome != "allowed" || ap.Decision != "allow_once" || ap.LatencyMS != 1500 {
		t.Errorf("approval 结果未回填: %+v", ap)
	}
	if records[2].Kind != "ask" || records[2].Ask == nil {
		t.Fatalf("ask record missing: %+v", records[2])
	}
	ask := records[2].Ask
	if ask.Outcome != "answered" || ask.Answered != 1 || ask.LatencyMS != 800 {
		t.Errorf("ask 结果未回填: %+v", ask)
	}
}

// 自动放行账本（approval_auto）落轨迹：它不是注意力事务（没有 id/等待），
// 直接成为一条独立记录，Decided 说明「没弹卡时谁替人放的行」。
func TestFoldApprovalAuto(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{"trigger": "user"}),
		entry(2, "approval_auto", map[string]any{
			"tool": "bash", "subject": "go test ./...",
			"source": "gate", "decided": "yolo",
		}),
	}
	tl := FoldTrajectory(entries)
	recs := tl.Turns[0].Records
	if len(recs) != 1 || recs[0].Kind != "approval_auto" || recs[0].Approval == nil {
		t.Fatalf("records = %+v, want 单条 approval_auto", recs)
	}
	ap := recs[0].Approval
	if ap.Decided != "yolo" || ap.Source != "gate" {
		t.Errorf("decided/source = %q/%q, want yolo/gate", ap.Decided, ap.Source)
	}
	if ap.Tool != "bash" || ap.Subject != "go test ./..." {
		t.Errorf("tool/subject = %q/%q", ap.Tool, ap.Subject)
	}
	if ap.ID != "" || ap.Outcome != "" {
		t.Errorf("自动放行不该有 id/outcome: %+v", ap)
	}
}

// T4：回合发起者判别字段（turn_started.trigger）落轨迹。唤醒轮此前只能靠
// 输入文本的 "[system]" 前缀反推；类型化之后由物理字段判定。旧日志缺该
// 字段时保持空（读端语义=user，omitempty 让旧轨迹输出逐字节不变）。
func TestFoldTurnTrigger(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{"trigger": "wake"}),
		entry(2, "user_message", map[string]any{"content": "[system] 后台任务完成"}),
		entry(3, "turn_done", map[string]any{}),
		entry(4, "turn_started", map[string]any{}),
		entry(5, "turn_done", map[string]any{}),
	}
	tl := FoldTrajectory(entries)
	if len(tl.Turns) != 2 {
		t.Fatalf("turns = %d, want 2", len(tl.Turns))
	}
	if tl.Turns[0].Trigger != "wake" {
		t.Errorf("唤醒轮 trigger = %q, want wake", tl.Turns[0].Trigger)
	}
	if tl.Turns[1].Trigger != "" {
		t.Errorf("旧日志轮 trigger = %q, want 空（读端按 user 兜底）", tl.Turns[1].Trigger)
	}
}

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

// TestFoldTurnDuration v4.31 轮级耗时（收 v4.26 欠账：WorkHeader 历史轮无
// 耗时数据源）：turn_done.Ts − turn_started.Ts（秒）×1000 → ms（对齐
// Record.DurationMs 换算先例）。turn_done 缺失（悬挂轮）与时钟异常
// （Ts ≤ StartedAt）DurationMs 保持 0（omitempty 序列化省略）。
func TestFoldTurnDuration(t *testing.T) {
	entryTs := func(seq, ts int64, kind string, payload any) session.LogEntry {
		b, _ := json.Marshal(payload)
		return session.LogEntry{Seq: seq, Ts: ts, Kind: kind, Payload: b}
	}
	// turn_started(ts=100) → 若干记录 → turn_done(ts=130)：耗时 30s = 30000ms
	tl := FoldTrajectory([]session.LogEntry{
		entryTs(1, 100, "turn_started", map[string]any{}),
		entryTs(2, 105, "user_message", map[string]any{"content": "hi"}),
		entryTs(3, 120, "tool_dispatch", map[string]any{"id": "t1", "name": "bash", "args": "echo hi"}),
		entryTs(4, 130, "turn_done", map[string]any{"err": ""}),
	})
	if len(tl.Turns) != 1 {
		t.Fatalf("turns = %d, want 1", len(tl.Turns))
	}
	if tl.Turns[0].DurationMs != 30000 {
		t.Fatalf("turn durationMs = %d, want 30000", tl.Turns[0].DurationMs)
	}
	// 悬挂轮（无 turn_done）：DurationMs 保持 0
	tl = FoldTrajectory([]session.LogEntry{
		entryTs(1, 100, "turn_started", map[string]any{}),
		entryTs(2, 105, "user_message", map[string]any{"content": "hi"}),
	})
	if len(tl.Turns) != 1 {
		t.Fatalf("hanging turns = %d, want 1", len(tl.Turns))
	}
	if tl.Turns[0].DurationMs != 0 {
		t.Fatalf("hanging turn durationMs = %d, want 0", tl.Turns[0].DurationMs)
	}
	// turn_done 不晚于 turn_started（时钟异常）：不产生负数耗时，保持 0
	tl = FoldTrajectory([]session.LogEntry{
		entryTs(1, 200, "turn_started", map[string]any{}),
		entryTs(2, 150, "turn_done", map[string]any{"err": ""}),
	})
	if tl.Turns[0].DurationMs != 0 {
		t.Fatalf("backdated turn durationMs = %d, want 0", tl.Turns[0].DurationMs)
	}
}

// TestFoldTurnDurationOmitEmpty 轮级耗时是 omitempty 可选字段：completed 轮
// 带 durationMs，悬挂轮不带——既有消费方（字段级断言）与 JSON 形状均不受
// 影响（golden 逐字节不变；无轮级耗时的旧断言序列化形状保持原样）。
func TestFoldTurnDurationOmitEmpty(t *testing.T) {
	entryTs := func(seq, ts int64, kind string, payload any) session.LogEntry {
		b, _ := json.Marshal(payload)
		return session.LogEntry{Seq: seq, Ts: ts, Kind: kind, Payload: b}
	}
	completed := FoldTrajectory([]session.LogEntry{
		entryTs(1, 100, "turn_started", map[string]any{}),
		entryTs(2, 110, "turn_done", map[string]any{"err": ""}),
	})
	b, err := json.Marshal(completed)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"durationMs":10000`) {
		t.Fatalf("completed turn should carry durationMs: %s", b)
	}
	hanging := FoldTrajectory([]session.LogEntry{
		entryTs(1, 100, "turn_started", map[string]any{}),
		entryTs(2, 110, "user_message", map[string]any{"content": "hi"}),
	})
	b, err = json.Marshal(hanging)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "durationMs") {
		t.Fatalf("hanging turn must omit durationMs: %s", b)
	}
}
