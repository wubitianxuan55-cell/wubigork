// 造价参考与复盘笔记（zaojia-database 蒸馏：案例指标 + 复盘经验）。
// 绑定门面 CostB（gen_bindings 按 GaeaCost* 前缀自动归入）。
package app

import (
	"fmt"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/costproject"
	"github.com/gaea/gaea/internal/gaea/costref"
	"github.com/gaea/gaea/internal/gaea/db"
)

// costRefStoreOverride 测试注入的隔离复盘笔记存储。
var (
	costRefStoreOverride    *costref.Store
	costRefStoreOverrideSet bool
)

// SetCostRefStoreForTest 注入隔离存储（测试用）。
func SetCostRefStoreForTest(s *costref.Store) {
	costRefStoreOverride = s
	costRefStoreOverrideSet = true
}

// ResetCostRefStoreForTest 清除测试注入。
func ResetCostRefStoreForTest() { costRefStoreOverrideSet = false }

// hubCostRefStore 构造复盘笔记存储（Hephaestus.db，与成本库同库）。
func (a *App) hubCostRefStore() *costref.Store {
	if costRefStoreOverrideSet {
		return costRefStoreOverride
	}
	userDir := config.MemoryUserDir()
	if userDir == "" {
		return costref.Open(nil)
	}
	return costref.Open(db.GetDatabase(userDir))
}

// GaeaCostIndicators 造价参考指标：对「已保存版本/已沉淀」测算项目的明细行做
// 价格聚合（按科目 title 或一级分类 category），供下次报价对标。
func (a *App) GaeaCostIndicators(group string) []costref.Indicator {
	if group != "category" {
		group = "title"
	}
	projStore := a.hubCostProjectStore()
	var items []costproject.Item
	for _, p := range projStore.ListProjects() {
		// 案例 = 有版本留痕（已保存版本/已沉淀）；临时工作稿不参与对标。
		if p.VersionCount <= 0 {
			continue
		}
		items = append(items, projStore.ListItems(p.ID)...)
	}
	return costref.ComputeIndicators(items, group)
}

// GaeaCostAttribution 归因对标（v4.6.1 补课：审计 §C ④「成本知识图谱+归因
// 对标零实现」）：把指定项目明细逐行与参考指标对标——参考池 = 除本项目外的
// 全部「已保存版本/已沉淀」项目（防自对标），按标题匹配 P25/P75/中位数带宽，
// 产出差幅等级 + 对总偏离的贡献金额 + TopDrivers 主因。无参考样本的条目
// 保留展示但不参与归因（不猜测）。
func (a *App) GaeaCostAttribution(projectID string) (costref.Attribution, error) {
	projStore := a.hubCostProjectStore()
	p, err := projStore.GetProject(projectID)
	if err != nil {
		return costref.Attribution{}, fmt.Errorf("项目不存在：%s", projectID)
	}
	var refItems []costproject.Item
	for _, rp := range projStore.ListProjects() {
		if rp.ID == projectID || rp.VersionCount <= 0 {
			continue // 排除本项目 + 无版本留痕的临时工作稿
		}
		refItems = append(refItems, projStore.ListItems(rp.ID)...)
	}
	indicators := costref.ComputeIndicators(refItems, "title")
	return costref.ComputeAttribution(projectID, p.Name, projStore.ListItems(projectID), indicators), nil
}

// GaeaCostNoteSave 新建/更新复盘笔记。
func (a *App) GaeaCostNoteSave(n costref.Note) (int64, error) {
	return a.hubCostRefStore().Save(n)
}

// GaeaCostNoteList 复盘笔记列表（关键词/状态过滤）。
func (a *App) GaeaCostNoteList(query, status string) []costref.Note {
	return a.hubCostRefStore().List(query, status)
}

// GaeaCostNoteDelete 删除复盘笔记。
func (a *App) GaeaCostNoteDelete(id int64) error {
	return a.hubCostRefStore().Delete(id)
}

// GaeaCostNoteBumpRef 引用次数 +1（agent/前端引用笔记时记录复用）。
func (a *App) GaeaCostNoteBumpRef(id int64) error {
	return a.hubCostRefStore().BumpRef(id)
}
