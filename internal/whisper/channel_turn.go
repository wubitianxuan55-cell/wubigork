// Package whisper — channel_turn.go
// 100% 对齐 ackem channels/companionTurn.ts
// 全链路gaea对话编排：合并引擎状态 → LLM 调用 → 后处理

package whisper

// ─── CompanionTurnInput ───────────────────────────────────────

// CompanionTurnInput gaea对话轮次输入
type CompanionTurnInput struct {
	UserMsg    string
	SessionID  string
	AdultMode  bool
	EngineID   string // LLM 引擎 ID
}

// ─── CompanionTurnResult ──────────────────────────────────────

// CompanionTurnResult gaea对话轮次输出
type CompanionTurnResult struct {
	AssistantReply string
	SystemPrompt   string
	Trace          TurnTrace
	SkipLLM        bool
	RedlineReply   string
}

// ─── CompanionTurn ────────────────────────────────────────────

// CompanionTurn 全链路gaea对话编排
// 等价于 ackem channels/companionTurn.ts 的 runCompanionTurn
func (o *Orchestrator) CompanionTurn(input CompanionTurnInput) *CompanionTurnResult {
	// 1. 运行 PreLLMTurn
	pre := o.PreLLMTurn(input.UserMsg)

	// 2. 红线熔断 → 直接返回安全回复
	if pre.SkipLLM {
		return &CompanionTurnResult{
			AssistantReply: pre.RedlineReply,
			SystemPrompt:   pre.SystemPrompt,
			Trace:          pre.Trace,
			SkipLLM:        true,
			RedlineReply:   pre.RedlineReply,
		}
	}

	// 3. 正常流程：返回 systemPrompt 和 psycheBlock 供 LLM 调用
	// LLM 调用由上层（app handler）执行，本模块只负责生成 prompt
	return &CompanionTurnResult{
		SystemPrompt: pre.SystemPrompt,
		Trace:        pre.Trace,
	}
}

// ─── ApplyLLMReply ────────────────────────────────────────────

// ApplyLLMReply 应用 LLM 回复到状态（post-turn 更新）
// P2增强：接入完整 FinalizeTurn 管线
func (o *Orchestrator) ApplyLLMReply(reply string, event Event) {
	ctx := PostTurnContext{
		SessionID:     o.SessionID,
		TurnIndex:     o.State.Counters.TotalTurns,
		AssistantText: reply,
		Event:         event,
		AdultMode:     o.AdultMode,
	}
	FinalizeTurn(o, ctx)
}
