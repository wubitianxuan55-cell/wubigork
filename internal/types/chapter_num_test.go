package types

import "testing"

// TestChapterNumOfParseMatrix 审计 P1 AP1-11：章号解析唯一口径矩阵——
// 补零/非补零/分支字母/无数字/超长前导全钉死（app 层本地副本已收敛到此）。
func TestChapterNumOfParseMatrix(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"001.md", 1},
		{"001a.md", 1},
		{"1.md", 1},
		{"0123.md", 123},
		{"0012345.md", 12345},
		{"summary.md", 0},
		{"", 0},
		{".md", 0},
	}
	for _, c := range cases {
		if got := ChapterNumOf(c.in); got != c.want {
			t.Errorf("ChapterNumOf(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
