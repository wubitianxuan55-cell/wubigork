package app

// gaea_cost_audit.go — 跨项目对标查漏（11.3）绑定面。
//
// 确定性统计对标引擎（语料覆盖率阈值+宁缺勿误闸），结论逐条带溯源引用；
// 复核增强而非替代——只产线索清单，绝不自动改数据。

import "github.com/gaea/gaea/internal/gaea/workcost"

// GaeaCostAuditScan 跨项目对标查漏：bill_gap 针对被审项目（projectID，0=只跑
// 定额组成对标），quota_gap 全语料逐同名组。
func (a *App) GaeaCostAuditScan(projectID int64) (*workcost.AuditReport, error) {
	rep, err := a.hubWorkcostStore().BillAuditScan(projectID)
	if err != nil {
		return nil, err
	}
	if rep.BillGaps == nil {
		rep.BillGaps = []workcost.AuditFinding{}
	}
	if rep.QuotaGaps == nil {
		rep.QuotaGaps = []workcost.AuditFinding{}
	}
	for i := range rep.BillGaps {
		if rep.BillGaps[i].Evidence == nil {
			rep.BillGaps[i].Evidence = []workcost.AuditEvidence{}
		}
	}
	for i := range rep.QuotaGaps {
		if rep.QuotaGaps[i].Evidence == nil {
			rep.QuotaGaps[i].Evidence = []workcost.AuditEvidence{}
		}
	}
	return rep, nil
}
