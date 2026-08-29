package cost

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

// near 近似相等(浮点断言,避免裸 ==)。
func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func assertNear(t *testing.T, got, want float64, what string) {
	t.Helper()
	if !near(got, want) {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func TestComputePriceBandEmpty(t *testing.T) {
	if b := ComputePriceBand(nil, ""); b != nil {
		t.Errorf("nil input = %+v, want nil", b)
	}
	if b := ComputePriceBand([]Summary{}, ""); b != nil {
		t.Errorf("empty input = %+v, want nil", b)
	}
	// 全部无效价格 → nil
	entries := []Summary{
		{Name: "a", Price: 0},
		{Name: "b", Price: -5},
	}
	if b := ComputePriceBand(entries, ""); b != nil {
		t.Errorf("all-invalid prices = %+v, want nil", b)
	}
	// 全部单位不匹配 → nil
	entries = []Summary{
		{Name: "a", Unit: "吨", Price: 100},
	}
	if b := ComputePriceBand(entries, "㎡"); b != nil {
		t.Errorf("all unit mismatch = %+v, want nil", b)
	}
}

func TestComputePriceBandSingleSample(t *testing.T) {
	b := ComputePriceBand([]Summary{{Name: "only", Unit: "㎡", Price: 85}}, "")
	if b == nil {
		t.Fatal("band = nil, want non-nil")
	}
	if b.Samples != 1 {
		t.Errorf("Samples = %d, want 1", b.Samples)
	}
	for what, got := range map[string]float64{
		"Min": b.Min, "Max": b.Max, "Mean": b.Mean,
		"Median": b.Median, "P25": b.P25, "P75": b.P75,
	} {
		assertNear(t, got, 85, what)
	}
	assertNear(t, b.SpreadPct, 0, "SpreadPct")
	if b.Outliers != 0 {
		t.Errorf("Outliers = %d, want 0", b.Outliers)
	}
	if b.Confidence != "低" {
		t.Errorf("Confidence = %q, want 低", b.Confidence)
	}
	if len(b.Sources) != 1 || b.Sources[0].Name != "only" {
		t.Errorf("Sources = %+v, want [only]", b.Sources)
	}
}

func TestComputePriceBandQuantiles(t *testing.T) {
	// 1..10:R-7 手算 P25=3.25、Median=5.5、P75=7.75。
	var entries []Summary
	for i := 1; i <= 10; i++ {
		entries = append(entries, Summary{Name: fmt.Sprintf("e%d", i), Price: float64(i)})
	}
	b := ComputePriceBand(entries, "")
	if b == nil {
		t.Fatal("band = nil, want non-nil")
	}
	if b.Samples != 10 {
		t.Errorf("Samples = %d, want 10", b.Samples)
	}
	assertNear(t, b.Min, 1, "Min")
	assertNear(t, b.Max, 10, "Max")
	assertNear(t, b.Mean, 5.5, "Mean")
	assertNear(t, b.P25, 3.25, "P25")
	assertNear(t, b.Median, 5.5, "Median")
	assertNear(t, b.P75, 7.75, "P75")
	assertNear(t, b.SpreadPct, (7.75-3.25)/5.5*100, "SpreadPct")
	if b.Outliers != 0 {
		t.Errorf("Outliers = %d, want 0", b.Outliers)
	}
	if b.Confidence != "高" {
		t.Errorf("Confidence = %q, want 高", b.Confidence)
	}
	// Sources 按 Price 升序且完整。
	if len(b.Sources) != 10 {
		t.Fatalf("Sources len = %d, want 10", len(b.Sources))
	}
	for i, s := range b.Sources {
		assertNear(t, s.Price, float64(i+1), fmt.Sprintf("Sources[%d].Price", i))
	}
}

func TestComputePriceBandOutliers(t *testing.T) {
	// 上端离群:1..9 全部落在 [Q1-1.5IQR, Q3+1.5IQR] = [-3.5, 14.5],100 为离群。
	prices := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 100}
	var entries []Summary
	for i, p := range prices {
		entries = append(entries, Summary{Name: fmt.Sprintf("e%d", i), Price: p})
	}
	b := ComputePriceBand(entries, "")
	if b == nil {
		t.Fatal("band = nil, want non-nil")
	}
	assertNear(t, b.P25, 3.25, "P25")
	assertNear(t, b.P75, 7.75, "P75")
	if b.Outliers != 1 {
		t.Errorf("upper outlier: Outliers = %d, want 1", b.Outliers)
	}
	// 离群样本仍保留在 Sources 中(升序末位)。
	if n := len(b.Sources); n != 10 || b.Sources[n-1].Price != 100 {
		t.Errorf("Sources tail = %+v, want last price 100", b.Sources)
	}

	// 下端离群:1 低于下界 44.5。
	prices = []float64{1, 50, 51, 52, 53, 54, 55, 56, 57, 58}
	entries = entries[:0]
	for i, p := range prices {
		entries = append(entries, Summary{Name: fmt.Sprintf("e%d", i), Price: p})
	}
	b = ComputePriceBand(entries, "")
	if b == nil {
		t.Fatal("band = nil, want non-nil")
	}
	if b.Outliers != 1 {
		t.Errorf("lower outlier: Outliers = %d, want 1", b.Outliers)
	}
	assertNear(t, b.Sources[0].Price, 1, "Sources[0].Price")
}

func TestComputePriceBandUnitFilter(t *testing.T) {
	entries := []Summary{
		{Name: "sq", Unit: "㎡", Price: 100},
		{Name: "ton", Unit: "吨", Price: 200},
		{Name: "blank", Unit: "", Price: 300},
		{Name: "sqSpace", Unit: " ㎡ ", Price: 400},
	}
	// 匹配:Unit 与 unit 一致(TrimSpace 后)的条目 + Unit 为空的条目保留。
	b := ComputePriceBand(entries, "㎡")
	if b == nil || b.Samples != 3 {
		t.Fatalf("unit ㎡ band = %+v, want 3 samples", b)
	}
	for _, s := range b.Sources {
		if s.Name == "ton" {
			t.Errorf("unit filter kept ton entry: %+v", s)
		}
	}
	// 不匹配单位 → 只剩 Unit 为空的条目。
	b = ComputePriceBand(entries, "工日")
	if b == nil || b.Samples != 1 || b.Sources[0].Name != "blank" {
		t.Fatalf("unit 工日 band = %+v, want [blank]", b)
	}
	// unit 为空不过滤。
	b = ComputePriceBand(entries, "")
	if b == nil || b.Samples != 4 {
		t.Fatalf("no-unit band = %+v, want 4 samples", b)
	}
}

func TestComputePriceBandInvalidPrice(t *testing.T) {
	entries := []Summary{
		{Name: "zero", Unit: "㎡", Price: 0},
		{Name: "neg", Unit: "㎡", Price: -10},
		{Name: "ok", Unit: "㎡", Price: 120},
		{Name: "ok2", Unit: "㎡", Price: 80},
	}
	b := ComputePriceBand(entries, "㎡")
	if b == nil || b.Samples != 2 {
		t.Fatalf("band = %+v, want 2 samples", b)
	}
	assertNear(t, b.Min, 80, "Min")
	assertNear(t, b.Max, 120, "Max")
	assertNear(t, b.Median, 100, "Median")
}

func TestComputePriceBandSourcesSortedStable(t *testing.T) {
	now := time.Now().UTC()
	entries := []Summary{
		{Name: "c", Unit: "㎡", Price: 100, UpdatedAt: now, Source: "定额", Region: "成都"},
		{Name: "a", Unit: "㎡", Price: 100, UpdatedAt: now, Source: "市场询价", Region: "上海"},
		{Name: "b", Unit: "㎡", Price: 50, UpdatedAt: now},
	}
	b := ComputePriceBand(entries, "㎡")
	if b == nil {
		t.Fatal("band = nil, want non-nil")
	}
	// 升序;同价(100)保持输入相对顺序 c 在 a 前(稳定排序)。
	want := []string{"b", "c", "a"}
	if len(b.Sources) != 3 {
		t.Fatalf("Sources len = %d, want 3", len(b.Sources))
	}
	for i, s := range b.Sources {
		if s.Name != want[i] {
			t.Errorf("Sources[%d].Name = %q, want %q", i, s.Name, want[i])
		}
	}
	// 字段透传。
	s := b.Sources[1]
	if s.Unit != "㎡" || s.Source != "定额" || s.Region != "成都" || !s.UpdatedAt.Equal(now) || s.Price != 100 {
		t.Errorf("Sources[1] pass-through mismatch: %+v", s)
	}
}

func TestSpreadPctZeroMedian(t *testing.T) {
	// Median==0 时离散度为 0(ComputePriceBand 过滤后价格恒 >0,此分支由工具函数直测)。
	assertNear(t, spreadPct(10, 20, 0), 0, "spreadPct with zero median")
	assertNear(t, spreadPct(10, 30, 20), 100, "spreadPct normal")
}

func TestRecommendPriceModes(t *testing.T) {
	b := &PriceBand{
		Samples: 12, Min: 60, Max: 120, Mean: 88.5, Median: 85,
		P25: 78, P75: 95, SpreadPct: 20, Outliers: 1, Confidence: "高",
	}
	cases := []struct {
		mode string
		want float64
		word string
	}{
		{"", 85, "中位数"},
		{"median", 85, "中位数"},
		{"mean", 88.5, "均值"},
		{"p25", 78, "P25 分位"},
		{"p75", 95, "P75 分位"},
		{"conservative", 60, "保守价"},
		{"unknown", 85, "中位数"}, // 未知 mode 按 median
	}
	for _, c := range cases {
		got, text := RecommendPrice(b, c.mode)
		assertNear(t, got, c.want, "price for mode "+c.mode)
		if !strings.Contains(text, c.word) {
			t.Errorf("mode %q text = %q, want contains %q", c.mode, text, c.word)
		}
		if !strings.Contains(text, "基于 12 条相似条目") || !strings.Contains(text, "置信度 高") {
			t.Errorf("mode %q text = %q, want samples & confidence", c.mode, text)
		}
		if !strings.Contains(text, "¥") {
			t.Errorf("mode %q text = %q, want ¥ symbol", c.mode, text)
		}
	}
	// 中位数文案精确抽查。
	if got, text := RecommendPrice(b, ""); got != 85 || text != "基于 12 条相似条目,中位数 ¥85.00,置信度 高" {
		t.Errorf("median = (%v, %q), want 精确文案", got, text)
	}
	// nil 分支。
	if got, text := RecommendPrice(nil, "median"); got != 0 || text != "" {
		t.Errorf("nil band = (%v, %q), want (0, \"\")", got, text)
	}
}

func TestConfidenceTiers(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{1, "低"}, {3, "低"}, {4, "中"}, {7, "中"}, {8, "高"}, {12, "高"},
	}
	for _, c := range cases {
		var entries []Summary
		for i := 0; i < c.n; i++ {
			entries = append(entries, Summary{Name: fmt.Sprintf("e%d", i), Price: float64(100 + i)})
		}
		b := ComputePriceBand(entries, "")
		if b == nil {
			t.Fatalf("n=%d band = nil", c.n)
		}
		if b.Confidence != c.want {
			t.Errorf("n=%d Confidence = %q, want %q", c.n, b.Confidence, c.want)
		}
	}
}

func TestRecommendPriceIntegration(t *testing.T) {
	// 12 条样本走完整链路:median=R-7 插值 86.5,置信度 高。
	prices := []float64{70, 75, 78, 80, 82, 85, 88, 90, 92, 95, 98, 105}
	var entries []Summary
	for i, p := range prices {
		entries = append(entries, Summary{Name: fmt.Sprintf("e%d", i), Unit: "㎡", Price: p})
	}
	b := ComputePriceBand(entries, "㎡")
	if b == nil || b.Samples != 12 {
		t.Fatalf("band = %+v, want 12 samples", b)
	}
	got, text := RecommendPrice(b, "")
	assertNear(t, got, 86.5, "median")
	if !strings.Contains(text, "基于 12 条相似条目") || !strings.Contains(text, "¥86.50") || !strings.Contains(text, "置信度 高") {
		t.Errorf("text = %q, want 含 12条/¥86.50/高", text)
	}
}
