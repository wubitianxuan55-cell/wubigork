package docmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildScanStampPDF 生成「扫描件带页码戳文本层」的合成 PDF：每页 BT/ET 只有
// 页码数字（扫描仪/水印常见形态），不构成正文——根修前第一步结果直接当正文
// 返回，永不触发 OCR；根修后应过不了 meaningfulText 闸落 OCR 兜底。
func buildScanStampPDF(t testing.TB, pages int) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")
	for i := 1; i <= pages; i++ {
		fmt.Fprintf(&sb, "/Type /Page\nBT /F1 12 Tf 72 720 Td (%d) Tj ET\n", i)
	}
	path := filepath.Join(t.TempDir(), fmt.Sprintf("scan-stamp-%d.pdf", pages))
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	return path
}

// TestStampNoiseText 页码戳噪声谓词单元测试：从严判定——数字/符号戳算噪声，
// 任何像正文的行（较长/含字母词/含汉字）都不算。
func TestStampNoiseText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"数字页码戳", "1\n2\n3", true},
		{"符号戳", ".\n.\n-", true},
		{"空/纯空白", "  \n\t", false},
		{"稀疏汉字正文", "甲\n乙\n丙\n丁", false},        // 含 CJK，非噪声
		{"单行英文正文", "Hello Flate World", false}, // 有字母词且行长
		{"短英文词", "hi\nok", false},               // hi 含字母词？hi=2 字母 → 词，非噪声
		{"CJK 页码戳仍不算噪声", "第1页", false},       // 已知漏检面，从严不误伤
	}
	for _, c := range cases {
		if got := stampNoiseText(c.in); got != c.want {
			t.Errorf("stampNoiseText(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestPDFToMarkdownLimit_NoiseTextLayerFallsToOCR 噪声文本层不过闸应落 OCR
// 兜底：堵住 pdftoppm（pdftoppmLookup seam）后报「需要 poppler」，而不是把
// 页码戳当正文返回。
func TestPDFToMarkdownLimit_NoiseTextLayerFallsToOCR(t *testing.T) {
	path := buildScanStampPDF(t, 3)

	orig := pdftoppmLookup
	pdftoppmLookup = func() string { return "" }
	t.Cleanup(func() { pdftoppmLookup = orig })

	md, _, _, err := ConvertLimit(path, "", 5)
	if err == nil {
		t.Fatalf("噪声文本层应落 OCR 兜底，得到 md=%q err=nil", md)
	}
	if !strings.Contains(err.Error(), "pdftoppm") {
		t.Fatalf("应报 pdftoppm 缺失（证明走了 OCR 渲染路径），得到 err=%v", err)
	}
}

// TestPDFToMarkdownLimit_TextPDFSkipsOCR 正例：文本 PDF 照常走文本路径，
// 不触发 OCR 渲染（seam 被调即失败），根修不误伤。
func TestPDFToMarkdownLimit_TextPDFSkipsOCR(t *testing.T) {
	path := buildPagePDF(t, 3)

	orig := pdftoppmLookup
	pdftoppmLookup = func() string {
		t.Error("文本 PDF 不应触发 OCR 渲染路径")
		return ""
	}
	t.Cleanup(func() { pdftoppmLookup = orig })

	md, total, truncated, err := ConvertLimit(path, "", 5)
	if err != nil {
		t.Fatalf("ConvertLimit: %v", err)
	}
	if !strings.Contains(md, "第 1 页内容") || !strings.Contains(md, "第 3 页内容") {
		t.Fatalf("文本内容应注入: %q", md)
	}
	if total != 3 || truncated {
		t.Fatalf("total=%d truncated=%v, want 3/false", total, truncated)
	}
}
