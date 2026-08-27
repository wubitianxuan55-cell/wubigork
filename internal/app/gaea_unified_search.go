package app

import (
	"strings"

	"github.com/gaea/gaea/internal/gaea/wssearch"
)

// UnifiedSearchView 是统一检索入口（T5-6）的返回视图：一次搜索同时出
// 「关键词全文 + 语义跨库」两组结果，前端一个搜索框两段展示。
// 记忆统一层第一刀扩展：新增 Brain（三脑命中）与 Files（文件语义命中）
// 两组——hub 搜索由「4 绑定前端拼装」收敛为「1 绑定后端聚合」。
type UnifiedSearchView struct {
	// Keyword 是工作区关键词全文命中（轻量 RAG，与 WorkspaceSearchHit 一致）。
	Keyword []WorkspaceSearchHit `json:"keyword"`
	// Semantic 是跨库语义命中（cost/knowledge/office/file）；embedding 不可用
	// 时为空数组而 Keyword 照常（与现有降级行为一致）。
	Semantic []SemanticHitView `json:"semantic"`
	// Brain 是三脑命中（brain.main/left/right）；三脑未装配（a.brain==nil）时
	// 为空数组，不报错。
	Brain []Hit `json:"brain"`
	// Files 是工作区文件语义命中（path/score/snippet）；embedding 不可用时为空数组。
	Files []FileSemanticHit `json:"files"`
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

// brainSearchHits 三脑统一检索的共用私有实现：三脑未装配（a.brain==nil）
// 时返回空数组而不报错（hub 搜索降级为 keyword/semantic/files 三组照常）。
func (a *App) brainSearchHits(query string) []Hit {
	if a.brain == nil {
		return []Hit{}
	}
	hits, err := a.brain.Search(query)
	if err != nil || len(hits) == 0 {
		return []Hit{}
	}
	return hits
}

// GaeaUnifiedSearch 统一检索入口（T5-6）：一个搜索框同时出「关键词全文 +
// 语义跨库」两组结果。记忆统一层第一刀扩展为四组：keyword（工作区全文）+
// semantic（跨库语义）+ brain（三脑命中）+ files（文件语义命中）。
// 内部串行调用现有各域检索的共用私有实现；空 query 返回空视图；
// embedding 不可用时 semantic/files 为空数组而 keyword/brain 照常。
func (a *App) GaeaUnifiedSearch(query string, topN int) (UnifiedSearchView, error) {
	view := UnifiedSearchView{
		Keyword:  []WorkspaceSearchHit{},
		Semantic: []SemanticHitView{},
		Brain:    []Hit{},
		Files:    []FileSemanticHit{},
	}
	if strings.TrimSpace(query) == "" {
		return view, nil
	}
	view.Keyword = a.workspaceSearchHits(query, topN)
	view.Brain = a.brainSearchHits(query)
	if topN <= 0 {
		topN = 10
	}
	files, err := a.fileSemanticHits(query, topN)
	if err != nil {
		return view, err
	}
	if files != nil {
		view.Files = files
	}
	sem, err := a.semanticSearchHits(query)
	if err != nil {
		return view, err
	}
	if sem != nil {
		view.Semantic = sem
	}
	return view, nil
}
