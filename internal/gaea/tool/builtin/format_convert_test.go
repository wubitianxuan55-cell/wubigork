package builtin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestFormatConvertWritesOutputFile verifies that format_convert generates the
// output file even when its parent directory does not exist yet.
func TestFormatConvertWritesOutputFile(t *testing.T) {
	docx := writeMinimalDocx(t)
	out := filepath.Join(t.TempDir(), "nested", "deep", "converted.md")

	args, err := json.Marshal(map[string]string{
		"path":   docx,
		"output": out,
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	res, err := (formatConvert{}).Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(res, "已转换并保存为") {
		t.Errorf("unexpected result: %s", res)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output file not generated: %v", err)
	}
	if !strings.HasPrefix(string(data), "# 文档转换:") {
		t.Errorf("unexpected output content: %.120s", data)
	}
}

func TestFormatConvertPreservesDocxImages(t *testing.T) {
	const png = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Wl9sAAAAASUVORK5CYII="
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "figure.png")
	raw, err := base64.StdEncoding.DecodeString(png)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imagePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	docx := filepath.Join(dir, "with-image.docx")
	script := `
from docx import Document
from docx.shared import Inches
d = Document()
d.add_heading("第一章 工艺", level=1)
d.add_paragraph("工艺流程图如下：")
d.add_picture(r"` + imagePath + `", width=Inches(0.25))
d.save(r"` + docx + `")
`
	if out, err := exec.Command("python", "-c", script).CombinedOutput(); err != nil {
		t.Skipf("python-docx 不可用，跳过: %v %s", err, out)
	}
	outPath := filepath.Join(dir, "converted", "方案.md")
	args, err := json.Marshal(map[string]any{
		"path": docx, "output": outPath, "preserve_images": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := (formatConvert{}).Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(res, "保留图片 1 张") {
		t.Fatalf("回执未说明图片: %s", res)
	}
	md, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(md), "data:image/") || !strings.Contains(string(md), "方案-assets/") {
		t.Fatalf("Markdown 图片未外置: %.800s", md)
	}
	entries, err := os.ReadDir(filepath.Join(filepath.Dir(outPath), "方案-assets"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("sidecar 图片: %v %+v", err, entries)
	}
}

func TestFormatConvertPreserveImagesWritePaths(t *testing.T) {
	dir := t.TempDir()
	args, err := json.Marshal(map[string]any{
		"path": "source.docx", "output": "exports/方案.md", "preserve_images": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	paths, err := (formatConvert{workDir: dir}).WritePaths(args)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[1] != filepath.Join(dir, "exports", "方案-assets") {
		t.Fatalf("write paths=%+v", paths)
	}
}

// writeMinimalDocx builds a tiny but structurally valid docx (word/document.xml
// only) so the built-in docx parser has something to read; markitdown being
// present or absent doesn't affect the assertions above.
func writeMinimalDocx(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sample.docx")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>项目周报</w:t></w:r></w:p>
  </w:body>
</w:document>`
	if _, err := w.Write([]byte(docXML)); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write docx: %v", err)
	}
	return path
}
