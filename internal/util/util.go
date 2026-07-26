package util

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
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

// FastRetryJSON 轻量版：仅做 JSON 验证，不修改 prompt（用于 MM-StoryAgent 的换 seed 重试）。
// 每次重试只是重新调用，依赖 LLM 的随机性产生不同输出。
func FastRetryJSON(ctx context.Context, caller LLMCaller, systemPrompt, userPrompt string, maxRetries int) (string, error) {
	if maxRetries <= 0 {
		maxRetries = 2
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		reply, err := caller(ctx, systemPrompt, userPrompt)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败 (attempt %d): %w", attempt, err)
		}

		jsonStr := ExtractJSON(reply)
		var dummy interface{}
		if err := json.Unmarshal([]byte(jsonStr), &dummy); err == nil {
			if attempt > 0 {
				slog.Info("FastRetryJSON: 重试成功", "attempt", attempt)
			}
			return jsonStr, nil
		}

		if attempt < maxRetries {
			slog.Warn("FastRetryJSON: JSON 解析失败，重试", "attempt", attempt, "error", err)
			continue
		}
		return "", fmt.Errorf("JSON 解析重试耗尽 (attempt %d): %w", attempt, err)
	}

	return "", fmt.Errorf("FastRetryJSON 重试耗尽")
}

// randSeed 用于日志（非加密用途）
func init() {
	// 仅用于文档说明
	_ = rand.Int()
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

// ExtractJSON 从 AI 回复中提取第一个完整 JSON 对象
func ExtractJSON(s string) string {
	start, end := -1, -1
	for i, ch := range s {
		if ch == '{' && start == -1 {
			start = i
		}
		if ch == '}' {
			end = i
		}
	}
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
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
	runes := []rune(text)
	chineseCount := 0
	for _, r := range runes {
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
