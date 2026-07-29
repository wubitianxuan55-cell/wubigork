// Package whisper — dispatch_router.go
// 对齐 ackem engine/dispatchRouter.ts
// 分发路由：决定本轮是继续对话还是进入 plan/skill 模式

package whisper

// ─── DispatchDecision ─────────────────────────────────────────

// DispatchDecision 分发决策
type DispatchDecision string

const (
	DispatchContinue  DispatchDecision = "continue"   // 继续对话
	DispatchPlan      DispatchDecision = "plan"       // 进入 plan 模式
	DispatchSkill     DispatchDecision = "skill"      // 触发技能
	DispatchAskInvoke DispatchDecision = "ask_invoke" // 询问是否调用扩展
)

// DispatchResult 分发结果
type DispatchResult struct {
	Decision    DispatchDecision `json:"decision"`
	PlanTopic   string           `json:"planTopic,omitempty"`
	SkillName   string           `json:"skillName,omitempty"`
	Confidence  float64          `json:"confidence"`
	Reasoning   string           `json:"reasoning"`
	AskMessage  string           `json:"askMessage,omitempty"`
}

// RouteDispatch 路由分发决策
func RouteDispatch(
	userMsg string,
	trust float64,
	stage RelationshipStage,
	adultMode bool,
) *DispatchResult {
	// 默认：继续对话
	result := &DispatchResult{
		Decision:   DispatchContinue,
		Confidence: 1.0,
		Reasoning:  "正常对话",
	}

	// 低信任 → 不触发任何扩展
	if trust < 20 {
		return result
	}

	// 检测是否需要 plan 模式（复杂任务请求）
	if isComplexTaskRequest(userMsg) && trust >= 40 {
		return &DispatchResult{
			Decision:   DispatchPlan,
			Confidence: 0.7,
			Reasoning:  "检测到复杂任务请求",
		}
	}

	return result
}

func isComplexTaskRequest(msg string) bool {
	complexIndicators := []string{
		"帮我做", "帮我查", "帮我整理", "帮我分析",
		"写一个", "创建一个", "生成一个",
		"总结一下", "调查一下", "研究一下",
	}
	for _, kw := range complexIndicators {
		if len(msg) >= len(kw) && containsSubstr(msg, kw) {
			return true
		}
	}
	return false
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
