// Package whisper — memory_graph.go
// 轻量知识图谱（从旧 memory.go 提取，对齐 ackem memory/knowledgeGraph.ts）

package whisper

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─── Triple ───────────────────────────────────────────────────

// Triple 知识图谱三元组
type Triple struct {
	ID            string    `json:"id"`
	Subject       string    `json:"subject"`
	Predicate     string    `json:"predicate"`
	Object        string    `json:"object"`
	Confidence    float64   `json:"confidence"`
	SourceFactIDs []string  `json:"sourceFactIds"`
	CreatedAt     time.Time `json:"createdAt"`
}

// ─── KnowledgeGraph ───────────────────────────────────────────

// KnowledgeGraph 轻量知识图谱（实体倒排索引）
type KnowledgeGraph struct {
	mu        sync.RWMutex
	triples   []Triple
	entityIdx map[string][]int // entity → triple indices
}

// NewKnowledgeGraph 创建空图谱
func NewKnowledgeGraph() *KnowledgeGraph {
	return &KnowledgeGraph{entityIdx: make(map[string][]int)}
}

// Add 添加三元组
func (kg *KnowledgeGraph) Add(subj, pred, obj string, conf float64, src []string) Triple {
	kg.mu.Lock()
	defer kg.mu.Unlock()
	b := make([]byte, 8)
	rand.Read(b)
	t := Triple{
		ID:            "kg_" + hex.EncodeToString(b),
		Subject:       subj,
		Predicate:     pred,
		Object:        obj,
		Confidence:    conf,
		SourceFactIDs: src,
		CreatedAt:     time.Now(),
	}
	idx := len(kg.triples)
	kg.triples = append(kg.triples, t)
	kg.addIdx(subj, idx)
	kg.addIdx(obj, idx)
	return t
}

func (kg *KnowledgeGraph) addIdx(entity string, idx int) {
	key := strings.ToLower(entity)
	kg.entityIdx[key] = append(kg.entityIdx[key], idx)
}

// Query 文本查询（一跳），返回匹配的三元组
func (kg *KnowledgeGraph) Query(text string, max int) []Triple {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	ql := strings.ToLower(text)
	type scored struct {
		t Triple
		s float64
	}
	var rs []scored
	for _, t := range kg.triples {
		tl := strings.ToLower(t.Subject + " " + t.Predicate + " " + t.Object)
		s := 0.0
		// 实体命中：query 包含该三元组自身的 subject/object（避免全局索引误加分）
		for _, ent := range []string{t.Subject, t.Object} {
			if el := strings.ToLower(ent); el != "" && strings.Contains(ql, el) {
				s += 3.0
			}
		}
		for _, w := range strings.Fields(ql) {
			if len([]rune(w)) >= 2 && strings.Contains(tl, w) {
				s += 1.0
			}
		}
		if s >= 0.1 {
			rs = append(rs, scored{t, s})
		}
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].s > rs[j].s })
	if max > 0 && len(rs) > max {
		rs = rs[:max]
	}
	var out []Triple
	for _, x := range rs {
		out = append(out, x.t)
	}
	return out
}

// BuildContextBlock 构建知识图谱上下文块
func (kg *KnowledgeGraph) BuildContextBlock(text string, budget int) string {
	hits := kg.Query(text, KGQueryMaxTriples)
	if len(hits) == 0 {
		return ""
	}
	var lines []string
	lines = append(lines, "【知识图谱】")
	chars := 0
	for _, t := range hits {
		line := "- " + t.Subject + " —" + t.Predicate + "→ " + t.Object
		if chars+len([]rune(line)) > budget {
			break
		}
		lines = append(lines, line)
		chars += len([]rune(line))
	}
	if len(lines) <= 1 {
		return ""
	}
	return strings.Join(lines, "\n")
}

// Size 三元组数量
func (kg *KnowledgeGraph) Size() int {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	return len(kg.triples)
}

// Restore 从持久化层灌入三元组（保留原 ID/CreatedAt，重建倒排索引）
func (kg *KnowledgeGraph) Restore(triples []Triple) {
	kg.mu.Lock()
	defer kg.mu.Unlock()
	for _, t := range triples {
		idx := len(kg.triples)
		kg.triples = append(kg.triples, t)
		kg.addIdx(t.Subject, idx)
		kg.addIdx(t.Object, idx)
	}
}

// ListAll 返回全部三元组（供持久化全量写回）
func (kg *KnowledgeGraph) ListAll() []Triple {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	out := make([]Triple, len(kg.triples))
	copy(out, kg.triples)
	return out
}

// ─── 子图召回(v4.3b)────────────────────────────────────────────

// GraphNode 子图节点(实体)
type GraphNode struct {
	ID     string  // 节点 ID(实体名;KG 无独立实体 ID)
	Name   string  // 实体名(与 ID 相同)
	Type   string  // 实体类型(KG 无实体类型概念,恒为空串)
	Weight float64 // 节点权重:关联边(去重后)最大权重;无关联边时为 1
}

// GraphEdge 子图边(三元组)
type GraphEdge struct {
	From   string  // 主语实体
	To     string  // 宾语实体
	Type   string  // 关系标签(三元组谓词)
	Weight float64 // 边权重:三元组 confidence;无权重(<=0)时为 1
}

// Subgraph 以 entity 为中心、hops 跳内的邻接子图。
type Subgraph struct {
	Nodes []GraphNode
	Edges []GraphEdge
}

// QuerySubgraph 返回以 entity 为中心、hops 跳内(含 hops=0 时仅自身节点)的邻接子图。
// 去重(节点按 ID、边按 From-To-Type);边只保留两端都在 Nodes 内的;hops<=0 视为 1;
// entity 不存在返回空子图(非 nil,空 slice)。
func (kg *KnowledgeGraph) QuerySubgraph(entity string, hops int) Subgraph {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	out := Subgraph{Nodes: []GraphNode{}, Edges: []GraphEdge{}}
	if hops <= 0 {
		hops = 1
	}

	// 索引按小写实体名组织(与 addIdx 一致)
	key := strings.ToLower(entity)
	idxs, ok := kg.entityIdx[key]
	if !ok {
		// entity 不存在:空子图(非 nil 空 slice)
		return out
	}
	if len(idxs) == 0 {
		// 孤立实体(索引中存在但无任何关联三元组):仅自身节点
		out.Nodes = append(out.Nodes, GraphNode{ID: entity, Name: entity, Weight: 1})
		return out
	}

	// 中心实体名取图谱中实际存储的写法(索引按小写键命中)
	center := ""
	for _, i := range idxs {
		if i < 0 || i >= len(kg.triples) {
			continue
		}
		t := kg.triples[i]
		switch {
		case strings.ToLower(t.Subject) == key:
			center = t.Subject
		case strings.ToLower(t.Object) == key:
			center = t.Object
		}
		if center != "" {
			break
		}
	}
	if center == "" {
		center = entity
	}

	// 节点集合:小写键 → 原始实体名;边集合:From|Type|To → 边(重复取权重最大)
	nodeSet := map[string]string{key: center}
	edgeMap := map[string]GraphEdge{}

	// 广度优先逐跳扩展(无向邻接:主语/宾语任一命中即连通),不做剪枝
	frontier := []string{key}
	for hop := 0; hop < hops && len(frontier) > 0; hop++ {
		var next []string
		for _, cur := range frontier {
			for _, ti := range kg.entityIdx[cur] {
				if ti < 0 || ti >= len(kg.triples) {
					continue
				}
				t := kg.triples[ti]
				ek := t.Subject + "\x00" + t.Predicate + "\x00" + t.Object
				ew := t.Confidence
				if ew <= 0 {
					ew = 1 // 无权重边按 1
				}
				if old, dup := edgeMap[ek]; !dup || old.Weight < ew {
					edgeMap[ek] = GraphEdge{From: t.Subject, To: t.Object, Type: t.Predicate, Weight: ew}
				}
				for _, nm := range []string{t.Subject, t.Object} {
					nk := strings.ToLower(nm)
					if _, exists := nodeSet[nk]; !exists {
						nodeSet[nk] = nm
						next = append(next, nk)
					}
				}
			}
		}
		frontier = next
	}

	// 节点权重:关联边(去重后)最大权重,无关联边为 1
	nodeWeight := map[string]float64{}
	for _, e := range edgeMap {
		for _, nk := range []string{strings.ToLower(e.From), strings.ToLower(e.To)} {
			if e.Weight > nodeWeight[nk] {
				nodeWeight[nk] = e.Weight
			}
		}
	}

	// 组装节点:中心节点在前,其余按名称排序(输出确定性)
	centerW := nodeWeight[key]
	if centerW <= 0 {
		centerW = 1
	}
	out.Nodes = append(out.Nodes, GraphNode{ID: center, Name: center, Weight: centerW})
	rest := make([]string, 0, len(nodeSet)-1)
	for nk, nm := range nodeSet {
		if nk == key {
			continue
		}
		rest = append(rest, nm)
	}
	sort.Strings(rest)
	for _, nm := range rest {
		w := nodeWeight[strings.ToLower(nm)]
		if w <= 0 {
			w = 1
		}
		out.Nodes = append(out.Nodes, GraphNode{ID: nm, Name: nm, Weight: w})
	}

	// 组装边:只保留两端都在节点集合内的,按 From/Type/To 排序
	for _, e := range edgeMap {
		if _, a := nodeSet[strings.ToLower(e.From)]; !a {
			continue
		}
		if _, b := nodeSet[strings.ToLower(e.To)]; !b {
			continue
		}
		out.Edges = append(out.Edges, e)
	}
	sort.Slice(out.Edges, func(i, j int) bool {
		if out.Edges[i].From != out.Edges[j].From {
			return out.Edges[i].From < out.Edges[j].From
		}
		if out.Edges[i].Type != out.Edges[j].Type {
			return out.Edges[i].Type < out.Edges[j].Type
		}
		return out.Edges[i].To < out.Edges[j].To
	})

	return out
}
