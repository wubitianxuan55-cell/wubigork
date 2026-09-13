// Package bookimport 拆书/导入反推的**纯规则解析层**（零 AI、零 IO 依赖，
// 便于单测与复用）。
//
// 规格来源：docs/distill/02-book-import.md §8.1「阶段 P0 · 解析引擎升级」，
// 算法逐条对齐 MuMu 的 txt_parser_service.py（file:line 见各函数注释）；
// 与 MuMu 的两处**有意偏离**已在对应函数处注明（宁留勿删 / 弃用字节计数）。
package bookimport

import (
	"bytes"
	"regexp"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

// ParsedChapter 解析出的原始章节（未编号、未裁剪）。
type ParsedChapter struct {
	Title   string
	Content string
}

// Warning 解析告警（对齐 MuMu 的 warnings 机制，规格 §8.1 第 8 条）。
type Warning struct {
	Code    string `json:"code"`    // chapter_too_short | chapter_too_long | duplicate_chapter_title | trimmed_for_extract_mode
	Message string `json:"message"` // 中文说明（直接可展示）
	Level   string `json:"level"`   // info | warning
}

// 告警码（与规格 §8.1 第 8 条逐字对齐）。
const (
	WarningChapterTooShort = "chapter_too_short"
	WarningChapterTooLong  = "chapter_too_long"
	WarningDuplicateTitle  = "duplicate_chapter_title"
	WarningTrimmedForMode  = "trimmed_for_extract_mode"
)

// 切分策略（报告字段，便于前端/日志显示与问题定位）。
const (
	StrategyStrong = "strong" // 强标题（第X章 / chapter N / 序章… / markdown #）
	StrategyWeak   = "weak"   // 弱标题（≤25 字、无标点、前后空行）
	StrategyWindow = "window" // 兜底窗口切分（3000~5000 字，标点边界）
	StrategySingle = "single" // 单章「全文」（无标题且长度不足窗口阈值）
	StrategyEPUB   = "epub"   // EPUB（按 spine 顺序，不走文本切分）
)

// ExtractMode 提取范围模式（对齐 MuMu 的 extract_mode；gaea 目前只走 full，
// tail 由后续「导入向导」接线）。
type ExtractMode string

const (
	ExtractFull ExtractMode = "full"
	ExtractTail ExtractMode = "tail"
)

// ParseOptions 解析选项。
type ParseOptions struct {
	// ExtractMode 提取范围：full（默认，空值即 full）| tail（只取末尾 N 章）。
	ExtractMode ExtractMode
	// TailChapterCount tail 模式下取的末尾章数；**必须为 5 的倍数**（非倍数向上取整），
	// >50 自动降级为 full（对齐规格 §8.1 第 7 条）。
	TailChapterCount int
}

// Report 解析报告。
type Report struct {
	Encoding         string    `json:"encoding"`         // 实际采用的编码名
	TotalChapters    int       `json:"totalChapters"`    // 切分出的总章数
	SelectedChapters int       `json:"selectedChapters"` // 裁剪后章数（full 模式与总章数相同）
	SplitStrategy    string    `json:"splitStrategy"`    // strong | weak | window | single | epub
	Warnings         []Warning `json:"warnings"`
}

// Result 解析结果（章节 + 报告）。
type Result struct {
	Chapters []ParsedChapter
	Report   Report
}

// 阈值（对齐规格 §8.1）。
const (
	minWindowRunes    = 3000 // 兜底窗口下界
	maxWindowRunes    = 5000 // 兜底窗口上界
	prefaceMinRunes   = 200  // 前置内容成「前言」章的门槛
	titleMaxRunes     = 200  // 标题截断上限
	weakTitleMaxRunes = 25   // 弱标题长度上限
	shortChapterRunes = 300  // 过短告警线
	longChapterRunes  = 12000 // 过长告知线
	tailMaxChapters   = 50   // tail 模式上限（超过降级 full）
	tailStep          = 5    // tail 章数步长
)

var (
	// strongHeadingRe 强标题：规格 §8.1 第 3 条 + **保留 gaea 更全的既有集合**
	// （序章/楔子/引子/前言/序言/尾声/后记/番外/外传/终章/大结局）与 markdown 标题。
	strongHeadingRe = regexp.MustCompile(`(?i)^\s*(?:第\s*[0-9０-９一二三四五六七八九十百千万零〇两]+\s*[章节回卷集部篇]` +
		`|chapter\s*\d+|chap\.\s*\d+|序章|楔子|引子|前言|序言|尾声|后记|番外|外传|终章|大结局)`)
	markdownHeadingRe = regexp.MustCompile(`^\s*#{1,6}\s+\S`)
	markdownStripRe   = regexp.MustCompile(`^\s*#{1,6}\s*`)
	// weakPunctuationRe 弱标题禁含的标点（规格 §8.1 第 4 条）。
	weakPunctuationRe = regexp.MustCompile(`[，。！？；：,.!?;:]`)
	trailingSpaceRe   = regexp.MustCompile(`[ \t]+\n`)
	manyNewlinesRe    = regexp.MustCompile(`\n{4,}`)
)

// ── 编码链（规格 §8.1 第 1 条 / §3.1）─────────────────────────

// Decode 解码字节流，返回正文与**实际采用的编码名**。
//
// 顺序：UTF-8（含 utf-8-sig 识别）→ GB18030 → GBK → Big5 → UTF-8 忽略非法字节。
// 与 MuMu（txt_parser_service.py:28-37）的两点差异：
//  1. **候选解码结果含替换符（U+FFFD）即判为误判并继续下探**——实测 GB18030/GBK
//     解码 Big5 字节不会返回 error，而是产出「材�彻…」式乱码（MuMu 会静默接受）；
//  2. 额外识别 UTF-16 LE/BE BOM（Windows 记事本另存常见；否则会被兜底路径打碎）。
func Decode(raw []byte) (string, string) {
	if out, name, ok := decodeUTF16BOM(raw); ok {
		return out, name
	}
	if utf8.Valid(raw) {
		if bytes.HasPrefix(raw, []byte{0xEF, 0xBB, 0xBF}) {
			return string(raw[3:]), "utf-8-sig"
		}
		return string(raw), "utf-8"
	}
	for _, cand := range []struct {
		name string
		dec  *encoding.Decoder
	}{
		{"gb18030", simplifiedchinese.GB18030.NewDecoder()},
		{"gbk", simplifiedchinese.GBK.NewDecoder()},
		{"big5", traditionalchinese.Big5.NewDecoder()},
	} {
		out, err := cand.dec.Bytes(raw)
		if err != nil || !utf8.Valid(out) {
			continue
		}
		if strings.ContainsRune(string(out), utf8.RuneError) {
			continue // 误判信号（见函数注释）
		}
		return string(out), cand.name
	}
	return strings.ToValidUTF8(string(raw), ""), "utf-8(ignore)"
}

// decodeUTF16BOM 处理 UTF-16 BOM（LE/BE）。
func decodeUTF16BOM(raw []byte) (string, string, bool) {
	if len(raw) < 2 {
		return "", "", false
	}
	var bigEndian bool
	switch {
	case raw[0] == 0xFF && raw[1] == 0xFE:
		bigEndian = false
	case raw[0] == 0xFE && raw[1] == 0xFF:
		bigEndian = true
	default:
		return "", "", false
	}
	body := raw[2:]
	units := make([]uint16, 0, len(body)/2)
	for i := 0; i+1 < len(body); i += 2 {
		if bigEndian {
			units = append(units, uint16(body[i])<<8|uint16(body[i+1]))
		} else {
			units = append(units, uint16(body[i+1])<<8|uint16(body[i]))
		}
	}
	name := "utf-16le"
	if bigEndian {
		name = "utf-16be"
	}
	return string(utf16.Decode(units)), name, true
}

// Clean 文本清洗（顺序敏感，逐条对齐规格 §8.1 第 2 条）。
func Clean(text string) string {
	s := strings.ReplaceAll(text, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\uFEFF", "")
	s = strings.ReplaceAll(s, "\u3000", "  ") // 全角空格 → 两个半角空格（不是删除）
	s = trailingSpaceRe.ReplaceAllString(s, "\n")
	s = manyNewlinesRe.ReplaceAllString(s, "\n\n\n")
	return strings.TrimSpace(s)
}

// ── 章节切分（规格 §8.1 第 3~6 条 / §3.2）──────────────────────

// Split 三级切分：强标题 → 弱标题 → 兜底窗口 → 单章全文。
// 返回切分结果与**命中的策略**（strong/weak/window/single）。
func Split(text string) ([]ParsedChapter, string) {
	cleaned := Clean(text)
	if strings.TrimSpace(cleaned) == "" {
		return nil, StrategySingle
	}
	lines := strings.Split(cleaned, "\n")
	// 分级判定：**强标题存在时不再叠加弱标题**（对齐 MuMu 的 strong-first 语义）——
	// 否则正文里的短行（如「内容甲」）会被当标题切碎。
	var strongIdx, weakIdx []int
	for i, line := range lines {
		if isStrongHeading(line) {
			strongIdx = append(strongIdx, i)
			continue
		}
		if isWeakHeading(lines, i) {
			weakIdx = append(weakIdx, i)
		}
	}
	if len(strongIdx) > 0 {
		return assembleChapters(lines, strongIdx), StrategyStrong
	}
	// 弱标题需**≥2 个候选**才算分章信号：单候选时整篇短文本会被误当「一章标题」，
	// gaea 口径退回「全文」单章（与既有行为一致）。
	if len(weakIdx) >= 2 {
		return assembleChapters(lines, weakIdx), StrategyWeak
	}
	if utf8.RuneCountInString(cleaned) > maxWindowRunes {
		return windowSplit(cleaned), StrategyWindow
	}
	return []ParsedChapter{{Title: "全文", Content: cleaned}}, StrategySingle
}

// Parse 解码 → 清洗 → 切分 → 裁剪 → 组装报告（对外主入口）。
func Parse(raw []byte, opts ParseOptions) Result {
	text, encName := Decode(raw)
	chapters, strategy := Split(text)
	total := len(chapters)
	selected, trimWarnings := applyExtractMode(chapters, opts)
	report := Report{
		Encoding:         encName,
		TotalChapters:    total,
		SelectedChapters: len(selected),
		SplitStrategy:    strategy,
		Warnings:         append(trimWarnings, buildWarnings(selected)...),
	}
	return Result{Chapters: selected, Report: report}
}

// isStrongHeading 强标题判定（行首锚定；markdown `#` 视同强标题）。
func isStrongHeading(line string) bool {
	return strongHeadingRe.MatchString(line) || markdownHeadingRe.MatchString(line)
}

// isWeakHeading 弱标题判定（规格 §8.1 第 4 条四条件与）：
// rune 长度 ≤25 ∧ 不含标点 ∧ 前行为空（或首行）∧ 后行为空（或末行）。
func isWeakHeading(lines []string, i int) bool {
	line := strings.TrimSpace(lines[i])
	if line == "" || markdownHeadingRe.MatchString(line) {
		return false
	}
	if utf8.RuneCountInString(line) > weakTitleMaxRunes || weakPunctuationRe.MatchString(line) {
		return false
	}
	if i > 0 && strings.TrimSpace(lines[i-1]) != "" {
		return false
	}
	if i < len(lines)-1 && strings.TrimSpace(lines[i+1]) != "" {
		return false
	}
	return true
}

// assembleChapters 按标题切出行块并组装章节（规格 §8.1 第 6 条）。
//
// 与 MuMu 的差异：首标题前正文不足 200 字时**并入首章**而不是丢弃
// （书名/作者/引子常在 2 行内，丢弃即丢信息——gaea 口径宁留勿删）。
func assembleChapters(lines []string, headingIdx []int) []ParsedChapter {
	var out []ParsedChapter
	preamble := strings.TrimSpace(strings.Join(lines[:headingIdx[0]], "\n"))
	prefaceKept := false
	if utf8.RuneCountInString(preamble) >= prefaceMinRunes {
		out = append(out, ParsedChapter{Title: "前言", Content: preamble})
		prefaceKept = true
	}
	for k, hi := range headingIdx {
		end := len(lines)
		if k+1 < len(headingIdx) {
			end = headingIdx[k+1]
		}
		title := cleanTitle(lines[hi])
		content := strings.TrimSpace(strings.Join(lines[hi+1:end], "\n"))
		if k == 0 && !prefaceKept && preamble != "" {
			if content == "" {
				content = preamble
			} else {
				content = preamble + "\n\n" + content
			}
		}
		if title == "" && content == "" {
			continue
		}
		out = append(out, ParsedChapter{Title: title, Content: content})
	}
	return out
}

// cleanTitle 标题清洗：去 markdown 前缀、TrimSpace、按 rune 截断 200 字。
func cleanTitle(line string) string {
	t := strings.TrimSpace(markdownStripRe.ReplaceAllString(strings.TrimSpace(line), ""))
	if t == "" {
		t = strings.TrimSpace(line)
	}
	return truncateRunes(t, titleMaxRunes)
}

// windowSplit 兜底窗口切分（规格 §8.1 第 5 条 / §3.2）：
// 在 [start+minWindow, start+maxWindow] 内取**最靠后的句子边界**，找不到才硬切。
func windowSplit(text string) []ParsedChapter {
	runes := []rune(text)
	n := len(runes)
	var out []ParsedChapter
	for start, no := 0, 1; start < n; no++ {
		idealEnd := start + maxWindowRunes
		if idealEnd > n {
			idealEnd = n
		}
		end := idealEnd
		if idealEnd < n {
			searchFrom := start + minWindowRunes
			if searchFrom > n {
				searchFrom = n
			}
			best := -1
			for i := searchFrom; i < idealEnd; i++ {
				if isSentenceBoundary(runes[i]) {
					best = i
				}
			}
			if best >= 0 {
				end = best + 1
			}
		}
		body := strings.TrimSpace(string(runes[start:end]))
		if body != "" {
			out = append(out, ParsedChapter{Title: "第" + itoa(no) + "章", Content: body})
		}
		start = end
	}
	return out
}

// isSentenceBoundary 窗口切分的边界字符（规格 §3.2：。！？!?\n）。
func isSentenceBoundary(r rune) bool {
	switch r {
	case '。', '！', '？', '!', '?', '\n':
		return true
	}
	return false
}

// ── 提取范围裁剪（规格 §8.1 第 7 条）───────────────────────────

// applyExtractMode 按选项裁剪末尾 N 章；返回裁剪结果与告警。
// tail 章数按 5 的倍数向上取整；>50 或非法值一律降级 full（不静默乱裁）。
func applyExtractMode(chapters []ParsedChapter, opts ParseOptions) ([]ParsedChapter, []Warning) {
	if opts.ExtractMode != ExtractTail || len(chapters) == 0 {
		return chapters, nil
	}
	want := opts.TailChapterCount
	if want <= 0 {
		return chapters, nil
	}
	if want%tailStep != 0 {
		want = (want/tailStep + 1) * tailStep
	}
	if want > tailMaxChapters || want >= len(chapters) {
		return chapters, nil
	}
	kept := chapters[len(chapters)-want:]
	w := Warning{
		Code:    WarningTrimmedForMode,
		Message: "按提取范围只保留了末尾 " + itoa(len(kept)) + " 章（共 " + itoa(len(chapters)) + " 章）",
		Level:   "info",
	}
	return kept, []Warning{w}
}

// ── 告警（规格 §8.1 第 8 条）───────────────────────────────────

// buildWarnings 组装章节级告警：过短 / 过长 / 标题重复。
func buildWarnings(chapters []ParsedChapter) []Warning {
	var out []Warning
	titleCount := map[string]int{}
	for _, ch := range chapters {
		if t := strings.TrimSpace(ch.Title); t != "" {
			titleCount[t]++
		}
		n := utf8.RuneCountInString(strings.TrimSpace(ch.Content))
		switch {
		case n > 0 && n < shortChapterRunes:
			out = append(out, Warning{
				Code:    WarningChapterTooShort,
				Message: "「" + displayTitle(ch.Title) + "」仅 " + itoa(n) + " 字，可能切分有误",
				Level:   "warning",
			})
		case n > longChapterRunes:
			out = append(out, Warning{
				Code:    WarningChapterTooLong,
				Message: "「" + displayTitle(ch.Title) + "」达 " + itoa(n) + " 字，可能未切分",
				Level:   "info",
			})
		}
	}
	for _, ch := range chapters {
		t := strings.TrimSpace(ch.Title)
		if t != "" && titleCount[t] > 1 {
			out = append(out, Warning{
				Code:    WarningDuplicateTitle,
				Message: "章节标题「" + truncateRunes(t, 40) + "」重复出现 " + itoa(titleCount[t]) + " 次",
				Level:   "warning",
			})
			titleCount[t] = 1 // 只报一次
		}
	}
	return out
}

func displayTitle(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return "未命名章"
	}
	return truncateRunes(t, 40)
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// itoa 小整数转十进制（避免为几处格式化引入 strconv 依赖噪音）。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
