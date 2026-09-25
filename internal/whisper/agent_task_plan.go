// Package whisper — agent_task_plan.go
// 100% 对齐 ackem desktop-agent/task-plan/
// 任务计划：解析、追踪、阶段管理

package whisper

// ─── AgentTaskPlan ────────────────────────────────────────────

// TaskPlanPhase 任务阶段
type TaskPlanPhase string

const (
	PhasePlanning   TaskPlanPhase = "planning"
	PhaseExecuting  TaskPlanPhase = "executing"
	PhaseVerifying  TaskPlanPhase = "verifying"
	PhaseDelivering TaskPlanPhase = "delivering"
	PhaseCompleted  TaskPlanPhase = "completed"
)

// TaskPlanStep 单个计划步骤
type TaskPlanStep struct {
	Index       int    `json:"index"`
	Description string `json:"description"`
	ToolName    string `json:"toolName"` // 使用的工具
	Status      string `json:"status"`   // pending/active/done/failed
	Result      string `json:"result"`
}

// AgentTaskPlan Agent 任务计划
type AgentTaskPlan struct {
	Goal           string         `json:"goal"`
	TotalSteps     int            `json:"totalSteps"`
	CompletedSteps int            `json:"completedSteps"`
	Steps          []TaskPlanStep `json:"steps"`
	Phase          TaskPlanPhase  `json:"phase"`
	PersistID      string         `json:"persistId"`
}

// ─── ParseTaskPlan ────────────────────────────────────────────



// ─── Phase Management ─────────────────────────────────────────






// ─── Plan Hint ────────────────────────────────────────────────



