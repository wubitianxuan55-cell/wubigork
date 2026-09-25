// Package core — 会话模式内存态（自 internal/office/session_mode.go 平迁，逐行等价）。
package core

import "sync"

type SessionModeStore struct {
	mu    sync.RWMutex
	modes map[string]bool
}

func NewSessionModeStore() *SessionModeStore {
	return &SessionModeStore{modes: make(map[string]bool)}
}

func (s *SessionModeStore) GetMode(sessionID string) bool {
	s.mu.RLock(); defer s.mu.RUnlock()
	return s.modes[sessionID]
}

func (s *SessionModeStore) SetMode(sessionID string, enabled bool) {
	s.mu.Lock(); defer s.mu.Unlock()
	if enabled { s.modes[sessionID] = true } else { delete(s.modes, sessionID) }
}
