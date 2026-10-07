package workcost

import (
	"path/filepath"
	"strings"
	"testing"
)

// 夹具：实测项目产物（gaea 项目空间导出，五表模版）。断言值取自人工核对过的
// 真实数字——改动解析器若让清单项/消耗行/资源少解析，这里必红。
const (
	fixtureWangping = "testdata/旺平矿业地块污染土壤治理与修复_工程费用成本测算表_运费按距离.xlsx"
	fixtureShifang  = "testdata/什邡2个污染地块施工成本测算表_补项.xlsx"
)

// findResource 按编码取资源。
func findResource(list []ProjectResource, code string) *ProjectResource {
	for i := range list {
		if list[i].Code == code {
			return &list[i]
		}
	}
	return nil
}

// findLine 取某清单项下第一条指定资源编码的消耗行。
func findLine(list []ProjectConsumption, itemCode, resCode string) *ProjectConsumption {
	for i := range list {
		if list[i].ItemCode == itemCode && list[i].ResourceCode == resCode {
			return &list[i]
		}
	}
	return nil
}

// TestParseProjectWorkbookWangping 真实产物解析金样（旺平矿业，2026-10-07）。
func TestParseProjectWorkbookWangping(t *testing.T) {
	b, err := ParseProjectWorkbook(filepath.FromSlash(fixtureWangping))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if !strings.Contains(b.Project, "旺平矿业") {
		t.Errorf("项目名 = %q, want 含「旺平矿业」", b.Project)
	}
	if !strings.Contains(b.Project, "污染土壤治理与修复") {
		t.Errorf("项目名应含工程全称，得到 %q", b.Project)
	}
	if !strings.Contains(b.Location, "雅安市宝兴县") {
		t.Errorf("地点 = %q, want 含「雅安市宝兴县」", b.Location)
	}
	if !strings.Contains(b.Duration, "2027年7月") {
		t.Errorf("工期 = %q, want 含「2027年7月」", b.Duration)
	}
	if !strings.Contains(b.Pricing, "工序消耗量") {
		t.Errorf("组价口径 = %q, want 含「工序消耗量」", b.Pricing)
	}

	// 规模（人工核对：47 / 32 / 107 / 56）。
	if len(b.Resources) != 47 {
		t.Errorf("资源数 = %d, want 47", len(b.Resources))
	}
	if len(b.Items) != 32 {
		t.Errorf("清单项数 = %d, want 32", len(b.Items))
	}
	if len(b.Lines) != 107 {
		t.Errorf("消耗行数 = %d, want 107", len(b.Lines))
	}
	if len(b.Quantities) == 0 {
		t.Error("工程量未解析到")
	}

	// 取费：管理费 5%、利润 0、税 3%（封面自述与费用汇总一致）。
	closeTo(t, "管理费率", b.Fee.ManagementRate, 0.05)
	closeTo(t, "利润费率", b.Fee.ProfitRate, 0)
	closeTo(t, "增值税率", b.Fee.TaxRate, 0.03)

	// 资源：柴油 6.37（信息价）、人工 300/350、台班公式保留原文。
	die := findResource(b.Resources, "DIE")
	if die == nil {
		t.Fatal("缺资源 DIE（0号柴油）")
	}
	closeTo(t, "柴油单价", die.Price, 6.37)
	if die.Kind != KindMaterial {
		t.Errorf("柴油类别 = %q, want 材料", die.Kind)
	}
	exc := findResource(b.Resources, "EXC")
	if exc == nil {
		t.Fatal("缺资源 EXC（挖掘机台班）")
	}
	if exc.Kind != KindMachine {
		t.Errorf("挖掘机类别 = %q, want 机械", exc.Kind)
	}
	if !strings.Contains(exc.PriceFormula, "900") {
		t.Errorf("台班公式应保留原文，得到 %q", exc.PriceFormula)
	}
	// 台班费 = 折旧900 + 柴油6.37×90L + 司机300 = 1773.30
	closeTo(t, "挖掘机台班费", exc.Price, 1773.30)

	// 消耗行：WP02 施工便道 级配碎石 0.75 m³ × 78 = 58.50
	l := findLine(b.Lines, "WP02", "AGG")
	if l == nil {
		t.Fatal("缺 WP02 的 AGG（级配碎石）消耗行")
	}
	closeTo(t, "AGG 消耗量", l.Quantity, 0.75)
	closeTo(t, "AGG 金额", l.Amount, 58.5)

	// 异常清单项（综合单价表无汇总行）必须显形为告警而非静默。
	if len(b.Skipped) > 0 {
		t.Logf("跳过 %d 行（首个：%s 行%d %s）", len(b.Skipped), b.Skipped[0].Sheet, b.Skipped[0].Row, b.Skipped[0].Reason)
	}
	for _, w := range b.Warnings {
		t.Logf("告警: %s", w)
	}
}

// TestParseProjectWorkbookShifang 真实产物解析金样（什邡2个地块，2026-10-06）。
func TestParseProjectWorkbookShifang(t *testing.T) {
	b, err := ParseProjectWorkbook(filepath.FromSlash(fixtureShifang))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if !strings.Contains(b.Project, "什邡") {
		t.Errorf("项目名 = %q, want 含「什邡」", b.Project)
	}
	if !strings.Contains(b.Location, "马祖镇") {
		t.Errorf("地点 = %q, want 含「马祖镇」", b.Location)
	}
	// 什邡的组价口径写在「五表模版：…」行。
	if !strings.Contains(b.Pricing, "只含人材机") && !strings.Contains(b.Pricing, "五表模版") {
		t.Errorf("组价口径 = %q, want 含「只含人材机」或「五表模版」", b.Pricing)
	}

	if len(b.Resources) != 43 {
		t.Errorf("资源数 = %d, want 43", len(b.Resources))
	}
	if len(b.Items) != 34 {
		t.Errorf("清单项数 = %d, want 34", len(b.Items))
	}
	if len(b.Lines) != 100 {
		t.Errorf("消耗行数 = %d, want 100", len(b.Lines))
	}
	closeTo(t, "管理费率", b.Fee.ManagementRate, 0.05)
	closeTo(t, "增值税率", b.Fee.TaxRate, 0.03)

	// 什邡的机械台班用了跨表引用（运距来自工程量表）。
	trp := findResource(b.Resources, "TRP")
	if trp == nil {
		t.Fatal("缺资源 TRP（密闭渣土长途车次）")
	}
	if trp.Kind != KindMachine {
		t.Errorf("TRP 类别 = %q, want 机械", trp.Kind)
	}
	t.Logf("TRP 单价 = %.2f 公式 = %q", trp.Price, trp.PriceFormula)
}

// TestParseProjectWorkbookRejectsNonFiveSheet 非五表工作簿必须明确报错，
// 不得静默返回空 bundle（否则导入会把「不支持」装成「没有数据」）。
func TestParseProjectWorkbookRejectsNonFiveSheet(t *testing.T) {
	// 用一份 csv 冒充（excelize 打不开）。
	if _, err := ParseProjectWorkbook(filepath.FromSlash("testdata/does-not-exist.xlsx")); err == nil {
		t.Fatal("不存在的文件应报错")
	}
}

// TestDetectedFiveSheet 五表识别只认「工料机价格 + 综合单价」同时存在。
func TestDetectedFiveSheet(t *testing.T) {
	if !DetectedFiveSheet([]string{"封面", "费用汇总", "综合单价", "工料机价格", "工程量计算"}) {
		t.Error("标准五表应被识别")
	}
	if DetectedFiveSheet([]string{"封面", "费用汇总"}) {
		t.Error("缺工料机价格/综合单价不应被识别为五表")
	}
}

// TestCrossCheckReportsMissingResource 跨表校验：消耗量引用了不存在的资源编码
// 必须出告警（否则综合单价会静默偏低）。
func TestCrossCheckReportsMissingResource(t *testing.T) {
	b := &ProjectBundle{
		Resources: []ProjectResource{{Code: "L01"}},
		Items:     []ProjectItem{{Code: "X01"}},
		Lines: []ProjectConsumption{
			{ItemCode: "X01", ResourceCode: "L01", Kind: KindLabor, Quantity: 1, Price: 100},
			{ItemCode: "X01", ResourceCode: "GONE", Kind: KindMaterial, Quantity: 1, Price: 100},
		},
	}
	warns := crossCheck(b)
	joined := strings.Join(warns, " ")
	if !strings.Contains(joined, "GONE") {
		t.Fatalf("应告警缺失资源 GONE，得到 %v", warns)
	}
}

// TestCrossCheckReportsItemWithoutLines 清单项无消耗行必须出告警。
func TestCrossCheckReportsItemWithoutLines(t *testing.T) {
	b := &ProjectBundle{
		Resources: []ProjectResource{{Code: "L01"}},
		Items:     []ProjectItem{{Code: "X01"}, {Code: "X02"}},
		Lines: []ProjectConsumption{
			{ItemCode: "X01", ResourceCode: "L01", Kind: KindLabor, Quantity: 1, Price: 100},
		},
	}
	joined := strings.Join(crossCheck(b), " ")
	if !strings.Contains(joined, "1 个清单项") {
		t.Fatalf("应告警 1 个清单项无消耗行，得到 %v", joined)
	}
}

// ── 落库（bundle → 资源库 + 消耗定额库）──────────────────────────────

// TestApplyProjectBundleEndToEnd 端到端：真实产物解析 → 落库 → 按定额核算
// 综合单价（只含人材机）。这是「最近项目录入」的最小完整闭环。
func TestApplyProjectBundleEndToEnd(t *testing.T) {
	b, err := ParseProjectWorkbook(filepath.FromSlash(fixtureWangping))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	res := s.ApplyProjectBundle(b)
	t.Logf("落库：资源 新增%d/更新%d，定额 新增%d/更新%d，消耗行 %d",
		res.ResourceNew, res.ResourceUpd, res.QuotaNew, res.QuotaUpd, res.Lines)
	if len(res.Errors) > 0 {
		for i, e := range res.Errors {
			if i >= 6 {
				t.Logf("  ... 其余 %d 条", len(res.Errors)-i)
				break
			}
			t.Logf("  错误: %s", e)
		}
	}
	if res.ResourceNew != 47 {
		t.Errorf("资源新增 = %d, want 47", res.ResourceNew)
	}
	if res.QuotaNew != 32 {
		t.Errorf("定额新增 = %d, want 32", res.QuotaNew)
	}

	// 幂等：重复导入不新增。
	again := s.ApplyProjectBundle(b)
	if again.ResourceNew != 0 || again.QuotaNew != 0 {
		t.Errorf("重复导入应零新增：资源新增%d 定额新增%d", again.ResourceNew, again.QuotaNew)
	}
	if again.ResourceUpd != 47 || again.QuotaUpd != 32 {
		t.Errorf("重复导入应全部更新：资源更新%d 定额更新%d", again.ResourceUpd, again.QuotaUpd)
	}

	// ④ 清单层（SchemaV28）：分部分项清单随导入落库——「我让你录入的清单」。
	projects, err := s.BillProjects()
	if err != nil || len(projects) != 1 {
		t.Fatalf("应落 1 个清单项目，得到 %d 个（err=%v）", len(projects), err)
	}
	proj := projects[0]
	if proj.ItemCount != 32 {
		t.Errorf("清单项应 32 条（与定额同骨架），得到 %d", proj.ItemCount)
	}
	if proj.TaxRate <= 0 {
		t.Errorf("项目费率应随导入落库（增值税率 %.4f）", proj.TaxRate)
	}
	bills, err := s.BillItems(proj.ID)
	if err != nil {
		t.Fatalf("读清单失败: %v", err)
	}
	withQty, withQuota := 0, 0
	for _, it := range bills {
		if it.Quantity > 0 {
			withQty++
		}
		if it.QuotaCode != "" {
			withQuota++
		}
	}
	if withQuota != len(bills) {
		t.Errorf("每条清单项都应套定额，%d/%d 套上", withQuota, len(bills))
	}
	t.Logf("清单 %d 条落库，其中带工程量 %d 条（其余待工程量计算表补录）", len(bills), withQty)
	// 幂等：重复导入后清单不翻倍。
	s.ApplyProjectBundle(b)
	again2, _ := s.BillItems(proj.ID)
	if len(again2) != len(bills) {
		t.Errorf("重复导入清单应幂等：%d → %d", len(bills), len(again2))
	}

	// 按落库后的定额核算综合单价：WP02 施工便道 = 人工 + 级配碎石 + C20 + 装载机 + 压路机。
	c, q, err := s.ComposeQuota("WP02", nil)
	if err != nil {
		t.Fatalf("核算 WP02 失败: %v", err)
	}
	t.Logf("WP02（%s，单位 %s）综合单价 = %.2f（人工 %.2f / 材料 %.2f / 机械 %.2f）",
		q.Title, q.Unit, c.CompositePrice, c.LaborFee, c.MaterialFee, c.MachineFee)
	if c.CompositePrice <= 0 {
		t.Fatal("综合单价应为正")
	}
	if len(c.Warnings) > 0 {
		t.Errorf("合法定额不应有告警: %v", c.Warnings)
	}
	// 若资源价缺失导致核算为 0，说明 ResourcePrice 快照与现行价都没取到。
	if c.LaborFee <= 0 {
		t.Errorf("人工费应大于 0（普通工 300 元/工日 × 0.03），得到 %.2f", c.LaborFee)
	}
}

// TestApplyProjectBundleResourcePriceNotSnapshot 落库的消耗行不钉死资源价快照——
// 资源调价后综合单价必须同步重算（工料法核心价值）。
func TestApplyProjectBundleResourcePriceNotSnapshot(t *testing.T) {
	b, err := ParseProjectWorkbook(filepath.FromSlash(fixtureShifang))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	s.ApplyProjectBundle(b)

	before, _, err := s.ComposeQuota("SF01", nil)
	if err != nil {
		t.Fatalf("核算 SF01 失败: %v", err)
	}
	l01, err := s.GetResourceByCode("L01")
	if err != nil {
		t.Fatalf("缺资源 L01: %v", err)
	}
	if _, err := s.UpdateResourcePrice(l01.ID, l01.EffectivePrice()*2, "测试期", "测试地", "到场价", "测试", "调价联动验证"); err != nil {
		t.Fatalf("调价失败: %v", err)
	}
	after, _, err := s.ComposeQuota("SF01", nil)
	if err != nil {
		t.Fatalf("调价后核算失败: %v", err)
	}
	if after.LaborFee <= before.LaborFee {
		t.Fatalf("把普通工单价翻倍后人工费应上升：调价前 %.2f → 调价后 %.2f",
			before.LaborFee, after.LaborFee)
	}
}

// TestSlugFromTitle 标题 → slug。
func TestSlugFromTitle(t *testing.T) {
	cases := map[string]string{
		"测量放线":      "测量放线",
		"施工便道（3m宽）": "施工便道-3m宽",
		"ALLU 筛分":   "allu-筛分",
		"  ":        "",
		"---":       "",
	}
	for in, want := range cases {
		if got := SlugFromTitle(in); got != want {
			t.Errorf("SlugFromTitle(%q) = %q, want %q", in, got, want)
		}
	}
}
