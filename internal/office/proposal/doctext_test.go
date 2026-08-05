package proposal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carmel/gooxml/document"
)

func TestExtractPageText_TXT(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("第一行内容\n第二行内容"), 0644); err != nil {
		t.Fatal(err)
	}
	pages, err := ExtractPageText(p)
	if err != nil {
		t.Fatalf("ExtractPageText: %v", err)
	}
	if len(pages) != 1 || pages[0].Page != 0 || pages[0].Text != "第一行内容\n第二行内容" {
		t.Fatalf("TXT pages 异常: %+v", pages)
	}
}

func TestExtractPageText_DOCX(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "b.docx")
	doc := newTestDocx(t, []string{"项目概况段落", "评分标准段落"})
	if err := doc.SaveToFile(p); err != nil {
		t.Fatal(err)
	}
	pages, err := ExtractPageText(p)
	if err != nil {
		t.Fatalf("ExtractPageText: %v", err)
	}
	if len(pages) != 1 || pages[0].Page != 0 {
		t.Fatalf("DOCX pages 异常: %+v", pages)
	}
	if !containsAny(pages[0].Text, "项目概况段落", "评分标准段落") {
		t.Errorf("DOCX 文本缺失: %q", pages[0].Text)
	}
}

func TestExtractPageText_PDF(t *testing.T) {
	pages, err := ExtractPageText("testdata/empty.pdf")
	if err != nil {
		t.Fatalf("ExtractPageText: %v", err)
	}
	if len(pages) != 1 || pages[0].Page != 1 {
		t.Fatalf("PDF pages 异常: %+v", pages)
	}
}

func newTestDocx(t *testing.T, paras []string) *document.Document {
	t.Helper()
	doc := document.New()
	for _, s := range paras {
		doc.AddParagraph().AddRun().AddText(s)
	}
	return doc
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
