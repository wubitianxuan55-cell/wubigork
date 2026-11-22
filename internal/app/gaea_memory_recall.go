package app

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// ── 星云召回：记忆中枢四臂融合检索（Nebula Memory 蒸馏 P1）────────────────
//
// 独立于 UnifiedSearch（共享搜索面不动）与注入路径（boot 缓存稳定红线），
// 只服务记忆中枢面板：词法（BM25）+ 语义（bge-m3 向量）+ 图谱（事件图
// references 邻居扩散）+ 置顶 四臂融合，命中带臂归因（可解释 wire 面）。
// embedding/图库不可用时对应臂如实降级（Degraded 里留说明），不假装命中。

// MemoryRecallHit 单条融合召回命中。
type MemoryRecallHit struct {
	Name    string   `json:"name"`
	Title   string   `json:"title"`
	Kind    string   `json:"kind"`
	Type    string   `json:"type"`
	Space   string   `json:"space"`
	Preview string   `json:"preview"`
	Score   float64  `json:"score"` // 融合分 [0,1]
	Arms    []string `json:"arms"`  // 命中臂归因（lexical/semantic/graph/pinned）
	Pinned  bool     `json:"pinned"`
}

// MemoryRecallView 星云召回视图。
type MemoryRecallView struct {
	Query string            `json:"query"`
	Total int               `json:"total"`
	Hits  []MemoryRecallHit `json:"hits"`
	// Degraded 降级臂说明（臂名 → 原因）：诚实暴露「哪条臂这次没干活」。
	Degraded map[string]string `json:"degraded,omitempty"`
}

// recallSeedCount 图谱臂种子数（词法/语义 top 合并后取前 N）。
const recallSeedCount = 5

// GaeaMemoryRecall 记忆中枢四臂融合召回（办公记忆库）。
func (a *App) GaeaMemoryRecall(query string, limit int) MemoryRecallView {
	view := MemoryRecallView{Query: query, Hits: []MemoryRecallHit{}, Degraded: map[string]string{}}
	q := strings.TrimSpace(query)
	if q == "" {
		return view
	}
	if limit <= 0 {
		limit = 8
	}
	if limit > 50 {
		limit = 50
	}

	ms := a.hubOfficeStore()
	mems := ms.List()
	byName := make(map[string]memory.Memory, len(mems))
	for _, m := range mems {
		byName[m.Name] = m
	}

	// ── 词法臂：BM25（memory_search 同一索引实现，nil docs=只索引记忆本体）。
	lexHits := map[string]float64{}
	maxLex := 0.0
	if idx := ms.BuildSearchIndex(nil); idx != nil {
		for _, s := range idx.Search(q) {
			if _, live := byName[s.Name]; !live {
				continue // doc:/外部条目不入融合面
			}
			if s.Score > maxLex {
				maxLex = s.Score
			}
			lexHits[s.Name] = s.Score
		}
	}
	if maxLex > 0 {
		for name, sc := range lexHits {
			lexHits[name] = sc / maxLex
		}
	}

	// ── 语义臂：office 向量（与跨库语义检索同一 Ensure/缓存口径，office 单库）。
	semHits := map[string]float64{}
	if e := a.localSearchEmbedder(); e == nil {
		view.Degraded[memory.RecallArmSemantic] = "本地 embedding 未配置（Herdsman bge-m3）"
	} else if st := a.hubSemanticStore(); st == nil || !st.Available() {
		view.Degraded[memory.RecallArmSemantic] = "向量索引存储不可用"
	} else {
		hits, err := a.recallOfficeVecHits(e, st, q, mems, limit)
		if err != nil {
			slog.Warn("星云召回语义臂失败（该臂降级跳过）", "error", err)
			view.Degraded[memory.RecallArmSemantic] = err.Error()
		} else {
			for _, h := range hits {
				if _, live := byName[h.ID]; live {
					semHits[h.ID] = clampUnit(h.Score)
				}
			}
		}
	}

	// ── 图谱臂：事件图 references 邻居 1 跳扩散（种子=词法/语义 top）。
	graphHits := map[string]float64{}
	if gdb := db.GetDatabase(config.MemoryUserDir()); gdb != nil {
		if _, err := memory.EnsureGraphFresh(gdb); err != nil {
			view.Degraded[memory.RecallArmGraph] = err.Error()
		} else if g, err := memory.MaterializedGraph(gdb); err != nil {
			view.Degraded[memory.RecallArmGraph] = err.Error()
		} else {
			graphHits = memory.ExpandGraphSeeds(recallTopSeeds(lexHits, semHits, recallSeedCount), recallEntityAdjacency(g))
		}
	} else {
		view.Degraded[memory.RecallArmGraph] = "语义图库不可用"
	}

	// ── 融合输入：任一臂命中或置顶的记忆参与；失效条目排除（注入面同口径）。
	inputs := make([]memory.RecallFuseInput, 0, len(lexHits)+len(semHits)+len(graphHits))
	for name, m := range byName {
		lex, sem, graph := lexHits[name], semHits[name], graphHits[name]
		if lex == 0 && sem == 0 && graph == 0 && !m.Pinned {
			continue
		}
		if !m.InvalidAt.IsZero() {
			continue
		}
		inputs = append(inputs, memory.RecallFuseInput{Name: name, Lexical: lex, Semantic: sem, Graph: graph, Pinned: m.Pinned})
	}
	fused := memory.FuseRecallArms(inputs)
	if len(fused) > limit {
		fused = fused[:limit]
	}
	view.Total = len(fused)

	for _, h := range fused {
		m, ok := byName[h.Name]
		if !ok {
			continue
		}
		preview := m.Description
		if preview == "" {
			preview = oneLineSummary(m.Body)
		}
		arms := h.Arms
		if arms == nil {
			arms = []string{}
		}
		view.Hits = append(view.Hits, MemoryRecallHit{
			Name:    m.Name,
			Title:   m.Title,
			Kind:    string(m.Kind),
			Type:    string(m.Type),
			Space:   m.Space,
			Preview: preview,
			Score:   h.Final,
			Arms:    arms,
			Pinned:  m.Pinned,
		})
	}
	return view
}

// recallOfficeVecHits 语义臂装配：office 单库 Ensure（缓存口径：向量条数与
// 源文档数一致时直接 SearchReady，只嵌 query）+ 查询。
func (a *App) recallOfficeVecHits(e *retrieval.Embedder, st *semantic.Store, q string, mems []memory.Memory, topN int) ([]semantic.Hit, error) {
	docs := make([]semantic.Doc, 0, len(mems))
	for _, m := range mems {
		docs = append(docs, semantic.Doc{ID: m.Name, Text: m.Title + " " + m.Description + " " + m.Body})
	}
	counts, err := st.Counts()
	if err != nil {
		return nil, err
	}
	if len(docs) > 0 && counts["office"] != len(docs) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if _, err := st.Ensure(ctx, e, "office", docs); err != nil {
			return nil, err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return st.SearchReady(ctx, e, "office", q, topN)
}

// recallTopSeeds 词法/语义命中合并取前 N（同条取两臂最高分；确定性排序）。
func recallTopSeeds(lex, sem map[string]float64, n int) []memory.RecallArmHit {
	names := make(map[string]bool, len(lex)+len(sem))
	for k := range lex {
		names[k] = true
	}
	for k := range sem {
		names[k] = true
	}
	hits := make([]memory.RecallArmHit, 0, len(names))
	for name := range names {
		score := lex[name]
		if sem[name] > score {
			score = sem[name]
		}
		hits = append(hits, memory.RecallArmHit{Name: name, Score: score})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Name < hits[j].Name
	})
	if len(hits) > n {
		hits = hits[:n]
	}
	return hits
}

// recallEntityAdjacency 从物化语义图抽取实体级无向邻接表（键=memory slug）。
// 实体节点的 Name 字段是展示标题（OpSave 后被 Title 覆盖），不是 slug——
// slug 从节点 id（mem:<project>/<name>）剥前缀得到。
func recallEntityAdjacency(g *memory.Graph) map[string][]string {
	slugOf := make(map[string]string)
	for _, n := range g.Nodes {
		if n.NType == memory.NodeEntity {
			slugOf[n.ID] = strings.TrimPrefix(n.ID, "mem:"+n.Project+"/")
		}
	}
	adj := map[string][]string{}
	seen := map[string]bool{}
	add := func(a, b string) {
		if a == b || a == "" || b == "" {
			return
		}
		key := a + "\x00" + b
		if seen[key] {
			return
		}
		seen[key] = true
		adj[a] = append(adj[a], b)
		adj[b] = append(adj[b], a)
	}
	for _, e := range g.Edges {
		if e.EType != memory.EdgeReferences {
			continue // produces/affects 连接 event/source 节点，不入实体邻接
		}
		add(slugOf[e.Src], slugOf[e.Tgt])
	}
	return adj
}

// clampUnit 把任意分数夹进 [0,1]。
func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
