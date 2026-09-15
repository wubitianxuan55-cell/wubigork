// partial 选段局部重写纯函数（t4-C3 余项）。
//
// 口径来源：docs/distill/04-plot-analysis.md §5.3（MuMu api/chapters.py
// L4959-L5100 的 gaea rune 版）：选段校验与 ±50 模糊重锚、长度模式四档、
// max_tokens 公式与四段指令构建。±500 上下文截取与选段拼接落盘在 handler 层。
//
// 命名说明：规格书（进度计划/gaea-partial-rewrite-20260916.md §1）将类型与
// 构造函数均写作 PartialLengthSpec，Go 包级类型与函数同名不可声明，故类型
// 改称 PartialSpec、构造函数保留 PartialLengthSpec——B 线调用点
// rewrite.PartialLengthSpec(...) 零改动。
package rewrite

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/types"
)

// 长度模式四档常量（MuMu schemas/chapter.py length_mode；similar 为缺省，
// 未知 mode 一律回 similar）。
const (
	LengthModeSimilar  = "similar"
	LengthModeExpand   = "expand"
	LengthModeCondense = "condense"
	LengthModeCustom   = "custom"
)

// partial 档位常量（MuMu L4971 / L5054-L5100）。
const (
	partialReanchorWindow = 50    // 模糊重锚 ±50 rune 窗口
	minPartialTargetWords = 50    // custom 档目标字数下钳
	maxPartialTargetWords = 10000 // custom 档目标字数上钳
	maxPartialInstrRunes  = 1000  // 用户指令 rune 截断上限
	partialMaxTokensFloor = 500   // max_tokens 下限
	partialMaxTokensCeil  = 8000  // max_tokens 上限
)

// ResolveSelection rune 口径选段校验与 ±50 模糊重锚（MuMu L4959-L4986 的
// rune 版）。start/end 为 rune 偏移；selected 为空时跳过重锚只做边界校验；
// content[start:end] 与 selected 不一致时在 [start-50, end+50) 窗口内重新
// 定位 selected（字节命中换算回 rune 偏移），窗口内仍不命中才报不匹配。
func ResolveSelection(content string, start, end int, selected string) (int, int, error) {
	runes := []rune(content)
	n := len(runes)
	if start < 0 || start >= n {
		return 0, 0, fmt.Errorf("起始位置超出内容范围")
	}
	if end > n {
		return 0, 0, fmt.Errorf("结束位置超出内容范围")
	}
	if start >= end {
		return 0, 0, fmt.Errorf("起始位置必须小于结束位置")
	}
	if selected == "" {
		return start, end, nil
	}
	if string(runes[start:end]) == selected {
		return start, end, nil
	}
	lo := start - partialReanchorWindow
	if lo < 0 {
		lo = 0
	}
	hi := end + partialReanchorWindow
	if hi > n {
		hi = n
	}
	searchArea := string(runes[lo:hi])
	byteIdx := strings.Index(searchArea, selected)
	if byteIdx < 0 {
		return 0, 0, fmt.Errorf("选中的文本与章节内容不匹配，请刷新后重试")
	}
	newStart := lo + len([]rune(searchArea[:byteIdx]))
	return newStart, newStart + len([]rune(selected)), nil
}

// PartialSpec 局部重写长度区间（rune）与提示文案（规格 §1 的
// PartialLengthSpec 类型，命名冲突说明见文件头）。
type PartialSpec struct {
	MinRunes int
	MaxRunes int
	Hint     string
}

// PartialLengthSpec 按长度模式换算选段字数区间（MuMu L5054-L5071）：
//   - similar（缺省，未知 mode 回退）：0.8×~1.2×；
//   - expand：1.2×~2.0×；condense：0.5×~0.8×；
//   - custom：targetWords 钳 [50,10000] 后 ±20%。
func PartialLengthSpec(mode string, selRunes, targetWords int) PartialSpec {
	switch mode {
	case LengthModeExpand:
		minR, maxR := int(float64(selRunes)*1.2), int(float64(selRunes)*2.0)
		return PartialSpec{
			MinRunes: minR,
			MaxRunes: maxR,
			Hint:     fmt.Sprintf("适当扩展内容（目标 %d-%d 字）", minR, maxR),
		}
	case LengthModeCondense:
		minR, maxR := int(float64(selRunes)*0.5), int(float64(selRunes)*0.8)
		return PartialSpec{
			MinRunes: minR,
			MaxRunes: maxR,
			Hint:     fmt.Sprintf("精简压缩内容（目标 %d-%d 字）", minR, maxR),
		}
	case LengthModeCustom:
		t := targetWords
		if t < minPartialTargetWords {
			t = minPartialTargetWords
		}
		if t > maxPartialTargetWords {
			t = maxPartialTargetWords
		}
		return PartialSpec{
			MinRunes: int(float64(t) * 0.8),
			MaxRunes: int(float64(t) * 1.2),
			Hint:     fmt.Sprintf("目标字数：约 %d 字（允许±20%%浮动）", t),
		}
	default: // similar 及未知 mode
		minR, maxR := int(float64(selRunes)*0.8), int(float64(selRunes)*1.2)
		return PartialSpec{
			MinRunes: minR,
			MaxRunes: maxR,
			Hint: fmt.Sprintf("保持与原文相近的字数（约 %d 字，允许 %d-%d 字浮动）",
				selRunes, minR, maxR),
		}
	}
}

// PartialMaxTokens 局部重写 max_tokens（MuMu L5100：中文 1 字 ≈ 3 token）：
// max(500, min(MaxRunes*3, 8000))。
func PartialMaxTokens(spec PartialSpec) int {
	tokens := spec.MaxRunes * 3
	if tokens > partialMaxTokensCeil {
		tokens = partialMaxTokensCeil
	}
	if tokens < partialMaxTokensFloor {
		tokens = partialMaxTokensFloor
	}
	return tokens
}

// BuildPartialInstruction 局部重写指令（四段固定顺序）：
// ①上下文（ctxBefore/ctxAfter 为调用方截好的 ±500 rune 片段，空白视为空
// → 章节开头/结尾占位；注明仅参考不得改动）②选中文本（只重写选段，选段
// 外一字不动）③用户指令（trim，>1000 rune 截断）④长度模式提示。
func BuildPartialInstruction(instr, selectedText, ctxBefore, ctxAfter string, spec PartialSpec) string {
	before, after := ctxBefore, ctxAfter
	if strings.TrimSpace(before) == "" {
		before = "（这是章节开头）"
	}
	if strings.TrimSpace(after) == "" {
		after = "（这是章节结尾）"
	}
	trimmed := strings.TrimSpace(instr)
	if r := []rune(trimmed); len(r) > maxPartialInstrRunes {
		trimmed = string(r[:maxPartialInstrRunes])
	}
	var b strings.Builder
	b.WriteString("# 局部重写指令\n")
	b.WriteString("\n## 上下文（仅参考，不得改动）\n")
	b.WriteString("【选段前文】\n" + before + "\n")
	b.WriteString("【选段后文】\n" + after + "\n")
	b.WriteString("\n## 选中文本（只重写以下【选段】，选段外一字不动）\n")
	b.WriteString(selectedText + "\n")
	b.WriteString("\n## 用户指令\n")
	b.WriteString(trimmed + "\n")
	b.WriteString("\n## 长度模式提示\n")
	b.WriteString(spec.Hint + "\n")
	return strings.TrimRight(b.String(), "\n")
}

// normalizePartial partial 请求归一（规格 进度计划/gaea-partial-rewrite-20260916.md
// §1；Source 空已在 NormalizeRequest 归 custom）：
//   - Source 非 custom → 局部重写仅支持自定义指令；
//   - CustomInstructions trim 空 → 重写指令不能为空；
//   - LengthMode 空 → similar；
//   - custom 档 TargetWordCount<=0 → 自定义长度需提供目标字数；
//   - StartPos<0 或 EndPos<=StartPos → 选区非法（内容长度校验在 handler）。
//
// 不套 whole 的字数缺省 3000/钳制：目标字数仅 custom 档有意义，区间换算
// 由 PartialLengthSpec（target 钳 [50,10000]）负责。
func normalizePartial(req types.RewriteRequest) (types.RewriteRequest, error) {
	if req.Source != types.RewriteSourceCustom {
		return req, fmt.Errorf("局部重写仅支持自定义指令")
	}
	if strings.TrimSpace(req.CustomInstructions) == "" {
		return req, fmt.Errorf("重写指令不能为空")
	}
	if req.LengthMode == "" {
		req.LengthMode = LengthModeSimilar
	}
	if req.LengthMode == LengthModeCustom && req.TargetWordCount <= 0 {
		return req, fmt.Errorf("自定义长度需提供目标字数")
	}
	if req.StartPos < 0 || req.EndPos <= req.StartPos {
		return req, fmt.Errorf("选区非法")
	}
	return req, nil
}
