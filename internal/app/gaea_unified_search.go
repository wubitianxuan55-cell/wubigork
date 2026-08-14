package app

import (
	"context"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
	"github.com/gaea/gaea/internal/gaea/wssearch"
)

// UnifiedSearchView 是统一检索入口（T5-6）的返回视图：一次搜索同时出
// 「关键词全文 + 语义跨库」两组结果，前端一个搜索框两段展示。
type UnifiedSearchView struct {
	// Keyword 是工作区关键词全文命中（轻量 RAG，与 WorkspaceSearchHit 一致）。
	Keyword []WorkspaceSearchHit `json:"keyword"`
	// Semantic 是跨库语义命中（cost/knowledge/office/file）；embedding 不可用
	// 时为空数组而 Keyword 照常（与现有降级行为一致）。
	Semantic []SemanticHitView `json:"semantic"`
}

// workspaceSearchHits 工作区关键词全文检索的共用私有实现：逻辑与
// GaeaWorkspaceSearch 完全一致（抽取自其方法体，统一入口不重复实现）。
func (a *App) workspaceSearchHits(query string, topN int) []WorkspaceSearchHit {
	hits := wssearch.Search(gaeaCwd(), query, topN)
	if len(hits) == 0 {
		return []WorkspaceSearchHit{}
	}
	out := make([]WorkspaceSearchHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, WorkspaceSearchHit{
			Path:      h.Path,
			Name:      h.Name,
			Size:      h.Size,
			ModTime:   h.ModTime,
			Score:     h.Score,
			Snippet:   h.Snippet,
			Truncated: h.Truncated,
			Skipped:   h.Skipped,
		})
	}
	return out
}

// semanticSearchHits 跨库统一语义检索的共用私有实现：逻辑与 GaeaSemanticSearch
// 完全一致（抽取自其方法体）。向量索引持久化在 Hephaestus.db semantic_vectors，
// Ensure 增量向量化、查询只嵌 query；embedding 不可用/未配置时返回空
// （不阻断关键词检索）。
func (a *App) semanticSearchHits(query string) ([]SemanticHitView, error) {
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

// GaeaUnifiedSearch 统一检索入口（T5-6）：一个搜索框同时出「关键词 + 语义跨库」
// 两组结果。内部串行调用现有关键词全文检索与跨库语义检索的共用私有实现；
// 空 query 返回空视图；embedding 不可用时 semantic 为空数组而 keyword 照常。
func (a *App) GaeaUnifiedSearch(query string, topN int) (UnifiedSearchView, error) {
	view := UnifiedSearchView{Keyword: []WorkspaceSearchHit{}, Semantic: []SemanticHitView{}}
	if strings.TrimSpace(query) == "" {
		return view, nil
	}
	view.Keyword = a.workspaceSearchHits(query, topN)
	sem, err := a.semanticSearchHits(query)
	if err != nil {
		return view, err
	}
	if sem != nil {
		view.Semantic = sem
	}
	return view, nil
}
