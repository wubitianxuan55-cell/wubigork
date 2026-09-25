// Package whisper — task_plan_parse.go
// 100% 对齐 ackem desktop-agent/task-plan/parseTaskPlan.ts
// 从 LLM 输出解析结构化任务计划
package whisper

import (
)

// TaskPlan 任务计划
type TaskPlan struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Steps     []TaskStep `json:"steps"`
	CreatedAt string     `json:"createdAt"`
	Status    string     `json:"status"` // active/completed/failed
}

// TaskStep 任务步骤
type TaskStep struct {
	Index       int    `json:"index"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Path        string `json:"path,omitempty"`
	Target      string `json:"target,omitempty"`
	Status      string `json:"status"` // pending/in_progress/passed/failed
	Result      string `json:"result,omitempty"`
}




