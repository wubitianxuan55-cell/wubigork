// Package whisper — agent_task_plan.go
// 100% 对齐 ackem desktop-agent/task-plan/
// 任务计划：解析、追踪、阶段管理

package whisper

import "strings"

// ─── AgentTaskPlan ────────────────────────────────────────────

// TaskPlanPhase 任务阶段
type TaskPlanPhase string

const (
	PhasePlanning    TaskPlanPhase = "planning"
	PhaseExecuting   TaskPlanPhase = "executing"
	PhaseVerifying   TaskPlanPhase = "verifying"
	PhaseDelivering  TaskPlanPhase = "delivering"
	PhaseCompleted   TaskPlanPhase = "completed"
)

// TaskPlanStep 单个计划步骤
type TaskPlanStep struct {
	Index       int    `json:"index"`
	Description string `json:"description"`
	ToolName    string `json:"toolName"`    // 使用的工具
	Status      string `json:"status"`      // pending/active/done/failed
	Result      string `json:"result"`
}

// AgentTaskPlan Agent 任务计划
type AgentTaskPlan struct {
	Goal            string         `json:"goal"`
	TotalSteps      int            `json:"totalSteps"`
	CompletedSteps  int            `json:"completedSteps"`
	Steps           []TaskPlanStep `json:"steps"`
	Phase           TaskPlanPhase  `json:"phase"`
	PersistID       string         `json:"persistId"`
}

// ─── ParseTaskPlan ────────────────────────────────────────────

// ParseTaskPlan 从任务描述中解析计划
func ParseTaskPlan(description string) *AgentTaskPlan {
	plan := &AgentTaskPlan{
		Goal:  description,
		Phase: PhasePlanning,
	}

	// 简单启发式：按换行或句号拆分步骤
	parts := splitTaskSteps(description)
	if len(parts) <= 1 {
		// 单步骤任务
		plan.TotalSteps = 1
		plan.Steps = []TaskPlanStep{{
			Index:       0,
			Description: description,
			Status:      "pending",
		}}
	} else {
		plan.TotalSteps = len(parts)
		for i, p := range parts {
			plan.Steps = append(plan.Steps, TaskPlanStep{
				Index:       i,
				Description: strings.TrimSpace(p),
				Status:      "pending",
			})
		}
	}

	return plan
}

func splitTaskSteps(desc string) []string {
	// 按"第N步"模式拆分
	var steps []string
	


	// 回退：按句号或分号拆分
	parts := strings.FieldsFunc(desc, func(r rune) bool {
		return r == '。' || r == '；' || r == '\n'
	})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) > 3 {
			steps = append(steps, p)
		}
	}

	if len(steps) == 0 {
		steps = append(steps, desc)
	}
	return steps
}

// ─── Phase Management ─────────────────────────────────────────

// CurrentPhase 当前阶段描述
func (tp *AgentTaskPlan) CurrentPhase() string {
	switch tp.Phase {
	case PhasePlanning:
		return "规划中"
	case PhaseExecuting:
		return "执行中"
	case PhaseVerifying:
		return "验证中"
	case PhaseDelivering:
		return "交付中"
	case PhaseCompleted:
		return "已完成"
	}
	return "未知"
}

// AdvancePhase 推进阶段
func (tp *AgentTaskPlan) AdvancePhase() {
	switch tp.Phase {
	case PhasePlanning:
		tp.Phase = PhaseExecuting
	case PhaseExecuting:
		if tp.CompletedSteps >= tp.TotalSteps {
			tp.Phase = PhaseVerifying
		}
	case PhaseVerifying:
		tp.Phase = PhaseDelivering
	case PhaseDelivering:
		tp.Phase = PhaseCompleted
	}
}

// MarkStepDone 标记步骤完成
func (tp *AgentTaskPlan) MarkStepDone(index int, result string) {
	if index >= 0 && index < len(tp.Steps) {
		tp.Steps[index].Status = "done"
		tp.Steps[index].Result = result
		tp.CompletedSteps = countDoneSteps(tp.Steps)
	}
}

// MarkStepFailed 标记步骤失败
func (tp *AgentTaskPlan) MarkStepFailed(index int, reason string) {
	if index >= 0 && index < len(tp.Steps) {
		tp.Steps[index].Status = "failed"
		tp.Steps[index].Result = reason
	}
}

func countDoneSteps(steps []TaskPlanStep) int {
	n := 0
	for _, s := range steps {
		if s.Status == "done" {
			n++
		}
	}
	return n
}

// ─── Plan Hint ────────────────────────────────────────────────

// BuildTaskPlanInjection 构建任务计划注入文本
func BuildTaskPlanInjection(plan *AgentTaskPlan) string {
	if plan == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("【任务计划】\n")
	sb.WriteString("目标：" + plan.Goal + "\n")
	sb.WriteString("进度：" + itoa(plan.CompletedSteps) + "/" + itoa(plan.TotalSteps) + "\n")

	for _, s := range plan.Steps {
		marker := "☐"
		if s.Status == "done" {
			marker = "☑"
		} else if s.Status == "failed" {
			marker = "☒"
		} else if s.Status == "active" {
			marker = "▶"
		}
		sb.WriteString(marker + " " + s.Description + "\n")
	}
	return sb.String()
}

// HasIncompleteSteps 是否有未完成步骤
func (tp *AgentTaskPlan) HasIncompleteSteps() bool {
	return tp.CompletedSteps < tp.TotalSteps
}

// NextPendingStep 获取下一个待执行步骤
func (tp *AgentTaskPlan) NextPendingStep() *TaskPlanStep {
	for i := range tp.Steps {
		if tp.Steps[i].Status == "pending" || tp.Steps[i].Status == "active" {
			return &tp.Steps[i]
		}
	}
	return nil
}
