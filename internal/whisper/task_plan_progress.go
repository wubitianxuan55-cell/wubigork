// Package whisper — task_plan_progress.go
// 100% 对齐 ackem desktop-agent/task-plan/taskPlanProgress.ts
// 任务计划进度推送
package whisper

import "fmt"

// TaskPlanProgress 任务计划进度
type TaskPlanProgress struct {
	PlanID       string  `json:"planId"`
	TotalSteps   int     `json:"totalSteps"`
	PassedSteps  int     `json:"passedSteps"`
	FailedSteps  int     `json:"failedSteps"`
	CurrentStep  int     `json:"currentStep"`
	CurrentLabel string  `json:"currentLabel"`
	Progress     float64 `json:"progress"` // 0.0 ~ 1.0
}

// ComputeTaskPlanProgress 计算任务计划进度
func ComputeTaskPlanProgress(plan *TaskPlan) TaskPlanProgress {
	progress := TaskPlanProgress{
		PlanID:     plan.ID,
		TotalSteps: len(plan.Steps),
	}

	for i, step := range plan.Steps {
		switch step.Status {
		case "passed":
			progress.PassedSteps++
		case "failed":
			progress.FailedSteps++
		case "in_progress":
			progress.CurrentStep = i + 1
			progress.CurrentLabel = step.Description
		}
	}

	if progress.TotalSteps > 0 {
		progress.Progress = float64(progress.PassedSteps) / float64(progress.TotalSteps)
	}

	return progress
}

// BuildTaskPlanProgressMessage 构建进度消息
func BuildTaskPlanProgressMessage(progress TaskPlanProgress) string {
	return fmt.Sprintf(
		"任务进度：%d/%d 步骤已完成（%.0f%%）",
		progress.PassedSteps,
		progress.TotalSteps,
		progress.Progress*100,
	)
}
