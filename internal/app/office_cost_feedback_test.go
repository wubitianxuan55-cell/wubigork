package app

// office_cost_feedback_test.go — 2026-11-26 造价反馈三批在工具层的回归钉：save_resource
// 尊重显式编码与按地区分行、列表给总数/还有更多/地区过滤、机检的悬空骨架与跨地区
// 提示、summary 的分部小计与舍入口径、import_project 的显式控制价。全部走隔离库
// （workcostTestApp），不碰用户正式库。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/workcost"
	"github.com/gaea/gaea/internal/office/xlsxedit"
)

func fbDicts(t *testing.T, s *workcost.Store, kind string, names ...string) {
	t.Helper()
	for _, n := range names {
		if err := s.DictAdd(workcost.DictItem{Kind: kind, Name: n}); err != nil {
			t.Fatalf("收进字典 %s 失败: %v", n, err)
		}
	}
}

func fbSave(t *testing.T, s *workcost.Store, r workcost.Resource) *workcost.Resource {
	t.Helper()
	got, err := s.SaveResource(r)
	if err != nil {
		t.Fatalf("建工料机 %s 失败: %v", r.Code, err)
	}
	return got
}

func toolRun(t *testing.T, tl interface {
	Execute(context.Context, json.RawMessage) (string, error)
}, args map[string]any) string {
	t.Helper()
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	out, err := tl.Execute(context.Background(), b)
	if err != nil {
		t.Fatalf("执行 %v 失败: %v", args["action"], err)
	}
	return out
}

func fbProject(t *testing.T, s *workcost.Store, name string, rates workcost.RateSet, profitIncludesRegulatory bool) int64 {
	t.Helper()
	pid, err := s.UpsertBillProject(name, "", "", "", "", "", &rates, profitIncludesRegulatory, 0)
	if err != nil {
		t.Fatalf("建项目失败: %v", err)
	}
	return pid
}

// save_resource：新建带码照用（此前静默换成自动码）；同身份不同地区是两行
// （此前按四要素判重被拒）；同地区同身份仍拒；码被别的资源占用仍拒。
func TestCostFeedbackSaveResourceHonorsCodeAndRegionRows(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	fbDicts(t, s, workcost.DictRegion, "四川省", "重庆市")
	fbSave(t, s, workcost.Resource{Code: "L01-3", Kind: workcost.KindLabor, Title: "普通工", Spec: "建安普工", Unit: "工日", CurrentPrice: 300, Region: "四川省"})

	labor := map[string]any{"kind": "人工", "title": "普通工", "spec": "建安普工", "unit": "工日", "current_price": 300.0, "region": "重庆市"}
	// 同身份、另一个地区：放行，且带什么码落什么码。
	dry := toolRun(t, tl, map[string]any{"action": "save_resource", "dry_run": true,
		"resource": mergeKeys(labor, map[string]any{"code": "L01-CD"})})
	if !strings.Contains(dry, "**编码**: L01-CD") || !strings.Contains(dry, "演练通过") {
		t.Fatalf("演练应把将落库的编码亮出来：%s", dry)
	}
	out := toolRun(t, tl, map[string]any{"action": "save_resource",
		"resource": mergeKeys(labor, map[string]any{"code": "L01-CD"})})
	if !strings.Contains(out, "**编码**: L01-CD") {
		t.Fatalf("显式给的编码应照用：%s", out)
	}
	for code, region := range map[string]string{"L01-CD": "重庆市", "L01-3": "四川省"} {
		r, err := s.GetResourceByCode(code)
		if err != nil || r == nil || r.Region != region {
			t.Fatalf("%s 应是 %s 的行: %+v %v", code, region, r, err)
		}
	}
	// 实测形态：ALU-CD 此前被换成自动码 ALLU。
	alu := toolRun(t, tl, map[string]any{"action": "save_resource", "resource": map[string]any{
		"code": "ALU-CD", "kind": "机械", "title": "ALLU筛分斗", "unit": "台班", "current_price": 800.0, "region": "四川省/成都市"}})
	if !strings.Contains(alu, "**编码**: ALU-CD") {
		t.Fatalf("ALU-CD 应照用：%s", alu)
	}
	// 同地区同身份、换了个码：仍拒，且话说清楚。
	_, err := clRun(t, tl, map[string]any{"action": "save_resource", "resource": mergeKeys(labor, map[string]any{"code": "L01-CD2"})})
	if err == nil || !strings.Contains(err.Error(), "同地区同身份只能有一行") || !strings.Contains(err.Error(), "工料机按地区分行") {
		t.Fatalf("同地区同身份应被拒并说明分行口径，得到：%v", err)
	}
}

// dims 的待整理不只给计数，还列条目（清单行带数字 id——classify.bills 要用）。
func TestCostFeedbackDimsListsPendingItems(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	fbSave(t, s, workcost.Resource{Code: "M1", Kind: workcost.KindMaterial, Title: "水泥", Unit: "t", CurrentPrice: 420}) // 无地区/分类/期
	if _, err := s.SaveQuota(workcost.Quota{Code: "D1", Title: "垫层", Unit: "m³", Items: []workcost.QuotaItem{{ResourceCode: "M1", Quantity: 1}}}, nil); err != nil {
		t.Fatal(err) // 无专业、无章节
	}
	var rates workcost.RateSet
	pid := fbProject(t, s, "百锦路", rates, false) // 无专业
	it, err := s.SaveBillItem(workcost.BillItem{ProjectID: pid, Code: "BJ01", Title: "垫层", Unit: "m³", Quantity: 10, QuotaCode: "D1"})
	if err != nil {
		t.Fatal(err)
	}
	out := toolRun(t, tl, map[string]any{"action": "dims"})
	for _, want := range []string{
		"**待整理明细**",
		"- 项目未标专业：#1 百锦路",
		fmt.Sprintf("- 清单无专业：#%d [BJ01] 垫层（百锦路）", it.ID),
		"- 定额未标专业：[D1] 垫层",
		"- 定额未归分部：[D1] 垫层",
		"- 工料机未标地区：[M1] 材料 水泥",
		"- 工料机未细分：[M1] 材料 水泥",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("dims 回执缺 %q：\n%s", want, out)
		}
	}
}

// 样表「清单条数定了先删末尾未用组」的做法在真实样表上走得通（写进 cost-estimate
// 技能的配方，这里钉住）：从末尾删不挪动任何引用，删完把两处合计区间收到最后
// 一个有用行，填完照样演练导回，且不再有空骨架。
func TestCostTemplateTrimTrailingGroupsRecipe(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	ws := t.TempDir()
	origCfg := ga.cfg
	ga.cfg = &config.Config{Workspace: ws}
	t.Cleanup(func() { ga.cfg = origCfg })
	fbDicts(t, s, workcost.DictSpecialty, "市政")
	fbDicts(t, s, workcost.DictRegion, "重庆市")
	labor := fbSave(t, s, workcost.Resource{Kind: workcost.KindLabor, Title: "普工", Unit: "工日", CurrentPrice: 300, Region: "重庆市"})
	c20 := fbSave(t, s, workcost.Resource{Kind: workcost.KindMaterial, Title: "商品混凝土", Spec: "C20", Unit: "m³", CurrentPrice: 400, Region: "重庆市"})

	out := toolRun(t, tl, map[string]any{"action": "template", "template": map[string]any{"name": "裁组"}})
	first := strings.SplitN(out, "\n", 2)[0]
	rel := strings.TrimSpace(strings.TrimPrefix(first, "**测算样表**: "))
	rel = rel[:strings.Index(rel, "（")]
	path := filepath.Join(ws, filepath.FromSlash(rel))

	// 15 条清单：综合单价从第 4+11×15=169 行起删 165 行；清单、工程量计算从第
	// 19 行起删 15 行；合计区间收到第 18 行（费用汇总的直接费合计在 E16）。
	if _, err := xlsxedit.ApplyOps(path, []xlsxedit.Op{
		{Type: "delete_rows", Sheet: "综合单价", Row: 169, Count: 165},
		{Type: "delete_rows", Sheet: "清单", Row: 19, Count: 15},
		{Type: "delete_rows", Sheet: "工程量计算", Row: 19, Count: 15},
		{Type: "set_formula", Sheet: "清单", Target: "I1", Formula: "=SUM(I4:I18)"},
		{Type: "set_formula", Sheet: "费用汇总", Target: "E16", Formula: "=SUM(清单!I4:I18)"},
	}); err != nil {
		t.Fatalf("裁组操作集失败: %v", err)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	set := func(sheet, cell string, v any) {
		t.Helper()
		var err error
		switch x := v.(type) {
		case string:
			err = f.SetCellStr(sheet, cell, x)
		case float64:
			err = f.SetCellFloat(sheet, cell, x, -1, 64)
		}
		if err != nil {
			t.Fatalf("填 %s!%s 失败: %v", sheet, cell, err)
		}
	}
	// 结构：只剩 15 组；第 15 组的清单行是第 158 行，清单第 18 行引它的金额列（带 ROUND）。
	if rows, _ := f.GetRows("综合单价"); len(rows) != 168 {
		t.Errorf("综合单价裁后应 168 行，实得 %d", len(rows))
	}
	if rows, _ := f.GetRows("清单"); len(rows) != 18 {
		t.Errorf("清单裁后应 18 行，实得 %d", len(rows))
	}
	for cell, want := range map[string]string{"H18": "ROUND(综合单价!H158,2)", "I1": "SUM(I4:I18)"} {
		if got, _ := f.GetCellFormula("清单", cell); got != want {
			t.Errorf("清单!%s 公式应为 %s，实得 %s", cell, want, got)
		}
	}
	if got, _ := f.GetCellFormula("费用汇总", "E16"); got != "SUM(清单!I4:I18)" {
		t.Errorf("费用汇总!E16 应收到第 18 行，实得 %s", got)
	}
	// 填第 1 组与第 15 组（各一块）。三级树表：清单行填 编码 C / 名称 D / 单位 E，
	// 定额行填 编码 C / 名称 D / 单位 E（换算系数 F 样表预置 1），明细行填 层级 B /
	// 工料机编码 C / 含量 F。
	up, bill := "综合单价", "清单"
	for _, g := range []struct {
		sumRow, quotaRow, detailRow, billRow int
		code, quota                          string
	}{{4, 5, 6, 4, "Q01", "D-A1"}, {158, 159, 160, 18, "Q15", "D-A15"}} {
		r := func(n int) string { return strconv.Itoa(n) }
		set(up, "C"+r(g.sumRow), g.code)
		set(up, "D"+r(g.sumRow), "垫层")
		set(up, "E"+r(g.sumRow), "m³")
		set(up, "C"+r(g.quotaRow), g.quota)
		set(up, "D"+r(g.quotaRow), "垫层浇筑")
		set(up, "E"+r(g.quotaRow), "m³")
		set(up, "B"+r(g.detailRow), "人工")
		set(up, "C"+r(g.detailRow), labor.Code)
		set(up, "F"+r(g.detailRow), 0.1)
		set(up, "B"+r(g.detailRow+1), "材料")
		set(up, "C"+r(g.detailRow+1), c20.Code)
		set(up, "F"+r(g.detailRow+1), 1.02)
		set(bill, "C"+r(g.billRow), g.code)
		set(bill, "D"+r(g.billRow), "垫层")
		set(bill, "F"+r(g.billRow), "m³")
		set(bill, "G"+r(g.billRow), 100.0)
	}
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	b, err := workcost.ParseProjectWorkbook(path)
	if err != nil {
		t.Fatalf("裁组填表后解析失败: %v", err)
	}
	if len(b.Items) != 2 || len(b.QuotaBlocks) != 2 {
		t.Errorf("应有 2 条清单、2 个定额块，实得 %d / %d", len(b.Items), len(b.QuotaBlocks))
	}
	if len(b.Skipped) != 0 {
		t.Errorf("裁完末尾未用组后不该再有被跳过的预置块: %+v", b.Skipped)
	}
	dry := toolRun(t, tl, map[string]any{"action": "import_project", "dry_run": true, "path": path,
		"specialty": "市政", "region": "重庆市", "period": "2026-06"})
	for _, want := range []string{"清单项 2 条", "定额块 2 个"} {
		if !strings.Contains(dry, want) {
			t.Errorf("演练回执缺 %q：\n%s", want, dry)
		}
	}
}

// mergeKeys 把 extra 并进 base（返回新表）。
func mergeKeys(base, extra map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// cost_project 列表：给总数与「还有更多」、limit 超限钳到 200 而不是回落 50、
// 地区过滤、定额列表不再打印恒为 0 的含量行、清单行带数字 id。
func TestCostFeedbackCostProjectLists(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	cp := costProjectTool{a: a}
	fbDicts(t, s, workcost.DictRegion, "四川省", "重庆市")
	fbDicts(t, s, workcost.DictSpecialty, "土壤修复")
	cd1 := fbSave(t, s, workcost.Resource{Code: "C20-CD", Kind: workcost.KindMaterial, Title: "商品混凝土", Spec: "C20", Unit: "m³", CurrentPrice: 430, Region: "四川省"})
	cd2 := fbSave(t, s, workcost.Resource{Code: "SN-CD", Kind: workcost.KindMaterial, Title: "水泥", Unit: "t", CurrentPrice: 420, Region: "四川省"})
	_ = cd2
	fbSave(t, s, workcost.Resource{Code: "C20-SN", Kind: workcost.KindMaterial, Title: "商品混凝土", Spec: "C20", Unit: "m³", CurrentPrice: 455, Region: "重庆市"})
	if _, err := s.SaveQuota(workcost.Quota{Code: "D1", Title: "垫层", Unit: "m³", Specialty: "土壤修复",
		Items: []workcost.QuotaItem{{ResourceCode: cd1.Code, Quantity: 1.02}}}, nil); err != nil {
		t.Fatal(err)
	}
	var rates workcost.RateSet
	pid := fbProject(t, s, "百锦路", rates, false)
	if _, err := s.SaveBillItem(workcost.BillItem{ProjectID: pid, Code: "BJ01", Title: "垫层", Unit: "m³", Quantity: 10, QuotaCode: "D1"}); err != nil {
		t.Fatal(err)
	}

	byRegion := toolRun(t, cp, map[string]any{"action": "resources", "region": "四川省"})
	if !strings.Contains(byRegion, "资源 2 条（共 2 条）") || strings.Contains(byRegion, "C20-SN") {
		t.Fatalf("地区过滤应只列四川的 2 条并报总数：%s", byRegion)
	}
	limited := toolRun(t, cp, map[string]any{"action": "resources", "limit": 2})
	if !strings.Contains(limited, "共 3 条，本次列 2 条；还有更多") {
		t.Fatalf("被截断时应报总数与还有更多：%s", limited)
	}
	clamped := toolRun(t, cp, map[string]any{"action": "resources", "limit": 500})
	if !strings.Contains(clamped, "资源 3 条（共 3 条）") || strings.Contains(clamped, "还有更多") {
		t.Fatalf("limit 超限应钳到 200 而不是回落 50：%s", clamped)
	}
	quotas := toolRun(t, cp, map[string]any{"action": "quotas", "limit": 5})
	if strings.Contains(quotas, "含量行 0") || !strings.Contains(quotas, "列表不带含量行") {
		t.Fatalf("定额列表不应再打印恒为 0 的含量行：%s", quotas)
	}
	items := toolRun(t, cp, map[string]any{"action": "items", "project_id": pid})
	if !strings.Contains(items, "id=") {
		t.Fatalf("清单行应带数字 id（classify/delete_items 都要用）：%s", items)
	}
}

// cost_audit：资源悬空附可改的回存骨架（整组含量行原样、悬空行标出）；定额引用
// 别的地区的资源给 INFO；费率>1 说清「5 是 5 倍」。
func TestCostFeedbackCostAudit(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	ca := costAuditTool{a: a}
	fbDicts(t, s, workcost.DictRegion, "四川省", "重庆市")
	fbDicts(t, s, workcost.DictSpecialty, "土壤修复")
	labor := fbSave(t, s, workcost.Resource{Code: "L01-3", Kind: workcost.KindLabor, Title: "普通工", Unit: "工日", CurrentPrice: 300, Region: "四川省"})
	if _, err := s.SaveQuota(workcost.Quota{Code: "D1", Title: "某工序", Unit: "m³", Specialty: "土壤修复",
		Items: []workcost.QuotaItem{{ResourceCode: labor.Code, Quantity: 0.5}, {ResourceCode: "NOI", Quantity: 0.02}}}, nil); err != nil {
		t.Fatal(err)
	}
	rates := workcost.RateSet{ManagementRate: 5}
	pid := fbProject(t, s, "旺平", rates, false)
	if err := s.UpdateBillProjectProfile(pid, "土壤修复", "重庆市", "2026-11"); err != nil {
		t.Fatalf("给项目标地区失败: %v", err)
	}
	if _, err := s.SaveBillItem(workcost.BillItem{ProjectID: pid, Code: "WP01", Title: "工序", Unit: "m³", Quantity: 10, QuotaCode: "D1"}); err != nil {
		t.Fatal(err)
	}

	out := toolRun(t, ca, map[string]any{"project_id": pid})
	for _, want := range []string{
		"[HARD] dangling_resource: 定额 D1 含量行引用资源 NOI 不存在",
		`"resource_code":"L01-3","quantity":0.5`, // 其余含量行原样带上，不丢
		`"resource_code":"NOI←改成在库编码","quantity":0.02`,
		`{"action":"save_quota","quota":{"code":"D1","items":[`,
		"[INFO] region_mismatch: 资源 L01-3「普通工」属「四川省」，与项目地区",
		"rate_over_one: 企业管理费 费率 5>1",
		"应填 0.05",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("机检回执缺 %q：\n%s", want, out)
		}
	}
	// region_mismatch 只在项目标了地区时才有意义：项目没标地区不该刷屏。
	if strings.Count(out, "region_mismatch") != 1 {
		t.Fatalf("region_mismatch 应恰 1 条：%s", out)
	}
}

// summary：分部小计、舍入口径写明、综合单价取到分再乘量、取费链走库内唯一实现
// （认「利润基数含规费」开关、逐段舍到分）。
func TestCostFeedbackSummaryDivisionAndRounding(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	cp := costProjectTool{a: a}
	res := fbSave(t, s, workcost.Resource{Code: "M1", Kind: workcost.KindMaterial, Title: "水泥", Unit: "t", CurrentPrice: 100.10})
	if _, err := s.SaveQuota(workcost.Quota{Code: "A", Title: "垫层", Unit: "m³",
		Items: []workcost.QuotaItem{{ResourceCode: res.Code, Quantity: 1}}}, nil); err != nil {
		t.Fatal(err)
	}
	rates := workcost.RateSet{ManagementRate: 0.05, RegulatoryRate: 0.02, ProfitRate: 0.04, TaxRate: 0.03}
	pid := fbProject(t, s, "舍入", rates, true) // 利润基数含规费
	it, err := s.SaveBillItem(workcost.BillItem{ProjectID: pid, Code: "WP01", Title: "垫层", Unit: "m³", Quantity: 100, QuotaCode: "A", Division: "A临建", ListPrice: 33.00})
	if err != nil {
		t.Fatal(err)
	}
	// 定额自身工程量 33.3：全精度合价 3333.33；取到分的单价 33.33 × 100 = 3333.00。
	// 原表综合单价快照故意写成 33.00：summary 逐行点名与快照的差。
	if _, err := s.AttachBillQuotaLink(it.ID, "A", 33.3); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveBillItem(workcost.BillItem{ProjectID: pid, Code: "WP02", Title: "围挡", Unit: "m", Quantity: 10, PriceOverride: 50, Division: "B措施"}); err != nil {
		t.Fatal(err)
	}

	out := toolRun(t, cp, map[string]any{"action": "summary", "project_id": pid})
	// 直接费 3333.00 + 500.00 = 3833.00；管理费 191.65；规费 76.66；
	// 利润 =（3833.00 + 191.65 + 76.66）× 4% = 164.05；税前 4265.36；税 127.96；含税 4393.32。
	for _, want := range []string{
		"综合单价=33.33 合价=3333.00",
		"导入快照=33.00（差 +0.33）",
		"与导入快照（原表综合单价）不一致 1 行，按工程量合计差 +33.00",
		"综合单价=50.00 合价=500.00",
		"── 分部小计（人材机直接费）──",
		"- A临建：3333.00（1 行）",
		"- B措施：500.00（1 行）",
		"人材机直接费=3833.00",
		"管理费=191.65｜利润=164.05｜规费=76.66｜税前=4265.36｜增值税=127.96",
		"含税合计=4393.32",
		"舍入口径：",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary 缺 %q：\n%s", want, out)
		}
	}
}

// import_project：control_price 显式给定覆盖费用汇总读到的值，回执注明来源。
func TestCostFeedbackImportProjectControlPriceOverride(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	ws := t.TempDir()
	origCfg := ga.cfg
	ga.cfg = &config.Config{Workspace: ws}
	t.Cleanup(func() { ga.cfg = origCfg })
	fbDicts(t, s, workcost.DictSpecialty, "市政")
	fbDicts(t, s, workcost.DictRegion, "重庆市")
	path := filepath.Join(ws, "样表.xlsx")
	if err := os.WriteFile(path, skill.CostEstimateSampleXLSX(), 0o644); err != nil {
		t.Fatal(err)
	}

	base := map[string]any{"action": "import_project", "dry_run": true, "path": path,
		"specialty": "市政", "region": "重庆市", "period": "2026-06"}
	out := toolRun(t, tl, base)
	if strings.Contains(out, "控制价 ¥0.") || strings.Contains(out, "疑似误读") {
		t.Fatalf("样表的控制价行金额是「—」，不应读出数：\n%s", out)
	}
	over := toolRun(t, tl, mergeKeys(base, map[string]any{"control_price": 8913200.0}))
	if !strings.Contains(over, "控制价 ¥8913200.00（调用方显式给定）") {
		t.Fatalf("显式控制价应覆盖并注明来源：\n%s", over)
	}
}
