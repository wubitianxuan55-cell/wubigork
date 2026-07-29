// Package whisper — companion_reply_log.go
// 100% 对齐 ackem memory/companionReplyLog.ts
// 伴侣回复日志：按日合并伴侣回复摘要为一条事实

package whisper

import (
	"fmt"
	"strings"
	"time"
)

const companionReplySubjectPrefix = "Ackem回复"
const dailySummaryMaxChars = 2400

func clipStr(s string, max int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// FormatReplyTimestamp 格式化回复时间戳（6月22日21点30分）
func FormatReplyTimestamp(d time.Time) string {
	m := int(d.Month())
	day := d.Day()
	h := d.Hour()
	min := d.Minute()
	if min == 0 {
		return fmt.Sprintf("%d月%d日%d点", m, day, h)
	}
	return fmt.Sprintf("%d月%d日%d点%d分", m, day, h, min)
}

// CompanionReplySubjectForDay 按日历日生成伴侣回复subject
func CompanionReplySubjectForDay(d time.Time) string {
	return companionReplySubjectPrefix + "·" + d.Format("2006-01-02")
}

// FormatCompanionReplyLine 格式化单条伴侣回复行
func FormatCompanionReplyLine(userMsg, assistantText string, now time.Time) string {
	uq := clipStr(userMsg, 48)
	body := clipStr(assistantText, 160)
	return fmt.Sprintf("%s，回复用户「%s」：%s", FormatReplyTimestamp(now), uq, body)
}

// WriteCompanionReplyLog 每轮同步写入伴侣回复摘要（同日合并）
func WriteCompanionReplyLog(store *FactStore, sessionID string, turnIndex int, userMsg, assistantText string) []string {
	now := time.Now()
	line := FormatCompanionReplyLine(userMsg, assistantText, now)
	if line == "" {
		return nil
	}

	subject := CompanionReplySubjectForDay(now)

	// 查找同日已有记录
	var existing *Fact
	for _, f := range store.ListActive() {
		if f.Subcategory == "OUR_BOND" && f.Subject == subject && f.SourceSessionID == sessionID {
			existing = f
			break
		}
	}

	if existing != nil {
		merged := clipStr(existing.Summary+"\n"+line, dailySummaryMaxChars)
		store.UpdateFact(existing.ID, map[string]interface{}{"summary": merged})
		return []string{existing.ID}
	}

	// 新建
	f := store.Add(MemoryFact{
		Domain:          "SOCIAL",
		Subcategory:     "OUR_BOND",
		Subject:         subject,
		Summary:         line,
		Weight:          0.6,
		Confidence:      1.0,
		Triggers:        []string{"Ackem回复"},
		SourceSessionID: sessionID,
		SourceTurnIndex: turnIndex,
		FactLayer:       "raw",
	})
	return []string{f.ID}
}
