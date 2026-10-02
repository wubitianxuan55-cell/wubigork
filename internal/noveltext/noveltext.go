// Package noveltext 小说正文的结构切分工具（段/句两级粒度，纯函数、零 LLM、零 IO）。
//
// 单源背景（审计 IN1-04）：novelgate（写后质量闸）与 novelreview（平台评审引擎）是
// 两套互不依赖的确定性体检，正文切分此前各有一份实现（review 的 splitParagraphs、
// gate 内联的 strings.Split(body, "\n") + 各自的切句逻辑），抽到本小包做单源，避免
// 切分口径分叉漂移：
//   - SplitParagraphs 段级粒度：novelreview 全部维度 + novelgate 的段落堆叠检查共用；
//   - SplitSentences  句级粒度：novelgate 的句长节奏（电报体）判据使用。
//
// 本包只管切分，不做任何质量判定——两套判据的阈值各自独立且**有意不同**
// （gate=句级阻断信号，review=段级建议信号），见两包内互引注释，勿在本包加判据。
package noveltext

import (
	"regexp"
	"strings"
)

// Paragraph 段落：同一份切分同时给出两种定位口径（审计 IN1-04 显式化）。
//
//   - Idx  段落序号（1-based，只数非空段）——novelreview 的证据定位口径（前端按段高亮）；
//   - Line 源行号（1-based，空行也占号）——novelgate「第 N 行」证据口径；
//
// 此前两套实现各算各的（review 数非空段、gate 数所有行），同一段落在两边编号不同
// 且无人写明——现由一次切分同时给出，口径差异留在字段名上，不再藏在实现里。
// Start/End 是 rune 区间 [Start,End)，相对传入的 runes。
type Paragraph struct {
	Idx   int
	Line  int
	Start int
	End   int
	Text  string // 去首尾空白后的段文（非空）
}

// SplitParagraphs 按换行切段：空段丢弃不占 Idx，但其行号语义保留（空行也占 Line 号）。
// 与被替换的 novelreview.splitParagraphs 切分结果逐字段一致（Idx/Start/End/Text），
// Line 为新增口径（novelgate 段落堆叠检查的证据行号）。
func SplitParagraphs(runes []rune) []Paragraph {
	var out []Paragraph
	start := 0
	line := 1 // 当前段所在的 1-based 源行号（每个 \n 后 +1）
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
		out = append(out, Paragraph{Idx: idx, Line: line, Start: start, End: end, Text: text})
	}
	for i, r := range runes {
		if r == '\n' {
			flush(i)
			start = i + 1
			line++
		}
	}
	flush(len(runes))
	return out
}

// sentenceSplitRe 句末标点（连续算一个分隔）。
var sentenceSplitRe = regexp.MustCompile(`[。！？!?…]+`)

// SplitSentences 按句末标点切句并去空白（自 novelgate.splitSentences 原样迁入，
// 切分结果逐句一致）。切出的句文不含句末标点，rune 计数即各消费方的「句长」口径。
func SplitSentences(text string) []string {
	parts := sentenceSplitRe.Split(text, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
