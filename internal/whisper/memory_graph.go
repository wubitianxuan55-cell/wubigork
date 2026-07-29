// Package whisper — memory_graph.go
// 轻量知识图谱（从旧 memory.go 提取，对齐 ackem memory/knowledgeGraph.ts）

package whisper

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

// ─── Triple ───────────────────────────────────────────────────

// Triple 知识图谱三元组
type Triple struct {
	ID            string   `json:"id"`
	Subject       string   `json:"subject"`
	Predicate     string   `json:"predicate"`
	Object        string   `json:"object"`
	Confidence    float64  `json:"confidence"`
	SourceFactIDs []string `json:"sourceFactIds"`
	CreatedAt     time.Time `json:"createdAt"`
}

// ─── KnowledgeGraph ───────────────────────────────────────────

// KnowledgeGraph 轻量知识图谱（实体倒排索引）
type KnowledgeGraph struct {
	triples   []Triple
	entityIdx map[string][]int // entity → triple indices
}

// NewKnowledgeGraph 创建空图谱
func NewKnowledgeGraph() *KnowledgeGraph {
	return &KnowledgeGraph{entityIdx: make(map[string][]int)}
}

// Add 添加三元组
func (kg *KnowledgeGraph) Add(subj, pred, obj string, conf float64, src []string) Triple {
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
	ql := strings.ToLower(text)
	type scored struct {
		t Triple
		s float64
	}
	var rs []scored
	for _, t := range kg.triples {
		tl := strings.ToLower(t.Subject + " " + t.Predicate + " " + t.Object)
		s := 0.0
		for e := range kg.entityIdx {
			if strings.Contains(ql, e) {
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
	return len(kg.triples)
}
