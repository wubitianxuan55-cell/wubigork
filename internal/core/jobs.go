// Package core — 会话任务并发管理（自 internal/office/job_manager.go 平迁，逐行等价）。
package core

import (
	"context"
	"sync"
)

type AgentJob struct {
	SessionID string
	Cancel    context.CancelFunc
	State     AgentJobState
}

type JobManager struct {
	mu      sync.RWMutex
	jobs    map[string]*AgentJob
	onState func(state AgentJobState)
}

func NewJobManager(onState func(AgentJobState)) *JobManager {
	return &JobManager{jobs: make(map[string]*AgentJob), onState: onState}
}

func (m *JobManager) Cancel(sessionID string) {
	m.mu.Lock(); defer m.mu.Unlock()
	if job, ok := m.jobs[sessionID]; ok { job.State.Phase = "cancelled"; job.State.Active = false; job.Cancel(); m.emit(sessionID) }
}

func (m *JobManager) GetState(sessionID string) *AgentJobState {
	m.mu.RLock(); defer m.mu.RUnlock()
	if job, ok := m.jobs[sessionID]; ok { s := job.State; return &s }
	return nil
}

func (m *JobManager) emit(sessionID string) {
	if m.onState == nil { return }
	if job, ok := m.jobs[sessionID]; ok { m.onState(job.State) }
}
