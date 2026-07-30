// Package office — session_mode.go
package office

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

func (s *SessionModeStore) ClearAll() {
	s.mu.Lock(); defer s.mu.Unlock()
	s.modes = make(map[string]bool)
}
