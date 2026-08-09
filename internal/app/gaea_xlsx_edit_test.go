package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestGaeaXlsxEdit_Validation(t *testing.T) {
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
	if _, err := a.GaeaXlsxEdit(filepath.ToSlash(rel), "Sheet1", "求和", "B1"); err == nil || !strings.Contains(err.Error(), "AI 客户端") {
		t.Fatalf("期望 AI 客户端错误，得到 %v", err)
	}
	if _, err := a.GaeaXlsxEdit("", "Sheet1", "求和", "B1"); err == nil || !strings.Contains(err.Error(), "路径") {
		t.Fatalf("期望路径错误，得到 %v", err)
	}
	if _, err := a.GaeaXlsxEdit(filepath.ToSlash(rel), "Sheet1", "", "B1"); err == nil || !strings.Contains(err.Error(), "指令") {
		t.Fatalf("期望指令错误，得到 %v", err)
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
