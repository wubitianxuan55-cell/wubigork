package xlsxpreview

// 条件格式提取回归（U4 缺口 #2）。
// excelize 写侧的 SetConditionalFormat 只写 cfRule 不落 styles.xml 的 dxfs 表
// （dxfId 指向内部样式 id，打开无 dxfs 内容）——真实 Excel/WPS 产物才有完整
// dxfs。故夹具分两步：excelize 造条件格式规则后，对 zip 内 styles.xml 注入
// 真实 Excel 形状的 dxfs 块，再走 Render 验证端到端提取。

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// injectDxfs 把文件内 styles.xml 的空 dxfs 替换为真实 Excel 形状的 dxf 表。
// excelize 写侧 dxfId=NewStyle 的样式 id（本夹具 red=1），故红样式放索引 1：
// id 0 = 绿（底色 C6EFCE），id 1 = 红（bold + 字色 9C0006 + 底色 FFC7CE）。
func injectDxfs(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	in, err := zip.NewReader(strings.NewReader(string(raw)), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	const dxfsXML = `<dxfs count="2">` +
		`<dxf><fill><patternFill><bgColor rgb="FFC6EFCE"/></patternFill></fill></dxf>` +
		`<dxf><font><b/><color rgb="FF9C0006"/></font><fill><patternFill><bgColor rgb="FFFFC7CE"/></patternFill></fill></dxf>` +
		`</dxfs>`
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	w := zip.NewWriter(out)
	for _, zf := range in.File {
		rd, err := zf.Open()
		if err != nil {
			t.Fatal(err)
		}
		data := new(strings.Builder)
		buf := make([]byte, 4096)
		for {
			n, err := rd.Read(buf)
			data.Write(buf[:n])
			if err != nil {
				break
			}
		}
		rd.Close()
		content := data.String()
		if zf.Name == "xl/styles.xml" {
			content = strings.Replace(content, `<dxfs count="0"></dxfs>`, dxfsXML, 1)
			content = strings.Replace(content, `<dxfs count="0"/>`, dxfsXML, 1)
		}
		item, err := w.Create(zf.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := item.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func buildCondXLSX(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cond.xlsx")
	f := excelize.NewFile()
	defer f.Close()

	red, err := f.NewStyle(&excelize.Style{
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
	// B2:B4 大于 100 → dxf 0（红）；A2:A4 等于文本 "设备" → dxf 1（绿）；
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
	injectDxfs(t, path)
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
