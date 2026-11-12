// Package whisper — 工作记忆（对齐 ackem memory/workingMemory.ts）
package whisper

import (
	"strings"
	"sync"

	"github.com/gaea/gaea/internal/gaea/strutil"
)

// ─── Exchange 对话交换 ────────────────────────────────────────

// Exchange 来源标记（众神殿来源字段契约，进度计划/
// gaea-chat-persona-optimization-20261105.md §三.3）：新写入恒为 SourceRaw
// （双通道刀3 后进 WM 的 userText 已是用户原文）；存量旧行无此字段
// （空=遗留形态，读取面按「无标记+含动态包裹标记=脏行」口径处理）。
// 随 rows_json 落 hermes.db chat_history，零 schema 迁移、旧数据零改写。
const ExchangeSourceRaw = "raw"

type Exchange struct {
	TurnIndex     int
	UserText      string
	AssistantText string
	Source        string `json:"source,omitempty"`
}

// ─── WorkingMemory 工作记忆缓冲区 ─────────────────────────────

// WorkingMemory 工作记忆缓冲区。T7-1.1：异步持久化协程与主流程
// 并发读写同一 map，必须加锁（Go 并发 map 读写直接 panic）。
type WorkingMemory struct {
	mu       sync.RWMutex
	sessions map[string][]Exchange
}

func NewWorkingMemory() *WorkingMemory {
	return &WorkingMemory{sessions: make(map[string][]Exchange)}
}

// forSession 返回会话缓冲区（只读，不初始化）。调用方必须已持有 mu（读或写锁）。
// 不存在时返回 nil：Push 对 nil append 会自动建新切片；GetRecent/GetAll 对 nil 返回空。
// 注意：不得在只读路径（RLock）里惰性写 map，否则并发读会触发写-写数据竞争。
func (wm *WorkingMemory) forSession(sessionID string) []Exchange {
	return wm.sessions[sessionID]
}

func (wm *WorkingMemory) Push(sessionID string, ex Exchange) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	buf := wm.forSession(sessionID)
	buf = append(buf, ex)
	if len(buf) > WorkingMemoryMaxExchanges*2 {
		buf = buf[len(buf)-WorkingMemoryMaxExchanges:]
	}
	wm.sessions[sessionID] = buf
}

// GetRecent 返回最近 N 轮对话（拷贝，调用方可安全持有）。
func (wm *WorkingMemory) GetRecent(sessionID string) []Exchange {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	buf := wm.forSession(sessionID)
	start := len(buf) - WorkingMemoryMaxExchanges
	if start < 0 {
		start = 0
	}
	out := make([]Exchange, len(buf)-start)
	copy(out, buf[start:])
	return out
}

// GetAll 返回所有对话（用于持久化，拷贝）。
func (wm *WorkingMemory) GetAll(sessionID string) []Exchange {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
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
		userLine := strutil.TruncateRunesSuffix(ex.UserText, 200, "…")
		asstLine := strutil.TruncateRunesSuffix(ex.AssistantText, 200, "…")
		block := "用户：" + userLine + "\ngaea：" + asstLine
		if chars+len([]rune(block)) > WorkingMemoryCharBudget {
			break
		}
		lines = append(lines, block)
		chars += len([]rune(block)) + 2
	}
	return strings.Join(lines, "\n")
}

func (wm *WorkingMemory) Clear(sessionID string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	delete(wm.sessions, sessionID)
}
