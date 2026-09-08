package cost

// v4.158.0 AI 组价复核闭环:CheckComposeComponents 合理性校验纯函数规则钉死用例。
// 规则编号与 composecheck.go 注释一一对应(R1 金额一致性/R2 非正值/R3 全局合计)。

import (
	"strings"
	"testing"
)

// comp 快捷构造组件行。
func comp(q, p, amount float64) Component {
	return Component{Kind: "材料", Title: "测试资源", Quantity: q, Price: p, Amount: amount}
}

// TestCheckComposeComponentsEmpty 空组件返回 nil(无拆解=无校验,不产生噪音)。
func TestCheckComposeComponentsEmpty(t *testing.T) {
	if got := CheckComposeComponents(nil, 100); got != nil {
		t.Fatalf("nil 组件应返回 nil, got %+v", got)
	}
	if got := CheckComposeComponents([]Component{}, 100); got != nil {
		t.Fatalf("空组件应返回 nil, got %+v", got)
	}
}

// TestCheckComposeComponentsR1AmountMismatch R1:金额 ≠ 含量×单价 → warn(该行)。
// 本例合计 120 恰好也超 100×1.05,故同时钉住全局 warn(顺序:行级在前)。
func TestCheckComposeComponentsR1AmountMismatch(t *testing.T) {
	checks := CheckComposeComponents([]Component{comp(10, 10, 120)}, 100)
	if len(checks) != 2 {
		t.Fatalf("want R1 warn + R3 warn 共 2 条, got %+v", checks)
	}
	c := checks[0]
	if c.Level != "warn" || c.Row != 0 {
		t.Fatalf("want warn@row0, got %+v", c)
	}
	if c.Msg != "金额 120.00 ≠ 含量×单价 100.00" {
		t.Fatalf("R1 文案不符: %q", c.Msg)
	}
	if checks[1].Level != "warn" || checks[1].Row != -1 {
		t.Fatalf("want 全局 warn@-1, got %+v", checks[1])
	}
}

// TestCheckComposeComponentsR1Tolerance R1 容差内不报:|amount|×0.01 覆盖小损耗,
// 小额走绝对下限 0.01。rec=0 跳过全局档,隔离行级规则。
func TestCheckComposeComponentsR1Tolerance(t *testing.T) {
	// 差 0.50 < 容差 max(0.01, 100×0.01)=1.005 → 不触发
	if got := CheckComposeComponents([]Component{comp(1, 100, 100.5)}, 0); len(got) != 0 {
		t.Fatalf("容差内不应触发, got %+v", got)
	}
	// 差 0.008 < 绝对下限 0.01(小额行:|amount|×0.01<0.01 时走绝对下限)→ 不触发
	if got := CheckComposeComponents([]Component{comp(1, 0.5, 0.508)}, 0); len(got) != 0 {
		t.Fatalf("0.01 容差内不应触发, got %+v", got)
	}
	// 差 0.012 > 绝对下限 0.01 → 触发
	if got := CheckComposeComponents([]Component{comp(1, 0.5, 0.512)}, 0); len(got) != 1 {
		t.Fatalf("超出绝对下限应触发, got %+v", got)
	}
}

// TestCheckComposeComponentsR2NonPositive R2:含量/单价非正 → warn(该行),
// 且非法行跳过 R1、不计入 R3 合计(合计只看合法行 40 → info 而非被 1000 抬成 warn)。
func TestCheckComposeComponentsR2NonPositive(t *testing.T) {
	checks := CheckComposeComponents([]Component{
		{Quantity: 0, Price: 50, Amount: 1000}, // 非法行:含量 0
		comp(4, 10, 40),                        // 合法行 40
	}, 100)
	if len(checks) != 2 {
		t.Fatalf("want R2 warn + R3 info 共 2 条, got %+v", checks)
	}
	if checks[0].Level != "warn" || checks[0].Row != 0 || checks[0].Msg != "含量/单价须为正" {
		t.Fatalf("R2 文案/定位不符: %+v", checks[0])
	}
	if checks[1].Level != "info" || checks[1].Row != -1 {
		t.Fatalf("非法行应被过滤出合计(40<50 → info), got %+v", checks[1])
	}

	// 负单价同样触发 R2(rec=0 隔离行级)。
	checksNeg := CheckComposeComponents([]Component{comp(2, -1, -2)}, 0)
	if len(checksNeg) != 1 || checksNeg[0].Msg != "含量/单价须为正" {
		t.Fatalf("负单价应触发 R2: %+v", checksNeg)
	}
}

// TestCheckComposeComponentsR3Over R3 超推荐价:Σ > recommended×1.05 → warn(全局)。
func TestCheckComposeComponentsR3Over(t *testing.T) {
	checks := CheckComposeComponents([]Component{comp(1, 60, 60), comp(1, 60, 60)}, 100)
	if len(checks) != 1 {
		t.Fatalf("want 1 条全局 warn, got %+v", checks)
	}
	c := checks[0]
	if c.Level != "warn" || c.Row != -1 {
		t.Fatalf("want warn@row-1, got %+v", c)
	}
	if c.Msg != "人材机合计 120.00 已超推荐价 100.00" {
		t.Fatalf("R3 warn 文案不符: %q", c.Msg)
	}
}

// TestCheckComposeComponentsR3Under R3 低于推荐价:Σ < recommended×0.5 → info(全局)。
func TestCheckComposeComponentsR3Under(t *testing.T) {
	checks := CheckComposeComponents([]Component{comp(1, 49, 49)}, 100)
	if len(checks) != 1 {
		t.Fatalf("want 1 条全局 info, got %+v", checks)
	}
	c := checks[0]
	if c.Level != "info" || c.Row != -1 {
		t.Fatalf("want info@row-1, got %+v", c)
	}
	if !strings.Contains(c.Msg, "仅为推荐价 49%") {
		t.Fatalf("R3 info 文案不符: %q", c.Msg)
	}
}

// TestCheckComposeComponentsR3Boundaries 边界:恰 1.05 不触发 warn、恰 0.5 不触发 info。
func TestCheckComposeComponentsR3Boundaries(t *testing.T) {
	if got := CheckComposeComponents([]Component{comp(1, 105, 105)}, 100); len(got) != 0 {
		t.Fatalf("恰 1.05 不应触发, got %+v", got)
	}
	if got := CheckComposeComponents([]Component{comp(1, 50, 50)}, 100); len(got) != 0 {
		t.Fatalf("恰 0.5 不应触发, got %+v", got)
	}
}

// TestCheckComposeComponentsOrder 顺序稳定:行级按行序(先 R2 后 R1),再全局。
func TestCheckComposeComponentsOrder(t *testing.T) {
	// 行0:R1 金额不一致;行1:R2 非法(跳过 R1);行2:干净行。
	// 合计(过滤行1)=140 < 500×0.5 → 全局 info。
	checks := CheckComposeComponents([]Component{
		comp(10, 10, 120),
		{Quantity: -1, Price: 5, Amount: 100},
		comp(1, 20, 20),
	}, 500)
	if len(checks) != 3 {
		t.Fatalf("want [R1@0, R2@1, info@-1], got %+v", checks)
	}
	wantRows := []int{0, 1, -1}
	wantLevels := []string{"warn", "warn", "info"}
	for i, c := range checks {
		if c.Row != wantRows[i] || c.Level != wantLevels[i] {
			t.Fatalf("第 %d 条 = %+v, want row=%d level=%s", i, c, wantRows[i], wantLevels[i])
		}
	}
}

// TestCheckComposeComponentsNoRecommendation 推荐价无效(<=0)时只留行级结论。
func TestCheckComposeComponentsNoRecommendation(t *testing.T) {
	checks := CheckComposeComponents([]Component{comp(1, 1, 5)}, 0)
	if len(checks) != 1 || checks[0].Row != 0 || checks[0].Level != "warn" {
		t.Fatalf("推荐价 0 应跳过全局档: %+v", checks)
	}
	if got := CheckComposeComponents([]Component{comp(1, 1, 1)}, -3); len(got) != 0 {
		t.Fatalf("推荐价负且行合法应无结论: %+v", got)
	}
}
