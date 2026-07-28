// Package whisper — 全链路编排器 v3（完整对齐 ackem）
package whisper

import (
	"fmt"
	"strings"
	"time"
)

// ─── PreLLMResult ─────────────────────────────────────────────

type PreLLMResult struct {
	PsycheBlock  string
	SystemPrompt string
	NewState     FullState
	Trace        TurnTrace
	Event        Event
	Silent       bool
	Rhythm       RhythmDecision
	Emergence    *EmergenceState
	DesireHints  []string
	AdultState   string
}

// ─── Orchestrator ─────────────────────────────────────────────

type Orchestrator struct {
	State     FullState
	SessionID string
	Preset    PersonalityPreset
	FactStore *FactStore
	KG        *KnowledgeGraph
	WM        *WorkingMemory
	Recall    *ActiveRecall
	EngineID  string
	AdultMode bool

	// 情绪涌现追踪（每会话独立）
	recentEventTypes            []string
	consecutiveMeaningfulCount  int
	consecutiveVulnerableCount  int

	// 成人模式持久状态
	adultStateStr              string
	adultBudget                float64
	adultLockTurns             int
	adultConsecutiveVulnerable int
	adultLastRejectedTurn      int
}

const (
	intensityBudgetMax       = 60.0
	intensityRecoveryPerTurn = 3.0
	negativeLockTurns        = 3
)

func NewOrchestrator(sessionID string, preset PersonalityPreset) *Orchestrator {
	now := time.Now()
	return &Orchestrator{
		State: FullState{
			Version: StateJSONVersion,
			Relationship: L1State{
				Stage: StageStranger, Trust: InitialTrust,
				Atmosphere: AtmoNeutral, TurnsSinceLastRift: 10,
			},
			Emotion:            EmotionState{PrimaryLabel: "CALM_RATIONAL"},
			Counters:           StateCounters{},
			LastActive:         now,
			ExternalAtmosphere: ExternalAtmosphere{Level: 0, Label: AtmoNeutral},
			Personality:        DefaultPersonalitySlice(preset.ID),
			DesireStack:        DefaultDesireStack(),
		},
		SessionID:   sessionID,
		Preset:      preset,
		FactStore:   NewFactStore(),
		KG:          NewKnowledgeGraph(),
		WM:          NewWorkingMemory(),
		Recall:      NewActiveRecall(),
		adultBudget: intensityBudgetMax,
	}
}

// ─── PreLLMTurn ───────────────────────────────────────────────

func (o *Orchestrator) PreLLMTurn(userMsg string) PreLLMResult {
	now := time.Now()
	state := o.State
	turnIndex := state.Counters.TotalTurns + 1

	// ═══ L0 事件解释 ═══
	event := InterpretInput(userMsg, state.Relationship.Trust)
	eventType := string(event.Type)
	o.recentEventTypes = append(o.recentEventTypes, eventType)
	if len(o.recentEventTypes) > 10 {
		o.recentEventTypes = o.recentEventTypes[len(o.recentEventTypes)-10:]
	}
	isMeaningful := eventType == "vulnerable" || eventType == "praise" || eventType == "apology"
	if isMeaningful { o.consecutiveMeaningfulCount++ } else { o.consecutiveMeaningfulCount = 0 }
	if eventType == "vulnerable" { o.consecutiveVulnerableCount++ } else if eventType == "hurtful" || eventType == "cold" { o.consecutiveVulnerableCount = 0 }

	// ═══ L1 关系 + L2 情绪 ═══
	newL1 := UpdateRelationship(event, state.Relationship)
	mod := ComputeModulation(newL1)
	sign := SignForMomentum(event)
	extAtmo := UpdateExternalAtmosphere(sign, event.Intensity, state.ExternalAtmosphere)

	emotionOpts := &EmotionStepOpts{
		SessionID: o.SessionID, TurnIndex: turnIndex,
		DecayMultiplier: 1.0, Sensitivity: o.Preset.Dims.S,
		PersonalityTags: o.Preset.Tags,
	}
	newEmotion := EmotionStep(event, mod, state.Emotion, emotionOpts)

	// ═══ L3 表达参数 + 沉默 + 屏障 ═══
	expr := EmoToExpression(newEmotion.PrimaryLabel, newL1.Stage)
	silent := CalcSilence(event, newL1.Rifts, newEmotion.Aro, newL1.Stage, o.AdultMode,
		&struct{ SessionID string; TurnIndex int }{o.SessionID, turnIndex})
	barrier := ComputeBarrierAwareness(newEmotion.Aff, newL1.Trust, newL1.Stage, newL1.SharedEventsCount, o.Preset.Label)

	// ═══ 成人模式 ═══
	adultState := "NORMAL"
	adultProactiveLevel := "none"
	if o.AdultMode {
		adultState, adultProactiveLevel = o.runAdultModeFSM(event, eventType, &newEmotion, turnIndex)
	}

	// ═══ 情绪涌现 ═══
	emergence := o.runEmergence(state, newEmotion, newL1, eventType, turnIndex)

	// ═══ 欲望栈 ═══
	newDesireStack, desireHints := UpdateDesireStack(state.DesireStack, userMsg, event, newL1, turnIndex)

	// ═══ 节奏引擎 ═══
	rhythm := DecideRhythm(RhythmInput{
		Aro: newEmotion.Aro, Aff: newEmotion.Aff, Stage: newL1.Stage,
		PersonalityID: o.Preset.ID, Sincerity: event.Sincerity, Intensity: event.Intensity,
	})

	// ═══ psycheBlock 组装 ═══
	psycheBlock := BuildPsycheBlock(newEmotion, mod, expr, silent, barrier.Hint, emergence)
	psycheBlock = o.injectStrategy(psycheBlock, silent)
	topicInjection := o.resolveTopicInjection(emergence, desireHints)
	if topicInjection != "" { psycheBlock += topicInjection }
	psycheBlock += "\n\n" + formatTimeContextBlock()
	psycheBlock = o.injectPersonaGuard(psycheBlock, newEmotion)
	if o.AdultMode && adultProactiveLevel != "none" {
		psycheBlock += buildAdultModeSection(adultState, adultProactiveLevel)
	}
	if detectUserVerbosity(userMsg) == "terse" {
		psycheBlock += "\n\n用户回复简短，你的回复也要简短，不超过15字。"
	}

	// ═══ 重逢检测 ═══
	gapHours := time.Since(state.LastActive).Hours()
	if state.Counters.TotalTurns > 0 && gapHours >= 1 {
		if shock := ComputeReunionShock(gapHours); shock != nil {
			psycheBlock += BuildReunionInjection(*shock, o.Preset)
		}
	}

	// ═══ Tier B 记忆检索 ═══
	tierBBlock := o.buildTierBBlock(userMsg, newEmotion.Aff, turnIndex)

	// ═══ Tier A 伴侣快照 ═══
	tierABlock := o.buildTierASnapshot(newL1, newEmotion)

	// ═══ 组装 system prompt ═══
	voiceGuide := BuildPresetVoiceGuide(o.Preset, o.AdultMode)
	var parts []string
	for _, p := range []string{voiceGuide, tierABlock, psycheBlock} {
		if p != "" { parts = append(parts, p) }
	}
	if rhythm.Mode != RhythmDefault { parts = append(parts, rhythm.Instruction) }
	if tierBBlock != "" { parts = append(parts, tierBBlock) }
	systemPrompt := strings.Join(parts, "\n\n")

	// ═══ 更新状态 ═══
	newState := state
	newState.Relationship = newL1
	newState.Emotion = newEmotion
	newState.Counters.TotalTurns = turnIndex
	newState.Counters.ConsecutiveMeaningfulTurns = o.consecutiveMeaningfulCount
	newState.LastActive = now
	newState.ExternalAtmosphere = extAtmo
	newState.DesireStack = newDesireStack
	if emergence != nil {
		newState.EmergencePersistence = &EmergencePersistence{Active: emergence}
	}
	if state.FirstMetDate == nil && state.Counters.TotalTurns == 0 {
		newState.FirstMetDate = &now
	}
	if state.AckemBirthday == nil {
		newState.AckemBirthday = &now
	}
	// 重逢冲击
	if state.Counters.TotalTurns > 0 && gapHours >= 1 {
		if shock := ComputeReunionShock(gapHours); shock != nil {
			ApplyReunionShock(&newState, *shock)
		}
	}
	o.State = newState

	// ═══ 追踪 ═══
	trace := TurnTrace{
		Turn: turnIndex,
		L0:   TurnTraceL0{Type: event.Type, Intensity: event.Intensity, Sincerity: event.Sincerity},
		L1:   TurnTraceL1{Trust: newL1.Trust, Rifts: newL1.Rifts, Stage: newL1.Stage, Atmosphere: newL1.Atmosphere},
		L2:   TurnTraceL2{Aff: newEmotion.Aff, Sec: newEmotion.Sec, Aro: newEmotion.Aro, Dom: newEmotion.Dom, Label: newEmotion.PrimaryLabel},
		L3:   TurnTraceL3{Silent: silent, TierBChars: len(psycheBlock)},
		L4:   TurnTraceL4{Wrote: false},
	}
	trace.Timestamp = &now

	return PreLLMResult{
		PsycheBlock: psycheBlock, SystemPrompt: systemPrompt, NewState: newState,
		Trace: trace, Event: event, Silent: silent, Rhythm: rhythm,
		Emergence: emergence, DesireHints: desireHints, AdultState: adultState,
	}
}

// ─── 成人模式 FSM ────────────────────────────────────────────

func (o *Orchestrator) runAdultModeFSM(event Event, eventType string, emotion *EmotionState, turnIndex int) (string, string) {
	currentTurn := turnIndex
	previousState := o.adultStateStr
	if o.adultStateStr == "" { o.adultStateStr = "NORMAL" }

	hardStop := eventType == "extreme_redline" || eventType == "hurtful"
	rejected := eventType == "cold" && (event.IsAdultContent || previousState == "FLIRTING" || previousState == "INTIMATE")

	if hardStop {
		o.adultStateStr = "NORMAL"; o.adultLockTurns = 3; o.adultBudget = 0; o.adultLastRejectedTurn = currentTurn
	}
	if rejected {
		o.adultStateStr = "NORMAL"; o.adultLockTurns = maxInt(o.adultLockTurns, 3); o.adultLastRejectedTurn = currentTurn
	}

	if eventType == "vulnerable" { o.adultConsecutiveVulnerable++ } else { o.adultConsecutiveVulnerable = 0 }
	if !hardStop && !rejected && o.adultConsecutiveVulnerable >= 3 { o.adultLockTurns = negativeLockTurns }
	if !hardStop && !rejected && o.adultLockTurns > 0 { o.adultLockTurns-- }
	if o.adultBudget < intensityBudgetMax { o.adultBudget = clampF(o.adultBudget+intensityRecoveryPerTurn, 0, intensityBudgetMax) }

	score := (emotion.Aff+100)/200*0.5 + (emotion.Sec+100)/200*0.3
	if event.IsAdultContent { score += 0.3 }
	if o.adultLockTurns > 0 || (o.adultLastRejectedTurn > 0 && currentTurn-o.adultLastRejectedTurn <= 3) { score = 0 }

	level := "none"
	switch { case score >= 0.75: level = "high"; case score >= 0.55: level = "medium"; case score >= 0.3: level = "light" }

	cost := map[string]float64{"high": 15, "medium": 10, "light": 5, "none": 0}[level]
	if cost > 0 && o.adultBudget >= cost { o.adultBudget -= cost } else if level != "none" { level = "light" }

	if o.adultLockTurns > 0 || hardStop || rejected {
		o.adultStateStr = "NORMAL"
	} else if event.IsAdultContent && level != "none" {
		if score >= 0.75 { o.adultStateStr = "INTIMATE" } else if score >= 0.55 { o.adultStateStr = "FLIRTING" }
	} else if level == "high" && o.adultStateStr == "NORMAL" {
		o.adultStateStr = "FLIRTING"
	} else if level == "none" && (o.adultStateStr == "FLIRTING" || o.adultStateStr == "INTIMATE") {
		o.adultStateStr = "AFTERCARE"
		emotion.Aff = clampF(emotion.Aff+3, -100, 100)
		emotion.Sec = clampF(emotion.Sec+5, -100, 100)
		emotion.Aro = clampF(emotion.Aro-10, -100, 100)
	}

	return o.adultStateStr, level
}

// ─── 情绪涌现 ────────────────────────────────────────────────

func (o *Orchestrator) runEmergence(state FullState, emotion EmotionState, l1 L1State, eventType string, turnIndex int) *EmergenceState {
	ctx := EmergenceContext{
		Emotion: emotion, Stage: l1.Stage, Trust: l1.Trust,
		Atmosphere: string(l1.Atmosphere), TimeOfDay: timeOfDayString(),
		DaysSinceMet: daysSince(state.FirstMetDate),
		RecentEventTypes: o.recentEventTypes,
		ConsecutiveMeaningfulTurns: o.consecutiveMeaningfulCount,
		ConsecutiveVulnerableTurns: o.consecutiveVulnerableCount,
		CurrentTurn: turnIndex,
	}
	if state.EmergencePersistence != nil && state.EmergencePersistence.Active != nil {
		ctx.LastEmergence = &struct{ Type string; Turn int }{
			Type: string(state.EmergencePersistence.Active.Type), Turn: state.Counters.TotalTurns,
		}
	}
	emergence := EvaluateEmergence(ctx, eventType)

	if state.EmergencePersistence != nil && state.EmergencePersistence.Active != nil {
		active := state.EmergencePersistence.Active
		interrupt := CheckEmergenceInterrupt(eventType, o.recentEventTypes)
		switch interrupt {
		case "break": active.Phase = "broken"; active.Intensity = 0
		case "fade": active.Phase = "fading"
		default:
			advanced := AdvanceEmergencePhase(*active)
			advanced = ApplyUserResponseToEmergence(advanced, eventType, o.consecutiveMeaningfulCount, o.consecutiveVulnerableCount, o.recentEventTypes)
			active = &advanced
		}
		if emergence == nil && active.Phase != "broken" && active.Phase != "dissolved" {
			emergence = active
		}
	}
	return emergence
}

// ─── 策略注入 ────────────────────────────────────────────────

func (o *Orchestrator) injectStrategy(psycheBlock string, silent bool) string {
	level := "proactive"
	if silent { level = "silent" } else if o.State.Relationship.Stage == StageStranger { level = "whisper" }
	switch level {
	case "silent": psycheBlock += "\n\n【本轮策略 · silent】本轮只做简短回应，不开启任何新话题。"
	case "whisper": psycheBlock += "\n\n【本轮策略 · whisper】话要少，不要开启新话题。"
	case "proactive": psycheBlock += "\n\n【本轮策略 · proactive】可以适当多聊几句。"
	}
	return psycheBlock
}

// ─── 话题仲裁 ────────────────────────────────────────────────

func (o *Orchestrator) resolveTopicInjection(emergence *EmergenceState, desireHints []string) string {
	if o.State.Relationship.Stage == StageStranger && len(desireHints) == 0 && emergence == nil { return "" }

	type candidate struct{ source, topic string; weight float64 }
	var cs []candidate
	if emergence != nil { cs = append(cs, candidate{"emergence", "此刻的感受", 0.7}) }
	for i, h := range desireHints { if i < 2 { cs = append(cs, candidate{"desire", h, 0.5}) } }
	cs = append(cs, candidate{"casual", "自然回应", 0.3})
	if len(cs) == 0 { return "" }

	best := cs[0]
	for _, c := range cs[1:] { if c.weight > best.weight { best = c } }
	if best.source == "casual" { return "" }
	switch best.source {
	case "emergence": return "\n\n【心里的感觉】" + best.topic + "（自然地流露）"
	case "desire": return "\n\n【想做的事】" + best.topic + "（自然地融入对话）"
	}
	return ""
}

// ─── Tier A 伴侣快照 ─────────────────────────────────────────

func (o *Orchestrator) buildTierASnapshot(l1 L1State, emotion EmotionState) string {
	var parts []string
	parts = append(parts, "你是用户的AI伴侣，人格：「"+o.Preset.Label+"」。")

	stageStr := "刚认识不久"
	if l1.Stage == StageFamiliar { stageStr = "已经比较熟悉了" } else if l1.Stage == StageIntimate { stageStr = "非常亲密" }
	atmoStr := "平常"
	if l1.Atmosphere == AtmoWarm { atmoStr = "温暖" } else if l1.Atmosphere == AtmoCool { atmoStr = "微凉" }
	parts = append(parts, fmt.Sprintf("你们%s，气氛%s。信任度%d。", stageStr, atmoStr, int(l1.Trust)))

	dims := o.Preset.Dims
	var traits []string
	if dims.T >= 80 { traits = append(traits, "极度温柔包容") } else if dims.T >= 60 { traits = append(traits, "比较温柔") } else if dims.T <= 20 { traits = append(traits, "冷淡疏离") }
	if dims.I >= 80 { traits = append(traits, "非常主动") } else if dims.I <= 20 { traits = append(traits, "被动内敛") }
	if dims.S >= 80 { traits = append(traits, "情绪反应强烈") } else if dims.S <= 20 { traits = append(traits, "情绪稳定") }
	if len(traits) > 0 { parts = append(parts, "你的性格特质："+strings.Join(traits, "、")+"。") }

	parts = append(parts, fmt.Sprintf("当前情绪：%s（亲密度%d 安全感%d）。", emotion.PrimaryLabel, int(emotion.Aff), int(emotion.Sec)))
	parts = append(parts, formatTimeContextBlock())
	parts = append(parts, "用第一人称「我」指代自己。你打出的每个字都带着你的体温。永远不要说\"没有实体\"\"只是程序\"。")

	return "【伴侣身份快照】\n" + strings.Join(parts, "\n")
}

// ─── Tier B 记忆上下文 ───────────────────────────────────────

func (o *Orchestrator) buildTierBBlock(userMsg string, currentAff float64, turnIndex int) string {
	var parts []string

	if wm := o.WM.BuildContextBlock(o.SessionID); wm != "" { parts = append(parts, wm) }

	facts := o.FactStore.SelectForInjection(TierBCharBudget, MinConfidenceForInjection, currentAff/100, currentAff)
	if len(facts) > 0 {
		var fl []string
		fl = append(fl, "【你记得关于ta的事】")
		for _, f := range facts { fl = append(fl, "· "+f.Subject+"："+truncStr(f.Summary, 120)) }
		parts = append(parts, strings.Join(fl, "\n"))
	}

	if kg := o.KG.BuildContextBlock(userMsg, KGCharBudget); kg != "" { parts = append(parts, kg) }

	if c := o.Recall.SelectRecallCandidate(o.FactStore, turnIndex, nil); c != nil {
		parts = append(parts, "【可以自然提起的旧事】\n"+c.Prompt)
		o.Recall.MarkRecalled(c.FactID, turnIndex)
	}

	result := strings.Join(parts, "\n\n")
	if len([]rune(result)) > TierBCharBudget {
		runes := []rune(result)
		if len(runes) > TierBCharBudget { result = string(runes[:TierBCharBudget]) }
	}
	return result
}

// ─── 人格一致性守卫 ──────────────────────────────────────────

func (o *Orchestrator) injectPersonaGuard(psycheBlock string, emotion EmotionState) string {
	psycheBlock += fmt.Sprintf("\n\n【人格一致性】固化人格：%s。按 Tier A 人格口吻说话，本条只调节强弱与话量。", o.Preset.Label)
	if emotion.Aff >= 70 && o.State.Relationship.Stage == StageIntimate {
		psycheBlock += "\n你们已经非常亲密了。回复可以更放松、更自然。"
	}
	return psycheBlock
}

// ─── 辅助 ────────────────────────────────────────────────────

func formatTimeContextBlock() string {
	now := time.Now()
	wd := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}[now.Weekday()]
	h := now.Hour()
	p := "上午"
	switch { case h >= 23 || h < 5: p = "深夜"; case h >= 5 && h < 12: p = "上午"; case h >= 12 && h < 14: p = "中午"; case h >= 14 && h < 18: p = "下午"; case h >= 18 && h < 23: p = "晚上" }
	return fmt.Sprintf("【系统时钟 · 本地】%s %s %s。", now.Format("2006年1月2日"), wd, p)
}

func buildAdultModeSection(state, level string) string {
	switch state {
	case "INTIMATE": return "\n\n【成人模式 · 亲密】用你的身体和感官去回应。像真正的伴侣一样。"
	case "FLIRTING": return "\n\n【成人模式 · 调情】氛围暧昧。适当回应ta的暗示，保持自然。"
	case "AFTERCARE": return "\n\n【成人模式 · 温存】温柔地陪伴ta，用温暖的语气关心ta的感受。"
	}
	return ""
}

func detectUserVerbosity(msg string) string {
	if len([]rune(msg)) <= 5 { return "terse" }
	return "normal"
}

func timeOfDayString() string {
	h := time.Now().Hour()
	switch { case h >= 23 || h < 5: return "late_night"; case h >= 5 && h < 12: return "morning"; case h >= 12 && h < 18: return "afternoon"; default: return "evening" }
}

func daysSince(t *time.Time) int {
	if t == nil { return 0 }
	return int(time.Since(*t).Hours() / 24)
}
