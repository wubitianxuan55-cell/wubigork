package app

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/xuri/excelize/v2"
)

const exportMarkdown = `# 项目周报

## 本周完成

- 完成土壤修复方案初稿
- 通过专家评审

## 关键数据

| 指标 | 数值 |
| --- | --- |
| 修复面积 | 120 亩 |
| 投资估算 | 8000 万元 |
`

func TestGaeaExportDeliverable_MD(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{}
	got, err := a.GaeaExportDeliverable(ExportDeliverableInput{Markdown: exportMarkdown, Format: "md"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Format != "md" || !strings.HasSuffix(got.Path, ".md") {
		t.Fatalf("结果异常: %+v", got)
	}
	b, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "项目周报") {
		t.Error("md 内容缺失")
	}
}

func TestGaeaExportDeliverable_XLSX(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{}
	got, err := a.GaeaExportDeliverable(ExportDeliverableInput{Markdown: exportMarkdown, Format: "xlsx"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(got.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got := f.GetSheetList(); len(got) == 0 {
		t.Fatal("xlsx 没有工作表")
	}
	// 表格应写入数据工作表
	found := false
	for _, sheet := range f.GetSheetList() {
		if v, _ := f.GetCellValue(sheet, "A2"); v == "修复面积" {
			found = true
			if b, _ := f.GetCellValue(sheet, "B2"); b != "120 亩" {
				t.Errorf("B2 = %q", b)
			}
		}
	}
	if !found {
		t.Error("表格数据未写入 xlsx")
	}
}

func TestGaeaExportDeliverable_Validation(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{}
	if _, err := a.GaeaExportDeliverable(ExportDeliverableInput{Markdown: "", Format: "docx"}); err == nil {
		t.Fatal("空内容应报错")
	}
	if _, err := a.GaeaExportDeliverable(ExportDeliverableInput{Markdown: "x", Format: "zip"}); err == nil || !strings.Contains(err.Error(), "格式") {
		t.Fatalf("非法格式应报错，得到 %v", err)
	}
	if _, err := a.GaeaExportDeliverable(ExportDeliverableInput{Markdown: "x", Format: "docx", Template: "海报"}); err == nil || !strings.Contains(err.Error(), "模板") {
		t.Fatalf("非法模板应报错，得到 %v", err)
	}
}

func TestGaeaExportDeliverable_SanitizeTitle(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{}
	got, err := a.GaeaExportDeliverable(ExportDeliverableInput{
		Markdown: "# 内容", Format: "md", Title: `方案<版本1>: "终稿" 2026/08?`,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`<`, `>`, `:`, `"`, `/`, `?`} {
		if strings.Contains(got.Name, bad) {
			t.Errorf("文件名含非法字符 %q: %s", bad, got.Name)
		}
	}
	if !strings.Contains(got.Name, "方案") {
		t.Errorf("文件名应保留中文标题: %s", got.Name)
	}
}

// TestGaeaExportDeliverable_DocxPptxSmoke 真实调用 python 技能脚本（默认跳过）：
//   GAEA_SMOKE_EXPORT=1 go test ./internal/app -run TestGaeaExportDeliverable_DocxPptxSmoke -v
func TestGaeaExportDeliverable_DocxPptxSmoke(t *testing.T) {
	if os.Getenv("GAEA_SMOKE_EXPORT") == "" {
		t.Skip("未设置 GAEA_SMOKE_EXPORT")
	}
	orig, _ := os.Getwd()
	root := orig
	for {
		if _, err := os.Stat(filepath.Join(root, ".gaea", "skills", "docx", "scripts", "create_docx.py")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("未找到仓库 .gaea/skills")
		}
		root = parent
	}
	ga.cfg = &gaeaConfig.Config{Workspace: root}
	t.Chdir(t.TempDir())
	a := &App{}
	for _, format := range []string{"docx", "pptx"} {
		got, err := a.GaeaExportDeliverable(ExportDeliverableInput{
			Markdown: exportMarkdown, Format: format, Title: "项目周报", Template: "报告", Cover: true, TOC: true,
		})
		if err != nil {
			t.Fatalf("%s 导出失败: %v", format, err)
		}
		zr, err := zip.OpenReader(filepath.Join(root, got.Path))
		if err != nil {
			t.Fatalf("%s 不是合法 zip: %v", format, err)
		}
		zr.Close()
	}
}
