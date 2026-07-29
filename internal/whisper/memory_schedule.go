// Package whisper — memory_schedule.go
// 100% 对齐 ackem memory/scheduler.ts
// 检索调度器：计算 RelevanceHint

package whisper

import "math"

// ─── ComputeRelevanceHint ─────────────────────────────────────

// ComputeRelevanceHint 计算检索方向提示
func ComputeRelevanceHint(l1 L1State, emotion EmotionState, turnIndex int) RelevanceHint {
	var sm float64
	switch l1.Stage {
	case StageIntimate:
		sm = 1.3
	case StageFamiliar:
		sm = 1.0
	default:
		sm = 0.6
	}

	tt := "low"
	if l1.Trust > 70 {
		tt = "high"
	} else if l1.Trust > 40 {
		tt = "building"
	}

	return RelevanceHint{
		StageMultiplier: sm,
		AroVolatility:   math.Abs(emotion.Aro) / 100,
		TrustTrajectory: tt,
	}
}
