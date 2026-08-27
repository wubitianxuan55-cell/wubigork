package costimport

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
)

func cellName(col, row int) string {
	s := ""
	for col > 0 {
		col--
		s = string(rune('A'+col%26)) + s
		col /= 26
	}
	return fmt.Sprintf("%s%d", s, row)
}

func TestParseAnalysisComponents(t *testing.T) {
	text := "人工费\n" +
		"1.混凝土基础浇筑85元/m³*0.77=65.45元\n" +
		"材料费\n" +
		"1.标志牌(含IV类反光膜)520元/m²*0.5=260元\n" +
		"2.镀锌钢管立柱9元/Kg*28.01Kg*1.03=259.65元\n" +
		"机械费\n" +
		"1.挖土方9元/m³*0.86=7.74元"
	comps := parseAnalysisComponents(text)
	if len(comps) != 4 {
		t.Fatalf("components = %d, want 4: %+v", len(comps), comps)
	}
	if comps[0].Kind != "人工" || comps[0].Title != "混凝土基础浇筑" || comps[0].Amount != 65.45 {
		t.Errorf("comp0 = %+v", comps[0])
	}
	if comps[1].Kind != "材料" || comps[1].Title != "标志牌(含IV类反光膜)" || comps[1].Amount != 260 {
		t.Errorf("comp1 = %+v", comps[1])
	}
	if comps[2].Kind != "材料" || comps[2].Amount != 259.65 {
		t.Errorf("comp2 = %+v", comps[2])
	}
	if comps[3].Kind != "机械" || comps[3].Title != "挖土方" || comps[3].Price != 9 || comps[3].Amount != 7.74 {
		t.Errorf("comp3 = %+v", comps[3])
	}

	// 合并段「人工费+机械费」与「3.9元」价格前缀不误剥。
	merged := parseAnalysisComponents("人工费+机械费\n1.护栏安装10元/m\n材料费\n1.钢筋网片（圆钢）3.9元/Kg*2.22Kg/㎡*1.02=8.83元/㎡")
	if len(merged) != 2 || merged[0].Kind != "人工+机械" || merged[0].Title != "护栏安装" {
		t.Fatalf("merged = %+v", merged)
	}
	if merged[1].Kind != "材料" || merged[1].Title != "钢筋网片（圆钢）" || merged[1].Price != 3.9 || merged[1].Amount != 8.83 {
		t.Errorf("merged material = %+v", merged[1])
	}
}

func TestParseManualFormatWorkbook(t *testing.T) {
	f := excelize.NewFile()
	sheet1 := "道路"
	if _, err := f.NewSheet(sheet1); err != nil {
		t.Fatal(err)
	}
	_ = f.DeleteSheet("Sheet1")
	rows1 := [][]interface{}{
		{"序号", "工程名称", "项目特征及工作内容", "计量单位", "工程数量", "报价", "", "综合单价分析", "人工费", "材料费", "机械费", "管理\n(3%)", "利润\n(10%)", "垫资\n(3%)"},
		{"", "", "", "", "", "含税单价\n(9%)", "含税合价", "", "", "", "", "", "", ""},
		{"土方工程", "", "", "", "", "", "", "", "", "", "", "", "", ""},
		{"1", "挖一般土方", "[项目特征]\n1.土壤类别：综合土\n2.挖土深度：2m以内\n[工作内容]\n1.土方开挖", "m³", "1", "3.79", "3.79", "机械费\n1.挖土方(甩土)3元/m³", "0", "0", "3", "0.09", "0.3", "0.09"},
		{"2", "挖一般土方", "[项目特征]\n1.土壤类别：杂填土\n2.挖土深度：4m以内\n[工作内容]\n1.土方开挖", "m³", "1", "7.59", "7.59", "机械费\n1.挖土方6元/m³", "0", "0", "6", "0.18", "0.6", "0.18"},
		{"3", "土方回填", "[项目特征]\n1.素土回填", "m³", "1", "8.85", "8.85", "机械费\n1.回填7元/m³", "0", "0", "7", "0.21", "0.7", "0.21"},
	}
	for i, r := range rows1 {
		for j, v := range r {
			_ = f.SetCellValue(sheet1, cellName(j+1, i+1), v)
		}
	}

	sheet2 := "交通"
	if _, err := f.NewSheet(sheet2); err != nil {
		t.Fatal(err)
	}
	rows2 := [][]interface{}{
		{"序号", "工程名称", "项目特征及工作内容", "计量单位", "工程数量", "报价", "", "综合单价分析", "人工费", "材料费", "机械费", "管理\n(3%)", "利润\n(10%)", "垫资\n(3%)"},
		{"", "", "", "", "", "含税单价\n(9%)", "含税合价", "", "", "", "", "", "", ""},
		{"标识标牌", "", "", "", "", "", "", "", "", "", "", "", "", ""},
		{"1", "单柱式标志牌", "[项目特征]\n1.φ800铝板\n[工作内容]\n1.安装", "套", "1", "1572.24", "1572.24", "人工费\n1.立杆标志牌安装50元\n材料费\n1.标志牌(含IV类反光膜)520元/m²*0.5=260元\n机械费\n1.挖土方9元/m³*0.86=7.74元", "118.95", "1110.96", "7.74", "37.13", "123.77", "37.13"},
	}
	for i, r := range rows2 {
		for j, v := range r {
			_ = f.SetCellValue(sheet2, cellName(j+1, i+1), v)
		}
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "市政成本测算手册.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}

	pv, err := Parse(path, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(pv.Rows) != 4 {
		t.Fatalf("rows = %d, want 4: %+v", len(pv.Rows), pv.Rows)
	}

	r0 := pv.Rows[0]
	if r0.Category != "综合单价/道路工程/土方工程" {
		t.Errorf("r0 category = %q", r0.Category)
	}
	if r0.Price != 3.79 || r0.MachineFee != 3 || r0.LaborFee != 0 || r0.MaterialFee != 0 ||
		r0.ManagementFee != 0.09 || r0.ProfitFee != 0.3 || r0.AdvanceFee != 0.09 || r0.TaxRate != 9 {
		t.Errorf("r0 fees = %+v", r0)
	}
	if len(r0.Components) != 1 || r0.Components[0].Kind != "机械" ||
		r0.Components[0].Title != "挖土方(甩土)" || r0.Components[0].Amount != 3 {
		t.Errorf("r0 components = %+v", r0.Components)
	}

	// 同名子目按特征片段去重：两个「挖一般土方」不同名。
	if pv.Rows[0].Title == pv.Rows[1].Title {
		t.Errorf("duplicate titles not disambiguated: %q == %q", pv.Rows[0].Title, pv.Rows[1].Title)
	}
	if !strings.Contains(pv.Rows[1].Title, "挖一般土方") || !strings.Contains(pv.Rows[1].Title, "挖土深度") {
		t.Errorf("r1 disambiguated title = %q", pv.Rows[1].Title)
	}
	if r3 := pv.Rows[3]; r3.Category != "综合单价/交通工程/标识标牌" || len(r3.Components) != 3 {
		t.Errorf("r3 = %+v", r3)
	}
}

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
