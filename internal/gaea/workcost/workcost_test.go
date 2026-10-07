package workcost

// ── 工料法核算内核用例 ──────────────────────────────────────────────
//
// 金样取自模版 `.gaea/exports/市政道路改造工程成本测算.xlsx` 的实测口径：
// 「成本测算」表 12-17 行就是 ComposeUnitPrice 的展开，其中第 18 行的
// 综合单价 = 含税总造价 ÷ 计量基数。本文件把取费基数链逐段钉住——
// 任何一段的基数改错（如利润误按直接费算，而非「直接费+企管+规费」），
// 对应用例即 FAIL。

import (
	"math"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// closeTo 金额比较（容差 1 分，核算链路逐段 round2 后仍可能有末位抖动）。
func closeTo(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.011 {
		t.Errorf("%s = %.4f, want %.4f", label, got, want)
	}
}

// TestComposeUnitPriceOnlyLaborMaterialMachine 口径铁律：综合单价**只含人材机**，
// 不含管理费/利润/税金（实测产物「综合单价」表原话：只含人材机，不含管理费/
// 利润/税金）。本用例钉住「综合单价 == 人材机合价」，任何把取费内联进单价的
// 改动都会让它变红。
//
// 构造人材机合价恰为 10000 元（人工 3000 / 材料 6000 / 机械 1000）。
func TestComposeUnitPriceOnlyLaborMaterialMachine(t *testing.T) {
	lines := []ComposeLine{
		{Kind: KindLabor, Title: "普通工", Unit: "工日", Quantity: 10, Price: 300},
		{Kind: KindMaterial, Title: "细粒式沥青混凝土 AC-13", Unit: "t", Quantity: 12, Price: 500},
		{Kind: KindMachine, Title: "挖掘机1.0m³", Unit: "台班", Quantity: 0.5, Price: 2000},
	}
	c := ComposeUnitPrice(lines)

	closeTo(t, "人工费", c.LaborFee, 3000)
	closeTo(t, "材料费", c.MaterialFee, 6000)
	closeTo(t, "机械费", c.MachineFee, 1000)
	closeTo(t, "人材机小计", c.Subtotal, 10000)
	closeTo(t, "综合单价（=人材机小计）", c.CompositePrice, 10000)
	if len(c.Warnings) != 0 {
		t.Errorf("合法组成行不应产生告警，得到 %v", c.Warnings)
	}
	// 反向：综合单价里绝不能出现取费（若有人把 RateSet 重新接回单价，这里会炸）。
	if c.CompositePrice != c.Subtotal {
		t.Fatal("综合单价必须等于人材机小计——取费不得进入综合单价")
	}
}

// TestComposeUnitPriceEmptyQuotaNonNullSlices 空定额（无含量行）是合法输入：
// Lines/Warnings 必须构造即非 nil——nil 切片经 Wails JSON 序列化成 null，前端
// `lines.length` 直接崩（v4.467 真机实证：清单分析弹窗整页进 ErrorBoundary，
// 用户表述为「无法查看组价明细/定额库打开失败」）。
func TestComposeUnitPriceEmptyQuotaNonNullSlices(t *testing.T) {
	c := ComposeUnitPrice(nil)
	if c.Lines == nil || c.Warnings == nil {
		t.Fatal("空定额的 Lines/Warnings 必须是空切片而非 nil（JSON null 会崩前端）")
	}
	if len(c.Lines) != 0 || len(c.Warnings) != 0 {
		t.Fatalf("空定额不应有行/告警，得到 %d 行 %d 告警", len(c.Lines), len(c.Warnings))
	}
	if c.CompositePrice != 0 {
		t.Fatalf("空定额综合单价应为 0，得到 %v", c.CompositePrice)
	}
}

// TestComposeProjectFeesSoilRemediation 取费链金样：百锦路地块土壤修复成本测算表
// 「费用汇总」52-66 行实测口径（企管 5%、利润 0、规费不计列、增值税 3%）。
//
//	直接费 = 10000
//	企业管理费 = 10000 × 5% = 500
//	利润 = (10000+500) × 0 = 0
//	规费 = 0（本测算不计列）
//	税前合计 = 10500
//	增值税 = 10500 × 3% = 315
//	含税总造价 = 10815
func TestComposeProjectFeesSoilRemediation(t *testing.T) {
	rates := RateSet{ManagementRate: 0.05, RegulatoryRate: 0, ProfitRate: 0, TaxRate: 0.03}
	f := ComposeProjectFees(10000, rates, FeePolicy{}, 0, 0, 0)

	closeTo(t, "直接费", f.DirectFee, 10000)
	closeTo(t, "企业管理费", f.ManagementFee, 500)
	closeTo(t, "利润", f.ProfitFee, 0)
	closeTo(t, "规费", f.RegulatoryFee, 0)
	closeTo(t, "税前合计", f.PreTaxTotal, 10500)
	closeTo(t, "增值税", f.TaxFee, 315)
	closeTo(t, "含税总造价", f.Total, 10815)
}

// TestComposeProjectFeesProfitBase 利润基数两种口径：默认「直接费+企管」，
// 开启 ProfitBaseIncludesRegulatory 后为「直接费+企管+规费」。
func TestComposeProjectFeesProfitBase(t *testing.T) {
	rates := RateSet{ManagementRate: 0.10, RegulatoryRate: 0.02, ProfitRate: 0.07}

	// 默认口径（百锦路）：利润 = (10000+1000)×7% = 770，不含规费 200。
	a := ComposeProjectFees(10000, rates, FeePolicy{}, 0, 0, 0)
	closeTo(t, "利润（不含规费基数）", a.ProfitFee, 770)

	// 市政道路口径：利润 = (10000+1000+200)×7% = 784。
	b := ComposeProjectFees(10000, rates, FeePolicy{ProfitBaseIncludesRegulatory: true}, 0, 0, 0)
	closeTo(t, "利润（含规费基数）", b.ProfitFee, 784)
	if b.ProfitFee <= a.ProfitFee {
		t.Fatal("含规费基数的利润应更高")
	}
}

// TestComposeProjectFeesControlPrice 招标控制价对照（实测产物 63-66 行）：
// 差额与利用率按含税总造价口径，超出即正数（超限）。
func TestComposeProjectFeesControlPrice(t *testing.T) {
	rates := RateSet{ManagementRate: 0.05, TaxRate: 0.03}
	f := ComposeProjectFees(100000, rates, FeePolicy{}, 0, 0, 120000)
	closeTo(t, "含税总造价", f.Total, 108150)
	closeTo(t, "与控制价差额", f.ControlDiff, -11850)
	if f.ControlDiff >= 0 {
		t.Fatal("低于控制价时差额应为负")
	}
	closeTo(t, "控制价利用率%", f.ControlUtilPct, 90.125)
}

// TestComposeProjectFeesZeroRateMeansNoFee 零值语义：费率 0 = 显式不取费，
// 不得回退默认值（实测产物里「规费不计列」「利润 0」是真实口径）。
func TestComposeProjectFeesZeroRateMeansNoFee(t *testing.T) {
	f := ComposeProjectFees(100, RateSet{}, FeePolicy{}, 0, 0, 0)
	closeTo(t, "直接费", f.DirectFee, 100)
	closeTo(t, "企管", f.ManagementFee, 0)
	closeTo(t, "规费", f.RegulatoryFee, 0)
	closeTo(t, "利润", f.ProfitFee, 0)
	closeTo(t, "增值税", f.TaxFee, 0)
	closeTo(t, "含税总造价（=直接费）", f.Total, 100)
}

// TestComposeProjectFeesMeasuresAndContingency 措施费/暂列金额进入税前合计基数。
func TestComposeProjectFeesMeasuresAndContingency(t *testing.T) {
	f := ComposeProjectFees(1000, RateSet{TaxRate: 0.09}, FeePolicy{}, 200, 300, 0)
	closeTo(t, "税前合计", f.PreTaxTotal, 1500)
	closeTo(t, "增值税", f.TaxFee, 135)
	closeTo(t, "含税总造价", f.Total, 1635)
}

// TestComposeLossRate 损耗率：含量 × 单价 × (1+损耗率)。沥青类材料按 3% 损耗。
func TestComposeLossRate(t *testing.T) {
	lines := []ComposeLine{
		{Kind: KindMaterial, Title: "改性沥青", Unit: "t", Quantity: 1, Price: 1000, LossRate: 0.03},
	}
	c := ComposeUnitPrice(lines)
	closeTo(t, "含损耗材料费", c.MaterialFee, 1030)
}

// TestComposeZeroLinesArePlaceholder 零含量/零单价是**合法占位**，不是错误：
// 实测模版保留「技术工/带班 ×0」这类计划资源行（工序模板列了但本项未用）。
// 改坏锚点：把 ZeroLines 计数改回告警，本用例即红。
func TestComposeZeroLinesArePlaceholder(t *testing.T) {
	lines := []ComposeLine{
		{Kind: KindMaterial, Title: "零价资源", Unit: "t", Quantity: 5, Price: 0},
		{Kind: KindLabor, Title: "零量资源", Unit: "工日", Quantity: 0, Price: 200},
		{Kind: KindMaterial, Title: "正常资源", Unit: "t", Quantity: 1, Price: 100},
	}
	c := ComposeUnitPrice(lines)
	closeTo(t, "小计（零行不计）", c.Subtotal, 100)
	if c.ZeroLines != 2 {
		t.Fatalf("零含量/零单价行应计 2 条，得到 %d", c.ZeroLines)
	}
	if len(c.Warnings) != 0 {
		t.Fatalf("零行是合法占位，不应告警：%v", c.Warnings)
	}
}

// TestComposeNegativeLineWarns 负值才是真非法，必须告警并排除出合计。
func TestComposeNegativeLineWarns(t *testing.T) {
	lines := []ComposeLine{
		{Kind: KindMaterial, Title: "负含量资源", Unit: "t", Quantity: -1, Price: 100},
		{Kind: KindMaterial, Title: "正常资源", Unit: "t", Quantity: 1, Price: 100},
	}
	c := ComposeUnitPrice(lines)
	closeTo(t, "小计（负行不计）", c.Subtotal, 100)
	if len(c.Warnings) != 1 {
		t.Fatalf("负含量应产生 1 条告警，得到 %v", c.Warnings)
	}
	if c.ZeroLines != 0 {
		t.Errorf("负值不应计入 ZeroLines，得到 %d", c.ZeroLines)
	}
}

// TestComposeUnknownKindNotSilentlyFolded 未知类别不静默并入三费——计入直接费
// 但留告警，并让「三费之和 ≠ 直接费」这一事实可被察觉。
func TestComposeUnknownKindNotSilentlyFolded(t *testing.T) {
	lines := []ComposeLine{{Kind: "人工+机械", Title: "合并段", Unit: "台班", Quantity: 1, Price: 500}}
	c := ComposeUnitPrice(lines)
	closeTo(t, "直接费（含未知类别）", c.Subtotal, 500)
	closeTo(t, "人工费", c.LaborFee, 0)
	if len(c.Warnings) == 0 {
		t.Fatal("未知类别应产生告警，而非静默归入某类")
	}
}

// ── 存储 + 核算联调 ─────────────────────────────────────────────────

// newWorkcostStore 建隔离的临时库并返回 store 与清理函数。
func newWorkcostStore(t *testing.T) (*Store, func()) {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase 返回 nil")
	}
	return Open(gdb), func() { db.CloseDatabase(dir) }
}

// TestResourceQuotaComposeFlow 全链路：建资源 → 建定额 → 核算单价；
// 并验证「资源调价后综合单价同步重算」（工料法的核心价值）。
func TestResourceQuotaComposeFlow(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	// ① 工料机资源：三类各一。
	labor, err := s.SaveResource(Resource{Kind: KindLabor, Title: "综合工日", Unit: "工日", BasePrice: 250, CurrentPrice: 265})
	if err != nil {
		t.Fatalf("保存人工资源失败: %v", err)
	}
	if labor.Code == "" {
		t.Fatal("资源应自动分配编码")
	}
	asphalt, err := s.SaveResource(Resource{Kind: KindMaterial, Title: "细粒式沥青混凝土 AC-13", Spec: "厚5cm", Unit: "t", BasePrice: 4000, CurrentPrice: 4200})
	if err != nil {
		t.Fatalf("保存材料资源失败: %v", err)
	}
	machine, err := s.SaveResource(Resource{Kind: KindMachine, Title: "沥青摊铺机", Unit: "台班", BasePrice: 4000, CurrentPrice: 4200})
	if err != nil {
		t.Fatalf("保存机械资源失败: %v", err)
	}

	// ② 消耗定额：每 t 沥青混合料的工料机消耗量。
	if _, err := s.SaveQuota(Quota{
		Code: "SZ-DL-001", Title: "细粒式沥青混凝土面层", Specialty: "市政", Chapter: "道路工程", Unit: "t",
		Items: []QuotaItem{
			{ResourceCode: labor.Code, Quantity: 0.048},
			{ResourceCode: asphalt.Code, Quantity: 1.0, LossRate: 0.03},
			{ResourceCode: machine.Code, Quantity: 0.0018},
		},
	}, nil); err != nil {
		t.Fatalf("保存定额失败: %v", err)
	}

	// ③ 核算：直接费 = 0.048×265 + 1×4200×1.03 + 0.0018×4200 = 12.72 + 4326 + 7.56
	c, _, err := s.ComposeQuota("SZ-DL-001", nil)
	if err != nil {
		t.Fatalf("核算失败: %v", err)
	}
	closeTo(t, "人工费", c.LaborFee, 12.72)
	closeTo(t, "材料费", c.MaterialFee, 4326)
	closeTo(t, "机械费", c.MachineFee, 7.56)
	closeTo(t, "直接费", c.Subtotal, 4346.28)

	// ④ 资源调价 → 同一定额的综合单价同步重算（无需改定额）。
	if _, err := s.UpdateResourcePrice(asphalt.ID, 4600, "2026年第3期", "成都", "到场价", "信息价", ""); err != nil {
		t.Fatalf("调价失败: %v", err)
	}
	c2, _, err := s.ComposeQuota("SZ-DL-001", nil)
	if err != nil {
		t.Fatalf("调价后核算失败: %v", err)
	}
	if c2.MaterialFee <= c.MaterialFee {
		t.Fatalf("资源涨价后材料费应上升：调价前 %.2f，调价后 %.2f", c.MaterialFee, c2.MaterialFee)
	}
	closeTo(t, "调价后材料费", c2.MaterialFee, 1*4600*1.03)
}

// TestComposeOverrideProjectLevel 项目级覆盖：覆盖含量与单价，逐项生效。
func TestComposeOverrideProjectLevel(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	labor, _ := s.SaveResource(Resource{Kind: KindLabor, Title: "综合工日", Unit: "工日", CurrentPrice: 200})
	if _, err := s.SaveQuota(Quota{
		Code: "Q-1", Title: "测试子目", Unit: "m³",
		Items: []QuotaItem{{ResourceCode: labor.Code, Quantity: 0.5}},
	}, nil); err != nil {
		t.Fatalf("保存定额失败: %v", err)
	}

	base, _, err := s.ComposeQuota("Q-1", nil)
	if err != nil {
		t.Fatalf("核算失败: %v", err)
	}
	closeTo(t, "全局标准", base.LaborFee, 100)

	// 项目级：含量调到 0.8、单价调到 260。
	ov := map[string]ComposeOverride{labor.Code: {Quantity: 0.8, Price: 260}}
	got, _, err := s.ComposeQuota("Q-1", ov)
	if err != nil {
		t.Fatalf("覆盖核算失败: %v", err)
	}
	closeTo(t, "项目覆盖后人工费", got.LaborFee, 208)

	// 只覆盖含量（单价传负=不覆盖），取全局现行价 200。
	onlyQty := map[string]ComposeOverride{labor.Code: {Quantity: 0.8, Price: -1}}
	got2, _, _ := s.ComposeQuota("Q-1", onlyQty)
	closeTo(t, "仅覆盖含量", got2.LaborFee, 160)
}

// TestDeleteResourceBlockedWhenReferenced 引用完整性：被定额引用的资源不许删。
func TestDeleteResourceBlockedWhenReferenced(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	r, _ := s.SaveResource(Resource{Kind: KindMaterial, Title: "水泥", Unit: "t", CurrentPrice: 400})
	if _, err := s.SaveQuota(Quota{Code: "Q-2", Title: "水泥砂浆", Unit: "m³",
		Items: []QuotaItem{{ResourceCode: r.Code, Quantity: 0.3}}}, nil); err != nil {
		t.Fatalf("保存定额失败: %v", err)
	}
	if err := s.DeleteResource(r.ID, false); err == nil {
		t.Fatal("被引用的资源不应被允许删除")
	}
	if err := s.DeleteResource(r.ID, true); err != nil {
		t.Fatalf("force 删除应成功: %v", err)
	}
	// 孤立引用：核算仍可进行，但留告警（不静默归零）。
	c, _, err := s.ComposeQuota("Q-2", nil)
	if err != nil {
		t.Fatalf("资源删除后核算仍应可用: %v", err)
	}
	if len(c.Warnings) == 0 {
		t.Fatal("资源已删除应产生告警")
	}
}

// TestSaveResourceIdentUpsert 资源身份唯一：同 (类别,名称,规格,单位) 再保存
// 走更新，不制造重复资源；改价不改身份。
func TestSaveResourceIdentUpsert(t *testing.T) {
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	a, _ := s.SaveResource(Resource{Kind: KindMaterial, Title: "中砂", Spec: "中粗", Unit: "m³", CurrentPrice: 120})
	b, err := s.SaveResource(Resource{Kind: KindMaterial, Title: "中砂", Spec: "中粗", Unit: "m³", CurrentPrice: 135})
	if err != nil {
		t.Fatalf("二次保存失败: %v", err)
	}
	if a.ID != b.ID || a.Code != b.Code {
		t.Fatalf("同身份资源应复用 id/code：a=(%d,%s) b=(%d,%s)", a.ID, a.Code, b.ID, b.Code)
	}
	list, _ := s.ListResources(KindMaterial, "中砂")
	if len(list) != 1 {
		t.Fatalf("同身份资源应只有一条，得到 %d 条", len(list))
	}
	if list[0].CurrentPrice != 135 {
		t.Fatalf("二次保存应更新现行价，得到 %v", list[0].CurrentPrice)
	}
}

// TestEffectivePriceFallback 现行价为 0 时回退基准价（未录现行价不等于免费）。
func TestEffectivePriceFallback(t *testing.T) {
	if got := (Resource{BasePrice: 300}).EffectivePrice(); got != 300 {
		t.Errorf("现行价缺失应回退基准价，得到 %v", got)
	}
	if got := (Resource{BasePrice: 300, CurrentPrice: 350}).EffectivePrice(); got != 350 {
		t.Errorf("现行价优先，得到 %v", got)
	}
}

// TestNormalizeKind 类别归一容错。
func TestNormalizeKind(t *testing.T) {
	cases := map[string]string{
		"人工费": KindLabor, "工日": KindLabor, "普工": KindLabor,
		"材料费": KindMaterial, "主材": KindMaterial, "": KindMaterial,
		"机械费": KindMachine, "台班": KindMachine, "机械台班": KindMachine,
	}
	for in, want := range cases {
		if got := normalizeKind(in); got != want {
			t.Errorf("normalizeKind(%q) = %q, want %q", in, got, want)
		}
	}
}
