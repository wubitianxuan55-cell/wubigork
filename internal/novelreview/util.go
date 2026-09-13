package novelreview

// 文本结构助手（纯函数；中文一律 rune 计数，与 internal/novelstyle 同口径）。

import (
	"sort"
	"strings"
	"unicode"
)

// splitParagraphs 按换行切段（空行不算段落，但保留其分隔语义）。
func splitParagraphs(runes []rune) []paragraph {
	var out []paragraph
	start := 0
	idx := 0
	flush := func(end int) {
		if end <= start {
			return
		}
		text := strings.TrimSpace(string(runes[start:end]))
		if text == "" {
			return
		}
		idx++
		out = append(out, paragraph{idx: idx, start: start, end: end, text: text})
	}
	for i, r := range runes {
		if r == '\n' {
			flush(i)
			start = i + 1
		}
	}
	flush(len(runes))
	return out
}

// countNonSpaceRunes 非空白字数。
func countNonSpaceRunes(runes []rune) int {
	n := 0
	for _, r := range runes {
		if !unicode.IsSpace(r) && r != 0x3000 {
			n++
		}
	}
	return n
}

// findAllRunes 返回 sub 在 runes 中所有出现的起始下标（非重叠）。
func findAllRunes(runes []rune, sub []rune) []int {
	var out []int
	if len(sub) == 0 || len(sub) > len(runes) {
		return out
	}
	for i := 0; i+len(sub) <= len(runes); i++ {
		match := true
		for j := range sub {
			if runes[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			out = append(out, i)
			i += len(sub) - 1
		}
	}
	return out
}

// countOccurrences 统计子串出现次数（非重叠）。
func countOccurrences(runes []rune, sub string) int {
	return len(findAllRunes(runes, []rune(sub)))
}

// findMarkerStarts 汇总多个判定词的命中起点（去重 + 升序）。
func findMarkerStarts(runes []rune, markers []string) []int {
	set := map[int]bool{}
	for _, m := range markers {
		for _, s := range findAllRunes(runes, []rune(m)) {
			set[s] = true
		}
	}
	out := make([]int, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Ints(out)
	return out
}

// longestGap 相邻命中之间的最长空档（rune 计），返回长度与起点。
func longestGap(runes []rune, hits []int) (int, int) {
	if len(hits) == 0 {
		return len(runes), 0
	}
	best, bestStart := hits[0], 0
	for i := 1; i < len(hits); i++ {
		if gap := hits[i] - hits[i-1]; gap > best {
			best, bestStart = gap, hits[i-1]
		}
	}
	if gap := len(runes) - hits[len(hits)-1]; gap > best {
		best, bestStart = gap, hits[len(hits)-1]
	}
	return best, bestStart
}

// containsQuote 段落内是否有对话引号。
func containsQuote(s string) bool {
	return strings.ContainsAny(s, "「」『』“”")
}

// containsAny 子串命中判定（多候选）。
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// countQuotedRunes 统计引号内字数（对话占比口径）。
func countQuotedRunes(runes []rune) int {
	pairs := [][2]rune{{'「', '」'}, {'『', '』'}, {'“', '”'}}
	total := 0
	for _, p := range pairs {
		depth := 0
		for _, r := range runes {
			switch r {
			case p[0]:
				depth++
			case p[1]:
				if depth > 0 {
					depth--
				}
			default:
				if depth > 0 && !unicode.IsSpace(r) {
					total++
				}
			}
		}
	}
	return total
}

// countRune 统计单个字符出现次数。
func countRune(runes []rune, target rune) int {
	n := 0
	for _, r := range runes {
		if r == target {
			n++
		}
	}
	return n
}

// spanAt 把 rune 下标定位到所属段落（0=无段落），证据窗口 30 rune。
func spanAt(runes []rune, paras []paragraph, pos int) EvidenceSpan {
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}
	end := pos + 30
	if end > len(runes) {
		end = len(runes)
	}
	return EvidenceSpan{Start: pos, End: end, Paragraph: paragraphIndexOf(paras, pos)}
}

// paragraphIndexOf 定位 rune 下标所属段落号（1-based，0=未命中）。
func paragraphIndexOf(paras []paragraph, pos int) int {
	for _, pa := range paras {
		if pos >= pa.start && pos < pa.end {
			return pa.idx
		}
	}
	return 0
}

// markerSpans 取前 limit 处子串命中的证据位置。
func markerSpans(runes []rune, paras []paragraph, sub string, limit int) []EvidenceSpan {
	var out []EvidenceSpan
	m := []rune(sub)
	for _, s := range findAllRunes(runes, m) {
		out = append(out, EvidenceSpan{Start: s, End: s + len(m), Paragraph: paragraphIndexOf(paras, s)})
		if len(out) >= limit {
			break
		}
	}
	return out
}

// dashSpans 破折号命中（——、—、--；跳过数字区间如 1990—2000）。
func dashSpans(runes []rune, paras []paragraph) []EvidenceSpan {
	var out []EvidenceSpan
	for i := 0; i < len(runes); i++ {
		if runes[i] != '—' && runes[i] != '-' {
			continue
		}
		j := i
		for j < len(runes) && (runes[j] == '—' || runes[j] == '-') {
			j++
		}
		if i > 0 && j < len(runes) && unicode.IsDigit(runes[i-1]) && unicode.IsDigit(runes[j]) {
			i = j - 1
			continue
		}
		if runes[i] == '-' && j-i < 2 {
			i = j - 1
			continue
		}
		out = append(out, EvidenceSpan{Start: i, End: j, Paragraph: paragraphIndexOf(paras, i)})
		i = j - 1
	}
	return out
}

// quotedSpanAfter 在 [from, from+window) 内找第一处引号实体，返回文本、字数与区间。
func quotedSpanAfter(runes []rune, from, window int) (string, int, int, int) {
	limit := from + window
	if limit > len(runes) {
		limit = len(runes)
	}
	for i := from; i < limit; i++ {
		var close rune
		switch runes[i] {
		case '「':
			close = '」'
		case '『':
			close = '』'
		case '“':
			close = '”'
		default:
			continue
		}
		for j := i + 1; j < limit; j++ {
			if runes[j] == close {
				inner := runes[i+1 : j]
				return string(inner), countNonSpaceRunes(inner), i, j + 1
			}
		}
	}
	return "", 0, 0, 0
}

// quotedSpanBefore 在 (upto-window, upto) 内**向前**找最近一处引号实体（「…」这三个字）。
func quotedSpanBefore(runes []rune, upto, window int) (string, int, int, int) {
	start := upto - window
	if start < 0 {
		start = 0
	}
	if upto > len(runes) {
		upto = len(runes)
	}
	for j := upto - 1; j >= start; j-- {
		var open rune
		switch runes[j] {
		case '」':
			open = '「'
		case '』':
			open = '『'
		case '”':
			open = '“'
		default:
			continue
		}
		for i := j - 1; i >= start; i-- {
			if runes[i] == open {
				inner := runes[i+1 : j]
				return string(inner), countNonSpaceRunes(inner), i, j + 1
			}
		}
	}
	return "", 0, 0, 0
}

// sortSpans 证据按位置升序。
func sortSpans(spans []EvidenceSpan) {
	sort.Slice(spans, func(i, j int) bool { return spans[i].Start < spans[j].Start })
}

// joinCN 中文顿号连接（空列表给空串）。
func joinCN(parts []string) string {
	return strings.Join(parts, "、")
}

// absFloat 绝对值。
func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
