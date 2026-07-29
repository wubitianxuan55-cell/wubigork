// Package whisper — resolve_display_title.go
// 100% 对齐 ackem paperCard/resolveDisplayTitle.ts
// 纸面卡 UI 展示标题解析：正文标题 > 规则主题 > 类型默认

package whisper

import "strings"

// PaperCardKind 纸面卡类型
type PaperCardKind string

const (
	CardPlan      PaperCardKind = "plan"
	CardKnowledge PaperCardKind = "knowledge"
	CardSearch    PaperCardKind = "search"
	CardTable     PaperCardKind = "table"
)

// kindLabels 纸面卡类型中文标签
var kindLabels = map[PaperCardKind]string{
	CardPlan:      "计划书",
	CardKnowledge: "知识整理",
	CardSearch:    "检索摘录",
	CardTable:     "对比表",
}

// ExtractTitleFromCardBody 从正文提取标题（以 # 开头的第一行）
func ExtractTitleFromCardBody(body string) string {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			title := strings.TrimLeft(line, "# ")
			title = strings.TrimSpace(title)
			if title != "" {
				return title
			}
		}
	}
	return ""
}

// IsPoorPaperCardTitle 判断标题是否太差（太短/太长/含类型词）
func IsPoorPaperCardTitle(title string) bool {
	t := strings.TrimSpace(title)
	runes := []rune(t)
	if len(runes) < 4 || len(runes) > 28 {
		return true
	}
	poorWords := []string{"计划书", "知识整理", "检索摘录", "对比表", "整理卡"}
	for _, w := range poorWords {
		if strings.Contains(t, w) {
			return true
		}
	}
	return false
}

// DefaultPaperCardTitle 默认纸面卡标题
func DefaultPaperCardTitle(kind PaperCardKind) string {
	if label, ok := kindLabels[kind]; ok {
		return label
	}
	return "整理"
}

// ResolvePaperCardDisplayTitle 解析纸面卡展示标题
func ResolvePaperCardDisplayTitle(kind PaperCardKind, ruleTopic, cardBody string) string {
	// 1. 正文标题优先
	if title := ExtractTitleFromCardBody(cardBody); title != "" {
		return title
	}

	// 2. 规则主题
	topic := strings.TrimSpace(ruleTopic)
	if len([]rune(topic)) > 28 {
		topic = string([]rune(topic)[:28])
	}
	if topic != "" && !IsPoorPaperCardTitle(topic) {
		return topic
	}

	// 3. 类型默认
	return DefaultPaperCardTitle(kind)
}
