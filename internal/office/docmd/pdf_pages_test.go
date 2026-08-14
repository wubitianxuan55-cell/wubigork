package docmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildStructuredPDF 生成带对象结构的合成 PDF（未压缩文本流，体积可控）：
// 每个页对象 /Type /Page 后紧跟其内容流对象（BT..ET 文本）。pagesTree 为 true 时
// 先写一个 /Type /Pages 页树根节点（真实 PDF 的页树声明，旧实现会误计一页）。
func buildStructuredPDF(t testing.TB, pageTexts []string, pagesTree bool) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")
	obj := 1
	if pagesTree {
		kids := make([]string, len(pageTexts))
		for i := range kids {
			kids[i] = fmt.Sprintf("%d 0 R", obj+1+i)
		}
		fmt.Fprintf(&sb, "%d 0 obj\n<< /Type /Pages /Kids [%s] /Count %d >>\nendobj\n",
			obj, strings.Join(kids, " "), len(pageTexts))
		obj++
	}
	for _, text := range pageTexts {
		if text != "" {
			fmt.Fprintf(&sb, "%d 0 obj\n<< /Type /Page /Parent 1 0 R /MediaBox [0 0 612 792] /Contents %d 0 R >>\nendobj\n", obj, obj+1)
			obj++
			fmt.Fprintf(&sb, "%d 0 obj\n<< /Length %d >>\nstream\nBT /F1 12 Tf 72 720 Td (%s) Tj ET\nendstream\nendobj\n", obj, len(text)+24, text)
			obj++
		} else {
			fmt.Fprintf(&sb, "%d 0 obj\n<< /Type /Page /Parent 1 0 R /MediaBox [0 0 612 792] >>\nendobj\n", obj)
			obj++
		}
	}
	sb.WriteString("trailer\n<< /Root 1 0 R >>\n%%EOF\n")
	path := filepath.Join(t.TempDir(), "structured.pdf")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	return path
}

// TestCountPDFPagesExcludesPagesTree: 3 页 PDF 含 /Type /Pages 干扰对象 → 页数 = 3。
func TestCountPDFPagesExcludesPagesTree(t *testing.T) {
	path := buildStructuredPDF(t, []string{"第一页文本", "第二页文本", "第三页文本"}, true)
	md, total, truncated, err := ConvertLimit(path, "", 0)
	if err != nil {
		t.Fatalf("ConvertLimit: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3（/Type /Pages 不应计入页数）", total)
	}
	if truncated {
		t.Errorf("truncated = true, want false")
	}
	for _, want := range []string{"第一页文本", "第二页文本", "第三页文本"} {
		if !strings.Contains(md, want) {
			t.Errorf("md missing %q:\n%s", want, md)
		}
	}
	// 直接验证计数函数（对页树干扰对象字节）
	content, _ := os.ReadFile(path)
	if n := countPDFPages(string(content)); n != 3 {
		t.Errorf("countPDFPages = %d, want 3", n)
	}
}

// TestPDFPageRangeFilter: pages=2-3 只返回第 2-3 页文本。
func TestPDFPageRangeFilter(t *testing.T) {
	path := buildStructuredPDF(t, []string{"第一页文本", "第二页文本", "第三页文本"}, false)
	md, total, _, err := ConvertLimit(path, "2-3", 0)
	if err != nil {
		t.Fatalf("ConvertLimit(2-3): %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if !strings.Contains(md, "第二页文本") || !strings.Contains(md, "第三页文本") {
		t.Errorf("range 2-3 缺页:\n%s", md)
	}
	if strings.Contains(md, "第一页文本") {
		t.Errorf("range 2-3 泄漏第 1 页:\n%s", md)
	}
}

// TestPDFSinglePage: 单页 PDF 页数与文本正确。
func TestPDFSinglePage(t *testing.T) {
	path := buildStructuredPDF(t, []string{"唯一一页"}, false)
	md, total, _, err := ConvertLimit(path, "", 0)
	if err != nil {
		t.Fatalf("ConvertLimit: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if !strings.Contains(md, "唯一一页") {
		t.Errorf("missing text: %q", md)
	}
}

// TestPDFPageCountNoInterference: 无 /Type /Pages 干扰对象时页数正确。
func TestPDFPageCountNoInterference(t *testing.T) {
	path := buildStructuredPDF(t, []string{"甲", "乙", "丙", "丁"}, false)
	_, total, _, err := ConvertLimit(path, "", 0)
	if err != nil {
		t.Fatalf("ConvertLimit: %v", err)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
}

// TestPDFPageTypeBoundary: /Type /Page 与 /Type /Pages 混合的边界：
// 双层页树（根 + 中间 /Type /Pages 节点）+ /Type/Page 无空格写法 + /Type /PageSuffix
// 更长 name。页对象数 = 2，/Type /Pages 与 /PageSuffix 都不计入。
func TestPDFPageTypeBoundary(t *testing.T) {
	content := "%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Pages /Kids [2 0 R] /Count 2 >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 >>\nendobj\n" +
		"3 0 obj\n<< /Type/Page /Parent 2 0 R /Contents 5 0 R >>\nendobj\n" +
		"4 0 obj\n<< /Type/Page /Parent 2 0 R /Contents 6 0 R >>\nendobj\n" +
		"5 0 obj\n<< /Length 20 >>\nstream\nBT (边界页一) Tj ET\nendstream\nendobj\n" +
		"6 0 obj\n<< /Length 20 >>\nstream\nBT (边界页二) Tj ET\nendstream\nendobj\n" +
		"7 0 obj\n<< /Type /PageSuffix /Fake true >>\nendobj\n" +
		"trailer\n<< /Root 1 0 R >>\n%%EOF\n"
	if n := countPDFPages(content); n != 2 {
		t.Errorf("countPDFPages = %d, want 2（/Type /Pages、/Type /PageSuffix 不计入，/Type/Page 计入）", n)
	}
	path := filepath.Join(t.TempDir(), "boundary.pdf")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	md, total, _, err := ConvertLimit(path, "", 0)
	if err != nil {
		t.Fatalf("ConvertLimit: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if !strings.Contains(md, "边界页一") || !strings.Contains(md, "边界页二") {
		t.Errorf("missing boundary page text:\n%s", md)
	}
}

// TestPDFMultiBTBlocksPerPage: 页内多个 BT..ET 块归同一页，页码由页对象决定，
// 不再由 BT 块自增（第 1 页有两个 BT 块时 pages=2 不得错位到第 2 个块）。
func TestPDFMultiBTBlocksPerPage(t *testing.T) {
	content := "%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Page /Contents 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Length 60 >>\nstream\nBT (块一) Tj ET\nBT (块二) Tj ET\nendstream\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Contents 4 0 R >>\nendobj\n" +
		"4 0 obj\n<< /Length 30 >>\nstream\nBT (第二页) Tj ET\nendstream\nendobj\n" +
		"trailer\n<< /Root 1 0 R >>\n%%EOF\n"
	path := filepath.Join(t.TempDir(), "multibt.pdf")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	md, total, _, err := ConvertLimit(path, "2", 0)
	if err != nil {
		t.Fatalf("ConvertLimit(2): %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if !strings.Contains(md, "第二页") {
		t.Errorf("page 2 missing:\n%s", md)
	}
	if strings.Contains(md, "块一") || strings.Contains(md, "块二") {
		t.Errorf("page 1 leaked into page 2（页码仍由 BT 块自增）:\n%s", md)
	}
	all, _, _, err := ConvertLimit(path, "", 0)
	if err != nil {
		t.Fatalf("ConvertLimit(all): %v", err)
	}
	if !strings.Contains(all, "块一") || !strings.Contains(all, "块二") {
		t.Errorf("full output missing page-1 blocks:\n%s", all)
	}
}

// TestPDFOCRTruncatedRange: OCR 回退的渲染范围必须与"已截断"提示一致。
// capPageSpec 收敛出 effSpec 与 truncated，pageBounds 用同一规格给出 OCR 渲染范围；
// 截断时渲染范围内的每一页都必须落在收敛后的规格里（OCR 循环用 pageInRange 过滤）。
func TestPDFOCRTruncatedRange(t *testing.T) {
	cases := []struct {
		name      string
		spec      string
		maxPages  int
		total     int
		wantSpec  string
		wantTrunc bool
		wantFirst int
		wantLast  int
	}{
		{"empty spec truncated", "", 3, 5, "1-3", true, 1, 3},
		{"range capped", "1-5", 3, 5, "1-3", true, 1, 3},
		{"range within cap", "2-3", 10, 5, "2-3", false, 2, 3},
		{"single page", "3", 10, 5, "3", false, 3, 3},
		{"no cap", "", 0, 5, "", false, 1, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			eff, trunc, err := capPageSpec(c.spec, c.maxPages, c.total)
			if err != nil {
				t.Fatalf("capPageSpec(%q,%d,%d): %v", c.spec, c.maxPages, c.total, err)
			}
			if eff != c.wantSpec || trunc != c.wantTrunc {
				t.Fatalf("capPageSpec(%q,%d,%d) = (%q,%v), want (%q,%v)",
					c.spec, c.maxPages, c.total, eff, trunc, c.wantSpec, c.wantTrunc)
			}
			f, l := pageBounds(eff, c.total)
			if f != c.wantFirst || l != c.wantLast {
				t.Errorf("pageBounds(%q,%d) = (%d,%d), want (%d,%d)",
					eff, c.total, f, l, c.wantFirst, c.wantLast)
			}
			if trunc {
				for p := f; p <= l; p++ {
					if !pageInRange(p, eff) {
						t.Errorf("page %d in OCR render range but outside effSpec %q", p, eff)
					}
				}
			}
		})
	}
}
