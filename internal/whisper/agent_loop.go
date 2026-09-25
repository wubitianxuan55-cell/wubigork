// Package whisper — agent_loop.go
// 100% 对齐 ackem desktop-agent/anthropicAgentLoop.ts + openAiAgentJobRunner.ts
// 通用 Agent 多轮循环：任务计划 → 工具执行 → 结果反馈 → 继续/完成

package whisper

// ─── AgentLoop 类型 ───────────────────────────────────────────

// AgentAction Agent 可执行的动作
type AgentAction struct {
	Name   string            `json:"name"`
	Args   map[string]string `json:"args"`
	Reason string            `json:"reason"`
}

// AgentToolResult 工具执行结果
type AgentToolResult struct {
	ToolName string `json:"toolName"`
	Success  bool   `json:"success"`
	Content  string `json:"content"`
	Summary  string `json:"summary"`
}

// AgentTurnResult 单轮 Agent 结果
type AgentTurnResult struct {
	AssistantText  string            `json:"assistantText"`
	ToolCalls      []AgentAction     `json:"toolCalls"`
	ToolResults    []AgentToolResult `json:"toolResults"`
	ShouldContinue bool              `json:"shouldContinue"`
	TaskCompleted  bool              `json:"taskCompleted"`
}

// AgentLoopConfig Agent 循环配置
type AgentLoopConfig struct {
	MaxToolRounds          int  `json:"maxToolRounds"`          // 最大工具轮数，默认 10
	MaxInvestigationRounds int  `json:"maxInvestigationRounds"` // 调查最大轮数
	RequireConfirm         bool `json:"requireConfirm"`         // 写操作需要确认
}


// ─── AgentLoop ────────────────────────────────────────────────

// AgentLoop 通用 Agent 多轮循环
type AgentLoop struct {
	Config     AgentLoopConfig
	TaskPlan   *AgentTaskPlan
	History    []AgentTurnResult
	RoundCount int
	TotalCost  int // token 估算
}







