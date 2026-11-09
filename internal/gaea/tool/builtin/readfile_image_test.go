package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadFile_ImageGuidesToOCRVision read_file 读图片应给出 ocr/vision 分工
// 引导（2026-11-09 办公识图根修），而不是通用 binary/markitdown 提示。
func TestReadFile_ImageGuidesToOCRVision(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "scan.png")
	// 带 NUL 的假 PNG：走二进制探测路径。
	if err := os.WriteFile(imgPath, []byte("\x89PNG\r\n\x1a\n\x00\x00"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := readFile{workDir: dir}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"scan.png"}`))
	if err == nil {
		t.Fatal("read_file on image should error with guidance")
	}
	msg := err.Error()
	if !strings.Contains(msg, `"ocr"`) || !strings.Contains(msg, `"vision"`) {
		t.Errorf("image error missing ocr/vision guidance: %q", msg)
	}
	if strings.Contains(msg, "markitdown") {
		t.Errorf("image error should not suggest markitdown: %q", msg)
	}
}

// TestReadFile_ScannedPDFGuidesToFormatConvert read_file 对转不出文本的 PDF
// 应指给 format_convert（扫描件自动 OCR）/summarize_file，不再误导装 markitdown
// （2026-11-09 扫描件 PDF fallback 刀③）。
func TestReadFile_ScannedPDFGuidesToFormatConvert(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "scan.pdf")
	// %PDF 头 + NUL：判二进制；markitdown 路径用 seam 堵死（真 markitdown
	// 装没装因机而异，不 hermetic）。
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\n\x00\x00trailer"), 0o644); err != nil {
		t.Fatal(err)
	}
	origConv := tryMarkItDownFn
	tryMarkItDownFn = func(context.Context, string) (string, bool) { return "", false }
	t.Cleanup(func() { tryMarkItDownFn = origConv })

	tool := readFile{workDir: dir}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"scan.pdf"}`))
	if err == nil {
		t.Fatal("read_file on unconvertible PDF should error with guidance")
	}
	msg := err.Error()
	if !strings.Contains(msg, "format_convert") || !strings.Contains(msg, "scanned PDF") {
		t.Errorf("PDF error missing format_convert/scanned guidance: %q", msg)
	}
	if strings.Contains(msg, "install markitdown") {
		t.Errorf("PDF error should not suggest installing markitdown: %q", msg)
	}
}

// TestReadFile_BinaryKeepsMarkItDownHint 非图片二进制仍走原 markitdown 提示
// （图片分支不得误伤通用二进制路径）。
func TestReadFile_BinaryKeepsMarkItDownHint(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "blob.bin")
	if err := os.WriteFile(binPath, []byte{'a', 0x00, 'b'}, 0o644); err != nil {
		t.Fatal(err)
	}

	tool := readFile{workDir: dir}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"blob.bin"}`))
	if err == nil {
		t.Fatal("read_file on binary should error")
	}
	if !strings.Contains(err.Error(), "markitdown") {
		t.Errorf("generic binary error lost markitdown hint: %q", err.Error())
	}
}
