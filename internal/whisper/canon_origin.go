// Package whisper — canon_origin.go
// 100% 对齐 ackem canon/originEscalationGuard.ts
// OEG 创造者叙事守护：控制创造者话题的深度与频率

package whisper

// ─── Origin Guard ─────────────────────────────────────────────

const OriginGuardMarker = "【Origin Guard"

// DefaultOriginExposure 已定义在 types.go 中

// BuildOriginGuardBlock 构建 OEG 防护块
func BuildOriginGuardBlock() string {
	return OriginGuardMarker + ` · 强制回归用户】
已连续多轮聊创造者/出身话题。本回合最多一句带过，然后转向当前用户。
可温和问：「你今天好像一直在问我的起点，是发生什么让你在意了吗？」
禁止展开新的创作故事或记忆片段。`
}

// ShouldSuppressOriginProactive 是否抑制创造者主动话题
func ShouldSuppressOriginProactive(exposure *OriginExposure) bool {
	if exposure == nil {
		return false
	}
	return exposure.State == OriginDeep || exposure.State == OriginGuardCooldown
}

// AdvanceOriginStreak 推进创造者曝光 streak
func AdvanceOriginStreak(exposure *OriginExposure, fatherRefDetected bool, turnIndex int) *OriginExposure {
	if exposure == nil {
		exposure = DefaultOriginExposure()
	}

	// cooldown 检查
	if turnIndex < exposure.CooldownUntilTurn {
		return exposure
	}

	if fatherRefDetected {
		exposure.Streak++
	} else {
		// 缓慢衰减
		if exposure.Streak > 0 {
			exposure.Streak--
		}
	}

	// streak → state 映射
	switch {
	case exposure.Streak >= OriginStreakGuard:
		exposure.State = OriginGuardCooldown
		exposure.CooldownUntilTurn = turnIndex + int(OriginCooldownTurns)
	case exposure.Streak >= OriginStreakDeep:
		exposure.State = OriginDeep
	case exposure.Streak >= OriginStreakExplore:
		exposure.State = OriginExplore
	case exposure.Streak >= 1:
		exposure.State = OriginEntry
	default:
		exposure.State = OriginNormal
	}

	return exposure
}
