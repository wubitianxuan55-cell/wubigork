package project

import (
	"fmt"
	"regexp"
	"testing"
)

// TestParseChapterFileName_EquivalentToLegacyRegex 等价钉（审计 IN1-03）：
// ParseChapterFileName 必须与历史正则 ^(\d{3})([a-z]?)\.md$ 在任意输入上同判。
// 该正则曾在三处重复（export.listChapters / stats.Collect /
// internal/graph/consistency.go），本用例以正则为准绳钉死唯一实现口径。
func TestParseChapterFileName_EquivalentToLegacyRegex(t *testing.T) {
	re := regexp.MustCompile(`^(\d{3})([a-z]?)\.md$`)
	names := []string{
		// 接受集
		"001.md", "000.md", "999.md", "012.md",
		"001a.md", "002b.md", "000z.md", "999a.md",
		// 拒绝集
		"1.md", "12.md", "0001.md", "001ab.md", "001A.md",
		"001.txt", "001.md.bak", "001a.txt", "abc.md", "summary.md",
		"00.md", "001.md ", " 001.md", "001.md5", "第001章.md",
		"", ".md", "001.mdmd",
	}
	for _, name := range names {
		var wantNum int
		var wantBranch string
		wantOK := false
		if m := re.FindStringSubmatch(name); m != nil {
			wantOK = true
			fmt.Sscanf(m[1], "%d", &wantNum)
			wantBranch = m[2]
		}
		gotNum, gotBranch, gotOK := ParseChapterFileName(name)
		if gotOK != wantOK || gotNum != wantNum || gotBranch != wantBranch {
			t.Errorf("ParseChapterFileName(%q) = (%d, %q, %v), want (%d, %q, %v)",
				name, gotNum, gotBranch, gotOK, wantNum, wantBranch, gotOK)
		}
	}
}

// TestParseChapterFileName_CanonicalForms 正稿家族代表形状（语义自文档，
// 与 ParseChapterFileName 注释中的 IN1-03 口径对照表互为印证）。
func TestParseChapterFileName_CanonicalForms(t *testing.T) {
	cases := []struct {
		name   string
		num    int
		branch string
		ok     bool
	}{
		{"001.md", 1, "", true},
		{"000.md", 0, "", true}, // 章号 0 合法，与历史 Sscanf %d 口径一致
		{"002a.md", 2, "a", true},
		{"003z.md", 3, "z", true},
		{"001ab.md", 0, "", false},   // 分支至多一个小写字母
		{"001A.md", 0, "", false},    // 分支仅小写
		{"12.md", 0, "", false},      // 必须恰三位数字
		{"0001.md", 0, "", false},    // 四位数字拒绝
		{"001.md.bak", 0, "", false}, // 严格 ".md" 结尾（Sscanf 迁移口径容忍，本口径不容忍）
		{"note.md", 0, "", false},
	}
	for _, c := range cases {
		num, branch, ok := ParseChapterFileName(c.name)
		if num != c.num || branch != c.branch || ok != c.ok {
			t.Errorf("ParseChapterFileName(%q) = (%d, %q, %v), want (%d, %q, %v)",
				c.name, num, branch, ok, c.num, c.branch, c.ok)
		}
	}
}
