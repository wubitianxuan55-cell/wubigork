package app

import (
	"strings"

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

// semanticSearchHits 跨库统一语义检索的共用私有实现（统一入口复用，T5-6）：
// 实现收敛到 gaea_semantic_search.go 的按需/缓存版 semanticSearchHitsOnDemand
// （T7-3：避免每查询先 Ensure 全量扫描；逻辑与 GaeaSemanticSearch 完全一致）。
func (a *App) semanticSearchHits(query string) ([]SemanticHitView, error) {
	return a.semanticSearchHitsOnDemand(query)
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
