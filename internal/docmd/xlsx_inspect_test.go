package docmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// writeXLSX 用给定的部件拼一个最小工作簿（夹具用；部件内容原样写进 zip）。
func writeXLSX(t *testing.T, parts map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range parts {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "book.xlsx")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const (
	xlsxMainNS  = `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`
	xlsxWorkRel = `http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet`
)

// 直接读工作表 XML：工作表顺序与名字取自 workbook.xml + rels（不是按部件名猜）；内联
// 字符串、富文本共享字符串、没写 r 属性的格子、只有空白的格子、布尔 / 错误格各自算对。
func TestXlsxInspect_ReadsSheetXML(t *testing.T) {
	path := writeXLSX(t, map[string]string{
		"xl/workbook.xml": `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"` +
			` xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>` +
			`<sheet name="报价单" sheetId="2" r:id="rId9"/><sheet name="汇总" sheetId="1" r:id="rId4"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId4" Type="` + xlsxWorkRel + `" Target="worksheets/sheet1.xml"/>` +
			`<Relationship Id="rId9" Type="` + xlsxWorkRel + `" Target="/xl/worksheets/second.xml"/></Relationships>`,
		"xl/sharedStrings.xml": `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
			`<si><t>土方开挖</t></si>` +
			`<si><r><rPr><b/></rPr><t>0402</t></r><r><t>05017001</t></r></si>` +
			`<si><t xml:space="preserve">&#10;  </t></si></sst>`,
		"xl/worksheets/sheet1.xml": xlsxMainNS + `<sheetData>` +
			`<row r="1"><c r="A1" t="inlineStr"><is><t>项目</t></is></c><c r="B1" t="inlineStr"><is><t>金额</t></is></c></row>` +
			`<row r="2"><c r="A2" t="s"><v>0</v></c><c r="B2"><v>5776.41</v></c></row>` +
			`<row r="4"><c t="s"><v>1</v></c><c><v>1</v></c></row>` +
			`<row r="5"><c r="A5" t="s"><v>0</v></c><c r="B5" t="s"><v>2</v></c></row>` +
			`</sheetData><mergeCells count="1"><mergeCell ref="A7:A8"/></mergeCells></worksheet>`,
		"xl/worksheets/second.xml": xlsxMainNS + `<sheetData><row r="1">` +
			`<c r="A1" t="b"><v>1</v></c><c r="B1" t="e"><v>#N/A</v></c><c r="C1"><v></v></c>` +
			`</row></sheetData></worksheet>`,
	})
	rec, err := xlsxInspect(path)
	if err != nil {
		t.Fatalf("xlsxInspect: %v", err)
	}
	// 汇总：行号最大到 5（第 3 行缺席也算在行数里）；7 个非空格子 = 表头 2 + 第 2 行 2 +
	// 第 4 行 2（富文本拼出的定额编码算文本数字）+ 第 5 行 1（空白串不算）。报价单：布尔 /
	// 错误 / 空值都不算，只贡献 1 行。
	want := ExcelReceipt{Sheets: 2, Rows: 6, MaxCols: 2, Merged: 1, Cells: 7, Numbers: 2, NumberText: 1}
	if !reflect.DeepEqual(rec, want) {
		t.Fatalf("回执=%+v，want %+v", rec, want)
	}
}

// 没有 workbook.xml：退回按部件名自然序列出工作表（sheet2 在 sheet10 前面）。
func TestXlsxInspect_FallsBackToPartNames(t *testing.T) {
	cell := func(v string) string {
		return xlsxMainNS + `<sheetData><row r="1"><c r="A1"><v>` + v + `</v></c></row></sheetData></worksheet>`
	}
	path := writeXLSX(t, map[string]string{
		"xl/worksheets/sheet10.xml": cell("3"),
		"xl/worksheets/sheet2.xml":  cell("2"),
		"xl/worksheets/sheet1.xml":  cell("1"),
	})
	sheets := xlsxSheets(mustOpenZip(t, path))
	got := []string{}
	for _, s := range sheets {
		got = append(got, s.name)
	}
	if strings.Join(got, ",") != "sheet1,sheet2,sheet10" {
		t.Fatalf("工作表顺序=%q", got)
	}
	if rec, err := xlsxInspect(path); err != nil || rec.Sheets != 3 || rec.Numbers != 3 || rec.Rows != 3 {
		t.Fatalf("回执=%+v err=%v", rec, err)
	}
}

// 疑似缺格走真实 XML：清单页第 6 行的工程量空着，报「清单!6」。
func TestXlsxInspect_SuspectRowFromXML(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(xlsxMainNS + `<sheetData><row r="1">`)
	for c, h := range []string{"序号", "项目编码", "项目名称", "单位", "工程量"} {
		fmt.Fprintf(&sb, `<c r="%c1" t="inlineStr"><is><t>%s</t></is></c>`, 'A'+c, h)
	}
	sb.WriteString(`</row>`)
	qty := []string{"1000", "1000", "9", "500", "", "100", "100"}
	for i, q := range qty {
		fmt.Fprintf(&sb, `<row r="%d"><c><v>%d</v></c><c t="inlineStr"><is><t>03041200100%d</t></is></c>`+
			`<c t="inlineStr"><is><t>名称</t></is></c><c t="inlineStr"><is><t>m</t></is></c>`, i+2, i+9, i)
		if q != "" {
			fmt.Fprintf(&sb, `<c><v>%s</v></c>`, q)
		}
		sb.WriteString(`</row>`)
	}
	sb.WriteString(`<row r="9"><c t="inlineStr"><is><t>合计</t></is></c></row></sheetData></worksheet>`)
	path := writeXLSX(t, map[string]string{"xl/worksheets/sheet1.xml": sb.String()})
	rec, err := xlsxInspect(path)
	if err != nil {
		t.Fatalf("xlsxInspect: %v", err)
	}
	if rec.SuspectCount != 1 || len(rec.Suspects) != 1 || rec.Suspects[0] != "sheet1!6" || rec.NumberText != 7 {
		t.Fatalf("第 6 行工程量空着应判疑似缺格：%+v", rec)
	}
}

// 损坏 / 空产物如实报错。
func TestXlsxInspect_RejectsBadProducts(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "garbage.xlsx")
	if err := os.WriteFile(bad, []byte("not an xlsx"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := xlsxInspect(bad); err == nil || !strings.Contains(err.Error(), "打不开") {
		t.Fatalf("不是 zip 应报打不开，得到 err=%v", err)
	}
	empty := writeXLSX(t, map[string]string{"docProps/app.xml": "<x/>"})
	if _, err := xlsxInspect(empty); err == nil || !strings.Contains(err.Error(), "没有工作表") {
		t.Fatalf("没有工作表应报错，得到 err=%v", err)
	}
	blank := writeXLSX(t, map[string]string{
		"xl/worksheets/sheet1.xml": xlsxMainNS + `<sheetData><row r="1"><c r="A1" t="s"/></row></sheetData></worksheet>`,
	})
	if _, err := xlsxInspect(blank); err == nil || !strings.Contains(err.Error(), "没有内容") {
		t.Fatalf("没有非空格子应报错，得到 err=%v", err)
	}
}

// 大表性能护栏：839 页清单的工作簿是 2.7 万行、18 万合并区——excelize 逐格调用的版本实测
// 5 分半，流式读必须在几秒内。
func TestXlsxInspect_LargeWorkbookIsFast(t *testing.T) {
	const rows, merges = 27000, 184000
	var sb strings.Builder
	sb.WriteString(xlsxMainNS + `<sheetData>`)
	for i := 1; i <= rows; i++ {
		fmt.Fprintf(&sb, `<row r="%d"><c r="A%d" s="3"><v>%d</v></c><c r="B%d" s="4" t="s"><v>%d</v></c></row>`, i, i, i, i, i%50)
	}
	sb.WriteString(`</sheetData><mergeCells count="` + fmt.Sprint(merges) + `">`)
	for i := 0; i < merges; i++ {
		fmt.Fprintf(&sb, `<mergeCell ref="A%d:B%d"/>`, i%rows+1, i%rows+1)
	}
	sb.WriteString(`</mergeCells></worksheet>`)
	sst := strings.Builder{}
	sst.WriteString(`<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&sst, `<si><t>项目 %d</t></si>`, i)
	}
	sst.WriteString(`</sst>`)
	path := writeXLSX(t, map[string]string{
		"xl/worksheets/sheet1.xml": sb.String(),
		"xl/sharedStrings.xml":     sst.String(),
	})

	start := time.Now()
	rec, err := xlsxInspect(path)
	if err != nil {
		t.Fatalf("xlsxInspect: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("大工作簿核对称应在几秒内，实际 %s", elapsed)
	}
	want := ExcelReceipt{Sheets: 1, Rows: rows, MaxCols: 2, Merged: merges, Cells: rows * 2, Numbers: rows, NumberText: 0}
	if !reflect.DeepEqual(rec, want) {
		t.Fatalf("回执=%+v，want %+v", rec, want)
	}
}

func mustOpenZip(t *testing.T, path string) *zip.Reader {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return &r.Reader
}
