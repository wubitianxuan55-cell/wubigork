// Package whisper — strategy_intensity.go
// 100% 对齐 ackem extensions/policy/intensityModulator.ts
// 强度调制器：输出 0.5~1.5 LLM 温度调制参数

package whisper

import "math"

// ComputeIntensityModifier 计算强度调制参数
// 基于情绪+关系+时间，输出 0.5~1.5（1.0 为基线）
func ComputeIntensityModifier(
	aff, aro, dom float64,
	stage RelationshipStage,
	timeOfDay string,
	isWeekend bool,
	hasRestHabit bool,
) float64 {
	mod := 1.0

	// 情绪调制
	if aff > 60 {
		mod += 0.2 // 开心，语气活泼
	} else if aff < 20 {
		mod -= 0.2 // 低落，语气平稳
	}
	if math.Abs(aro) > 60 {
		mod += 0.1 // 兴奋/焦虑，可以多话
	}
	if dom < -30 {
		mod -= 0.1 // 不安，更谨慎
	}

	// 关系调制
	if stage == StageIntimate {
		mod += 0.1
	} else if stage == StageStranger {
		mod -= 0.1
	}

	// 时间调制
	if timeOfDay == "late_night" {
		mod -= 0.15
	}
	if isWeekend && timeOfDay == "morning" {
		mod += 0.1
	}

	// 习惯调制
	if hasRestHabit {
		mod -= 0.1
	}

	return clampF(mod, 0.5, 1.5)
}
