package app

// office_cost_library_template_test.go — cost_library 的测算样表闭环（2026-11-25
// 删渲染器拍板后的形态）：template 复制样表（版式真源）→ 按契约填两组 →
// import_project 导回 → 库里还原成 1:N 挂接。样表来源三层都钉住：内嵌种子、
// 工作区覆盖、golden 解析。全部走隔离库；工作区指到临时目录，样表落在它的
// .gaea/exports 下，不碰仓库。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

// TestCostLibrarySampleGolden 内嵌样表必须过解析链：这是「样表=版式真源」的
// 凭据——样表一旦改到导不回，template 出口就是废纸。
func TestCostLibrarySampleGolden(t *testing.T) {
	path := filepath.Join(t.TempDir(), "样表.xlsx")
	if err := os.WriteFile(path, skill.CostEstimateSampleXLSX(), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := workcost.ParseProjectWorkbook(path)
	if err != nil {
		t.Fatalf("内嵌样表解析失败: %v", err)
	}
	if len(b.Sheets) != 6 || b.Project != "成本测算模版" {
		t.Fatalf("样表应 6 张表、封面标题=成本测算模版，得到 %v / %q", b.Sheets, b.Project)
	}
	if len(b.Resources) != 60 {
		t.Fatalf("样表应预填 60 条样例工料机，得到 %d", len(b.Resources))
	}
	if b.Fee.ManagementRate != 0.05 || b.Fee.ProfitRate != 0.04 || b.Fee.RegulatoryRate != 0 || b.Fee.TaxRate != 0.09 {
		t.Errorf("样表费率样例应 5%%/4%%/0%%/9%%，得到 %+v", b.Fee)
	}
	if len(b.Skipped) != 0 {
		t.Errorf("样表不应有被跳过的行: %+v", b.Skipped)
	}
	// 两层 ROUND 预置在样表里（2026-11-26：样表由用户用 gaea 改好，换作内置种子）：
	// 定额单价舍到分、清单综合单价舍到分。样表哪一层丢了，工作簿与库现算就又出
	// 尾差——三场实测都为此对账了好几轮。
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("打开样表失败: %v", err)
	}
	defer func() { _ = f.Close() }()
	for _, tc := range []struct{ sheet, cell, want string }{
		{"综合单价", "M5", "ROUND(SUM(N6:N9),2)"},
		{"综合单价", "M10", "ROUND(SUM(N11:N14),2)"},
		{"综合单价", "M324", "ROUND(SUM(N325:N328),2)"},
		{"清单", "H4", "ROUND(综合单价!S4,2)"},
		{"清单", "H33", "ROUND(综合单价!S323,2)"},
	} {
		if got, _ := f.GetCellFormula(tc.sheet, tc.cell); got != tc.want {
			t.Errorf("样表 %s!%s 公式应为 %s，实得 %s", tc.sheet, tc.cell, tc.want, got)
		}
	}
}

// TestCostLibraryTemplateCopiesSample template 动作=复制样表：内嵌种子兜底、
// 工作区覆盖优先、路径可指定；回执带导入契约与三道门。
func TestCostLibraryTemplateCopiesSample(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	tl := costLibraryTool{a: a}
	ws := t.TempDir()
	origCfg := ga.cfg
	ga.cfg = &config.Config{Workspace: ws}
	t.Cleanup(func() { ga.cfg = origCfg })

	out := clMust(t, tl, map[string]any{"action": "template", "template": map[string]any{
		"name": "南岸测算",
	}}, "**测算样表**")
	for _, want := range []string{"导入契约", "xlsx_recalc", "xlsx_audit", "dry_run=true", "内置样表"} {
		if !strings.Contains(out, want) {
			t.Errorf("回执缺 %q：%s", want, out)
		}
	}
	// 回执给的是工作区相对路径；按它找回文件，字节应与种子一致。
	first := strings.SplitN(out, "\n", 2)[0]
	rel := strings.TrimSpace(strings.TrimPrefix(first, "**测算样表**: "))
	rel = rel[:strings.Index(rel, "（")]
	if rel == "" || filepath.IsAbs(rel) || !strings.HasPrefix(rel, ".gaea/exports/南岸测算-") {
		t.Fatalf("回执应给出工作区内的相对路径，得到 %q", first)
	}
	path := filepath.Join(ws, filepath.FromSlash(rel))
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("样表应已落盘: %v", err)
	}
	if string(got) != string(skill.CostEstimateSampleXLSX()) {
		t.Error("复制语义：落盘样表应与内嵌种子逐字节一致")
	}

	// 指定路径（顶层 path 同义）。
	out2 := clMust(t, tl, map[string]any{"action": "template", "path": ".gaea/work/样表副本.xlsx"}, "**测算样表**")
	if !strings.Contains(out2, ".gaea/work/样表副本.xlsx") {
		t.Errorf("指定路径应被尊重：%s", out2)
	}
	if _, err := os.Stat(filepath.Join(ws, ".gaea", "work", "样表副本.xlsx")); err != nil {
		t.Errorf("指定路径的样表应落盘: %v", err)
	}

	// 工作区覆盖：.gaea/skills/cost-estimate/样表 存在时优先于种子。
	overrideDir := filepath.Join(ws, ".gaea", "skills", "cost-estimate")
	if err := os.MkdirAll(overrideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	override := filepath.Join(overrideDir, "样表-成本测算模版.xlsx")
	f := excelize.NewFile()
	if err := f.SaveAs(override); err != nil {
		t.Fatalf("写工作区覆盖样表失败: %v", err)
	}
	_ = f.Close()
	out3 := clMust(t, tl, map[string]any{"action": "template"}, "**测算样表**")
	if !strings.Contains(out3, "工作区样表") {
		t.Errorf("覆盖态回执应注明来源=工作区样表：%s", out3)
	}
	third := strings.SplitN(out3, "\n", 2)[0]
	rel3 := third[strings.Index(third, ": ")+2:]
	rel3 = rel3[:strings.Index(rel3, "（")]
	got3, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(rel3)))
	if err != nil {
		t.Fatal(err)
	}
	want3, err := os.ReadFile(override)
	if err != nil {
		t.Fatal(err)
	}
	if string(got3) != string(want3) {
		t.Error("工作区覆盖样表应优先于内嵌种子")
	}
}

// TestCostLibraryTemplateRoundTrip 样表闭环：拿样表 → 按契约填两组 →
// import_project 导回 → 库里还原 1:N 挂接与系数。填法=「复制组、填关键列」，
// 与真实模型在样表上的动作同构。
func TestCostLibraryTemplateRoundTrip(t *testing.T) {
	a, cleanup := workcostTestApp(t)
	defer cleanup()
	s := a.hubWorkcostStore()
	tl := costLibraryTool{a: a}
	ws := t.TempDir()
	origCfg := ga.cfg
	ga.cfg = &config.Config{Workspace: ws}
	t.Cleanup(func() { ga.cfg = origCfg })

	for _, d := range []struct{ kind, name string }{
		{workcost.DictSpecialty, "市政"}, {workcost.DictRegion, "重庆中心城区"},
	} {
		clMust(t, tl, map[string]any{"action": "dict", "dict": map[string]any{"op": "add", "kind": d.kind, "name": d.name}}, "已新增")
	}
	save := func(r workcost.Resource) *workcost.Resource {
		t.Helper()
		got, err := s.SaveResource(r)
		if err != nil {
			t.Fatalf("建工料机 %s 失败: %v", r.Title, err)
		}
		return got
	}
	labor := save(workcost.Resource{Kind: workcost.KindLabor, Title: "普工", Unit: "工日", CurrentPrice: 300, Region: "重庆中心城区"})
	c20 := save(workcost.Resource{Kind: workcost.KindMaterial, Title: "商品混凝土", Spec: "C20", Unit: "m³", CurrentPrice: 400, Region: "重庆中心城区"})
	exc := save(workcost.Resource{Kind: workcost.KindMachine, Title: "挖掘机", Unit: "台班", CurrentPrice: 1000, Region: "重庆中心城区"})

	out := clMust(t, tl, map[string]any{"action": "template", "template": map[string]any{"name": "南岸测算"}}, "**测算样表**")
	first := strings.SplitN(out, "\n", 2)[0]
	rel := strings.TrimSpace(strings.TrimPrefix(first, "**测算样表**: "))
	rel = rel[:strings.Index(rel, "（")]
	path := filepath.Join(ws, filepath.FromSlash(rel))

	// 按契约填两组：Q01 挂两条定额（第二条系数 0.5），Q02 挂一条；封面标题改
	// 真实项目名。样表几何与旧渲染模版一致（每组 11 行），填同样的关键列。
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("打开样表失败: %v", err)
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
	set("封面", "A2", "南岸测算")
	up, bill := "综合单价", "清单"
	set(up, "B4", "Q01")
	set(up, "C4", "垫层")
	set(up, "D4", "m³")
	set(up, "D5", "m³")
	set(up, "F5", "D-A1")
	set(up, "G5", "垫层浇筑")
	set(up, "E6", "人工")
	set(up, "I6", labor.Code)
	set(up, "L6", 0.1)
	set(up, "E7", "材料")
	set(up, "I7", c20.Code)
	set(up, "L7", 1.02)
	set(up, "D10", "m³")
	set(up, "F10", "D-A2")
	set(up, "G10", "垫层夯实")
	set(up, "H10", 0.5)
	set(up, "E11", "机械")
	set(up, "I11", exc.Code)
	set(up, "L11", 0.02)
	set(up, "B15", "Q02")
	set(up, "C15", "面层")
	set(up, "D15", "m²")
	set(up, "D16", "m²")
	set(up, "F16", "D-B1")
	set(up, "G16", "面层摊铺")
	set(up, "E17", "材料")
	set(up, "I17", c20.Code)
	set(up, "L17", 0.03)
	set(up, "E18", "人工")
	set(up, "I18", labor.Code)
	set(up, "L18", 0.05)
	set(bill, "B4", "A路基")
	set(bill, "C4", "Q01")
	set(bill, "D4", "垫层")
	set(bill, "F4", "m³")
	set(bill, "G4", 100.0)
	set(bill, "B5", "B路面")
	set(bill, "C5", "Q02")
	set(bill, "D5", "面层")
	set(bill, "F5", "m²")
	set(bill, "G5", 2000.0)
	if err := f.Save(); err != nil {
		t.Fatalf("保存填好的样表失败: %v", err)
	}
	_ = f.Close()

	// 导回：先演练（回执报出定额块），再正式。
	scope := map[string]any{"specialty": "市政", "region": "重庆中心城区", "period": "2026-06"}
	dry := map[string]any{"action": "import_project", "dry_run": true, "path": path}
	for k, v := range scope {
		dry[k] = v
	}
	clMust(t, tl, dry, "定额块 3 个")
	if projects, _ := s.BillProjects(); len(projects) != 0 {
		t.Fatalf("演练不应落库，得到 %d 个项目", len(projects))
	}
	real := map[string]any{"action": "import_project", "path": path}
	for k, v := range scope {
		real[k] = v
	}
	clMust(t, tl, real, "**落库**")

	// 库里：项目（封面标题）与口径、三条定额各带含量行、1:N 挂接带系数。
	projects, _ := s.BillProjects()
	var pid int64
	for _, p := range projects {
		if p.Name == "南岸测算" {
			pid = p.ID
			if p.Specialty != "市政" || p.Region != "重庆中心城区" || p.PricePeriod != "2026-06" {
				t.Errorf("项目口径应随导入口径落库，得到 %+v", p)
			}
			// 费率取自样表费用汇总的费率列（样例口径 5%/0%/4%/9%）。
			if p.ManagementRate != 0.05 || p.TaxRate != 0.09 {
				t.Errorf("项目费率应取自样表费用汇总（5%%/9%%），得到 %+v", p)
			}
		}
	}
	if pid == 0 {
		t.Fatalf("导回后应有项目「南岸测算」（封面标题），得到 %+v", projects)
	}
	for code, want := range map[string]int{"D-A1": 2, "D-A2": 1, "D-B1": 2} {
		q, err := s.GetQuota(code)
		if err != nil {
			t.Fatalf("定额 %s 不在库: %v", code, err)
		}
		if len(q.Items) != want || q.Specialty != "市政" {
			t.Errorf("定额 %s 应有 %d 行含量、专业市政，得到 %d 行、专业 %q", code, want, len(q.Items), q.Specialty)
		}
	}
	q01, err := s.FindBillItem(pid, "Q01")
	if err != nil || q01 == nil {
		t.Fatalf("清单 Q01 不在库: %v", err)
	}
	if strings.Join(q01.QuotaCodes, ",") != "D-A1,D-A2" || q01.LinkQuantities["D-A1"] != 100 || q01.LinkQuantities["D-A2"] != 50 {
		t.Errorf("Q01 应挂 D-A1+D-A2、工程量 100/50，得到 %v %v", q01.QuotaCodes, q01.LinkQuantities)
	}
	q02, _ := s.FindBillItem(pid, "Q02")
	if q02 == nil || strings.Join(q02.QuotaCodes, ",") != "D-B1" || q02.LinkQuantities["D-B1"] != 2000 || q02.Division != "B路面" {
		t.Errorf("Q02 应挂 D-B1、工程量 2000、分部 B路面，得到 %+v", q02)
	}
}
