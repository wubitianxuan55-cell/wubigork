// Package proposal — 来源定位器：把 AI 摘录（quote）映射到原文偏移/页码
package proposal

import "strings"

type normIndex struct {
	origStart int
	origLen   int
}

// normalizeWithIndex 去掉全部空白，并记录每个归一化字符的原文偏移
func normalizeWithIndex(s string) (string, []normIndex) {
	var sb strings.Builder
	var idx []normIndex
	for i, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		sb.WriteRune(r)
		idx = append(idx, normIndex{origStart: i, origLen: len(string(r))})
	}
	return sb.String(), idx
}

// LocateQuote 在 markdown 中定位 quote。
// 先精确匹配；失败后做忽略空白匹配（仅比较非空白字符序列）。
// 返回原文的 [start, end) 字节区间。
func LocateQuote(markdown, quote string) (int, int, bool) {
	if quote == "" {
		return 0, 0, false
	}
	if idx := strings.Index(markdown, quote); idx >= 0 {
		return idx, idx + len(quote), true
	}
	nm, nidx := normalizeWithIndex(markdown)
	nq, _ := normalizeWithIndex(quote)
	if nq == "" {
		return 0, 0, false
	}
	if pos := strings.Index(nm, nq); pos >= 0 {
		startRune := len([]rune(nm[:pos]))
		nqRunes := len([]rune(nq))
		start := nidx[startRune].origStart
		last := nidx[startRune+nqRunes-1]
		return start, last.origStart + last.origLen, true
	}
	return 0, 0, false
}

// LocatePage 返回包含 quote 的页码（按页文本定位；未命中返回 0）
func LocatePage(pages []PageText, quote string) int {
	if quote == "" {
		return 0
	}
	for _, p := range pages {
		if p.Text == "" {
			continue
		}
		if _, _, ok := LocateQuote(p.Text, quote); ok {
			return p.Page
		}
	}
	return 0
}
