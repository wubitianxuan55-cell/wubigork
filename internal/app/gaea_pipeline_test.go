package app

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/office/docmd"
	"github.com/xuri/excelize/v2"
)

// TestOfficeFullPipeline 办公全流程端到端：解析 → 修订式编辑 → 接受修订 →
// 提取 Markdown（对话成果）→ 多形态交付。全部纯 Go，无外部依赖。
func TestOfficeFullPipeline(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "pipeline.docx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, docxWithText(t, "合同期限为 30 天。"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	// 1. 框选即改：修订式写入
	pv, err := a.GaeaDocxApplyEdit(filepath.ToSlash(rel), "30 天", "60 天")
	if err != nil {
		t.Fatal(err)
	}
	if pv.Kind != "docx" {
		t.Fatalf("编辑后预览 kind = %q", pv.Kind)
	}
	// 2. 接受修订：新文落地
	if _, err := a.GaeaDocxAcceptChanges(filepath.ToSlash(rel), true); err != nil {
		t.Fatal(err)
	}
	// 3. 提取 Markdown（模拟把成果交给交付出口）
	md, err := docmd.Convert(filepath.ToSlash(rel), "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "60 天") || strings.Contains(md, "30 天") {
		t.Fatalf("提取结果异常: %q", md)
	}
	// 4. 统一交付：Markdown → xlsx / md
	xr, err := a.GaeaExportDeliverable(ExportDeliverableInput{Markdown: md, Format: "xlsx", Title: "合同修订稿"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(xr.Path)
	if err != nil {
		t.Fatalf("交付 xlsx 无效: %v", err)
	}
	found := false
	for _, sheet := range f.GetSheetList() {
		if v, _ := f.GetCellValue(sheet, "A1"); strings.Contains(v, "合同") {
			found = true
		}
	}
	f.Close()
	if !found {
		t.Error("交付 xlsx 缺少标题内容")
	}
}

// TestSmokeRealPipeline 真实文档全链路走查（默认跳过）：
//   真实 docx → 修订式编辑 → 接受修订 → 提取 Markdown → 模板化导出 docx
//   GAEA_SMOKE_PIPELINE=<真实 docx 路径> go test ./internal/app -run TestSmokeRealPipeline -v
func TestSmokeRealPipeline(t *testing.T) {
	src := os.Getenv("GAEA_SMOKE_PIPELINE")
	if src == "" {
		t.Skip("未设置 GAEA_SMOKE_PIPELINE")
	}
	t.Chdir(t.TempDir())
	workdir, _ := os.Getwd()

	rel := filepath.Join(".gaea", "uploads", "real.docx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, b, 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	// 1. 修订式编辑（真实 26MB 文档）
	if _, err := a.GaeaDocxApplyEdit(filepath.ToSlash(rel), "申报类型", "申报类型（全链路走查）"); err != nil {
		t.Fatal(err)
	}
	// 2. 接受修订
	if _, err := a.GaeaDocxAcceptChanges(filepath.ToSlash(rel), true); err != nil {
		t.Fatal(err)
	}
	// 3. 提取 Markdown（对话成果）
	md, err := docmd.Convert(filepath.ToSlash(rel), "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "申报类型（全链路走查）") {
		t.Fatal("提取结果缺少编辑后的文本")
	}
	// 4. 模板化导出 docx（报告模板）
	got, err := a.GaeaExportDeliverable(ExportDeliverableInput{
		Markdown: md, Format: "docx", Title: "真实文档走查", Template: "报告", Cover: true, TOC: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(filepath.Join(workdir, got.Path))
	if err != nil {
		t.Fatalf("交付 docx 无效: %v", err)
	}
	defer zr.Close()
	var docXML []byte
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			docXML, _ = io.ReadAll(rc)
			rc.Close()
		}
	}
	if !strings.Contains(string(docXML), "申报类型（全链路走查）") {
		t.Error("交付 docx 内容缺失")
	}
	t.Logf("全链路走查完成：编辑→接受→提取→导出 %s（%d 字节）", got.Name, got.Size)
}
