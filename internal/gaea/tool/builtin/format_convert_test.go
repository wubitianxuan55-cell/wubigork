package builtin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
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
