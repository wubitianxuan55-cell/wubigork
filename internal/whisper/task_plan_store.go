// Package whisper — task_plan_store.go
// 100% 对齐 ackem desktop-agent/task-plan/taskPlanStore.ts
// 任务计划持久化（内存版）+ 继续意图检测

package whisper

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// DesktopTaskVerify 验收规则
type DesktopTaskVerify struct {
	Type      string `json:"type"`
	Path      string `json:"path,omitempty"`
	MinBytes  int    `json:"minBytes,omitempty"`
	Substring string `json:"substring,omitempty"`
	Action    string `json:"action,omitempty"`
	Result    string `json:"result,omitempty"`
}

// DesktopTaskStep 桌面任务步骤
type DesktopTaskStep struct {
	ID      string                 `json:"id"`
	Label   string                 `json:"label"`
	Action  string                 `json:"action"`
	Path    string                 `json:"path"`
	Status  string                 `json:"status"`
	Options map[string]interface{} `json:"options,omitempty"`
	Verify  []DesktopTaskVerify    `json:"verify"`
}

// DesktopTaskPlan 桌面助手任务计划
type DesktopTaskPlan struct {
	ID          string            `json:"id"`
	GoalSummary string            `json:"goalSummary"`
	Steps       []DesktopTaskStep `json:"steps"`
	CreatedAt   string            `json:"createdAt"`
}

// DesktopTaskProgress 任务进度
type DesktopTaskProgress struct {
	Plan             DesktopTaskPlan   `json:"plan"`
	CompletedStepIDs []string          `json:"completedStepIds"`
	PendingSteps     []DesktopTaskStep `json:"pendingSteps"`
	AllPassed        bool              `json:"allPassed"`
}

// DesktopPersistedPlan 持久化状态
type DesktopPersistedPlan struct {
	SessionID        string          `json:"sessionId"`
	Plan             DesktopTaskPlan `json:"plan"`
	CompletedStepIDs []string        `json:"completedStepIds"`
	AllPassed        bool            `json:"allPassed"`
	UpdatedAt        string          `json:"updatedAt"`
	Status           string          `json:"status"`
}

// TaskPlanStore 任务计划存储（内存版）
type TaskPlanStore struct {
	mu    sync.RWMutex
	plans map[string]*DesktopPersistedPlan
}

func NewTaskPlanStore() *TaskPlanStore {
	return &TaskPlanStore{plans: make(map[string]*DesktopPersistedPlan)}
}

func (s *TaskPlanStore) Load(sessionID string) *DesktopPersistedPlan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p := s.plans[sessionID]
	if p == nil || p.Status != "active" {
		return nil
	}
	return p
}

func (s *TaskPlanStore) Save(sessionID string, plan DesktopTaskPlan, progress DesktopTaskProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[sessionID] = &DesktopPersistedPlan{
		SessionID:        sessionID,
		Plan:             plan,
		CompletedStepIDs: progress.CompletedStepIDs,
		AllPassed:        progress.AllPassed,
		UpdatedAt:        time.Now().Format(time.RFC3339),
		Status:           map[bool]string{true: "completed", false: "active"}[progress.AllPassed],
	}
}

func (s *TaskPlanStore) Clear(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.plans, sessionID)
}

var continueRE = regexp.MustCompile(`^(继续|接着|接着做|接着来|完成剩余|做完|把.+做完|继续上次|继续刚才|未完成|剩下的|下一步)`)
var continueTaskRE = regexp.MustCompile(`继续.*(任务|步骤|执行|删除|完成)`)

func IsContinueTaskPlanIntent(text string) bool {
	t := strings.TrimSpace(text)
	if t == "" {
		return false
	}
	return continueRE.MatchString(t) || continueTaskRE.MatchString(t)
}

func BuildContinueTaskPlanHint(state *DesktopPersistedPlan, progress DesktopTaskProgress) string {
	var pendingLabels []string
	for _, s := range progress.PendingSteps {
		pendingLabels = append(pendingLabels, s.Label)
	}
	pending := strings.Join(pendingLabels, "；")
	var parts []string
	parts = append(parts,
		"【续做上次未完成的电脑助手任务】",
		"目标："+state.Plan.GoalSummary,
		"已完成 "+itoa(len(progress.CompletedStepIDs))+"/"+itoa(len(state.Plan.Steps))+" 步。",
	)
	if pending != "" {
		parts = append(parts, "待完成："+pending)
	}
	parts = append(parts, "请从下一步继续调用 use_computer，直至全部验收通过。")
	return strings.Join(parts, "\n")
}
