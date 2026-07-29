// Package whisper — sentence_split.go
// 100% 对齐 ackem chat/sentenceBubbleStream.ts (核心算法部分)
// 句子切分：安全断句+括号/引号感知+完整句子剥离

package whisper

import "strings"

// FindSafeSentenceBreak 找到安全断句位置（跳过括号/引号内的句号）
func FindSafeSentenceBreak(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return -1
	}

	depth := 0
	inQuote := rune(0)
	runes := []rune(text)

	for i, ch := range runes {
		if inQuote != 0 {
			if ch == inQuote {
				inQuote = 0
			}
			continue
		}
		switch ch {
		case '"', '\'', '「', '『':
			inQuote = ch
			continue
		case '（', '(', '【', '[':
			depth++
			continue
		case '）', ')', '】', ']':
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth > 0 {
			continue
		}
		if ch == '。' || ch == '！' || ch == '？' || ch == '!' || ch == '?' || ch == '…' {
			return i + 1
		}
	}
	return -1
}

// PeelCompleteSentences 从缓冲区剥离完整句子
func PeelCompleteSentences(buffer string) (sentences []string, remainder string) {
	rest := buffer
	for {
		end := FindSafeSentenceBreak(rest)
		if end <= 0 {
			break
		}
		sentence := strings.TrimSpace(string([]rune(rest)[:end]))
		if sentence != "" {
			sentences = append(sentences, sentence)
		}
		rest = strings.TrimLeftFunc(string([]rune(rest)[end:]), func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n'
		})
	}
	return sentences, rest
}

// SplitIntoSentences 将文本切分为句子列表
func SplitIntoSentences(text string) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	var out []string
	rest := trimmed
	for len([]rune(rest)) > 0 {
		sentences, remainder := PeelCompleteSentences(rest)
		if len(sentences) == 0 {
			out = append(out, strings.TrimSpace(rest))
			break
		}
		out = append(out, sentences...)
		rest = remainder
	}
	// 过滤空字符串
	var filtered []string
	for _, s := range out {
		if s != "" {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
