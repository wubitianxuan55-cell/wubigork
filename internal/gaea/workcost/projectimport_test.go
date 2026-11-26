package workcost

import (
	"math"
	"path/filepath"
	"strings"
	"testing"
)

// 夹具：实测项目产物（gaea 办公代理按三级树表样表填出的测算簿，2026-11-27）。
// 断言值取自工作簿缓存值与人工可复核的口算——改动解析器若让清单项 / 定额块 /
// 消耗行少解析，这里必红。旧「行类」版式的旺平 / 什邡夹具随版式退役一并删除
// （2026-11-27 用户拍板：以后不再上传旧版式的表）。
const fixtureYanjiang = "testdata/沿江包1成本测算表_三级树表.xlsx"

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

// findItem 按编码取清单项。
func findItem(list []ProjectItem, code string) *ProjectItem {
	for i := range list {
		if list[i].Code == code {
			return &list[i]
		}
	}
	return nil
}

// TestParseProjectWorkbookYanjiang 真实产物解析金样（沿江包1，三级树表，2026-11-27）。
// 断言值取自工作簿缓存值与人工可复核的口算——改动解析器若让清单项 / 定额块 /
// 消耗行少解析，这里必红。
func TestParseProjectWorkbookYanjiang(t *testing.T) {
	b, err := ParseProjectWorkbook(filepath.FromSlash(fixtureYanjiang))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	// 封面：顶部 B2 是书名带（「…· 施工成本测算书」），项目名取「项目名称」标签值；
	// 标签（B:C）与值（D:J）之间隔着细分隔列，地点 / 工期也要配得上对。
	if b.Project != "沿江包1（鑫乐洋地块）土壤及地下水修复成本测算" {
		t.Errorf("项目名 = %q", b.Project)
	}
	if !strings.Contains(b.Location, "重庆市大足区雍溪镇") {
		t.Errorf("地点 = %q, want 含「重庆市大足区雍溪镇」", b.Location)
	}
	if !strings.Contains(b.Duration, "143 日历天") {
		t.Errorf("工期 = %q, want 含「143 日历天」", b.Duration)
	}
	if b.CoverSpecialty != "土壤修复" || b.CoverPeriod != "2026-12" {
		t.Errorf("封面口径自述 = 专业 %q / 计价期 %q, want 土壤修复 / 2026-12（合写标签「专业 / 计价期」）",
			b.CoverSpecialty, b.CoverPeriod)
	}

	// 规模（工作簿逐行核对：清单 33 / 工料机 41 / 定额块 35 / 消耗行 68）。
	if len(b.Resources) != 41 {
		t.Errorf("资源数 = %d, want 41", len(b.Resources))
	}
	if len(b.Items) != 33 {
		t.Errorf("清单项数 = %d, want 33", len(b.Items))
	}
	if len(b.QuotaBlocks) != 35 {
		t.Errorf("定额块数 = %d, want 35", len(b.QuotaBlocks))
	}
	if len(b.Lines) != 68 {
		t.Errorf("消耗行数 = %d, want 68", len(b.Lines))
	}
	if len(b.Skipped) != 0 {
		t.Errorf("实填表不应有被跳过的行: %+v", b.Skipped)
	}
	if len(b.Warnings) != 0 {
		t.Errorf("实填表不应有告警: %v", b.Warnings)
	}

	// 取费：企管 5%、利润与规费不计取、增值税 3%；各段金额是重算后的缓存值。
	closeTo(t, "管理费率", b.Fee.ManagementRate, 0.05)
	closeTo(t, "利润费率", b.Fee.ProfitRate, 0)
	closeTo(t, "规费费率", b.Fee.RegulatoryRate, 0)
	closeTo(t, "增值税率", b.Fee.TaxRate, 0.03)
	if !b.Fee.HasAmounts {
		t.Fatal("费用汇总有缓存金额，HasAmounts 应为 true")
	}
	closeTo(t, "直接费", b.Fee.DirectAmount, 6242754.43)
	closeTo(t, "管理费", b.Fee.ManagementAmount, 312137.72)
	closeTo(t, "税前合计", b.Fee.PreTaxAmount, 6554892.15)
	closeTo(t, "含税总造价", b.Fee.TotalAmount, 6751538.91)

	// 三级树表逐行核对（人工可复核的口算）：
	//   AC01 污染土壤分层清挖 = 挖机 0.012×1773.30 + 普工 0.05×300 = 21.28 + 15 = 36.28
	//   AC03 污染土壤筛分预处理 = 两条定额：ALLU 0.02×800 = 16；挖机 0.008×1773.30 = 14.19 → 30.19
	// 实填表是变长组（AC01 一块两行明细、AC02 一块一行、AC03 两块各一行）——
	// 按固定 11 行网格推行号的实现在这里必错。
	byCode := map[string]*ProjectItem{}
	for i := range b.Items {
		byCode[b.Items[i].Code] = &b.Items[i]
	}
	for code, want := range map[string]float64{"AC01": 36.28, "AC02": 6, "AC03": 30.19, "AC04": 29509.83} {
		it := byCode[code]
		if it == nil {
			t.Fatalf("缺清单项 %s", code)
		}
		closeTo(t, code+" 原表综合单价", it.ListPrice, want)
		if it.Quantity <= 0 || it.Unit == "" || it.Division == "" {
			t.Errorf("%s 应带工程量 / 单位 / 分部，实得 %+v", code, it)
		}
		if !strings.HasPrefix(it.ListPriceExpr, "ROUND(H") {
			t.Errorf("%s 综合单价计算式应取清单行 G 列公式原文，得到 %q", code, it.ListPriceExpr)
		}
	}
	if it := byCode["AC01"]; it.ListPriceExpr != "ROUND(H4/F4,2)" {
		t.Errorf("AC01 综合单价计算式 = %q, want ROUND(H4/F4,2)", it.ListPriceExpr)
	}
	blocksOf := func(item string) []ProjectQuotaBlock {
		var out []ProjectQuotaBlock
		for _, blk := range b.QuotaBlocks {
			if blk.ItemCode == item {
				out = append(out, blk)
			}
		}
		return out
	}
	ac03 := blocksOf("AC03")
	if len(ac03) != 2 || ac03[0].Code != "V-BJ19A" || ac03[1].Code != "V-BJ19B" || ac03[0].Title != "污染土壤筛分破碎" || ac03[0].Unit == "" {
		t.Errorf("AC03 应挂两条定额 V-BJ19A / V-BJ19B（带定额自己的名称与单位），实得 %+v", ac03)
	}
	if ac01 := blocksOf("AC01"); len(ac01) != 1 || ac01[0].Code != "V-BJ17A" || ac01[0].Factor != 1 {
		t.Errorf("AC01 应挂一条定额 V-BJ17A（系数 1），实得 %+v", ac01)
	}
	// 明细行：名称 / 单位在表里是 VLOOKUP 取价公式，解析要取缓存值。
	l1 := findLine(b.Lines, "AC01", "M001")
	if l1 == nil {
		t.Fatal("缺 AC01 的 M001（挖掘机台班）消耗行")
	}
	closeTo(t, "M001 消耗量", l1.Quantity, 0.012)
	closeTo(t, "M001 金额", l1.Amount, 21.28)
	if l1.Kind != KindMachine || l1.ResourceTitle != "挖掘机1.0m³台班" || l1.Unit != "台班" || l1.QuotaCode != "V-BJ17A" {
		t.Errorf("M001 消耗行 = %+v", *l1)
	}
	if l2 := findLine(b.Lines, "AC01", "L004"); l2 == nil || l2.Kind != KindLabor || l2.Amount != 15 {
		t.Errorf("AC01 的 L004 消耗行 = %+v", l2)
	}
	// 含量读的是存储值：活性炭 0.00015、PAM 0.00002。含量列格式是 0.####（最多
	// 四位小数），按显示值读会变成 0.0001 / 0——实测「废水处理运行」表里 34.70、
	// 读进库 33.91。
	if l := findLine(b.Lines, "AC14", "P019"); l == nil || l.Quantity != 0.00015 || l.Amount != 1.94 {
		t.Errorf("P019 含量应读存储值 0.00015，实得 %+v", l)
	}
	if l := findLine(b.Lines, "AC14", "P020"); l == nil || l.Quantity != 0.00002 || l.Amount != 0.14 {
		t.Errorf("P020 含量应读存储值 0.00002，实得 %+v", l)
	}
	// 每条消耗行都必须挂在某个定额块下（三级树表的人材机不能悬空）。
	blockKeys := map[string]bool{}
	for _, blk := range b.QuotaBlocks {
		blockKeys[blk.ItemCode+"\x00"+blk.Code] = true
	}
	for _, l := range b.Lines {
		if !blockKeys[l.ItemCode+"\x00"+l.QuotaCode] {
			t.Errorf("消耗行 %s / %s 没有挂在所属清单的定额块上（quota=%q）", l.ItemCode, l.ResourceCode, l.QuotaCode)
		}
	}

	// 复算对表：按工作簿口径从解析结果重算每条清单综合单价（明细舍分 → 定额
	// 舍分 → Σ(定额×系数) 再舍分），与表里 G 列缓存值逐条比；合价合计对直接费。
	r2 := func(v float64) float64 { return math.Round(v*100) / 100 }
	quotaH, factor := map[string]float64{}, map[string]float64{}
	for _, blk := range b.QuotaBlocks {
		f := blk.Factor
		if f <= 0 {
			f = 1
		}
		factor[blk.ItemCode+"\x00"+blk.Code] = f
	}
	for _, l := range b.Lines {
		quotaH[l.ItemCode+"\x00"+l.QuotaCode] += r2(l.Quantity * l.Price)
	}
	sum := 0.0
	for _, it := range b.Items {
		h := 0.0
		for key, v := range quotaH {
			if strings.HasPrefix(key, it.Code+"\x00") {
				h += r2(v) * factor[key]
			}
		}
		closeTo(t, it.Code+" 复算综合单价", r2(h), it.ListPrice)
		sum += r2(it.ListPrice * it.Quantity)
	}
	closeTo(t, "Σ(综合单价×工程量) 应等于费用汇总直接费", sum, b.Fee.DirectAmount)
}

// TestParseProjectWorkbookRejectsNonFiveSheet 非五表工作簿必须明确报错，
// 不得静默返回空 bundle（否则导入会把「不支持」装成「没有数据」）。
func TestParseProjectWorkbookRejectsNonFiveSheet(t *testing.T) {
	// 用一份 csv 冒充（excelize 打不开）。
	if _, err := ParseProjectWorkbook(filepath.FromSlash("testdata/does-not-exist.xlsx")); err == nil {
		t.Fatal("不存在的文件应报错")
	}
}

// TestIsEstimateWorkbook 测算簿的判据：有「综合单价」表就算（2026-11-27 用户拍板：
// 导入的硬契约只有综合单价表，其余表是参考——清单、工料机价格缺了也认）。此前要求
// 三张表同在，少一张整份拒收，清单表一按单项工程改版式工程量就整批读成 0。
func TestIsEstimateWorkbook(t *testing.T) {
	if !IsEstimateWorkbook([]string{"综合单价"}) {
		t.Fatal("只有综合单价表的测算簿应被认出")
	}
	if !IsEstimateWorkbook([]string{"封面", "分部分项清单", "取费表", "综合单价分析表", "工料机价格表", "工程量计算"}) {
		t.Fatal("表名带前后缀的测算簿应被认出")
	}
	if IsEstimateWorkbook([]string{"封面", "清单", "费用汇总", "工料机价格", "工程量计算"}) {
		t.Fatal("没有综合单价表不算测算簿")
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
	b, err := ParseProjectWorkbook(filepath.FromSlash(fixtureYanjiang))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	s, cleanup := newWorkcostStore(t)
	defer cleanup()

	res := s.ApplyProjectBundle(b)
	for _, e := range res.Errors {
		t.Errorf("落库出错: %s", e)
	}
	// 35 个定额块落成 33 条定额：V-66（废水处理运行）、V-70（固废密闭运输）各被
	// 两个清单共用，同码同内容复用同一条。
	if res.ResourceNew != 41 || res.QuotaNew != 33 || res.QuotaUpd != 2 || res.Lines != 68 {
		t.Errorf("落库：资源新增%d / 定额新增%d、更新%d / 消耗行%d，want 41 / 33、2 / 68",
			res.ResourceNew, res.QuotaNew, res.QuotaUpd, res.Lines)
	}

	// 幂等：重复导入不新增。
	again := s.ApplyProjectBundle(b)
	if len(again.Errors) > 0 {
		t.Errorf("重复导入不应出错: %v", again.Errors)
	}
	if again.ResourceNew != 0 || again.QuotaNew != 0 {
		t.Errorf("重复导入应零新增：资源新增%d 定额新增%d", again.ResourceNew, again.QuotaNew)
	}
	if again.ResourceUpd != 41 || again.QuotaUpd != 35 {
		t.Errorf("重复导入应全部更新：资源更新%d 定额更新%d", again.ResourceUpd, again.QuotaUpd)
	}

	// ④ 清单层（SchemaV28）：分部分项清单随导入落库——「我让你录入的清单」。
	projects, err := s.BillProjects()
	if err != nil || len(projects) != 1 {
		t.Fatalf("应落 1 个清单项目，得到 %d 个（err=%v）", len(projects), err)
	}
	proj := projects[0]
	if proj.Name != "沿江包1（鑫乐洋地块）土壤及地下水修复成本测算" {
		t.Errorf("项目名 = %q", proj.Name)
	}
	if proj.ItemCount != 33 {
		t.Errorf("清单项应 33 条，得到 %d", proj.ItemCount)
	}
	closeTo(t, "项目增值税率", proj.TaxRate, 0.03)
	bills, err := s.BillItems(proj.ID)
	if err != nil {
		t.Fatalf("读清单失败: %v", err)
	}
	var ac01Bill *BillItem
	for i := range bills {
		it := &bills[i]
		if it.Quantity <= 0 || it.QuotaCode == "" || it.ListPrice <= 0 {
			t.Errorf("清单项 %s 应带工程量 / 定额 / 原表综合单价快照，实得 %+v", it.Code, *it)
		}
		if it.Code == "AC01" {
			ac01Bill = it
		}
	}
	if ac01Bill == nil {
		t.Fatal("落库清单缺 AC01")
	}
	closeTo(t, "落库 AC01 原表综合单价", ac01Bill.ListPrice, 36.28)
	// 一清单多定额：AC03 的两条定额按 1:N 落挂接，系数 1 → 挂接量 = 清单工程量。
	var ac03 *BillItem
	for i := range bills {
		if bills[i].Code == "AC03" {
			ac03 = &bills[i]
		}
	}
	links, err := s.BillQuotaLinks(ac03.ID)
	if err != nil {
		t.Fatalf("读 AC03 挂接失败: %v", err)
	}
	if len(links) != 2 || links[0].QuotaCode != "V-BJ19A" || links[1].QuotaCode != "V-BJ19B" {
		t.Fatalf("AC03 应挂两条定额，实得 %+v", links)
	}
	for _, lk := range links {
		closeTo(t, "AC03 挂接 "+lk.QuotaCode+" 的工程量", lk.Quantity, ac03.Quantity)
	}
	// 共用定额的两个清单：库里那一条的现算单价要同时对得上两个清单的原表综合单价。
	for _, shared := range []struct{ quota, a, b string }{{"V-66", "AC14", "AC16"}, {"V-70", "AC06", "AC09"}} {
		c, _, err := s.ComposeQuota(shared.quota, nil)
		if err != nil {
			t.Fatalf("核算 %s 失败: %v", shared.quota, err)
		}
		for _, code := range []string{shared.a, shared.b} {
			for _, it := range bills {
				if it.Code == code {
					closeTo(t, shared.quota+" 现算 vs "+code+" 原表综合单价", c.CompositePrice, it.ListPrice)
				}
			}
		}
	}

	// 幂等：重复导入后清单不翻倍，快照随重导入刷新（改簿重导=口径可更新）。
	s.ApplyProjectBundle(b)
	again2, _ := s.BillItems(proj.ID)
	if len(again2) != len(bills) {
		t.Errorf("重复导入清单应幂等：%d → %d", len(bills), len(again2))
	}

	// 按落库后的定额核算综合单价：V-BJ17A 污染土壤分层清挖 = 挖机 21.28 + 普工 15。
	c, q, err := s.ComposeQuota("V-BJ17A", nil)
	if err != nil {
		t.Fatalf("核算 V-BJ17A 失败: %v", err)
	}
	t.Logf("V-BJ17A（%s，单位 %s）综合单价 = %.2f（人工 %.2f / 材料 %.2f / 机械 %.2f）",
		q.Title, q.Unit, c.CompositePrice, c.LaborFee, c.MaterialFee, c.MachineFee)
	closeTo(t, "V-BJ17A 人工费", c.LaborFee, 15)
	closeTo(t, "V-BJ17A 机械费", c.MachineFee, 21.28)
	closeTo(t, "V-BJ17A 综合单价（应与工作簿 AC01 的 36.28 逐分一致）", c.CompositePrice, 36.28)
	if len(c.Warnings) > 0 {
		t.Errorf("合法定额不应有告警: %v", c.Warnings)
	}
}

// TestApplyProjectBundleResourcePriceNotSnapshot 落库的消耗行不钉死资源价快照——
// 资源调价后综合单价必须同步重算（工料法核心价值）。
func TestApplyProjectBundleResourcePriceNotSnapshot(t *testing.T) {
	b, err := ParseProjectWorkbook(filepath.FromSlash(fixtureYanjiang))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	s.ApplyProjectBundle(b)

	before, _, err := s.ComposeQuota("V-BJ17A", nil)
	if err != nil {
		t.Fatalf("核算 V-BJ17A 失败: %v", err)
	}
	l004, err := s.GetResourceByCode("L004")
	if err != nil {
		t.Fatalf("缺资源 L004: %v", err)
	}
	if _, err := s.UpdateResourcePrice(l004.ID, l004.EffectivePrice()*2, "2026-12", "", "到场价", "测试", "调价联动验证"); err != nil {
		t.Fatalf("调价失败: %v", err)
	}
	after, _, err := s.ComposeQuota("V-BJ17A", nil)
	if err != nil {
		t.Fatalf("调价后核算失败: %v", err)
	}
	// 普工 300 → 600：人工费 0.05×300 = 15 → 30，综合单价 36.28 → 51.28。
	closeTo(t, "调价前人工费", before.LaborFee, 15)
	closeTo(t, "调价后人工费", after.LaborFee, 30)
	closeTo(t, "调价后综合单价", after.CompositePrice, 51.28)
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
