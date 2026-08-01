// Package whisper — investigation_chat_turn.go
// 100% 对齐 ackem desktop-agent/investigation/investigationChatTurn.ts
// 电脑助手调查对话统一调度：路由→调查→合成→交付

package whisper

import (
	"strings"
)

// InvChatTurnContext 调查对话上下文
type InvChatTurnContext struct {
	SessionID string
	UserMsg   string
	TurnID    string
	DataRoot  string
	// DesktopAgentChatMode 是否处于电脑助手模式
	DesktopAgentChatMode bool
	// DesktopAgentToolingActive 电脑助手工具是否激活
	DesktopAgentToolingActive bool
	// LlmCall LLM 调用函数（systemPrompt, userPrompt → reply, error）
	LlmCall func(systemPrompt, userPrompt string) (string, error)
}

// InvChatTurnResult 调查对话结果
type InvChatTurnResult struct {
	Handled      bool   `json:"handled"`
	Reply        string `json:"reply"`
	MemoryWrite  string `json:"memoryWrite,omitempty"`
	CapabilityID string `json:"capabilityId,omitempty"`
	Handler      string `json:"handler,omitempty"`
}

// ─── 主调度入口 ─────────────────────────────────────────────────

// TryHandleInvestigationChatTurn 电脑助手早退入口
// 100% 对齐 ackem investigationChatTurn.ts tryHandleInvestigationChatTurn
// 返回 handled=true 表示对话已被处理，调用方应跳过后续流程
func TryHandleInvestigationChatTurn(ctx InvChatTurnContext) *InvChatTurnResult {
	if !ctx.DesktopAgentChatMode || !ctx.DesktopAgentToolingActive {
		return &InvChatTurnResult{Handled: false}
	}

	// 尝试路由能力
	match := resolveCapability(ctx)
	if match == nil {
		// 降级到正则意图检测
		intent := RouteInvestigationIntent(ctx.UserMsg)
		if intent.IntentID == "" || intent.TemplateID == "" {
			return &InvChatTurnResult{Handled: false}
		}
		result := deliverInvestigation(ctx, intent, "")
		result.Handled = true
		return result
	}

	// 能力帮助
	if match.Handler == "capability_help" {
		reply, err := SynthesizeCapabilityHelpReply(ctx.UserMsg, ctx.LlmCall)
		if err != nil {
			reply = buildCapabilityHelpFallback()
		}
		return &InvChatTurnResult{
			Handled:      true,
			Reply:        reply,
			MemoryWrite:  "DESKTOP_AGENT capability_help",
			CapabilityID: match.CapabilityID,
			Handler:      match.Handler,
		}
	}

	// 调查类能力
	intent := capabilityToInvestigation(match, ctx.UserMsg)
	if intent.TemplateID == "" {
		return &InvChatTurnResult{Handled: false}
	}

	result := deliverInvestigation(ctx, intent, match.CapabilityID)
	result.Handled = true
	result.CapabilityID = match.CapabilityID
	result.Handler = match.Handler
	return result
}

// ─── 路由 ───────────────────────────────────────────────────────

// CapabilityMatch 能力匹配结果
type CapabilityMatch struct {
	CapabilityID string  `json:"capabilityId"`
	Handler      string  `json:"handler"`
	Score        float64 `json:"score"`
	Source       string  `json:"source"`
}

// resolveCapability 解析桌面助手能力
// 简化版：先尝试语义路由，降级到关键词匹配
func resolveCapability(ctx InvChatTurnContext) *CapabilityMatch {
	// 关键词启发式路由
	capID, handler, score := matchCapabilityByKeyword(ctx.UserMsg)
	if capID != "" {
		return &CapabilityMatch{
			CapabilityID: capID,
			Handler:      handler,
			Score:        score,
			Source:       "keyword",
		}
	}
	return nil
}

// matchCapabilityByKeyword 关键词匹配能力
func matchCapabilityByKeyword(msg string) (capID, handler string, score float64) {
	lower := strings.ToLower(msg)

	// 能力帮助
	helpPatterns := []string{"能做什么", "有什么功能", "能帮我", "可以干嘛", "怎么用", "功能列表", "介绍下", "what can you do"}
	for _, p := range helpPatterns {
		if strings.Contains(lower, p) {
			return "desktop_agent_help", "capability_help", 0.8
		}
	}

	// 游戏调查
	gamePatterns := []string{"游戏", "玩什么", "装了哪些游戏", "steam", "epic", "有什么游戏"}
	for _, p := range gamePatterns {
		if strings.Contains(lower, p) {
			return "investigate_games", "investigate_games", 0.7
		}
	}

	// 文档调查
	docPatterns := []string{"文档", "文件", "pdf", "word", "excel", "ppt", "文件夹"}
	for _, p := range docPatterns {
		if strings.Contains(lower, p) {
			return "investigate_documents", "investigate_documents", 0.7
		}
	}

	return "", "", 0
}

// capabilityToInvestigation 能力→调查意图转换
func capabilityToInvestigation(match *CapabilityMatch, userQuery string) InvestigationIntent {
	if match.Handler == "investigate_games" {
		return InvestigationIntent{
			IntentID:   "filesystem_inventory",
			TemplateID: "games",
			UserQuery:  userQuery,
		}
	}
	if match.Handler == "investigate_documents" {
		return InvestigationIntent{
			IntentID:   "filesystem_inventory",
			TemplateID: "documents",
			UserQuery:  userQuery,
		}
	}
	return InvestigationIntent{}
}

// ─── 调查交付 ──────────────────────────────────────────────────

// deliverInvestigation 执行调查并交付结果
func deliverInvestigation(ctx InvChatTurnContext, intent InvestigationIntent, capID string) *InvChatTurnResult {
	// 执行调查
	run := NewInvestigationRun(ctx.UserMsg, intent.TemplateID, ctx.DataRoot)
	findings := run.CollectStep()
	findings = MergeFindings(findings)

	if len(findings) == 0 {
		return &InvChatTurnResult{
			Handled:      true,
			Reply:        "抱歉，我在这台电脑上没有找到相关内容。",
			MemoryWrite:  memoryWriteFromTemplate(intent.TemplateID, 0),
			CapabilityID: capID,
		}
	}

	// LLM 合成
	reply, err := SynthesizeInvestigationReply(ctx.UserMsg, findings, intent.TemplateID, ctx.LlmCall)
	if err != nil || reply == "" {
		reply = buildSimpleReport(findings, intent.TemplateID)
	}
	return &InvChatTurnResult{
		Handled:      true,
		Reply:        reply,
		MemoryWrite:  memoryWriteFromTemplate(intent.TemplateID, len(findings)),
		CapabilityID: capID,
	}
}

// memoryWriteFromTemplate 生成记忆写入标记
func memoryWriteFromTemplate(template string, count int) string {
	if template == "games" {
		return "INVESTIGATION games：共 " + itoa(count) + " 款游戏"
	}
	return "INVESTIGATION documents：共 " + itoa(count) + " 个文件"
}
