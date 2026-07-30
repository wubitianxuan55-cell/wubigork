// Package office — job_manager.go
package office

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

func (m *JobManager) Start(sessionID string, label string) (context.Context, bool) {
	m.mu.Lock(); defer m.mu.Unlock()
	if job, ok := m.jobs[sessionID]; ok && job.State.Active { return nil, false }
	ctx, cancel := context.WithCancel(context.Background())
	m.jobs[sessionID] = &AgentJob{SessionID: sessionID, Cancel: cancel, State: AgentJobState{SessionID: sessionID, Phase: "executing", Label: label, Active: true}}
	m.emit(sessionID)
	return ctx, true
}

func (m *JobManager) Cancel(sessionID string) {
	m.mu.Lock(); defer m.mu.Unlock()
	if job, ok := m.jobs[sessionID]; ok { job.State.Phase = "cancelled"; job.State.Active = false; job.Cancel(); m.emit(sessionID) }
}

func (m *JobManager) Complete(sessionID string, errMsg string) {
	m.mu.Lock(); defer m.mu.Unlock()
	if job, ok := m.jobs[sessionID]; ok {
		if errMsg != "" { job.State.Phase = "failed"; job.State.Error = errMsg } else { job.State.Phase = "completed" }
		job.State.Active = false; m.emit(sessionID)
	}
}

func (m *JobManager) UpdatePhase(sessionID string, phase, label string) {
	m.mu.Lock(); defer m.mu.Unlock()
	if job, ok := m.jobs[sessionID]; ok { job.State.Phase = phase; job.State.Label = label; m.emit(sessionID) }
}

func (m *JobManager) IsRunning(sessionID string) bool { m.mu.RLock(); defer m.mu.RUnlock(); job, ok := m.jobs[sessionID]; return ok && job.State.Active }

func (m *JobManager) GetState(sessionID string) *AgentJobState {
	m.mu.RLock(); defer m.mu.RUnlock()
	if job, ok := m.jobs[sessionID]; ok { s := job.State; return &s }
	return nil
}

func (m *JobManager) Cleanup(sessionID string) {
	m.mu.Lock(); defer m.mu.Unlock()
	if job, ok := m.jobs[sessionID]; ok && job.State.Active { job.Cancel() }
	delete(m.jobs, sessionID)
}

func (m *JobManager) emit(sessionID string) {
	if m.onState == nil { return }
	if job, ok := m.jobs[sessionID]; ok { m.onState(job.State) }
}
