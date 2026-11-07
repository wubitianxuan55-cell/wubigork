package pantheon

import (
	"regexp"
	"strings"
)

// 第二次蒸馏 E4：席位陈词末尾的 NEEDS INPUT 解析。
//
// 移植自 OpenMausBot runtime-coordinator.ts 的 needsInputQuestion，并加了
// 两条本地硬化：
//   - 代码块/引用里的 NEEDS INPUT 不触发（那是引用提示词，不是提问）；
//   - 问题与 Default: 拆开，Default 只作为问题卡上的一键选项，不自动回答。
//
// 只认最后一行（或 Default: 的上一行）——正文中间提到 NEEDS INPUT: 不触发。
type seatQuestion struct {
	Question string
	Default  string
}

var defaultLinePattern = regexp.MustCompile(`(?i)\bdefault:\s*([^\r\n]+)`)

func seatNeedsInputQuestion(text string) (seatQuestion, bool) {
	original := strings.Split(text, "\n")
	lastOriginal := lastNonEmptyLine(original, len(original)-1)
	if lastOriginal < 0 {
		return seatQuestion{}, false
	}
	surface := maskSeatQuestionSurface(text)
	lines := strings.Split(surface, "\n")
	// 如果全文最后一条非空行本身是围栏/引用（掩码后为空），说明消息以代码块
	// 或引用收尾，不是以提问收尾——不触发。
	if strings.TrimSpace(lines[lastOriginal]) == "" {
		return seatQuestion{}, false
	}
	last := lastNonEmptyLine(lines, len(lines)-1)
	if last < 0 {
		return seatQuestion{}, false
	}
	questionAt := last
	if isDefaultLine(lines[last]) {
		questionAt = lastNonEmptyLine(lines, last-1)
	}
	if questionAt < 0 {
		return seatQuestion{}, false
	}
	line := strings.TrimSpace(lines[questionAt])
	lower := strings.ToLower(line)
	if !strings.HasPrefix(lower, "needs input:") {
		return seatQuestion{}, false
	}
	body := strings.Join(append([]string{line[len("needs input:"):]}, lines[questionAt+1:]...), "\n")
	body = strings.TrimSpace(body)
	if body == "" {
		return seatQuestion{}, false
	}
	question := body
	defaultValue := ""
	if match := defaultLinePattern.FindStringSubmatchIndex(body); match != nil {
		question = strings.TrimSpace(body[:match[0]])
		defaultValue = strings.TrimSpace(body[match[2]:match[3]])
	}
	if question == "" {
		return seatQuestion{}, false
	}
	return seatQuestion{Question: question, Default: defaultValue}, true
}

func lastNonEmptyLine(lines []string, from int) int {
	for index := from; index >= 0; index-- {
		if strings.TrimSpace(lines[index]) != "" {
			return index
		}
	}
	return -1
}

func isDefaultLine(line string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "default:")
}

// maskSeatQuestionSurface 把围栏代码块与引用行替换为空行，避免把提示词里
// 的 NEEDS INPUT 示例当成席位真提问。未闭合围栏之后的全部内容都视为代码。
func maskSeatQuestionSurface(text string) string {
	lines := strings.Split(text, "\n")
	fence := ""
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			lines[index] = ""
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = trimmed[:3]
			lines[index] = ""
			continue
		}
		if strings.HasPrefix(trimmed, ">") {
			lines[index] = ""
		}
	}
	return strings.Join(lines, "\n")
}
