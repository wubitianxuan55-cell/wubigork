// Package session holds the conversation history for one task — the Session
// type, its serialization helpers, and branch metadata for conversation trees.
package session

import (
	"sync"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// Session holds the conversation history for one task. The run loop (one turn at
// a time) is the only writer, but a frontend can read History/Save from another
// goroutine while a turn appends, so mu guards Messages. Direct Messages reads on
// the run-loop goroutine stay lock-free (serial with its own writes); cross-
// goroutine access goes through Snapshot.
type Session struct {
	mu             sync.RWMutex
	Messages       []provider.Message
	rewriteVersion int // bumped each time the log is rewritten (compact/fold)
	// logSeq 是会话最后消费的事件日志 seq（事件日志模式下恢复/检查点游标）。
	// 旧行为（legacy）下保持 0，不参与任何既有逻辑。
	logSeq int64
}

// New initializes a session with an optional system prompt.
func New(system string) *Session {
	s := &Session{}
	if system != "" {
		s.Messages = append(s.Messages, provider.Message{Role: provider.RoleSystem, Content: system})
	}
	return s
}

// Add appends a message.
func (s *Session) Add(m provider.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, m)
}

// PrependSystem inserts a system message at the front of the message log,
// before any existing messages. Used by sub-agents to inject a cache-stable
// template prefix that same-kind sub-agents can share.
func (s *Session) PrependSystem(content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append([]provider.Message{{Role: provider.RoleSystem, Content: content}}, s.Messages...)
}

// Truncate cuts the message log back to n messages. Used to roll back
// session state on error or cancellation.
func (s *Session) Truncate(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n < len(s.Messages) {
		s.Messages = s.Messages[:n]
	}
}

// Replace swaps the whole message log — used by compaction, which rewrites the
// middle of the history.
func (s *Session) Replace(msgs []provider.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = msgs
}

// Snapshot returns a copy of the messages, safe to read from another goroutine
// while a turn appends. Frontends (History, Save) use it instead of touching the
// live slice.
func (s *Session) Snapshot() []provider.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]provider.Message(nil), s.Messages...)
}

// HasContent returns true when the session carries at least one user,
// assistant, or tool message — i.e. more than just a system prompt. An
// "empty" conversation that has never been used should not be persisted.
func (s *Session) HasContent() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.Messages {
		if m.Role != provider.RoleSystem {
			return true
		}
	}
	return false
}

// RewriteVersion returns the current rewrite version.
func (s *Session) RewriteVersion() int { return s.rewriteVersion }

// ReplayFromLog 把事件日志条目投影为消息并整体替换当前 Messages，
// 同时记录最后消费的 log seq（事件日志模式下「日志 + 投影」薄封装的核心）。
// 旧行为（legacy）下不会调用，Messages 语义不变。
func (s *Session) ReplayFromLog(entries []LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = ProjectMessages(entries)
	s.logSeq = lastLogSeq(entries)
}

// LogSeq 返回会话最后消费的事件日志 seq（0 = 非事件日志模式/未消费）。
func (s *Session) LogSeq() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.logSeq
}

// IncrementRewrite bumps the rewrite version by 1.
func (s *Session) IncrementRewrite() { s.rewriteVersion++ }
