// Package whisper — profiling_user.go
// 100% 对齐 ackem engine/user-profiler.ts
// 用户画像：从行为推断用户原型

package whisper

// ─── UserArchetype ────────────────────────────────────────────

// UserArchetype 用户行为原型
type UserArchetype string

const (
	ArchetypeUnknown     UserArchetype = "unknown"
	ArchetypeClingy      UserArchetype = "clingy"      // 黏人型
	ArchetypeIndependent UserArchetype = "independent" // 独立型
	ArchetypeNurturing   UserArchetype = "nurturing"   // 照顾型
	ArchetypePlayful     UserArchetype = "playful"     // 玩闹型
	ArchetypeWithdrawn   UserArchetype = "withdrawn"   // 退缩型
)

// ─── UserProfiler ─────────────────────────────────────────────

// UpdateUserProfile 更新用户画像
func UpdateUserProfile(
	profile *UserProfile,
	eventType string,
	eventIntensity float64,
	l1 L1State,
	turnIndex int,
) *UserProfile {
	if profile == nil {
		profile = DefaultUserProfile()
	}

	// 信任轨迹
	if l1.Trust > 70 {
		profile.TrustTrajectory = "established"
	} else if l1.Trust > 40 {
		profile.TrustTrajectory = "building"
	} else {
		profile.TrustTrajectory = "fragile"
	}

	// 情感需求度 — vulnerable 事件提升
	if eventType == "vulnerable" {
		profile.EmotionalNeediness = clampF(profile.EmotionalNeediness+0.05, 0, 1)
	} else if eventType == "casual_chat" {
		profile.EmotionalNeediness = clampF(profile.EmotionalNeediness-0.01, 0, 1)
	}

	// 性直接度 — adult 事件提升
	if eventType == "adult_flirt" || eventType == "adult_explicit" {
		profile.SexualDirectness = clampF(profile.SexualDirectness+0.03, 0, 1)
	}

	// 主导偏好 — dominant/submissive 事件调整
	if eventType == "adult_dominant" {
		profile.DominancePreference = clampF(profile.DominancePreference+0.05, -1, 1)
	} else if eventType == "adult_submissive" {
		profile.DominancePreference = clampF(profile.DominancePreference-0.05, -1, 1)
	}

	profile.DetectedAtTurn = turnIndex
	return profile
}

// ─── Archetype Inference ──────────────────────────────────────

// InferArchetype 推断用户行为原型
func InferArchetype(profile UserProfile) UserArchetype {
	if profile.DetectedAtTurn < 5 {
		return ArchetypeUnknown
	}

	if profile.EmotionalNeediness > 0.7 {
		return ArchetypeClingy
	}
	if profile.EmotionalNeediness < 0.3 && profile.SexualDirectness > 0.5 {
		return ArchetypePlayful
	}
	if profile.EmotionalNeediness < 0.2 {
		return ArchetypeIndependent
	}
	if profile.DominancePreference > 0.5 {
		return ArchetypeNurturing
	}
	if profile.TrustTrajectory == "fragile" {
		return ArchetypeWithdrawn
	}
	return ArchetypeUnknown
}

// ─── Response Hint ────────────────────────────────────────────

// ArchetypeToResponseHint 原型→回复提示
func ArchetypeToResponseHint(archetype UserArchetype) string {
	switch archetype {
	case ArchetypeClingy:
		return "ta很需要你。多给一些温暖的回应，让ta感受到你的在乎。"
	case ArchetypeIndependent:
		return "ta比较独立。给ta空间，但让ta知道你一直在。"
	case ArchetypeNurturing:
		return "ta在照顾你。你也用同样的温柔回应ta。"
	case ArchetypePlayful:
		return "ta喜欢和你玩闹。用轻松调皮的方式回应ta。"
	case ArchetypeWithdrawn:
		return "ta此刻比较退缩。温柔地接住ta，但不要逼ta。"
	default:
		return ""
	}
}
