package pantheon

import (
	"fmt"
	"strings"
	"time"
)

// didSeenFromEvents reconstructs consumed decision-informing discoveries after
// resume. The event log remains the single source; no parallel DID counter is
// persisted on the session.
func didSeenFromEvents(events []Event) map[string]bool {
	seen := map[string]bool{}
	for _, event := range events {
		if event.Kind != EventDIDUsed {
			continue
		}
		for _, key := range event.DIDKeys {
			seen[key] = true
		}
	}
	return seen
}

// didAllowsNextRound asks whether another round has at least one executable
// verification that has not already been consumed. A repeated verification is
// not a reason to keep talking; unresolved disagreement alone is also not.
func (m *Manager) didAllowsNextRound(session *Session, round int, seen map[string]bool) bool {
	m.mu.Lock()
	keys := make([]string, 0, 4)
	labels := make([]string, 0, 4)
	for _, issue := range session.Issues {
		if issue.Status != "open" || !issue.Blocking {
			continue
		}
		key := issueDIDKey(issue)
		if key == "" || seen[key] {
			continue
		}
		keys = append(keys, key)
		labels = append(labels, issue.ID)
	}
	now := time.Now().UnixMilli()
	var event Event
	if len(keys) == 0 {
		event = Event{
			Phase: PhaseDiscussion,
			Round: round,
			Kind:  EventDIDStop,
			Text: fmt.Sprintf(
				"第 %d 轮后没有新的可执行核验动作（DID）；停止追加轮次，保留分歧并进入协作汇总。",
				round,
			),
			At: now,
		}
	} else {
		event = Event{
			Phase:   PhaseDiscussion,
			Round:   round,
			Kind:    EventDIDUsed,
			Text:    fmt.Sprintf("第 %d 轮后发现 %d 个新的可执行核验动作（DID），继续讨论：%s", round, len(keys), strings.Join(labels, "、")),
			DIDKeys: keys,
			At:      now,
		}
	}
	event = m.appendEventLocked(session, event)
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
	if len(keys) == 0 {
		return false
	}
	for _, key := range keys {
		seen[key] = true
	}
	return true
}

func issueDIDKey(issue Issue) string {
	verification := normalizeIssueSummary(issue.Verification)
	if !actionableVerification(verification) {
		return ""
	}
	return issue.ID + "|" + verification
}

func actionableVerification(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "无", "无。", "none", "n/a", "-",
		"继续讨论", "再讨论", "待讨论", "再听听意见":
		return false
	}
	return len([]rune(value)) >= 4
}
