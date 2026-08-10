package docmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildPagePDF 生成指定页数、每页带唯一文本的合成 PDF（纯文本流，无需 poppler）。
func buildPagePDF(t testing.TB, pages int) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")
	for i := 1; i <= pages; i++ {
		fmt.Fprintf(&sb, "/Type /Page\nBT /F1 12 Tf 72 720 Td (第 %d 页内容) Tj ET\n", i)
	}
	path := filepath.Join(t.TempDir(), fmt.Sprintf("pages-%d.pdf", pages))
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	return path
}

func TestCapPageSpec(t *testing.T) {
	cases := []struct {
		name      string
		spec      string
		maxPages  int
		total     int
		wantSpec  string
		wantTrunc bool
		wantErr   bool
	}{
		{"no cap", "", 0, 1500, "", false, false},
		{"no cap with spec", "1-10", 0, 1500, "1-10", false, false},
		{"under cap", "", 500, 300, "", false, false},
		{"empty spec capped", "", 500, 1500, "1-500", true, false},
		{"spec within cap", "1-100", 500, 1500, "1-100", false, false},
		{"spec over cap", "1-2000", 500, 1500, "1-500", true, false},
		{"comma spec clamped", "1-3,7-9,600-700", 500, 1500, "1-3,7-9", true, false},
		{"spec starts over cap", "600-700", 500, 1500, "", false, true},
		{"single page over cap", "800", 500, 1500, "", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, trunc, err := capPageSpec(c.spec, c.maxPages, c.total)
			if c.wantErr {
				if err == nil {
					t.Fatalf("capPageSpec(%q,%d,%d) expected error, got %q", c.spec, c.maxPages, c.total, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("capPageSpec(%q,%d,%d) unexpected error: %v", c.spec, c.maxPages, c.total, err)
			}
			if got != c.wantSpec || trunc != c.wantTrunc {
				t.Errorf("capPageSpec(%q,%d,%d) = (%q,%v), want (%q,%v)",
					c.spec, c.maxPages, c.total, got, trunc, c.wantSpec, c.wantTrunc)
			}
		})
	}
}

func TestParsePageSpecBounds(t *testing.T) {
	cases := []struct {
		spec      string
		wantFirst int
		wantLast  int
		wantOK    bool
	}{
		{"1-5", 1, 5, true},
		{"1,3,5", 1, 5, true},
		{"3", 3, 3, true},
		{"7-9,1-3", 1, 9, true},
		{"", 1, 0, false},
		{"abc", 1, 0, false},
	}
	for _, c := range cases {
		f, l, ok := parsePageSpecBounds(c.spec)
		if f != c.wantFirst || l != c.wantLast || ok != c.wantOK {
			t.Errorf("parsePageSpecBounds(%q) = (%d,%d,%v), want (%d,%d,%v)",
				c.spec, f, l, ok, c.wantFirst, c.wantLast, c.wantOK)
		}
	}
}

func TestClampPageSpec(t *testing.T) {
	cases := []struct {
		spec string
		max  int
		want string
	}{
		{"1-3,7-9,600-700", 500, "1-3,7-9"},
		{"1-3", 2, "1-2"},
		{"3", 2, ""},
		{"1,3,5", 4, "1,3"},
		{"1-3,4-6", 5, "1-3,4-5"},
	}
	for _, c := range cases {
		if got := clampPageSpec(c.spec, c.max); got != c.want {
			t.Errorf("clampPageSpec(%q,%d) = %q, want %q", c.spec, c.max, got, c.want)
		}
	}
}

func TestPageBounds(t *testing.T) {
	if f, l := pageBounds("", 1200); f != 1 || l != 1200 {
		t.Errorf("pageBounds empty = (%d,%d), want (1,1200)", f, l)
	}
	if f, l := pageBounds("1-3,7-9", 1200); f != 1 || l != 9 {
		t.Errorf("pageBounds spec = (%d,%d), want (1,9)", f, l)
	}
}

func TestConvertLimit_PDFPageCap(t *testing.T) {
	path := buildPagePDF(t, 1200)

	md, total, truncated, err := ConvertLimit(path, "", 100)
	if err != nil {
		t.Fatalf("ConvertLimit: %v", err)
	}
	if total != 1200 {
		t.Errorf("total = %d, want 1200", total)
	}
	if !truncated {
		t.Errorf("truncated = false, want true")
	}
	if !strings.Contains(md, "第 1 页内容") {
		t.Errorf("md missing first page text")
	}
	if strings.Contains(md, "第 101 页内容") {
		t.Errorf("md contains page beyond cap (第 101 页)")
	}

	// 显式页码范围在上限内不截断
	md2, total2, trunc2, err := ConvertLimit(path, "1-3", 100)
	if err != nil {
		t.Fatalf("ConvertLimit(1-3): %v", err)
	}
	if total2 != 1200 || trunc2 {
		t.Errorf("total2/trunc2 = %d/%v, want 1200/false", total2, trunc2)
	}
	if !strings.Contains(md2, "第 2 页内容") || strings.Contains(md2, "第 4 页内容") {
		t.Errorf("range 1-3 extraction wrong")
	}

	// 无上限时完整转换
	md4, total4, trunc4, err := ConvertLimit(path, "", 0)
	if err != nil || trunc4 {
		t.Fatalf("no cap: err=%v trunc=%v", err, trunc4)
	}
	if total4 != 1200 || !strings.Contains(md4, "第 1200 页内容") {
		t.Errorf("no-cap extraction wrong: total=%d", total4)
	}
}

// TestConvertLimitProgress_TextPDFNoProgress 文本型 PDF 不走 OCR，进度回调不应触发，
// 但页数上限与截断标记仍应生效。
func TestConvertLimitProgress_TextPDFNoProgress(t *testing.T) {
	path := buildPagePDF(t, 10)
	called := false
	_, total, truncated, err := ConvertLimitProgress(path, "", 5, func(done, n int) { called = true })
	if err != nil {
		t.Fatalf("ConvertLimitProgress: %v", err)
	}
	if total != 10 || !truncated {
		t.Errorf("total/truncated = %d/%v, want 10/true", total, truncated)
	}
	if called {
		t.Errorf("progress fired for text PDF")
	}
}
