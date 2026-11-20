// Package textutils provides tool output truncation, normalization, and
// terminal-width helpers used by the agent package.
package textutils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// MaxToolOutputBytes caps a single tool result before it goes into the model's
// context window. 48 KiB keeps the overhead per call bounded while leaving room
// for meaningful output.
const MaxToolOutputBytes = 48 * 1024

const tailScanChars = 2048

const MaxToolResultLines = 320

var signalKeywords = []string{
	"error", "Error", "ERROR",
	"fatal", "Fatal", "FATAL",
	"panic", "PANIC",
	"exception", "Exception",
	"failed", "Failed",
	"timeout", "Timeout",
	"denied", "Denied",
	"cannot", "invalid",
}

// FirstLine returns the first line of s (up to, but not including, the first
// newline).
func FirstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// TruncateToolOutput is the convenience wrapper that uses the default limits.
func TruncateToolOutput(s string) (string, string) {
	return TruncateToolOutputWith(s, MaxToolResultLines, MaxToolOutputBytes)
}

// TruncateToolOutputWith keeps the most important lines up to the limits.
// 字节回路抽到 emitSelected（budget.go，D1 与分页方向共用）。
func TruncateToolOutputWith(s string, maxLines, maxBytes int) (string, string) {
	s = Normalize(s)

	lines := strings.Split(s, "\n")
	originalLines := len(lines)
	originalBytes := len(s)

	if originalBytes <= maxBytes && originalLines <= maxLines {
		return s, ""
	}

	direction := "head+tail"
	if hasErrorInTail(s, tailScanChars) {
		direction = "tail"
	}
	selected := selectHygieneLines(lines, maxLines, direction)
	policy := "头部+均匀采样+信号行+尾部"
	if direction == "tail" {
		policy = "尾部连续行+信号行"
	}
	return emitSelected(selected, maxBytes, originalBytes, originalLines, policy)
}

// Normalize strips ANSI escape sequences, normalises line endings, compresses
// blank-line runs and repeated consecutive lines.
func Normalize(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' && s[j] != 'K' && s[j] != 'h' && s[j] != 'l' {
				if (s[j] >= '0' && s[j] <= '9') || s[j] == ';' || s[j] == '?' {
					j++
				} else {
					break
				}
			}
			if j < len(s) {
				i = j + 1
				continue
			}
		}
		out.WriteByte(s[i])
		i++
	}

	lines := strings.Split(strings.ReplaceAll(out.String(), "\r\n", "\n"), "\n")
	var result []string
	blankRun := 0
	prev := ""
	repeatCount := 0

	flushRepeat := func() {
		if repeatCount > 1 {
			result = append(result, fmt.Sprintf("[previous line repeated %d time(s)]", repeatCount-1))
		}
		repeatCount = 0
	}

	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			flushRepeat()
			blankRun++
			if blankRun <= 2 {
				result = append(result, "")
			}
			prev = ""
			continue
		}
		blankRun = 0
		if line == prev {
			repeatCount++
			continue
		}
		flushRepeat()
		result = append(result, line)
		prev = line
		repeatCount = 1
	}
	flushRepeat()
	return strings.Join(result, "\n")
}

// HasSignalKeyword reports whether the line contains a signal keyword.
func HasSignalKeyword(line string) bool {
	lower := strings.ToLower(line)
	for _, kw := range signalKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func hasErrorInTail(s string, n int) bool {
	start := len(s) - n
	if start < 0 {
		start = 0
	}
	tail := strings.ToLower(s[start:])
	for _, kw := range signalKeywords {
		if strings.Contains(tail, kw) {
			return true
		}
	}
	return false
}

func selectHygieneLines(lines []string, maxLines int, direction string) []string {
	if len(lines) <= maxLines {
		return lines
	}

	indexes := make(map[int]bool)
	headCount := min(80, maxLines/4)
	// D3（2026-11-20 缺陷清单）：尾巴收窄（120→24），腾出的份额给中段均匀
	// 采样——纯头尾口径下 900 行实测只保 1–80+798–900，中段 82% 整段蒸发；
	// 同质内容白保、异质内容致命。错误尾方向（direction=tail）语义不变。
	tailCount := min(24, maxLines/6)
	if headCount < 1 {
		headCount = 1
	}
	if tailCount < 1 {
		tailCount = 1
	}

	if direction == "tail" {
		// Error detected in tail: skip head, keep only tail + signal.
		tailCount = min(120, maxLines/3)
		if tailCount < 1 {
			tailCount = 1
		}
	} else {
		for i := 0; i < headCount && i < len(lines); i++ {
			indexes[i] = true
		}
	}
	for i := len(lines) - tailCount; i < len(lines); i++ {
		if i >= 0 {
			indexes[i] = true
		}
	}

	signalCount := 0
	for i := 0; i < len(lines) && len(indexes) < maxLines; i++ {
		if HasSignalKeyword(lines[i]) && !indexes[i] {
			indexes[i] = true
			signalCount++
			if signalCount >= 48 {
				break
			}
		}
	}

	// D3：中段均匀采样——被丢弃的区间按剩余名额等距取样，模型对全文保有
	// 低分辨率视野（read_file 输出带原始行号，跳跃处自明；无行号输出靠下方
	// gap 标记）。
	if direction != "tail" && len(indexes) < maxLines-2 {
		lo := headCount
		hi := len(lines) - tailCount
		if hi > lo {
			want := maxLines - len(indexes) - 1
			if want > 0 {
				step := float64(hi-lo) / float64(want+1)
				for k := 1; k <= want; k++ {
					idx := lo + int(float64(k)*step)
					if idx >= 0 && idx < len(lines) {
						indexes[idx] = true
					}
					if len(indexes) >= maxLines {
						break
					}
				}
			}
		}
	}

	// 排序输出 + 采样 gap 标记：相邻选中行号差 >1 时插一行省略提示，防采样
	// 片段被误读为连续输出（gap 标记不占 maxLines 内容名额，总量设 8 行硬帽）。
	ordered := make([]int, 0, len(indexes))
	for i := 0; i < len(lines); i++ {
		if indexes[i] {
			ordered = append(ordered, i)
		}
	}
	result := make([]string, 0, len(ordered)+8)
	markers := 0
	prev := -2
	for _, i := range ordered {
		if i-prev > 1 && len(result) > 0 && markers < 8 {
			result = append(result, fmt.Sprintf("[... %d line(s) elided ...]", i-prev-1))
			markers++
		}
		result = append(result, lines[i])
		prev = i
	}
	return result
}

func snapToRuneBoundary(s string, lo, hi int) string {
	for lo > 0 && !utf8.RuneStart(s[lo]) {
		lo--
	}
	for hi < len(s) && !utf8.RuneStart(s[hi]) {
		hi++
	}
	return s[lo:hi]
}

// === Width helpers ===

// ansiSGR matches ANSI Select-Graphic-Rendition sequences.
var ansiSGR = regexp.MustCompile("\x1b\\[[0-9;]*m")

// VisibleWidth returns the column count of s after stripping ANSI SGR codes.
func VisibleWidth(s string) int {
	return runewidth.StringWidth(ansiSGR.ReplaceAllString(s, ""))
}

// StreamedRows counts how many rows the cursor has descended after raw text
// of length s was printed at the given terminal width.
func StreamedRows(s string, width int) int {
	if width <= 0 {
		width = 80
	}
	rows := 0
	for _, line := range strings.Split(s, "\n") {
		if w := VisibleWidth(line); w > 0 {
			rows += (w - 1) / width
		}
	}
	rows += strings.Count(s, "\n")
	return rows
}
