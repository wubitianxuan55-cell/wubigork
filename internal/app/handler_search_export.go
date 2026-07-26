package app

import (
	"fmt"

	"github.com/wubigork/wubigork/internal/export"
	"github.com/wubigork/wubigork/internal/search"
)

// ── 搜索 ────────────────────────────────────────────────────

// Search 全局搜索
func (a *App) Search(query string) (map[string][]search.Result, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	return search.New(pm).SearchAll(query)
}

// ── 导出 ────────────────────────────────────────────────────

// ExportAll 一键导出全部格式
func (a *App) ExportAll() (map[string]string, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	return export.New(pm).ExportAll()
}
