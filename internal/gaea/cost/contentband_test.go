package cost

// v4.209.0 造价刀路池 §6 收官:CheckContentBaseline 含量对照纯函数规则钉死用例。
// 规则与 contentband.go 注释一一对应(标题+单位归一匹配/分位带/带外 warn/
// 同值退化带 ±5%/样本不足静默/非法行跳过)。

import (
	"strings"
	"testing"
)

// pcomp 快捷构造池内组件行。
func pcomp(title, unit string, q float64) Component {
	return Component{Kind: "材料", Title: title, Unit: unit, Quantity: q, Price: 1, Amount: q}
}

// poolOf 把若干组件行包装成池(单条目一批)。
func poolOf(css ...[]Component) [][]Component { return css }

// TestCheckContentBaselineOutsideBand 带外双向:高于 P75 / 低于 P25 各产一条
// warn,文案含样本数、边界与偏差;带内行静默。
func TestCheckContentBaselineOutsideBand(t *testing.T) {
	// 池:5 例水泥含量 280~320(中位 300),2 例人工工日 0.4~0.6。
	cement := []Component{
		pcomp("C32.5 水泥", "kg", 280),
		pcomp("C32.5 水泥", "kg", 300),
		pcomp("C32.5 水泥", "kg", 300),
		pcomp("C32.5 水泥", "kg", 310),
		pcomp("C32.5 水泥", "kg", 320),
	}
	labor := []Component{
		pcomp("普通工", "工日", 0.4),
		pcomp("普通工", "工日", 0.5),
		pcomp("普通工", "工日", 0.6),
	}
	target := []Component{
		{Title: "C32.5 水泥", Unit: "kg", Quantity: 420}, // 高于 P75=315
		{Title: "C32.5 水泥", Unit: "kg", Quantity: 300}, // 带内,静默
		{Title: "普通工", Unit: "工日", Quantity: 0.2},      // 低于 P25=0.45
	}
	checks := CheckContentBaseline(target, poolOf(cement, labor))
	if len(checks) != 2 {
		t.Fatalf("want 2 条带外 warn, got %+v", checks)
	}
	if checks[0].Row != 0 || checks[0].Level != "warn" {
		t.Fatalf("row0 应为偏高 warn, got %+v", checks[0])
	}
	if !strings.Contains(checks[0].Msg, "高") || !strings.Contains(checks[0].Msg, "5 例") {
		t.Fatalf("偏高文案应含方向与样本数: %q", checks[0].Msg)
	}
	if checks[1].Row != 2 || !strings.Contains(checks[1].Msg, "低") {
		t.Fatalf("row2 应为偏低 warn: %+v", checks[1])
	}
}

// TestCheckContentBaselineNormalization 标题/单位归一匹配:全角、大小写、
// 空白差异同键命中;单位不同(kg vs t)一律不比。
func TestCheckContentBaselineNormalization(t *testing.T) {
	pool := poolOf([]Component{
		pcomp("C32.5 水泥", "kg", 300),
		pcomp("ｃ３２．５ 水泥", "kg", 300),
		pcomp("C32.5  水泥", "KG", 300),
	})
	// 目标全角+小写+多空白 → 命中 3 例;300 在同值带 ±5% 内,静默。
	target := []Component{{Title: "ｃ３２．５水泥", Unit: "kg", Quantity: 305}}
	if got := CheckContentBaseline(target, pool); got != nil {
		t.Fatalf("归一命中且带内应静默, got %+v", got)
	}
	// 同标题不同单位:不构成样本,静默。
	target2 := []Component{{Title: "C32.5 水泥", Unit: "t", Quantity: 99}}
	if got := CheckContentBaseline(target2, pool); got != nil {
		t.Fatalf("单位不同不可比应静默, got %+v", got)
	}
	// 一方无单位:不可比,静默。
	target3 := []Component{{Title: "C32.5 水泥", Quantity: 99}}
	if got := CheckContentBaseline(target3, pool); got != nil {
		t.Fatalf("一方缺单位应静默, got %+v", got)
	}
	// 双方都无单位:可比。
	pool2 := poolOf([]Component{
		pcomp("中砂", "", 1.1),
		pcomp("中砂", "", 1.2),
		pcomp("中砂", "", 1.3),
	})
	target4 := []Component{{Title: "中砂", Quantity: 5}} // 高于 P75=1.25
	got := CheckContentBaseline(target4, pool2)
	if len(got) != 1 || !strings.Contains(got[0].Msg, "高") {
		t.Fatalf("双方无单位应可比并报偏高, got %+v", got)
	}
}

// TestCheckContentBaselineDegenerateBand 同值退化带:P25==P75 时按 ±5% 容差
// 判带外——同值样本不该把微小差异标异常,显著偏离仍报。
func TestCheckContentBaselineDegenerateBand(t *testing.T) {
	pool := poolOf([]Component{
		pcomp("防水涂料", "kg", 2),
		pcomp("防水涂料", "kg", 2),
		pcomp("防水涂料", "kg", 2),
	})
	inBand := []Component{{Title: "防水涂料", Unit: "kg", Quantity: 2.08}} // +4% 带内
	if got := CheckContentBaseline(inBand, pool); got != nil {
		t.Fatalf("±5%% 内应静默, got %+v", got)
	}
	outBand := []Component{{Title: "防水涂料", Unit: "kg", Quantity: 2.2}} // +10% 带外
	got := CheckContentBaseline(outBand, pool)
	if len(got) != 1 {
		t.Fatalf("±5%% 外应报, got %+v", got)
	}
}

// TestCheckContentBaselineGuards 样本不足/非法行/空入参:全部静默或跳过,
// 不产生噪音结论。
func TestCheckContentBaselineGuards(t *testing.T) {
	pool := poolOf([]Component{pcomp("钢筋", "t", 0.1), pcomp("钢筋", "t", 0.1)}) // 仅 2 例
	target := []Component{{Title: "钢筋", Unit: "t", Quantity: 1}}
	if got := CheckContentBaseline(target, pool); got != nil {
		t.Fatalf("样本 <3 应静默, got %+v", got)
	}
	// 目标非法行(Quantity<=0)跳过,不参与对照。
	pool3 := poolOf([]Component{pcomp("钢筋", "t", 0.1), pcomp("钢筋", "t", 0.1), pcomp("钢筋", "t", 0.1)})
	target3 := []Component{{Title: "钢筋", Unit: "t", Quantity: 0}}
	if got := CheckContentBaseline(target3, pool3); got != nil {
		t.Fatalf("非法行应跳过, got %+v", got)
	}
	if CheckContentBaseline(nil, pool3) != nil || CheckContentBaseline(target3, nil) != nil {
		t.Fatalf("空入参应返回 nil")
	}
	// 池内 Quantity<=0 的样本不计入。
	poolBad := poolOf([]Component{
		pcomp("钢筋", "t", 0),
		pcomp("钢筋", "t", 0.1),
		pcomp("钢筋", "t", -1),
	})
	if got := CheckContentBaseline(target, poolBad); got != nil {
		t.Fatalf("有效样本不足应静默, got %+v", got)
	}
}
