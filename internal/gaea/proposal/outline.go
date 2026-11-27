package proposal

// outline.go — 方案库纯函数面（无库依赖）：源文件标题锚点定位、检索词切分与
// 转义、命中段落定位。存储层（proposal.go）与工具层共用。

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
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

// ── 错字容错定位（2026-11-27 AI 反馈 #11）──────────────────────────
//
// 扫描件 OCR 会把标题认错一两个字（「抑制异味」→「抑制您味」）。outline 条目带
// display_title 时，title 定位不到就在标题样行里找与显示名只差一两个字的那一行：
// 模型给的是修正后的写法，不该被源文件的错字卡住。

// leadingNumber 取标题开头的编号（「5.2.5.2 」「第一章」「一、」「（一）」），没有返回空。
func leadingNumber(s string) string {
	if m := reChapter.FindStringSubmatch(s); m != nil {
		return s[:len(s)-len(m[len(m)-1])]
	}
	if m := reSection.FindStringSubmatch(s); m != nil {
		return s[:len(s)-len(m[len(m)-1])]
	}
	if m := reNumeric2.FindStringSubmatch(s); m != nil {
		return s[:len(s)-len(m[2])]
	}
	if m := reCnOrd.FindStringSubmatch(s); m != nil {
		return s[:len(s)-len(m[2])]
	}
	if m := reCnParen.FindStringSubmatch(s); m != nil {
		return s[:len(s)-len(m[2])]
	}
	return ""
}

// headingCore 标题去掉开头编号后的正文（比对错字用）。
func headingCore(s string) string {
	if n := leadingNumber(s); n != "" {
		return strings.TrimSpace(strings.TrimPrefix(s, n))
	}
	return s
}

// maxTitleTypos 允许的编辑距离：短标题最多差 1 个字，长一点的最多 2 个。
func maxTitleTypos(n int) int {
	if n < 4 {
		return 0 // 太短的标题差一个字就是另一个标题，不容错
	}
	if n <= 5 {
		return 1
	}
	return 2
}

// editDistance 两个 rune 串的编辑距离（插入/删除/替换各算 1）。
func editDistance(a, b []rune) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// typoDistance 报告一行标题与显示名 s 的差距：不是标题样行、或差得太远，返回 -1。
// 两边都先去编号再比（「5.2.5.2 抑制您味」对「抑制异味」）。
func typoDistance(line, s string) int {
	rs := []rune(headingCore(normHeading(s)))
	limit := maxTitleTypos(len(rs))
	if limit == 0 {
		return -1
	}
	core := []rune(headingCore(normHeading(line)))
	if len(core) == 0 || abs(len(core)-len(rs)) > limit {
		return -1
	}
	if d := editDistance(core, rs); d <= limit {
		return d
	}
	return -1
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// findHeadingByDisplay 在 lines[from:] 的标题样行里找与显示名 display 最接近的一行
// （差距最小的；同距取最靠前的）。返回 0-based 行号与差距，找不到返回 -1。
func findHeadingByDisplay(lines []string, from int, display string) (int, int) {
	bestIdx, bestDist := -1, -1
	for i := from; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" || strings.HasPrefix(t, "|") || len([]rune(t)) > headingLikeMaxRunes || isSentenceEnd([]rune(t)) {
			continue
		}
		if d := typoDistance(t, display); d >= 0 && (bestDist < 0 || d < bestDist) {
			bestIdx, bestDist = i, d
			if d == 0 {
				break
			}
		}
	}
	return bestIdx, bestDist
}

// ── 认标题（2026-11-27 AI 反馈 #12）：正文没有井号标题时按编号形态找标题样行 ──

const cnDigits = `[一二三四五六七八九十百千零〇两\d]+`

var (
	reATX      = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*$`)
	reChapter  = regexp.MustCompile(`^第` + cnDigits + `[\s　]*(章|篇|部分|部|卷|编|册)[\s　:：、.．]*(.+)$`)
	reSection  = regexp.MustCompile(`^第` + cnDigits + `[\s　]*节[\s　:：、.．]*(.+)$`)
	reNumeric2 = regexp.MustCompile(`^(\d{1,4}(?:\.\d{1,4})+)[\s　:：]*(\D.+)$`)
	reNum1     = regexp.MustCompile(`^(\d{1,3})[\s　]+([^\s\d].*)$`)
	reCnOrd    = regexp.MustCompile(`^([一二三四五六七八九十]{1,3})[\s　]*[、.．][\s　]*(.+)$`)
	reCnParen  = regexp.MustCompile(`^[（(][\s　]*([一二三四五六七八九十]{1,3})[\s　]*[）)][\s　]*(.+)$`)
	reListOrd  = regexp.MustCompile(`^(?:[（(][\s　]*(\d{1,2})[\s　]*[）)]|(\d{1,2})[\s　]*[.、])[^\d\s][\s　]*(.*)$`)
	// reTOCLead 目录行：标题后跟点线与页码（「第一章 工程概况 …… 5」）。
	reTOCLead = regexp.MustCompile(`(?:[.…·]{3,}|[-—–]{4,})[\s　]*\d{1,4}[\s　]*$`)
)

// HeadingCandidate 认出的一个标题样行。
type HeadingCandidate struct {
	Line    int    // 1-based 源文件行号
	Level   int    // 建议层级 1~4（按编号形态与上下文推的，以模型最终认定为准）
	Kind    string // 形态：井号 / 章 / 节 / 数字 / 一、/ （一）/ 1. / 表格行
	Title   string // 进 outline 的 title（源文件写法，带文档自带编号）
	InTable bool   // 藏在表格行里（低置信）
	Weak    bool   // 拿不准：单个数字开头、又对不上多级编号首位数字的行
}

// numberingOf 报告一行（已去粗体记号）的编号形态：章 / 节 / 数字 / 一、/（一）/ 1.，
// 以及数字编号的级数。认不出返回空串。
func numberingOf(s string) (fam string, depth int) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "*", ""))
	switch {
	case reChapter.MatchString(s):
		return "chapter", 1
	case reSection.MatchString(s):
		return "section", 2
	}
	if m := reNumeric2.FindStringSubmatch(s); m != nil {
		return "num", strings.Count(m[1], ".") + 1
	}
	switch {
	case reCnOrd.MatchString(s):
		return "cn1", 0
	case reCnParen.MatchString(s):
		return "cn2", 0
	case reListOrd.MatchString(s):
		return "list", 0
	case reNum1.MatchString(s):
		return "num", 1
	}
	return "", 0
}

func kindName(fam string) string {
	switch fam {
	case "chapter":
		return "章"
	case "section":
		return "节"
	case "num":
		return "数字"
	case "cn1":
		return "一、"
	case "cn2":
		return "（一）"
	case "list":
		return "1."
	}
	return "井号"
}

// tableHeadingCells 表格行里像编号标题的单元格（「| **6.资源配置方案** |」——
// 扫描件的章名常藏在表格行里）。
func tableHeadingCells(line string) []string {
	var out []string
	for _, cell := range strings.Split(strings.Trim(line, "|"), "|") {
		cell = strings.TrimSpace(strings.Trim(strings.TrimSpace(cell), "*"))
		if cell == "" {
			continue
		}
		if fam, _ := numberingOf(cell); fam == "chapter" || fam == "section" || fam == "num" || fam == "list" {
			out = append(out, normHeading(cell))
		}
	}
	return out
}

// DetectHeadings 在源文件行里找标题样行并推建议层级。只认编号形态（井号 / 第X章 /
// 第X节 / 1.1 / 一、/（一）/ 1.），不判语义——结果给模型过目，层级以模型最终
// 认定为准。目录行（标题后跟点线页码）、代码块、表格内的正文都不算；表格行里
// 带章节编号的单元格单列为低置信项。
func DetectHeadings(lines []string) []HeadingCandidate {
	type rawCand struct {
		c      HeadingCandidate
		fam    string // 编号形态（"" = 只有井号）
		depth  int    // 数字编号的级数
		hashes int    // 井号个数（非井号行为 0）
		num    string // 单级数字编号的数字（判它是不是某章的章号）
	}
	var raws []rawCand
	atxMin := 0
	inFence := false
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence || t == "" {
			continue
		}
		if strings.HasPrefix(t, "|") {
			for _, title := range tableHeadingCells(t) {
				fam, _ := numberingOf(title)
				raws = append(raws, rawCand{c: HeadingCandidate{Line: i + 1, Kind: "表格行", Title: title, InTable: true, Weak: true}, fam: fam})
			}
			continue
		}
		if reTOCLead.MatchString(t) || strings.HasPrefix(t, "![") || strings.HasPrefix(t, "<!--") {
			continue
		}
		if m := reATX.FindStringSubmatch(t); m != nil {
			hashes := len([]rune(m[1]))
			if atxMin == 0 || hashes < atxMin {
				atxMin = hashes
			}
			title := normHeading(m[2])
			fam, depth := numberingOf(title)
			raws = append(raws, rawCand{c: HeadingCandidate{Line: i + 1, Kind: kindName(fam), Title: title}, fam: fam, depth: depth, hashes: hashes})
			continue
		}
		if len([]rune(t)) > headingLikeMaxRunes || isSentenceEnd([]rune(t)) {
			continue
		}
		fam, depth := numberingOf(t)
		if fam == "" {
			continue
		}
		title := normHeading(t)
		if headingCore(title) == "" {
			continue // 只有编号没有标题文字
		}
		r := rawCand{c: HeadingCandidate{Line: i + 1, Kind: kindName(fam), Title: title}, fam: fam, depth: depth}
		if fam == "num" && depth == 1 {
			if m := reNum1.FindStringSubmatch(t); m != nil {
				r.num = m[1]
			}
		}
		raws = append(raws, r)
	}

	// 单级数字开头（「5 施工方案」）像不像章标题，看多级编号的首位数字认不认它。
	implied := map[string]bool{}
	for _, r := range raws {
		if r.fam == "num" && r.depth >= 2 {
			if i := strings.Index(r.c.Title, "."); i > 0 {
				implied[r.c.Title[:i]] = true
			}
		}
	}
	hasChapter, hasSection := false, false
	for _, r := range raws {
		hasChapter = hasChapter || r.fam == "chapter"
		hasSection = hasSection || r.fam == "section"
	}

	type frame struct {
		fam   string
		level int
	}
	var stack []frame
	clamp := func(n int) int { return max(1, min(MaxLevels, n)) }
	out := make([]HeadingCandidate, 0, len(raws))
	for _, r := range raws {
		if r.fam == "num" && r.depth == 1 && r.hashes == 0 {
			r.c.Weak = r.c.InTable || !(implied[r.num] || (len(implied) == 0 && r.num != "" && len(r.num) <= 2))
		}
		// 表格行里的章名只是旁证：给层级，但不参与上下文推算。
		if r.c.InTable {
			r.c.Level = clamp(r.depth)
			out = append(out, r.c)
			continue
		}
		structured := r.hashes > 0 || r.fam == "chapter" || r.fam == "section" || r.fam == "num"
		known := false // 沿用了已在上下文里的层级，不必再入栈
		var level int
		switch {
		case r.hashes > 0 && r.fam == "":
			level = clamp(r.hashes - atxMin + 1)
		case r.fam == "chapter":
			level = 1
		case r.fam == "section":
			level = 1
			if hasChapter {
				level = 2
			}
		case r.fam == "num":
			level = clamp(r.depth)
			if hasChapter && r.depth == 1 {
				level = 2
			}
		default:
			// 一、/（一）/1. 的深浅看上下文：同一形态沿用已推的层级，新形态落在当前层之下。
			level = -1
			for k := len(stack) - 1; k >= 0; k-- {
				if stack[k].fam == r.fam {
					level, stack, known = stack[k].level, stack[:k+1], true
					break
				}
			}
			if level < 0 {
				level = 1
				if n := len(stack); n > 0 {
					level = clamp(stack[n-1].level + 1)
				}
			}
		}
		r.c.Level = level
		out = append(out, r.c)
		if known || r.c.Weak {
			continue // 拿不准的行不参与推层级
		}
		if structured {
			for n := len(stack); n > 0 && stack[n-1].level >= level; n-- {
				stack = stack[:n-1]
			}
		}
		stack = append(stack, frame{fam: r.fam, level: level})
	}
	return out
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
