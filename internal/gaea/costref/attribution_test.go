package costref

import (
	"math"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/costproject"
)

// v4.6.1 归因对标纯函数：带宽判定、贡献金额、TopDrivers 排序与 Summary。
func TestComputeAttribution(t *testing.T) {
	items := []costproject.Item{
		{Title: "C30 混凝土", Unit: "m³", Quantity: 10, Price: 600}, // 高于 P75(550) → 高
		{Title: "HRB400 钢筋", Unit: "t", Quantity: 5, Price: 3000}, // P25-P75 内 → 正常
		{Title: "模板", Unit: "m²", Quantity: 8, Price: 40},         // 低于 P25(42) → 低
		{Title: "无参考项", Unit: "m", Quantity: 2, Price: 999},      // 无指标 → 无参考
	}
	indicators := []Indicator{
		{Key: "C30 混凝土", Samples: 10, Median: 500, P25: 450, P75: 550},
		{Key: "HRB400 钢筋", Samples: 8, Median: 3050, P25: 2900, P75: 3200},
		{Key: "模板", Samples: 6, Median: 45, P25: 42, P75: 50},
	}
	at := ComputeAttribution("p1", "项目甲", items, indicators)

	// 无参考项：保留但不进 TopDrivers
	if len(at.Items) != 4 {
		t.Fatalf("items = %d, want 4（含无参考项）", len(at.Items))
	}
	if len(at.TopDrivers) != 3 {
		t.Fatalf("topDrivers = %d, want 3（无参考项不参与归因）", len(at.TopDrivers))
	}
	byTitle := map[string]AttributionItem{}
	for _, it := range at.Items {
		byTitle[it.Title] = it
	}
	if byTitle["C30 混凝土"].Level != "高" || byTitle["HRB400 钢筋"].Level != "正常" || byTitle["模板"].Level != "低" {
		t.Fatalf("档位判定错误: %+v", byTitle)
	}
	// C30 贡献 = (600-500)×10 = 1000；钢筋 = (3000-3050)×5 = -250；模板 = (40-45)×8 = -40
	if math.Abs(byTitle["C30 混凝土"].Contribution-1000) > 0.01 ||
		math.Abs(byTitle["HRB400 钢筋"].Contribution+250) > 0.01 ||
		math.Abs(byTitle["模板"].Contribution+40) > 0.01 {
		t.Fatalf("贡献金额错误: %+v", byTitle)
	}
	// TopDrivers 按 |贡献| 降序：C30(1000) → 钢筋(250) → 模板(40)
	if at.TopDrivers[0].Title != "C30 混凝土" || at.TopDrivers[1].Title != "HRB400 钢筋" || at.TopDrivers[2].Title != "模板" {
		t.Fatalf("TopDrivers 排序错误: %+v", at.TopDrivers)
	}
	// Summary 含主因
	if !strings.Contains(at.Summary, "C30 混凝土") {
		t.Fatalf("Summary 应含主因: %q", at.Summary)
	}
	// 总偏离 = 710；参考基准 = 500×10+3050×5+45×8 = 5000+15250+360 = 20610
	if math.Abs(at.TotalDiff-710) > 0.01 {
		t.Fatalf("totalDiff = %v, want 710", at.TotalDiff)
	}
}

// 带宽退化（P25==P75 单样本）：±10% 兜底，避免一切标异常。
func TestComputeAttribution_BandFallback(t *testing.T) {
	items := []costproject.Item{
		{Title: "独苗", Quantity: 1, Price: 540}, // 中位 500，+8% → 正常
		{Title: "独苗", Quantity: 1, Price: 560}, // 中位 500，+12% → 高
	}
	at := ComputeAttribution("p1", "项目", items, []Indicator{
		{Key: "独苗", Samples: 1, Median: 500, P25: 500, P75: 500},
	})
	byPrice := map[float64]string{}
	for _, it := range at.Items {
		byPrice[it.Price] = it.Level
	}
	if byPrice[540] != "正常" || byPrice[560] != "高" {
		t.Fatalf("带宽退化兜底错误: %+v", byPrice)
	}
}
