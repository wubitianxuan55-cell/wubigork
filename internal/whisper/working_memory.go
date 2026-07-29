// Package whisper — 工作记忆（对齐 ackem memory/workingMemory.ts）
package whisper

import "strings"

// ─── Exchange 对话交换 ────────────────────────────────────────

type Exchange struct {
	TurnIndex     int
	UserText      string
	AssistantText string
}

// ─── WorkingMemory 工作记忆缓冲区 ─────────────────────────────

type WorkingMemory struct {
	sessions map[string][]Exchange
}

func NewWorkingMemory() *WorkingMemory {
	return &WorkingMemory{sessions: make(map[string][]Exchange)}
}

func (wm *WorkingMemory) forSession(sessionID string) []Exchange {
	buf := wm.sessions[sessionID]
	if buf == nil {
		buf = []Exchange{}
		wm.sessions[sessionID] = buf
	}
	return buf
}

func (wm *WorkingMemory) Push(sessionID string, ex Exchange) {
	buf := wm.forSession(sessionID)
	buf = append(buf, ex)
	if len(buf) > WorkingMemoryMaxExchanges*2 {
		buf = buf[len(buf)-WorkingMemoryMaxExchanges:]
	}
	wm.sessions[sessionID] = buf
}

// GetRecent 返回最近 N 轮对话
func (wm *WorkingMemory) GetRecent(sessionID string) []Exchange {
	buf := wm.forSession(sessionID)
	start := len(buf) - WorkingMemoryMaxExchanges
	if start < 0 {
		start = 0
	}
	return buf[start:]
}

// GetAll 返回所有对话（用于持久化）
func (wm *WorkingMemory) GetAll(sessionID string) []Exchange {
	buf := wm.forSession(sessionID)
	out := make([]Exchange, len(buf))
	copy(out, buf)
	return out
}

func (wm *WorkingMemory) BuildContextBlock(sessionID string) string {
	recent := wm.GetRecent(sessionID)
	if len(recent) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "【近期对话上下文（最近几轮）】")
	chars := 0
	for _, ex := range recent {
		userLine := truncateString(ex.UserText, 200)
		asstLine := truncateString(ex.AssistantText, 200)
		block := "用户：" + userLine + "\n伴侣：" + asstLine
		if chars+len([]rune(block)) > WorkingMemoryCharBudget {
			break
		}
		lines = append(lines, block)
		chars += len([]rune(block)) + 2
	}
	return strings.Join(lines, "\n")
}

func (wm *WorkingMemory) Clear(sessionID string) {
	delete(wm.sessions, sessionID)
}

func truncateString(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
