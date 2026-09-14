package analysis

import (
	"strings"
	"unicode"

	"github.com/gaea/gaea/internal/types"
)

// ── 分析标注层（t4-C4 首刀：keyword → 正文 rune 偏移）──────────────
//
// 口径来源：docs/distill/04-plot-analysis.md §3.10/§8.4 A1/A2。
// 关键修正（相对 MuMu）：① 去标点匹配必须做「坐标反投影」（indexMap 记录
// 被删字符的原始 rune 索引，MuMu 缺失导致坐标偏移）；② keyword 定位失败的
// 锚定条目**丢弃标注**而非留 -1（-1 会被下游当无位置触发错位回查；
// MuMu L746 对策）。suggestion 类天然无锚点，Pos=-1 仅列条目不高亮（契约语义）。

// stripPunctWithMap 去标点/空白归一化：返回干净 rune 串与反投影索引表——
// clean 的第 j 个 rune 对应原文的第 indexMap[j] 个 rune。
func stripPunctWithMap(s string) (string, []int) {
	r := []rune(s)
	var clean strings.Builder
	indexMap := make([]int, 0, len(r))
	for i, c := range r {
		if unicode.IsPunct(c) || unicode.IsSpace(c) {
			continue
		}
		clean.WriteRune(c)
		indexMap = append(indexMap, i)
	}
	return clean.String(), indexMap
}

// indexRuneSub 在主 rune 串中找子串（rune 级朴素匹配，正文量级足够）。
func indexRuneSub(full, sub []rune) int {
	if len(sub) == 0 || len(sub) > len(full) {
		return -1
	}
	for i := 0; i+len(sub) <= len(full); i++ {
		match := true
		for j := range sub {
			if full[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// stripPunctAll 去标点/空白（无反投影需求侧）。
func stripPunctAll(s string) string {
	c, _ := stripPunctWithMap(s)
	return c
}

// LocateKeyword 正文定位（§8.4 A2 四段式）：精确 rune 匹配 → 去标点匹配
// （反投影回原文坐标）→ 长 keyword 前 15 rune 前缀匹配 → 未命中 (-1,0)。
func LocateKeyword(full, kw string) (int, int) {
	if full == "" || strings.TrimSpace(kw) == "" {
		return -1, 0
	}
	fr := []rune(full)
	kr := []rune(strings.TrimSpace(kw))

	// ① 精确匹配（坐标天然正确）
	if i := indexRuneSub(fr, kr); i >= 0 {
		return i, len(kr)
	}
	// ② 去标点匹配 + 坐标反投影（MuMu 缺 indexMap 的坐标缺陷修正点）
	clean, indexMap := stripPunctWithMap(full)
	ck := []rune(stripPunctAll(kw))
	if len(ck) > 0 {
		if j := indexRuneSub([]rune(clean), ck); j >= 0 {
			start := indexMap[j]
			endIdx := j + len(ck) - 1
			if endIdx >= len(indexMap) {
				endIdx = len(indexMap) - 1
			}
			return start, indexMap[endIdx] - start + 1
		}
	}
	// ③ 长 keyword 前 15 rune 前缀（模型尾部改写容错）
	if len(kr) > 10 {
		prefix := kr[:15]
		if i := indexRuneSub(fr, prefix); i >= 0 {
			return i, 15
		}
	}
	return -1, 0
}

// importanceOf strength(1-10) → 0-1（缺省语义 5，与记忆规则表同口径）。
func importanceOf(strength int) float64 {
	if strength <= 0 {
		strength = 5
	}
	if strength > 10 {
		strength = 10
	}
	return float64(strength) / 10.0
}

// BuildAnnotations 从 V2 分析载荷构建标注（纯函数，确定性输出）。
//
// 锚定类（hook/foreshadow/plot_point）必须定位成功，失败即丢弃该条（规格
// 对 MuMu「留 -1 错位回查」缺陷的修正）；hook 的 Content 是原文摘录可作
// 兜底锚点，foreshadow/plot_point 只认 keyword（描述是转述不可定位）；
// suggestion 类无锚点，Pos=-1 仅列条目不高亮（契约语义）。
func BuildAnnotations(content string, v2 *types.AnalysisResultV2) []types.Annotation {
	if v2 == nil {
		return nil
	}
	out := make([]types.Annotation, 0, 8)
	addAnchored := func(typ, title, contentText, anchor string, importance float64, tags []string) {
		contentText = strings.TrimSpace(contentText)
		if contentText == "" {
			return
		}
		pos, length := LocateKeyword(content, anchor)
		if pos < 0 {
			return
		}
		out = append(out, types.Annotation{
			Type: typ, Title: title, Content: contentText,
			Importance: importance, Pos: pos, Length: length, Tags: tags,
		})
	}

	for _, h := range v2.Hooks {
		anchor := strings.TrimSpace(h.Keyword.Text)
		if anchor == "" {
			anchor = strings.TrimSpace(h.Content) // 钩子 Content=原文摘录，可兜底定位
		}
		addAnchored("hook", strings.TrimSpace(h.Type), h.Content, anchor, importanceOf(h.Strength), nil)
	}
	for _, f := range v2.Foreshadows {
		addAnchored("foreshadow", strings.TrimSpace(f.Title), f.Content,
			strings.TrimSpace(f.Keyword.Text), importanceOf(f.Strength), []string{f.Type})
	}
	for _, p := range v2.PlotPoints {
		addAnchored("plot_point", "", p.Content,
			strings.TrimSpace(p.Keyword.Text), p.Importance, nil)
	}
	for _, s := range v2.Suggestions {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, types.Annotation{Type: "suggestion", Content: t, Pos: -1})
		}
	}
	return out
}
