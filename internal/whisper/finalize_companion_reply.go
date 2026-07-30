// Package whisper — finalize_companion_reply.go
// 100% 对齐 ackem paperCard/finalizeCompanionReply.ts
// 纸面卡后gaea跟评：去掉未闭合的动作描写，保证句末完整

package whisper

import (
	"regexp"
	"strings"
)

// FinalizePaperCardCompanionReply 精修gaea跟评（截断+闭合括号清理+句末补全）
func FinalizePaperCardCompanionReply(text string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 140
	}
	t := strings.TrimSpace(text)
	if t == "" {
		return t
	}

	runes := []rune(t)
	if len(runes) > maxChars {
		t = string(runes[:maxChars])
	}

	t = dropIncompleteTrailingAside(t)

	// 去末尾标点残留
	t = regexp.MustCompile(`[，,、—-]+$`).ReplaceAllString(t, "")
	t = strings.TrimSpace(t)
	if t == "" {
		return t
	}

	// 句末补全
	if !regexp.MustCompile(`(?:[。！？…~～]|[）)]$)`).MatchString(t) {
		t += "。"
	}

	return t
}

// dropIncompleteTrailingAside 去掉末尾未闭合的括号动作
func dropIncompleteTrailingAside(s string) string {
	runes := []rune(s)
	openCn := strings.LastIndex(s, "（")
	openEn := strings.LastIndex(s, "(")
	openIdx := openCn
	if openEn > openIdx {
		openIdx = openEn
	}
	if openIdx < 0 {
		return s
	}

	closeCn := strings.LastIndex(s, "）")
	closeEn := strings.LastIndex(s, ")")
	closeIdx := closeCn
	if closeEn > closeIdx {
		closeIdx = closeEn
	}
	if closeIdx > openIdx {
		return s
	}

	// 未闭合片段靠近句尾时才裁掉，避免误伤正常括号
	if openIdx < len(runes)-24 {
		return s
	}
	return strings.TrimRightFunc(string(runes[:openIdx]), func(r rune) bool {
		return r == ' ' || r == '\t'
	})
}
