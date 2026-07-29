// Package whisper — temporal_date.go
// 100% 对齐 ackem engine/temporalAwareness/specialDateDetector.ts
// 特殊日期检测：生日、纪念日、节日

package whisper

import (
	"time"
)

// ─── SpecialDate 特殊日期 ─────────────────────────────────────

// SpecialDate 特殊日期信息
type SpecialDate struct {
	Label     string // 日期标签（如 "你的生日"、"我们认识的第100天"）
	Priority  string // high/medium/low
	DateLabel string // 可读日期（如 "3月15日"）
}

// TemporalSignal 时间信号
type TemporalSignal struct {
	TemporalHint *struct {
		DateLabel string
		Priority  string
	}
	SpecialDates []SpecialDate
}

// ─── 检测 ──────────────────────────────────────────────────────

// DetectSpecialDates 检测特殊日期
func DetectSpecialDates(
	now time.Time,
	firstMetDate *time.Time,
	ackemBirthday *time.Time,
) *TemporalSignal {
	signal := &TemporalSignal{}

	// 纪念日：每满100天
	if firstMetDate != nil {
		days := int(now.Sub(*firstMetDate).Hours() / 24)
		if days > 0 && days%100 == 0 {
			signal.SpecialDates = append(signal.SpecialDates, SpecialDate{
				Label:     "认识的第" + itoa(days) + "天",
				Priority:  "high",
				DateLabel: now.Format("1月2日"),
			})
			signal.TemporalHint = &struct {
				DateLabel string
				Priority  string
			}{DateLabel: "认识的第" + itoa(days) + "天", Priority: "high"}
		}
		// 每月纪念日
		if now.Day() == firstMetDate.Day() && days >= 30 {
			months := days / 30
			signal.SpecialDates = append(signal.SpecialDates, SpecialDate{
				Label:     "认识的第" + itoa(months) + "个月",
				Priority:  "medium",
				DateLabel: now.Format("1月2日"),
			})
		}
	}

	// AI 生日
	if ackemBirthday != nil && now.Month() == ackemBirthday.Month() && now.Day() == ackemBirthday.Day() {
		signal.SpecialDates = append(signal.SpecialDates, SpecialDate{
			Label:     "我的生日",
			Priority:  "high",
			DateLabel: now.Format("1月2日"),
		})
		signal.TemporalHint = &struct {
			DateLabel string
			Priority  string
		}{DateLabel: "我的生日", Priority: "high"}
	}

	return signal
}

// DetectFastSpecialDateType 快速检测特殊日期类型（无需复杂逻辑）
func DetectFastSpecialDateType(now time.Time, firstMetDate *time.Time) string {
	if firstMetDate == nil {
		return ""
	}
	days := int(now.Sub(*firstMetDate).Hours() / 24)
	if days > 0 && days%100 == 0 {
		return "centennial"
	}
	if now.Day() == firstMetDate.Day() {
		return "monthly"
	}
	return ""
}

// ─── 季节检测 ──────────────────────────────────────────────────

// DetectSeason 检测当前季节
func DetectSeason(now time.Time) string {
	m := int(now.Month())
	switch {
	case m == 12 || m <= 2:
		return "winter"
	case m <= 5:
		return "spring"
	case m <= 8:
		return "summer"
	default:
		return "autumn"
	}
}

// IsSpecialSeason 是否为特殊季节
func IsSpecialSeason(season string) bool {
	return season == "winter" || season == "spring"
}
