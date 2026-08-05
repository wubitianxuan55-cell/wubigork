package proposal

import (
	"os"
	"testing"

	"github.com/carmel/gooxml/document"
)

func TestRenderDocx_CoverTOCAndTable(t *testing.T) {
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("某修复投标方案", "soil-remediation-bid", "需求", "环保工程", proj.ID, []ProposalSection{
		{Title: "第一章 项目概况", Level: 1, Children: []ProposalSection{
			{Title: "1.1 项目概况", Level: 2, Content: "项目位于某区。\n\n| 项目 | 内容 |\n|---|---|\n| 面积 | 10000 m² |"},
		}},
		{Title: "第二章 技术路线", Level: 1, Content: "采用固化稳定化工艺。"},
	})
	path, err := renderDocxToFile(p, ExportOptions{IncludeCover: true, IncludeTOC: true}, s.store.ExportDir())
	if err != nil {
		t.Fatalf("renderDocxToFile: %v", err)
	}
	doc, err := document.Open(path)
	if err != nil {
		t.Fatalf("打开 docx 失败: %v", err)
	}
	if len(doc.Tables()) != 1 {
		t.Fatalf("表格数 = %d, want 1", len(doc.Tables()))
	}
	text := ""
	for _, para := range doc.Paragraphs() {
		for _, run := range para.Runs() {
			text += run.Text()
		}
	}
	for _, want := range []string{"某修复投标方案", "第一章", "第二章", "1.1"} {
		if !containsAny(text, want) {
			t.Errorf("文档缺少 %q", want)
		}
	}
	_ = os.Remove(path)
}

func TestExportSectionDocx_OnlySubtree(t *testing.T) {
	s := newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
	proj, _ := s.store.EnsureDefaultProject()
	p, _ := s.store.Create("方案", "blank", "", "其他", proj.ID, []ProposalSection{
		{Title: "第一章", Level: 1, Children: []ProposalSection{
			{ID: "secA", Title: "1.1 背景", Level: 2, Content: "背景内容"},
		}},
		{Title: "第二章", Level: 1, Content: "第二章内容"},
	})
	path, err := s.ExportSectionDocx(p.ID, "secA", ExportOptions{})
	if err != nil {
		t.Fatalf("ExportSectionDocx: %v", err)
	}
	doc, err := document.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	text := ""
	for _, para := range doc.Paragraphs() {
		for _, run := range para.Runs() {
			text += run.Text()
		}
	}
	if !containsAny(text, "1.1 背景", "背景内容") {
		t.Fatalf("单章导出内容异常: %s", text)
	}
	if containsAny(text, "第二章内容") {
		t.Fatal("单章导出混入其他章节")
	}
	_ = os.Remove(path)
}
