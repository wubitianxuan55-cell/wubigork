package app

// 路由账本（阶段七 7.1-2 A 线）：功能域维度的用量/成本归因视图。数据源=
// modelengine 统计汇总的 PerFeature 三维桶行（feature×engine×model，由
// RecordCall 全量无旁路落账）；本文件只做视图组装（total 聚合 + 透传行），
// 零样本返回空视图不报错。纯函数 buildRouteLedger 可注入测试（先例
// buildUsageOverview）。
import (
	"time"

	"github.com/gaea/gaea/internal/modelengine"
)

// RouteLedgerTotal 账本总计（口径=统计汇总顶层：与功能行同源同窗）。
type RouteLedgerTotal struct {
	Calls       int64   `json:"calls"`
	TokensIn    int64   `json:"tokens_in"`
	TokensOut   int64   `json:"tokens_out"`
	CostCNY     float64 `json:"cost_cny"`
	AvgMs       int64   `json:"avg_ms"`
	SuccessRate float64 `json:"success_rate"` // 0-1；无调用时 0
}

// RouteLedgerView 路由账本（Wails 绑定，模型中心「成本归因」tab 消费）。
type RouteLedgerView struct {
	GeneratedAt string                            `json:"generated_at"`
	Total       RouteLedgerTotal                  `json:"total"`
	Features    []modelengine.FeatureUsageSummary `json:"features"` // feature="" 桶=历史/未标记，排序固定最后（统计层产出）
}

// GaeaRouteLedger 返回功能域成本归因账本。
func (a *App) GaeaRouteLedger() (RouteLedgerView, error) {
	return buildRouteLedger(a.GetModelCallStats(), time.Now()), nil
}

// buildRouteLedger 纯组装：total 取汇总顶层精确值（不经行内成功率反推，
// 无舍入损耗）；features 透传 PerFeature（含空桶处理）。
func buildRouteLedger(stats modelengine.ModelStatsSummary, now time.Time) RouteLedgerView {
	out := RouteLedgerView{
		GeneratedAt: now.Format(time.RFC3339),
		Features:    append([]modelengine.FeatureUsageSummary{}, stats.PerFeature...),
	}
	if out.Features == nil {
		out.Features = []modelengine.FeatureUsageSummary{}
	}
	out.Total = RouteLedgerTotal{
		Calls:       stats.TotalCalls,
		TokensIn:    stats.InputTokens,
		TokensOut:   stats.OutputTokens,
		CostCNY:     stats.TotalCost,
		AvgMs:       stats.AvgDurationMs,
		SuccessRate: 0,
	}
	if stats.TotalCalls > 0 {
		out.Total.SuccessRate = float64(stats.SuccessCalls) / float64(stats.TotalCalls)
	}
	return out
}
