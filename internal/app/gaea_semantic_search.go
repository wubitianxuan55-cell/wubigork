package app

import ()

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
// T5-6：实现收敛到 gaea_unified_search.go 的 semanticSearchHits（统一入口复用）。
func (a *App) GaeaSemanticSearch(query string) ([]SemanticHitView, error) {
	return a.semanticSearchHits(query)
}
