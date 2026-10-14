package app

// ── 数据健康页（四期 4.1/4.2）绑定面 ─────────────────────────
//
// GaeaCostHealthScan：五类扫描（只读，概览页随开随扫）；
// GaeaCostDedupeResources：资源去重/跨身份语义合并（作者显式裁决）。

import "github.com/gaea/gaea/internal/gaea/workcost"

// GaeaCostHealthScan 造价数据健康五类扫描。
func (a *App) GaeaCostHealthScan() (*workcost.HealthReport, error) {
	return a.hubWorkcostStore().HealthScan()
}

// GaeaCostDedupeResources 资源去重：keep 保留、merges 合并（引用重定向到
// keep 的 code，调价历史并入，merged 条目删除）。跨身份合并=作者显式裁决
// （「编码未统一」扫描的处置动作）。
func (a *App) GaeaCostDedupeResources(keepID int64, mergeIDs []int64) (*workcost.DedupeResult, error) {
	return a.hubWorkcostStore().DedupeResources(keepID, mergeIDs)
}
