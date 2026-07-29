// Package whisper — plan_json_parse.go
// 100% 对齐 ackem desktop-agent/task-plan/planJsonParse.ts
// LLM JSON 修复：去代码块+引号修复+尾逗号清理

package whisper

import (
	"encoding/json"
	"regexp"
	"strings"
)

// stripCodeFence 去掉 markdown 代码块包裹
func stripCodeFence(text string) string {
	text = regexp.MustCompile(`(?i)^\x60\x60\x60(?:json)?\s*`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`(?i)\s*\x60\x60\x60\s*$`).ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

// repairJsonText 常见 LLM JSON 瑕疵修复
func repairJsonText(text string) string {
	s := stripCodeFence(strings.TrimSpace(text))
	// 中文引号 → 英文引号
	s = strings.NewReplacer("\u201c", "\"", "\u201d", "\"", "\u2018", "'", "\u2019", "'").Replace(s)
	// 尾逗号清理
	s = regexp.MustCompile(`,\s*([}\]])`).ReplaceAllString(s, "$1")
	// 无引号key修复
	s = regexp.MustCompile(`\b(\w+)\s*:`).ReplaceAllString(s, "\"$1\":")
	return s
}

// ExtractJsonObject 从 LLM 文本中提取 JSON 对象
func ExtractJsonObject(text string) map[string]interface{} {
	t := stripCodeFence(strings.TrimSpace(text))
	if t == "" {
		return nil
	}

	attempts := []string{t, repairJsonText(t)}
	start := strings.Index(t, "{")
	end := strings.LastIndex(t, "}")
	if start >= 0 && end > start {
		slice := t[start : end+1]
		attempts = append(attempts, slice, repairJsonText(slice))
	}

	for _, candidate := range attempts {
		if candidate == "" {
			continue
		}
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(candidate), &result); err == nil && result != nil {
			return result
		}
	}
	return nil
}

// BuildJsonRepairUserMessage 构建 JSON 修复 user prompt
func BuildJsonRepairUserMessage(invalidOutput string) string {
	output := invalidOutput
	if len(output) > 2000 {
		output = output[:2000]
	}
	return "上一次输出不是合法 JSON。请只输出一个 JSON 对象，不要 markdown，不要解释。\n" +
		"必须包含 goalSummary (string) 和 steps (array)。\n" +
		"错误输出如下：\n" + output
}
