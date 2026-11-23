package proposal

// outline.go — 方案库纯函数面（无库依赖）：源文件标题锚点定位、检索词切分与
// 转义、命中段落定位。存储层（proposal.go）与工具层共用。

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// ── 标题锚点定位（录入形态 B：模型只给标题与层级，正文由源文件逐字切出）──────

// headingLikeMaxRunes 「标题样短行」上限：超过它的正文行不算标题前缀匹配。
const headingLikeMaxRunes = 60

// normHeading 把一行归一成可比较的标题文本：去 BOM、去 ATX 井号前缀与行尾
// 井号、去加粗/删除线记号、去首尾空白与常见收尾标点、压缩内部空白。
func normHeading(s string) string {
	s = strings.TrimPrefix(s, "\uFEFF")
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		i := strings.IndexFunc(s, func(r rune) bool { return r != '#' })
		if i < 0 {
			return ""
		}
		if i > 6 {
			return "" // 七个以上井号不是 ATX 标题（防把分割线当标题）
		}
		s = strings.TrimSpace(s[i:])
		s = strings.Trim(s, "# \t")
	}
	s = strings.TrimSpace(s)
	for _, m := range []string{"**", "__", "~~"} {
		s = strings.ReplaceAll(s, m, "")
	}
	s = strings.TrimSpace(s)
	// 首部只去中文标点与空白——不去 '.'，免得把「.5 概述」这类编号削掉。
	s = strings.TrimLeft(s, " \t：:，,；;、。")
	s = strings.Trim(s, " \t：:。，,；;、")
	return strings.Join(strings.Fields(s), " ")
}

// onlyPageRef 判定标题之后的残余是否只剩页码/点线（目录行形态）。
func onlyPageRef(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	digit := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digit = true
		case strings.ContainsRune(" \t.．…·-‐—–~,，、", r):
		default:
			return false
		}
	}
	return digit
}

// headingGrade 给「这一行是不是标题 title 所在行」定级，越高越像：
//
//	5  整行相等
//	4  Markdown 标题行（# 开头）且含标题
//	3  短行（≤60 字）、不以句末标点收尾，以标题开头
//	2  短行、不以句末标点收尾，含标题
//	1  其他含标题的行（正文句、长行）
//	0  目录行（标题后只剩页码/点线）或不含
//
// 目录里每个标题都会出现一次，锚到目录行等于整册切错，恒 0。正文句以标题
// 开头很常见（标题「监测」对「监测点布置与预警值。」），所以句末标点收尾的
// 行排在真标题之后。
func headingGrade(line, title string) int {
	l := normHeading(line)
	t := normHeading(title)
	if l == "" || t == "" {
		return 0
	}
	ll, tl := strings.ToLower(l), strings.ToLower(t)
	if ll == tl {
		return 5
	}
	i := strings.LastIndex(ll, tl)
	if i < 0 || onlyPageRef(ll[i+len(tl):]) {
		return 0
	}
	if strings.HasPrefix(strings.TrimSpace(line), "#") {
		return 4
	}
	// 句末标点看原始行：normHeading 会把行尾句号削掉，削完就分不出标题和正文句了。
	rs := []rune(l)
	if len(rs) > headingLikeMaxRunes || isSentenceEnd([]rune(strings.TrimSpace(line))) {
		return 1
	}
	if strings.HasPrefix(ll, tl) {
		return 3
	}
	return 2
}

// HeadingMissError 标题在源文件里定位不到：带序号与附近的标题样行，模型据此
// 修正 outline 重发（fail-closed：整批不落库）。
type HeadingMissError struct {
	Title    string
	Ordinal  int
	FromLine int
	Near     []string // 「行号: 文本」
}

func (e *HeadingMissError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "outline 第 %d 项 %q 在源文件第 %d 行之后找不到", e.Ordinal, e.Title, e.FromLine)
	if e.FromLine == 0 {
		b.WriteString("（全文找不到）")
	}
	if len(e.Near) > 0 {
		b.WriteString("；从该行起的标题样行：")
		for _, n := range e.Near {
			b.WriteString("\n  " + n)
		}
	}
	return b.String()
}

// findHeading 在 lines[from:]（0-based）里定位 title 所在行，返回 0-based 行号
// 与匹配级（headingGrade）。hint>0 时模型给了 1-based 行号：只校验该行（须在
// from 之后且确含标题），不再前向搜索——模型明确指行时尊重它，指错就报错。
// 否则取匹配级最高、同级取最靠前的行。
func findHeading(lines []string, from int, title string, hint int) (int, int, error) {
	miss := &HeadingMissError{Title: title, FromLine: from}
	if hint > 0 {
		idx := hint - 1
		if idx < from || idx >= len(lines) {
			return 0, 0, fmt.Errorf("outline 标题 %q 的 line=%d 越界（须在第 %d 行到第 %d 行之间）",
				title, hint, from+1, len(lines))
		}
		g := headingGrade(lines[idx], title)
		if g == 0 {
			return 0, 0, fmt.Errorf("outline 标题 %q 指向第 %d 行，但该行不含此标题（或是目录行）：%q",
				title, hint, truncRunes(lines[idx], 80))
		}
		return idx, g, nil
	}
	best, bestIdx := 0, -1
	shown := 0
	for i := from; i < len(lines); i++ {
		if g := headingGrade(lines[i], title); g > best {
			best, bestIdx = g, i
			if g == 5 {
				break
			}
		}
		if shown < 12 && looksLikeHeading(lines[i]) {
			miss.Near = append(miss.Near, fmt.Sprintf("%d: %s", i+1, truncRunes(strings.TrimSpace(lines[i]), 60)))
			shown++
		}
	}
	if bestIdx < 0 {
		return 0, 0, miss
	}
	return bestIdx, best, nil
}

// looksLikeHeading 粗判标题样行：ATX 标题，或较短的、不以句读收尾的非空行。
func looksLikeHeading(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" {
		return false
	}
	if strings.HasPrefix(t, "#") {
		return true
	}
	if len([]rune(t)) > headingLikeMaxRunes || strings.HasPrefix(t, "|") {
		return false
	}
	return !isSentenceEnd([]rune(t))
}

// isSentenceEnd 判定文本是否以句末标点收尾（句号/问叹号/逗号/分号）。冒号
// 不算——「一、编制依据：」是常见标题形态。
func isSentenceEnd(rs []rune) bool {
	if len(rs) == 0 {
		return false
	}
	return strings.ContainsRune("。，；！？.,;!?", rs[len(rs)-1])
}

// prefixHash 源文件前 n 行的指纹：分批录入期间每批核对，源文件被改过即拒。
func prefixHash(lines []string, n int) string {
	if n > len(lines) {
		n = len(lines)
	}
	h := sha256.New()
	for _, l := range lines[:n] {
		h.Write([]byte(l))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func truncRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// ── 检索词切分与转义 ────────────────────────────────────────────────

// termSeparators 检索词分隔符：空白 + 中英文标点。词内保留 -_. 等编号字符
// （「HJ 25.1-2019」会被空白切成 HJ 与 25.1-2019 两词，AND 语义等价）。
const termSeparators = " \t\r\n　,，、。；;:：/\\|()（）[]【】「」『』<>《》\"'“”‘’?？!！*^~=+&"

// SplitTerms 把检索式切成去重保序的词；只由分隔符构成的词丢弃。
func SplitTerms(query string) []string {
	fields := strings.FieldsFunc(query, func(r rune) bool { return strings.ContainsRune(termSeparators, r) })
	seen := make(map[string]bool, len(fields))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}

// quoteFTS FTS5 MATCH 短语转义：双引号包裹（内部双引号双写），与
// semantic 包同口径。调用方须保证词长 ≥3 字符——trigram 下更短的短语恒零行。
func quoteFTS(term string) string {
	return `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
}

// escapeLike LIKE 通配符转义（配合 ESCAPE '\'）。
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// likePattern 包成两端通配的 LIKE 模式。
func likePattern(term string) string { return "%" + escapeLike(term) + "%" }

// ── 命中段落定位 ───────────────────────────────────────────────────

// Passage 正文里一段命中窗口：Start/End 为 rune 下标（含头不含尾）。
type Passage struct {
	Start int
	End   int
	Terms []string // 窗口内出现的词

	anchor, anchorEnd int // 生成该窗口的那次命中位置（吸附边界时不可越过）
}

// FindPassages 在 text 里找出最多 maxN 个互不重叠的命中窗口，按「覆盖词数
// 多者优先、同分靠前者优先」挑选，返回时按文中位置排序（保持叙述顺序）。
// before/after 是命中词前后各带多少 rune。无命中返回 nil。
func FindPassages(text string, terms []string, before, after, maxN int) []Passage {
	if text == "" || len(terms) == 0 || maxN <= 0 {
		return nil
	}
	rs := []rune(text)
	low := []rune(strings.ToLower(text))
	if len(low) != len(rs) {
		low = rs // ToLower 改变了 rune 数（罕见）：退回大小写敏感匹配
	}
	type occ struct {
		term int
		pos  int
		end  int
	}
	var occs []occ
	for ti, t := range terms {
		tr := []rune(strings.ToLower(t))
		if len(tr) == 0 || len(tr) > len(low) {
			continue
		}
		for i := 0; i+len(tr) <= len(low); i++ {
			if low[i] != tr[0] {
				continue
			}
			if runeEqual(low[i:i+len(tr)], tr) {
				occs = append(occs, occ{term: ti, pos: i, end: i + len(tr)})
			}
		}
	}
	if len(occs) == 0 {
		return nil
	}
	type cand struct {
		passage Passage
		order   int
	}
	cands := make([]cand, 0, len(occs))
	for n, o := range occs {
		st := o.pos - before
		if st < 0 {
			st = 0
		}
		en := o.end + after
		if en > len(rs) {
			en = len(rs)
		}
		seen := map[int]bool{o.term: true}
		for _, p := range occs {
			if !seen[p.term] && p.pos < en && p.end > st {
				seen[p.term] = true
			}
		}
		termsIn := make([]string, 0, len(seen))
		for ti := range seen {
			termsIn = append(termsIn, terms[ti])
		}
		sort.Strings(termsIn)
		cands = append(cands, cand{passage: Passage{Start: st, End: en, Terms: termsIn, anchor: o.pos, anchorEnd: o.end}, order: n})
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if len(cands[i].passage.Terms) != len(cands[j].passage.Terms) {
			return len(cands[i].passage.Terms) > len(cands[j].passage.Terms)
		}
		return cands[i].order < cands[j].order
	})
	picked := make([]Passage, 0, maxN)
	for _, c := range cands {
		overlap := false
		for _, p := range picked {
			if c.passage.Start < p.End && c.passage.End > p.Start {
				overlap = true
				break
			}
		}
		if !overlap {
			picked = append(picked, c.passage)
			if len(picked) >= maxN {
				break
			}
		}
	}
	sort.Slice(picked, func(i, j int) bool { return picked[i].Start < picked[j].Start })
	for i := range picked {
		picked[i].Start, picked[i].End = snapBounds(rs, picked[i].Start, picked[i].End, picked[i].anchor, picked[i].anchorEnd)
	}
	return picked
}

func runeEqual(a, b []rune) bool {
	for i := range b {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// snapRadius 吸附时向前/向后找句读或换行的最大距离（rune）。
const snapRadius = 80

// snapBounds 把窗口边界吸附到句子/段落边界：起点前移到命中之前最近一个换行
// 或句读之后，终点后移到命中之后最近的换行或句读处（各在 snapRadius 内）。
// 命中本身永远留在窗口里——这是给人和模型看的「命中位置上下文」，不能为了
// 对齐句子把命中词切掉。
func snapBounds(rs []rune, st, en, anchor, anchorEnd int) (int, int) {
	isBreak := func(r rune) bool {
		return r == '\n' || strings.ContainsRune("。；！？", r)
	}
	lim := anchor - snapRadius
	if lim < st {
		lim = st
	}
	for i := anchor - 1; i >= lim; i-- {
		if isBreak(rs[i]) {
			st = i + 1
			break
		}
	}
	lim = anchorEnd + snapRadius
	if lim > en {
		lim = en
	}
	for i := anchorEnd; i < lim; i++ {
		if isBreak(rs[i]) {
			en = i + 1
			break
		}
	}
	return st, en
}

// RenderPassage 把窗口文本输出为带命中标记（【词】）的字符串；terms 为空则原样。
func RenderPassage(rs []rune, p Passage, terms []string) string {
	if p.Start < 0 {
		p.Start = 0
	}
	if p.End > len(rs) {
		p.End = len(rs)
	}
	if p.End <= p.Start {
		return ""
	}
	body := rs[p.Start:p.End]
	if len(terms) == 0 {
		return string(body)
	}
	low := []rune(strings.ToLower(string(body)))
	if len(low) != len(body) {
		return string(body)
	}
	marked := make([]bool, len(body))
	for _, t := range terms {
		tr := []rune(strings.ToLower(t))
		if len(tr) == 0 {
			continue
		}
		for i := 0; i+len(tr) <= len(low); i++ {
			if low[i] == tr[0] && runeEqual(low[i:i+len(tr)], tr) {
				for k := i; k < i+len(tr); k++ {
					marked[k] = true
				}
			}
		}
	}
	var b strings.Builder
	open := false
	for i, r := range body {
		if marked[i] && !open {
			b.WriteString("【")
			open = true
		} else if !marked[i] && open {
			b.WriteString("】")
			open = false
		}
		b.WriteRune(r)
	}
	if open {
		b.WriteString("】")
	}
	return b.String()
}
