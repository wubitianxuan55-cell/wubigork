package app

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestGaeaPreview_Mermaid 验证 .mmd 图表文件可按 markdown 预览（渲染成图）。
func TestGaeaPreview_Mermaid(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "diagram-test.mmd")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	code := "flowchart LR\nA-->B"
	if err := os.WriteFile(rel, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	got := a.GaeaPreview(filepath.ToSlash(rel))
	if got.Kind != "markdown" {
		t.Fatalf("kind = %q, want markdown", got.Kind)
	}
	if !strings.HasPrefix(got.Body, "```mermaid\n") || !strings.Contains(got.Body, "flowchart LR") {
		t.Errorf("body = %q, want mermaid 围栏包裹", got.Body)
	}
}

// TestGaeaPreview_Missing 不存在的文件返回 error。
func TestGaeaPreview_Missing(t *testing.T) {
	t.Chdir(t.TempDir())
	a := &App{}
	got := a.GaeaPreview("nope.mmd")
	if got.Kind != "error" {
		t.Fatalf("kind = %q, want error", got.Kind)
	}
}

// TestGaeaPreview_BareFilenameFallback 验证裸文件名（无目录分隔符）在常见输出
// 目录（exports 等）中可被解析，使“输出文件：成本测算.xlsx”这类引用可直接预览。
func TestGaeaPreview_BareFilenameFallback(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join("exports", "成本测算.xlsx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "预算")
	f.SetCellValue("预算", "A1", "项目")
	if err := f.SaveAs(rel); err != nil {
		t.Fatal(err)
	}
	f.Close()

	a := &App{}
	got := a.GaeaPreview("成本测算.xlsx")
	if got.Kind != "xlsx" {
		t.Fatalf("kind = %q, want xlsx（裸文件名应解析到 exports 目录）", got.Kind)
	}
	if want := filepath.ToSlash(rel); got.Path != want {
		t.Fatalf("Path = %q, want %q", got.Path, want)
	}
}

// TestGaeaPreview_Docx 验证 .docx 返回原始字节 dataUrl（前端 docx-preview 保真渲染）
// 且 body 附带轻量 Markdown 文本（交付卡片缩略图用）。
func TestGaeaPreview_Docx(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "preview-test.docx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, minimalDocx(t), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	got := a.GaeaPreview(filepath.ToSlash(rel))
	if got.Kind != "docx" {
		t.Fatalf("kind = %q, want docx", got.Kind)
	}
	wantPrefix := "data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64,"
	if !strings.HasPrefix(got.DataURL, wantPrefix) {
		t.Errorf("dataUrl = %.60s..., want prefix %q", got.DataURL, wantPrefix)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(got.DataURL, wantPrefix))
	if err != nil {
		t.Fatalf("dataUrl base64 解码失败: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("dataUrl 不是合法 zip/docx: %v", err)
	}
	var documentXML []byte
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			documentXML, _ = io.ReadAll(rc)
			rc.Close()
		}
	}
	if !bytes.Contains(documentXML, []byte("preview-test")) {
		t.Errorf("docx 内容缺失 preview-test 标记")
	}
	if !strings.Contains(got.Body, "preview-test") {
		t.Errorf("body = %q, want 包含 preview-test 文本（缩略图用）", got.Body)
	}
}

// TestPreviewThumbText 验证缩略图正文截断：超过上限时按 UTF-8 字符边界安全截断。
func TestPreviewThumbText(t *testing.T) {
	short := "前几行正文"
	if got := previewThumbText(short); got != short {
		t.Fatalf("短文本不应截断: got %q", got)
	}

	// 4KB 上限 + 中文字符（3 字节/字），截断点可能在字符中间 → 必须回退到完整字符。
	long := strings.Repeat("中", maxThumbBytes+10)
	got := previewThumbText(long)
	if len(got) > maxThumbBytes {
		t.Fatalf("截断后长度 %d > 上限 %d", len(got), maxThumbBytes)
	}
	if strings.Contains(got, "\xef\xbf") { // UTF-8 半字残留（EF BF BD 是替换符）
		t.Errorf("截断处出现半字残留: %q", got[len(got)-6:])
	}
	if got[len(got)-1] != []byte("中")[0] {
		t.Errorf("截断应停在完整字符边界, 尾部字节 = %x", got[len(got)-1])
	}
}

// minimalDocx 构造一个最小合法 docx（zip 包），内容包含 preview-test 标记。
func minimalDocx(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
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
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>preview-test 保真预览</w:t></w:r></w:p>
  </w:body>
</w:document>`,
	}
	for name, body := range files {
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
	return buf.Bytes()
}

// TestGaeaPreview_Xlsx 验证 .xlsx 返回结构化单元格 JSON（kind=xlsx）。
func TestGaeaPreview_Xlsx(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "preview-test.xlsx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "预算")
	f.SetCellValue("预算", "A1", "项目")
	f.SetCellValue("预算", "B1", "金额")
	f.SetCellValue("预算", "A2", "设备")
	f.SetCellValue("预算", "B2", 120.5)
	f.SetCellFormula("预算", "B3", "SUM(B2)")
	if err := f.SaveAs(rel); err != nil {
		t.Fatal(err)
	}
	f.Close()

	a := &App{}
	got := a.GaeaPreview(filepath.ToSlash(rel))
	if got.Kind != "xlsx" {
		t.Fatalf("kind = %q, want xlsx", got.Kind)
	}
	if !strings.Contains(got.Body, `"name":"预算"`) || !strings.Contains(got.Body, `"formula":"SUM(B2)"`) {
		t.Errorf("body 缺少工作表/公式信息: %.200s", got.Body)
	}
}
