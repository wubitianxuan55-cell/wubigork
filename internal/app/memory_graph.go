package app

import (
	"sort"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// ── 记忆语义图谱（阶段五 5.1，docs/gaea-memory-graph-51-design-2026-09.md）────
//
// 与 GaeaMemoryGraph（按标签/分类拼边的「关联图」）不同，这里是**事件日志投影**
// 的语义图：memory_events 唯一事实源 → ProjectEvents 纯函数投影 → mem_graph_*
// 物化（读取前按水位懒重建，删库可自愈）。节点三类（entity/event/source），
// [MEM:name] 图节点寻址（实体 id=mem:<project>/<name>）；悬空互引不入图但带
// 计数留痕。前端 GraphView 复用同一渲染面（source="semantic" 档）。

// SemanticGraphView 语义图谱视图（GraphView 可渲染形态 + 日志统计）。
type SemanticGraphView struct {
	Nodes []GraphNode `json:"nodes"`
	Links []GraphLink `json:"links"`
	// 日志与悬空统计（图页统计行/悬空徽标）。
	EventCount   int64    `json:"eventCount"`   // 日志事件总数（追加式=max seq）
	EntityCount  int      `json:"entityCount"`  // 实体节点数
	DanglingRefs []string `json:"danglingRefs"` // 悬空互引目标（排序去重）
	// ProjectedSeq 是本次展示消费到的日志水位（对账口径透明）。
	ProjectedSeq int64 `json:"projectedSeq"`
}

// semanticViewEventCap 是视图层事件节点上限（图渲染面在实体/来源全保留的前提下
// 只带最新事件——事件随触达无限增长，全量进 3D 力导图既卡也无信息增量）。
const semanticViewEventCap = 220

// GaeaMemorySemanticGraph 返回事件日志投影的语义图谱（懒重建后读物化）。
func (a *App) GaeaMemorySemanticGraph() SemanticGraphView {
	view := SemanticGraphView{Nodes: []GraphNode{}, Links: []GraphLink{}, DanglingRefs: []string{}}
	gdb := db.GetDatabase(config.MemoryUserDir())
	if gdb == nil {
		return view
	}
	if _, err := memory.EnsureGraphFresh(gdb); err != nil {
		return view
	}
	g, err := memory.MaterializedGraph(gdb)
	if err != nil {
		return view
	}
	stats := memory.GraphMaterializationStats(gdb)
	view.EventCount = stats.Events
	view.DanglingRefs = stats.Dangling
	if view.DanglingRefs == nil {
		view.DanglingRefs = []string{}
	}

	// 视图裁剪：事件节点超限时丢最旧（id=ev:<seq>，数值序=时间序）。
	events := make([]memory.GraphNode, 0, len(g.Nodes))
	others := make([]memory.GraphNode, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		if n.NType == memory.NodeEvent {
			events = append(events, n)
		} else {
			others = append(others, n)
		}
	}
	sort.Slice(events, func(i, j int) bool { return eventSeqOf(events[i].ID) > eventSeqOf(events[j].ID) })
	keptEvents := events
	if len(keptEvents) > semanticViewEventCap {
		keptEvents = keptEvents[:semanticViewEventCap]
	}
	kept := make(map[string]bool, len(g.Nodes))
	for _, n := range others {
		kept[n.ID] = true
	}
	for _, n := range keptEvents {
		kept[n.ID] = true
	}
	for _, n := range g.Nodes {
		if !kept[n.ID] {
			continue
		}
		view.Nodes = append(view.Nodes, GraphNode{
			ID:   n.ID,
			Name: n.Name,
			Type: n.NType,
			Desc: nodeDesc(n),
			Val:  n.Weight,
		})
	}
	for _, e := range g.Edges {
		if !kept[e.Src] || !kept[e.Tgt] {
			continue
		}
		view.Links = append(view.Links, GraphLink{Source: e.Src, Target: e.Tgt, Type: e.EType})
	}
	for _, n := range g.Nodes {
		if n.NType == memory.NodeEntity {
			view.EntityCount++
		}
	}
	view.ProjectedSeq = g.MaxSeq
	return view
}

// eventSeqOf 从事件节点 id（ev:<seq>）取回 seq（解析失败返回 0，排最后）。
func eventSeqOf(id string) int64 {
	s := strings.TrimPrefix(id, "ev:")
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// nodeDesc 是语义图节点的展示描述（实体带状态/类型口径；事件/来源原样）。
func nodeDesc(n memory.GraphNode) string {
	switch n.NType {
	case memory.NodeEntity:
		parts := make([]string, 0, 4)
		if n.State != "" {
			parts = append(parts, n.State)
		}
		if n.MemType != "" {
			parts = append(parts, n.MemType)
		}
		if n.Kind != "" {
			parts = append(parts, n.Kind)
		}
		if n.Space != "" {
			parts = append(parts, n.Space)
		}
		if d := strings.TrimSpace(n.Desc); d != "" {
			parts = append(parts, d)
		}
		return strings.Join(parts, " · ")
	default:
		return n.Desc
	}
}
