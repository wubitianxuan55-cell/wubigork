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

// CloneFullState 深拷贝 FullState（T7-1.1 ①）：
// 简单 struct 拷贝只复制指针本身，UserProfile/OriginExposure/EmergencePersistence
// 等可变对象仍与主流程共享——异步持久化在回合锁外 json.Marshal 快照时，会与
// 下一轮 PreLLMTurn 的原地修改（UpdateUserProfile/AdvanceOriginStreak/涌现相位）
// 数据竞争。这里逐指针/切片/map 深拷贝，保证快照真正独立。
func CloneFullState(s FullState) FullState {
	clone := s

	// 值指针字段：解引用复制（可变对象必须独立）
	if s.PersonalityBaseline != nil {
		v := *s.PersonalityBaseline
		clone.PersonalityBaseline = &v
	}
	if s.UserProfile != nil {
		v := *s.UserProfile
		clone.UserProfile = &v
	}
	if s.OriginExposure != nil {
		v := *s.OriginExposure
		clone.OriginExposure = &v
	}
	if s.FirstMetDate != nil {
		v := *s.FirstMetDate
		clone.FirstMetDate = &v
	}
	if s.AckemBirthday != nil {
		v := *s.AckemBirthday
		clone.AckemBirthday = &v
	}
	if s.Counters.LastConsolidationTurn != nil {
		v := *s.Counters.LastConsolidationTurn
		clone.Counters.LastConsolidationTurn = &v
	}
	if s.Counters.LastMirrorCheckTurn != nil {
		v := *s.Counters.LastMirrorCheckTurn
		clone.Counters.LastMirrorCheckTurn = &v
	}

	// 欲望栈：深拷贝槽位（*Desire 及其 *int 字段）
	if s.DesireStack.Slots != nil {
		slots := make([]*Desire, len(s.DesireStack.Slots))
		for i, d := range s.DesireStack.Slots {
			if d == nil {
				continue
			}
			dc := *d
			if d.ExpressedAtTurn != nil {
				et := *d.ExpressedAtTurn
				dc.ExpressedAtTurn = &et
			}
			slots[i] = &dc
		}
		clone.DesireStack.Slots = slots
	}

	// 离线思绪：值切片深拷贝
	if s.OfflineThoughts != nil {
		clone.OfflineThoughts = make([]OfflineThought, len(s.OfflineThoughts))
		copy(clone.OfflineThoughts, s.OfflineThoughts)
	}

	// 情绪涌现：深拷贝 Active（含 Context map）与 History 切片
	if s.EmergencePersistence != nil {
		ep := &EmergencePersistence{}
		if s.EmergencePersistence.Active != nil {
			a := *s.EmergencePersistence.Active
			if s.EmergencePersistence.Active.Context != nil {
				a.Context = make(map[string]interface{}, len(s.EmergencePersistence.Active.Context))
				for k, v := range s.EmergencePersistence.Active.Context {
					a.Context[k] = v
				}
			}
			ep.Active = &a
		}
		if s.EmergencePersistence.History != nil {
			ep.History = make([]EmergenceHistoryEntry, len(s.EmergencePersistence.History))
			copy(ep.History, s.EmergencePersistence.History)
		}
		clone.EmergencePersistence = ep
	}

	return clone
}
