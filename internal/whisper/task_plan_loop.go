// Package whisper — task_plan_loop.go
// 100% 对齐 ackem desktop-agent/task-plan/taskPlanLoop.ts
// Agent 循环门控：判断是否继续执行任务计划
package whisper

// TaskPlanLoopState 任务计划循环状态
type TaskPlanLoopState struct {
	Round         int  `json:"round"`
	MaxRounds     int  `json:"maxRounds"`
	AllPassed     bool `json:"allPassed"`
	ShouldStop    bool `json:"shouldStop"`
	StopReason    string `json:"stopReason,omitempty"`
}

// CheckTaskPlanLoop 检查是否应继续 Agent 循环
// 返回 (shouldContinue, reason)
func CheckTaskPlanLoop(plan *TaskPlan, round, maxRounds int) TaskPlanLoopState {
	state := TaskPlanLoopState{
		Round:     round,
		MaxRounds: maxRounds,
	}

	// 检查是否所有步骤都已通过
	allPassed := true
	for _, step := range plan.Steps {
		if step.Status != "passed" && step.Status != "failed" {
			allPassed = false
			break
		}
	}

	if allPassed {
		state.AllPassed = true
		state.ShouldStop = true
		state.StopReason = "所有步骤已完成"
		return state
	}

	// 超过最大轮数
	if round >= maxRounds {
		state.ShouldStop = true
		state.StopReason = "达到最大轮数上限"
		return state
	}

	// 检查是否有未完成的步骤
	hasPending := false
	for _, step := range plan.Steps {
		if step.Status == "pending" || step.Status == "in_progress" {
			hasPending = true
			break
		}
	}

	if !hasPending {
		state.ShouldStop = true
		state.StopReason = "无待执行步骤"
		return state
	}

	return state
}

// GetNextPendingStep 获取下一个待执行步骤
func GetNextPendingStep(plan *TaskPlan) *TaskStep {
	for i := range plan.Steps {
		if plan.Steps[i].Status == "pending" {
			plan.Steps[i].Status = "in_progress"
			return &plan.Steps[i]
		}
	}
	return nil
}

// MarkStepComplete 标记步骤完成
func MarkStepComplete(plan *TaskPlan, stepIndex int, passed bool, result string) {
	if stepIndex < 0 || stepIndex >= len(plan.Steps) {
		return
	}
	if passed {
		plan.Steps[stepIndex].Status = "passed"
	} else {
		plan.Steps[stepIndex].Status = "failed"
	}
	plan.Steps[stepIndex].Result = result
}

// BuildTaskPlanNudge 生成任务计划继续提示
func BuildTaskPlanNudge(plan *TaskPlan) string {
	var pending []string
	for _, step := range plan.Steps {
		if step.Status == "pending" || step.Status == "in_progress" {
			pending = append(pending, step.Description)
		}
	}
	if len(pending) == 0 {
		return ""
	}
	nudge := "【任务计划】以下步骤尚未完成，必须继续执行：\n"
	for i, p := range pending {
		nudge += "- " + p
		if i < len(pending)-1 {
			nudge += "\n"
		}
	}
	return nudge
}
