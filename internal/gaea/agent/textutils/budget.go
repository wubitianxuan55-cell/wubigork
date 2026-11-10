// Package textutils — D1 输出预算三件套（dsh-ops 蒸馏 2026-11-10 第二波）：
// 预算按**转义后 JSON 字节**计（`"`/`\`/换行 2B、控制符 6B——32MiB 文件曾炸
// 200MB stdout 的 NUL 6 倍乘数教训），信封序列化前的 payload 层截断——旧口径
// 在信封序列化后按原文字节截断，劈坏 JSON 且实际进上下文的字节可超帽。
package textutils

import (
	"fmt"
	"strings"
)

// escapeCost 表：gaea 信封 SetEscapeHTML(false)，只有 `"`、`\` 与 <0x20 被
// 转义（\b\f\n\r\t 两字符，其余控制符 \u00XX 六字符）。
var escapeCost = [256]int{}

func init() {
	for i := 0; i < 256; i++ {
		escapeCost[i] = 1
	}
	escapeCost['"'] = 2
	escapeCost['\\'] = 2
	escapeCost['\b'] = 2
	escapeCost['\f'] = 2
	escapeCost['\n'] = 2
	escapeCost['\r'] = 2
	escapeCost['\t'] = 2
	for i := 0; i < 0x20; i++ {
		if escapeCost[i] == 1 {
			escapeCost[i] = 6
		}
	}
}

// EscapeJSONLen 返回 s 作为 JSON 字符串字面量内容的精确转义后字节数
// （不含首尾引号；SetEscapeHTML(false) 口径，与 tool.mustMarshal 一致）。
func EscapeJSONLen(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n += escapeCost[s[i]]
	}
	return n
}

// TruncateToEscapedBudget 把文本截到「转义后字节数 ≤ maxEscapedBytes」。
// 迭代收敛：按原文字节选行→量转义长→超则按超出量缩预算重来（≤4 轮），
// 原文每次从原始串重新选（对已截断串再截会二次丢段）。控制符占比病态
// 的输入 4 轮后仍超则返回最后候选（调用方 backstop 兜底，不劣于现状）。
func TruncateToEscapedBudget(s string, maxLines, maxEscapedBytes int) (string, string) {
	const markerBudget = 176 // 截断标记+换行的最坏原文预留
	budget := maxEscapedBytes - markerBudget
	if budget < 1 {
		budget = 1
	}
	cand, notice := TruncateToolOutputWith(s, maxLines, budget)
	for i := 0; i < 4; i++ {
		if EscapeJSONLen(cand) <= maxEscapedBytes || budget <= 0 {
			return cand, notice
		}
		budget -= EscapeJSONLen(cand) - maxEscapedBytes + 64
		if budget < 1 {
			budget = 1
		}
		cand, notice = TruncateToolOutputWith(s, maxLines, budget)
	}
	return cand, notice
}

// TruncateToolOutputPaged 分页方向截断（D1）：保留**头部** maxLines-1 行+
// 尾行（续读标记），丢弃中段——分页读语义下「保尾砍头」会让模型按尾行
// offset 续页从而静默跳过被砍的中段（head+tail 旧口径的真缺陷）。
func TruncateToolOutputPaged(s string, maxLines, maxBytes int) (string, string) {
	s = Normalize(s)
	lines := strings.Split(s, "\n")
	originalLines := len(lines)
	originalBytes := len(s)
	if originalBytes <= maxBytes && originalLines <= maxLines {
		return s, ""
	}
	keep := maxLines - 1
	if keep < 1 {
		keep = 1
	}
	if keep > len(lines) {
		keep = len(lines)
	}
	selected := make([]string, 0, keep+1)
	selected = append(selected, lines[:keep]...)
	if originalLines > keep {
		selected = append(selected, lines[originalLines-1]) // 续读标记尾行必活
	}
	return emitSelected(selected, maxBytes, originalBytes, originalLines)
}

// emitSelected 是行选择的共用字节回路（TruncateToolOutputWith/Paged 共享；
// 标记文案保持旧口径防行为漂移）。
func emitSelected(selected []string, maxBytes, originalBytes, originalLines int) (string, string) {
	var out strings.Builder
	used := 0
	kept := 0
	for _, line := range selected {
		lineBytes := len(line) + 1
		if used+lineBytes > maxBytes {
			if kept > 0 {
				break
			}
			if maxBytes < len(line) {
				line = snapToRuneBoundary(line, 0, maxBytes)
			}
		}
		if kept > 0 {
			out.WriteString("\n")
			used++
		}
		out.WriteString(line)
		used += len(line)
		kept++
	}
	omitted := originalBytes - used
	omittedLines := originalLines - kept
	if omitted < 0 {
		omitted = 0
	}
	if omittedLines < 0 {
		omittedLines = 0
	}
	if omitted == 0 && omittedLines == 0 {
		return out.String(), ""
	}
	notice := fmt.Sprintf("tool output truncated: %d of %d bytes, %d of %d lines elided",
		omitted, originalBytes, omittedLines, originalLines)
	out.WriteString(fmt.Sprintf(
		"\n\n[%d byte(s), %d line(s) elided — use read_file with offset+limit, grep with narrower pattern, or bash with head/tail for details]",
		omitted, omittedLines))
	return out.String(), notice
}
