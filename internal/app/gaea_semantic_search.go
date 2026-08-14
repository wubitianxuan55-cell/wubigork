package app

import (
	"context"
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
// bge-m3。向量索引持久化在 Hephaestus.db semantic_vectors，Ensure 增量向量化、
// 查询只嵌 query；embedding 不可用/未配置时返回空（不阻断关键词检索）。
// 资料（file）依赖文件索引定时维护（每 10 分钟增量重建），检索时不再扫描。
func (a *App) GaeaSemanticSearch(query string) ([]SemanticHitView, error) {
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

	kindDocs := map[string][]semantic.Doc{}

	// 成本库
	costList := a.hubCostStore().List()
	if len(costList) > 0 {
		docs := make([]semantic.Doc, 0, len(costList))
		keep := make(map[string]bool, len(costList))
		for _, s := range costList {
			docs = append(docs, semantic.Doc{ID: s.Name, Text: retrieval.DocText(s)})
			keep[s.Name] = true
		}
		_, _ = st.Stale("cost", keep)
		kindDocs["cost"] = docs
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
			_, _ = st.Stale("knowledge", keep)
			kindDocs["knowledge"] = docs
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
		_, _ = st.Stale("office", keep)
		kindDocs["office"] = docs
	}

	// 工作区资料：文件索引由 startFileIndexCron 维护（增量重建），检索直接
	// 用已持久化向量（docs=nil → Ensure 跳过，SearchReady 查库）。
	kindDocs["file"] = nil

	hits, err := st.SearchMany(ctx, e, kindDocs, query, 6, 20)
	if err != nil {
		return nil, err
	}
	out := make([]SemanticHitView, 0, len(hits))
	for _, h := range hits {
		out = append(out, SemanticHitView{Kind: h.Kind, Name: h.ID, Score: h.Score, Text: h.Text})
	}
	return out, nil
}
