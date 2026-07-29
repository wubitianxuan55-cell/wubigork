// Package whisper — trigger_extractor.go
// 100% 对齐 ackem memory/triggerExtractor.ts
// 从事实 subject+summary 中自动提取触发词

package whisper

import "strings"

// stopwords 中文停用词集
var stopwords = map[string]bool{
	"的": true, "了": true, "是": true, "在": true, "我": true, "你": true, "他": true, "她": true, "它": true,
	"这": true, "那": true, "很": true, "都": true, "也": true, "就": true, "还": true, "要": true, "会": true,
	"有": true, "不": true, "没": true, "和": true, "与": true, "或": true, "但": true, "而": true, "所": true,
	"被": true, "把": true, "让": true, "从": true, "到": true, "对": true, "向": true, "给": true, "跟": true,
	"为": true, "以": true, "因为": true, "所以": true, "如果": true, "虽然": true, "但是": true,
	"上": true, "下": true, "中": true, "里": true, "外": true,
	"一个": true, "什么": true, "怎么": true, "哪里": true, "为什么": true, "怎么样": true,
	"可以": true, "可能": true, "应该": true, "能够": true, "已经": true, "开始": true, "继续": true,
}

// ExtractTriggers 从事实际题+摘要中提取触发词（最多5个）
func ExtractTriggers(subject, summary string) []string {
	text := subject + " " + summary

	// 按 CJK 标点+空白分割
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == ',' || r == '，' || r == '。' || r == '！' || r == '？' ||
			r == '、' || r == '；' || r == '：' || r == '"' || r == '\'' ||
			r == '（' || r == '）' || r == '【' || r == '】' || r == '《' || r == '》'
	})

	var words []string
	seen := make(map[string]bool)
	for _, f := range fields {
		f = strings.TrimSpace(f)
		runes := []rune(f)
		if len(runes) < 2 {
			continue
		}
		if stopwords[f] {
			continue
		}
		if !seen[f] {
			seen[f] = true
			words = append(words, f)
		}
		// 也添加 bigram
		for i := 0; i < len(runes)-1; i++ {
			bg := string(runes[i : i+2])
			if !stopwords[bg] && !seen[bg] {
				seen[bg] = true
				words = append(words, bg)
			}
		}
	}

	// 去重，取前 5 个
	unique := make([]string, 0, 5)
	seen2 := make(map[string]bool)
	for _, w := range words {
		if !seen2[w] {
			seen2[w] = true
			unique = append(unique, w)
		}
	}
	if len(unique) > 5 {
		unique = unique[:5]
	}
	return unique
}
