package app

// office_cost_import_test.go — 模型读表导入的契约（2026-11-25 用户拍板：造价资料格式
// 不固定，导入由模型读资料、整理后整批提交）：开导入 → 分批入暂存 → 勾稽（回读核对 /
// 控制数 / 单价对照）→ 提交。回读核对用真 xlsx 与 csv 验，全走隔离库。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/workcost"
)

// impBook 造一份「不是标准五表」的资料：列序、表名都和模版不一样，且工料机价格表里
// 夹着一行整格同一段话的说明。模型读表时自己判断那不是工料机，不提交它。
func impBook(t *testing.T) string {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetSheetName("Sheet1", "封面"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"工料机价格", "综合单价", "费用汇总"} {
		if _, err := f.NewSheet(name); err != nil {
			t.Fatal(err)
		}
	}
	row := func(sheet, cell string, vals []interface{}) {
		t.Helper()
		if err := f.SetSheetRow(sheet, cell, &vals); err != nil {
			t.Fatal(err)
		}
	}
	row("封面", "A1", []interface{}{"某某地块修复成本测算"})
	row("封面", "A3", []interface{}{"地点", "四川乐山"})
	row("工料机价格", "A1", []interface{}{"编码", "类别", "名称", "规格", "单位", "不含税单价", "来源"})
	row("工料机价格", "A2", []interface{}{"L01", "人工", "普工", "", "工日", 180, "市场"})
	row("工料机价格", "A3", []interface{}{"C20", "材料", "商品混凝土", "C20", "m³", 430, "信息价"})
	// 第 4 行是说明：整格同一段话（真实资料与真实库里都有这种行）。
	note := "F列输入为蓝字黄底。台班、车次、砖砌体为公式。改柴油价后一起变。"
	if err := f.MergeCell("工料机价格", "A4", "G4"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{"A4", "B4", "C4", "D4", "E4", "F4", "G4"} {
		if err := f.SetCellValue("工料机价格", c, note); err != nil {
			t.Fatal(err)
		}
	}
	row("工料机价格", "A5", []interface{}{"EXC", "机械", "挖掘机台班", "1.0m³", "台班", 1773.3, "台班公式"})
	row("综合单价", "A1", []interface{}{"序号", "清单编码", "项目名称", "单位", "行类", "资源编码", "资源名称", "消耗单位", "消耗量", "单价", "合价"})
	row("综合单价", "A2", []interface{}{1, "WP01", "垫层", "m³", "汇总", "", "", "", "", "", 528.6})
	row("综合单价", "A3", []interface{}{"1.1", "WP01", "垫层", "m³", "人工", "L01", "普工", "工日", 0.5, 180, 90})
	row("综合单价", "A4", []interface{}{"1.2", "WP01", "垫层", "m³", "材料", "C20", "商品混凝土", "m³", 1.02, 430, 438.6})
	row("费用汇总", "A1", []interface{}{"序号", "分部", "编码", "项目名称", "项目特征", "单位", "工程量", "综合单价"})
	row("费用汇总", "A2", []interface{}{1, "A临建", "WP01", "垫层", "C20 垫层", "m³", 100, 528.6})
	path := filepath.Join(t.TempDir(), "某某地块成本测算.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	return path
}

// impOpen 开一次导入并取回会话 id。extra 里的键覆盖默认入参（project / controls）。
func impOpen(t *testing.T, tl costLibraryTool, source string, extra map[string]any) string {
	t.Helper()
	args := map[string]any{"action": "import_open", "source": source, "project": map[string]any{
		"name": "某某地块修复", "specialty": "土壤修复", "region": "乐山市", "price_period": "2026年6月",
	}}
	for k, v := range extra {
		args[k] = v
	}
	out := clMust(t, tl, args, "已开导入：imp-")
	m := regexp.MustCompile(`imp-\d+`).FindString(out)
	if m == "" {
		t.Fatalf("回执里应有会话号：%s", out)
	}
	return m
}

// impResources 模型整理好的工料机行（每行带来源位置）。说明行（工料机价格 第 4 行）不提交。
func impResources() []map[string]any {
	return []map[string]any{
		{"code": "L01", "kind": "人工", "title": "普工", "unit": "工日", "price": 180, "src": map[string]any{"sheet": "工料机价格", "row": 2}},
		{"code": "C20", "kind": "材料", "title": "商品混凝土", "spec": "C20", "unit": "m³", "price": 430, "src": map[string]any{"sheet": "工料机价格", "row": 3}},
		{"code": "EXC", "kind": "机械", "title": "挖掘机台班", "spec": "1.0m³", "unit": "台班", "price": 1773.3, "src": map[string]any{"sheet": "工料机价格", "row": 5}},
	}
}

// impQuotas 模型整理好的定额：汇总行是定额本体，明细行是含量行（各带来源位置）。
func impQuotas() []map[string]any {
	return []map[string]any{{
		"code": "WP01", "title": "垫层", "unit": "m³", "src": map[string]any{"sheet": "综合单价", "row": 2},
		"items": []map[string]any{
			{"resource_code": "L01", "quantity": 0.5, "src": map[string]any{"sheet": "综合单价", "row": 3}},
			{"resource_code": "C20", "quantity": 1.02, "src": map[string]any{"sheet": "综合单价", "row": 4}},
		},
	}}
}

// impBill 模型整理好的一条清单行（综合单价现算 0.5×180 + 1.02×430 = 528.6）。
func impBill(code string, quantity, listPrice float64) map[string]any {
	return map[string]any{
		"code": code, "title": "垫层", "unit": "m³", "division": "A临建", "quantity": quantity,
		"quota_code": "WP01", "feature": "C20 垫层", "list_price": listPrice,
		"src": map[string]any{"sheet": "费用汇总", "row": 2},
	}
}

func impDict(t *testing.T, s *workcost.Store) {
	t.Helper()
	for _, d := range []workcost.DictItem{
		{Kind: workcost.DictSpecialty, Name: "土壤修复"},
		{Kind: workcost.DictRegion, Name: "乐山市"},
	} {
		if err := s.DictAdd(d); err != nil {
			t.Fatal(err)
		}
	}
}

func impAdd(t *testing.T, tl costLibraryTool, id, kind string, rows []map[string]any) string {
	t.Helper()
	return clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, kind: rows}, "退回 0")
}

// TestCostImportOpen 开导入：口径不过规矩、资料读不到，当场拒。
func TestCostImportOpen(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	impDict(t, a.hubWorkcostStore())
	tl := costLibraryTool{a: a}
	book := impBook(t)

	clMustFail(t, tl, map[string]any{"action": "import_open", "source": book, "project": map[string]any{"specialty": "不存在专业"}},
		"导入口径不合规矩")
	clMustFail(t, tl, map[string]any{"action": "import_open", "source": book, "project": map[string]any{"price_period": "2026年第7期"}},
		"不是年月写法")
	clMustFail(t, tl, map[string]any{"action": "import_open", "source": filepath.Join(t.TempDir(), "没有这份资料.xlsx")}, "读不到资料")
	out := clMust(t, tl, map[string]any{"action": "import_open", "source": book, "project": map[string]any{"specialty": "土壤修复", "region": "乐山市"}},
		"回读核对可用")
	if !strings.Contains(out, "专业 土壤修复｜地区 乐山市") {
		t.Fatalf("开导入回执应显形口径：%s", out)
	}
	// 不支持回读的格式如实说。
	pdf := filepath.Join(t.TempDir(), "定额本.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}
	clMust(t, tl, map[string]any{"action": "import_open", "source": pdf}, "这种格式不支持回读核对")
}

// TestCostImportEndToEnd 开导入 → 分批入暂存（库里不动）→ 勾稽全对上 → 提交落库；
// 再导一次同一批：全部走更新，不制造重复。
func TestCostImportEndToEnd(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	impDict(t, s)
	book := impBook(t)

	id := impOpen(t, tl, book, map[string]any{"controls": map[string]any{
		"resources": 3, "quotas": 1, "bills": 1, "bill_total": 52860, "resource_price_sum": 2383.3,
	}})
	out := clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources": impResources()},
		"**本批工料机**: 提交 3 行 → 入暂存 3（其中覆盖 0）｜退回 0")
	if !strings.Contains(out, "import_check") {
		t.Fatalf("入暂存回执应指路勾稽：%s", out)
	}
	impAdd(t, tl, id, "quotas", impQuotas())
	impAdd(t, tl, id, "bills", []map[string]any{impBill("WP01", 100, 528.6)})
	clMust(t, tl, map[string]any{"action": "import_check", "import_id": id}, "**暂存**: 工料机 3｜定额 1｜清单 1")

	// 暂存期间库里一行不动。
	if list, _ := s.ListResources("", ""); len(list) != 0 {
		t.Fatalf("入暂存不应写库，库里已有 %d 条工料机", len(list))
	}
	if projects, _ := s.BillProjects(); len(projects) != 0 {
		t.Fatal("入暂存不应建项目")
	}

	// 勾稽：控制数、回读核对、单价对照全部对上。
	out = clMust(t, tl, map[string]any{"action": "import_check", "import_id": id}, "🔍 勾稽（只读）")
	for _, want := range []string{"工料机行数：3 / 3 ✓", "定额条数：1 / 1 ✓", "清单行数：1 / 1 ✓", "52860.00 / 52860.00 ✓",
		"已核对 7 行", "不符 0 处｜未带来源位置的 0 行", "可比 1 条｜偏差超 1% 的 0 条", "全部对上，可以提交"} {
		if !strings.Contains(out, want) {
			t.Errorf("勾稽报告应含 %q：%s", want, out)
		}
	}

	// 演练提交：出「将写入」报告，仍不写库。
	clMust(t, tl, map[string]any{"action": "import_commit", "import_id": id, "dry_run": true}, "**将写入**: 某某地块修复｜工料机 3｜定额 1｜清单 1")
	if list, _ := s.ListResources("", ""); len(list) != 0 {
		t.Fatal("演练提交不应写库")
	}

	// 正式提交。
	out = clMust(t, tl, map[string]any{"action": "import_commit", "import_id": id}, "✅ 已完成：提交导入")
	for _, want := range []string{"**项目**: #", "专业 土壤修复｜地区 乐山市｜计价期 2026-06",
		"工料机 新建 3 / 更新 0｜定额 新建 1 / 更新 0｜清单 1 行｜价格事实 3 条"} {
		if !strings.Contains(out, want) {
			t.Errorf("提交回执应含 %q：%s", want, out)
		}
	}
	projects, _ := s.BillProjects()
	if len(projects) != 1 || projects[0].Specialty != "土壤修复" || projects[0].Region != "乐山市" || projects[0].PricePeriod != "2026-06" {
		t.Fatalf("项目口径应随导入落库，得到 %+v", projects)
	}
	c20, err := s.GetResourceByCode("C20")
	if err != nil || c20.Region != "乐山市" || c20.PriceDate != "2026-06" || c20.CurrentPrice != 430 {
		t.Fatalf("工料机应带地区与计价期，得到 %+v err=%v", c20, err)
	}
	if facts, _ := s.ResourcePrices(c20.ID); len(facts) != 1 || facts[0].Price != 430 || facts[0].Period != "2026-06" || facts[0].Region != "乐山市" {
		t.Fatalf("提交应记价格事实（价 + 期 + 地区），得到 %+v", facts)
	}
	q, _ := s.GetQuota("WP01")
	if q.Specialty != "土壤修复" || len(q.Items) != 2 || q.Items[0].Title != "普工" || q.Items[1].Title != "商品混凝土" {
		t.Fatalf("定额专业应取项目专业、含量行展示字段应从资源库派生，得到 %+v", q)
	}
	items, _ := s.BillItems(projects[0].ID)
	if len(items) != 1 || items[0].QuotaCode != "WP01" || len(items[0].QuotaCodes) != 1 || items[0].Specialty != "土壤修复" || items[0].Quantity != 100 {
		t.Fatalf("清单行与挂接应落库、专业跟随项目，得到 %+v", items)
	}
	// 会话用完即弃。
	clMustFail(t, tl, map[string]any{"action": "import_check", "import_id": id}, "找不到导入会话")

	// 再导一次同一批：全部走更新，行数不变。
	id2 := impOpen(t, tl, book, nil)
	impAdd(t, tl, id2, "resources", impResources())
	impAdd(t, tl, id2, "quotas", impQuotas())
	impAdd(t, tl, id2, "bills", []map[string]any{impBill("WP01", 120, 528.6)})
	clMust(t, tl, map[string]any{"action": "import_commit", "import_id": id2}, "工料机 新建 0 / 更新 3｜定额 新建 0 / 更新 1｜清单 1 行")
	if list, _ := s.ListResources("", ""); len(list) != 3 {
		t.Fatalf("重导不应制造重复工料机，得到 %d 条", len(list))
	}
	if items, _ = s.BillItems(projects[0].ID); len(items) != 1 || items[0].Quantity != 120 {
		t.Fatalf("同码清单行应走更新，得到 %+v", items)
	}
}

// TestCostImportAddRejectsNonConforming 入暂存时的规矩校验：字典外的值、四类之外的类别、
// 占用别人的编码、引用不在库也不在暂存区——逐行退回并说明原因，库里一行不动。
func TestCostImportAddRejectsNonConforming(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	impDict(t, s)
	if _, err := s.SaveResource(workcost.Resource{Code: "L01", Kind: workcost.KindMaterial, Title: "中砂", Unit: "m³"}); err != nil {
		t.Fatal(err)
	}
	id := impOpen(t, tl, impBook(t), nil)

	clMustFail(t, tl, map[string]any{"action": "import_add", "import_id": id}, "至少一组")
	clMustFail(t, tl, map[string]any{"action": "import_add", "import_id": "imp-999999", "resources": impResources()}, "找不到导入会话")
	clMustFail(t, tl, map[string]any{"action": "import_add", "import_id": ""}, "需要 import_id")

	out := clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources": []map[string]any{
		{"code": "X1", "kind": "一段备注", "title": "脏类别", "unit": "项", "price": 1},
		{"code": "X2", "kind": "材料", "title": "字典外地区", "unit": "t", "region": "不在字典的地方", "price": 1},
		{"code": "L01", "kind": "材料", "title": "顶替别人的编码", "unit": "t", "price": 1},
		{"code": "C20", "kind": "材料", "title": "商品混凝土", "unit": "m³", "price": 430},
	}}, "**本批工料机**: 提交 4 行 → 入暂存 1（其中覆盖 0）｜退回 3")
	for _, want := range []string{"第 1 行 X1：", "人工/材料/机械/外委", "第 2 行 X2：", "不在地区字典里", "第 3 行 L01：", "已被「中砂」占用"} {
		if !strings.Contains(out, want) {
			t.Errorf("退回原因应含 %q：%s", want, out)
		}
	}
	// 引用不在库也不在暂存区的：退回并指路。
	out = clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, "quotas": []map[string]any{{
		"code": "Q9", "title": "x", "unit": "m", "items": []map[string]any{{"resource_code": "NOPE", "quantity": 1}},
	}}}, "先把工料机提交进暂存区")
	if !strings.Contains(out, "退回 1") {
		t.Fatalf("引用缺失的定额应退回：%s", out)
	}
	out = clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, "bills": []map[string]any{
		{"code": "B1", "title": "缺单位", "quantity": 1, "quota_code": "Q9"},
		{"code": "B2", "title": "工程量为零", "unit": "m", "quantity": 0},
	}}, "缺 title/unit")
	for _, want := range []string{"先把定额提交进暂存区", "工程量 0 必须>0"} {
		if !strings.Contains(out, want) {
			t.Errorf("退回原因应含 %q：%s", want, out)
		}
	}
	// 同码重发=覆盖暂存区里的那一行（L01 在这个用例里是库里「中砂」的编码，照旧退回）。
	clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, "resources": impResources()}, "入暂存 2（其中覆盖 1）｜退回 1")
	if list, _ := s.ListResources("", ""); len(list) != 1 {
		t.Fatalf("入暂存不应写库，库里应有且只有原先的 1 条，得到 %d", len(list))
	}
	// 丢弃：暂存清掉，库里照旧。
	clMust(t, tl, map[string]any{"action": "import_discard", "import_id": id}, "库里本来就没写过")
	clMustFail(t, tl, map[string]any{"action": "import_check", "import_id": id}, "找不到导入会话")
	if list, _ := s.ListResources("", ""); len(list) != 1 {
		t.Fatal("丢弃不应动库")
	}
}

// TestCostImportReadback 回读核对是数字搬运的第二道把关：提交的数不在它声称的来源行里
// 就报出来；来源位置写错也报；模型换算的字段与没带来源位置的行如实计数、不算问题。
func TestCostImportReadback(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	impDict(t, s)
	id := impOpen(t, tl, impBook(t), nil)
	impAdd(t, tl, id, "resources", impResources())

	// 定额 WP02 的含量行把 0.5 抄成了 0.8（明细行第 3 行里没有 0.8）。它的编码、名称是
	// 为样例另起的，标成 derived 不核。
	quotas := append(impQuotas(), map[string]any{
		"code": "WP02", "title": "垫层（抄错示例）", "unit": "m³", "src": map[string]any{"sheet": "综合单价", "row": 2},
		"derived": []string{"编码", "名称"},
		"items":   []map[string]any{{"resource_code": "L01", "quantity": 0.8, "src": map[string]any{"sheet": "综合单价", "row": 3}}},
	})
	clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, "quotas": quotas}, "退回 0")

	// 四条清单行：工程量抄错 / 来源位置写错（表名多一个字） / 没带来源位置 / 工程量是换算来的。
	// 编码是为这几条样例另起的，标成 derived 不核。
	slip := impBill("B-SLIP", 120, 528.6)
	slip["derived"] = []string{"编码"}
	where := impBill("B-WHERE", 100, 528.6)
	where["src"] = map[string]any{"sheet": "费用汇总表", "row": 2}
	where["derived"] = []string{"编码"}
	plain := impBill("B-PLAIN", 100, 528.6)
	delete(plain, "src")
	plain["derived"] = []string{"编码"}
	calc := impBill("B-CALC", 999, 528.6)
	calc["derived"] = []string{"编码", "工程量"}
	clMust(t, tl, map[string]any{"action": "import_add", "import_id": id, "bills": []map[string]any{slip, where, plain, calc}}, "退回 0")

	out := clMust(t, tl, map[string]any{"action": "import_check", "import_id": id}, "🔍 勾稽（只读）")
	for _, want := range []string{
		"已核对 11 行",
		"不符 3 处｜未带来源位置的 1 行｜模型换算 6 个字段",
		"清单 B-SLIP（费用汇总 第 2 行）工程量：提交 120——该行里的数：1、100、528.6",
		"清单 B-WHERE（费用汇总表 第 2 行）来源位置：提交 （无）——找不到来源行（表名或行号不对）",
		"定额 WP02 第 1 行含量行（综合单价 第 3 行）含量：提交 0.8——该行里的数：1.1、0.5、180、90",
		"有 3 处要处理",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("回读核对应报 %q：%s", want, out)
		}
	}
	// 没带来源位置的行与换算字段不报不符。
	for _, mustNot := range []string{"B-PLAIN", "B-CALC"} {
		if strings.Contains(out, mustNot) {
			t.Errorf("不该把 %s 报成不符：%s", mustNot, out)
		}
	}
	// 单价对照与工程量无关：四条清单的定额现算都是 528.6，与原表一致。
	if !strings.Contains(out, "可比 4 条｜偏差超 1% 的 0 条") {
		t.Errorf("工程量抄错不应误报成单价偏差：%s", out)
	}
}

// TestCostImportControlsAndPrice 控制数对不上（漏行漏项）与单价偏差超 1% 都要点名。
func TestCostImportControlsAndPrice(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	impDict(t, s)
	id := impOpen(t, tl, impBook(t), map[string]any{"controls": map[string]any{"bills": 2, "bill_total": 99999}})
	impAdd(t, tl, id, "resources", impResources())
	impAdd(t, tl, id, "quotas", impQuotas())
	// 原表综合单价故意给错：资料上 528.6，提交 469.2。
	impAdd(t, tl, id, "bills", []map[string]any{impBill("WP01", 100, 469.2)})
	out := clMust(t, tl, map[string]any{"action": "import_check", "import_id": id}, "🔍 勾稽（只读）")
	for _, want := range []string{
		"清单行数：2 / 1 ✗",
		"清单合计（Σ 工程量 × 原表综合单价）：99999.00 / 46920.00 ✗",
		"可比 1 条｜偏差超 1% 的 1 条",
		"原表 ¥469.20 / 现算 ¥528.60（+12.7%）",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("勾稽报告应含 %q：%s", want, out)
		}
	}
}

// TestCostImportCsvAndNoSource csv 资料同样回读核对；没有可回读的资料时行如实计为未核对，
// 不算问题。
func TestCostImportCsvAndNoSource(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	impDict(t, s)
	csvPath := filepath.Join(t.TempDir(), "价格.csv")
	if err := os.WriteFile(csvPath, []byte("\uFEFF编码,名称,单位,单价\r\nC30,商品混凝土,m³,450\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c30 := func(price float64) []map[string]any {
		return []map[string]any{{"code": "C30", "kind": "材料", "title": "商品混凝土", "spec": "C30", "unit": "m³", "price": price, "src": map[string]any{"row": 2}}}
	}
	id := impOpen(t, tl, csvPath, nil)
	impAdd(t, tl, id, "resources", c30(450))
	clMust(t, tl, map[string]any{"action": "import_check", "import_id": id}, "已核对 1 行 5 个字段｜不符 0 处")

	// 单价抄错：450 报成 540。
	id2 := impOpen(t, tl, csvPath, nil)
	impAdd(t, tl, id2, "resources", c30(540))
	clMust(t, tl, map[string]any{"action": "import_check", "import_id": id2}, "工料机 C30（第 2 行）单价：提交 540——该行里的数：450")

	// 没登记资料路径：无法回读，如实计为未核对，不算问题。
	id3 := impOpen(t, tl, "", nil)
	impAdd(t, tl, id3, "resources", impResources())
	out := clMust(t, tl, map[string]any{"action": "import_check", "import_id": id3}, "这次导入没有可回读的资料")
	if !strings.Contains(out, "3 行全部未核对") || !strings.Contains(out, "没有要处理的问题") {
		t.Fatalf("没有可回读资料时应计为未核对而不是问题：%s", out)
	}
}

// TestCostImportCommitGate 提交前的两道闸：清单行没有项目可归属、暂存行因字典后来变了
// 而不再合规——都是整批不写，回执点名，会话留着改完重提。
func TestCostImportCommitGate(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	impDict(t, s)
	book := impBook(t)

	// ① 没给项目名，却提交了清单行。
	id0 := impOpen(t, tl, book, map[string]any{"project": map[string]any{"region": "乐山市", "specialty": "土壤修复"}})
	impAdd(t, tl, id0, "resources", impResources())
	impAdd(t, tl, id0, "quotas", impQuotas())
	impAdd(t, tl, id0, "bills", []map[string]any{impBill("WP01", 100, 528.6)})
	if _, err := clRun(t, tl, map[string]any{"action": "import_commit", "import_id": id0}); err == nil || !strings.Contains(err.Error(), "必须归属项目") {
		t.Fatalf("没有项目却提交清单行应被拒，err=%v", err)
	}
	if list, _ := s.ListResources("", ""); len(list) != 0 {
		t.Fatal("闸门拦下时不应写任何数据")
	}

	// ② 暂存之后，字典里的地区被删了：暂存行的地区（取项目口径）不再合规。
	id := impOpen(t, tl, book, nil)
	impAdd(t, tl, id, "resources", impResources())
	if err := s.DictDelete(workcost.DictRegion, "乐山市"); err != nil {
		t.Fatal(err)
	}
	_, err := clRun(t, tl, map[string]any{"action": "import_commit", "import_id": id})
	if err == nil || !strings.Contains(err.Error(), "已不合规") || !strings.Contains(err.Error(), "不在地区字典里") {
		t.Fatalf("暂存行不合规应整批不写并点名，err=%v", err)
	}
	if list, _ := s.ListResources("", ""); len(list) != 0 {
		t.Fatal("闸门拦下时不应写任何数据")
	}
	// 会话还在：字典补回来后可以重新提交。
	if err := s.DictAdd(workcost.DictItem{Kind: workcost.DictRegion, Name: "乐山市"}); err != nil {
		t.Fatal(err)
	}
	clMust(t, tl, map[string]any{"action": "import_commit", "import_id": id}, "工料机 新建 3 / 更新 0")
}

// TestCostImportArgsMatchDocumentedKeys 入参结构的 JSON 键必须与工具说明里的键一字不差——
// 错一个字母，模型按说明传的值就会被静默丢掉。逐键用说明里的写法喂进去，断言值落到了
// 对应字段。
func TestCostImportArgsMatchDocumentedKeys(t *testing.T) {
	// 模型看到的键分布在两处：常驻 schema 与回执携带的行契约——两者同出自
	// costLibraryDocumentedSchema，对它逐键核对（2026-11-25 工具梳理批）。
	schema := string(costLibraryDocumentedSchema())
	for _, needle := range []string{`"import_id"`, `"source"`, `"project"`, `"controls"`, `"resources"`, `"quotas"`, `"bills"`, `"src"`, `"row"`, `"sheet"`, `"derived"`} {
		if !strings.Contains(schema, needle) {
			t.Errorf("工具说明里应有键 %s", needle)
		}
	}
	var probe any
	if err := json.Unmarshal([]byte(schema), &probe); err != nil {
		t.Fatalf("cost_library 的说明不是合法 JSON：%v", err)
	}
	var p costLibraryArgs
	if err := json.Unmarshal([]byte(`{
		"action":"import_add","import_id":"imp-1","source":"a.xlsx",
		"project":{"name":"p","region":"乐山市","specialty":"土壤修复","price_period":"2026-06","file_name":"a.xlsx","location":"乐山","duration":"x","pricing":"y"},
		"controls":{"resources":3,"quotas":1,"bills":1,"bill_total":1.5,"resource_price_sum":2.5},
		"resources":[{"code":"C20","kind":"材料","title":"商品混凝土","unit":"m³","price":430,"src":{"sheet":"工料机价格","row":3},"derived":["单价"]}],
		"quotas":[{"code":"WP01","title":"垫层","unit":"m³","items":[{"resource_code":"C20","quantity":1.02,"loss_rate":0.03,"src":{"row":4}}]}],
		"bills":[{"code":"WP01","title":"垫层","unit":"m³","quantity":100,"quota_code":"WP01","list_price":528.6}]
	}`), &p); err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		what string
		got  string
		want string
	}{
		{"import_id", p.ImportID, "imp-1"},
		{"source", p.Source, "a.xlsx"},
		{"project.name", p.Project.Name, "p"},
		{"project.price_period", p.Project.PricePeriod, "2026-06"},
		{"controls.bills", strconv.Itoa(*p.Controls.Bills), "1"},
		{"resources[0].src.sheet", p.Resources[0].Src.Sheet, "工料机价格"},
		{"resources[0].src.row", strconv.Itoa(p.Resources[0].Src.Row), "3"},
		{"resources[0].price", strconv.FormatFloat(*p.Resources[0].Price, 'f', -1, 64), "430"},
		{"quotas[0].items[0].quantity", strconv.FormatFloat(p.Quotas[0].Items[0].Quantity, 'f', -1, 64), "1.02"},
		{"bills[0].list_price", strconv.FormatFloat(p.Bills[0].ListPrice, 'f', -1, 64), "528.6"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q（入参键与工具说明不一致？）", c.what, c.got, c.want)
		}
	}
	if p.Resources[0].Derived[0] != "单价" {
		t.Errorf("derived 没落到字段上： %+v", p.Resources[0].Derived)
	}
}
