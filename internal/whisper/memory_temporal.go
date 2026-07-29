// Package whisper — memory_temporal.go
// 100% 对齐 ackem memory/temporalContextModulator.ts
// 时间上下文调制：时段/周末/季节影响检索权重

package whisper

import (
	"time"
)

// ─── BuildTemporalContext ─────────────────────────────────────

// BuildTemporalContext 构建时间上下文
func BuildTemporalContext(gapHours float64, now time.Time) TemporalContext {
	h := now.Hour()
	var timeOfDay string
	switch {
	case h >= 23 || h < 5:
		timeOfDay = "late_night"
	case h >= 5 && h < 12:
		timeOfDay = "morning"
	case h >= 12 && h < 18:
		timeOfDay = "afternoon"
	default:
		timeOfDay = "evening"
	}

	w := int(now.Weekday())
	m := int(now.Month())
	var season string
	switch {
	case m == 12 || m <= 2:
		season = "winter"
	case m <= 5:
		season = "spring"
	case m <= 8:
		season = "summer"
	default:
		season = "autumn"
	}

	return TemporalContext{
		TimeOfDay: timeOfDay,
		IsWeekend: w == 0 || w == 6,
		Month:     m,
		Season:    season,
		Hour:      h,
		Weekday:   w,
		GapHours:  gapHours,
		LocalDate: now.Format("2006-01-02"),
	}
}

// ─── ComputeTemporalBoost ─────────────────────────────────────

// ComputeTemporalBoost 计算时间调制因子
func ComputeTemporalBoost(ctx TemporalContext) float64 {
	// 深夜减少注入
	if ctx.TimeOfDay == "late_night" {
		return 0.8
	}
	// 周末更多回忆
	if ctx.IsWeekend {
		return 1.15
	}
	// 冬季温馨回忆
	if ctx.Season == "winter" {
		return 1.1
	}
	return 1.0
}

// ─── ComputeWeekdayMoodBias ───────────────────────────────────

// ComputeWeekdayMoodBias 计算星期情绪偏差
func ComputeWeekdayMoodBias(weekday int) float64 {
	switch weekday {
	case 1: // 周一
		return -2
	case 5, 6: // 周五、周六
		return 2
	default:
		return 0
	}
}

// FormatTimeContextBlock 已迁移至 context_time.go，使用 FormatTimeContextBlockNow()
