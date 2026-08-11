package costimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
)

func newTestStore(t *testing.T) *cost.Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	return cost.Open(gdb)
}

func TestParseCSV_MappingAndMatch(t *testing.T) {
	store := newTestStore(t)
	if err := store.Save(cost.Entry{Name: "hp300", Title: "HP300 高频液压振动锤", Category: "机械", Unit: "台班", Price: 3200, Source: "市场询价", Status: "现行"}); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	csvPath := filepath.Join(dir, "报价单.csv")
	csv := "序号,材料名称,规格型号,单位,单价(元),供应商,备注\n" +
		"1,HP300 高频液压振动锤,300kW,台班,\"3,200.00\",XX租赁,\n" +
		"2,P.O 42.5 水泥,,吨,480 元,海螺,\n" +
		"3,临时材料,,,,-,\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}

	pv, err := Parse(csvPath, store)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(pv.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d: %+v", len(pv.Rows), pv.Rows)
	}

	// 第一行：映射 + 价格归一化 + 命中既有条目（覆盖更新）。
	r0 := pv.Rows[0]
	if r0.Title != "HP300 高频液压振动锤" || r0.Spec != "300kW" || r0.Unit != "台班" {
		t.Errorf("row0 mapped wrong: %+v", r0)
	}
	if r0.Price != 3200 {
		t.Errorf("price parse = %v, want 3200", r0.Price)
	}
	if r0.ExistingName != "hp300" || !strings.Contains(r0.MatchNote, "覆盖更新") {
		t.Errorf("expected existing match on hp300, got %+v", r0)
	}
	if r0.Skip {
		t.Errorf("row0 should not be skipped: %+v", r0)
	}

	// 第二行：新增条目。
	if r1 := pv.Rows[1]; r1.MatchNote != "新增" || r1.Price != 480 || r1.Source != "海螺" {
		t.Errorf("row1 wrong: %+v", r1)
	}

	// 第三行：缺单价 → Skip。
	if r2 := pv.Rows[2]; !r2.Skip || !strings.Contains(r2.SkipReason, "单价") {
		t.Errorf("row2 should be skipped: %+v", r2)
	}
}

func TestParseXLSX(t *testing.T) {
	dir := t.TempDir()
	xlsxPath := filepath.Join(dir, "测算表.xlsx")
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", "科目")
	_ = f.SetCellValue(sheet, "B1", "单位")
	_ = f.SetCellValue(sheet, "C1", "单价")
	_ = f.SetCellValue(sheet, "D1", "规格")
	_ = f.SetCellValue(sheet, "A2", "挖掘机 220")
	_ = f.SetCellValue(sheet, "B2", "台班")
	_ = f.SetCellValue(sheet, "C2", 2600)
	_ = f.SetCellValue(sheet, "D2", "220kW")
	if err := f.SaveAs(xlsxPath); err != nil {
		t.Fatal(err)
	}

	pv, err := Parse(xlsxPath, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(pv.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(pv.Rows))
	}
	r := pv.Rows[0]
	if r.Title != "挖掘机 220" || r.Unit != "台班" || r.Price != 2600 || r.Spec != "220kW" {
		t.Errorf("xlsx row wrong: %+v", r)
	}
}

func TestParseSkipsTitleRowBeforeHeader(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "报价单-带标题行.csv")
	csv := "XX 项目报价单\n" +
		"序号,材料名称,规格型号,单位,单价(元),备注\n" +
		"1,HP300 高频液压振动锤,300kW,台班,3200,\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}

	pv, err := Parse(csvPath, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(pv.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d: %+v", len(pv.Rows), pv.Rows)
	}
	r := pv.Rows[0]
	if r.Title != "HP300 高频液压振动锤" || r.Unit != "台班" || r.Price != 3200 {
		t.Errorf("row wrong: %+v", r)
	}
	if !strings.Contains(pv.Message, "已跳过前 1 行") {
		t.Errorf("expected skip-title message, got %q", pv.Message)
	}
}

func TestParseSingleCellTitleRowNotMistakenForHeader(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "报价单-单格标题.csv")
	csv := "材料报价清单\n" +
		"材料名称,规格型号,单位,单价(元),备注\n" +
		"P.O 42.5 水泥,,吨,480,\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}

	pv, err := Parse(csvPath, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(pv.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d: %+v", len(pv.Rows), pv.Rows)
	}
	r := pv.Rows[0]
	if r.Title != "P.O 42.5 水泥" || r.Price != 480 {
		t.Errorf("row wrong: %+v", r)
	}
}

func TestParseVerticalParamTable(t *testing.T) {
	// 复刻真实「三轴搅拌桩成本测算表」结构：标题行 + 说明行后是
	// 竖排参数表（参数名|数值|说明|单位），无横向表头。
	dir := t.TempDir()
	xlsxPath := filepath.Join(dir, "三轴搅拌桩成本测算表.xlsx")
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	rows := [][]string{
		{"三轴搅拌桩（SMW工法）成本测算表 —— 测算说明与参数假设"},
		{"适用桩型：Φ850@600（三轴，中心距600mm，套打一孔）｜水泥掺量20%"},
		{"一、桩型几何参数"},
		{"桩径 D (mm)", "850", "设计桩径", "mm"},
		{"水泥掺量 (%)", "0.2", "水泥质量 / 土体质量", "—"},
		{"土体密度 (t/m³)", "1.8", "天然土体密度", "t/m³"},
		{"二、材料参数（单价均为到现场价）"},
		{"水泥单价 P.O42.5 散装 (元/t)", "450", "华东市场参考 420~480", "元/t"},
		{"膨润土单价 (元/t)", "900", "市场参考 800~1000", "元/t"},
		{"外加剂单价 (元/t)", "3500", "市场参考", "元/t"},
		{"水费（每幅·米） (元)", "1", "制浆用水分摊", "元"},
		{"三、机械、人工及其他参数"},
		{"桩机台班费（含制浆站/泵/空压机/吊车配套） (元/台班)", "6500", "三轴搅拌桩机租赁+折旧+动力", "元/台班"},
		{"班组人工费 (元/台班)", "1200", "机长+操作手+电工+辅助工约6人", "元/台班"},
		{"置换土外运单价 (元/m³)", "67", "装车+运输+消纳", "元/m³"},
		{"检测费（每幅·米） (元)", "10", "试块强度、28d无侧限抗压等分摊", "元"},
		{"四、综合费率"},
		{"管理费率 (%)", "0.04", "按直接费计", "—"},
		{"五、主要测算结果（自动）"},
		{"综合单价（每幅·米，含税） (元)", "856.73", "", ""},
		{"折合每立方米综合单价 (元/m³)", "573.11", "", ""},
	}
	for i, r := range rows {
		for j, v := range r {
			cell, err := excelize.CoordinatesToCellName(j+1, i+1)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := f.SaveAs(xlsxPath); err != nil {
		t.Fatal(err)
	}

	pv, err := Parse(xlsxPath, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(pv.Rows) != 10 {
		t.Fatalf("expected 10 price rows, got %d: %+v", len(pv.Rows), pv.Rows)
	}
	byTitle := map[string]Row{}
	for _, r := range pv.Rows {
		byTitle[r.Title] = r
	}
	for title, wantPrice := range map[string]float64{
		"水泥单价 P.O42.5 散装 (元/t)":                450,
		"膨润土单价 (元/t)":                      900,
		"外加剂单价 (元/t)":                      3500,
		"水费（每幅·米） (元)":                     1,
		"桩机台班费（含制浆站/泵/空压机/吊车配套） (元/台班)": 6500,
		"班组人工费 (元/台班)":                     1200,
		"置换土外运单价 (元/m³)":                   67,
		"检测费（每幅·米） (元)":                    10,
		"综合单价（每幅·米，含税） (元)":               856.73,
		"折合每立方米综合单价 (元/m³)":               573.11,
	} {
		r, ok := byTitle[title]
		if !ok {
			t.Errorf("缺少条目 %q", title)
			continue
		}
		if r.Price != wantPrice {
			t.Errorf("%s price = %v, want %v", title, r.Price, wantPrice)
		}
		if r.Skip {
			t.Errorf("%s should not be skipped: %s", title, r.SkipReason)
		}
	}
	// 自动归类：综合单价/材料/机械/人工/运输/检测。
	for title, wantCat := range map[string]string{
		"水泥单价 P.O42.5 散装 (元/t)":                "材料",
		"桩机台班费（含制浆站/泵/空压机/吊车配套） (元/台班)": "机械",
		"班组人工费 (元/台班)":                        "人工",
		"置换土外运单价 (元/m³)":                      "运输",
		"检测费（每幅·米） (元)":                       "检测",
		"综合单价（每幅·米，含税） (元)":                 "综合单价",
	} {
		if r, ok := byTitle[title]; !ok || r.Category != wantCat {
			t.Errorf("%s category = %q, want %q", title, byTitle[title].Category, wantCat)
		}
	}
	if !strings.Contains(pv.Message, "纵向参数表") {
		t.Errorf("expected vertical-table message, got %q", pv.Message)
	}
}

// TestSmokeParseRealCostXlsx 用真实工作区测算表验证解析（默认跳过）：
//   GAEA_SMOKE_COST_XLSX=<xlsx 路径> go test ./internal/gaea/costimport -run TestSmokeParseRealCostXlsx -v
func TestSmokeParseRealCostXlsx(t *testing.T) {
	src := os.Getenv("GAEA_SMOKE_COST_XLSX")
	if src == "" {
		t.Skip("未设置 GAEA_SMOKE_COST_XLSX")
	}
	pv, err := Parse(src, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(pv.Rows) == 0 {
		t.Fatalf("真实文件没有解析出任何条目: %+v", pv)
	}
	t.Logf("rows=%d message=%q", len(pv.Rows), pv.Message)
	for _, r := range pv.Rows {
		t.Logf("  %q | ¥%v/%s | %s", r.Title, r.Price, r.Unit, r.Source)
	}
}

func TestRawTableAndSlugConsistency(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.tsv")
	if err := os.WriteFile(p, []byte("名称\t单位\t价格\n水泥\t吨\t480\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cols, rows, err := RawTable(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 3 || len(rows) != 1 || rows[0][0] != "水泥" {
		t.Errorf("RawTable = %v %v", cols, rows)
	}
	// 导入行名称与 cost.SlugName 一致（保证与 cost_save 覆盖同一键）。
	if got := cost.SlugName("P.O 42.5 水泥"); got != "p-o-42-5-水泥" {
		t.Errorf("slug = %q", got)
	}
}
