package app

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/contextview"
	"github.com/gaea/gaea/internal/gaea/trajectory"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
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

// GaeaContextNodeDetail 懒加载浏览器节点的「完整调用」详情（v4.80）：按 seq
// 回读当前会话日志（tool_result 配对 dispatch 取参数；user/assistant 取全文），
// 不随节点列表整包下发。仅当前会话有效（与 GaeaContextView 同源）。
func (a *App) GaeaContextNodeDetail(seq int64) (contextview.NodeDetail, error) {
	c := gaeaCtrl()
	if c == nil {
		return contextview.NodeDetail{}, fmt.Errorf("会话未就绪")
	}
	entries, err := session.ReadEntriesFor(c.SessionPath())
	if err != nil {
		return contextview.NodeDetail{}, err
	}
	d, ok := contextview.NodeDetailFor(entries, seq)
	if !ok {
		return contextview.NodeDetail{}, fmt.Errorf("未找到 seq=%d 的可展开节点", seq)
	}
	resolveNodeImages(&d, gaeaCwd())
	return d, nil
}

// resolveNodeImages 把详情里的图片引用解析为缩略卡数据（2.5b 后半）：相对
// 引用按 gaea cwd 解析为绝对路径；文件存在则仅解码头部取尺寸，并按官方
// patch 口径（⌈w/28⌉×⌈h/28⌉，先档位缩放再封顶）估算 token。文件缺失/
// 解码失败诚实降级（Exists=false / 尺寸留零），绝不阻塞详情主体。
func resolveNodeImages(d *contextview.NodeDetail, cwd string) {
	if len(d.ImageRefs) == 0 {
		return
	}
	images := make([]contextview.NodeImage, 0, len(d.ImageRefs))
	for _, ref := range d.ImageRefs {
		img := contextview.NodeImage{Ref: ref, Path: ref}
		if !filepath.IsAbs(ref) && cwd != "" {
			img.Path = filepath.Join(cwd, ref)
			img.RefCwd = cwd
		}
		f, err := os.Open(img.Path)
		if err != nil {
			images = append(images, img) // Exists=false：前端灰态「文件不存在」
			continue
		}
		cfg, _, err := image.DecodeConfig(f)
		f.Close()
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
			// 文件在但尺寸未知（svg/ico 等非栅格或不受支持格式）。
			img.Exists = true
			images = append(images, img)
			continue
		}
		img.Exists = true
		img.Width, img.Height = cfg.Width, cfg.Height
		est := contextview.EstimateImageTokens(cfg.Width, cfg.Height)
		img.ScaledW, img.ScaledH = est.ScaledW, est.ScaledH
		img.StdTokens, img.HighTokens = est.StdTokens, est.HighTokens
		images = append(images, img)
	}
	d.Images = images
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

// runMatchesNode 判定一个已落盘 run 是否已被树上的节点承载：
// ref 直等（合成节点 id=sa_ ref），或任务摘要前缀双向（与前端
// AgentTree.matchRunForNode / 后端富化同口径）。
func runMatchesNode(r SubagentRunView, node trajectory.AgentNode) bool {
	if r.Ref != "" && r.Ref == node.ID {
		return true
	}
	return node.Task != "" && r.Task != "" &&
		(strings.HasPrefix(r.Task, node.Task) || strings.HasPrefix(node.Task, r.Task))
}

// enrichAgentNetwork 把 subagents/ meta 合并进 Agent 网络子代理节点：
// 按任务摘要匹配（子代理 transcript 首条 user 消息 ≈ task 调用的 prompt 预览）。
// 树来自事件日志折叠，「零工具调用」的子代理（纯调研/禁用工具）在日志里
// 没有子记录、FoldAgentNetwork 不为其建节点——这里对没有任何节点承载的
// run 补挂合成节点（id=sa_ ref），保证任何已落盘运行都在树上可见。
func enrichAgentNetwork(net *trajectory.AgentNetwork, runs SubagentRunsView) {
	if !runs.Available || len(runs.Runs) == 0 {
		return
	}
	matched := make([]bool, len(runs.Runs))
	for i := range net.Root.Children {
		node := &net.Root.Children[i]
		for j, r := range runs.Runs {
			if runMatchesNode(r, *node) {
				node.Status = r.Status
				node.Model = r.Model
				if node.Task == "" {
					node.Task = r.Task
				}
				matched[j] = true
				break
			}
		}
	}
	for j, r := range runs.Runs {
		if matched[j] || r.Kind == "model_tool" {
			continue
		}
		status := r.Status
		if status == "failed" {
			status = "error"
		} else if status != "running" {
			status = "completed"
		}
		net.Root.Children = append(net.Root.Children, trajectory.AgentNode{
			ID:        r.Ref,
			Name:      r.Task,
			Kind:      "subagent",
			Status:    status,
			Model:     r.Model,
			Task:      r.Task,
			ToolCalls: r.ToolCalls,
			FirstTs:   r.CreatedAt.Unix(),
			LastTs:    r.UpdatedAt.Unix(),
		})
	}
}

// GaeaSubagentContextView 返回指定子代理（sa_ ref）会话的上下文构成快照
// （2.5e 后半：Agent 网络节点 → 子代理上下文跳转）。与 GaeaContextView
// 同一折叠管线，仅数据源换成子代理 transcript；ref 非法/transcript 缺失
// 诚实报错。
func (a *App) GaeaSubagentContextView(sessionPath, ref string) (contextview.ContextTimeline, error) {
	if !gaeaAgent.ValidRunRef(ref) {
		return contextview.ContextTimeline{}, fmt.Errorf("非法的子代理引用 %q", ref)
	}
	dir := sessionDirForPath(sessionPath)
	if dir == "" {
		return contextview.ContextTimeline{}, fmt.Errorf("会话路径无法定位目录")
	}
	transcript := filepath.Join(dir, "subagents", ref+".jsonl")
	// 子代理 transcript 是独立 jsonl（无 legacy 投影语义），直接读修复管道。
	entries, err := session.ReadLogRepaired(transcript)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return contextview.EmptyTimeline(), nil
		}
		return contextview.ContextTimeline{}, err
	}
	tl := contextview.FoldTimeline(entries, 0, 0)
	return tl, nil
}
