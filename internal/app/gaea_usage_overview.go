package app

// 分流统计（D3-2）：打通 gaea 侧调用记录（含费用估算）与 Herdsman
// events.jsonl 本地遥测，输出「本地 vs 云端」分流视图与节省对比。
// 口径：云端 = 经 gaea 调用云端引擎（xai/deepseek/opencode-*）的用量；
// 本地 = Herdsman events.jsonl 全量（含 herdsman UI 直接调用）+ gaea 侧
// 本地引擎（ollama/cosyvoice 等）用量；节省 = 本地调用若按用户云端实际
// 混合单价折算需花费的金额（本地实际费用 ≈ 0）。

import (
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/modelengine"
)

// UsageSide 单侧（云端/本地）用量聚合。
type UsageSide struct {
	Calls           int64    `json:"calls"`
	SuccessCalls    int64    `json:"success_calls"`
	FailCalls       int64    `json:"fail_calls"`
	InputTokens     int64    `json:"input_tokens"`
	OutputTokens    int64    `json:"output_tokens"`
	TotalTokens     int64    `json:"total_tokens"`
	CacheHitTokens  int64    `json:"cache_hit_tokens"`  // KV 缓存命中 prompt token（gaea 侧记录）
	CacheMissTokens int64    `json:"cache_miss_tokens"` // KV 缓存未命中 prompt token（gaea 侧记录）
	TotalDurationMs int64    `json:"total_duration_ms"`
	Cost            float64  `json:"cost"` // 估算费用（CNY）；本地恒为 0
	Engines         []string `json:"engines"`
}

// SavingsView 节省对比。
type SavingsView struct {
	// RefPricePerMTok 参考云端单价（¥/百万 token）：用户云端实际混合单价
	// （云端总费用 / 云端总 token），云端无用量时回退 deepseek-v4-flash 官价。
	RefPricePerMTok float64 `json:"ref_price_per_mtok"`
	// WouldCostCloud 本地调用若按参考单价走云端需花费（CNY）。
	WouldCostCloud float64 `json:"would_cost_cloud"`
	// Saved 节省金额（CNY，≈ WouldCostCloud，本地实际费用为 0）。
	Saved float64 `json:"saved"`
	// Note 口径说明（中文，供前端展示）。
	Note string `json:"note"`
}

// UsageOverview 分流统计总览（Wails 绑定）。
type UsageOverview struct {
	Cloud   UsageSide   `json:"cloud"`
	Local   UsageSide   `json:"local"`
	Savings SavingsView `json:"savings"`
	// KV 缓存命中率（全局口径，数据来自 gaea 侧调用记录；events.jsonl 无缓存字段）。
	CacheHitTokens  int64   `json:"cache_hit_tokens"`  // 全局缓存命中 prompt token
	CacheMissTokens int64   `json:"cache_miss_tokens"` // 全局缓存未命中 prompt token
	CacheHitRate    float64 `json:"cache_hit_rate"`    // hit/(hit+miss)；无数据时 0
}

// cloudEngineSet 云端引擎（费用估算表有价；herdsman/ollama/cosyvoice 为本地）。
var cloudEngineSet = map[string]bool{
	"xai": true, "deepseek": true, "opencode-go": true, "opencode-zen": true,
}

// GaeaUsageOverview 汇总分流统计（模型中心「本地 vs 云端」面板）。
func (a *App) GaeaUsageOverview() UsageOverview {
	hm, _ := a.HerdsmanModelStats() // events 不可用时走 gaea 侧本地记录
	return buildUsageOverview(a.GetModelCallStats(), hm)
}

// buildUsageOverview 纯聚合（可注入测试）：gaeaStats 为 gaea 侧调用记录，
// hm 为 Herdsman events.jsonl 聚合（PerModel 为空视为不可用）。
func buildUsageOverview(gaeaStats modelengine.ModelStatsSummary, hm HerdsmanModelStats) UsageOverview {
	out := UsageOverview{}

	// 1. gaea 侧调用记录（云端 + 本地引擎）。
	// 云端/本地按引擎分类；herdsman 引擎的调用与 token 由 events.jsonl 全量
	// 覆盖（含 herdsman UI 直接调用），gaea 侧 herdsman 记录只在 events
	// 不可用时作为兜底。
	var cloudTokens int64
	var herdGaea, nonHerdGaea usageSideCounters
	seenCloud, seenLocal := map[string]bool{}, map[string]bool{}
	for _, pm := range gaeaStats.PerModel {
		switch {
		case cloudEngineSet[pm.EngineID]:
			out.Cloud.add(fromPM(pm))
			cloudTokens += pm.TotalTokens
			seenCloud[pm.EngineID] = true
		case pm.EngineID == "herdsman":
			herdGaea.add(pm)
			seenLocal[pm.EngineID] = true
		default:
			nonHerdGaea.add(pm)
			seenLocal[pm.EngineID] = true
		}
	}
	out.Cloud.Cost = gaeaStats.TotalCost // 全部为云端引擎估算费用（本地引擎不计价）
	out.Cloud.Engines = sortedKeys(seenCloud)
	out.Local.Engines = sortedKeys(seenLocal)

	// 2. Herdsman events.jsonl 本地全量（含 herdsman UI 直接调用）。
	if len(hm.PerModel) > 0 {
		for _, m := range hm.PerModel {
			out.Local.Calls += int64(m.Calls)
			out.Local.SuccessCalls += int64(m.Succeeded)
			out.Local.FailCalls += int64(m.Failed)
			out.Local.InputTokens += m.InputTokens
			out.Local.OutputTokens += m.OutputTokens
			out.Local.TotalTokens += m.InputTokens + m.OutputTokens
			out.Local.TotalDurationMs += m.TotalDurationMs
		}
		// 其他本地引擎（ollama/cosyvoice 等）在 events.jsonl 之外。
		out.Local.add(nonHerdGaea)
	} else {
		// events.jsonl 不可用：退化到 gaea 侧本地引擎记录（herdsman + 其他）。
		out.Local.add(nonHerdGaea)
		out.Local.add(herdGaea)
	}

	// 3. KV 缓存命中率：全局 = 云端 + 本地中 gaea 侧记录的缓存字段
	// （events.jsonl 无缓存字段，贡献为 0；分母为 0 时命中率取 0）。
	out.CacheHitTokens = out.Cloud.CacheHitTokens + out.Local.CacheHitTokens
	out.CacheMissTokens = out.Cloud.CacheMissTokens + out.Local.CacheMissTokens
	if denom := out.CacheHitTokens + out.CacheMissTokens; denom > 0 {
		out.CacheHitRate = float64(out.CacheHitTokens) / float64(denom)
	}

	// 4. 节省对比：参考单价 = 云端实际混合单价；无云端用量时回退 deepseek-v4-flash 官价。
	ref := 1.5 // deepseek-v4-flash 混合价（输入 1 + 输出 2 的均值，¥/MTok）
	note := "参考单价取 deepseek-v4-flash 官价混合均价（¥1.5/百万 token）；开启本地路由后按此折算节省"
	if cloudTokens > 0 && out.Cloud.Cost > 0 {
		ref = out.Cloud.Cost / float64(cloudTokens) * 1e6
		note = "参考单价取云端实际混合单价（总费用 / 总 token）；开启本地路由后按此折算节省"
	}
	out.Savings.RefPricePerMTok = ref
	out.Savings.WouldCostCloud = ref * float64(out.Local.TotalTokens) / 1e6
	out.Savings.Saved = out.Savings.WouldCostCloud
	out.Savings.Note = note

	return out
}

// usageSideCounters 按引擎聚合的中间计数。
type usageSideCounters struct {
	calls, succ, fail, in, out, dur int64
	cacheHit, cacheMiss             int64
}

func (c *usageSideCounters) add(pm modelengine.ModelUsageStats) {
	c.calls += pm.CallCount
	c.succ += pm.SuccessCount
	c.fail += pm.FailCount
	c.in += pm.InputTokens
	c.out += pm.OutputTokens
	c.dur += pm.TotalDurationMs
	c.cacheHit += pm.CacheHitTokens
	c.cacheMiss += pm.CacheMissTokens
}

// fromPM 把单条调用统计转为中间计数。
func fromPM(pm modelengine.ModelUsageStats) usageSideCounters {
	return usageSideCounters{
		calls: pm.CallCount, succ: pm.SuccessCount, fail: pm.FailCount,
		in: pm.InputTokens, out: pm.OutputTokens, dur: pm.TotalDurationMs,
		cacheHit: pm.CacheHitTokens, cacheMiss: pm.CacheMissTokens,
	}
}

// add 把中间计数并入 UsageSide。
func (s *UsageSide) add(c usageSideCounters) {
	s.Calls += c.calls
	s.SuccessCalls += c.succ
	s.FailCalls += c.fail
	s.InputTokens += c.in
	s.OutputTokens += c.out
	s.TotalTokens += c.in + c.out
	s.CacheHitTokens += c.cacheHit
	s.CacheMissTokens += c.cacheMiss
	s.TotalDurationMs += c.dur
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// engineIsCloud 供前端/测试复用的引擎分类判断。
func engineIsCloud(engineID string) bool { return cloudEngineSet[strings.TrimSpace(engineID)] }
