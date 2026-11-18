package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent/session"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/contextview"
	"github.com/gaea/gaea/internal/gaea/trajectory"
)

// enrichAgentNetwork：树上已有节点（事件日志折叠产物）按任务前缀匹配 run 富化。
func TestEnrichAgentNetwork_MatchesExistingNode(t *testing.T) {
	net := trajectory.AgentNetwork{Ok: true, Root: trajectory.AgentNode{
		ID: "root", Kind: "root",
		Children: []trajectory.AgentNode{{ID: "call_1", Kind: "subagent", Task: "调研土壤修复", Status: "running"}},
	}}
	runs := SubagentRunsView{Available: true, Runs: []SubagentRunView{{
		Ref: "sa_1", Kind: "subagent", Status: "completed", Model: "m1", Task: "调研土壤修复并输出报告",
	}}}
	enrichAgentNetwork(&net, runs)
	n := net.Root.Children[0]
	if n.Status != "completed" || n.Model != "m1" {
		t.Fatalf("existing node not enriched: %+v", n)
	}
	if len(net.Root.Children) != 1 {
		t.Fatalf("matched run must not be duplicated, got %d children", len(net.Root.Children))
	}
}

// enrichAgentNetwork：零工具调用的子代理在事件日志里没有子记录、不成树节点，
// 必须按 run 补挂合成节点（id=sa_ ref），否则任务管理树永远缺行（真机回归：
// 纯调研子代理整批从任务管理消失）。
func TestEnrichAgentNetwork_AppendsZeroToolRuns(t *testing.T) {
	net := trajectory.AgentNetwork{Ok: true, Root: trajectory.AgentNode{ID: "root", Kind: "root"}}
	created := time.Date(2026, 9, 4, 14, 1, 45, 0, time.Local)
	runs := SubagentRunsView{Available: true, Runs: []SubagentRunView{
		{Ref: "sa_a", Kind: "subagent", Status: "completed", Model: "grok", Task: "纯调研 Devin", ToolCalls: 0, CreatedAt: created, UpdatedAt: created.Add(time.Minute)},
		{Ref: "sa_b", Kind: "subagent", Status: "failed", Task: "纯调研 Codex", CreatedAt: created, UpdatedAt: created},
	}}
	enrichAgentNetwork(&net, runs)
	if len(net.Root.Children) != 2 {
		t.Fatalf("zero-tool runs must be appended as synthetic nodes, got %d", len(net.Root.Children))
	}
	a := net.Root.Children[0]
	if a.ID != "sa_a" || a.Kind != "subagent" || a.Status != "completed" || a.Model != "grok" || a.Task != "纯调研 Devin" {
		t.Fatalf("synthetic node fields wrong: %+v", a)
	}
	if a.FirstTs != created.Unix() || a.LastTs != created.Add(time.Minute).Unix() {
		t.Fatalf("synthetic node timestamps wrong: first=%d last=%d", a.FirstTs, a.LastTs)
	}
	if b := net.Root.Children[1]; b.Status != "error" {
		t.Fatalf("failed run must map to error status, got %q", b.Status)
	}
}

// enrichAgentNetwork：model_tool 运行不进树（前端「本地模型工具」区块单独渲染，
// 合成进树会同名重复）；ref 直等已承载的 run 不重复补挂。
func TestEnrichAgentNetwork_SkipsModelToolAndRefMatched(t *testing.T) {
	net := trajectory.AgentNetwork{Ok: true, Root: trajectory.AgentNode{
		ID: "root", Kind: "root",
		Children: []trajectory.AgentNode{{ID: "sa_x", Kind: "subagent", Task: "", Status: "running"}},
	}}
	runs := SubagentRunsView{Available: true, Runs: []SubagentRunView{
		{Ref: "sa_x", Kind: "subagent", Status: "completed", Task: "有树节点的运行"},
		{Ref: "mt_1", Kind: "model_tool", Status: "completed", Task: "vision 调用"},
	}}
	enrichAgentNetwork(&net, runs)
	if len(net.Root.Children) != 1 {
		t.Fatalf("ref-matched + model_tool runs must not append nodes, got %d", len(net.Root.Children))
	}
	if n := net.Root.Children[0]; n.Status != "completed" || n.Task != "有树节点的运行" {
		t.Fatalf("ref-matched node not enriched: %+v", n)
	}
}

// TestGaeaTrajectoryAndNetworkExplicitSessionPath 真实接线：磁盘事件日志 +
// 显式会话路径（v4.181）。内核 ctrl=nil（缺省=空快照）时，显式路径仍按传入
// 会话读取——证明路径优先级与「UI 查看会话 ≠ 内核活跃会话」解耦。
func TestGaeaTrajectoryAndNetworkExplicitSessionPath(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()
	ga.cfg = &gaeaConfig.Config{Workspace: t.TempDir()}
	ga.ctrl = nil

	// 轨迹/网络绑定只用 gaeaCtrl 与会话日志路径，不依赖 App 配置。
	a := &App{core: &core{}}

	// 缺省（内核 nil）→ 空快照不报错
	if tr, err := a.GaeaTrajectory(); err != nil || len(tr.Turns) != 0 {
		t.Fatalf("缺省 Trajectory 应为空快照: turns=%d err=%v", len(tr.Turns), err)
	}
	if net, err := a.GaeaAgentNetwork(); err != nil || len(net.Root.Children) != 0 {
		t.Fatalf("缺省 AgentNetwork 应为空快照: children=%d err=%v", len(net.Root.Children), err)
	}

	// 磁盘事件日志：一轮 user + assistant（Kind 与运行期事件序一致）。
	// 路径按真实 UI 会话切换形态构造（<ws>/.gaea/sessions/work/<file>）——
	// v4.358 起显式会话路径须过 sessionDirForPath 校验（读侧收口）。
	ws := t.TempDir()
	p := filepath.Join(ws, ".gaea", "sessions", "work", "hist-session.jsonl")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	w, err := session.OpenLog(session.LogPathFor(p), "", "")
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	entries := []session.LogEntry{
		{Kind: "turn_started", Payload: mustRaw(t, map[string]string{})},
		{Kind: session.KindUserMessage, Payload: mustRaw(t, map[string]string{"content": "历史会话的提问"})},
		{Kind: session.KindAssistantMessage, Payload: mustRaw(t, map[string]any{"id": "a1", "text": "历史会话的答复"})},
	}
	for i, e := range entries {
		if _, err := w.AppendRaw(e.Kind, e.Payload); err != nil {
			t.Fatalf("append entry %d: %v", i, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// 显式路径 → 按该会话读取（非空）。
	tr, err := a.GaeaTrajectory(p)
	if err != nil {
		t.Fatalf("显式路径 Trajectory: %v", err)
	}
	if len(tr.Turns) == 0 {
		t.Fatal("显式路径 Trajectory 应折叠出轮次")
	}
	net, err := a.GaeaAgentNetwork(p)
	if err != nil {
		t.Fatalf("显式路径 AgentNetwork: %v", err)
	}
	if !net.Ok {
		t.Fatal("显式路径 AgentNetwork 应 ok=true")
	}
	// 空串参数 = 缺省语义（内核 nil → 空），不误读显式路径残留
	if tr2, err := a.GaeaTrajectory(""); err != nil || len(tr2.Turns) != 0 {
		t.Fatalf("空串参数应回落缺省（内核 nil → 空）: turns=%d err=%v", len(tr2.Turns), err)
	}
	_ = json.Marshal
}

// TestGaeaContextViewAndNodeDetailExplicitSessionPath 是 v4.181 欠账收刀：
// 时间线/节点详情此前恒读内核 ctrl，UI 切历史会话后不跟。显式路径优先，
// 内核 nil 时缺省仍空快照（ContextView）/报会话未就绪（NodeDetail）。
func TestGaeaContextViewAndNodeDetailExplicitSessionPath(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()
	oldCfg, oldCtrl := ga.cfg, ga.ctrl
	defer func() { ga.cfg, ga.ctrl = oldCfg, oldCtrl }()
	ga.cfg = &gaeaConfig.Config{Workspace: t.TempDir()}
	ga.ctrl = nil

	a := &App{core: &core{}}

	if tl, err := a.GaeaContextView(); err != nil || !tl.Ok {
		t.Fatalf("缺省 ContextView 应为空快照: ok=%v err=%v", tl.Ok, err)
	}
	// 内核 nil → 压缩水位块缺席（前端退回静态阈值，不伪造水位）。
	if tl, _ := a.GaeaContextView(); tl.Compact != nil {
		t.Fatal("内核 nil 时 Compact 水位块应为 nil")
	}
	if _, err := a.GaeaContextNodeDetail(1); err == nil {
		t.Fatal("内核 nil 时 NodeDetail 缺省应报会话未就绪")
	}

	ws := t.TempDir()
	p := filepath.Join(ws, ".gaea", "sessions", "work", "hist-session.jsonl")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	w, err := session.OpenLog(session.LogPathFor(p), "", "")
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	entries := []session.LogEntry{
		{Kind: "turn_started", Payload: mustRaw(t, map[string]string{})},
		{Kind: session.KindUserMessage, Payload: mustRaw(t, map[string]string{"content": "历史会话的提问"})},
		{Kind: session.KindRequestHeader, Payload: mustRaw(t, map[string]any{
			"system": "历史系统提示词",
			"tools":  []map[string]string{{"name": "read_file", "schema": `{"type":"object"}`}},
		})},
		{Kind: session.KindAssistantMessage, Payload: mustRaw(t, map[string]any{"id": "a1", "text": "历史会话的答复"})},
		{Kind: session.KindRequestHeader, Payload: mustRaw(t, map[string]any{
			"system": "历史系统提示词",
			"tools":  []map[string]string{{"name": "read_file", "schema": `{"type":"object"}`}},
		})},
		{Kind: "tool_dispatch", Payload: mustRaw(t, map[string]any{
			"id": "b1", "name": "bash", "args": `{"command":"echo one"}`,
		})},
		{Kind: session.KindToolResult, Payload: mustRaw(t, map[string]any{
			"id": "b1", "name": "bash", "output": "one",
		})},
		{Kind: "tool_dispatch", Payload: mustRaw(t, map[string]any{
			"id": "b2", "name": "bash", "args": `{"command":"echo two"}`,
		})},
		{Kind: session.KindToolResult, Payload: mustRaw(t, map[string]any{
			"id": "b2", "name": "bash", "output": "two",
		})},
	}
	seqs := make([]int64, 0, len(entries))
	for i, e := range entries {
		seq, err := w.AppendRaw(e.Kind, e.Payload)
		if err != nil {
			t.Fatalf("append entry %d: %v", i, err)
		}
		seqs = append(seqs, seq)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	tl, err := a.GaeaContextView(p)
	if err != nil {
		t.Fatalf("显式路径 ContextView: %v", err)
	}
	if !tl.Ok {
		t.Fatal("显式路径 ContextView 应 ok=true")
	}
	if tl.Stats.Turns == 0 && len(tl.Nodes) == 0 && len(tl.Requests) == 0 {
		t.Fatal("显式路径 ContextView 应折叠出内容")
	}
	if _, err := a.GaeaContextNodeDetail(99999, p); err == nil {
		t.Fatal("显式路径上不存在的 seq 应报错（证明读的是该会话而非内核）")
	}
	first, err := a.GaeaContextSurface(seqs[2], p)
	if err != nil || !first.Ok || len(first.Nodes) == 0 {
		t.Fatalf("历史 surface: ok=%v nodes=%d err=%v", first.Ok, len(first.Nodes), err)
	}
	firstHasAssistant := false
	for _, node := range first.Nodes {
		firstHasAssistant = firstHasAssistant || node.Cat == "assistant"
	}
	if firstHasAssistant {
		t.Fatal("首个 request_header 不得看到其后生成的 assistant 答复")
	}
	second, err := a.GaeaContextSurface(seqs[4], p)
	if err != nil || !second.Ok {
		t.Fatalf("第二个历史 surface: %+v err=%v", second, err)
	}
	secondHasAssistant := false
	for _, node := range second.Nodes {
		secondHasAssistant = secondHasAssistant || node.Cat == "assistant"
	}
	if !secondHasAssistant {
		t.Fatal("第二个 request_header 应看到上一请求生成的 assistant 答复")
	}
	dna, err := a.GaeaContextDna([]int64{seqs[2], seqs[4]}, p)
	if err != nil || len(dna) != 2 || !dna[0].Ok || !dna[1].Ok {
		t.Fatalf("批量 DNA: %+v err=%v", dna, err)
	}
	firstDnaHasAssistant := false
	for _, band := range dna[0].Bands {
		firstDnaHasAssistant = firstDnaHasAssistant || band.Cat == "assistant"
	}
	if firstDnaHasAssistant {
		t.Fatal("首个 DNA 样本不得看到其后的 assistant 答复")
	}
	secondDnaHasAssistant := false
	for _, band := range dna[1].Bands {
		secondDnaHasAssistant = secondDnaHasAssistant || band.Cat == "assistant"
	}
	if !secondDnaHasAssistant {
		t.Fatal("第二个 DNA 样本应包含上一请求的 assistant 答复")
	}
	page, err := a.GaeaContextCommands(0, 1, p)
	if err != nil || page.Total != 2 || page.AtStart || len(page.Commands) != 1 {
		t.Fatalf("commands page: %+v err=%v", page, err)
	}
	older, err := a.GaeaContextCommands(page.Commands[0].Seq, 1, p)
	if err != nil || !older.AtStart || len(older.Commands) != 1 || older.Commands[0].Seq >= page.Commands[0].Seq {
		t.Fatalf("older commands page: %+v err=%v", older, err)
	}
	if _, err := a.GaeaContextSurface(99999, p); err == nil {
		t.Fatal("不存在 seq 的历史 surface 应报错，不得回退 live")
	}
	if _, err := a.GaeaContextSurface(seqs[2]); err == nil {
		t.Fatal("内核 nil 时历史 surface 缺省应报会话未就绪")
	}
	if _, err := a.GaeaContextDna([]int64{seqs[2]}); err == nil {
		t.Fatal("内核 nil 时批量 DNA 缺省应报会话未就绪")
	}
	if _, err := a.GaeaContextCommands(0, 10); err == nil {
		t.Fatal("内核 nil 时命令分页缺省应报会话未就绪")
	}
	if tl2, err := a.GaeaContextView(""); err != nil || tl2.Stats.Turns != 0 {
		t.Fatalf("空串参数应回落缺省（内核 nil → 空）: turns=%d err=%v", tl2.Stats.Turns, err)
	}
}

func TestResolveNodeAttachmentsConfinesAndStatsFiles(t *testing.T) {
	ws := t.TempDir()
	uploadDir := filepath.Join(ws, ".gaea", "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	inside := filepath.Join(uploadDir, "report-a.xlsx")
	if err := os.WriteFile(inside, []byte("xlsx"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "secret.pdf")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatalf("WriteFile outside: %v", err)
	}
	d := contextview.NodeDetail{
		AttachmentRefs: []string{
			".gaea/uploads/report-a.xlsx",
			outside,
		},
	}
	resolveNodeAttachments(&d, ws)
	if len(d.Attachments) != 2 {
		t.Fatalf("attachments = %+v", d.Attachments)
	}
	if !d.Attachments[0].Exists || d.Attachments[0].Size != 4 || d.Attachments[0].Name != "report-a.xlsx" {
		t.Fatalf("inside attachment = %+v", d.Attachments[0])
	}
	if d.Attachments[0].MimeType == "" {
		t.Fatal("inside attachment MIME should be resolved")
	}
	if d.Attachments[1].Exists || d.Attachments[1].Size != 0 {
		t.Fatalf("outside attachment must stay inaccessible: %+v", d.Attachments[1])
	}
}

// buildSessionNetwork：runs 携带血缘盖章时以会话为骨架建树——嵌套派生按
// parentRef 挂载、统计按 spawnCallId 精确联结（不再任务前缀猜）、活动行直达节点。
func TestBuildSessionNetwork_StampedRunsBuildSessionTree(t *testing.T) {
	net := trajectory.AgentNetwork{Ok: true, Root: trajectory.AgentNode{
		ID: "root", Kind: "root", Status: "completed", Tokens: 42,
		Children: []trajectory.AgentNode{
			{ID: "call_parent", Kind: "subagent", Task: "父任务预览", Status: "completed", ToolCalls: 5, Errors: 1, Tokens: 900, FirstTs: 100, LastTs: 200},
			{ID: "call_child", Kind: "subagent", Task: "子任务预览", Status: "completed", ToolCalls: 2, Tokens: 300, FirstTs: 150, LastTs: 190},
		},
	}}
	created := time.Date(2026, 11, 17, 10, 0, 0, 0, time.Local)
	runs := SubagentRunsView{Available: true, Runs: []SubagentRunView{
		{Ref: "sa_child", Kind: "subagent", Status: "completed", Task: "子任务全文", ParentRef: "sa_parent", SpawnCallID: "call_child", CreatedAt: created.Add(time.Minute), UpdatedAt: created.Add(2 * time.Minute), LastText: "子代理最终答复", LastTool: "read_file: a.go"},
		{Ref: "sa_parent", Kind: "subagent", Status: "running", Task: "父任务全文", SpawnCallID: "call_parent", CreatedAt: created, UpdatedAt: created.Add(3 * time.Minute), LastText: "父代理进行中"},
	}}
	buildSessionNetwork(&net, runs)
	if len(net.Root.Children) != 1 {
		t.Fatalf("stamped runs build one root child, got %d", len(net.Root.Children))
	}
	parent := net.Root.Children[0]
	if parent.ID != "sa_parent" || parent.Status != "running" || parent.LastText != "父代理进行中" {
		t.Fatalf("parent session node wrong: %+v", parent)
	}
	if parent.ToolCalls != 5 || parent.Errors != 1 || parent.Tokens != 900 || parent.FirstTs != 100 || parent.LastTs != 200 {
		t.Fatalf("spawnCallId stats join missing: %+v", parent)
	}
	if len(parent.Children) != 1 {
		t.Fatalf("nested run must nest under parentRef, got %d children", len(parent.Children))
	}
	child := parent.Children[0]
	if child.ID != "sa_child" || child.Task != "子任务全文" || child.LastTool != "read_file: a.go" || child.LastText != "子代理最终答复" {
		t.Fatalf("child session node wrong: %+v", child)
	}
	if child.ToolCalls != 2 || child.Tokens != 300 {
		t.Fatalf("child stats join missing: %+v", child)
	}
	if net.Root.Status != "running" {
		t.Fatalf("root must surface running subtree, got %q", net.Root.Status)
	}
	if net.Root.Tokens != 42 {
		t.Fatalf("root own stats must stay untouched, got %d", net.Root.Tokens)
	}
}

// buildSessionNetwork：旧数据（无盖章）回落 legacy 富化路径，行为不变；
// 新旧混合会话里旧 run 走启发式兜底联结统计。
func TestBuildSessionNetwork_UnstampedFallsBackToLegacy(t *testing.T) {
	net := trajectory.AgentNetwork{Ok: true, Root: trajectory.AgentNode{
		ID: "root", Kind: "root",
		Children: []trajectory.AgentNode{{ID: "call_1", Kind: "subagent", Task: "调研土壤修复", Status: "running"}},
	}}
	runs := SubagentRunsView{Available: true, Runs: []SubagentRunView{{
		Ref: "sa_1", Kind: "subagent", Status: "completed", Model: "m1", Task: "调研土壤修复并输出报告",
	}}}
	buildSessionNetwork(&net, runs)
	if n := net.Root.Children[0]; n.ID != "call_1" || n.Status != "completed" || n.Model != "m1" {
		t.Fatalf("legacy fallback not applied: %+v", n)
	}
}

// buildSessionNetwork：ghost 父（parentRef 指向不存在的 run）按 root 挂载降级，
// 绝不丢节点。
func TestBuildSessionNetwork_GhostParentFallsBackToRoot(t *testing.T) {
	net := trajectory.AgentNetwork{Ok: true, Root: trajectory.AgentNode{ID: "root", Kind: "root"}}
	created := time.Date(2026, 11, 17, 10, 0, 0, 0, time.Local)
	runs := SubagentRunsView{Available: true, Runs: []SubagentRunView{{
		Ref: "sa_orphan", Kind: "subagent", Status: "completed", Task: "孤儿运行",
		ParentRef: "sa_ghost", SpawnCallID: "call_x", CreatedAt: created, UpdatedAt: created,
	}}}
	buildSessionNetwork(&net, runs)
	if len(net.Root.Children) != 1 || net.Root.Children[0].ID != "sa_orphan" {
		t.Fatalf("ghost-parent run must stay visible at root, got %+v", net.Root.Children)
	}
}

// buildSessionNetwork：未被 run 承载的 legacy 节点（ephemeral 派发等无 meta
// 场景）原样保留在 root——「任何派发过的 agent 都可见」不变量。
func TestBuildSessionNetwork_OrphanLegacyNodesKept(t *testing.T) {
	net := trajectory.AgentNetwork{Ok: true, Root: trajectory.AgentNode{
		ID: "root", Kind: "root",
		Children: []trajectory.AgentNode{
			{ID: "call_ephemeral", Kind: "subagent", Task: "临时派发", Status: "completed", ToolCalls: 3, Tokens: 100},
		},
	}}
	created := time.Date(2026, 11, 17, 10, 0, 0, 0, time.Local)
	runs := SubagentRunsView{Available: true, Runs: []SubagentRunView{{
		Ref: "sa_1", Kind: "subagent", Status: "completed", Task: "持久运行",
		SpawnCallID: "call_persisted", CreatedAt: created, UpdatedAt: created,
	}}}
	buildSessionNetwork(&net, runs)
	ids := map[string]bool{}
	for _, n := range net.Root.Children {
		ids[n.ID] = true
	}
	if !ids["sa_1"] || !ids["call_ephemeral"] {
		t.Fatalf("session node + orphan legacy node both expected, got %v", ids)
	}
}

// 子代理实弹批：嵌套层（级别≥2）事件日志 task 节点 ID 被 subSinkFor 重写为
// "父call/本call" 复合，而 SpawnCallID 盖的是本层原始 call id——直等失配曾
// 让 legacy 节点成 orphan 与会话节点并存（同一孙代两张卡）。尾段找回+父链
// 前缀消歧后应并成一颗会话子树。
func TestBuildSessionNetwork_NestedCompositeCallIDJoins(t *testing.T) {
	net := trajectory.AgentNetwork{Ok: true, Root: trajectory.AgentNode{
		ID: "root", Kind: "root",
		Children: []trajectory.AgentNode{
			{ID: "call_1", Kind: "subagent", Task: "父任务预览", Status: "completed", ToolCalls: 4, Tokens: 700, FirstTs: 10, LastTs: 90},
			// 嵌套孙代节点：复合 ID（本层原始 call id 是 call_2）。
			{ID: "call_1/call_2", Kind: "subagent", Task: "孙任务预览", Status: "completed", ToolCalls: 2, Tokens: 260, FirstTs: 40, LastTs: 80},
		},
	}}
	created := time.Date(2026, 11, 18, 10, 0, 0, 0, time.Local)
	runs := SubagentRunsView{Available: true, Runs: []SubagentRunView{
		{Ref: "sa_child", Kind: "subagent", Status: "completed", Task: "孙任务全文", ParentRef: "sa_parent", SpawnCallID: "call_2", CreatedAt: created.Add(time.Minute), UpdatedAt: created.Add(2 * time.Minute)},
		{Ref: "sa_parent", Kind: "subagent", Status: "completed", Task: "父任务全文", SpawnCallID: "call_1", CreatedAt: created, UpdatedAt: created.Add(3 * time.Minute)},
	}}
	buildSessionNetwork(&net, runs)
	if len(net.Root.Children) != 1 {
		t.Fatalf("复合 ID 联键后 root 应只剩会话骨架 1 个孩子, got %d", len(net.Root.Children))
	}
	parent := net.Root.Children[0]
	if parent.ID != "sa_parent" || parent.ToolCalls != 4 || parent.Tokens != 700 {
		t.Fatalf("父会话节点统计联结缺失: %+v", parent)
	}
	if len(parent.Children) != 1 || parent.Children[0].ID != "sa_child" {
		t.Fatalf("孙代应挂在父会话节点下: %+v", parent.Children)
	}
	if parent.Children[0].ToolCalls != 2 || parent.Children[0].Tokens != 260 {
		t.Fatalf("孙代统计联结缺失: %+v", parent.Children[0])
	}
}
