package app

import (
	"testing"
	"time"

	"github.com/gaea/gaea/internal/modelengine"
)

// TestBuildRouteLedgerEmpty 零样本：空视图不报错、features 非-nil 空表、total 全零。
func TestBuildRouteLedgerEmpty(t *testing.T) {
	v := buildRouteLedger(modelengine.ModelStatsSummary{}, time.Unix(0, 0))
	if v.Features == nil || len(v.Features) != 0 {
		t.Fatalf("零样本 features 应为非-nil 空表: %+v", v.Features)
	}
	if v.Total != (RouteLedgerTotal{}) {
		t.Fatalf("零样本 total 应全零: %+v", v.Total)
	}
	if v.GeneratedAt == "" {
		t.Fatal("generated_at 不得为空")
	}
}

// TestBuildRouteLedgerTotals total 口径=汇总顶层精确值（非行内成功率反推）。
func TestBuildRouteLedgerTotals(t *testing.T) {
	stats := modelengine.ModelStatsSummary{
		TotalCalls: 100, SuccessCalls: 92, FailCalls: 8,
		InputTokens: 1000, OutputTokens: 500,
		TotalCost: 1.25, AvgDurationMs: 2300,
		PerFeature: []modelengine.FeatureUsageSummary{
			{Feature: "chat", EngineID: "xai", Model: "grok-4", Calls: 80, TokensIn: 800, TokensOut: 400, CostCNY: 1.0, AvgMs: 2400, SuccessRate: 0.9},
			{Feature: "", EngineID: "xai", Model: "grok-4", Calls: 20, TokensIn: 200, TokensOut: 100, CostCNY: 0.25, AvgMs: 1900, SuccessRate: 1},
		},
	}
	v := buildRouteLedger(stats, time.Unix(0, 0))
	if v.Total.Calls != 100 || v.Total.SuccessRate != 0.92 {
		t.Fatalf("total 口径应取顶层精确值: %+v", v.Total)
	}
	if v.Total.TokensIn != 1000 || v.Total.TokensOut != 500 || v.Total.CostCNY != 1.25 || v.Total.AvgMs != 2300 {
		t.Fatalf("total 聚合不符: %+v", v.Total)
	}
	if len(v.Features) != 2 {
		t.Fatalf("features 透传应保序保量: %+v", v.Features)
	}
	if v.Features[0].Feature != "chat" || v.Features[1].Feature != "" {
		t.Fatalf("行序透传（统计层已定序，未标记最后）: %+v", v.Features)
	}
}
