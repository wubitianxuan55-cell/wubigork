package session

// 3.0 Step 1: 派生 API（纯函数，日志重放计算，供绑定层后续接线）。
//   - 会话标题：首条 user_message 的文本（截断）；
//   - token/成本统计：usage 事件累加（含 pricing 折叠的成本）。

import (
	"encoding/json"
	"strings"
)

// TitlePreviewMax 是派生标题的最大 rune 数。
const TitlePreviewMax = 60

// DeriveTitle 从日志派生会话标题：首条 user_message 的文本（去首尾空白、
// 超长截断加省略号）。无 user_message 返回空串。
func DeriveTitle(entries []LogEntry) string {
	for _, e := range entries {
		if e.Kind != KindUserMessage {
			continue
		}
		var p userLogPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			continue
		}
		s := strings.TrimSpace(p.Content)
		if s == "" {
			continue
		}
		r := []rune(s)
		if len(r) > TitlePreviewMax {
			return string(r[:TitlePreviewMax-1]) + "…"
		}
		return s
	}
	return ""
}

// TokenStats 是日志重放累计的 token/成本统计。
type TokenStats struct {
	PromptTokens     int64 `json:"promptTokens"`
	CompletionTokens int64 `json:"completionTokens"`
	TotalTokens      int64 `json:"totalTokens"`
	CacheHitTokens   int64 `json:"cacheHitTokens"`
	CacheMissTokens  int64 `json:"cacheMissTokens"`
	ReasoningTokens  int64 `json:"reasoningTokens"`
	// UsageCount 是 usage 事件条数（≈ API 调用轮次，含子代理/压缩调用）。
	UsageCount int `json:"usageCount"`
	// Cost 是按 pricing 累加的估算成本（计价单位 per 1M tokens）。
	Cost     float64 `json:"cost"`
	Currency string  `json:"currency,omitempty"`
	// MainCost / SubagentCost 按 UsageSource 拆分。
	MainCost     float64 `json:"mainCost"`
	SubagentCost float64 `json:"subagentCost"`
}

// DeriveStats 从日志累加 usage 事件得到 token/成本统计。
func DeriveStats(entries []LogEntry) TokenStats {
	var st TokenStats
	for _, e := range entries {
		if e.Kind != "usage" {
			continue
		}
		var p usageLogPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			continue
		}
		st.PromptTokens += p.PromptTokens
		st.CompletionTokens += p.CompletionTokens
		st.TotalTokens += p.TotalTokens
		st.CacheHitTokens += p.CacheHitTokens
		st.CacheMissTokens += p.CacheMissTokens
		st.ReasoningTokens += p.ReasoningTokens
		st.UsageCount++
		cost := (float64(p.CacheHitTokens)*p.CacheHitPrice +
			float64(p.CacheMissTokens)*p.Input +
			float64(p.CompletionTokens)*p.Output) / 1e6
		st.Cost += cost
		if p.Currency != "" {
			st.Currency = p.Currency
		}
		switch p.Source {
		case "subagent":
			st.SubagentCost += cost
		case "main", "executor", "":
			st.MainCost += cost
		}
	}
	return st
}
