// Package whisper — session_mode.go
// 100% 对齐 ackem desktop-agent/sessionMode.ts
// 桌面助手会话模式开关：持久化 + 读写

package whisper

import "sync"

// SessionModeStore 会话模式存储（内存版，Go侧无文件IO）
type SessionModeStore struct {
	mu   sync.RWMutex
	modes map[string]bool // sessionID → enabled
}

// NewSessionModeStore 创建会话模式存储
func NewSessionModeStore() *SessionModeStore {
	return &SessionModeStore{modes: make(map[string]bool)}
}

// GetDesktopAgentChatMode 获取会话的桌面助手模式
func (s *SessionModeStore) GetDesktopAgentChatMode(sessionID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modes[sessionID]
}

// SetDesktopAgentChatMode 设置会话的桌面助手模式
func (s *SessionModeStore) SetDesktopAgentChatMode(sessionID string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if enabled {
		s.modes[sessionID] = true
	} else {
		delete(s.modes, sessionID)
	}
}

// ClearAllSessions 清除所有会话模式
func (s *SessionModeStore) ClearAllSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modes = make(map[string]bool)
}
