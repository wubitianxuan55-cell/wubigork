package app

import (
	"errors"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/contextview"
	"github.com/gaea/gaea/internal/gaea/trajectory"
)

// GaeaContextView 返回当前会话的上下文构成快照（dsh-context Go 移植 Phase A）：
// 六分类当前组成、逐请求趋势、上下文事件、模型可见节点与归档。
// 会话或日志不存在时返回空快照（ok=true），不报错——前端空态渲染。
// 事件日志缺失时回退 legacy 会话投影（旧会话仍可看板，见 session.ReadEntriesFor）。
func (a *App) GaeaContextView() (contextview.ContextTimeline, error) {
	c := gaeaCtrl()
	if c == nil {
		return contextview.EmptyTimeline(), nil
	}
	entries, err := session.ReadEntriesFor(c.SessionPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return contextview.EmptyTimeline(), nil
		}
		return contextview.EmptyTimeline(), err
	}
	_, window := c.ContextSnapshot()
	return contextview.FoldTimeline(entries, int64(window), 0), nil
}

// GaeaTrajectory 返回当前会话的轨迹时间线（dsh 轨迹标签的 Go 移植）：
// 按 轮次 → 步骤 组织用户输入、推理、回复、工具调用与过程事件。
// 会话或日志不存在时返回空快照（ok=true），不报错。
// 事件日志缺失时回退 legacy 会话投影（旧会话仍可看板，见 session.ReadEntriesFor）。
func (a *App) GaeaTrajectory() (trajectory.Trajectory, error) {
	c := gaeaCtrl()
	if c == nil {
		return trajectory.EmptyTrajectory(), nil
	}
	entries, err := session.ReadEntriesFor(c.SessionPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return trajectory.EmptyTrajectory(), nil
		}
		return trajectory.EmptyTrajectory(), err
	}
	return trajectory.FoldTrajectory(entries), nil
}

// GaeaAgentNetwork 返回当前会话的 Agent 网络（主 agent 根 + 子代理树），
// 并用 subagents/ meta 富化子代理节点的任务摘要/状态/模型。
// 事件日志缺失时回退 legacy 会话投影（见 session.ReadEntriesFor）。
func (a *App) GaeaAgentNetwork() (trajectory.AgentNetwork, error) {
	c := gaeaCtrl()
	if c == nil {
		return trajectory.EmptyAgentNetwork(), nil
	}
	path := c.SessionPath()
	entries, err := session.ReadEntriesFor(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return trajectory.EmptyAgentNetwork(), nil
		}
		return trajectory.EmptyAgentNetwork(), err
	}
	_, window := c.ContextSnapshot()
	net := trajectory.FoldAgentNetwork(entries, int64(window))
	enrichAgentNetwork(&net, a.GaeaSubagentRuns(path))
	return net, nil
}

// enrichAgentNetwork 把 subagents/ meta 合并进 Agent 网络子代理节点：
// 按任务摘要匹配（子代理 transcript 首条 user 消息 ≈ task 调用的 prompt 预览）。
func enrichAgentNetwork(net *trajectory.AgentNetwork, runs SubagentRunsView) {
	if !runs.Available || len(runs.Runs) == 0 {
		return
	}
	for i := range net.Root.Children {
		node := &net.Root.Children[i]
		for _, r := range runs.Runs {
			if node.Task != "" && r.Task != "" &&
				(strings.HasPrefix(r.Task, node.Task) || strings.HasPrefix(node.Task, r.Task)) {
				node.Status = r.Status
				node.Model = r.Model
				if node.Task == "" {
					node.Task = r.Task
				}
				break
			}
		}
	}
}
