package app

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/agent/session"
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
	p := filepath.Join(t.TempDir(), "hist-session.jsonl")
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
