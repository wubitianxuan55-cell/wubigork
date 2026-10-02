package util

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// ── JSON 解析重试（蒸馏自 MM-StoryAgent 的 success_check_fn + retry loop）──

// LLMCaller 抽象 LLM 调用，用于 RetryJSON 的重试循环。
// systemPrompt 和 userPrompt 会因重试而修改（注入格式修复提示）。
type LLMCaller func(ctx context.Context, systemPrompt, userPrompt string) (string, error)

// RetryJSON 调用 LLM → 解析 JSON → 失败则重试（最多 maxRetries 次）。
// 每次重试会注入更强的格式约束到 systemPrompt 中。
func RetryJSON(ctx context.Context, caller LLMCaller, systemPrompt, userPrompt string, maxRetries int) (string, error) {
	if maxRetries <= 0 {
		maxRetries = 2
	}

	currentSystem := systemPrompt
	currentUser := userPrompt

	for attempt := 0; attempt <= maxRetries; attempt++ {
		reply, err := caller(ctx, currentSystem, currentUser)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败 (attempt %d): %w", attempt, err)
		}

		jsonStr := ExtractJSON(reply)
		if jsonStr == "" || jsonStr == reply {
			// 没有找到 JSON，重试
			if attempt < maxRetries {
				slog.Warn("RetryJSON: 未找到 JSON，重试", "attempt", attempt)
				currentSystem += "\n\n重要：请严格输出 JSON 格式，用 ```json 代码块包裹。"
				continue
			}
			return "", fmt.Errorf("未找到有效的 JSON (attempt %d): %s", attempt, Truncate(reply, 200))
		}

		// 验证 JSON 可解析
		var dummy interface{}
		if err := json.Unmarshal([]byte(jsonStr), &dummy); err != nil {
			if attempt < maxRetries {
				slog.Warn("RetryJSON: JSON 解析失败，重试", "attempt", attempt, "error", err)
				currentSystem += fmt.Sprintf("\n\n你的上一次输出 JSON 解析失败 (%v)。请确保输出严格合法的 JSON。", err)
				// 微调 temperature 效果：修改 prompt 措辞
				if attempt == 1 {
					currentUser += "\n\n（请确保输出合法 JSON，不要包含注释或尾部逗号）"
				}
				continue
			}
			return "", fmt.Errorf("JSON 解析失败 (attempt %d): %w\n原始: %s", attempt, err, Truncate(jsonStr, 300))
		}

		if attempt > 0 {
			slog.Info("RetryJSON: 重试成功", "attempt", attempt)
		}
		return jsonStr, nil
	}

	return "", fmt.Errorf("JSON 解析重试耗尽 (%d 次)", maxRetries+1)
}

// SafeGo 安全启动 goroutine，自动 recover panic 并记录日志。
// 用于替代 handler 中重复的 goroutine + panic recover 模式。
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("goroutine panic", "panic", r)
			}
		}()
		fn()
	}()
}

// ExtractJSON 从 AI 回复中提取第一个完整的 JSON 对象（或顶层数组）。
//
// 与旧实现（「第一个 { 到最后一个 }」）相比，语义变化仅限修复 P0-22：
//   - 按括号深度配平切分，回复里再出现第二个对象、或后文顺带提到 {} 时不再污染结果；
//   - 字符串字面量内部的花括号与转义引号（\" 与 \\）不计入深度；
//   - 支持顶层数组（gaea_xlsx_edit.go 等调用方的目标是 []xlsxedit.Op），
//     但对象优先于纯标量数组，避免正文里的 [1] 这类引用噪声抢先被当成结果；
//   - 带 ```json 围栏时优先在围栏内提取。
//
// 向后兼容：找不到任何完整 JSON 片段时原样返回 s（不 trim、不返回空串），
// 与旧实现一致；调用方仍可用 json.Unmarshal 的成功与否判断是否真的拿到了 JSON。
func ExtractJSON(s string) string {
	if seg, ok := extractFromJSONFence(s); ok {
		return seg
	}
	if seg, ok := extractBalancedJSON(s); ok {
		return seg
	}
	return s
}

// extractBalancedJSON 从左到右扫描，返回第一个「完整的、合法的、且值得优先返回」的 JSON 片段。
// 结构完整但非法（例如正文里的 [附录]、{模型胡说}）的候选会被跳过，以免正文噪声抢在真正的
// JSON 之前；若所有候选都不满足优先级，则退化为返回第一个结构完整片段，保证行为可预期。
func extractBalancedJSON(s string) (string, bool) {
	firstComplete := ""
	hasComplete := false
	for i := 0; i < len(s); i++ {
		if c := s[i]; c != '{' && c != '[' {
			continue
		}
		seg, ok := scanBalanced(s, i)
		if !ok {
			continue
		}
		if json.Valid([]byte(seg)) && isPreferredCandidate(seg) {
			return seg, true
		}
		if !hasComplete {
			firstComplete, hasComplete = seg, true
		}
	}
	if hasComplete {
		return firstComplete, true
	}
	return "", false
}

// isPreferredCandidate 判断已配平的片段是否值得优先返回。
// 对象一律优先；数组只在是「结构化数组」（首元素为对象或数组）时才优先，
// 这样正文里的 [1]、[2]、["注"] 这类引用噪声不会抢在真正的 JSON 对象之前，
// 同时 [{"op":1},{"op":2}] 这类真正的顶层数组仍能被正确提取。
// 被降级的候选不会丢失：若无更优候选，extractBalancedJSON 会兜底返回它，
// 因此纯标量数组（[1,2,3]）与字符串数组（["a","b"]）单独出现时行为不变。
func isPreferredCandidate(seg string) bool {
	if seg == "" {
		return false
	}
	if seg[0] == '{' {
		return true
	}
	for i := 1; i < len(seg); i++ {
		switch seg[i] {
		case ' ', '\t', '\n', '\r':
			continue
		case '{', '[':
			return true
		default:
			return false
		}
	}
	return false
}

// scanBalanced 从 s[start]（'{' 或 '['）出发做括号配平扫描，深度归零时返回该完整片段。
// 字符串内部按「转义优先」状态机处理，因此 "}"、"{" 与 \" 不会影响深度。
func scanBalanced(s string, start int) (string, bool) {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth <= 0 {
				return s[start : i+1], true
			}
		}
	}
	return "", false
}

// extractFromJSONFence 在带 json 语言标记的 markdown 代码围栏内提取 JSON。
// 只认 ```json / ```jsonc 这类标记，裸围栏与其它语言围栏交给全局扫描统一处理。
func extractFromJSONFence(s string) (string, bool) {
	rest := s
	for {
		i := strings.Index(rest, "```")
		if i < 0 {
			return "", false
		}
		rest = rest[i+3:]
		nl := strings.IndexByte(rest, '\n')
		if nl < 0 {
			return "", false
		}
		lang := strings.ToLower(strings.TrimSpace(rest[:nl]))
		body := rest[nl+1:]
		end := strings.Index(body, "```")
		closed := end >= 0
		block := body
		if closed {
			block = body[:end]
		}
		if strings.HasPrefix(lang, "json") {
			if seg, ok := extractBalancedJSON(block); ok {
				return seg, true
			}
		}
		if !closed {
			return "", false
		}
		rest = body[end+3:]
	}
}

// Truncate 按 rune 截断字符串
func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// Max 返回两个 int 中较大值
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min 返回两个 int 中较小值
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RefLimit 素材长度阈值：超过此值的参考素材需先归纳压缩
const RefLimit = 12000

// TruncateRef 截断过长的素材（作为归纳失败的兜底方案）
func TruncateRef(s string) (string, bool) {
	runes := []rune(s)
	if len(runes) <= RefLimit {
		return s, false
	}
	return string(runes[:RefLimit]) + "\n\n（注：参考素材过长，已自动截断至前 12000 字符。建议精简素材后重试。）", true
}

// MustMarshal 序列化为 JSON 缩进格式，失败时 panic（仅用于已知结构体）
func MustMarshal(v interface{}) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic("util.MustMarshal: " + err.Error())
	}
	return b
}

// MustMarshalCompact 序列化为紧凑 JSON，失败时 panic
func MustMarshalCompact(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic("util.MustMarshalCompact: " + err.Error())
	}
	return b
}

// EstimateTokens 粗略估算文本的 token 数：中文 1 字 ≈ 1.5 tokens, 英文 1 词 ≈ 1.3 tokens
func EstimateTokens(text string) int {
	chineseCount := 0
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseCount++
		}
	}
	englishWords := len(strings.Fields(text)) - chineseCount/2
	if englishWords < 0 {
		englishWords = 0
	}
	return int(float64(chineseCount)*1.5 + float64(englishWords)*1.3)
}

// MarkedSection 标记区段结果
type MarkedSection struct {
	RawJSON string
}

// ParseMarkedSections 从 AI 回复中解析指定标记之间的 JSON 区段
// 用于 worldview/character 等模块的 ---MARKER--- ... ---END_MARKER--- 模式
func ParseMarkedSections(reply, marker, endMarker string) ([]MarkedSection, error) {
	var sections []MarkedSection
	for {
		start := strings.Index(reply, marker)
		if start == -1 {
			break
		}
		end := strings.Index(reply[start:], endMarker)
		if end == -1 {
			break
		}
		jsonStr := reply[start+len(marker) : start+end]
		reply = reply[start+end+len(endMarker):]

		if trimmed := strings.TrimSpace(jsonStr); len(trimmed) > 0 {
			sections = append(sections, MarkedSection{RawJSON: trimmed})
		}
	}
	if len(sections) == 0 {
		return nil, fmt.Errorf("未找到标记区段: %s ... %s", marker, endMarker)
	}
	return sections, nil
}
