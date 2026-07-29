// Package whisper — task_plan_parse.go
// 100% 对齐 ackem desktop-agent/task-plan/parseTaskPlan.ts
// 从 LLM 输出解析结构化任务计划
package whisper

import "encoding/json"

// TaskPlan 任务计划
type TaskPlan struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Steps     []TaskStep    `json:"steps"`
	CreatedAt string        `json:"createdAt"`
	Status    string        `json:"status"` // active/completed/failed
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

// ParseTaskPlanFromLLM 从 LLM 输出解析任务计划
func ParseTaskPlanFromLLM(raw string) (*TaskPlan, error) {
	// 提取 JSON 块
	jsonStr := extractPlanJSON(raw)
	if jsonStr == "" {
		return nil, nil
	}

	var plan TaskPlan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, err
	}

	// 基本校验
	if plan.Title == "" || len(plan.Steps) == 0 {
		return nil, nil
	}

	// 设置默认值
	for i := range plan.Steps {
		if plan.Steps[i].Status == "" {
			plan.Steps[i].Status = "pending"
		}
		plan.Steps[i].Index = i + 1
	}

	if plan.Status == "" {
		plan.Status = "active"
	}

	return &plan, nil
}

// extractPlanJSON 从 LLM 输出中提取 JSON
func extractPlanJSON(raw string) string {
	// 尝试查找 ```plan-structured 或 ```json 块
	markers := []string{"```plan-structured", "```json", "```"}
	for _, m := range markers {
		start := indexOf(raw, m)
		if start < 0 {
			continue
		}
		start += len(m)
		end := indexOf(raw[start:], "```")
		if end < 0 {
			continue
		}
		return raw[start : start+end]
	}

	// 最后尝试直接找 { }
	start := indexOf(raw, "{")
	end := lastIndexOf(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}

	return ""
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func lastIndexOf(s, sub string) int {
	for i := len(s) - len(sub); i >= 0; i-- {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
