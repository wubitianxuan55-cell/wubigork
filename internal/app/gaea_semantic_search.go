package app

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// SemanticHitView 是跨库统一语义检索的命中视图。
type SemanticHitView struct {
	Kind  string  `json:"kind"` // cost / knowledge / office / file
	Name  string  `json:"name"`
	Score float64 `json:"score"`
	Text  string  `json:"text"`
}

// SemanticIndexStatus 各库向量索引状态（D3-1：持久化向量索引可见性）。
type SemanticIndexStatus struct {
	Available bool              `json:"available"`
	Counts    map[string]int    `json:"counts"` // kind → 向量条数
	Detail    map[string]string `json:"detail,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// GaeaSemanticIndexStatus 返回向量索引（semantic_vectors）各 kind 的条数，
// 供前端展示「跨库语义检索」索引健康度。
func (a *App) GaeaSemanticIndexStatus() SemanticIndexStatus {
	e := a.localSearchEmbedder()
	if e == nil {
		return SemanticIndexStatus{Available: false, Error: "本地 embedding 未配置（Herdsman bge-m3）"}
	}
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return SemanticIndexStatus{Available: false, Error: "向量索引存储不可用"}
	}
	counts, err := st.Counts()
	if err != nil {
		return SemanticIndexStatus{Available: false, Error: err.Error()}
	}
	return SemanticIndexStatus{Available: true, Counts: counts}
}

// GaeaSemanticSearch 跨库统一语义检索（成本/知识/办公记忆/工作区资料），本地
// bge-m3。向量索引持久化在 Hephaestus.db semantic_vectors；查询路径按需/缓存：
// 索引就绪（向量条数与源文档数一致）时直接 SearchReady（只嵌 query），不再每
// 查询先 Ensure 全量扫描比对；未就绪（首次/数据增删）才按需增量 Ensure。
// 资料（file）依赖文件索引定时维护（每 10 分钟增量重建），检索时不再扫描。
// embedding 不可用/未配置时返回空（不阻断关键词检索）。
func (a *App) GaeaSemanticSearch(query string) ([]SemanticHitView, error) {
	return a.semanticSearchHitsOnDemand(query)
}

// semanticSearchHitsOnDemand 跨库语义检索的按需/缓存实现（T7-3「名实相符」）：
// 避免每查询先 Ensure 全量扫描（vectorDocs 全表比对 + 潜在重嵌）。判断依据为
// kind 的向量条数与源文档数是否一致：
//   - 一致 → 索引已就绪（缓存命中），直接 SearchReady（每 kind 只嵌 query）；
//   - 不一致 → 数据有增删（或首次访问），按需增量 Ensure + Stale 清理后查询。
//
// 内容快照比对（Ensure 的正文变化检测）仍保留——在数据增删时自动重嵌；查询
// 热路径不再承担该扫描成本。file 向量由后台定时任务/文件监听维护，直接查库。
func (a *App) semanticSearchHitsOnDemand(query string) ([]SemanticHitView, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	e := a.localSearchEmbedder()
	if e == nil {
		return nil, nil
	}
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 一次 COUNT 查询拿到各 kind 现有向量条数（廉价信号，替代每查询全量扫描）。
	counts, err := st.Counts()
	if err != nil {
		return nil, err
	}

	// 成本库
	costList := a.hubCostStore().List()
	if len(costList) > 0 {
		docs := make([]semantic.Doc, 0, len(costList))
		keep := make(map[string]bool, len(costList))
		for _, s := range costList {
			docs = append(docs, semantic.Doc{ID: s.Name, Text: retrieval.DocText(s)})
			keep[s.Name] = true
		}
		if counts["cost"] != len(docs) {
			if _, err := st.Ensure(ctx, e, "cost", docs); err != nil {
				return nil, err
			}
			_, _ = st.Stale("cost", keep)
		}
	}

	// 工程知识库
	if ks, err := knowledge.Global().Store(); err == nil {
		entries := ks.ReadAll()
		if len(entries) > 0 {
			docs := make([]semantic.Doc, 0, len(entries))
			keep := make(map[string]bool, len(entries))
			for _, e2 := range entries {
				docs = append(docs, semantic.Doc{
					ID:   e2.Name,
					Text: e2.Title + " " + e2.Category + " " + e2.Body,
				})
				keep[e2.Name] = true
			}
			if counts["knowledge"] != len(docs) {
				if _, err := st.Ensure(ctx, e, "knowledge", docs); err != nil {
					return nil, err
				}
				_, _ = st.Stale("knowledge", keep)
			}
		}
	}

	// 办公记忆 facts
	facts := a.hubOfficeStore().List()
	if len(facts) > 0 {
		docs := make([]semantic.Doc, 0, len(facts))
		keep := make(map[string]bool, len(facts))
		for _, m := range facts {
			docs = append(docs, semantic.Doc{
				ID:   m.Name,
				Text: m.Title + " " + m.Description + " " + m.Body,
			})
			keep[m.Name] = true
		}
		if counts["office"] != len(docs) {
			if _, err := st.Ensure(ctx, e, "office", docs); err != nil {
				return nil, err
			}
			_, _ = st.Stale("office", keep)
		}
	}

	// 工作区资料：文件索引由定时任务维护，docs=nil → 直接 SearchReady 查库。
	const perKind = 6
	var all []SemanticHitView
	for _, kind := range []string{"cost", "knowledge", "office", "file"} {
		hits, err := st.SearchReady(ctx, e, kind, query, perKind)
		if err != nil {
			return nil, err
		}
		for _, h := range hits {
			all = append(all, SemanticHitView{Kind: h.Kind, Name: h.ID, Score: h.Score, Text: h.Text})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Score > all[j].Score })
	if len(all) > 20 {
		all = all[:20]
	}
	return all, nil
}
