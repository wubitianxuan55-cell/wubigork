// Package textsim 标题/短文本相似度（查重/合并提示用）：归一化后按
// CJK 二元组 + 字母数字词的集合 Dice 系数计算。纯本地、无依赖。
package textsim

import (
	"strings"
	"unicode"
)

// Similarity 返回 0~1 的相似度（1=完全相同）。
func Similarity(a, b string) float64 {
	ta := unique(tokens(a))
	tb := unique(tokens(b))
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	set := make(map[string]bool, len(ta))
	for _, t := range ta {
		set[t] = true
	}
	inter := 0
	for _, t := range tb {
		if set[t] {
			inter++
		}
	}
	return float64(2*inter) / float64(len(ta)+len(tb))
}

func tokens(s string) []string {
	s = strings.ToLower(strings.TrimSpace(s))
	var out []string
	var cjk []rune
	var word strings.Builder
	flushCJK := func() {
		if len(cjk) == 0 {
			return
		}
		out = append(out, string(cjk))
		for i := 0; i+1 < len(cjk); i++ {
			out = append(out, string([]rune{cjk[i], cjk[i+1]}))
		}
		cjk = cjk[:0]
	}
	flushWord := func() {
		if word.Len() > 0 {
			out = append(out, word.String())
			word.Reset()
		}
	}
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Han, r):
			flushWord()
			cjk = append(cjk, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			flushCJK()
			word.WriteRune(r)
		default:
			flushCJK()
			flushWord()
		}
	}
	flushCJK()
	flushWord()
	return out
}

func unique(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
