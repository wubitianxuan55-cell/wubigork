package contextview

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// ─── 刀1（v4.542）扩展面黄金测试：assistant 三拆 / skill 分类 / 命令执行 /
// provider 切换事件 / 浏览器视图上限 ───

func msgPayload(text, reasoning string, toolCalls ...map[string]string) map[string]any {
	tcs := make([]any, 0, len(toolCalls))
	for _, tc := range toolCalls {
		tcs = append(tcs, tc)
	}
	return map[string]any{"text": text, "reasoning": reasoning, "tool_calls": tcs}
}

// TestFoldAssistantSplit 校验 assistant 三拆：message 携带 text+reasoning+
// 内嵌 tool_calls（迁移路径）时，节点与 stats 的三子计数、分类合计一致。
func TestFoldAssistantSplit(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload("sys")),
		entry(3, "message", msgPayload("answer text", "thinking hard",
			map[string]string{"name": "read_file", "args": `{"path":"a.go"}`})),
		entry(4, "usage", map[string]any{"promptTokens": 500, "completionTokens": 100}),
		entry(5, "turn_done", map[string]any{}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	ans := int64(len("answer text") / 4)
	reas := int64(len("thinking hard") / 4)
	call := int64((len("read_file") + 1 + len(`{"path":"a.go"}`)) / 4)
	total := ans + reas + call
	if tl.Stats.AssistantAnswerTokens != ans {
		t.Fatalf("AssistantAnswerTokens = %d, want %d", tl.Stats.AssistantAnswerTokens, ans)
	}
	if tl.Stats.AssistantReasoning != reas {
		t.Fatalf("AssistantReasoning = %d, want %d", tl.Stats.AssistantReasoning, reas)
	}
	if tl.Stats.AssistantCallTokens != call {
		t.Fatalf("AssistantCallTokens = %d, want %d", tl.Stats.AssistantCallTokens, call)
	}
	if tl.Current.Assistant != total {
		t.Fatalf("Current.Assistant = %d, want %d（三拆合计）", tl.Current.Assistant, total)
	}
	var node *SurfaceNode
	for i := range tl.Nodes {
		if tl.Nodes[i].Cat == catAssistant {
			node = &tl.Nodes[i]
		}
	}
	if node == nil {
		t.Fatal("assistant node missing")
	}
	if node.Tokens != total || node.AnswerTokens != ans || node.Reasoning != reas || node.CallTokens != call {
		t.Fatalf("assistant node sub-counts mismatch: %+v", node)
	}
}

// TestFoldCallTokensFromDispatch 校验运行期参数重放账：message 不内嵌
// tool_calls（EventLogSink 只落 text/reasoning），参数账在 tool_dispatch
// 挂到最后一条 assistant 节点，压缩随节点回收。
func TestFoldCallTokensFromDispatch(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload("sys")),
		entry(3, "message", msgPayload("running", "")),
		entry(4, "tool_dispatch", map[string]any{"id": "c1", "name": "bash", "args": `{"command":"ls"}`}),
		entry(5, "tool_result", map[string]any{"id": "c1", "name": "bash", "output": "files"}),
		entry(6, "usage", map[string]any{"promptTokens": 500, "completionTokens": 50}),
		entry(7, "turn_done", map[string]any{}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if tl.Stats.AssistantCallTokens == 0 {
		t.Fatal("dispatch call tokens should accrue into AssistantCallTokens")
	}
	var node *SurfaceNode
	for i := range tl.Nodes {
		if tl.Nodes[i].Cat == catAssistant {
			node = &tl.Nodes[i]
		}
	}
	if node == nil || node.CallTokens == 0 {
		t.Fatalf("call tokens should attach to last assistant node, got %+v", node)
	}
	if node.Tokens != node.AnswerTokens+node.Reasoning+node.CallTokens {
		t.Fatalf("node.Tokens should be sum of three sub-counts: %+v", node)
	}
	if tl.Current.Assistant != node.Tokens {
		t.Fatalf("Current.Assistant = %d, node.Tokens = %d（应一致）", tl.Current.Assistant, node.Tokens)
	}
}

// TestFoldSkillCategory 校验 run_skill 结果单列 skill 分类：节点带技能名、
// stats.Skills 计数、skillTok 计账、压缩回收。
func TestFoldSkillCategory(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload("sys")),
		entry(3, "message", msgPayload("load skill", "")),
		entry(4, "tool_dispatch", map[string]any{"id": "s1", "name": "run_skill", "args": `{"name":"pdf-toolkit"}`}),
		entry(5, "tool_result", map[string]any{"id": "s1", "name": "run_skill", "output": "skill body content here"}),
		entry(6, "usage", map[string]any{"promptTokens": 400, "completionTokens": 30}),
		entry(7, "compaction_done", map[string]any{"trigger": "ratio"}),
		entry(8, "usage", map[string]any{"promptTokens": 100, "completionTokens": 10}),
		entry(9, "turn_done", map[string]any{}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if tl.Stats.Skills != 1 {
		t.Fatalf("Stats.Skills = %d, want 1", tl.Stats.Skills)
	}
	var skillNode *SurfaceNode
	for i := range tl.Archive {
		if tl.Archive[i].Cat == catSkill {
			skillNode = &tl.Archive[i]
		}
	}
	if skillNode == nil {
		t.Fatal("skill node should exist in archive after compaction")
	}
	if skillNode.Name != "pdf-toolkit" || skillNode.Tool != "run_skill" {
		t.Fatalf("skill node Name/Tool mismatch: %+v", skillNode)
	}
	if tl.Current.Skill != 0 {
		t.Fatalf("skill tokens should be reclaimed by compaction, got %d", tl.Current.Skill)
	}
	if tl.Stats.Compacts != 1 {
		t.Fatalf("Compacts = %d, want 1", tl.Stats.Compacts)
	}
}

// TestFoldCommands 校验命令执行折叠：completed/failed/killed 三态、bg 标记、
// JSON 信封退出码、OutChars、CommandStats 汇总。
func TestFoldCommands(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload("sys")),
		// 完成 + JSON 信封退出码
		entry(3, "tool_dispatch", map[string]any{"id": "b1", "name": "bash", "args": `{"command":"ls -la"}`}),
		entry(4, "tool_result", map[string]any{"id": "b1", "name": "bash", "output": `{"ok":true,"exit_code":2,"stdout":"f.go","stderr":""}`}),
		// 失败（err 返回）
		entry(5, "tool_dispatch", map[string]any{"id": "b2", "name": "bash", "args": `{"command":"exit 1"}`}),
		entry(6, "tool_result", map[string]any{"id": "b2", "name": "bash", "output": "", "err": "command failed"}),
		// 后台
		entry(7, "tool_dispatch", map[string]any{"id": "b3", "name": "bash", "args": `{"command":"npm run dev","run_in_background":true}`}),
		entry(8, "tool_result", map[string]any{"id": "b3", "name": "bash", "output": "job bg_1 started"}),
		// 非命令工具不进时间线
		entry(9, "tool_dispatch", map[string]any{"id": "r1", "name": "read_file", "args": `{"path":"a.go"}`}),
		entry(10, "tool_result", map[string]any{"id": "r1", "name": "read_file", "output": "package main"}),
		entry(11, "usage", map[string]any{"promptTokens": 300, "completionTokens": 40}),
		// killed：派发后无结果直接收回合
		entry(12, "tool_dispatch", map[string]any{"id": "b4", "name": "bash", "args": `{"command":"sleep 999"}`}),
		entry(13, "turn_done", map[string]any{}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if len(tl.Commands) != 4 {
		t.Fatalf("Commands = %d, want 4", len(tl.Commands))
	}
	c0, c1, c2, c3 := tl.Commands[0], tl.Commands[1], tl.Commands[2], tl.Commands[3]
	if c0.Status != CommandCompleted || c0.Exit != 2 || c0.OutChars == 0 {
		t.Fatalf("command 0 should be completed exit=2 with outChars: %+v", c0)
	}
	if c1.Status != CommandFailed || !c1.Err {
		t.Fatalf("command 1 should be failed: %+v", c1)
	}
	if c2.Status != CommandCompleted || !c2.Background {
		t.Fatalf("command 2 should be completed+background: %+v", c2)
	}
	if c3.Status != CommandKilled || c3.End == 0 {
		t.Fatalf("command 3 should be killed with End from turn_done: %+v", c3)
	}
	for _, c := range tl.Commands {
		if c.Tool != "bash" || !c.Derived {
			t.Fatalf("command should be bash+derived: %+v", c)
		}
	}
	st := tl.CommandStats
	if st == nil || st.Runs != 4 || st.Failed != 1 || st.Bg != 1 {
		t.Fatalf("CommandStats mismatch: %+v", st)
	}
	if st.Ms == 0 {
		t.Fatal("CommandStats.Ms should sum closed command durations")
	}
}

// TestFoldProviderSwitch 校验 provider 戳折叠出 switch 事件：同名不造事件、
// 换名一条事件带 From/To。
func TestFoldProviderSwitch(t *testing.T) {
	hdr := func(provider string) map[string]any {
		p := headerPayload("sys").(map[string]any)
		p["provider"] = provider
		return p
	}
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", hdr("deepseek")),
		entry(3, "usage", map[string]any{"promptTokens": 100}),
		entry(4, "request_header", hdr("deepseek")),
		entry(5, "usage", map[string]any{"promptTokens": 100}),
		entry(6, "request_header", hdr("strata")),
		entry(7, "usage", map[string]any{"promptTokens": 100}),
		entry(8, "turn_done", map[string]any{}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	switches := 0
	var ev ContextEvent
	for _, e := range tl.Events {
		if e.Kind == "switch" {
			switches++
			ev = e
		}
	}
	if switches != 1 {
		t.Fatalf("switch events = %d, want 1", switches)
	}
	if ev.From != "deepseek" || ev.To != "strata" {
		t.Fatalf("switch event From/To mismatch: %+v", ev)
	}
}

// TestFoldBrowserWindowTrim 校验视图上限：节点超出 maxNodes 丢最旧并标注
// NodesDropped/NodesFloor；差分快照走增量计数器不受视图截断影响（最后一条
// 请求的 Assistant 分类 token 仍含被截断节点的账）。
func TestFoldBrowserWindowTrim(t *testing.T) {
	var entries []session.LogEntry
	entries = append(entries, entry(1, "turn_started", map[string]any{}))
	entries = append(entries, entry(2, "request_header", headerPayload("sys")))
	seq := int64(3)
	// maxNodes+30 条 user 消息 → 节点超限
	for i := 0; i < maxNodes+30; i++ {
		entries = append(entries, entry(seq, "user_message", map[string]any{"content": fmt.Sprintf("msg %d", i)}))
		seq++
	}
	entries = append(entries, entry(seq, "usage", map[string]any{"promptTokens": 10000}))
	tl := FoldTimeline(entries, 1_000_000, 0)
	if len(tl.Nodes) != maxNodes {
		t.Fatalf("Nodes = %d, want %d", len(tl.Nodes), maxNodes)
	}
	if tl.NodesDropped != 31 { // 30 条超限 user 节点 + 1 条最旧 system header 节点
		t.Fatalf("NodesDropped = %d, want 31", tl.NodesDropped)
	}
	if tl.NodesFloor != tl.Nodes[0].Seq {
		t.Fatalf("NodesFloor = %d, want first surviving seq %d", tl.NodesFloor, tl.Nodes[0].Seq)
	}
	if tl.Current.User == 0 {
		t.Fatal("User token counter must survive view truncation")
	}
}

// TestFoldEventsCap 校验事件上限：超限丢最旧；stats 计数器是完整口径不丢账
// （inject/mode 两类事件内核尚不发射，这里用 prune 事件灌满上限）。
func TestFoldEventsCap(t *testing.T) {
	var entries []session.LogEntry
	entries = append(entries, entry(1, "turn_started", map[string]any{}))
	entries = append(entries, entry(2, "request_header", headerPayload("sys")))
	seq := int64(3)
	for i := 0; i < maxEvents+10; i++ {
		entries = append(entries, entry(seq, "tool_dispatch", map[string]any{"id": fmt.Sprint(i), "name": "read_file", "args": `{"path":"a.go"}`}))
		seq++
		entries = append(entries, entry(seq, "tool_result", map[string]any{"id": fmt.Sprint(i), "name": "read_file", "output": "x", "truncated": true}))
		seq++
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if len(tl.Events) != maxEvents {
		t.Fatalf("Events = %d, want %d", len(tl.Events), maxEvents)
	}
	if tl.Stats.Prunes != maxEvents+10 {
		t.Fatalf("Prunes = %d, want %d（计数不随视图截断丢账）", tl.Stats.Prunes, maxEvents+10)
	}
}

// TestFoldEmptyCommandsMarshalAsArrays 回归：Commands 切片恒非 nil。
func TestFoldEmptyCommandsMarshalAsArrays(t *testing.T) {
	b, _ := json.Marshal(FoldTimeline(nil, 0, 0))
	if !strings.Contains(string(b), `"commands":[]`) {
		t.Fatalf("commands should marshal as [], got %s", b)
	}
}

// TestFoldLegacyCommandsFromEmbeddedToolCalls（真实数据探针定罪的补账）：
// 迁移日志零 tool_dispatch——命令时间线此前恒空而 bash 结果在。legacy 内嵌
// tool_calls 即派发事实，按 id 开挂起命令、tool_result 既有配对照常收尾。
func TestFoldLegacyCommandsFromEmbeddedToolCalls(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload("sys")),
		entry(3, "message", msgPayload("跑一下构建", "",
			map[string]string{"id": "tc1", "name": "bash", "args": `{"command":"go build ./..."}`},
			map[string]string{"id": "tc2", "name": "bash", "args": `{"command":"npm run dev","run_in_background":true}`},
		)),
		entry(4, "tool_result", map[string]any{"id": "tc1", "name": "bash", "output": `{"ok":true,"exit_code":0,"stdout":"ok"}`}),
		entry(5, "tool_result", map[string]any{"id": "tc2", "name": "bash", "output": "job bg_1 started"}),
		entry(6, "turn_done", map[string]any{}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if len(tl.Commands) != 2 {
		t.Fatalf("Commands = %d, want 2（legacy 内嵌 tool_calls 推导）", len(tl.Commands))
	}
	c0, c1 := tl.Commands[0], tl.Commands[1]
	if c0.Status != CommandCompleted || c0.Exit != 0 || c0.ResultSeq != 4 {
		t.Fatalf("legacy command 0: %+v", c0)
	}
	if c1.Status != CommandCompleted || !c1.Background {
		t.Fatalf("legacy command 1 (bg): %+v", c1)
	}
	if c0.Derived != true || c0.Tool != "bash" {
		t.Fatalf("legacy command 应为 bash+derived: %+v", c0)
	}
	// 三拆账不受影响：内嵌参数仍走 applyAssistant 的 call 计数
	if tl.Stats.AssistantCallTokens == 0 {
		t.Fatal("AssistantCallTokens 应含内嵌参数账")
	}
}

// TestFoldRuntimeNoDoubleCommand（互斥性回归）：运行期 message 不内嵌
// tool_calls（EventLogSink 只落 text/reasoning），命令只由 tool_dispatch 开——
// 同一命令不得双记。
func TestFoldRuntimeNoDoubleCommand(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "request_header", headerPayload("sys")),
		// 运行期形态：message 只有 text（无 tool_calls）
		entry(3, "message", msgPayload("running", "")),
		entry(4, "tool_dispatch", map[string]any{"id": "c1", "name": "bash", "args": `{"command":"ls"}`}),
		entry(5, "tool_result", map[string]any{"id": "c1", "name": "bash", "output": "files"}),
		entry(6, "turn_done", map[string]any{}),
	}
	tl := FoldTimeline(entries, 1_000_000, 0)
	if len(tl.Commands) != 1 {
		t.Fatalf("Commands = %d, want 1（运行期单记）", len(tl.Commands))
	}
}
