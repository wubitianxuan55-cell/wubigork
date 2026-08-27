// 测算项目与沉淀闭环（zaojia-database 蒸馏：我的项目/工程量清单/版本留痕 →
// 沉淀回成本库）。绑定门面 CostB（gen_bindings 按 GaeaCost* 前缀自动归入）。
package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/costproject"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/config"
)

// costProjectStoreOverride 测试注入的隔离测算项目存储。
var (
	costProjectStoreOverride    *costproject.Store
	costProjectStoreOverrideSet bool
)

// SetCostProjectStoreForTest 注入隔离存储（测试用）。
func SetCostProjectStoreForTest(s *costproject.Store) {
	costProjectStoreOverride = s
	costProjectStoreOverrideSet = true
}

// ResetCostProjectStoreForTest 清除测试注入。
func ResetCostProjectStoreForTest() { costProjectStoreOverrideSet = false }

// hubCostProjectStore 构造测算项目存储（Hephaestus.db，与成本库同库）。
func (a *App) hubCostProjectStore() *costproject.Store {
	if costProjectStoreOverrideSet {
		return costProjectStoreOverride
	}
	userDir := config.MemoryUserDir()
	if userDir == "" {
		return costproject.Open(nil)
	}
	return costproject.Open(db.GetDatabase(userDir))
}

// GaeaCostProjectSave 新建/更新测算项目，返回项目 id。
func (a *App) GaeaCostProjectSave(p costproject.Project) (string, error) {
	return a.hubCostProjectStore().SaveProject(p)
}

// GaeaCostProjectList 返回测算项目列表（含条目数/合计/版本数）。
func (a *App) GaeaCostProjectList() []costproject.ProjectSummary {
	return a.hubCostProjectStore().ListProjects()
}

// GaeaCostProjectGet 返回单个测算项目（不存在返回 nil）。
func (a *App) GaeaCostProjectGet(id string) *costproject.Project {
	p, err := a.hubCostProjectStore().GetProject(id)
	if err != nil {
		return nil
	}
	return p
}

// GaeaCostProjectDelete 删除项目及其明细/版本（级联）。
func (a *App) GaeaCostProjectDelete(id string) error {
	return a.hubCostProjectStore().DeleteProject(id)
}

// GaeaCostEstimateItemSave 新建/更新测算明细行（金额自动重算）。
func (a *App) GaeaCostEstimateItemSave(i costproject.Item) (int64, error) {
	return a.hubCostProjectStore().SaveItem(i)
}

// GaeaCostEstimateItemDelete 删除测算明细行。
func (a *App) GaeaCostEstimateItemDelete(id int64) error {
	return a.hubCostProjectStore().DeleteItem(id)
}

// GaeaCostEstimateItems 返回项目全部明细行。
func (a *App) GaeaCostEstimateItems(projectID string) []costproject.Item {
	return a.hubCostProjectStore().ListItems(projectID)
}

// GaeaCostEstimateVersionSave 保存不可变版本快照，返回该版本。
func (a *App) GaeaCostEstimateVersionSave(projectID, note string) (*costproject.Version, error) {
	v, err := a.hubCostProjectStore().SaveVersion(projectID, note)
	if err != nil {
		return nil, err
	}
	// 保存版本后项目状态置为「已保存版本」。
	if p, e := a.hubCostProjectStore().GetProject(projectID); e == nil && p != nil {
		p.Status = "已保存版本"
		_, _ = a.hubCostProjectStore().SaveProject(*p)
	}
	return v, nil
}

// GaeaCostEstimateVersions 返回项目版本列表（新→旧）。
func (a *App) GaeaCostEstimateVersions(projectID string) []costproject.Version {
	return a.hubCostProjectStore().ListVersions(projectID)
}

// GaeaCostEstimateSediment 沉淀选中的明细行回成本库（UPSERT cost_entries）：
// 单价/单位/分类/标题按明细写入，来源标注「测算沉淀：{项目名}」；若明细引用了
// 既有成本条目，保留其规格/地区/口径/有效期。返回写入条数。
func (a *App) GaeaCostEstimateSediment(projectID string, itemIDs []int64) (int, error) {
	costStore := a.hubCostStore()
	if !costStore.Available() {
		return 0, fmt.Errorf("成本库不可用")
	}
	p, err := a.hubCostProjectStore().GetProject(projectID)
	if err != nil {
		return 0, fmt.Errorf("测算项目不存在: %w", err)
	}
	want := map[int64]bool{}
	for _, id := range itemIDs {
		want[id] = true
	}
	applied := 0
	for _, item := range a.hubCostProjectStore().ListItems(projectID) {
		if !want[item.ID] {
			continue
		}
		if item.Price <= 0 || strings.TrimSpace(item.Title) == "" {
			continue // 缺单价/无标题行不沉淀（对标「缺单价标记」）
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = cost.SlugName(item.Title)
		}
		entry := cost.Entry{
			Name: name, Title: item.Title,
			Category:     leafCategory(item.CategoryPath),
			CategoryPath: item.CategoryPath,
			Unit:         item.Unit, Price: item.Price, Status: "现行",
			Source: fmt.Sprintf("测算沉淀：%s", p.Name),
			Body:   item.Note,
		}
		if item.EntryName != "" {
			if existing, e := costStore.Get(item.EntryName); e == nil && existing != nil {
				entry.Spec = existing.Spec
				entry.Region = existing.Region
				entry.PriceType = existing.PriceType
				entry.ValidUntil = existing.ValidUntil
				entry.Tags = existing.Tags
				// 人材机二级组成/费率（综合单价子目明细）：沉淀更新时保留。
				entry.Components = existing.Components
				entry.LaborFee, entry.MaterialFee, entry.MachineFee =
					existing.LaborFee, existing.MaterialFee, existing.MachineFee
				entry.ManagementFee, entry.ProfitFee, entry.AdvanceFee, entry.TaxRate =
					existing.ManagementFee, existing.ProfitFee, existing.AdvanceFee, existing.TaxRate
			}
		}
		if err := costStore.Save(entry); err != nil {
			return applied, fmt.Errorf("沉淀 %s 失败: %w", item.Title, err)
		}
		applied++
	}
	if applied > 0 {
		p.Status = "已沉淀"
		p.UpdatedAt = time.Now().UTC()
		_, _ = a.hubCostProjectStore().SaveProject(*p)
	}
	return applied, nil
}

// leafCategory 取分类路径叶子名（与 cost.Store.Save 的 category 语义一致）。
func leafCategory(path string) string {
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.TrimSpace(parts[i]) != "" {
			return strings.TrimSpace(parts[i])
		}
	}
	return "其他"
}
