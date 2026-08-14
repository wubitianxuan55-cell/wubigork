// Package whisper — task_plan_store.go
// 100% 对齐 ackem desktop-agent/task-plan/taskPlanStore.ts
// 任务计划存储（内存版 + 文件持久化）+ 继续意图检测
//
// 持久化（T5-4b 轻语任务计划持久化）：
//   - 活动任务计划（status=active）落盘到 whisper_data/task_plan.json（原子写）；
//   - 状态变化（Save）时更新；完成/取消（AllPassed 或 Clear）时清除；
//   - 重启后由 ReloadFromDisk 加载回内存（whisper 模块初始化处调用）。

package whisper

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileutil"
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

// taskPlanFileName 活动任务计划落盘文件名（位于 whisper_data 下）
const taskPlanFileName = "task_plan.json"

// persistedPlansFile task_plan.json 文件结构：仅保存活动（未完成）计划。
// 数组包一层，多会话并存时可全部恢复，单会话场景行为一致。
type persistedPlansFile struct {
	Plans []DesktopPersistedPlan `json:"plans"`
}

// TaskPlanStore 任务计划存储（内存版 + 文件持久化）
type TaskPlanStore struct {
	mu       sync.RWMutex
	plans    map[string]*DesktopPersistedPlan
	dataRoot string // 轻语数据根目录（空=纯内存模式，不落盘）
}

func NewTaskPlanStore() *TaskPlanStore {
	return &TaskPlanStore{plans: make(map[string]*DesktopPersistedPlan)}
}

// NewTaskPlanStoreWithDataRoot 创建带数据根目录的任务计划存储（Save/Clear 会原子落盘）
func NewTaskPlanStoreWithDataRoot(dataRoot string) *TaskPlanStore {
	return &TaskPlanStore{plans: make(map[string]*DesktopPersistedPlan), dataRoot: dataRoot}
}

// SetDataRoot 设置数据根目录（可在创建后补充；空目录回到纯内存模式）
func (s *TaskPlanStore) SetDataRoot(dataRoot string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dataRoot = dataRoot
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

// Save 保存任务计划与进度：内存更新后原子落盘（完成态不落盘，磁盘上仅保留活动计划）。
func (s *TaskPlanStore) Save(sessionID string, plan DesktopTaskPlan, progress DesktopTaskProgress) {
	s.mu.Lock()
	s.plans[sessionID] = &DesktopPersistedPlan{
		SessionID:        sessionID,
		Plan:             plan,
		CompletedStepIDs: progress.CompletedStepIDs,
		AllPassed:        progress.AllPassed,
		UpdatedAt:        time.Now().Format(time.RFC3339),
		Status:           map[bool]string{true: "completed", false: "active"}[progress.AllPassed],
	}
	s.mu.Unlock()
	if err := s.persist(); err != nil {
		slog.Error("轻语任务计划落盘失败", "sessionID", sessionID, "error", err)
	}
}

// Clear 清除指定会话的任务计划（内存删除 + 磁盘同步清除）
func (s *TaskPlanStore) Clear(sessionID string) {
	s.mu.Lock()
	delete(s.plans, sessionID)
	s.mu.Unlock()
	if err := s.persist(); err != nil {
		slog.Error("轻语任务计划清除落盘失败", "sessionID", sessionID, "error", err)
	}
}

// ActivePlan 返回当前活动任务计划（多个会话并存时取最近更新者）；无则 nil。
func (s *TaskPlanStore) ActivePlan() *DesktopPersistedPlan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var best *DesktopPersistedPlan
	for _, p := range s.plans {
		if p == nil || p.Status != "active" {
			continue
		}
		if best == nil || p.UpdatedAt > best.UpdatedAt {
			best = p
		}
	}
	return best
}

// Resume 恢复入口：把指定会话的活动计划重新标记为进行中（刷新 updatedAt 并落盘）。
// 返回是否存在该活动计划（即是否成功恢复）。
func (s *TaskPlanStore) Resume(sessionID string) bool {
	s.mu.Lock()
	p := s.plans[sessionID]
	if p == nil || p.Status != "active" {
		s.mu.Unlock()
		return false
	}
	p.UpdatedAt = time.Now().Format(time.RFC3339)
	s.mu.Unlock()
	if err := s.persist(); err != nil {
		slog.Error("轻语任务计划恢复落盘失败", "sessionID", sessionID, "error", err)
	}
	return true
}

// ReloadFromDisk 从磁盘加载上次未完成的任务计划回内存（whisper 模块初始化处调用）。
// 文件不存在视为无遗留计划，不报错。
func (s *TaskPlanStore) ReloadFromDisk() error {
	if s.dataRoot == "" {
		return nil
	}
	data, err := os.ReadFile(s.persistPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var file persistedPlansFile
	if err := json.Unmarshal(data, &file); err != nil {
		slog.Warn("轻语任务计划文件解析失败（忽略遗留计划）", "path", s.persistPath(), "error", err)
		return nil
	}
	s.mu.Lock()
	for i := range file.Plans {
		p := file.Plans[i]
		if p.SessionID == "" || p.Status != "active" {
			continue
		}
		s.plans[p.SessionID] = &p
	}
	s.mu.Unlock()
	if len(file.Plans) > 0 {
		slog.Info("轻语任务计划已从磁盘恢复", "count", len(file.Plans))
	}
	return nil
}

// persist 把当前全部活动计划原子写盘；无活动计划时删除文件（完成/取消清除）。
func (s *TaskPlanStore) persist() error {
	s.mu.RLock()
	active := make([]DesktopPersistedPlan, 0, len(s.plans))
	for _, p := range s.plans {
		if p != nil && p.Status == "active" {
			active = append(active, *p)
		}
	}
	s.mu.RUnlock()
	return s.writeFile(active)
}

// writeFile 原子写盘（复用 fileutil.AtomicWrite：临时文件 + rename，崩溃不留半截文件）
func (s *TaskPlanStore) writeFile(plans []DesktopPersistedPlan) error {
	if s.dataRoot == "" {
		return nil // 纯内存模式
	}
	path := s.persistPath()
	if len(plans) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	data, err := json.MarshalIndent(persistedPlansFile{Plans: plans}, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.AtomicWrite(path, data, 0o644)
}

// persistPath 任务计划文件路径
func (s *TaskPlanStore) persistPath() string {
	return filepath.Join(s.dataRoot, taskPlanFileName)
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
