package app

import (
	"math"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

// TestBuildUsageOverview_CloudAndLocal 云端 + 本地（events.jsonl）分流：
// 本地 token = events 全量；节省按云端实际混合单价折算。
func TestBuildUsageOverview_CloudAndLocal(t *testing.T) {
	gaeaStats := modelengine.ModelStatsSummary{
		TotalCost: 3.0, // 云端 3 元
		PerModel: []modelengine.ModelUsageStats{
			{EngineID: "xai", Model: "grok-4.20", CallCount: 2, SuccessCount: 2,
				InputTokens: 1_000_000, OutputTokens: 500_000, TotalTokens: 1_500_000},
			{EngineID: "herdsman", Model: "qwen3-8b", CallCount: 3, SuccessCount: 3,
				InputTokens: 100, OutputTokens: 50, TotalTokens: 150},
		},
	}
	hm := HerdsmanModelStats{PerModel: []HerdsmanModelStat{
		{Model: "qwen3-8b", Calls: 5, Succeeded: 5, InputTokens: 1_000, OutputTokens: 500},
		{Model: "zimage-turbo", Calls: 1, Succeeded: 1, InputTokens: 200, OutputTokens: 100},
	}}
	o := buildUsageOverview(gaeaStats, hm)

	if o.Cloud.Calls != 2 || o.Cloud.TotalTokens != 1_500_000 || o.Cloud.Cost != 3.0 {
		t.Fatalf("cloud = %+v", o.Cloud)
	}
	if o.Cloud.Engines[0] != "xai" {
		t.Fatalf("cloud engines = %v", o.Cloud.Engines)
	}
	// 本地：events 全量（5+1 次调用、1800 token），herdsman 不重复计。
	if o.Local.Calls != 6 || o.Local.TotalTokens != 1800 {
		t.Fatalf("local = %+v", o.Local)
	}
	// 参考单价 = 3 元 / 1.5M token = 2 元/MTok；节省 = 2 × 1800 / 1e6。
	if math.Abs(o.Savings.RefPricePerMTok-2.0) > 1e-9 {
		t.Fatalf("ref price = %v, want 2.0", o.Savings.RefPricePerMTok)
	}
	want := 2.0 * 1800 / 1e6
	if math.Abs(o.Savings.Saved-want) > 1e-9 {
		t.Fatalf("saved = %v, want %v", o.Savings.Saved, want)
	}
}

// TestBuildUsageOverview_NoEventsFallback events.jsonl 不可用时：
// 本地 = gaea 侧本地引擎记录（herdsman + 其他）。
func TestBuildUsageOverview_NoEventsFallback(t *testing.T) {
	gaeaStats := modelengine.ModelStatsSummary{
		TotalCost: 0,
		PerModel: []modelengine.ModelUsageStats{
			{EngineID: "herdsman", Model: "qwen3-8b", CallCount: 4, SuccessCount: 4,
				InputTokens: 400, OutputTokens: 200, TotalTokens: 600},
			{EngineID: "ollama", Model: "qwen2.5", CallCount: 1, SuccessCount: 1,
				InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
		},
	}
	o := buildUsageOverview(gaeaStats, HerdsmanModelStats{})

	if o.Local.Calls != 5 || o.Local.TotalTokens != 615 {
		t.Fatalf("local = %+v", o.Local)
	}
	// 无云端用量 → 参考单价回退 1.5 元/MTok。
	if math.Abs(o.Savings.RefPricePerMTok-1.5) > 1e-9 {
		t.Fatalf("ref price = %v, want 1.5", o.Savings.RefPricePerMTok)
	}
}

// TestBuildUsageOverview_CacheHitRate 云端/本地拆分携带缓存命中 token：
// 命中率 = hit/(hit+miss)；全局口径 = 云端 + 本地（gaea 侧记录）。
func TestBuildUsageOverview_CacheHitRate(t *testing.T) {
	gaeaStats := modelengine.ModelStatsSummary{
		TotalCost: 1.0,
		PerModel: []modelengine.ModelUsageStats{
			{EngineID: "deepseek", Model: "deepseek-v4-flash", CallCount: 2, SuccessCount: 2,
				InputTokens: 1_000, OutputTokens: 100, TotalTokens: 1_100,
				CacheHitTokens: 600, CacheMissTokens: 400},
			{EngineID: "xai", Model: "grok-4.20", CallCount: 1, SuccessCount: 1,
				InputTokens: 200, OutputTokens: 20, TotalTokens: 220,
				CacheHitTokens: 50, CacheMissTokens: 150},
			{EngineID: "ollama", Model: "qwen2.5", CallCount: 1, SuccessCount: 1,
				InputTokens: 100, OutputTokens: 10, TotalTokens: 110,
				CacheHitTokens: 90, CacheMissTokens: 10},
		},
	}
	o := buildUsageOverview(gaeaStats, HerdsmanModelStats{})
	if o.Cloud.CacheHitTokens != 650 || o.Cloud.CacheMissTokens != 550 {
		t.Fatalf("cloud cache = %d/%d, want 650/550", o.Cloud.CacheHitTokens, o.Cloud.CacheMissTokens)
	}
	if o.Local.CacheHitTokens != 90 || o.Local.CacheMissTokens != 10 {
		t.Fatalf("local cache = %d/%d, want 90/10", o.Local.CacheHitTokens, o.Local.CacheMissTokens)
	}
	// 全局命中率 = (600+50+90)/((600+50+90)+(400+150+10)) = 740/1300
	if o.CacheHitTokens != 740 || o.CacheMissTokens != 560 {
		t.Fatalf("global cache = %d/%d, want 740/560", o.CacheHitTokens, o.CacheMissTokens)
	}
	want := 740.0 / 1300.0
	if math.Abs(o.CacheHitRate-want) > 1e-9 {
		t.Fatalf("CacheHitRate = %v, want %v", o.CacheHitRate, want)
	}
}

// TestBuildUsageOverview_CacheHitRateNoData 无任何缓存数据时命中率为 0
// （分母为 0 不 panic、不产生脏数据）。
func TestBuildUsageOverview_CacheHitRateNoData(t *testing.T) {
	gaeaStats := modelengine.ModelStatsSummary{
		TotalCost: 1.0,
		PerModel: []modelengine.ModelUsageStats{
			{EngineID: "deepseek", Model: "deepseek-v4-flash", CallCount: 1, SuccessCount: 1,
				InputTokens: 100, OutputTokens: 10, TotalTokens: 110},
		},
	}
	o := buildUsageOverview(gaeaStats, HerdsmanModelStats{})
	if o.CacheHitTokens != 0 || o.CacheMissTokens != 0 || o.CacheHitRate != 0 {
		t.Fatalf("no-data cache = %d/%d rate %v, want 0/0/0",
			o.CacheHitTokens, o.CacheMissTokens, o.CacheHitRate)
	}
	if o.Cloud.CacheHitTokens != 0 || o.Cloud.CacheMissTokens != 0 {
		t.Fatalf("cloud cache = %d/%d, want 0/0", o.Cloud.CacheHitTokens, o.Cloud.CacheMissTokens)
	}
}

// TestBuildUsageOverview_CloudOnly 只有云端用量：本地为空，节省为 0。
func TestBuildUsageOverview_CloudOnly(t *testing.T) {
	gaeaStats := modelengine.ModelStatsSummary{
		TotalCost: 10,
		PerModel: []modelengine.ModelUsageStats{
			{EngineID: "deepseek", Model: "deepseek-v4-flash", CallCount: 10, SuccessCount: 9, FailCount: 1,
				InputTokens: 2_000_000, OutputTokens: 1_000_000, TotalTokens: 3_000_000},
		},
	}
	o := buildUsageOverview(gaeaStats, HerdsmanModelStats{})
	if o.Cloud.Calls != 10 || o.Cloud.SuccessCalls != 9 || o.Cloud.FailCalls != 1 {
		t.Fatalf("cloud = %+v", o.Cloud)
	}
	if o.Local.TotalTokens != 0 || o.Savings.Saved != 0 {
		t.Fatalf("local = %+v, savings = %+v", o.Local, o.Savings)
	}
	if o.Cloud.Engines[0] != "deepseek" {
		t.Fatalf("cloud engines = %v", o.Cloud.Engines)
	}
}
