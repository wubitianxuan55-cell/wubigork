package strutil

import "testing"

// TestContainsHanIdenticalToOriginals verifies ContainsHan matches the local
// implementations it replaces (costimport.hasHan and knowledge.isCJStr —
// identical bodies; the latter a misnomer for CJK) and pins the fixed
// semantics: Han runes only — kanji inside Japanese text DO count (they are
// Han); pure Kana/Hangul/punctuation do not.
func TestContainsHanIdenticalToOriginals(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"170Kg/m³*0.2m", false}, // 纯表达式行无汉字（costimport 既有口径注释）
		{"plain ascii", false},
		{"人工费", true},
		{"混排 m³ 混排", true},
		{"日本語カタカナ", true},   // 「日本語」是汉字（Han），假名不单独构成否决
		{"カタカナひらがな", false}, // 纯假名非 Han
		{"한국어", false},      // 谚文非 Han
		{"　", false},        // U+3000 全角空格非 Han
		{"「主题」", true},
	}
	for _, c := range cases {
		if got := ContainsHan(c.in); got != c.want {
			t.Fatalf("ContainsHan(%q)=%v, want %v", c.in, got, c.want)
		}
	}
}
