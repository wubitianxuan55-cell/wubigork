package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestGaeaXlsxPlanEdit_Validation(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "edit.xlsx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "x")
	if err := f.SaveAs(rel); err != nil {
		t.Fatal(err)
	}
	f.Close()

	a := &App{core: &core{}} // core 存在、client 为 nil
	if _, err := a.GaeaXlsxPlanEdit(filepath.ToSlash(rel), "Sheet1", "求和", "B1"); err == nil || !strings.Contains(err.Error(), "AI 客户端") {
		t.Fatalf("期望 AI 客户端错误，得到 %v", err)
	}
	if _, err := a.GaeaXlsxPlanEdit("", "Sheet1", "求和", "B1"); err == nil || !strings.Contains(err.Error(), "路径") {
		t.Fatalf("期望路径错误，得到 %v", err)
	}
	if _, err := a.GaeaXlsxPlanEdit(filepath.ToSlash(rel), "Sheet1", "", "B1"); err == nil || !strings.Contains(err.Error(), "指令") {
		t.Fatalf("期望指令错误，得到 %v", err)
	}
}

func TestGaeaXlsxApplyEdit_AppliesApprovedOps(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "apply.xlsx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", 100)
	f.SetCellValue("Sheet1", "A2", 200)
	if err := f.SaveAs(rel); err != nil {
		t.Fatal(err)
	}
	f.Close()

	a := &App{core: &core{}}
	slashed := filepath.ToSlash(rel)
	opsJSON := `[{"type":"set_formula","sheet":"Sheet1","target":"A3","formula":"SUM(A1:A2)"},{"type":"set_value","sheet":"Sheet1","target":"A2","value":250}]`
	r, err := a.GaeaXlsxApplyEdit(slashed, opsJSON)
	if err != nil {
		t.Fatalf("应用失败：%v", err)
	}
	if r.Applied != 2 || !strings.Contains(r.Summary, "A3") {
		t.Fatalf("结果异常：applied=%d summary=%s", r.Applied, r.Summary)
	}
	if !strings.Contains(r.Preview, `"ref":"A3"`) {
		t.Fatal("预览未包含 A3")
	}
	f2, err := excelize.OpenFile(rel)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()
	if formula, _ := f2.GetCellFormula("Sheet1", "A3"); formula != "SUM(A1:A2)" {
		t.Fatalf("公式未落盘：%q", formula)
	}
	if v, _ := f2.GetCellValue("Sheet1", "A2"); v != "250" {
		t.Fatalf("落盘值 = %q，期望 250", v)
	}

	// 校验：空 / 非法操作集
	if _, err := a.GaeaXlsxApplyEdit(slashed, ""); err == nil || !strings.Contains(err.Error(), "操作集") {
		t.Fatalf("期望操作集为空错误，得到 %v", err)
	}
	if _, err := a.GaeaXlsxApplyEdit(slashed, "not-json"); err == nil || !strings.Contains(err.Error(), "操作集") {
		t.Fatalf("期望操作集无效错误，得到 %v", err)
	}
	if _, err := a.GaeaXlsxApplyEdit("", "[]"); err == nil || !strings.Contains(err.Error(), "路径") {
		t.Fatalf("期望路径错误，得到 %v", err)
	}
}

func TestGaeaXlsxChart_NativeEmbed(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "chart.xlsx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "城市")
	f.SetCellValue("Sheet1", "B1", "金额")
	f.SetCellValue("Sheet1", "A2", "北京")
	f.SetCellValue("Sheet1", "B2", 100)
	f.SetCellValue("Sheet1", "A3", "上海")
	f.SetCellValue("Sheet1", "B3", 200)
	f.SetCellValue("Sheet1", "A4", "广州")
	f.SetCellValue("Sheet1", "B4", 300)
	if err := f.SaveAs(rel); err != nil {
		t.Fatal(err)
	}
	f.Close()

	a := &App{core: &core{}}
	r, err := a.GaeaXlsxChart(XlsxChartInput{
		Rel: filepath.ToSlash(rel), Sheet: "Sheet1", ChartType: "bar", Title: "测试图表",
	})
	if err != nil {
		t.Fatalf("生成图表失败：%v", err)
	}
	if r.Sheet != "Sheet1" || r.Anchor != "D1" {
		t.Fatalf("锚点异常：sheet=%s anchor=%s", r.Sheet, r.Anchor)
	}
	if r.Labels != 3 || len(r.Values) != 3 || r.Values[0] != 100 {
		t.Fatalf("数据异常：labels=%d values=%v", r.Labels, r.Values)
	}
	if len(r.LabelList) != 3 || r.LabelList[0] != "北京" {
		t.Fatalf("类别异常：%v", r.LabelList)
	}
	if !strings.HasSuffix(r.Path, ".xlsx") {
		t.Fatalf("产物应为嵌入了图表的工作簿本身：%s", r.Path)
	}
	// 原生图表对象已写入：锚点处 DeleteChart 命中
	f2, err := excelize.OpenFile(rel)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()
	if err := f2.DeleteChart("Sheet1", r.Anchor); err != nil {
		t.Fatalf("图表未嵌入（DeleteChart 失败）：%v", err)
	}
	// 数据未被改动
	if v, _ := f2.GetCellValue("Sheet1", "B3"); v != "200" {
		t.Fatalf("数据被改动：B3 = %q", v)
	}
}

func TestGaeaXlsxSetCell_WriteValueAndFormula(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "set.xlsx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", 100)
	f.SetCellValue("Sheet1", "A2", 200)
	f.SetCellFormula("Sheet1", "A3", "SUM(A1:A2)")
	if err := f.SaveAs(rel); err != nil {
		t.Fatal(err)
	}
	f.Close()

	a := &App{core: &core{}}
	slashed := filepath.ToSlash(rel)

	// 直接写数值
	r, err := a.GaeaXlsxSetCell(slashed, "Sheet1", "A2", "250")
	if err != nil {
		t.Fatalf("写数值失败：%v", err)
	}
	if r.Applied != 1 || !strings.Contains(r.Summary, "A2") {
		t.Fatalf("结果异常：applied=%d summary=%s", r.Applied, r.Summary)
	}
	if !strings.Contains(r.Preview, `"ref":"A2"`) {
		t.Fatalf("预览未包含 A2：%s", r.Preview[:min(len(r.Preview), 300)])
	}
	check := func() string {
		f2, err := excelize.OpenFile(rel)
		if err != nil {
			t.Fatal(err)
		}
		defer f2.Close()
		v, _ := f2.GetCellValue("Sheet1", "A2")
		return v
	}
	if v := check(); v != "250" {
		t.Fatalf("落盘值 = %q，期望 250", v)
	}

	// 写公式（等号开头）
	if _, err := a.GaeaXlsxSetCell(slashed, "Sheet1", "B1", "=A1+A2"); err != nil {
		t.Fatalf("写公式失败：%v", err)
	}
	f3, err := excelize.OpenFile(rel)
	if err != nil {
		t.Fatal(err)
	}
	formula, _ := f3.GetCellFormula("Sheet1", "B1")
	f3.Close()
	if !strings.Contains(strings.ToUpper(formula), "A1") || !strings.Contains(strings.ToUpper(formula), "A2") {
		t.Fatalf("公式未写入：%q", formula)
	}

	// 校验
	if _, err := a.GaeaXlsxSetCell("", "Sheet1", "A1", "1"); err == nil || !strings.Contains(err.Error(), "路径") {
		t.Fatalf("期望路径错误，得到 %v", err)
	}
	if _, err := a.GaeaXlsxSetCell(slashed, "Sheet1", "1A", "1"); err == nil || !strings.Contains(err.Error(), "单元格引用") {
		t.Fatalf("期望单元格引用错误，得到 %v", err)
	}
}

func TestGaeaXlsxRowOps_InsertAndDelete(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "rows.xlsx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "one")
	f.SetCellValue("Sheet1", "A2", "two")
	f.SetCellValue("Sheet1", "A3", "three")
	f.SetCellValue("Sheet1", "B1", 1)
	f.SetCellValue("Sheet1", "B2", 2)
	f.SetCellValue("Sheet1", "B3", 3)
	f.SetCellFormula("Sheet1", "B4", "SUM(B1:B3)")
	f.MergeCell("Sheet1", "A5", "B5")
	f.SetCellValue("Sheet1", "A5", "merge")
	if err := f.SaveAs(rel); err != nil {
		t.Fatal(err)
	}
	f.Close()

	a := &App{core: &core{}}
	slashed := filepath.ToSlash(rel)

	r, err := a.GaeaXlsxRowOps(slashed, "Sheet1", "insert_before", "A2")
	if err != nil {
		t.Fatalf("插入行失败：%v", err)
	}
	if r.Applied != 1 || !strings.Contains(r.Summary, "上方插入空行") {
		t.Fatalf("插入结果异常：applied=%d summary=%s", r.Applied, r.Summary)
	}
	if !strings.Contains(r.Preview, `"value":"two"`) {
		t.Fatal("预览未包含插入后的数据")
	}

	// 落盘验证：A2 变空，旧 A2 顺延到 A3、A3 到 A4
	check := func() map[string]string {
		f2, err := excelize.OpenFile(rel)
		if err != nil {
			t.Fatal(err)
		}
		defer f2.Close()
		out := map[string]string{}
		for _, ref := range []string{"A1", "A2", "A3", "A4"} {
			v, _ := f2.GetCellValue("Sheet1", ref)
			out[ref] = v
		}
		return out
	}
	vals := check()
	if vals["A2"] != "" || vals["A3"] != "two" || vals["A4"] != "three" {
		t.Fatalf("插入后错位：%v", vals)
	}

	// 删除 A3（原 two）→ 数据上移
	if _, err := a.GaeaXlsxRowOps(slashed, "Sheet1", "delete", "A3"); err != nil {
		t.Fatalf("删除行失败：%v", err)
	}
	vals = check()
	if vals["A3"] != "three" || vals["A4"] != "" {
		t.Fatalf("删除后错位：%v", vals)
	}

	// 校验
	if _, err := a.GaeaXlsxRowOps(slashed, "Sheet1", "bad", "A1"); err == nil || !strings.Contains(err.Error(), "不支持") {
		t.Fatalf("期望不支持操作错误，得到 %v", err)
	}
	if _, err := a.GaeaXlsxRowOps(slashed, "Sheet1", "insert_before", "1A"); err == nil || !strings.Contains(err.Error(), "单元格引用") {
		t.Fatalf("期望单元格引用错误，得到 %v", err)
	}
}

func TestGaeaXlsxColOps_InsertAndDelete(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "cols.xlsx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "a1")
	f.SetCellValue("Sheet1", "B1", "b1")
	f.SetCellValue("Sheet1", "A2", "a2")
	f.SetCellValue("Sheet1", "B2", "b2")
	if err := f.SaveAs(rel); err != nil {
		t.Fatal(err)
	}
	f.Close()

	a := &App{core: &core{}}
	slashed := filepath.ToSlash(rel)

	r, err := a.GaeaXlsxColOps(slashed, "Sheet1", "insert_before", "B1")
	if err != nil {
		t.Fatalf("插入列失败：%v", err)
	}
	if r.Applied != 1 || !strings.Contains(r.Summary, "B 列左侧") {
		t.Fatalf("插入结果异常：applied=%d summary=%s", r.Applied, r.Summary)
	}
	read := func() map[string]string {
		f2, err := excelize.OpenFile(rel)
		if err != nil {
			t.Fatal(err)
		}
		defer f2.Close()
		out := map[string]string{}
		for _, ref := range []string{"A1", "B1", "C1", "D1"} {
			v, _ := f2.GetCellValue("Sheet1", ref)
			out[ref] = v
		}
		return out
	}
	vals := read()
	if vals["B1"] != "" || vals["C1"] != "b1" {
		t.Fatalf("插入后错位：%v", vals)
	}

	// 删除 B 列（空列）→ 数据左移
	if _, err := a.GaeaXlsxColOps(slashed, "Sheet1", "delete", "B1"); err != nil {
		t.Fatalf("删除列失败：%v", err)
	}
	vals = read()
	if vals["B1"] != "b1" || vals["C1"] != "" {
		t.Fatalf("删除后错位：%v", vals)
	}

	if _, err := a.GaeaXlsxColOps(slashed, "Sheet1", "bad", "A1"); err == nil || !strings.Contains(err.Error(), "不支持") {
		t.Fatalf("期望不支持操作错误，得到 %v", err)
	}
	if _, err := a.GaeaXlsxColOps(slashed, "Sheet1", "insert_before", "1A"); err == nil || !strings.Contains(err.Error(), "单元格引用") {
		t.Fatalf("期望单元格引用错误，得到 %v", err)
	}
}
