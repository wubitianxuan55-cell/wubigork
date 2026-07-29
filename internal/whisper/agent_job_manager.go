// Package whisper — agent_job_manager.go
// 100% 对齐 ackem desktop-agent/agentJobManager.ts
// 后台 Agent 任务管理器：生命周期 + 状态广播
package whisper

import (
	"context"
	"sync"
)

// AgentJobState 任务状态
type AgentJobState struct {
	SessionID string `json:"sessionId"`
	Phase     string `json:"phase"` // pending/waiting_confirm/executing/completed/failed/cancelled
	Label     string `json:"label"`
	Active    bool   `json:"active"`
	Error     string `json:"error,omitempty"`
}

// AgentJob 后台任务
type AgentJob struct {
	SessionID string
	Cancel    context.CancelFunc
	State     AgentJobState
}

// AgentJobManager 后台任务管理器
type AgentJobManager struct {
	mu       sync.RWMutex
	jobs     map[string]*AgentJob // sessionID → job
	onState  func(state AgentJobState) // 状态变更回调（供 Wails 前端绑定）
}

// NewAgentJobManager 创建任务管理器
func NewAgentJobManager(onState func(AgentJobState)) *AgentJobManager {
	return &AgentJobManager{
		jobs:    make(map[string]*AgentJob),
		onState: onState,
	}
}

// StartJob 启动后台任务
func (m *AgentJobManager) StartJob(sessionID string, label string) (context.Context, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[sessionID]; ok && job.State.Active {
		return nil, false // 已有活跃任务
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.jobs[sessionID] = &AgentJob{
		SessionID: sessionID,
		Cancel:    cancel,
		State: AgentJobState{
			SessionID: sessionID,
			Phase:     "executing",
			Label:     label,
			Active:    true,
		},
	}

	m.emit(sessionID)
	return ctx, true
}

// CancelJob 取消后台任务
func (m *AgentJobManager) CancelJob(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[sessionID]; ok {
		job.State.Phase = "cancelled"
		job.State.Active = false
		job.Cancel()
		m.emit(sessionID)
	}
}

// CompleteJob 完成任务
func (m *AgentJobManager) CompleteJob(sessionID string, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[sessionID]; ok {
		if errMsg != "" {
			job.State.Phase = "failed"
			job.State.Error = errMsg
		} else {
			job.State.Phase = "completed"
		}
		job.State.Active = false
		m.emit(sessionID)
	}
}

// UpdateJobPhase 更新任务阶段
func (m *AgentJobManager) UpdateJobPhase(sessionID string, phase, label string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[sessionID]; ok {
		job.State.Phase = phase
		job.State.Label = label
		m.emit(sessionID)
	}
}

// IsJobRunning 检查是否有活跃任务
func (m *AgentJobManager) IsJobRunning(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[sessionID]
	return ok && job.State.Active
}

// GetJobState 获取任务状态
func (m *AgentJobManager) GetJobState(sessionID string) *AgentJobState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if job, ok := m.jobs[sessionID]; ok {
		state := job.State
		return &state
	}
	return nil
}

// CleanupSession 清理指定会话的所有任务
func (m *AgentJobManager) CleanupSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.jobs[sessionID]; ok && job.State.Active {
		job.Cancel()
	}
	delete(m.jobs, sessionID)
}

// emit 发送状态变更
func (m *AgentJobManager) emit(sessionID string) {
	if m.onState == nil {
		return
	}
	if job, ok := m.jobs[sessionID]; ok {
		m.onState(job.State)
	}
}
