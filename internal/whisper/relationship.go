// Package whisper — L1 关系引擎（100% 对齐 ackem engine/relationship.ts）
package whisper

// ─── 事件正负分类 ─────────────────────────────────────────────

var positiveTypes = map[EventType]bool{
	EvtPraise: true, EvtTease: true, EvtVulnerable: true, EvtApology: true,
}

var negativeTypes = map[EventType]bool{
	EvtCold: true, EvtHurtful: true,
}

func trustDelta(event Event) float64 {
	switch event.Type {
	case EvtPraise:
		return TrustPraise
	case EvtApology:
		return TrustApology
	case EvtVulnerable:
		return TrustVulnerable
	case EvtTease:
		return TrustTease
	case EvtCold:
		return TrustCold
	case EvtHurtful:
		return TrustHurtful
	case EvtCasualChat:
		return TrustCasual
	case EvtQuestion:
		return TrustQuestion
	default:
		return 0
	}
}

// SignForMomentum 事件对情感动量的符号贡献
func SignForMomentum(event Event) float64 {
	if positiveTypes[event.Type] {
		return 1
	}
	if negativeTypes[event.Type] {
		return -1
	}
	return 0
}

// ComputeModulation 计算 L1 调制系数
func ComputeModulation(l1 L1State) Modulation {
	trustMod := TrustModMin + (l1.Trust/100.0)*(TrustModMax-TrustModMin)
	riftMod := mathMax(RiftModMin, 1.0-float64(l1.Rifts)*RiftModDecayPerRift)

	stageWeight := StageWeightFamiliar
	if l1.Stage == StageStranger {
		stageWeight = StageWeightStranger
	}
	if l1.Stage == StageIntimate {
		stageWeight = StageWeightIntimate
	}

	return Modulation{
		TrustMod:    trustMod,
		RiftMod:     riftMod,
		StageWeight: stageWeight,
		Atmosphere:  l1.Atmosphere,
	}
}

// UpdateExternalAtmosphere P1-4 外场气氛更新
func UpdateExternalAtmosphere(sign, intensity float64, prev ExternalAtmosphere) ExternalAtmosphere {
	alpha := ExternalMomentumAlpha
	delta := (1 - alpha) * intensity * sign
	level := clampF(prev.Level*alpha+delta, -1, 1)

	var label Atmosphere
	if level > ExternalWarmThreshold {
		label = AtmoWarm
	} else if level < ExternalCoolThreshold {
		label = AtmoCool
	} else {
		label = AtmoNeutral
	}
	return ExternalAtmosphere{Level: level, Label: label}
}

// evolveStage 关系阶段进化 FSM
func evolveStage(s L1State) RelationshipStage {
	switch s.Stage {
	case StageStranger:
		if s.ConsecutivePositiveTurns > StageWarmupTurns {
			return StageFamiliar
		}
	case StageFamiliar:
		if s.Trust > StageIntimateTrust && s.SharedEventsCount >= StageIntimateEvents {
			return StageIntimate
		}
	case StageIntimate:
		if s.Rifts > StageDowngradeRifts || s.Trust < StageDowngradeTrust {
			return StageFamiliar
		}
	}
	return s.Stage
}

// iceBreakResult 破冰结果
type iceBreakResult struct {
	trustBonus       float64
	forcedAtmosphere *Atmosphere
}

// applyIceBreak P1-2 破冰检测
func applyIceBreak(event Event, l1 L1State) iceBreakResult {
	if event.Type == EvtApology &&
		l1.Trust <= IceBreakTrustThreshold &&
		event.Sincerity >= IceBreakSincerityThreshold {
		atmo := AtmoNeutral
		return iceBreakResult{trustBonus: IceBreakTrustBonus, forcedAtmosphere: &atmo}
	}
	return iceBreakResult{trustBonus: 0}
}

// UpdateRelationship L1 主入口：事件 → 新 L1State
func UpdateRelationship(event Event, prev L1State) L1State {
	if event.Type == EvtExtremeRedline || event.IsExtremeRedline {
		return prev
	}

	trust := clampF(prev.Trust+trustDelta(event), 0, 100)

	// P1-2 破冰
	ice := applyIceBreak(event, L1State{Trust: trust, Stage: prev.Stage})
	trust = clampF(trust+ice.trustBonus, 0, 100)

	rifts := prev.Rifts
	turnsSinceLastRift := prev.TurnsSinceLastRift + 1

	if event.Type == EvtHurtful && prev.TurnsSinceLastRift >= RiftHurtfulCooldown {
		rifts++
		turnsSinceLastRift = 0
	}

	if event.Type == EvtApology && rifts > 0 && prev.ConsecutivePositiveTurns >= RiftRepairPositiveStreak {
		rifts = max(0, rifts-1)
	}

	consecutivePositiveTurns := prev.ConsecutivePositiveTurns
	if positiveTypes[event.Type] {
		consecutivePositiveTurns++
	} else if negativeTypes[event.Type] {
		consecutivePositiveTurns = 0
	}

	sign := SignForMomentum(event)
	affectionMomentum := MomentumAlpha*prev.AffectionMomentum + (1-MomentumAlpha)*event.Intensity*sign

	var atmosphere Atmosphere
	if ice.forcedAtmosphere != nil {
		atmosphere = *ice.forcedAtmosphere
	} else {
		atmosphere = AtmoNeutral
		if affectionMomentum > AtmosphereWarmThreshold {
			atmosphere = AtmoWarm
		} else if affectionMomentum < AtmosphereCoolThreshold {
			atmosphere = AtmoCool
		}
	}

	merged := L1State{
		Stage:                    prev.Stage, // P0修复: FSM演进需要传入当前阶段
		Trust:                    trust,
		Rifts:                    rifts,
		AffectionMomentum:        affectionMomentum,
		Atmosphere:               atmosphere,
		ConsecutivePositiveTurns: consecutivePositiveTurns,
		TurnsSinceLastRift:       turnsSinceLastRift,
		SharedEventsCount:        prev.SharedEventsCount,
	}
	merged.Stage = evolveStage(merged)

	return merged
}
