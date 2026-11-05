package builtin

// websearch_coverage_test.go — 覆盖门 v0（基线校准版）集成回归：多实体部分
// 命中拦截（F30 基线误判案例）与正题过门。

import (
	"context"
	"testing"

	"github.com/gaea/gaea/internal/gaea/config"
)

// TestRunSearch_CoverageGatePartialHit_F30 基线 F30 实测误判回归：「火星 房地产
// 均价 2026」3 实体仅命中「火星」（火星百科）——任一命中口径误判 OK，
// 过半口径必须判 coverage-miss（F 类不得判 OK 红线）。
func TestRunSearch_CoverageGatePartialHit_F30(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	SetSearchEngineOrder([]string{"core-stub-1"})
	coreStubSearch["core-stub-1"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		return []SearchResult{
			{Title: "火星（太阳系八大行星之一）_百度百科", URL: "https://baike.example/mars", Snippet: "火星是太阳系行星", Source: "bing"},
			{Title: "火星的基本事实 | NASA中文", URL: "https://nasa.example/mars", Snippet: "火星概况", Source: "bing"},
		}, nil
	}
	outcome, err := RunSearch(context.Background(), "火星 房地产 均价 2026", 5)
	if err != nil {
		t.Fatalf("非空结果不该报错: %v", err)
	}
	if outcome.State != SearchStateDegraded || outcome.DegradedReason != "coverage-miss" {
		t.Errorf("State/Reason = %q/%q, want degraded/coverage-miss（部分实体命中不得标 OK）", outcome.State, outcome.DegradedReason)
	}
}

// TestRunSearch_CoverageGateMultiEntityPass 多实体全命中过门（E22 苹果 价格 型）。
func TestRunSearch_CoverageGateMultiEntityPass(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	SetSearchEngineOrder([]string{"core-stub-1"})
	coreStubSearch["core-stub-1"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		return []SearchResult{
			{Title: "苹果价格行情_今日苹果价格多少钱一斤", URL: "https://fruit.example/apple", Snippet: "苹果批发价格走势", Source: "bing"},
		}, nil
	}
	outcome, _ := RunSearch(context.Background(), "苹果 价格", 5)
	if outcome.State != SearchStateOK {
		t.Errorf("State = %q (%s), want ok（全实体命中应过门）", outcome.State, outcome.DegradedReason)
	}
}
