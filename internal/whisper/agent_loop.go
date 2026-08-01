// Package whisper — agent_loop.go
// 100% 对齐 ackem desktop-agent/anthropicAgentLoop.ts + openAiAgentJobRunner.ts
// 通用 Agent 多轮循环：任务计划 → 工具执行 → 结果反馈 → 继续/完成

package whisper

import "fmt"

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

// DefaultAgentConfig 默认配置
func DefaultAgentConfig() AgentLoopConfig {
	return AgentLoopConfig{
		MaxToolRounds:          10,
		MaxInvestigationRounds: 5,
		RequireConfirm:         true,
	}
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

// NewAgentLoop 创建 Agent 循环
func NewAgentLoop(cfg AgentLoopConfig) *AgentLoop {
	return &AgentLoop{Config: cfg}
}

// Start 开始新的 Agent 任务
func (al *AgentLoop) Start(taskDescription string) *AgentTaskPlan {
	al.TaskPlan = ParseTaskPlan(taskDescription)
	al.History = nil
	al.RoundCount = 0
	al.TotalCost = 0
	return al.TaskPlan
}

// Continue 判断是否应继续循环
func (al *AgentLoop) ShouldContinue(lastResult AgentTurnResult) bool {
	if lastResult.TaskCompleted {
		return false
	}
	if al.RoundCount >= al.Config.MaxToolRounds {
		return false
	}
	if !lastResult.ShouldContinue {
		return false
	}
	return len(lastResult.ToolCalls) > 0
}

// RecordTurn 记录一轮结果
func (al *AgentLoop) RecordTurn(result AgentTurnResult) {
	al.History = append(al.History, result)
	al.RoundCount++

	// 更新任务计划进度
	if al.TaskPlan != nil && len(result.ToolResults) > 0 {
		al.TaskPlan.CompletedSteps++
	}
}

// BuildSystemHint 构建系统提示注入
func (al *AgentLoop) BuildSystemHint() string {
	if al.TaskPlan == nil {
		return ""
	}
	hint := fmt.Sprintf(`【Agent 任务计划】
目标：%s
步骤：%d 个 | 已完成：%d 个
当前阶段：%s`,
		al.TaskPlan.Goal,
		al.TaskPlan.TotalSteps,
		al.TaskPlan.CompletedSteps,
		al.TaskPlan.CurrentPhase(),
	)

	if al.RoundCount > 0 {
		hint += fmt.Sprintf("\n已执行 %d 轮工具调用。", al.RoundCount)
	}
	if al.RoundCount >= al.Config.MaxToolRounds-1 {
		hint += "\n⚠ 本轮是最后一轮工具调用。如果任务尚未完成，请在回复中总结当前进展。"
	}

	return hint
}

// BuildToolResultsContext 构建工具结果上下文
func (al *AgentLoop) BuildToolResultsContext(lastResult AgentTurnResult) string {
	if len(lastResult.ToolResults) == 0 {
		return ""
	}
	ctx := "【工具执行结果】\n"
	for _, r := range lastResult.ToolResults {
		status := "✓"
		if !r.Success {
			status = "✗"
		}
		ctx += fmt.Sprintf("%s %s：%s\n", status, r.ToolName, r.Summary)
	}
	return ctx
}

// IsComplete 任务是否完成
func (al *AgentLoop) IsComplete() bool {
	if al.TaskPlan == nil {
		return al.RoundCount > 0
	}
	return al.TaskPlan.CompletedSteps >= al.TaskPlan.TotalSteps
}
