// Package whisper — agent_tool_round.go
// 100% 对齐 ackem desktop-agent/openAiToolRound.ts
// 单轮工具执行：批量工具调用 + 结果汇总

package whisper

import "strings"

// ─── ToolRound ────────────────────────────────────────────────

// ToolRound 单轮工具执行
type ToolRound struct {
	Actions []AgentAction
	Results []AgentToolResult
	Writes  []string // 写入的文件路径
}

// ExecuteToolRound 执行一轮工具调用
func ExecuteToolRound(actions []AgentAction, executor ToolExecutor) *ToolRound {
	round := &ToolRound{Actions: actions}

	for _, act := range actions {
		result := executor.Execute(act)
		round.Results = append(round.Results, result)

		if result.Success && isWriteAction(act.Name) {
			if path, ok := act.Args["path"]; ok {
				round.Writes = append(round.Writes, path)
			}
		}
	}

	return round
}

func isWriteAction(name string) bool {
	writeActions := map[string]bool{
		"write_file": true, "create_file": true, "save_file": true,
		"move_file": true, "delete_file": true, "rename_file": true,
	}
	return writeActions[name]
}

// ─── ToolExecutor 接口 ────────────────────────────────────────

// ToolExecutor 工具执行器接口（由外部实现注入）
type ToolExecutor interface {
	Execute(action AgentAction) AgentToolResult
	ValidateAction(action AgentAction) error
}

// ─── ShouldContinueToolLoop ───────────────────────────────────

// ShouldContinueToolLoop 判断是否应继续工具循环
func ShouldContinueToolLoop(
	round *ToolRound,
	roundCount int,
	maxRounds int,
	plan *AgentTaskPlan,
) (bool, string) {
	// 超过最大轮数
	if roundCount >= maxRounds {
		return false, "达到最大工具轮数"
	}

	// 所有工具都失败了 → 不再继续
	allFailed := true
	for _, r := range round.Results {
		if r.Success {
			allFailed = false
			break
		}
	}
	if allFailed && len(round.Results) > 0 {
		return false, "所有工具执行失败"
	}

	// 任务计划完成
	if plan != nil && plan.CompletedSteps >= plan.TotalSteps {
		return false, "任务计划已完成"
	}

	return true, ""
}

// BuildToolRoundContext 构建工具轮次上下文（注入 LLM）
func BuildToolRoundContext(round *ToolRound) string {
	if round == nil || len(round.Results) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("【上一轮工具执行结果】\n")
	for _, r := range round.Results {
		status := "✓"
		if !r.Success {
			status = "✗"
		}
		sb.WriteString(status + " " + r.ToolName + "：" + r.Summary + "\n")
		if r.Content != "" && len(r.Content) > 200 {
			sb.WriteString("  详情：" + truncStr(r.Content, 200) + "\n")
		}
	}
	return sb.String()
}

// ─── Confirm Gate ─────────────────────────────────────────────

// ConfirmGate 确认门控结果
type ConfirmGate struct {
	RequiresConfirm bool
	Reason          string
	Actions         []AgentAction
}

// EvaluateConfirmGate 评估是否需要确认
func EvaluateConfirmGate(actions []AgentAction, writtenPaths []string) *ConfirmGate {
	gate := &ConfirmGate{}

	for _, act := range actions {
		if isWriteAction(act.Name) {
			gate.RequiresConfirm = true
			gate.Actions = append(gate.Actions, act)
		}
	}

	if gate.RequiresConfirm {
		gate.Reason = "以下操作需要确认："
		for _, act := range gate.Actions {
			gate.Reason += "\n· " + act.Name + " " + act.Args["path"]
		}
	}

	return gate
}
