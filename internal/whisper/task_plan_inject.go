// Package whisper — task_plan_inject.go
// 100% 对齐 ackem desktop-agent/task-plan/injectTaskPlan.ts
// 任务计划注入 system prompt
package whisper

import "fmt"

// InjectTaskPlanToSystemPrompt 将任务计划注入 system prompt
func InjectTaskPlanToSystemPrompt(basePrompt string, plan *TaskPlan) string {
	if plan == nil || len(plan.Steps) == 0 {
		return basePrompt
	}

	planBlock := "\n\n【当前任务计划】\n"
	planBlock += fmt.Sprintf("标题：%s\n", plan.Title)
	for _, step := range plan.Steps {
		statusIcon := "⬜"
		switch step.Status {
		case "passed":
			statusIcon = "✅"
		case "failed":
			statusIcon = "❌"
		case "in_progress":
			statusIcon = "🔄"
		}
		planBlock += fmt.Sprintf("%s 步骤 %d：%s", statusIcon, step.Index, step.Description)
		if step.Action != "" {
			planBlock += fmt.Sprintf(" [%s]", step.Action)
		}
		if step.Path != "" {
			planBlock += fmt.Sprintf(" → %s", step.Path)
		}
		planBlock += "\n"
	}
	planBlock += "规则：严格按照以上步骤顺序执行，每步完成后使用 use_computer 验证结果。全部完成后验收。\n"

	return basePrompt + planBlock
}
