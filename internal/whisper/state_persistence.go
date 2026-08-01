// Package whisper — 状态持久化（100% 对齐 ackem engine/state-persistence.ts）
package whisper

import "time"

// ─── 默认状态工厂 ──────────────────────────────────────────────

// DefaultL1 默认关系状态
func DefaultL1() L1State {
	return L1State{
		Stage:                    StageStranger,
		Trust:                    InitialTrust,
		Rifts:                    0,
		AffectionMomentum:        0,
		Atmosphere:               AtmoNeutral,
		ConsecutivePositiveTurns: 0,
		TurnsSinceLastRift:       99,
		SharedEventsCount:        0,
	}
}

// DefaultUserProfile 默认用户画像
func DefaultUserProfile() *UserProfile {
	now := time.Now().Format(time.RFC3339)
	return &UserProfile{
		DominantArchetype:   "unknown",
		SexualDirectness:    0.3,
		DominancePreference: 0,
		EmotionalNeediness:  0.3,
		TrustTrajectory:     "building",
		LastUpdated:         now,
		DetectedAtTurn:      0,
	}
}

// DefaultCounters 默认状态计数器
func DefaultCounters() StateCounters {
	return StateCounters{
		TotalTurns:                 0,
		SharedEventsCount:          0,
		ConsecutiveMeaningfulTurns: 0,
		LastConsolidationTurn:      nil,
		LastMirrorCheckTurn:        nil,
	}
}

// DefaultOriginExposure 默认创造者曝光状态
func DefaultOriginExposure() *OriginExposure {
	return &OriginExposure{
		State:             OriginNormal,
		Streak:            0,
		CooldownUntilTurn: 0,
	}
}

// DefaultFullState 构建默认引擎完整状态
func DefaultFullState(personality PersonalitySlice) FullState {
	now := time.Now()
	return FullState{
		Version:      StateJSONVersion,
		Relationship: DefaultL1(),
		Emotion: EmotionState{
			Aff: 0, Sec: 0, Aro: 0, Dom: 0,
			PrimaryLabel: "CALM_RATIONAL",
			IsLocked:     false,
		},
		Counters:             DefaultCounters(),
		LastActive:           now,
		ExternalAtmosphere:   ExternalAtmosphere{Level: 0, Label: AtmoNeutral},
		PersonalityBaseline:  &PersonalityDims{T: personality.T, I: personality.I, S: personality.S, O: personality.O, R: personality.R},
		Personality:          personality,
		UserProfile:          DefaultUserProfile(),
		DesireStack:          DefaultDesireStack(),
		OfflineThoughts:      []OfflineThought{},
		EmergencePersistence: &EmergencePersistence{Active: nil, History: []EmergenceHistoryEntry{}},
		OriginExposure:       DefaultOriginExposure(),
	}
}

// ─── 序列化辅助 ────────────────────────────────────────────────

// CloneFullState 深拷贝 FullState（简单 struct 拷贝即可，所有字段是值类型或指针）
func CloneFullState(s FullState) FullState {
	clone := s
	// 拷贝切片以避免共享底层数组
	if s.OfflineThoughts != nil {
		clone.OfflineThoughts = make([]OfflineThought, len(s.OfflineThoughts))
		copy(clone.OfflineThoughts, s.OfflineThoughts)
	}
	if s.DesireStack.Slots != nil {
		slots := make([]*Desire, len(s.DesireStack.Slots))
		copy(slots, s.DesireStack.Slots)
		clone.DesireStack.Slots = slots
	}
	return clone
}
