// Package whisper — context_time.go
// 对齐 ackem context/localTime.ts
// 统一本地时间工具（消除 orchestrator.go 中的重复实现）

package whisper

import (
	"fmt"
	"time"
)

var weekdayZH = [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

// LocalDateString 本地日期 YYYY-MM-DD
func LocalDateString(t time.Time) string {
	return fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), t.Day())
}

// FormatLocalTime 本地时间 HH:MM
func FormatLocalTime(t time.Time) string {
	return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
}

// FormatLocalWeekdayZH 中文星期
func FormatLocalWeekdayZH(t time.Time) string {
	return weekdayZH[t.Weekday()]
}

// TimeOfDayString 时段描述（中文，用于 prompt 显示）
func TimeOfDayString(t time.Time) string {
	h := t.Hour()
	switch {
	case h >= 5 && h < 9:
		return "清晨"
	case h >= 9 && h < 12:
		return "上午"
	case h >= 12 && h < 14:
		return "中午"
	case h >= 14 && h < 18:
		return "下午"
	case h >= 18 && h < 21:
		return "傍晚"
	case h >= 21 && h < 23:
		return "晚上"
	default:
		return "深夜"
	}
}

// IsWeekend 是否周末
func IsWeekend(t time.Time) bool {
	d := t.Weekday()
	return d == time.Saturday || d == time.Sunday
}

// FormatTimeContextBlock 格式化时间上下文块
func FormatTimeContextBlock(t time.Time) string {
	return fmt.Sprintf("【系统时钟 · 本地】%s %s %s。",
		LocalDateString(t), FormatLocalWeekdayZH(t), TimeOfDayString(t))
}

// FormatTimeContextBlockNow 当前时间便捷版
func FormatTimeContextBlockNow() string {
	return FormatTimeContextBlock(time.Now())
}

// TimeOfDayStringNow 当前时段便捷版
func TimeOfDayStringNow() string {
	return TimeOfDayString(time.Now())
}

// UserAsksLocalClock 检测用户是否在问时间
func UserAsksLocalClock(text string) bool {
	patterns := []string{
		"几点", "几时", "什么时间", "现在时间", "当前时间",
		"今天几号", "今天日期", "几月几号", "星期几",
	}
	for _, p := range patterns {
		if len(text) >= len(p) {
			for i := 0; i <= len(text)-len(p); i++ {
				if text[i:i+len(p)] == p {
					return true
				}
			}
		}
	}
	return false
}
