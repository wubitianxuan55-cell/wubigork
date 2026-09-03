package modelengine

import (
	"math"
	"sync"
)

// ── 自定义引擎用户价目 v1（引擎级统一价：每百万 tokens，币种 CNY）─────────
//
// 欠账：自定义引擎（type=custom，A 刀）的模型不在官方价目目录/内置定价表内，
// stats 的费用折算算不出来（多数恒 0，仅模型名撞上内置前缀时误用官方价）。
// v1 让用户在模型中心给自定义引擎填「输入/输出每百万 tokens 单价」，随引擎
// 配置落盘（engines.json），费用折算优先消费用户价目；无价目时行为与现状
// 完全一致（0=未填=不计价）。
//
// 落点说明：estimatePrice 是包级函数（record/summary/EstimateCostCNY 多处
// 复用，均无 Manager 接收者），故用包级注册表承接——Manager 在引擎配置
// 加载/保存/删除后 SyncUserPrices 全量重建，查价方（estimatePrice）只读。
// 锁序：一律先 m.mu（快照引擎价目）后 customPriceMu（写表）；查价方只持
// customPriceMu，无反向获取，无死锁面。
//
// v1 边界（刻意不做）：不做按模型分别定价（中转站多模型价差按引擎统一价
// 近似，精确化留 v2 模型级 map）；币种固定 CNY（TotalCost 统一人民币口径，
// 免币种选择器；USD 服务商请自行换算后填写）；本地引擎（ollama/herdsman）
// 恒不计价，不消费用户价目。

var (
	customPriceMu    sync.RWMutex
	customPriceTable = map[string]modelPrice{} // engineID → 用户价目（Currency 恒 "CNY"、Unit 恒 ""=每百万 tokens）
)

// userEnginePrice 查用户价目（estimatePrice 的最高优先层）。无条目返回
// false，查价落到既有目录/内置表——无价目引擎行为与现状完全一致。
func userEnginePrice(engineID string) (modelPrice, bool) {
	customPriceMu.RLock()
	defer customPriceMu.RUnlock()
	p, ok := customPriceTable[engineID]
	return p, ok
}

// replaceUserPriceTable 原子替换整表（SyncUserPrices 与 LoadState 的落地点；
// 分出小函数让持 m.mu 写锁的 LoadState 也能在解锁前重建，避免锁重入）。
func replaceUserPriceTable(prices map[string]modelPrice) {
	customPriceMu.Lock()
	customPriceTable = prices
	customPriceMu.Unlock()
}

// sanitizeUserPricePtr 清洗用户价目指针：nil / 非有限（NaN/Inf）/ <=0 一律
// 归 nil。0=未填=不计价（与现状一致）；负数与 NaN/Inf 属手改文件或异常
// 输入，同样不采纳。返回值可直接回写 EngineConfig（omitempty 序列化）。
func sanitizeUserPricePtr(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := *p
	if v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) {
		return &v
	}
	return nil
}

// mergeUserPrice SaveEngine 的价目合并语义（对齐既有「部分更新」口径）：
// cfg 指针 nil = 不修改（返回 existing，地址框/启停等局部保存不会误清除）；
// 非 nil = 清洗后回写（合法正数=设置，<=0/NaN/Inf=清除为 nil）。
func mergeUserPrice(existing, cfg *float64) *float64 {
	if cfg == nil {
		return existing
	}
	return sanitizeUserPricePtr(cfg)
}

// snapshotUserPrices 在「m.mu 已持锁（读或写）」前提下快照各引擎的用户
// 价目表（仅收录清洗后有值的引擎；无价目引擎不进表）。
func (m *Manager) snapshotUserPrices() map[string]modelPrice {
	prices := make(map[string]modelPrice, 4)
	for id, e := range m.engines {
		inP, outP := sanitizeUserPricePtr(e.UserPriceIn), sanitizeUserPricePtr(e.UserPriceOut)
		if inP == nil && outP == nil {
			continue
		}
		p := modelPrice{Currency: "CNY"} // 币种固定 CNY：TotalCost 统一人民币口径，直进总额不做汇率折算
		if inP != nil {
			p.InputPerM = *inP
		}
		if outP != nil {
			p.OutputPerM = *outP
		}
		prices[id] = p
	}
	return prices
}

// SyncUserPrices 用当前引擎配置全量重建用户价目表：有价的写入，无价/已删
// 引擎的旧条目清除。LoadState/SaveEngine/RemoveCustomEngine 修改引擎后调用；
// Add/UpdateCustomEngine 不触及价目字段，无需调用。
func (m *Manager) SyncUserPrices() {
	m.mu.RLock()
	prices := m.snapshotUserPrices()
	m.mu.RUnlock()
	replaceUserPriceTable(prices)
}
