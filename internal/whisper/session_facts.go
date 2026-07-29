// Package whisper — session_facts.go
// 100% 对齐 ackem memory/sessionFacts.ts
// 会话事实过滤：构建快照时仅使用当前会话写入的事实，避免跨会话泄漏

package whisper

// FilterFactsForSession 过滤仅属于指定会话的事实
func FilterFactsForSession(facts []*Fact, sessionID string) []*Fact {
	sid := sessionID
	if sid == "" {
		sid = "default"
	}

	var result []*Fact
	for _, f := range facts {
		src := f.SourceSessionID
		if src == "" {
			result = append(result, f)
			continue
		}
		if src == sid {
			result = append(result, f)
		}
	}
	return result
}

// SummariesForSession 获取会话事实摘要列表
func SummariesForSession(facts []*Fact, sessionID string, limit int) []string {
	filtered := FilterFactsForSession(facts, sessionID)
	if limit > len(filtered) {
		limit = len(filtered)
	}
	summaries := make([]string, 0, limit)
	for _, f := range filtered[:limit] {
		summaries = append(summaries, f.Summary)
	}
	return summaries
}
