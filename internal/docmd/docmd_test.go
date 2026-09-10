package docmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carmel/gooxml/document"
)

// buildDocxWithTable creates a .docx containing a styled table (with w:tcPr /
// shading, exactly what python-docx emits) plus headings and a list.
func buildDocxWithTable(t *testing.T) string {
	t.Helper()
	doc := document.New()
	h1 := doc.AddParagraph()
	h1.SetStyle("Heading1")
	h1.AddRun().AddText("项目周报")
	h2 := doc.AddParagraph()
	h2.SetStyle("Heading2")
	h2.AddRun().AddText("数据汇总")
	table := doc.AddTable()
	table.AddRow()
	table.AddRow()
	table.AddRow()
	cells := [][]string{{"指标", "本周", "上周"}, {"需求完成", "8", "5"}, {"缺陷修复", "3", "2"}}
	for i, row := range table.Rows() {
		for j := 0; j < len(cells[i]); j++ {
			cell := row.AddCell()
			para := cell.AddParagraph()
			para.AddRun().AddText(cells[i][j])
		}
	}
	list := doc.AddParagraph()
	list.AddRun().AddText("完成核心模块开发")

	dir := t.TempDir()
	path := filepath.Join(dir, "table.docx")
	if err := doc.SaveToFile(path); err != nil {
		t.Fatalf("save docx: %v", err)
	}
	return path
}

func TestConvertDocxTableRoundTrip(t *testing.T) {
	path := buildDocxWithTable(t)
	md, err := Convert(path, "")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	for _, want := range []string{
		"# 项目周报",
		"## 数据汇总",
		"| 指标 | 本周 | 上周 |",
		"| 需求完成 | 8 | 5 |",
		"| 缺陷修复 | 3 | 2 |",
		"完成核心模块开发",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}
	// 表格内容必须干净：不允许泄漏 XML 属性块
	for _, junk := range []string{"<w:tcPr>", "<w:shd", "<w:rPr>", "<w:t>", "<w:p>"} {
		if strings.Contains(md, junk) {
			t.Errorf("markdown leaked raw XML %q:\n%s", junk, md)
		}
	}
}

func TestConvertDocxAttrParagraphs(t *testing.T) {
	// 带属性的段落（w14:paraId 等）与内嵌 tabs 不得干扰文本提取
	xml := `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p w14:paraId="12345678"><w:r><w:t>标题段落</w:t></w:r></w:p>
<w:p><w:pPr><w:tabs><w:tab w:val="right" w:pos="9000"/></w:tabs></w:pPr><w:r><w:t>带制表位的内容</w:t></w:r></w:p>
</w:body></w:document>`
	dir := t.TempDir()
	path := filepath.Join(dir, "attr.docx")
	if err := writeRawDocx(path, xml); err != nil {
		t.Fatalf("write raw docx: %v", err)
	}
	md, err := Convert(path, "")
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !strings.Contains(md, "标题段落") {
		t.Errorf("missing attr paragraph text: %q", md)
	}
	if !strings.Contains(md, "带制表位的内容") {
		t.Errorf("missing tabs paragraph text: %q", md)
	}
	if strings.Contains(md, "<w:tabs>") || strings.Contains(md, "<w:tab") {
		t.Errorf("leaked tabs XML: %q", md)
	}
}

// writeRawDocx packs a minimal docx zip with the given document.xml.
func writeRawDocx(path, docXML string) error {
	entries := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/document.xml": docXML,
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return zipWrite(f, entries)
}
