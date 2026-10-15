package app

// ── 企业定额自生成（11.6）绑定面：Preview→Apply 双段式 ──
//
// Preview=跨项目含量分歧聚类预览（只读，宁缺勿误不成立簇带人可读原因）；
// Apply=作者显式点名聚类词干后落 ENT-* 企业定额。传输面只有词干列表——
// 候选重建走同一纯函数，Preview 与 Apply 天然一致，前端不搬运整表。

import "github.com/gaea/gaea/internal/gaea/workcost"

// GaeaCostQuotaFlywheelPreview 扫描跨项目定额含量分歧，提出企业定额候选。
func (a *App) GaeaCostQuotaFlywheelPreview() (workcost.QuotaFlywheelPreview, error) {
	prev, err := a.hubWorkcostStore().QuotaFlywheelPreview()
	if err != nil {
		return workcost.QuotaFlywheelPreview{}, err
	}
	if prev.Candidates == nil {
		prev.Candidates = []workcost.QuotaFlywheelCandidate{}
	}
	if prev.Skipped == nil {
		prev.Skipped = []workcost.QuotaFlywheelSkip{}
	}
	for i := range prev.Candidates {
		if prev.Candidates[i].Members == nil {
			prev.Candidates[i].Members = []workcost.QuotaFlywheelMember{}
		}
		if prev.Candidates[i].Rows == nil {
			prev.Candidates[i].Rows = []workcost.QuotaFlywheelRow{}
		}
	}
	return prev, nil
}

// GaeaCostQuotaFlywheelApply 作者确认后生成企业定额（ENT-<原码词干>，整组
// 替换=重生成即飞轮进化；只覆写自有命名空间）。
func (a *App) GaeaCostQuotaFlywheelApply(origCodes []string) (*workcost.QuotaFlywheelResult, error) {
	return a.hubWorkcostStore().QuotaFlywheelApply(origCodes)
}
