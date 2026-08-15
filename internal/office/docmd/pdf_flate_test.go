package docmd

// pdf_flate_test.go — T7-3：FlateDecode 压缩文本流还原后再做 BT/ET 文本提取。
// 数字型 PDF（正文压缩在 /Filter /FlateDecode 流里）不再因提取失败被误判为扫描件。

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildFlatePDF 构造一个带 FlateDecode 压缩文本流的 PDF fixture：
// 单页 + 一个 /Filter /FlateDecode 的内容流，流内是 BT/ET 文本块。
func buildFlatePDF(t *testing.T, pageText string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write([]byte(pageText)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%%PDF-1.4\n"+
		"1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n"+
		"2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n"+
		"3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >> endobj\n"+
		"4 0 obj << /Length %d /Filter /FlateDecode >>\nstream\n%s\nendstream\nendobj\n"+
		"trailer << /Root 1 0 R >>\n%%EOF\n", buf.Len(), buf.String())
}

// writeTempPDF 把 PDF 内容写入临时文件并返回路径。
func writeTempPDF(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "flate.pdf")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestPDFFlateDecodeTextExtraction FlateDecode 压缩文本流可被还原并提取出正文：
// 数字型 PDF 不再因提取失败误判为扫描件（提取成功即不会走 OCR 回退）。
func TestPDFFlateDecodeTextExtraction(t *testing.T) {
	pdf := buildFlatePDF(t, "BT /F1 12 Tf 72 712 Td (Hello Flate World) Tj ET")
	p := writeTempPDF(t, pdf)

	md, err := pdfToMarkdown(p, "")
	if err != nil {
		t.Fatalf("pdfToMarkdown: %v", err)
	}
	if !strings.Contains(md, "Hello Flate World") {
		t.Fatalf("FlateDecode 文本未提取出来: %q", md)
	}
	// 不能是「扫描件」判定：结果必须来自文本提取（含可读正文而非 OCR 提示）。
	if strings.Contains(md, "OCR") {
		t.Fatalf("数字型 PDF 不应走 OCR 回退: %q", md)
	}
}

// TestPDFFlateDecodeChineseText FlateDecode 流内 CJK 文本（UTF-16BE hex TJ）同样还原。
func TestPDFFlateDecodeChineseText(t *testing.T) {
	// "项目周报" 的 UTF-16BE 十六进制（FEFF BOM）
	pdf := buildFlatePDF(t, "BT /F1 12 Tf 72 720 Td [<FEFF9879 76EE 5468 62A5> 8 <0020>] TJ ET")
	p := writeTempPDF(t, pdf)

	md, err := pdfToMarkdown(p, "")
	if err != nil {
		t.Fatalf("pdfToMarkdown: %v", err)
	}
	if !strings.Contains(md, "项目周报") {
		t.Fatalf("FlateDecode CJK 文本未提取出来: %q", md)
	}
}

// TestDecodeFlateStreamsLeavesUncompressedUntouched 非 FlateDecode 流不被解压/改写。
func TestDecodeFlateStreamsLeavesUncompressedUntouched(t *testing.T) {
	in := "1 0 obj << /Length 10 >>\nstream\nBT (plain) Tj ET\nendstream\nendobj\n"
	got := decodeFlateStreams(in)
	if got != in {
		t.Fatalf("非 FlateDecode 流被改写:\n原始: %q\n结果: %q", in, got)
	}
}

// TestDecodeFlateStreamsImageStreamNoLeak FlateDecode 图像流（解压后无 BT/ET）
// 还原后不得被当成文本（后续 stripNonTextStreams 会按 BT/ET 剔除）。
func TestDecodeFlateStreamsImageStreamNoLeak(t *testing.T) {
	// 解压后是二进制垃圾（无 BT/ET/Tj）
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write([]byte("q 700 0 0 150 0 75 cm /fzImg0 Do Q")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	in := "5 0 obj << /Length " + fmt.Sprint(buf.Len()) + " /Filter /FlateDecode /Subtype /Image >>\nstream\n" +
		buf.String() + "\nendstream\nendobj\n"

	decoded := decodeFlateStreams(in)
	// 解压成功 → 流体被替换为解压内容
	if !strings.Contains(decoded, "fzImg0") {
		t.Fatalf("图像流应被解压还原（后续按 BT/ET 判定去留）: %q", decoded)
	}
	// 但 stripNonTextStreams 后必须被剔除（不含 BT/ET）
	stripped := stripNonTextStreams(decoded)
	if strings.Contains(stripped, "fzImg0") {
		t.Fatalf("图像流解压后不应残留: %q", stripped)
	}
}

// TestDecodeFlateStreamsBrokenStreamKept 解压失败（损坏/非 zlib 数据）保持原样。
func TestDecodeFlateStreamsBrokenStreamKept(t *testing.T) {
	in := "6 0 obj << /Length 8 /Filter /FlateDecode >>\nstream\nnotzlib!!\nendstream\nendobj\n"
	got := decodeFlateStreams(in)
	if got != in {
		t.Fatalf("解压失败流不应被改写:\n原始: %q\n结果: %q", in, got)
	}
}
