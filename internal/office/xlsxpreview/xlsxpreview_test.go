package xlsxpreview

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// buildXLSX 用 excelize 构造含样式/公式/合并/多 sheet 的工作簿。
func buildXLSX(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "preview.xlsx")
	f := excelize.NewFile()
	defer f.Close()

	// Sheet1：表头 + 数据 + 公式 + 合并 + 样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"4472C4"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Border: []excelize.Border{
			{Type: "left", Style: 1}, {Type: "right", Style: 1},
			{Type: "top", Style: 1}, {Type: "bottom", Style: 1},
		},
	})
	pctStyle, _ := f.NewStyle(&excelize.Style{CustomNumFmt: strPtr("0.00%")})

	f.SetSheetName("Sheet1", "预算")
	f.SetCellValue("预算", "A1", "项目")
	f.SetCellValue("预算", "B1", "金额")
	f.SetCellValue("预算", "A2", "设备")
	f.SetCellValue("预算", "B2", 120.5)
	f.SetCellValue("预算", "A3", "人工")
	f.SetCellValue("预算", "B3", 80)
	f.SetCellFormula("预算", "B4", "SUM(B2:B3)")
	f.SetCellValue("预算", "A4", "合计")
	f.SetCellStyle("预算", "A1", "B1", headerStyle)
	f.SetCellStyle("预算", "B2", "B3", pctStyle)
	f.MergeCell("预算", "A5", "B5")
	f.SetCellValue("预算", "A5", "合并单元格")
	f.SetColWidth("预算", "A", "A", 16)
	f.SetColWidth("预算", "B", "B", 14)

	// Sheet2：第二个工作表
	f.NewSheet("明细")
	f.SetCellValue("明细", "A1", "日期")
	f.SetCellValue("明细", "B1", "备注")
	f.SetCellValue("明细", "A2", "2026-08-09")

	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func strPtr(s string) *string { return &s }

func TestRender(t *testing.T) {
	j, err := Render(buildXLSX(t))
	if err != nil {
		t.Fatal(err)
	}
	var pr Preview
	if err := json.Unmarshal([]byte(j), &pr); err != nil {
		t.Fatalf("预览不是合法 JSON: %v", err)
	}
	if len(pr.Sheets) != 2 {
		t.Fatalf("sheets = %d, want 2", len(pr.Sheets))
	}
	sh := pr.Sheets[0]
	if sh.Name != "预算" {
		t.Fatalf("sheet name = %q", sh.Name)
	}
	// 表头样式：加粗 + 填充 + 居中 + 边框
	var hdr *Cell
	for _, row := range sh.Rows {
		for i := range row {
			if row[i].Ref == "A1" {
				hdr = &row[i]
			}
		}
	}
	if hdr == nil {
		t.Fatal("缺少 A1")
	}
	if hdr.Style == nil || !hdr.Style.Bold || hdr.Style.Fill != "4472C4" || hdr.Style.Align != "center" || !hdr.Style.Border {
		t.Errorf("A1 样式不正确: %+v", hdr.Style)
	}
	// 公式单元格
	var sum *Cell
	for _, row := range sh.Rows {
		for i := range row {
			if row[i].Ref == "B4" {
				sum = &row[i]
			}
		}
	}
	if sum == nil || sum.Formula == "" {
		t.Fatalf("B4 应有公式: %+v", sum)
	}
	if !strings.Contains(sum.Formula, "SUM") {
		t.Errorf("公式内容 = %q", sum.Formula)
	}
	// 合并
	foundMerge := false
	for _, m := range sh.Merged {
		if m == "A5:B5" {
			foundMerge = true
		}
	}
	if !foundMerge {
		t.Errorf("缺少合并 A5:B5: %v", sh.Merged)
	}
	// 列宽
	if sh.ColWidths["A"] != 16 || sh.ColWidths["B"] != 14 {
		t.Errorf("列宽不正确: %v", sh.ColWidths)
	}
	// 第二张表
	if pr.Sheets[1].Name != "明细" {
		t.Errorf("sheet2 name = %q", pr.Sheets[1].Name)
	}
}

func TestRenderTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.xlsx")
	f := excelize.NewFile()
	// 超过 MaxRows 的稀疏行（excelize 用行号定位，无需逐行写入）
	if err := f.SetCellValue("Sheet1", "A1", "start"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetCellValue("Sheet1", "A2100", "end"); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()

	j, err := Render(path)
	if err != nil {
		t.Fatal(err)
	}
	var pr Preview
	if err := json.Unmarshal([]byte(j), &pr); err != nil {
		t.Fatal(err)
	}
	if !pr.Sheets[0].Truncated {
		t.Error("应标记截断")
	}
	if len(pr.Sheets[0].Rows) != MaxRows {
		t.Errorf("rows = %d, want %d", len(pr.Sheets[0].Rows), MaxRows)
	}
}

func TestNeedsRecalc(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "needs.xlsx")
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", 100)
	f.SetCellFormula("Sheet1", "A2", "A1*2")
	f.SetCellValue("Sheet1", "B1", "text")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()

	need, err := NeedsRecalc(path)
	if err != nil {
		t.Fatal(err)
	}
	if !need {
		t.Fatal("含无缓存值公式的工作簿应判定为需要重算")
	}

	plain := filepath.Join(dir, "plain.xlsx")
	f2 := excelize.NewFile()
	f2.SetCellValue("Sheet1", "A1", 42)
	if err := f2.SaveAs(plain); err != nil {
		t.Fatal(err)
	}
	f2.Close()
	need, err = NeedsRecalc(plain)
	if err != nil {
		t.Fatal(err)
	}
	if need {
		t.Fatal("无公式工作簿不应需要重算")
	}
}
