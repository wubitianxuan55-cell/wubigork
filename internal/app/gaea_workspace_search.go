package app

import ()

// WorkspaceSearchHit 是工作区全文搜索的一条命中（轻量 RAG）。
type WorkspaceSearchHit struct {
	Path    string  `json:"path"`    // 工作区相对路径（/ 分隔）
	Name    string  `json:"name"`    // 文件名
	Size    int64   `json:"size"`    // 字节数
	ModTime int64   `json:"modTime"` // unix 毫秒
	Score   float64 `json:"score"`   // 相关度
	Snippet string  `json:"snippet"` // 命中片段
	// Truncated 标记正文被 20 万字符上限截断（索引不全）。
	Truncated bool `json:"truncated"`
	// Skipped 非空表示文件未索引（如 >5MB），展示可见原因。
	Skipped string `json:"skipped,omitempty"`
}

// GaeaWorkspaceSearch 工作区全文搜索（P1-① 轻量 RAG）：在 docx/xlsx/pdf/md/txt/csv
// 等文件正文里定位关键词，返回命中片段供预览/一键 @ 引用。
// 对标 Cursor 本地索引 / Cherry Studio 知识库检索的「关键词先行」版本。
// T5-6：实现收敛到 gaea_unified_search.go 的 workspaceSearchHits（统一入口复用）。
func (a *App) GaeaWorkspaceSearch(query string, limit int) []WorkspaceSearchHit {
	return a.workspaceSearchHits(query, limit)
}
