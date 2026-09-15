// Package rewrite 驱动式整章重写引擎（t4-C3）。
//
// 口径来源：docs/distill/04-plot-analysis.md §5.2（chapter_regenerator.py
// L110-L237 / prompt_service.py L2628-L2753）。纯规则零 IO：修改指令构建、
// 输出清理、diff 统计、请求归一。LLM 调用与版本落盘在 app/project 层。
package rewrite

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/textsim"
	"github.com/gaea/gaea/internal/types"
)

// focusMap 重点优化方向五选映射（MuMu chapter_regenerator.py:143-149，
// 未知 key 静默丢弃）。
var focusMap = map[string]string{
	"pacing":      "节奏把控 - 调整叙事速度，避免拖沓或过快",
	"emotion":     "情感渲染 - 深化人物情感表达，增强感染力",
	"description": "场景描写 - 丰富环境细节，增强画面感",
	"dialogue":    "对话质量 - 让对话更自然真实，推动剧情",
	"conflict":    "冲突强度 - 强化矛盾冲突，提升戏剧张力",
}

// 请求默认值与钳制（MuMu schemas/regeneration.py L15-L38）。
const (
	DefaultTargetWords = 3000
	MinTargetWords     = 500
	MaxTargetWords     = 10000
)

// NormalizeRequest 请求归一与前置校验（对齐 MuMu L4489-L4500 + schema 默认值）：
//   - source 为空 → custom；非法值报错；
//   - custom：自定义指令必填；analysis_suggestions：至少选 1 条建议索引；
//     mixed：两者至少其一；
//   - TargetWordCount 0 → 3000，钳制 [500,10000]；
//   - FocusAreas 未知 key 静默丢弃；
//   - Mode 为空 → whole；partial 分支（t4-C3 余项）恒 custom 来源：仅自定义
//     指令、长度模式缺省 similar、custom 档需目标字数、选区方向校验，细则
//     见 normalizePartial（规格 进度计划/gaea-partial-rewrite-20260916.md §1）。
func NormalizeRequest(req types.RewriteRequest) (types.RewriteRequest, error) {
	if req.Mode == "" {
		req.Mode = types.RewriteModeWhole
	}
	if req.Source == "" {
		req.Source = types.RewriteSourceCustom
	}
	// partial 分支（t4-C3 余项）：不走 whole 的建议/混合与字数缺省链路，
	// 细则见 normalizePartial；whole 既有路径零变化。
	if req.Mode == types.RewriteModePartial {
		return normalizePartial(req)
	}
	hasInstr := strings.TrimSpace(req.CustomInstructions) != ""
	hasIdx := len(req.SuggestionIdxs) > 0
	switch req.Source {
	case types.RewriteSourceCustom:
		if !hasInstr {
			return req, fmt.Errorf("自定义指令不能为空")
		}
	case types.RewriteSourceSuggestions:
		if !hasIdx {
			return req, fmt.Errorf("至少选择一条分析建议")
		}
	case types.RewriteSourceMixed:
		if !hasInstr && !hasIdx {
			return req, fmt.Errorf("自定义指令与分析建议至少提供其一")
		}
	default:
		return req, fmt.Errorf("未知重写来源 %q", req.Source)
	}
	if req.TargetWordCount <= 0 {
		req.TargetWordCount = DefaultTargetWords
	}
	if req.TargetWordCount < MinTargetWords {
		req.TargetWordCount = MinTargetWords
	}
	if req.TargetWordCount > MaxTargetWords {
		req.TargetWordCount = MaxTargetWords
	}
	kept := req.FocusAreas[:0:0]
	for _, f := range req.FocusAreas {
		if _, ok := focusMap[f]; ok {
			kept = append(kept, f)
		}
	}
	req.FocusAreas = kept
	return req, nil
}

// BuildModificationInstruction 修改指令构建（MuMu L110-L179，四段顺序固定）：
// 改进问题（仅选中的建议，越界静默跳过）→ 自定义要求 → 重点方向 → 保留元素。
func BuildModificationInstruction(req types.RewriteRequest, suggestions []string) string {
	var b strings.Builder
	b.WriteString("# 章节修改指令\n")

	if req.Source == types.RewriteSourceSuggestions || req.Source == types.RewriteSourceMixed {
		if picked := pickSuggestions(req.SuggestionIdxs, suggestions); len(picked) > 0 {
			b.WriteString("\n## 需要改进的问题（来自AI分析）：\n")
			for i, s := range picked {
				fmt.Fprintf(&b, "%d. %s\n", i+1, s)
			}
		}
	}
	if s := strings.TrimSpace(req.CustomInstructions); s != "" {
		b.WriteString("\n## 用户自定义修改要求：\n")
		b.WriteString(s + "\n")
	}
	if len(req.FocusAreas) > 0 {
		b.WriteString("\n## 重点优化方向：\n")
		for _, area := range req.FocusAreas {
			if desc, ok := focusMap[area]; ok {
				b.WriteString("- " + desc + "\n")
			}
		}
	}
	if p := req.Preserve; p != nil {
		var lines []string
		if p.Structure {
			lines = append(lines, "- 保持原章节的整体结构和情节框架")
		}
		if len(p.Dialogues) > 0 {
			lines = append(lines, "- 必须保留以下关键对话：")
			for _, d := range p.Dialogues {
				lines = append(lines, "  * "+d)
			}
		}
		if len(p.PlotPoints) > 0 {
			lines = append(lines, "- 必须保留以下关键情节点：")
			for _, pp := range p.PlotPoints {
				lines = append(lines, "  * "+pp)
			}
		}
		if p.CharacterTraits {
			lines = append(lines, "- 保持所有角色的性格特征和行为模式一致")
		}
		if len(lines) > 0 {
			b.WriteString("\n## 必须保留的元素：\n")
			b.WriteString(strings.Join(lines, "\n") + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// pickSuggestions 取选中索引的建议，越界/空文本静默跳过（MuMu L129）。
func pickSuggestions(idxs []int, suggestions []string) []string {
	out := make([]string, 0, len(idxs))
	for _, idx := range idxs {
		if idx < 0 || idx >= len(suggestions) {
			continue
		}
		if s := strings.TrimSpace(suggestions[idx]); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// rewriteOutputPrefixes 模型输出的常见前缀（MuMu L5131-L5151，命中即 break）。
var rewriteOutputPrefixes = []string{
	"重写后：", "重写后:", "改写后：", "改写后:",
	"以下是重写后的内容：", "以下是重写后的内容:", "重写内容：", "重写内容:",
}

// CleanRewriteOutput 输出清理：去首尾空白 → 移除常见前缀（命中即剥一次）→
// 成对包裹引号剥离（“ ” " " 「」 『』）。
func CleanRewriteOutput(s string) string {
	s = strings.TrimSpace(s)
	for _, p := range rewriteOutputPrefixes {
		if strings.HasPrefix(s, p) {
			s = strings.TrimPrefix(s, p)
			break
		}
	}
	s = strings.TrimSpace(s)
	for _, pair := range [][2]string{
		{"“", "”"}, {"\"", "\""}, {"「", "」"}, {"『", "』"},
	} {
		r := []rune(s)
		if len(r) >= 2 && string(r[0]) == pair[0] && string(r[len(r)-1]) == pair[1] {
			s = strings.TrimSpace(string(r[1 : len(r)-1]))
			break
		}
	}
	return s
}

// Diff 重写前后统计（MuMu L207-L237 的 gaea 等价物：difflib.SequenceMatcher
// 相似度以本地 textsim 的 CJK 二元组 Dice 系数替代——纯本地无依赖）。
type Diff struct {
	OriginalLen   int     `json:"originalLen"`
	NewLen        int     `json:"newLen"`
	Change        int     `json:"change"`        // 新-旧（rune，负=变短）
	ChangePercent float64 `json:"changePercent"` // 相对原文百分比
	Similarity    float64 `json:"similarity"`    // 0-100
}

// ComputeDiff 统计重写前后篇幅与相似度。
func ComputeDiff(original, rewritten string) Diff {
	ro, rn := []rune(original), []rune(rewritten)
	d := Diff{OriginalLen: len(ro), NewLen: len(rn), Change: len(rn) - len(ro)}
	if d.OriginalLen > 0 {
		d.ChangePercent = float64(d.Change) / float64(d.OriginalLen) * 100
	}
	d.Similarity = textsim.Similarity(original, rewritten) * 100
	return d
}
