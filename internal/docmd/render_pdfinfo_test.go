package docmd

import (
	"testing"
)

// 实测反馈②：PDFPageCount/parsePdfinfoPages 单测（pdfinfo 外呼走 seam 堵）。

func TestParsePdfinfoPages(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"常规", "Title: a.pdf\nCreator: Word\nPages:          175\n", 175},
		{"CRLF", "Pages:         12\r\nEncrypted:      no\r\n", 12},
		{"无 Pages 行", "Title: x\n", 0},
		{"非数字", "Pages:         n/a\n", 0},
		{"零与负数拒收", "Pages:         0\n", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parsePdfinfoPages(tc.in); got != tc.want {
				t.Fatalf("parsePdfinfoPages(%q)=%d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestPDFPageCount_NoPdfinfo(t *testing.T) {
	old := pdfinfoLookup
	pdfinfoLookup = func() string { return "" }
	defer func() { pdfinfoLookup = old }()
	n, err := PDFPageCount("whatever.pdf")
	if err != nil || n != 0 {
		t.Fatalf("缺 pdfinfo 应降级 (0,nil)，实得 (%d,%v)", n, err)
	}
}
