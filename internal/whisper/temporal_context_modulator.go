// Package whisper — temporal_context_modulator.go
// 100% 对齐 ackem memory/temporalContextModulator.ts (特殊日期情绪偏移部分)
// 特殊日期情绪偏移——覆盖周日曲线
// 基础时间感知功能（BuildTemporalContext/ComputeTemporalBoost/ComputeWeekdayMoodBias）见 memory_temporal.go

package whisper

// ComputeSpecialDateMoodBias 特殊日期的情绪偏移——覆盖周日曲线
func ComputeSpecialDateMoodBias(specialType string) (affDelta, secDelta float64) {
	switch specialType {
	case "ackem_birthday":
		return +3.0, +1.5
	case "birthday":
		return +3.0, +1.0
	case "first_met_anniversary", "relationship":
		return +2.0, +0.5
	case "holiday_spring":
		return +1.5, +0.3
	case "holiday_valentine":
		return +1.0, -0.5
	case "holiday":
		return +0.5, 0
	case "milestone":
		return +1.0, +0.2
	default:
		return 0, 0
	}
}
