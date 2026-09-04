package app

import (
	"testing"
	"time"

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
