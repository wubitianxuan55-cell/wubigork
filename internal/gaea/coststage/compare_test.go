package coststage

import (
	"math"
	"strings"
	"testing"
)

// approx 浮点近似相等(1e-9 容差)。
func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// sv 构造一条阶段值(省略无关字段)。
func sv(stage string, amount float64) StageValue {
	return StageValue{ProjectID: "p", Stage: stage, Amount: amount}
}

// row 取出对比表中指定阶段的行。
func row(t *testing.T, rows []CompareRow, stage string) CompareRow {
	t.Helper()
	for _, r := range rows {
		if r.Stage == stage {
			return r
		}
	}
	t.Fatalf("对比表缺少阶段 %q:%+v", stage, rows)
	return CompareRow{}
}

// mustRows 计算对比表,断言非 nil。
func mustRows(t *testing.T, values []StageValue) []CompareRow {
	t.Helper()
	rows := ComputeComparison(values)
	if rows == nil {
		t.Fatal("ComputeComparison 返回 nil")
	}
	return rows
}

// ── ComputeComparison ──────────────────────────────────────────────

func TestComputeComparisonFullFiveStages(t *testing.T) {
	// 乱序输入,仍按 StageOrder 输出固定 5 行
	rows := mustRows(t, []StageValue{
		sv(StageBudget, 115),
		sv(StageFinal, 160),
		sv(StageEstimate, 100),
		sv(StageDesign, 105),
		sv(StageSettlement, 130),
	})
	if len(rows) != 5 {
		t.Fatalf("len = %d, want 5", len(rows))
	}
	for i, r := range rows {
		if r.Stage != StageOrder[i] {
			t.Fatalf("rows[%d].Stage = %q, want %q", i, r.Stage, StageOrder[i])
		}
	}

	est := row(t, rows, StageEstimate)
	if !est.HasValue || est.HasPrev || est.PrevStage != "" || est.Amount != 100 {
		t.Errorf("估算行 = %+v", est)
	}
	if est.ChainDiff != 0 || est.ChainDiffPct != 0 || est.BaseDiff != 0 || est.BaseDiffPct != 0 {
		t.Errorf("估算行环比/累计差应为 0:%+v", est)
	}

	des := row(t, rows, StageDesign)
	if !des.HasValue || !des.HasPrev || des.PrevStage != StageEstimate || des.Amount != 105 {
		t.Errorf("概算行 = %+v", des)
	}
	if des.ChainDiff != 5 || !approx(des.ChainDiffPct, 5.0) {
		t.Errorf("概算环比 = %+v", des)
	}
	if des.BaseDiff != 5 || !approx(des.BaseDiffPct, 5.0) {
		t.Errorf("概算累计差 = %+v", des)
	}

	bud := row(t, rows, StageBudget)
	if !bud.HasPrev || bud.PrevStage != StageDesign || bud.ChainDiff != 10 {
		t.Errorf("预算行环比 = %+v", bud)
	}
	if !approx(bud.ChainDiffPct, (115-105)/105.0*100) {
		t.Errorf("预算环比差幅 = %v, want %v", bud.ChainDiffPct, (115-105)/105.0*100)
	}
	if bud.BaseDiff != 15 || !approx(bud.BaseDiffPct, 15.0) {
		t.Errorf("预算累计差 = %+v", bud)
	}

	set := row(t, rows, StageSettlement)
	if !set.HasPrev || set.PrevStage != StageBudget || set.ChainDiff != 15 {
		t.Errorf("结算行环比 = %+v", set)
	}
	if set.BaseDiff != 30 || !approx(set.BaseDiffPct, 30.0) {
		t.Errorf("结算累计差 = %+v", set)
	}

	fin := row(t, rows, StageFinal)
	if !fin.HasPrev || fin.PrevStage != StageSettlement || fin.ChainDiff != 30 {
		t.Errorf("决算行环比 = %+v", fin)
	}
	if fin.BaseDiff != 60 || !approx(fin.BaseDiffPct, 60.0) {
		t.Errorf("决算累计差 = %+v", fin)
	}
}

func TestComputeComparisonMissingMiddle(t *testing.T) {
	// 缺概算:预算环比基准应为估算
	rows := mustRows(t, []StageValue{sv(StageEstimate, 100), sv(StageBudget, 115)})

	des := row(t, rows, StageDesign)
	if des.HasValue || des.Amount != 0 || des.HasPrev || des.PrevStage != "" {
		t.Errorf("缺阶段行应为空值:%+v", des)
	}
	if des.ChainDiff != 0 || des.ChainDiffPct != 0 || des.BaseDiff != 0 || des.BaseDiffPct != 0 {
		t.Errorf("缺阶段行数值应为 0:%+v", des)
	}

	bud := row(t, rows, StageBudget)
	if !bud.HasPrev || bud.PrevStage != StageEstimate || bud.ChainDiff != 15 {
		t.Errorf("预算环比基准应为估算:%+v", bud)
	}
	if !approx(bud.ChainDiffPct, 15.0) || !approx(bud.BaseDiffPct, 15.0) {
		t.Errorf("预算差幅 = %+v", bud)
	}

	if r := row(t, rows, StageSettlement); r.HasValue {
		t.Errorf("未录入阶段不应有值:%+v", r)
	}
	if r := row(t, rows, StageFinal); r.HasValue {
		t.Errorf("未录入阶段不应有值:%+v", r)
	}
}

func TestComputeComparisonMissingFirst(t *testing.T) {
	// 缺估算:累计差基准 = 首个有值阶段(概算)
	rows := mustRows(t, []StageValue{sv(StageDesign, 105), sv(StageBudget, 115), sv(StageFinal, 160)})

	if r := row(t, rows, StageEstimate); r.HasValue {
		t.Errorf("估算未录入不应有值:%+v", r)
	}
	des := row(t, rows, StageDesign)
	if !des.HasValue || des.HasPrev || des.PrevStage != "" {
		t.Errorf("首个有值阶段无环比基准:%+v", des)
	}
	if des.BaseDiff != 0 || des.BaseDiffPct != 0 {
		t.Errorf("首个有值阶段累计差应为 0:%+v", des)
	}
	bud := row(t, rows, StageBudget)
	if !bud.HasPrev || bud.PrevStage != StageDesign || bud.ChainDiff != 10 {
		t.Errorf("预算环比 = %+v", bud)
	}
	if !approx(bud.ChainDiffPct, (115-105)/105.0*100) {
		t.Errorf("预算环比差幅 = %v", bud.ChainDiffPct)
	}
	if bud.BaseDiff != 10 || !approx(bud.BaseDiffPct, (115-105)/105.0*100) {
		t.Errorf("预算累计差(基准=概算) = %+v", bud)
	}
	fin := row(t, rows, StageFinal)
	if fin.BaseDiff != 55 || !approx(fin.BaseDiffPct, (160-105)/105.0*100) {
		t.Errorf("决算累计差(基准=概算) = %+v", fin)
	}
}

func TestComputeComparisonNilCases(t *testing.T) {
	if got := ComputeComparison(nil); got != nil {
		t.Fatalf("空输入应返回 nil,got %+v", got)
	}
	if got := ComputeComparison([]StageValue{}); got != nil {
		t.Fatalf("空输入应返回 nil,got %+v", got)
	}
	// 只有 1 个有值阶段
	if got := ComputeComparison([]StageValue{sv(StageEstimate, 100)}); got != nil {
		t.Fatalf("单阶段应返回 nil,got %+v", got)
	}
	// 金额全为 0:不算有值
	if got := ComputeComparison([]StageValue{sv(StageEstimate, 0), sv(StageBudget, 0)}); got != nil {
		t.Fatalf("全 0 应返回 nil,got %+v", got)
	}
	// 0 金额阶段不算有值:只剩 1 个有值
	if got := ComputeComparison([]StageValue{sv(StageEstimate, 0), sv(StageBudget, 115)}); got != nil {
		t.Fatalf("0 金额不算有值,应返回 nil,got %+v", got)
	}
	// 未知阶段不参与对比
	if got := ComputeComparison([]StageValue{sv("未知", 999)}); got != nil {
		t.Fatalf("仅未知阶段应返回 nil,got %+v", got)
	}
}

func TestComputeComparisonZeroAmountStageNotValued(t *testing.T) {
	// 中间阶段金额为 0:不算有值,决算环比基准跳到估算
	rows := mustRows(t, []StageValue{sv(StageEstimate, 100), sv(StageBudget, 0), sv(StageFinal, 160)})
	bud := row(t, rows, StageBudget)
	if bud.HasValue || bud.Amount != 0 {
		t.Errorf("0 金额阶段不算有值:%+v", bud)
	}
	fin := row(t, rows, StageFinal)
	if !fin.HasPrev || fin.PrevStage != StageEstimate || fin.ChainDiff != 60 {
		t.Errorf("决算环比基准应为估算:%+v", fin)
	}
	if !approx(fin.ChainDiffPct, 60.0) || !approx(fin.BaseDiffPct, 60.0) {
		t.Errorf("决算差幅 = %+v", fin)
	}
}

func TestComputeComparisonNegativeBaseProtection(t *testing.T) {
	// 基准金额 <=0 时差幅 % 恒为 0(除零保护);此处直接用 pctChange 验证
	if pctChange(100, 0) != 0 || pctChange(100, -5) != 0 {
		t.Fatal("基准 <=0 时差幅应为 0")
	}
	if !approx(pctChange(110, 100), 10.0) {
		t.Fatalf("pctChange(110,100) = %v", pctChange(110, 100))
	}
}

// ── ExtractDeviations ──────────────────────────────────────────────

func TestExtractDeviationsLevelBoundaries(t *testing.T) {
	cases := []struct {
		name       string
		from, to   float64
		wantLevel  string
		wantSuffix string
	}{
		{"4.99 正常档", 100, 104.99, "正常", "处于正常波动范围"},
		{"5.0 关注档下界", 100, 105, "关注", "建议核查工程量或单价差异"},
		{"15.0 关注档上界", 100, 115, "关注", "建议核查工程量或单价差异"},
		{"15.01 异常档", 100, 115.01, "异常", "异常偏离,建议核查变更签证与调价依据"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rows := mustRows(t, []StageValue{sv(StageEstimate, c.from), sv(StageDesign, c.to)})
			devs := ExtractDeviations(rows)
			if len(devs) != 1 {
				t.Fatalf("devs = %+v, want 1 条", devs)
			}
			d := devs[0]
			if d.FromStage != StageEstimate || d.ToStage != StageDesign {
				t.Errorf("方向 = %+v", d)
			}
			if d.Diff != c.to-c.from {
				t.Errorf("Diff = %v, want %v", d.Diff, c.to-c.from)
			}
			if d.Direction != "上升" {
				t.Errorf("Direction = %q, want 上升", d.Direction)
			}
			if d.Level != c.wantLevel {
				t.Errorf("Level = %q, want %q(pct=%v)", d.Level, c.wantLevel, d.DiffPct)
			}
			if !strings.Contains(d.Suggestion, c.wantSuffix) {
				t.Errorf("Suggestion = %q,缺后缀 %q", d.Suggestion, c.wantSuffix)
			}
		})
	}
}

func TestExtractDeviationsDirection(t *testing.T) {
	// 下降
	rows := mustRows(t, []StageValue{sv(StageEstimate, 100), sv(StageDesign, 90)})
	devs := ExtractDeviations(rows)
	if len(devs) != 1 || devs[0].Direction != "下降" {
		t.Fatalf("下降方向 = %+v", devs)
	}
	if !approx(devs[0].DiffPct, -10.0) {
		t.Errorf("DiffPct = %v, want -10", devs[0].DiffPct)
	}
	if devs[0].Level != "关注" {
		t.Errorf("Level = %q, want 关注", devs[0].Level)
	}
	// Diff == 0:按契约 Diff>=0 记上升
	rows = mustRows(t, []StageValue{sv(StageEstimate, 100), sv(StageDesign, 100)})
	devs = ExtractDeviations(rows)
	if len(devs) != 1 || devs[0].Direction != "上升" || devs[0].Level != "正常" {
		t.Fatalf("平值方向 = %+v", devs)
	}
}

func TestExtractDeviationsSkippedStages(t *testing.T) {
	// 缺概算、缺结算:相邻有值阶段对 = 估算→预算、预算→决算
	rows := mustRows(t, []StageValue{sv(StageEstimate, 100), sv(StageBudget, 115), sv(StageFinal, 160)})
	devs := ExtractDeviations(rows)
	if len(devs) != 2 {
		t.Fatalf("devs = %+v, want 2 条", devs)
	}
	if devs[0].FromStage != StageEstimate || devs[0].ToStage != StageBudget || devs[0].Diff != 15 {
		t.Errorf("第一条偏差 = %+v", devs[0])
	}
	if devs[1].FromStage != StageBudget || devs[1].ToStage != StageFinal || devs[1].Diff != 45 {
		t.Errorf("第二条偏差 = %+v", devs[1])
	}
}

func TestExtractDeviationsNilCases(t *testing.T) {
	if got := ExtractDeviations(nil); got != nil {
		t.Fatalf("空输入应返回 nil,got %+v", got)
	}
	// 只有 1 个有值阶段(缺首阶段场景):手工构造对比行
	rows := make([]CompareRow, len(StageOrder))
	for i, st := range StageOrder {
		rows[i].Stage = st
	}
	rows[1] = CompareRow{Stage: StageDesign, Amount: 105, HasValue: true}
	if got := ExtractDeviations(rows); got != nil {
		t.Fatalf("仅 1 个有值阶段应返回 nil,got %+v", got)
	}
}

func TestDeviationSuggestionFormat(t *testing.T) {
	d := deviationSuggestion("概算", "预算", 8.1)
	if d != "预算较概算 +8.1%,建议核查工程量或单价差异" {
		t.Fatalf("关注档文案 = %q", d)
	}
	d = deviationSuggestion("预算", "结算", 18.2)
	if d != "结算较预算 +18.2%,异常偏离,建议核查变更签证与调价依据" {
		t.Fatalf("异常档文案 = %q", d)
	}
	d = deviationSuggestion("估算", "概算", -2.3)
	if d != "概算较估算 -2.3%,处于正常波动范围" {
		t.Fatalf("正常档文案(下降) = %q", d)
	}
}
