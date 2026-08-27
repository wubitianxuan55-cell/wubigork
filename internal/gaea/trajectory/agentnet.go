package trajectory

import (
	"encoding/json"
	"strings"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// FoldAgentNetwork 从会话日志折叠出 Agent 网络：root 主节点 + 子代理节点
// （拥有子工具记录的工具调用）。纯函数：同输入必同输出。
func FoldAgentNetwork(entries []session.LogEntry, window int64) AgentNetwork {
	root := AgentNode{ID: "root", Name: "主 agent", Kind: "root", Status: "completed"}
	// tool id → 记录（dispatch+result 合并）
	tools := map[string]*agentTool{}
	order := []string{}
	for _, e := range entries {
		if e.Kind == "tool_dispatch" {
			var p struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Args     string `json:"args"`
				Partial  bool   `json:"partial,omitempty"`
				ParentID string `json:"parentId,omitempty"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil || p.ID == "" {
				continue
			}
			t, ok := tools[p.ID]
			if !ok {
				t = &agentTool{id: p.ID, parentID: p.ParentID, ts: e.Ts}
				tools[p.ID] = t
				order = append(order, p.ID)
			}
			t.name = p.Name
			t.args = p.Args
			t.parentID = p.ParentID
			continue
		}
		if e.Kind == "tool_result" {
			var p struct {
				ID     string `json:"id"`
				Name   string `json:"name,omitempty"`
				Output string `json:"output,omitempty"`
				Err    string `json:"err,omitempty"`
			}
			if err := json.Unmarshal(e.Payload, &p); err != nil || p.ID == "" {
				continue
			}
			t, ok := tools[p.ID]
			if !ok {
				t = &agentTool{id: p.ID, ts: e.Ts}
				tools[p.ID] = t
				order = append(order, p.ID)
			}
			if p.Name != "" {
				t.name = p.Name
			}
			t.output = p.Output
			t.err = p.Err
			t.resultTs = e.Ts
			t.result = true
		}
	}

	// 统计子记录归属：parentID → 子记录列表
	children := map[string][]string{}
	for _, id := range order {
		t := tools[id]
		if t.parentID != "" {
			children[t.parentID] = append(children[t.parentID], id)
		}
	}

	// 子代理节点 = 拥有子记录的元工具调用；其余记录归 root 或对应子代理统计。
	subIDs := map[string]bool{}
	for pid := range children {
		if _, ok := tools[pid]; ok {
			subIDs[pid] = true
		}
	}

	// 先建子代理节点（含自身的 stats），再聚合到 root。
	subNodes := map[string]AgentNode{}
	for _, id := range order {
		t := tools[id]
		if !subIDs[id] {
			continue
		}
		node := AgentNode{
			ID:     id,
			Name:   t.name,
			Kind:   "subagent",
			Status: "completed",
			Task:   taskSummary(t.args),
		}
		aggregateNode(&node, t, children, tools)
		subNodes[id] = node
	}

	// root 聚合：无 parent 且非子代理节点的记录 + 子代理节点本身。
	for _, id := range order {
		t := tools[id]
		if t.parentID != "" {
			continue // 归子代理统计，已在 aggregateNode 内处理
		}
		if subIDs[id] {
			continue // 子代理节点作为子节点挂载，不入 root 直接统计
		}
		aggregateTool(&root, t)
	}
	root.Children = make([]AgentNode, 0, len(subNodes))
	rootRunning := false
	for _, id := range order {
		if subIDs[id] {
			n := subNodes[id]
			if n.Status == "running" {
				rootRunning = true
			}
			root.Children = append(root.Children, n)
		}
	}
	if root.Errors > 0 {
		root.Status = "error"
	} else if rootRunning || rootHasRunning(tools, children) {
		root.Status = "running"
	}
	return AgentNetwork{Ok: true, Window: window, Root: root}
}

type agentTool struct {
	id       string
	name     string
	args     string
	output   string
	err      string
	parentID string
	ts       int64
	resultTs int64
	result   bool
}

// aggregateNode 统计子代理节点的子记录（不含 spawn 调用本身）。
func aggregateNode(node *AgentNode, t *agentTool, children map[string][]string, tools map[string]*agentTool) {
	for _, cid := range children[t.id] {
		ct := tools[cid]
		if ct == nil {
			continue
		}
		aggregateToolStats(node, ct)
	}
	node.Status = "completed"
	if node.Errors > 0 {
		node.Status = "error"
		return
	}
	for _, cid := range children[t.id] {
		ct := tools[cid]
		if ct != nil && !ct.result {
			node.Status = "running"
			break
		}
	}
}

func aggregateTool(node *AgentNode, t *agentTool) {
	aggregateToolStats(node, t)
}

func aggregateToolStats(node *AgentNode, t *agentTool) {
	node.ToolCalls++
	if t.err != "" {
		node.Errors++
	}
	if t.ts > 0 && (node.FirstTs == 0 || t.ts < node.FirstTs) {
		node.FirstTs = t.ts
	}
	last := t.resultTs
	if last == 0 {
		last = t.ts
	}
	if last > 0 && last > node.LastTs {
		node.LastTs = last
	}
	node.Tokens += estimateTokens(t.args) + estimateTokens(t.output) + estimateTokens(t.err)
}

// rootHasRunning 检查 root 直属（无 parent）工具调用是否有未完成项。
func rootHasRunning(tools map[string]*agentTool, children map[string][]string) bool {
	for id, t := range tools {
		if t.parentID != "" {
			continue
		}
		if _, isSub := children[id]; isSub {
			continue
		}
		if !t.result {
			return true
		}
	}
	return false
}

// taskSummary 从 task 调用参数提取 description（优先）或 prompt 预览。
func taskSummary(args string) string {
	if args == "" {
		return ""
	}
	var p struct {
		Description string `json:"description"`
		Prompt      string `json:"prompt"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return ""
	}
	s := strings.TrimSpace(p.Description)
	if s == "" {
		s = strings.TrimSpace(p.Prompt)
	}
	return preview(s, 120)
}
