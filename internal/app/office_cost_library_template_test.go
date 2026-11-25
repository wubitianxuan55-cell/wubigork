package app

// office_cost_library_template_test.go — cost_library 的测算模版闭环（2026-11-25）：
// template 按库生成六表模版 → 按填法填两组 → import_project 导回 → 库里还原成
// 1:N 挂接。口径过规矩：字典外的专业/地区、非年月的期数当场拒。全部走隔离库；
// 工作区指到临时目录，模版落在它的 .gaea/exports 下，不碰仓库。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/workcost"
)

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
		{workcost.DictSpecialty, "市政"}, {workcost.DictRegion, "重庆中心城区"}, {workcost.DictRegion, "成都市区"},
	} {
		clMust(t, tl, map[string]any{"action": "dict", "dict": map[string]any{"op": "add", "kind": d.kind, "name": d.name}}, "已新增")
	}
	// 最新项目带费率：模版的费率口径取自它。
	rates := workcost.RateSet{ManagementRate: 0.05, RegulatoryRate: 0.02, ProfitRate: 0.04, TaxRate: 0.03}
	if _, err := s.UpsertBillProject("新项目Q", "", "", "", "", "", &rates, false, 0); err != nil {
		t.Fatalf("建带费率的项目失败: %v", err)
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
	save(workcost.Resource{Kind: workcost.KindMaterial, Title: "中砂", Unit: "m³", CurrentPrice: 110, Region: "成都市区"})

	// 口径不过规矩：字典外的专业、非年月的期数，当场拒、不生成。
	clMustFail(t, tl, map[string]any{"action": "template", "template": map[string]any{"specialty": "不存在专业"}}, "不在专业字典里")
	clMustFail(t, tl, map[string]any{"action": "template", "template": map[string]any{"period": "2026年第7期"}}, "不是年月写法")

	out := clMust(t, tl, map[string]any{"action": "template", "template": map[string]any{
		"specialty": "市政", "region": "重庆中心城区", "name": "南岸测算",
	}}, "**测算模版**")
	for _, want := range []string{"专业 市政｜地区 重庆中心城区", "已从造价库预填 3 条", "取自库内最近项目「新项目Q」", "cost-estimate 技能"} {
		if !strings.Contains(out, want) {
			t.Errorf("回执缺 %q：%s", want, out)
		}
	}
	// 回执给的是工作区相对路径；按它找回文件。
	first := strings.SplitN(out, "\n", 2)[0]
	rel := strings.TrimSpace(strings.TrimPrefix(first, "**测算模版**: "))
	if rel == "" || filepath.IsAbs(rel) || !strings.HasPrefix(rel, ".gaea/exports/南岸测算-") {
		t.Fatalf("回执应给出工作区内的相对路径，得到 %q", first)
	}
	path := filepath.Join(ws, filepath.FromSlash(rel))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("模版文件应已落盘: %v", err)
	}

	// 按填法填两组：Q01 挂两条定额（第二条系数 0.5），Q02 挂一条；其余组留空。
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("打开模版失败: %v", err)
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
		t.Fatalf("保存填好的模版失败: %v", err)
	}
	_ = f.Close()

	// 导回：先演练（回执报出定额块），再正式。
	scope := map[string]any{"specialty": "市政", "region": "重庆中心城区", "period": "2026-06"}
	dry := map[string]any{"action": "import_project", "dry_run": true, "path": path}
	for k, v := range scope {
		dry[k] = v
	}
	clMust(t, tl, dry, "定额块 3 个")
	if projects, _ := s.BillProjects(); len(projects) != 1 {
		t.Fatalf("演练不应落库，得到 %d 个项目", len(projects))
	}
	real := map[string]any{"action": "import_project", "path": path}
	for k, v := range scope {
		real[k] = v
	}
	clMust(t, tl, real, "**落库**")

	// 库里：项目（封面标题=文件名）与口径、三条定额各带自己的含量行、1:N 挂接带系数。
	projects, _ := s.BillProjects()
	var pid int64
	for _, p := range projects {
		if p.Name == "南岸测算" {
			pid = p.ID
			if p.Specialty != "市政" || p.Region != "重庆中心城区" || p.PricePeriod != "2026-06" {
				t.Errorf("项目口径应随导入口径落库，得到 %+v", p)
			}
			if p.ManagementRate != 0.05 || p.TaxRate != 0.03 {
				t.Errorf("项目费率应取自模版费用汇总（最近项目口径），得到 %+v", p)
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
