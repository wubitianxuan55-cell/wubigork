// Package whisper — finalize_new_facts.go
// 100% 对齐 ackem memory/finalizeNewFacts.ts
// 记忆写入收尾：追踪写入结果、通知前端

package whisper

import "sync"

// ─── MemoryWriteTracker ──────────────────────────────────────

type memoryWriteRecord struct {
	TurnIndex       int
	StructuredCount int
	NoteCount       int
}

var (
	lastWriteBySession   = make(map[string]memoryWriteRecord)
	lastWriteBySessionMu sync.RWMutex
)

// RecordMemoryWriteResult 记录记忆写入结果
func RecordMemoryWriteResult(sessionID string, turnIndex int, facts []MemoryFact) {
	structured := 0
	notes := 0
	for _, f := range facts {
		if f.Subcategory == "NOTE" {
			notes++
		} else {
			structured++
		}
	}

	lastWriteBySessionMu.Lock()
	lastWriteBySession[sessionID] = memoryWriteRecord{
		TurnIndex:       turnIndex,
		StructuredCount: structured,
		NoteCount:       notes,
	}
	lastWriteBySessionMu.Unlock()
}

// PeekLastMemoryWrite 查看最近一次记忆写入结果
func PeekLastMemoryWrite(sessionID string) (turnIndex, structuredCount, noteCount int, ok bool) {
	lastWriteBySessionMu.RLock()
	rec, exists := lastWriteBySession[sessionID]
	lastWriteBySessionMu.RUnlock()
	if !exists {
		return 0, 0, 0, false
	}
	return rec.TurnIndex, rec.StructuredCount, rec.NoteCount, true
}

// ─── UserInfoBlock ───────────────────────────────────────────

// BuildUserInfoBlock 组装用户信息块（供 context.go 注入 system prompt）
func BuildUserInfoBlock(fs *FactStore) string {
	if fs == nil {
		return ""
	}
	var lines []string

	// 从事实中提取用户名
	for _, f := range fs.ListActive() {
		if f.Subcategory == "BASIC_PROFILE" && (f.Subject == "用户姓名" || f.Subject == "用户昵称") {
			lines = append(lines, "用户称呼："+f.Summary)
			break
		}
	}

	// 简单档案：取权重最高的 5 条事实作为参考
	top := fs.SelectCoreFacts(5)
	if len(top) > 0 {
		var facts []string
		for _, f := range top {
			facts = append(facts, "· "+f.Summary)
		}
		lines = append(lines, "【关于 ta 的笔记 · 仅供你内心参考】")
		lines = append(lines, joinStrings(facts, "\n"))
	}

	if len(lines) == 0 {
		return ""
	}
	return joinStrings(lines, "\n")
}
