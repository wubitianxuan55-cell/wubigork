package trajectory

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

func TestFoldAgentNetworkEmpty(t *testing.T) {
	net := FoldAgentNetwork(nil, 1_000_000)
	if !net.Ok || net.Window != 1_000_000 {
		t.Fatalf("empty net wrong: %+v", net)
	}
	if net.Root.ID != "root" || net.Root.ToolCalls != 0 || len(net.Root.Children) != 0 {
		t.Fatalf("root wrong: %+v", net.Root)
	}
}

func TestFoldAgentNetworkWithSubagent(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "并行调研两个方向"}),
		// 子代理 A：task 调用 + 它的两个子调用（parentId 指向 taskA）
		entry(3, "tool_dispatch", map[string]any{"id": "taskA", "name": "task", "args": `{"description":"调研A","prompt":"调研模块A的现状"}`, "partial": false}),
		entry(4, "tool_dispatch", map[string]any{"id": "a1", "name": "grep", "args": `{"pattern":"A"}`, "partial": false, "parentId": "taskA"}),
		entry(5, "tool_result", map[string]any{"id": "a1", "name": "grep", "output": "matchA"}),
		entry(6, "tool_dispatch", map[string]any{"id": "a2", "name": "read_file", "args": `{"path":"a.go"}`, "partial": false, "parentId": "taskA"}),
		entry(7, "tool_result", map[string]any{"id": "a2", "name": "read_file", "output": "fileA"}),
		entry(8, "tool_result", map[string]any{"id": "taskA", "name": "task", "output": "<task-result>done</task-result>"}),
		// 子代理 B：task 调用 + 一个错误子调用
		entry(9, "tool_dispatch", map[string]any{"id": "taskB", "name": "task", "args": `{"description":"调研B","prompt":"调研模块B"}`, "partial": false}),
		entry(10, "tool_dispatch", map[string]any{"id": "b1", "name": "bash", "args": "rm -rf /", "partial": false, "parentId": "taskB"}),
		entry(11, "tool_result", map[string]any{"id": "b1", "name": "bash", "output": "", "err": "denied"}),
		entry(12, "tool_result", map[string]any{"id": "taskB", "name": "task", "output": "<task-result>partial</task-result>"}),
		// root 直属工具
		entry(13, "tool_dispatch", map[string]any{"id": "m1", "name": "ls", "args": "{}", "partial": false}),
		entry(14, "tool_result", map[string]any{"id": "m1", "name": "ls", "output": "files"}),
	}
	net := FoldAgentNetwork(entries, 0)
	root := net.Root
	if root.ToolCalls != 1 {
		t.Fatalf("root toolCalls = %d, want 1（直属 m1）", root.ToolCalls)
	}
	if len(root.Children) != 2 {
		t.Fatalf("subagents = %d, want 2", len(root.Children))
	}
	a := root.Children[0]
	if a.Name != "task" || a.Task != "调研A" || a.ToolCalls != 2 || a.Status != "completed" {
		t.Fatalf("subagent A wrong: %+v", a)
	}
	if a.Tokens <= 0 || a.FirstTs == 0 || a.LastTs == 0 {
		t.Fatalf("subagent A stats missing: %+v", a)
	}
	b := root.Children[1]
	if b.ToolCalls != 1 || b.Errors != 1 || b.Status != "error" {
		t.Fatalf("subagent B wrong: %+v", b)
	}
}

func TestFoldAgentNetworkRunning(t *testing.T) {
	entries := []session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "tool_dispatch", map[string]any{"id": "taskX", "name": "task", "args": `{"description":"X","prompt":"x"}`, "partial": false}),
		entry(3, "tool_dispatch", map[string]any{"id": "x1", "name": "grep", "args": "{}", "partial": false, "parentId": "taskX"}),
		// 没有 x1 的 result → 子代理 running
	}
	net := FoldAgentNetwork(entries, 0)
	if len(net.Root.Children) != 1 {
		t.Fatalf("children = %d, want 1", len(net.Root.Children))
	}
	if net.Root.Children[0].Status != "running" {
		t.Fatalf("subagent status = %q, want running", net.Root.Children[0].Status)
	}
	if net.Root.Status != "running" {
		t.Fatalf("root status = %q, want running", net.Root.Status)
	}
}
