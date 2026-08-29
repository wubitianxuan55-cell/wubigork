package app

import (
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/costinquiry"
	"github.com/gaea/gaea/internal/gaea/db"
)

// costInquiryStoreOverride 测试注入的隔离询价库存储;nil 时使用真实用户库。
var costInquiryStoreOverride *costinquiry.Store
var costInquiryStoreOverrideSet bool

// SetCostInquiryStoreForTest 注入隔离的询价库存储（测试用）。
func SetCostInquiryStoreForTest(s *costinquiry.Store) {
	costInquiryStoreOverride = s
	costInquiryStoreOverrideSet = true
}

// ResetCostInquiryStoreForTest 清除测试注入。
func ResetCostInquiryStoreForTest() { costInquiryStoreOverrideSet = false }

// hubCostInquiryStore 构造询价库存储（Hephaestus.db，与成本库同库）。
func (a *App) hubCostInquiryStore() *costinquiry.Store {
	if costInquiryStoreOverrideSet {
		return costInquiryStoreOverride
	}
	userDir := config.MemoryUserDir()
	if userDir == "" {
		return costinquiry.Open(nil)
	}
	return costinquiry.Open(db.GetDatabase(userDir))
}

// GaeaCostInquirySave 新建/更新询价数据点（id<=0 新建）。
func (a *App) GaeaCostInquirySave(r costinquiry.Record) (int64, error) {
	return a.hubCostInquiryStore().Save(r)
}

// GaeaCostInquiryList 询价数据点列表（关键词/limit 过滤）。
func (a *App) GaeaCostInquiryList(query string, limit int) []costinquiry.Record {
	return a.hubCostInquiryStore().List(query, limit)
}

// GaeaCostInquiryDelete 删除询价数据点。
func (a *App) GaeaCostInquiryDelete(id int64) error {
	return a.hubCostInquiryStore().Delete(id)
}

// GaeaCostInquiryExpiring 到期预警：valid_until 非空且 <= today+days 的数据点。
func (a *App) GaeaCostInquiryExpiring(days int) []costinquiry.Record {
	return a.hubCostInquiryStore().ListExpiring(days)
}

// GaeaCostInquiryAdjust 调差建议：成本库条目 vs 最新询价数据点（差幅 > 2%）。
func (a *App) GaeaCostInquiryAdjust() []costinquiry.AdjustSuggestion {
	store := a.hubCostStore()
	if !store.Available() {
		return nil
	}
	return a.hubCostInquiryStore().SuggestAdjustments(store.List())
}
