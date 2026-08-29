package app

import (
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/coststage"
	"github.com/gaea/gaea/internal/gaea/db"
)

// costStageStoreOverride 测试注入的隔离五算存储;nil 时使用真实用户库。
var costStageStoreOverride *coststage.Store
var costStageStoreOverrideSet bool

// SetCostStageStoreForTest 注入隔离的五算存储（测试用）。
func SetCostStageStoreForTest(s *coststage.Store) {
	costStageStoreOverride = s
	costStageStoreOverrideSet = true
}

// ResetCostStageStoreForTest 清除测试注入。
func ResetCostStageStoreForTest() { costStageStoreOverrideSet = false }

// hubCostStageStore 构造五算存储（Hephaestus.db，与成本库同库）。
func (a *App) hubCostStageStore() *coststage.Store {
	if costStageStoreOverrideSet {
		return costStageStoreOverride
	}
	userDir := config.MemoryUserDir()
	if userDir == "" {
		return coststage.Open(nil)
	}
	return coststage.Open(db.GetDatabase(userDir))
}

// GaeaCostStageSave 保存五算阶段值（(project_id, stage) UPSERT）。
func (a *App) GaeaCostStageSave(v coststage.StageValue) error {
	return a.hubCostStageStore().SaveStage(v)
}

// GaeaCostStages 返回项目五算阶段值（按 估/概/预/结/决 顺序）。
func (a *App) GaeaCostStages(projectID string) []coststage.StageValue {
	return a.hubCostStageStore().ListStages(projectID)
}

// GaeaCostStageCompare 五算对比表（固定 5 行，缺阶段标记无值）。
func (a *App) GaeaCostStageCompare(projectID string) []coststage.CompareRow {
	return coststage.ComputeComparison(a.hubCostStageStore().ListStages(projectID))
}

// GaeaCostStageDeviations 五算相邻阶段偏差特征（正常/关注/异常三档）。
func (a *App) GaeaCostStageDeviations(projectID string) []coststage.Deviation {
	return coststage.ExtractDeviations(coststage.ComputeComparison(a.hubCostStageStore().ListStages(projectID)))
}
