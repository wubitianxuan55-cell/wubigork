package tts

import (
	"strings"
	"unicode/utf8"
)

const (
	TempDirName = "gaea-tts"
	OutputWAV   = "speech.wav"
)

// SplitSentences 将文本按中文标点拆分为句子（每句尽量短，利于流式播放）
func SplitSentences(text string) []string {
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '。' || r == '！' || r == '？' || r == '\n'
	})

	var sentences []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if utf8.RuneCountInString(p) > 100 {
			subParts := strings.FieldsFunc(p, func(r rune) bool {
				return r == '，' || r == '；' || r == '：'
			})
			for _, sp := range subParts {
				sp = strings.TrimSpace(sp)
				if sp != "" {
					sentences = append(sentences, sp)
				}
			}
		} else {
			sentences = append(sentences, p)
		}
	}
	return sentences
}
