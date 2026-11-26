package workcost

// projectimport_tree_test.go —— 三级树表解析用例（合成工作簿，2026-11-27）。
//
// 真实产物的金样在 projectimport_test.go（沿江包1）；这里钉的是金样里没出现的
// 形态：换算系数 ≠1 / 留空、漏填层级、缺编码、重复清单、非三级树表拒收、封面
// 项目名两处写法。旧「行类」版式的用例随版式退役删除（用户拍板以后不再上传）。

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// treeUnitPriceHeader 三级树表综合单价的表头（样表 v5.5 逐字）。
var treeUnitPriceHeader = []any{"序号", "层级", "编码", "名称", "单位", "工程量", "单价", "金额", "清单工程量", "合价", "人工", "材料", "机械", "备注"}

// treeSampleRows 两组变长的清单：Q01 挂两条定额（系数 1.5 / 留空），Q02 挂一条
// （系数 0.5）；明细行填 编码 + 层级 + 含量，名称 / 单位 / 单价留空（真实表里是
// 取价公式）。金额列是按工作簿口径口算的数，供断言。
var treeSampleRows = [][]any{
	{"1", "清单", "Q01", "垫层", "m³", 1, 322, 322, 10, 3220},
	{"1.1", "定额", "D-A1", "垫层浇筑", "m³", 1.5, 230, 230},
	{"", "人工", "L01", "", "", 0.1, "", 30},
	{"", "材料", "C20", "", "", 0.5, "", 200},
	{"1.2", "定额", "D-A2", "垫层夯实", "m³", "", 44, 44},
	{"", "机械", "EXC", "", "", 0.02, "", 40},
	{"", "外委", "OUT", "", "", 1, "", 4},
	{"2", "清单", "Q02", "围挡", "m", 1, 50, 50, 100, 5000},
	{"2.1", "定额", "D-B1", "围挡搭设", "m", 0.5, 100, 100},
	{"", "材料", "C20", "", "", 1, "", 100},
}

// treeSheetRows 综合单价的行集：前两行留给标题带与说明行，表头落在第 3 行
// （与真实样表一致，数据行号从 4 起）。
func treeSheetRows(data ...[]any) [][]any {
	return append([][]any{{}, {}, treeUnitPriceHeader}, data...)
}

func blockOf(b *ProjectBundle, item, code string) *ProjectQuotaBlock {
	for i := range b.QuotaBlocks {
		if b.QuotaBlocks[i].ItemCode == item && b.QuotaBlocks[i].Code == code {
			return &b.QuotaBlocks[i]
		}
	}
	return nil
}

// 变长组 + 换算系数：行型只看层级列，不按固定 11 行网格推；定额行「工程量」列
// 是换算系数（1.5 / 留空=1 / 0.5），明细行「工程量」列是含量（每定额单位）。
func TestParseUnitPriceTreeVariableGroupsAndFactors(t *testing.T) {
	// 表头落在第 3 行（同真实样表：上面是标题带与说明行），数据从第 4 行起。
	rows := append(treeSheetRows(treeSampleRows...), []any{"", "", "", "合计", "", "", "", 6720}) // 小计行：层级空、无编码
	f := sheetWithRows(t, "综合单价", rows)
	b := &ProjectBundle{Resources: []ProjectResource{{Code: "L01", Kind: "人工", Title: "普工", Unit: "工日"}}}
	if err := parseUnitPriceSheet(f, "综合单价", b); err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(b.Items) != 2 || b.Items[0].Code != "Q01" || b.Items[1].Code != "Q02" {
		t.Fatalf("两条清单都应识别，实得 %+v", b.Items)
	}
	closeTo(t, "Q01 原表综合单价（清单行 G 列）", b.Items[0].ListPrice, 322)
	closeTo(t, "Q02 原表综合单价", b.Items[1].ListPrice, 50)
	if len(b.Skipped) != 0 {
		t.Errorf("变长组不该有跳过行（小计行静默），实得 %+v", b.Skipped)
	}

	for _, tc := range []struct {
		item, code, title, unit string
		factor                  float64
	}{
		{"Q01", "D-A1", "垫层浇筑", "m³", 1.5}, {"Q01", "D-A2", "垫层夯实", "m³", 0}, {"Q02", "D-B1", "围挡搭设", "m", 0.5},
	} {
		blk := blockOf(b, tc.item, tc.code)
		if blk == nil {
			t.Fatalf("缺定额块 %s / %s，实得 %+v", tc.item, tc.code, b.QuotaBlocks)
		}
		if blk.Title != tc.title || blk.Unit != tc.unit || blk.Factor != tc.factor {
			t.Errorf("定额块 %+v, want 标题 %q / 单位 %q / 系数 %v", *blk, tc.title, tc.unit, tc.factor)
		}
	}
	if len(b.QuotaBlocks) != 3 {
		t.Errorf("定额块数 = %d, want 3", len(b.QuotaBlocks))
	}

	if len(b.Lines) != 5 {
		t.Fatalf("消耗行数 = %d, want 5，实得 %+v", len(b.Lines), b.Lines)
	}
	if l := findLine(b.Lines, "Q01", "L01"); l == nil || l.QuotaCode != "D-A1" || l.Kind != KindLabor ||
		l.Quantity != 0.1 || l.ResourceTitle != "普工" || l.Unit != "工日" || l.Amount != 30 {
		t.Errorf("L01 消耗行（名称 / 单位应回退工料机价格表）= %+v", l)
	}
	if l := findLine(b.Lines, "Q01", "C20"); l == nil || l.QuotaCode != "D-A1" || l.Kind != KindMaterial ||
		l.Amount != 200 || l.ResourceTitle != "" {
		t.Errorf("C20 消耗行（工料机价格表里没有这一行，名称应为空）= %+v", l)
	}
	if l := findLine(b.Lines, "Q01", "EXC"); l == nil || l.QuotaCode != "D-A2" || l.Kind != KindMachine || l.Amount != 40 {
		t.Errorf("EXC 消耗行应挂在第二条定额 D-A2 上，实得 %+v", l)
	}
	if l := findLine(b.Lines, "Q01", "OUT"); l == nil || l.QuotaCode != "D-A2" || l.Kind != KindOutsourced {
		t.Errorf("外委明细应归一成外委类别并挂在 D-A2 上，实得 %+v", l)
	}
	if l := findLine(b.Lines, "Q02", "C20"); l == nil || l.QuotaCode != "D-B1" || l.Amount != 100 {
		t.Errorf("Q02 的 C20 消耗行 = %+v", l)
	}
}

// 跳过必须点名：每一种残缺行都要显形成跳过行并说清原因，只有真正的空槽才静默。
// 「上方没有可用」的结构行其后内容不得错挂到上一组 / 上一块。
func TestParseUnitPriceTreeSkipsAreNamed(t *testing.T) {
	rows := [][]any{
		{}, {}, treeUnitPriceHeader, // 表头在第 3 行，数据行号与行尾注记一致
		{"", "定额", "D-ORPHAN", "孤儿定额", "m³", 1, 0, 0},   // 4  上面没有清单行
		{"", "人工", "L01", "", "", 0.1, 300, 30},         // 5  挂在孤儿块上
		{"1", "清单", "", "没填编码的清单", "m³", 1, 0, 0},       // 6  缺清单编码
		{"1.1", "定额", "D-X", "挂在坏清单下", "m³", 1, 0, 0},   // 7
		{"", "定额", "", "", "", 1, 0, 0},                 // 8  样表预置的空定额块：静默
		{"", "", "", "", "", "", "", ""},                // 9  空槽：静默
		{"2", "清单", "Q02", "第二条", "m³", 1, 0, 0},        // 10
		{"2.1", "定额", "", "没填编码的定额", "m³", 1, 0, 0},     // 11 缺定额编码
		{"", "材料", "C20", "", "", 0.5, 400, 200},        // 12 挂在缺码的块上
		{"2.2", "定额", "D-OK", "正常的块", "m³", 1, 0, 0},    // 13
		{"", "材料", "", "", "", 2, 400, 800},             // 14 有含量无编码
		{"", "", "C30", "", "", 0.5, 400, 200},          // 15 漏填层级
		{"", "人工", "L01", "", "", 0.1, 300, 30},         // 16 正常明细
		{"2", "清单", "Q02", "重复的清单", "m³", 1, 0, 0},      // 17 清单编码重复
		{"2.3", "定额", "D-DUP", "重复组里的块", "m³", 1, 0, 0}, // 18
		{"", "汇总", "L02", "", "", 0.1, 300, 30},         // 19 层级不是六种（旧版词）
	}
	b := &ProjectBundle{}
	if err := parseUnitPriceSheet(sheetWithRows(t, "综合单价", rows), "综合单价", b); err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(b.Items) != 1 || b.Items[0].Code != "Q02" || len(b.QuotaBlocks) != 1 || b.QuotaBlocks[0].Code != "D-OK" ||
		len(b.Lines) != 1 || b.Lines[0].QuotaCode != "D-OK" {
		t.Fatalf("只应识别 Q02 + 一条正常定额块 D-OK + 一条消耗行，实得 items=%+v blocks=%+v lines=%+v",
			b.Items, b.QuotaBlocks, b.Lines)
	}
	want := map[int][]string{
		4:  {"D-ORPHAN", "上方没有可用的清单行"},
		5:  {"L01", "上方没有可用的定额行"},
		6:  {"没填编码的清单", "缺清单编码"},
		7:  {"D-X", "上方没有可用的清单行"},
		11: {"没填编码的定额", "缺定额编码"},
		12: {"C20", "上方没有可用的定额行"},
		14: {"有含量 2 却没有工料机编码"},
		15: {"C30", "层级为空"},
		17: {"Q02", "重复出现"},
		18: {"D-DUP", "上方没有可用的清单行"},
		19: {"「汇总」不是"},
	}
	got := map[int]string{}
	for _, sk := range b.Skipped {
		got[sk.Row] = sk.Reason
	}
	for row, whys := range want {
		for _, why := range whys {
			if !strings.Contains(got[row], why) {
				t.Errorf("第 %d 行应因「%s」跳过，实得 %q", row, why, got[row])
			}
		}
	}
	if len(b.Skipped) != len(want) {
		t.Errorf("跳过行数 = %d, want %d（空槽与预置空块必须静默）: %+v", len(b.Skipped), len(want), b.Skipped)
	}
}

// 换算系数为负：显形为告警并按 1 计，定额块照常保留。
func TestParseUnitPriceTreeNegativeFactor(t *testing.T) {
	rows := append([][]any{treeUnitPriceHeader},
		[]any{"1", "清单", "Q01", "垫层", "m³", 1, 0, 0},
		[]any{"1.1", "定额", "D-A1", "垫层浇筑", "m³", -1, 0, 0},
		[]any{"", "人工", "L01", "", "", 0.1, 300, 30})
	b := &ProjectBundle{}
	if err := parseUnitPriceSheet(sheetWithRows(t, "综合单价", rows), "综合单价", b); err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if blk := blockOf(b, "Q01", "D-A1"); blk == nil || blk.Factor != 0 {
		t.Fatalf("负系数的定额块应保留并按 0（跟随）记，实得 %+v", blk)
	}
	if len(b.Warnings) != 1 || !strings.Contains(b.Warnings[0], "换算系数为负") {
		t.Errorf("负系数应告警，实得 %v", b.Warnings)
	}
}

// 旧「行类」版式拒收：退役后不再认，必须报错——此前是告警后继续导，定额与
// 消耗行整批丢失而清单行照常入库。
func TestParseProjectWorkbookRejectsOldRowKindLayout(t *testing.T) {
	f := newTreeWorkbook(t)
	putRow(t, f, sheetUnitPrice, 3, []any{"序号", "清单编码", "项目名称", "计量单位", "行类", "定额编码", "定额名称", "系数", "资源编码", "资源名称", "消耗量单位", "消耗量", "单价", "合价"})
	putRow(t, f, sheetUnitPrice, 4, []any{"1", "Q1", "污染土清挖", "m³", "汇总", "", "", "", "", "", "", "", "", 70})
	path := saveTreeWorkbook(t, f)

	b, err := ParseProjectWorkbook(path)
	if err == nil || !strings.Contains(err.Error(), "不是三级树表") {
		t.Fatalf("旧版式应整份拒收并说明原因，实得 err=%v bundle=%+v", err, b)
	}
	if b != nil && (len(b.Items) > 0 || len(b.Lines) > 0) {
		t.Errorf("拒收时不应返回半截解析结果: %+v", b)
	}
	// 单表同理：函数级也返回 error，而不是只记一条告警。
	single := sheetWithRows(t, "综合单价", [][]any{
		{"序号", "清单编码", "项目名称", "单位", "行类", "资源编码", "资源名称", "消耗单位", "消耗量", "单价", "合价"},
		{"1", "Q1", "清挖", "m³", "汇总", "", "", "", "", "", 70},
	})
	sb := &ProjectBundle{}
	if err := parseUnitPriceSheet(single, "综合单价", sb); err == nil || len(sb.Items) != 0 {
		t.Errorf("旧版式综合单价应报错且不出清单项，实得 err=%v items=%+v", err, sb.Items)
	}
}

// 封面项目名：顶部标题带最左一段（B2）与「项目名称」标签值二选一。
func TestParseCoverProjectName(t *testing.T) {
	cases := []struct {
		name, b2, g2, value, want, wantWarn string
	}{
		{"v5.5：书名在 G2，标签值与 B2 一致", "甲项目地块修复", "施 工 成 本 测 算 书", "甲项目地块修复", "甲项目地块修复", ""},
		{"v5.5 空白样表：标签值是占位", "（填写实际项目名称）", "施 工 成 本 测 算 书", "（与 B2 同步填写）", "（填写实际项目名称）", ""},
		{"两处不一致：按 B2 导并点名", "新项目地块修复", "施 工 成 本 测 算 书", "上一个项目的名字", "新项目地块修复", "两处不一致：标题「新项目地块修复」≠「项目名称」标签值「上一个项目的名字」"},
		{"v5.1：B2 是书名带，项目名在标签值", "土壤及地下水修复工程 · 施工成本测算书", "", "沿江包1（鑫乐洋地块）成本测算", "沿江包1（鑫乐洋地块）成本测算", ""},
		{"B2 包含标签值（简称）", "沿江包1（鑫乐洋地块）", "施 工 成 本 测 算 书", "沿江包1（鑫乐洋地块）土壤修复成本测算", "沿江包1（鑫乐洋地块）", ""},
		{"没有标签：只有标题", "单一标题的测算项目", "", "", "单一标题的测算项目", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cover := [][]any{{}, {"", tc.b2, "", "", "", "", tc.g2}}
			for r := 3; r <= 9; r++ {
				cover = append(cover, []any{})
			}
			cover = append(cover, []any{"", "项目名称", "", tc.value})
			f := sheetWithRows(t, "封面", cover)
			// v5.5 封面的合并区：项目名 B2:F2、书名 G2:J2、标签 B:C、值 D:J。
			mergeCover(t, f, "B2:F2", "B10:C10", "D10:J10")
			if tc.g2 != "" {
				mergeCover(t, f, "G2:J2")
			}
			b := &ProjectBundle{}
			parseCover(f, "封面", b)
			if b.Project != tc.want {
				t.Errorf("项目名 = %q, want %q", b.Project, tc.want)
			}
			warned := ""
			for _, w := range b.Warnings {
				warned += w
			}
			if tc.wantWarn == "" && warned != "" {
				t.Errorf("不该有告警: %v", b.Warnings)
			}
			if tc.wantWarn != "" && !strings.Contains(warned, tc.wantWarn) {
				t.Errorf("应告警 %q，实得 %v", tc.wantWarn, b.Warnings)
			}
		})
	}
}

// 封面标签与值之间隔着细分隔列（标签 B:C、值 D:J）：地点 / 工期要配得上对；
// 「专业 / 计价期」合写标签拆成两段；占位值不采信。
func TestParseCoverGapLabelPairsAndScope(t *testing.T) {
	gapCover := func(rows [][]any) *excelize.File {
		cover := [][]any{{}, {"", "沿江包1成本测算"}}
		for r := 3; r <= 9; r++ {
			cover = append(cover, []any{})
		}
		return sheetWithRows(t, "封面", append(cover, rows...))
	}
	b := &ProjectBundle{}
	parseCover(gapCover([][]any{
		{"", "专业 / 计价期", "", "土壤修复 / 2026-12"},
		{"", "建设地点", "", "重庆市大足区雍溪镇"},
		{"", "工期", "", "143 日历天"},
		{"", "计价方式", "", "清单计价 · 工料法（综合单价只含人材机）"},
	}), "封面", b)
	if b.Project != "沿江包1成本测算" {
		t.Errorf("项目名 = %q", b.Project)
	}
	if b.CoverSpecialty != "土壤修复" || b.CoverPeriod != "2026-12" {
		t.Errorf("合写标签应拆出 专业 / 计价期，实得 %q / %q", b.CoverSpecialty, b.CoverPeriod)
	}
	if b.Location != "重庆市大足区雍溪镇" || b.Duration != "143 日历天" {
		t.Errorf("地点 / 工期 = %q / %q（标签与值之间隔一列也要配上对）", b.Location, b.Duration)
	}
	if b.Pricing != "清单计价 · 工料法（综合单价只含人材机）" {
		t.Errorf("组价口径自述 = %q", b.Pricing)
	}

	blank := &ProjectBundle{}
	parseCover(gapCover([][]any{
		{"", "专业 / 计价期", "", "（专业） / （YYYY-MM）"},
		{"", "建设地点", "", "（省 / 市 / 区）"},
	}), "封面", blank)
	if blank.CoverSpecialty != "" || blank.CoverPeriod != "" || blank.Location != "" {
		t.Errorf("占位值不该被当成口径 / 地点: %+v", blank)
	}
}

// 换算系数贯穿到落库：清单挂接定额的工程量 = 清单工程量 × 系数（留空 = 1）。
func TestApplyProjectBundleCarriesQuotaFactor(t *testing.T) {
	f := newTreeWorkbook(t)
	putRow(t, f, sheetCover, 2, []any{"", "系数测算"})
	for i, r := range [][]any{
		{"L01", "", "人工", "普工", "", "工日", 300},
		{"C20", "", "材料", "商品混凝土", "", "m³", 400},
		{"EXC", "", "机械", "挖掘机", "", "台班", 2000},
		{"OUT", "", "外委", "检测", "", "项", 4},
	} {
		putRow(t, f, sheetResourcePr, 4+i, r)
	}
	for i, r := range [][]any{
		{"1", "A基础", "Q01", "垫层", "", "m³", 10},
		{"2", "B措施", "Q02", "围挡", "", "m", 100},
	} {
		putRow(t, f, sheetBill, 4+i, r)
	}
	for i, r := range treeSampleRows {
		putRow(t, f, sheetUnitPrice, 4+i, r)
	}
	b, err := ParseProjectWorkbook(saveTreeWorkbook(t, f))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(b.Items) != 2 || len(b.QuotaBlocks) != 3 || len(b.Lines) != 5 || len(b.Skipped) != 0 {
		t.Fatalf("规模不对：清单 %d / 定额块 %d / 消耗行 %d / 跳过 %+v", len(b.Items), len(b.QuotaBlocks), len(b.Lines), b.Skipped)
	}
	for _, w := range b.Warnings {
		t.Errorf("不该有告警: %s", w)
	}

	s, cleanup := newWorkcostStore(t)
	defer cleanup()
	res := s.ApplyProjectBundle(b)
	for _, e := range res.Errors {
		t.Errorf("落库出错: %s", e)
	}
	projects, err := s.BillProjects()
	if err != nil || len(projects) != 1 || projects[0].Name != "系数测算" || projects[0].ItemCount != 2 {
		t.Fatalf("应落 1 个项目「系数测算」2 条清单，实得 %+v（err=%v）", projects, err)
	}
	bills, err := s.BillItems(projects[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	wantLinks := map[string]map[string]float64{
		"Q01": {"D-A1": 15, "D-A2": 10}, // 10 × 1.5；留空 = 1
		"Q02": {"D-B1": 50},             // 100 × 0.5
	}
	for _, it := range bills {
		links, err := s.BillQuotaLinks(it.ID)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]float64{}
		for _, lk := range links {
			got[lk.QuotaCode] = lk.Quantity
		}
		for code, want := range wantLinks[it.Code] {
			closeTo(t, it.Code+" 挂接 "+code+" 的工程量（清单工程量 × 系数）", got[code], want)
		}
		if len(got) != len(wantLinks[it.Code]) {
			t.Errorf("%s 挂接 = %v, want %v", it.Code, got, wantLinks[it.Code])
		}
	}
	// 定额单价 = Σ(含量 × 资源价)，不含系数（系数在挂接上）。
	for code, want := range map[string]float64{"D-A1": 230, "D-A2": 44, "D-B1": 400} {
		c, _, err := s.ComposeQuota(code, nil)
		if err != nil {
			t.Fatalf("核算 %s 失败: %v", code, err)
		}
		closeTo(t, code+" 综合单价", c.CompositePrice, want)
	}
}

// mergeCover 合并封面上的区域（写成 A1:B2）。
func mergeCover(t *testing.T, f *excelize.File, ranges ...string) {
	t.Helper()
	for _, rng := range ranges {
		from, to, ok := strings.Cut(rng, ":")
		if !ok {
			t.Fatalf("合并区域要写成 A1:B2，得到 %q", rng)
		}
		if err := f.MergeCell(sheetCover, from, to); err != nil {
			t.Fatal(err)
		}
	}
}

// ── 合成工作簿 ──────────────────────────────────────────────────────

var treeHeaders = map[string][]string{
	sheetBill:       {"序号", "分部", "编码", "项目名称", "工序特征", "单位", "工程量", "综合单价", "合价（元）"},
	sheetUnitPrice:  {"序号", "层级", "编码", "名称", "单位", "工程量", "单价", "金额", "清单工程量", "合价", "人工", "材料", "机械", "备注"},
	sheetResourcePr: {"编码", "自有码", "类别", "名称", "规格/计算式", "单位", "不含税单价（元）", "来源", "备注"},
	sheetQuantity:   {"编码", "名称", "数值", "单位", "公式/口径", "来源", "去向", "备注"},
	sheetSummary:    {"序号", "费用名称", "计算基数 / 说明", "费率", "金额（元）", "备注"},
}

// newTreeWorkbook 合成一份六表三级树表工作簿：数据表带样表表头（第 3 行），取费
// 链在行 17–21（企管 5% / 利润 4% / 规费 0 / 增值税 9%），封面留空。只写解析器
// 要读的格，公式格不写（合成表没有缓存值）。
func newTreeWorkbook(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	for i, name := range []string{sheetCover, sheetSummary, sheetBill, sheetUnitPrice, sheetResourcePr, sheetQuantity} {
		if i == 0 {
			if err := f.SetSheetName("Sheet1", name); err != nil {
				t.Fatal(err)
			}
		} else if _, err := f.NewSheet(name); err != nil {
			t.Fatal(err)
		}
		if hdr := treeHeaders[name]; hdr != nil {
			putRow(t, f, name, 3, hdr)
		}
	}
	for i, row := range [][]any{
		{"二", "企业管理费", "直接费 × 管理费率", 0.05},
		{"三", "利润", "（直接费 + 管理费）× 利润率", 0.04},
		{"四", "规费", "直接费 × 规费率", 0},
		{"五", "税前合计（不含税造价）", "直接费 + 管理费 + 利润 + 规费", "—"},
		{"六", "增值税", "税前合计 × 增值税率", 0.09},
	} {
		putRow(t, f, sheetSummary, 17+i, row)
	}
	return f
}

// putRow 把一行值从 A 列起写到第 row 行（空串也占位，保持列位）。
func putRow[T any](t *testing.T, f *excelize.File, sheet string, row int, vals []T) {
	t.Helper()
	for j, v := range vals {
		cell, _ := excelize.CoordinatesToCellName(j+1, row)
		if err := f.SetCellValue(sheet, cell, v); err != nil {
			t.Fatalf("写 %s!%s 失败: %v", sheet, cell, err)
		}
	}
}

// saveTreeWorkbook 落到临时目录并返回路径（ParseProjectWorkbook 只读盘上文件）。
func saveTreeWorkbook(t *testing.T, f *excelize.File) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tree.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return path
}

// 硬契约只有综合单价表（2026-11-27 用户拍板）：其余表是参考，缺了不拦导入。
// 没有清单表时，工程量取综合单价表「清单工程量」列（重算后的缓存值）。
func TestParseProjectWorkbookOnlyUnitPriceRequired(t *testing.T) {
	f := newTreeWorkbook(t)
	for i, r := range treeSampleRows {
		putRow(t, f, sheetUnitPrice, 4+i, r) // 清单行的「清单工程量」列：Q01 10、Q02 100
	}
	for i, r := range [][]any{
		{"L01", "", "人工", "普工", "", "工日", 300},
		{"C20", "", "材料", "商品混凝土", "", "m³", 400},
	} {
		putRow(t, f, sheetResourcePr, 4+i, r)
	}
	for _, name := range []string{sheetCover, sheetBill, sheetSummary, sheetQuantity} {
		if err := f.DeleteSheet(name); err != nil {
			t.Fatal(err)
		}
	}
	b, err := ParseProjectWorkbook(saveTreeWorkbook(t, f))
	if err != nil {
		t.Fatalf("只有综合单价 + 工料机价格两张表也应能导入: %v", err)
	}
	if len(b.Sheets) != 2 || len(b.Items) != 2 || len(b.QuotaBlocks) != 3 || len(b.Lines) != 5 {
		t.Fatalf("规模不对：表 %v / 清单 %d / 定额块 %d / 消耗行 %d", b.Sheets, len(b.Items), len(b.QuotaBlocks), len(b.Lines))
	}
	closeTo(t, "Q01 工程量（取综合单价「清单工程量」列）", b.Items[0].Quantity, 10)
	closeTo(t, "Q02 工程量（取综合单价「清单工程量」列）", b.Items[1].Quantity, 100)
	for _, w := range b.Warnings {
		if strings.Contains(w, "工程量为 0") {
			t.Errorf("工程量读到了，不该报「工程量为 0」: %s", w)
		}
	}
	for _, want := range []string{"没有费用汇总表"} {
		if !strings.Contains(strings.Join(b.Warnings, "\n"), want) {
			t.Errorf("没有费用汇总表应告警 %q，实得 %v", want, b.Warnings)
		}
	}

	// 连工料机价格表也没有：照样解析，告警点明消耗行按库内既有编码解析。
	f = newTreeWorkbook(t)
	for i, r := range treeSampleRows {
		putRow(t, f, sheetUnitPrice, 4+i, r)
	}
	for _, name := range []string{sheetCover, sheetBill, sheetSummary, sheetQuantity, sheetResourcePr} {
		if err := f.DeleteSheet(name); err != nil {
			t.Fatal(err)
		}
	}
	b, err = ParseProjectWorkbook(saveTreeWorkbook(t, f))
	if err != nil {
		t.Fatalf("只有综合单价一张表也应能解析: %v", err)
	}
	if len(b.Items) != 2 || len(b.Lines) != 5 || len(b.Resources) != 0 {
		t.Fatalf("规模不对：清单 %d / 消耗行 %d / 工料机 %d", len(b.Items), len(b.Lines), len(b.Resources))
	}
	if !strings.Contains(strings.Join(b.Warnings, "\n"), "没有工料机价格表") {
		t.Errorf("没有工料机价格表应告警，实得 %v", b.Warnings)
	}
}

// 清单表是参考表：模型可以按单项工程改版式——上面放单项工程汇总块、明细表加一列
// 「单项工程」顶替「分部」、按单项工程分段插标题行、工程量格留空。表头要认带「编码」
// 的那一行（汇总块也有「名称 / 工程量」列）；标题行与「小计」行不出清单项。
func TestParseBillSheetDividedBySingleProject(t *testing.T) {
	f := newTreeWorkbook(t)
	putRow(t, f, sheetCover, 2, []any{"", "沿江包1测算"})
	for i, r := range treeSampleRows {
		putRow(t, f, sheetUnitPrice, 4+i, r)
	}
	for i, r := range [][]any{
		{"L01", "", "人工", "普工", "", "工日", 300},
		{"C20", "", "材料", "商品混凝土", "", "m³", 400},
		{"EXC", "", "机械", "挖掘机", "", "台班", 2000},
	} {
		putRow(t, f, sheetResourcePr, 4+i, r)
	}
	// 清单表改成按单项工程划分的版式。
	if err := f.DeleteSheet(sheetBill); err != nil {
		t.Fatal(err)
	}
	if _, err := f.NewSheet(sheetBill); err != nil {
		t.Fatal(err)
	}
	bill := [][]any{
		{"分部分项工程量清单", "", "", "", "", "", "直接费合计（元）", 14220},
		{},
		{"序号", "单项工程名称", "单位", "工程量", "金额（元）"}, // 单项工程汇总块：有「名称 / 工程量」列，但不是清单明细
		{"一", "污染土壤修复工程", "项", "1", 8220},
		{"二", "地下水处理工程", "项", "1", 6000},
		{},
		{"序号", "单项工程", "编码", "名称", "单位", "工程量", "综合单价", "合价（元）"},
		{"一", "污染土壤修复工程"}, // 分段标题行
		{"1", "污染土壤修复工程", "Q01", "垫层", "m³", 10},
		{"2", "污染土壤修复工程", "Q02", "围挡", "", ""}, // 工程量留空：退回综合单价「清单工程量」列
		{"二", "地下水处理工程"},
		{"1", "地下水处理工程", "Q03", "抽提井", "口", 88}, // 只在清单表出现的项
		{"", "", "", "小计", "", 88},
	}
	for i, r := range bill {
		putRow(t, f, sheetBill, i+1, r)
	}

	b, err := ParseProjectWorkbook(saveTreeWorkbook(t, f))
	if err != nil {
		t.Fatalf("清单表按单项工程改版式后不应拒收: %v", err)
	}
	if len(b.Items) != 3 || b.Items[0].Code != "Q01" || b.Items[1].Code != "Q02" || b.Items[2].Code != "Q03" {
		t.Fatalf("应识别 Q01 / Q02（来自综合单价）与 Q03（只在清单表），实得 %+v", b.Items)
	}
	closeTo(t, "Q01 工程量（清单表）", b.Items[0].Quantity, 10)
	closeTo(t, "Q02 工程量（清单表留空 → 综合单价「清单工程量」列）", b.Items[1].Quantity, 100)
	closeTo(t, "Q03 工程量", b.Items[2].Quantity, 88)
	if b.Items[0].Division != "污染土壤修复工程" || b.Items[1].Division != "污染土壤修复工程" || b.Items[2].Division != "地下水处理工程" {
		t.Errorf("「单项工程」列应顶替「分部」，实得 %q / %q / %q", b.Items[0].Division, b.Items[1].Division, b.Items[2].Division)
	}
	if b.Items[0].Title != "垫层" || b.Items[0].Unit != "m³" || b.Items[1].Unit != "m" {
		t.Errorf("清单表不应改写综合单价表读到的名称与单位，实得 %+v", b.Items[:2])
	}
	warns := strings.Join(b.Warnings, "\n")
	if !strings.Contains(warns, "Q03") || !strings.Contains(warns, "消耗量待补") {
		t.Errorf("只在清单表出现的项应告警待补消耗量，实得 %v", b.Warnings)
	}
	if strings.Contains(warns, "工程量为 0") || strings.Contains(warns, "清单表未识别到表头") {
		t.Errorf("不该有这些告警: %v", b.Warnings)
	}
}
