// Package whisper — dispatch_router.go
// 100% 对齐 ackem engine/dispatchRouter.ts（简化版：gaea 无扩展系统）
// 分发路由：关键词→语义评分→LLM精判→决策

package whisper

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// ─── 类型 ──────────────────────────────────────────────────────

// DispatchDecision 分发决策
type DispatchDecision string

const (
	DispatchChat       DispatchDecision = "chat"        // 普通对话
	DispatchPlan       DispatchDecision = "plan"        // 进入 plan/skill 模式
	DispatchAutoInvoke DispatchDecision = "auto_invoke" // 自动触发
	DispatchAskInvoke  DispatchDecision = "ask_invoke"  // 询问是否触发
	DispatchSilent     DispatchDecision = "silent"      // 候选存在但不触发
)

// DispatchResult 分发结果
type DispatchResult struct {
	Decision    DispatchDecision `json:"decision"`
	ExtensionID string           `json:"extensionId,omitempty"`
	Confidence  float64          `json:"confidence"`
	Reasoning   string           `json:"reasoning"`
	AskMessage  string           `json:"askMessage,omitempty"`
	PlanTopic   string           `json:"planTopic,omitempty"`
}

// RouteDispatchInput 路由调度输入
type RouteDispatchInput struct {
	UserMessage   string
	SessionID     string
	Candidates    []DispatchCandidate
	Now           time.Time
	PersonalityID string
	RecentContext string
	EmotionLabel  string
	MemoryBlock   string
	ActivityHint  string
	LlmCall       func(prompt string) (string, error)
}

// DispatchCandidate 分发候选条目
type DispatchCandidate struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Summary   string   `json:"summary"`
	Keywords  []string `json:"keywords"`
	Scenarios []string `json:"scenarios"`
	Habits    []string `json:"habits"`
	Score     float64  `json:"score"`
}

// ─── 阈值 ──────────────────────────────────────────────────────

const (
	dispatchAutoThreshold = 0.85
	dispatchAskThreshold  = 0.60
)

// 人设乘数：不同人格对 tool use 的敏感度
var personalityDispatchMod = map[string]float64{
	"deredere": 1.15,
	"tsundere": 0.90,
	"kuudere":  1.25,
	"genki":    0.85,
}

// ─── RouteDispatch ─────────────────────────────────────────────

// RouteDispatch 核心路由决策
func RouteDispatch(input RouteDispatchInput) *DispatchResult {
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}

	// 1. 复杂任务检测 → plan
	if isComplexTaskRequest(input.UserMessage) {
		topic := extractComplexTopic(input.UserMessage)
		return &DispatchResult{
			Decision: DispatchPlan, PlanTopic: topic,
			Confidence: 0.85, Reasoning: "complex_task_detected",
		}
	}

	// 2. 收集候选
	candidates := input.Candidates
	if len(candidates) == 0 {
		// 尝试从用户消息中简单关键词匹配内置候选
		candidates = collectBuiltinCandidates(input.UserMessage)
	}
	if len(candidates) == 0 {
		return &DispatchResult{Decision: DispatchChat, Confidence: 1.0, Reasoning: "no_candidates"}
	}

	// 3. 无 LLM → 退回对话
	if input.LlmCall == nil {
		return &DispatchResult{Decision: DispatchChat, Confidence: 1.0, Reasoning: "no_llm_call"}
	}

	// 4. LLM 精判
	prompt := buildDispatchLlmPrompt(input.UserMessage, candidates, input.RecentContext,
		input.EmotionLabel, input.MemoryBlock, input.ActivityHint, now)

	raw, err := input.LlmCall(prompt)
	if err != nil {
		return &DispatchResult{Decision: DispatchChat, Confidence: 1.0, Reasoning: "llm_error"}
	}

	match := parseDispatchLlmMatch(raw)
	if !match.Matched || match.ExtensionID == "" {
		return &DispatchResult{Decision: DispatchSilent, Reasoning: "llm_no_match"}
	}

	// 5. 找对应候选
	var entry *DispatchCandidate
	for i := range candidates {
		if candidates[i].ID == match.ExtensionID {
			entry = &candidates[i]
			break
		}
	}
	if entry == nil {
		return &DispatchResult{Decision: DispatchSilent, Reasoning: "unknown_candidate"}
	}

	confidence := match.Confidence
	if confidence == 0 {
		confidence = entry.Score
	}

	// 6. 人设乘数 + 阈值
	multiplier := personalityDispatchMod[input.PersonalityID]
	if multiplier == 0 {
		multiplier = 1.0
	}
	adjusted := math.Min(1.0, confidence*multiplier)

	switch {
	case adjusted >= dispatchAutoThreshold:
		return &DispatchResult{
			Decision: DispatchAutoInvoke, ExtensionID: entry.ID,
			Confidence: adjusted, Reasoning: match.Reasoning,
		}
	case adjusted >= dispatchAskThreshold:
		return &DispatchResult{
			Decision: DispatchAskInvoke, ExtensionID: entry.ID,
			Confidence: adjusted, Reasoning: match.Reasoning,
			AskMessage: fmt.Sprintf("要不要我帮你用「%s」？%s", entry.Name, entry.Summary),
		}
	default:
		return &DispatchResult{
			Decision: DispatchSilent, ExtensionID: entry.ID,
			Confidence: adjusted, Reasoning: match.Reasoning,
		}
	}
}

// ─── 复杂任务检测 ──────────────────────────────────────────────

func isComplexTaskRequest(msg string) bool {
	indicators := []string{
		"帮我做", "帮我查", "帮我整理", "帮我分析", "帮我写",
		"写一个", "创建一个", "生成一个", "总结一下",
		"调查一下", "研究一下", "计划一下", "安排一下",
		"翻译", "计算", "比较一下",
	}
	for _, kw := range indicators {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}

func extractComplexTopic(msg string) string {
	// 提取 "帮我做X" 中的 X
	prefixes := []string{"帮我做", "帮我查", "帮我写", "帮我整理", "帮我分析"}
	for _, p := range prefixes {
		if idx := strings.Index(msg, p); idx >= 0 {
			rest := msg[idx+len(p):]
			rest = strings.TrimSpace(rest)
			if len(rest) > 40 {
				rest = rest[:40]
			}
			return rest
		}
	}
	return truncateStr(msg, 60)
}

// ─── 内置候选 ──────────────────────────────────────────────────

func collectBuiltinCandidates(msg string) []DispatchCandidate {
	var candidates []DispatchCandidate

	builtins := []struct {
		ID, Name, Summary string
		Keywords          []string
	}{
		{"weather", "天气查询", "查询当前或未来天气", []string{"天气", "下雨", "温度", "刮风", "下雪", "晴天"}},
		{"reminder", "提醒", "设置提醒事项", []string{"提醒我", "记得", "别忘了", "闹钟", "定时"}},
		{"search", "搜索", "搜索信息", []string{"搜索", "查一下", "帮我找", "搜一下"}},
		{"note", "笔记", "记录笔记", []string{"记下来", "笔记", "记录", "备忘录"}},
		{"calc", "计算", "数学计算", []string{"计算", "等于", "多少", "换算"}},
	}

	for _, b := range builtins {
		for _, kw := range b.Keywords {
			if strings.Contains(msg, kw) {
				candidates = append(candidates, DispatchCandidate{
					ID: b.ID, Name: b.Name, Summary: b.Summary,
					Keywords: b.Keywords,
				})
				break
			}
		}
	}
	return candidates
}

// ─── LLM Prompt ────────────────────────────────────────────────

type llmDispatchMatch struct {
	Matched     bool    `json:"matched"`
	ExtensionID string  `json:"extension_id"`
	Confidence  float64 `json:"confidence"`
	Reasoning   string  `json:"reasoning"`
}

func buildDispatchLlmPrompt(
	userMsg string, candidates []DispatchCandidate,
	recentCtx, emotionLabel, memoryBlock, activityHint string,
	now time.Time,
) string {
	var candidateLines []string
	for _, c := range candidates {
		candidateLines = append(candidateLines, fmt.Sprintf(
			"- ID: %s\n  功能：%s\n  适用场景：%s\n  关键词：%s",
			c.ID, c.Summary, strings.Join(c.Scenarios, "；"), strings.Join(c.Keywords, "；"),
		))
	}

	var parts []string
	parts = append(parts, "你是一个调度判断器。根据用户消息和上下文，判断是否应该触发以下功能。")
	parts = append(parts, "宁可漏掉，不要误触发。只返回 JSON。")
	parts = append(parts, "")
	parts = append(parts, fmt.Sprintf("时间：%s", now.Format(time.RFC3339)))
	parts = append(parts, fmt.Sprintf("情绪：%s", emotionLabel))
	if recentCtx != "" {
		parts = append(parts, fmt.Sprintf("最近对话：%s", truncateStr(recentCtx, 400)))
	}
	if activityHint != "" {
		parts = append(parts, fmt.Sprintf("用户场景：%s", activityHint))
	}
	if memoryBlock != "" {
		parts = append(parts, fmt.Sprintf("相关记忆：%s", truncateStr(memoryBlock, 1200)))
	}
	parts = append(parts, "")
	parts = append(parts, "候选功能：")
	parts = append(parts, strings.Join(candidateLines, "\n"))
	parts = append(parts, "")
	parts = append(parts, fmt.Sprintf("用户消息：\"%s\"", userMsg))
	parts = append(parts, "")
	parts = append(parts, `返回 JSON：{ "matched": boolean, "extension_id"?: string, "confidence"?: number, "reasoning"?: string }`)

	return strings.Join(parts, "\n")
}

func parseDispatchLlmMatch(raw string) llmDispatchMatch {
	trimmed := strings.TrimSpace(raw)
	i := strings.Index(trimmed, "{")
	j := strings.LastIndex(trimmed, "}")
	if i < 0 || j <= i {
		return llmDispatchMatch{}
	}
	var match llmDispatchMatch
	if err := json.Unmarshal([]byte(trimmed[i:j+1]), &match); err != nil {
		return llmDispatchMatch{}
	}
	return match
}
