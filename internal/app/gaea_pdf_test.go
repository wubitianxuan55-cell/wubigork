package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGaeaConvertToPdf_Validation(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{core: &core{}}
	if _, err := a.GaeaConvertToPdf(""); err == nil || !strings.Contains(err.Error(), "路径") {
		t.Fatalf("期望路径错误，得到 %v", err)
	}
	if _, err := a.GaeaConvertToPdf("missing.docx"); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("期望不存在错误，得到 %v", err)
	}
	// 不支持的扩展名
	rel := filepath.Join(".gaea", "uploads", "x.png")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GaeaConvertToPdf(filepath.ToSlash(rel)); err == nil || !strings.Contains(err.Error(), "暂不支持") {
		t.Fatalf("期望暂不支持错误，得到 %v", err)
	}
	// .doc 明确提示另存为 docx
	rel2 := filepath.Join(".gaea", "uploads", "y.doc")
	if err := os.WriteFile(rel2, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GaeaConvertToPdf(filepath.ToSlash(rel2)); err == nil || !strings.Contains(err.Error(), ".docx") {
		t.Fatalf("期望 .doc 另存提示，得到 %v", err)
	}
}

func TestGaeaExportDeliverable_PdfFormatAccepted(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{core: &core{}}
	// pdf 格式合法：测试环境没有 create_docx.py/soffice，
	// 预期以依赖缺失错误结束，而不是「不支持的交付格式」
	_, err := a.GaeaExportDeliverable(ExportDeliverableInput{
		Markdown: "# 标题\n正文", Format: "pdf", Title: "测试",
	})
	if err != nil && strings.Contains(err.Error(), "不支持的交付格式") {
		t.Fatalf("pdf 格式应被接受，得到 %v", err)
	}
	// 空内容仍拒绝
	if _, err := a.GaeaExportDeliverable(ExportDeliverableInput{Markdown: "  ", Format: "pdf"}); err == nil || !strings.Contains(err.Error(), "为空") {
		t.Fatalf("期望内容为空错误，得到 %v", err)
	}
}

func TestFindSoffice_NoPanic(t *testing.T) {
	// 有 LibreOffice 返回路径、没有返回空串，都不应 panic
	p := findSoffice()
	if p != "" {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("findSoffice 返回了不存在的路径 %q: %v", p, err)
		}
	}
}

// TestConvertToPdf_Smoke 真实 soffice 转换（默认跳过）：
//   GAEA_SMOKE_PDF=1 go test ./internal/app -run TestConvertToPdf_Smoke -v
func TestConvertToPdf_Smoke(t *testing.T) {
	if os.Getenv("GAEA_SMOKE_PDF") != "1" {
		t.Skip("设置 GAEA_SMOKE_PDF=1 运行真实 PDF 转换 smoke")
	}
	if findSoffice() == "" {
		t.Skip("未找到 soffice")
	}
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "doc.txt")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, []byte("gaea PDF smoke 测试"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{core: &core{}}
	r, err := a.GaeaConvertToPdf(filepath.ToSlash(rel))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(r.Path, ".pdf") || r.Size == 0 {
		t.Fatalf("结果异常: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(gaeaCwd(), r.Path)); err != nil {
		t.Fatalf("PDF 未落盘：%v", err)
	}
}
