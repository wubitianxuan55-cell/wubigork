package docmd

import (
	"strings"
	"testing"
)

// TestExtractPDFTextHexTJ 验证 TJ 数组中的十六进制字符串（UTF-16BE）可被提取。
func TestExtractPDFTextHexTJ(t *testing.T) {
	// "项目周报" 的 UTF-16BE 十六进制（FEFF BOM + 4 个汉字）
	block := "BT /F1 12 Tf 72 720 Td [<FEFF9879 76EE 5468 62A5> 8 <0020>] TJ ET"
	got := extractPDFText(block)
	if !strings.Contains(got, "项目周报") {
		t.Fatalf("hex TJ 提取失败: %q", got)
	}
}

// TestDecodePDFHex 覆盖 BOM、无 BOM、奇数长度与非法输入。
func TestDecodePDFHex(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"FEFF9879 76EE 5468 62A5", "项目周报"},
		{"9879 76EE 5468 62A5", "项目周报"},
		{"48656C6C6F", "Hello"},
		{"41", "A"},
		{"zz", "zz"}, // 非法 hex 原样返回
		{"", ""},
	}
	for _, c := range cases {
		if got := decodePDFHex(c.in); got != c.want {
			t.Errorf("decodePDFHex(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestStripNonTextStreams 只保留含 BT/ET/Tj 的文本流，剔除图像/ICC 等二进制流。
func TestStripNonTextStreams(t *testing.T) {
	textStream := "BT /F1 12 Tf (hello) Tj ET"
	imageStream := "q 700 0 0 150 0 75 cm /fzImg0 Do Q"
	iccStream := "mntrRGB XYZ acsp Artifex Software sRGB ICC Profile"
	in := "<</Length 1>>\nstream\n" + textStream + "\nendstream\nendobj\n" +
		"<</Length 2>>\nstream\n" + imageStream + "\nendstream\nendobj\n" +
		"<</Length 3>>\nstream\n" + iccStream + "\nendstream\nendobj\n"
	got := stripNonTextStreams(in)
	if !strings.Contains(got, textStream) {
		t.Fatalf("文本流应被保留:\n%s", got)
	}
	if strings.Contains(got, imageStream) || strings.Contains(got, iccStream) {
		t.Fatalf("图像/ICC 流应被剔除:\n%s", got)
	}
}

// TestStripNonTextStreamsFakeKeyword 二进制里按字节出现的 "stream" 不应被当成关键字。
func TestStripNonTextStreamsFakeKeyword(t *testing.T) {
	in := "<</Length 10>>\nstream\nBT junk stream inside TJ ET\nendstream\nendobj\n"
	got := stripNonTextStreams(in)
	// 流体含 BT/TJ/ET → 保留；但如果误判关键字，会切断内容。
	if !strings.Contains(got, "junk stream inside") {
		t.Fatalf("流内容被误切: %q", got)
	}
}
