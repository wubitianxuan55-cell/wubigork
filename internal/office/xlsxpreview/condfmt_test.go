package xlsxpreview

// 条件格式提取回归（U4 缺口 #2）。
// excelize 通过 NewConditionalStyle 生成 dxfs 条件样式；夹具直接走官方写侧
// 产出真实形状的 styles.xml，再走 Render 验证端到端提取。

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func buildCondXLSX(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cond.xlsx")
	f := excelize.NewFile()
	defer f.Close()

	red, err := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "9C0006"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFC7CE"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	f.SetCellValue("Sheet1", "A1", "科目")
	f.SetCellValue("Sheet1", "B1", "金额")
	for i, v := range []float64{120, 80, 200} {
		cell := "B" + string(rune('2'+i))
		f.SetCellValue("Sheet1", cell, v)
	}
	// B2:B4 大于 100 与 A2:A4 等于文本 "设备" 共用红色 dxf；
	// 阈值引用单元格（$D$1）→ 不可静态判定，应计入 skipped。
	if err := f.SetConditionalFormat("Sheet1", "B2:B4", []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "greater than", Value: "100", Format: &red},
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetConditionalFormat("Sheet1", "A2:A4", []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "=", Value: `"设备"`, Format: &red},
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetConditionalFormat("Sheet1", "B2:B4", []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: "greater than", Value: "$D$1", Format: &red},
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRenderExtractsCellIsConditionalFormats(t *testing.T) {
	raw, err := Render(buildCondXLSX(t))
	if err != nil {
		t.Fatal(err)
	}
	var pr Preview
	if err := json.Unmarshal([]byte(raw), &pr); err != nil {
		t.Fatal(err)
	}
	if len(pr.Sheets) == 0 {
		t.Fatal("应有工作表")
	}
	sh := pr.Sheets[0]
	if len(sh.CondRules) != 2 {
		t.Fatalf("应提取 2 条 CellIs 规则，got %d (%+v)", len(sh.CondRules), sh.CondRules)
	}
	byRange := map[string]CondRule{}
	for _, r := range sh.CondRules {
		byRange[r.Range] = r
	}
	gt, ok := byRange["B2:B4"]
	if !ok {
		t.Fatalf("缺 B2:B4 规则: %+v", sh.CondRules)
	}
	if gt.Op != "greaterThan" || len(gt.Formulas) != 1 || gt.Formulas[0] != "100" {
		t.Errorf("B2:B4 规则口径错: %+v", gt)
	}
	if gt.Fill != "FFC7CE" || gt.FontColor != "9C0006" || !gt.Bold {
		t.Errorf("dxf 样式应带出（6 位 hex 口径）: %+v", gt)
	}
	eq, ok := byRange["A2:A4"]
	if !ok {
		t.Fatalf("缺 A2:A4 规则: %+v", sh.CondRules)
	}
	if eq.Op != "equal" || len(eq.Formulas) != 1 || eq.Formulas[0] != `"设备"` {
		t.Errorf("A2:A4 规则口径错: %+v", eq)
	}
	if sh.CondSkipped != 1 {
		t.Errorf("引用单元格阈值的规则应计入 skipped=1，got %d", sh.CondSkipped)
	}
}

func TestRenderNoConditionalFormatOmitsFields(t *testing.T) {
	raw, err := Render(buildXLSX(t))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "condRules") || strings.Contains(raw, "condSkipped") {
		t.Error("无条件格式时 JSON 不应携带条件格式字段（omitempty）")
	}
}
