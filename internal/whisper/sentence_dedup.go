// Package whisper — sentence_dedup.go
// 100% 对齐 ackem chat/sentenceDedup.ts
// 句子去重：bigram相似度 + 包含检测，防止多波/流式回复中的重复

package whisper

import (
	"regexp"
	"strings"
)

// TurnDedupState 轮次去重状态
type TurnDedupState struct {
	DisplayedSentences []string
}

// NewTurnDedupState 创建去重状态
func NewTurnDedupState() *TurnDedupState {
	return &TurnDedupState{}
}

var orphanOnlyRE = regexp.MustCompile(`^[()（）\[\]【】，,、；;：:…\s]+$`)

// normalizeSentence 规范化句子（去标点/空白/小写）
func normalizeSentence(s string) string {
	s = strings.TrimSpace(s)
	// 去空白
	s = regexp.MustCompile(`[\s\u3000]+`).ReplaceAllString(s, "")
	// 去标点
	s = regexp.MustCompile(`[。！？!?….,，、；;：:""''「」『』（）()【】\[\]]+`).ReplaceAllString(s, "")
	return strings.ToLower(s)
}

// isSubsumed 检查a是否被b包含或相同
func isSubsumed(a, b string) bool {
	na := normalizeSentence(a)
	nb := normalizeSentence(b)
	if na == "" || nb == "" {
		return false
	}
	if na == nb {
		return true
	}
	maxLen := len([]rune(na))
	if len([]rune(nb)) > maxLen {
		maxLen = len([]rune(nb))
	}
	if maxLen <= 24 {
		return strings.Contains(na, nb) || strings.Contains(nb, na)
	}
	return false
}

// bigramSimilarity 计算两个字符串的bigram相似度
func bigramSimilarity(a, b string) float64 {
	na := normalizeSentence(a)
	nb := normalizeSentence(b)
	if len([]rune(na)) < 2 || len([]rune(nb)) < 2 {
		if na == nb {
			return 1
		}
		return 0
	}

	bg := func(s string) map[string]bool {
		set := make(map[string]bool)
		runes := []rune(s)
		for i := 0; i < len(runes)-1; i++ {
			set[string(runes[i:i+2])] = true
		}
		return set
	}

	sa := bg(na)
	sb := bg(nb)
	inter := 0
	for x := range sa {
		if sb[x] {
			inter++
		}
	}
	union := len(sa) + len(sb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// ShouldEmitSentence 判断句子是否应该发射（去重检测）
func ShouldEmitSentence(sentence string, displayed []string) bool {
	trimmed := strings.TrimSpace(sentence)
	if trimmed == "" {
		return false
	}
	if orphanOnlyRE.MatchString(trimmed) {
		return false
	}

	for _, prior := range displayed {
		if isSubsumed(trimmed, prior) {
			return false
		}
		if normalizeSentence(trimmed) == normalizeSentence(prior) {
			return false
		}
		maxLen := len([]rune(trimmed))
		if len([]rune(prior)) > maxLen {
			maxLen = len([]rune(prior))
		}
		threshold := 0.55
		if maxLen <= 40 {
			threshold = 0.68
		}
		if bigramSimilarity(trimmed, prior) > threshold {
			return false
		}
	}
	return true
}

// RecordDisplayedSentence 记录已显示的句子
func (s *TurnDedupState) RecordDisplayedSentence(sentence string) {
	t := strings.TrimSpace(sentence)
	if t != "" {
		s.DisplayedSentences = append(s.DisplayedSentences, t)
	}
}

// Reset 重置去重状态
func (s *TurnDedupState) Reset() {
	s.DisplayedSentences = nil
}
