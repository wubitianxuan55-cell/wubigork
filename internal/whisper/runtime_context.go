// Package whisper — runtime_context.go
// 100% 对齐 ackem context/runtimeContext.ts
// 运行时上下文构建器：统一入口，聚合用户/陪伴/时间/活动

package whisper

import (
	"time"
)

// BuildRuntimeContextInput 构建运行时上下文输入
type BuildRuntimeContextInput struct {
	SessionID           string
	LastActiveAt        time.Time
	RecentUserExchanges []string
	MemoryFactSummaries []string
	GameActive          bool
	Now                 time.Time
}

// BuildRuntimeContext 统一构建运行时上下文
func BuildRuntimeContext(input BuildRuntimeContextInput) RuntimeContext {
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}

	// 时间上下文
	timeCtx := buildTimeRuntimeContext(now)

	// 用户运行时
	userCtx := ResolveUserRuntimeContext(input.LastActiveAt, input.RecentUserExchanges, now)

	// 用户活动推断
	activity := ResolveUserActivity(ResolveUserActivityInput{
		RecentUserSnippets:  userCtx.RecentUserSnippets,
		MemoryFactSummaries: input.MemoryFactSummaries,
		Time:                timeCtx,
		GameActive:          input.GameActive,
	})

	// 陪伴在场（简化版：总是 active）
	companionCtx := CompanionRuntimeContext{
		Mode:              CompanionActive,
		LastInteractionMs: now.UnixMilli(),
		IdleDurationMs:    0,
	}

	return RuntimeContext{
		CapturedAt: now.Format(time.RFC3339),
		SessionID:  input.SessionID,
		User:       userCtx,
		Companion:  companionCtx,
		Time:       timeCtx,
		Activity:   activity,
	}
}

// buildTimeRuntimeContext 构建时间运行时上下文
func buildTimeRuntimeContext(now time.Time) TimeRuntimeContext {
	h := now.Hour()
	m := now.Minute()
	w := int(now.Weekday())

	var tod string
	switch {
	case h >= 5 && h < 9:
		tod = "morning"
	case h >= 9 && h < 12:
		tod = "forenoon"
	case h >= 12 && h < 14:
		tod = "afternoon"
	case h >= 14 && h < 18:
		tod = "afternoon"
	case h >= 18 && h < 23:
		tod = "evening"
	default:
		tod = "late_night"
	}

	return TimeRuntimeContext{
		LocalDate: now.Format("2006-01-02"),
		LocalTime: now.Format("15:04"),
		TimeOfDay: tod,
		Hour:      h,
		Minute:    m,
		IsWeekend: w == 0 || w == 6,
	}
}
