package xlsxedit

// audit_test.go — 审计引擎纯逻辑测试（全 hermetic，不碰 LibreOffice/COM）：
//   - excelize 合成：clean 文件零发现、无缓存公式=missing_cache、自引用=self_ref；
//   - 手写最小 OOXML（zip 直写 XML）：t="e" 错误缓存格=error_value、
//     超大面积 dimension=truncated——excelize 公开 API 写不出这两态，从文件层造。

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// buildWorkbook 用 excelize 合成一个最简单文件（Sheet1，缺省态）。
func buildWorkbook(t *testing.T, dir, name string, fill func(f *excelize.File)) string {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	fill(f)
	p := filepath.Join(dir, name)
	if err := f.SaveAs(p); err != nil {
		t.Fatalf("保存测试工作簿失败：%v", err)
	}
	return p
}

func TestAuditCleanFileHasNoFindings(t *testing.T) {
	dir := t.TempDir()
	p := buildWorkbook(t, dir, "clean.xlsx", func(f *excelize.File) {
		_ = f.SetCellValue("Sheet1", "A1", "单价")
		_ = f.SetCellValue("Sheet1", "B1", "数量")
		_ = f.SetCellValue("Sheet1", "C1", 42.5)
	})
	rep, err := Audit(p)
	if err != nil {
		t.Fatalf("audit 失败：%v", err)
	}
	if len(rep.Findings) != 0 || rep.TotalFormulas != 0 || rep.Truncated {
		t.Fatalf("clean 文件不应有发现或截断：%+v", rep)
	}
	if rep.Sheets != 1 {
		t.Fatalf("应计 1 个工作表：%+v", rep)
	}
}

// 有公式无缓存（excelize SetCellFormula 不写缓存值——openpyxl/excelize
// 生成文件的常态）应报 missing_cache。
func TestAuditMissingCacheFormula(t *testing.T) {
	dir := t.TempDir()
	p := buildWorkbook(t, dir, "nocache.xlsx", func(f *excelize.File) {
		_ = f.SetCellValue("Sheet1", "A1", 10)
		_ = f.SetCellValue("Sheet1", "A2", 20)
		_ = f.SetCellFormula("Sheet1", "A3", "=SUM(A1:A2)")
	})
	rep, err := Audit(p)
	if err != nil {
		t.Fatalf("audit 失败：%v", err)
	}
	if rep.TotalFormulas != 1 {
		t.Fatalf("应计 1 个公式：%+v", rep)
	}
	if len(rep.Findings) != 1 || rep.Findings[0].Kind != "missing_cache" || rep.Findings[0].Cell != "A3" {
		t.Fatalf("应报 A3 missing_cache：%+v", rep.Findings)
	}
}

// 自引用应报 self_ref（$ 与大小写归一）；正常跨格引用不得误报。
func TestAuditSelfRef(t *testing.T) {
	dir := t.TempDir()
	p := buildWorkbook(t, dir, "selfref.xlsx", func(f *excelize.File) {
		_ = f.SetCellFormula("Sheet1", "B2", "=$b$2*2")
		_ = f.SetCellFormula("Sheet1", "C3", "=B2+1")
	})
	rep, err := Audit(p)
	if err != nil {
		t.Fatalf("audit 失败：%v", err)
	}
	refs := 0
	for _, fd := range rep.Findings {
		if fd.Kind == "self_ref" {
			refs++
			if fd.Cell != "B2" {
				t.Fatalf("自引用应在 B2：%+v", fd)
			}
		}
	}
	if refs != 1 {
		t.Fatalf("应恰好 1 条自引用：%+v", rep.Findings)
	}
}

// 手写 OOXML 的 t="e" 缓存错误格应报 error_value 且透传错误串。
func TestAuditErrorCachedCell(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "errcell.xlsx")
	if err := os.WriteFile(p, minimalWorkbook(`<dimension ref="A1:B2"/>
<sheetData>
<row r="1"><c r="A1"><v>1</v></c></row>
<row r="2"><c r="B2" t="e"><f>1/0</f><v>#DIV/0!</v></c></row>
</sheetData>`), 0o644); err != nil {
		t.Fatalf("写夹具失败：%v", err)
	}
	rep, err := Audit(p)
	if err != nil {
		t.Fatalf("audit 失败：%v", err)
	}
	errs := 0
	for _, fd := range rep.Findings {
		if fd.Kind == "error_value" {
			errs++
			if fd.Value != "#DIV/0!" || fd.Formula != "1/0" {
				t.Fatalf("错误值与公式应透传：%+v", fd)
			}
		}
	}
	if errs != 1 || rep.TotalFormulas != 1 {
		t.Fatalf("应恰好 1 条错误值 1 个公式：%+v", rep)
	}
}

// 声明面积超限的表应整体截断且不拖死回合（真表里 dimension 通常是准的）。
func TestAuditTruncatesHugeDimension(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "huge.xlsx")
	if err := os.WriteFile(p, minimalWorkbook(`<dimension ref="A1:ZZ3000"/>
<sheetData><row r="1"><c r="A1"><v>1</v></c></row></sheetData>`), 0o644); err != nil {
		t.Fatalf("写夹具失败：%v", err)
	}
	rep, err := Audit(p)
	if err != nil {
		t.Fatalf("audit 失败：%v", err)
	}
	if !rep.Truncated {
		t.Fatalf("面积 %d > 上限 %d 应截断：%+v", 702*3000, AuditMaxCells, rep)
	}
	if rep.TotalFormulas != 0 {
		t.Fatalf("截断路径不应产出公式计数：%+v", rep)
	}
}

// dimension 陈旧（excelize 写出恒 "A1"）时以实际内容范围兜底——A3 公式必须被扫到。
func TestAuditStaleDimensionStillScansContent(t *testing.T) {
	dir := t.TempDir()
	p := buildWorkbook(t, dir, "staledim.xlsx", func(f *excelize.File) {
		_ = f.SetCellValue("Sheet1", "A1", 10)
		_ = f.SetCellValue("Sheet1", "A2", 20)
		_ = f.SetCellFormula("Sheet1", "C99", "=A1+A2")
	})
	rep, err := Audit(p)
	if err != nil {
		t.Fatalf("audit 失败：%v", err)
	}
	if rep.TotalFormulas != 1 || len(rep.Findings) != 1 || rep.Findings[0].Cell != "C99" {
		t.Fatalf("陈旧 dimension 不应漏扫 C99：%+v", rep)
	}
}

// ── 最小 OOXML 夹具（zip 直写，excelize 可打开）──────────────

// minimalWorkbook 用给定 sheetData 片段造一个最小可打开的 .xlsx。
// 注意：<c> 必须包在 <row r="N"> 里（裸单元格会让 Rows 迭代器读到 0 行）。
func minimalWorkbook(sheetData string) []byte {
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`,
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets>
<calcPr calcId="0" fullCalcOnLoad="1"/>
</workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`,
		"xl/worksheets/sheet1.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` + sheetData + `</worksheet>`,
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			panic(err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			panic(err)
		}
	}
	_ = w.Close()
	return buf.Bytes()
}
