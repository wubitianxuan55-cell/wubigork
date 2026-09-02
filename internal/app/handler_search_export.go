package app

import (
	"fmt"

	"github.com/gaea/gaea/internal/export"
	"github.com/gaea/gaea/internal/search"
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

// ExportAll 一键导出全部格式。
// onlyMainline：true 时仅导出主线章节（跳过 NNN[a-z].md 分支文件），false 为
// 历史默认行为（分支章节一并导出）。方法名不变（绑定面零变更）。
func (a *App) ExportAll(onlyMainline bool) (map[string]string, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	em := export.New(pm)
	if onlyMainline {
		em.SetMainlineOnly(true)
	}
	return em.ExportAll()
}
