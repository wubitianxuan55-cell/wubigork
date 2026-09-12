package memory

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ── 记忆语义图谱投影（阶段五 5.1）──────────────────────────────────
//
// ProjectEvents 是事件日志 → 语义图谱的纯函数投影：同一份事件输入（seq 升序）
// 永远得到同一张图（可重建性出口判据的承担者）。图里三种节点——
//
//	entity  记忆实体（[MEM:name] 图节点寻址；id=mem:<project>/<name>）
//	event   一次记忆事件（id=ev:<seq>）
//	source  事件来源（沉淀会话；id=src:<project>/<session|local>）
//
// 与三类边——
//
//	produces   source → event  （事件从哪个会话来）
//	affects    event  → entity （事件动了哪条记忆）
//	references entity → entity （实体互引：最新 save 的 [MEM:x]/[[x]]）
//
// 悬空拒写：references 边只在两端实体都存在（日志里出现过实体）时物化；悬空
// 目标不进图，但计入 Dangling 诊断列表（排序去重）——不静默。cite 事件永不
// 创建实体节点（模型可能幻觉键名），只留事件节点与来源边。

// 节点/边类型常量（前端语义色按 ntype 取，见 domainColors.ts）。
const (
	NodeEntity = "entity"
	NodeEvent  = "event"
	NodeSource = "source"

	EdgeProduces   = "produces"
	EdgeAffects    = "affects"
	EdgeReferences = "references"
)

// GraphNode 投影图谱节点。
type GraphNode struct {
	ID      string   `json:"id"`
	NType   string   `json:"ntype"` // entity / event / source
	Name    string   `json:"name"`
	Desc    string   `json:"desc"`
	Weight  float64  `json:"weight"`
	State   string   `json:"state,omitempty"`   // 实体：active / archived
	Kind    string   `json:"kind,omitempty"`    // 实体：semantic/episodic/procedural
	MemType string   `json:"memType,omitempty"` // 实体：user/feedback/project/reference
	Tags    []string `json:"tags,omitempty"`
	Space   string   `json:"space,omitempty"`
	Project string   `json:"project,omitempty"`
}

// GraphEdge 投影图谱边（有向）。
type GraphEdge struct {
	Src   string `json:"src"`
	Tgt   string `json:"tgt"`
	EType string `json:"etype"` // produces / affects / references
}

// Graph 是一次投影的产物（节点/边均按 ID 排序，两次投影可逐字节比较）。
type Graph struct {
	Nodes    []GraphNode
	Edges    []GraphEdge
	Dangling []string // 悬空互引目标（有引用、无实体；排序去重）
	MaxSeq   int64    // 投影消费到的最大事件序号
}

// entityState 是投影折算中一个实体的累积状态（last-write-wins：以最新 save
// 事件的元数据为准，archive/unarchive 只翻状态位）。
type entityState struct {
	node GraphNode
}

// entityID / eventID / sourceID 是三类节点的稳定寻址（[MEM:name] 图寻址的
// 落点：图上实体 id 恒为 mem:<project>/<name>）。
func entityID(project, name string) string { return fmt.Sprintf("mem:%s/%s", project, name) }
func eventID(seq int64) string             { return fmt.Sprintf("ev:%d", seq) }
func sourceID(project, session string) string {
	if session == "" {
		session = "local"
	}
	return fmt.Sprintf("src:%s/%s", project, session)
}

// ProjectEvents 把事件日志投影成语义图谱。events 必须按 seq 升序（LoadEvents
// 的输出序）；函数不改输入、不读时钟、不落 IO——同一输入两次调用输出一致。
func ProjectEvents(events []Event) *Graph {
	g := &Graph{}
	idx := map[string]int{} // 节点 id → g.Nodes 下标（fold 中元数据会更新，走 upsert）
	upsert := func(n GraphNode) {
		if i, ok := idx[n.ID]; ok {
			g.Nodes[i] = n
			return
		}
		idx[n.ID] = len(g.Nodes)
		g.Nodes = append(g.Nodes, n)
	}
	addEdge := func(src, tgt, etype string) {
		g.Edges = append(g.Edges, GraphEdge{Src: src, Tgt: tgt, EType: etype})
	}

	entities := map[string]*entityState{}
	latestRefs := map[string][]string{} // 实体 → 最新 save 的互引集（覆盖式）

	for _, e := range events {
		if e.Seq > g.MaxSeq {
			g.MaxSeq = e.Seq
		}
		evID := eventID(e.Seq)
		upsert(GraphNode{
			ID: evID, NType: NodeEvent, Name: eventTitle(e),
			Desc: eventDesc(e) + eventTimeNote(e), Weight: 0.5,
			Space: e.Space, Project: e.Project,
		})
		srcID := sourceID(e.Project, e.SourceSession)
		upsert(GraphNode{ID: srcID, NType: NodeSource, Name: sourceName(e), Weight: 0.8, Project: e.Project})
		addEdge(srcID, evID, EdgeProduces)

		switch e.Op {
		case OpSave, OpArchive, OpUnarchive, OpDelete, OpTouch, OpChangeType:
			id := entityID(e.Project, e.Name)
			st, ok := entities[id]
			if !ok {
				st = &entityState{node: GraphNode{
					ID: id, NType: NodeEntity, Name: e.Name,
					Weight: 1, State: "active", Project: e.Project,
				}}
				entities[id] = st
			}
			if e.Op == OpSave {
				if t := oneLine(e.Title); t != "" {
					st.node.Name = t
				}
				st.node.Desc = oneLine(e.Desc)
				st.node.Kind = e.Kind
				st.node.MemType = e.Type
				st.node.Tags = e.Tags
				st.node.Space = e.Space
				latestRefs[id] = e.Refs
			}
			switch e.Op {
			case OpArchive, OpDelete:
				st.node.State = "archived"
			case OpUnarchive, OpSave:
				st.node.State = "active"
			}
			upsert(st.node)
			addEdge(evID, id, EdgeAffects)
		case OpCite:
			// cite 永不创建实体（键可能悬空）；命中已知实体时补轻量 affects。
			id := entityID(e.Project, e.Name)
			if _, ok := entities[id]; ok {
				addEdge(evID, id, EdgeAffects)
			}
		}
	}

	// references 边：每个实体最新 save 的互引集，两端实体都在场才物化
	// （悬空拒写）；互引目标与源实体同项目（facts 唯一键=(project, name)，
	// 跨项目同名是两条记忆，互引不跨）。
	for id, refs := range latestRefs {
		for _, r := range refs {
			tgt := entityID(projectOfEntity(id), r)
			if _, ok := entities[tgt]; ok {
				addEdge(id, tgt, EdgeReferences)
			} else {
				g.Dangling = append(g.Dangling, r)
			}
		}
	}

	// 确定性输出：节点按 ID、边按 (Src,Tgt,EType)、悬空去重排序。
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	sort.Slice(g.Edges, func(i, j int) bool {
		a, b := g.Edges[i], g.Edges[j]
		if a.Src != b.Src {
			return a.Src < b.Src
		}
		if a.Tgt != b.Tgt {
			return a.Tgt < b.Tgt
		}
		return a.EType < b.EType
	})
	g.Dangling = dedupeSorted(g.Dangling)
	return g
}

// projectOfEntity 从实体 id（mem:<project>/<name>）取回 project 段。project
// slug（路径分隔符已替换为 -）与 name（kebab-case）都不含 "/"，末个 "/" 必是
// 分隔符。
func projectOfEntity(entID string) string {
	rest := strings.TrimPrefix(entID, "mem:")
	if i := strings.LastIndex(rest, "/"); i >= 0 {
		return rest[:i]
	}
	return rest
}

// eventTitle / eventDesc / sourceName 是事件/来源节点的展示文案。
func eventTitle(e Event) string {
	name := e.Name
	if t := oneLine(e.Title); t != "" {
		name = t
	}
	return e.Op + " · " + name
}

// eventDesc / eventTimeNote 是事件节点的展示文案。双时间轴标注烘进 Desc：
// desc 随投影进 mem_graph_nodes 物化，「物化=日志投影」的不变量因此天然保持
// （时间轴不作为节点字段——物化表没有这两列，字段化会破坏逐字段可比）。
// 只在事实时间与记录时间分歧≥1 秒才标注：写入路径 At 取写入时刻、与盖的
// recorded_at 同毫秒级是常态，零噪音；分歧只出现在「事实在过去、写入在当下」
// 的回填/导入/做梦衍生类路径，那正是要透出的审计信息。格式化用本地时区
// （物化是本机展示缓存，重建可复现以同机为准）。
func eventDesc(e Event) string {
	if d := oneLine(e.Desc); d != "" {
		return d
	}
	return oneLine(e.Excerpt)
}

func eventTimeNote(e Event) string {
	if e.At <= 0 || e.RecordedAt <= 0 {
		return ""
	}
	d := e.RecordedAt - e.At
	if d < 0 {
		d = -d
	}
	if d < int64(time.Second/time.Millisecond) {
		return ""
	}
	return fmt.Sprintf("（发生 %s · 记录 %s）",
		time.UnixMilli(e.At).Format("2006-01-02 15:04"),
		time.UnixMilli(e.RecordedAt).Format("2006-01-02 15:04"))
}

func sourceName(e Event) string {
	if e.SourceSession != "" {
		return e.SourceSession
	}
	return "本机写入"
}

// dedupeSorted 去重 + 排序（悬空列表的确定性出口）。
func dedupeSorted(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	sort.Strings(in)
	out := in[:1]
	for _, s := range in[1:] {
		if s != out[len(out)-1] {
			out = append(out, s)
		}
	}
	return out
}
