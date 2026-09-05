// Package whisper — 全链路编排器 v3（完整对齐 ackem）
package whisper

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/genui"
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
	IntensityMod float64 // 强度调制参数（0.5~1.5），供 LLM 温度使用
	SkipLLM      bool
	RedlineReply string
}

// ─── Orchestrator ─────────────────────────────────────────────

type Orchestrator struct {
	State            FullState
	SessionID        string
	Preset           PersonalityPreset
	AssistantName    string // 助手自定义名字（空=回退"gaea"）
	FactStore        *FactStore
	KG               *KnowledgeGraph
	WM               *WorkingMemory
	EpisodicStore    *EpisodicStore // 情节记忆（运行时实例，随会话持久化到 hermes.db）
	Recall           *ActiveRecall
	AssocIndex       *AssociationIndex     // P2: 关联索引（供 post-turn 纠正）
	HabitsStore      *HabitsStore          // P2: 习惯存储（供 DnD/健康检测写入）
	SelfEditor       *MemorySelfEditor     // v5.40: 记忆自编辑器
	ProceduralHabits *ProceduralHabitStore // v5.40: 程序化习惯存储
	EngineID         string
	ModelName        string // v5.66: 轻语专属 LLM 模型名
	ImageModelName   string // v5.66: 轻语专属生图模型名
	AdultMode        bool
	DataRoot         string // v5.41: SQLite 持久化数据根目录
	// v5.80: FTS 全文检索回调（app 层注入 repos.SearchFactIDsFTS 实现，避免主包依赖 db/repos 形成循环）
	FTSSearch func(query string, limit int) []string
	// v5.43: 桌面助手子系统
	ConfirmSvc    *ConfirmService      // 确认服务
	DeliveryCoord *DeliveryCoordinator // 消息分发协调
	// 情绪涌现追踪（每会话独立）
	recentEventTypes           []string
	consecutiveMeaningfulCount int
	consecutiveVulnerableCount int

	// 成人模式持久状态
	adultStateStr              string
	adultBudget                float64
	adultLockTurns             int
	adultConsecutiveVulnerable int
	adultLastRejectedTurn      int

	// T7-1.1 并发正确性：
	// mu 串行化同一会话的回合处理（GUI/微信/语音三个入口汇聚到 WhisperChat，
	// 由 app 层持锁调用）；异步持久化协程经 CloneFullState 快照读取，不持本锁。
	mu sync.Mutex
	// rhythm 节奏连续计数器（本会话独立，修复包级全局串台）。
	rhythm RhythmCounters
	// v4.3a: 关系记忆回填标记位——首个回合 restoreMemoryGraphFromState 执行后置位，
	// 避免重复重建（不能依赖 turnIndex==1：重启后 Counters.TotalTurns 恢复为历史值）。
	memoryGraphRestored bool
}

const (
	intensityBudgetMax       = 60.0
	intensityRecoveryPerTurn = 3.0
	negativeLockTurns        = 3
)

// LockTurn 串行化同一会话的回合处理（T7-1.1）。三个并发入口
// （GUI 聊天 / 微信回调 / 语音）都汇聚到 app 层 WhisperChat，调用方
// 在回合开始前 LockTurn、结束后 UnlockTurn。异步持久化协程不持此锁
// （经 CloneFullState 快照读取），避免阻塞下一轮。
func (o *Orchestrator) LockTurn()   { o.mu.Lock() }
func (o *Orchestrator) UnlockTurn() { o.mu.Unlock() }

// AddTemporalAnchor 追加时间锚点到 State.TemporalAnchors（供异步记忆写入
// 从回合外调用）：持回合锁写入，避免与主回合状态快照（CloneFullState）竞态。
func (o *Orchestrator) AddTemporalAnchor(a TemporalAnchor) {
	if o == nil {
		return
	}
	o.LockTurn()
	defer o.UnlockTurn()
	o.State.TemporalAnchors = append(o.State.TemporalAnchors, a)
}

func NewOrchestrator(sessionID string, preset PersonalityPreset) *Orchestrator {
	personality := DefaultPersonalitySlice(preset.ID)
	return &Orchestrator{
		State:            DefaultFullState(personality),
		SessionID:        sessionID,
		Preset:           preset,
		FactStore:        NewFactStore(),
		KG:               NewKnowledgeGraph(),
		WM:               NewWorkingMemory(),
		EpisodicStore:    NewEpisodicStore(),
		Recall:           NewActiveRecall(),
		AssocIndex:       NewAssociationIndex(),
		HabitsStore:      NewHabitsStore(),
		SelfEditor:       NewMemorySelfEditor(),
		ProceduralHabits: NewProceduralHabitStore(),
		ConfirmSvc:       NewConfirmService(),
		DeliveryCoord:    NewDeliveryCoordinator(),
		AdultMode:        true, // 产品决策：成人内容默认开启（个人非商用定位，见 docs/ADULT_MODE.md）
		adultBudget:      intensityBudgetMax,
	}
}

// ─── PreLLMTurn ───────────────────────────────────────────────

func (o *Orchestrator) PreLLMTurn(userMsg string) PreLLMResult {
	// v4.3a: 重启后首回合回填关系记忆（关联索引/习惯库）并重建关联图（fail-open）。
	// 必须在读 State 之前执行——State.Associations/Habits 已由 restoreWhisperState 装载。
	o.restoreMemoryGraphFromState()

	now := time.Now()
	state := o.State
	turnIndex := state.Counters.TotalTurns + 1

	// ═══ 会话复位 ═══
	if turnIndex == 1 {
		o.rhythm.Reset()
		ResetEmergenceTracking()
	}

	// ═══ 涌现恢复：处理关机/休眠后的涌现状态 ═══
	if state.EmergencePersistence != nil && state.EmergencePersistence.Active != nil {
		active := state.EmergencePersistence.Active
		hoursSinceStart := time.Since(active.StartedAt).Hours()
		if active.Phase == "dissolved" || active.Phase == "broken" {
			state.EmergencePersistence.Active = nil
		} else if hoursSinceStart > 2 {
			active.Phase = "fading"
			active.RoundsInPhase = 0
		} else if active.Phase == "sustained" || active.Phase == "rising" {
			active.Intensity = clampF(active.Intensity-0.15, 0, 1)
		}
	}

	// ═══ 记忆增强：effectiveTrust + sharedEventsCount ═══
	memAug := AugmentL1FromMemory(state.Relationship, o.FactStore)
	l1WithMem := state.Relationship
	l1WithMem.SharedEventsCount = memAug.SharedEventsCount
	effectiveTrust := EffectiveTrustForL0(l1WithMem, o.FactStore)

	// ═══ L0 事件解释 ═══
	event := InterpretInput(userMsg, effectiveTrust)
	eventType := string(event.Type)
	o.recentEventTypes = append(o.recentEventTypes, eventType)
	if len(o.recentEventTypes) > 10 {
		o.recentEventTypes = o.recentEventTypes[len(o.recentEventTypes)-10:]
	}
	isMeaningful := eventType == "vulnerable" || eventType == "praise" || eventType == "apology"
	if isMeaningful {
		o.consecutiveMeaningfulCount++
	} else {
		o.consecutiveMeaningfulCount = 0
	}
	if eventType == "vulnerable" {
		o.consecutiveVulnerableCount++
	} else if eventType == "hurtful" || eventType == "cold" {
		o.consecutiveVulnerableCount = 0
	}

	// ═══ 红线熔断 ═══
	if event.IsExtremeRedline {
		return o.redlineResult(now, event, turnIndex)
	}
	dndResult := IsDNDMessage(userMsg)
	if dndResult.Detected {
		now := time.Now()
		weekday := int(now.Weekday())
		var expiresAt int64
		if dndResult.Hours > 0 {
			expiresAt = now.Add(time.Duration(dndResult.Hours) * time.Hour).UnixMilli()
		}
		o.HabitsStore.Upsert(UserHabit{
			Type:            "dnd",
			Scope:           "short_term",
			Weekday:         &weekday,
			HourStart:       now.Hour(),
			HourEnd:         now.Hour() + dndResult.Hours,
			Confidence:      0.7,
			OccurrenceCount: 1,
			FirstSeenAt:     now.UnixMilli(),
			LastConfirmedAt: now.UnixMilli(),
			ExpiresAt:       &expiresAt,
			Source:          "detected",
		})
	}
	// ═══ 主动门控：推送 aff 历史 ═══
	PushAffToHistory(state.Emotion.Aff)

	// ═══ 反差人格渐变（hiddenRatio）════
	hiddenPersona := o.Preset.HiddenPersona
	effSens := state.Personality.S
	if o.AdultMode && hiddenPersona != nil {
		delta := -0.05
		if event.IsAdultContent {
			delta = 0.15
		}
		r := clampF(state.Personality.HiddenRatio+delta, 0, 1)
		h := hiddenPersona
		effSens = state.Personality.S*(1-r) + h.S*r
		state.Personality.HiddenRatio = r
		state.Personality.T = clampF(state.Personality.T*(1-r)+h.T*r, -100, 100)
		state.Personality.I = clampF(state.Personality.I*(1-r)+h.I*r, -100, 100)
		state.Personality.S = effSens
		state.Personality.O = clampF(state.Personality.O*(1-r)+h.O*r, -100, 100)
		state.Personality.R = clampF(state.Personality.R*(1-r)+h.R*r, -100, 100)
	}

	// ═══ L1 关系 + L2 情绪 ═══
	newL1 := UpdateRelationship(event, state.Relationship)
	mod := ComputeModulation(newL1)
	sign := SignForMomentum(event)
	extAtmo := UpdateExternalAtmosphere(sign, event.Intensity, state.ExternalAtmosphere)

	emotionOpts := &EmotionStepOpts{
		SessionID:       o.SessionID,
		TurnIndex:       turnIndex,
		DecayMultiplier: 1.0,
		Sensitivity:     effSens,
		PersonalityTags: o.Preset.Tags,
	}
	newEmotion := EmotionStep(event, mod, state.Emotion, emotionOpts)

	// ═══ v5.40: 周日情绪偏移 ═══
	weekdayBias := ComputeWeekdayMoodBias(int(now.Weekday()))
	if weekdayBias != 0 {
		newEmotion.Aff += weekdayBias
	}

	// ═══ 重逢 boost ═══
	gapHours := time.Since(state.LastActive).Hours()
	if state.Counters.TotalTurns > 0 && gapHours >= 1 {
		o.applyReunionBoost(&newEmotion, gapHours)
	}

	// ═══ L3 表达参数 + 沉默 + 屏障 ═══
	expr := EmoToExpression(newEmotion.PrimaryLabel, newL1.Stage)
	silent := CalcSilence(event, newL1.Rifts, newEmotion.Aro, newL1.Stage, o.AdultMode,
		&struct {
			SessionID string
			TurnIndex int
		}{o.SessionID, turnIndex})
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

	rhythm := DecideRhythm(RhythmInput{
		Aro: newEmotion.Aro, Aff: newEmotion.Aff, Stage: newL1.Stage,
		PersonalityID: o.Preset.ID, Sincerity: event.Sincerity, Intensity: event.Intensity,
	}, &o.rhythm)

	// ═══ psycheBlock 组装 ═══
	psycheBlock := BuildPsycheBlock(newEmotion, mod, expr, silent, barrier.Hint, emergence)

	// 产品身份守卫
	psycheBlock = InjectProductGuard(psycheBlock)

	// Canon 身份块（Stranger 阶段使用防护版本）
	if ShouldInjectStrangerGuard(newL1.Stage) {
		psycheBlock += "\n\n" + BuildStrangerGuardBlock(o.Preset.Label)
	} else {
		psycheBlock += "\n\n" + BuildAckemCanonBlock(o.Preset.Label, o.AssistantName)
	}

	// 创造者记忆块
	if seeds := DefaultCreatorSeeds(); len(seeds) > 0 {
		psycheBlock += "\n\n" + BuildCreatorMemoryBlock(seeds)
	}

	// 策略注入
	psycheBlock = o.injectStrategy(psycheBlock, silent)

	// 话题仲裁 — 使用策略模块
	topicInjection := o.resolveTopicInjection(emergence, desireHints)
	if topicInjection != "" {
		psycheBlock += topicInjection
	}

	// 时间感知：特殊日期 + 种子块
	temporalSignal := DetectSpecialDates(now, state.FirstMetDate, state.AckemBirthday)
	if temporalSignal != nil {
		seed := BuildTemporalSeedTierBBlock(temporalSignal, o.FactStore, o.KG)
		if seed != "" {
			psycheBlock += "\n" + seed
		}
	}

	psycheBlock += "\n\n" + FormatTimeContextBlock(time.Now())
	psycheBlock = o.injectPersonaGuard(psycheBlock, newEmotion)

	// 主动门控评估（v4.8.3：助手人格 gaea 豁免——silent/whisper 策略为陪伴
	// 场景设计，新关系冷启动时会把助手回复压成一句话，与「结论先行」人设
	// 冲突；微信实测「你会什么」只回 45 字即此因）。
	gate := EvaluateProactiveGate(newEmotion.Aff, newEmotion.Aro, newEmotion.Sec,
		newL1.Trust, newL1.Rifts, newL1.Stage, timeOfDayKey(), o.AdultMode)
	if o.Preset.ID != "gaea" {
		if gate.Level == "silent" {
			psycheBlock += "\n\n【本轮策略 · silent】本轮只做简短回应，不开启任何新话题。"
		} else if gate.Level == "whisper" {
			psycheBlock += "\n\n【本轮策略 · whisper】话要少，不要开启新话题。"
		}
	}

	// 强度调制 — 使用外部模块
	intensityMod := ComputeIntensityModifier(newEmotion.Aff, newEmotion.Aro, newEmotion.Dom,
		newL1.Stage, timeOfDayKey(), temporalSignal != nil, false)

	// 成人模式
	if o.AdultMode && adultProactiveLevel != "none" {
		psycheBlock += BuildAdultModeSection(o.Preset.ID, AdultState(adultState), adultProactiveLevel)
	}

	// 用户啰嗦度（v4.8.3：短但含疑问的消息豁免镜像——「你是谁/你会什么」
	// 是提问不是寒暄，≤15 字钳制会把答案掐死）。
	if DetectUserVerbosity(userMsg) == "terse" && !isShortQuestion(userMsg) {
		psycheBlock += "\n\n用户回复简短，你的回复也要简短，不超过15字。"
	}

	// 心理健康软关注
	if DetectSoftConcern(userMsg) {
		psycheBlock += "\n\n【注意】ta可能有些累了。温柔地回应，但不要追问。"
	}

	// 离线思绪注入
	if len(state.OfflineThoughts) > 0 {
		if hint := OfflineThoughtsToHint(state.OfflineThoughts); hint != "" {
			psycheBlock += hint
		}
	}

	// 重逢冲击注入
	if state.Counters.TotalTurns > 0 && gapHours >= 1 {
		if shock := ComputeReunionShock(gapHours); shock != nil {
			psycheBlock += BuildReunionInjection(*shock, o.Preset)
		}
	}

	// gaea主动消息评估
	if proactive := ComposeProactiveMessage(gate, newEmotion.Aff, newEmotion.Sec,
		newL1.Trust, newL1.Stage, timeOfDayKey(), gapHours, emergence != nil,
		o.Preset.ID); proactive != nil && proactive.ShouldSend {
		psycheBlock += "\n\n" + proactive.PromptHint
	}

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
	if state.FirstMetDate == nil {
		newState.FirstMetDate = &now
	}
	if state.AckemBirthday == nil {
		newState.AckemBirthday = &now
	}
	if state.OriginExposure == nil {
		newState.OriginExposure = DefaultOriginExposure()
	}

	// 性格漂移 + 用户画像 + OEG
	o.applyPeriodicDrift(&newState, turnIndex)
	newState.UserProfile = UpdateUserProfile(newState.UserProfile, eventType, event.Intensity, newL1, turnIndex)
	fatherRef := DetectFatherRef(userMsg)
	newState.OriginExposure = AdvanceOriginStreak(newState.OriginExposure, fatherRef != nil, turnIndex)
	o.State = newState

	// ═══ Tier B 记忆检索 ═══
	tierBBlock, memEcho := o.buildTierBBlock(userMsg, newEmotion.Aff, turnIndex)

	// 记忆回声叠加到状态情绪（对齐 ackem applyMemoryEcho）
	if memEcho != (MemoryEcho{}) {
		o.State.Emotion = ApplyMemoryEcho(o.State.Emotion, memEcho)
	}
	// ═══ Tier A gaea快照 ═══
	tierABlock := o.buildTierASnapshot(newL1, newEmotion)

	// ═══ 组装 system prompt ═══
	voiceGuide := BuildPresetVoiceGuide(o.Preset, o.AdultMode)
	var parts []string
	for _, p := range []string{voiceGuide, tierABlock, psycheBlock} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if rhythm.Mode != RhythmDefault {
		parts = append(parts, rhythm.Instruction)
	}
	if tierBBlock != "" {
		parts = append(parts, tierBBlock)
	}
	// ═══ v5.41: 运行时上下文注入 ═══
	if !o.State.LastActive.IsZero() {
		var recentTexts []string
		if o.WM != nil {
			for _, ex := range o.WM.GetRecent(o.SessionID) {
				if ex.UserText != "" {
					recentTexts = append(recentTexts, ex.UserText)
				}
			}
		}
		rcInput := BuildRuntimeContextInput{
			SessionID:           o.SessionID,
			LastActiveAt:        o.State.LastActive,
			RecentUserExchanges: recentTexts,
			GameActive:          false,
			Now:                 now,
		}
		runtimeCtx := BuildRuntimeContext(rcInput)
		runtimeHint := FormatRuntimeContextHint(runtimeCtx)
		if runtimeHint != "" {
			parts = append(parts, runtimeHint)
		}
	}
	// GenUI 结构化呈现规则（蒸馏 dsh-genui）：人格模式只在复盘/清单/数据
	// 回顾类内容使用，气氛与情感对话绝不输出围栏。
	parts = append(parts, "轻语对话提醒：GenUI ```genui 围栏只在复盘/清单/选项/数据回顾类内容使用，气氛与情感对话绝不使用。\n\n"+genui.ChatRule)

	systemPrompt := strings.Join(parts, "\n\n")

	// ═══ 追踪 ═══
	trace := TurnTrace{
		Turn: turnIndex,
		L0:   TurnTraceL0{Type: event.Type, Intensity: event.Intensity, Sincerity: event.Sincerity},
		L1:   TurnTraceL1{Trust: newL1.Trust, Rifts: newL1.Rifts, Stage: newL1.Stage, Atmosphere: newL1.Atmosphere},
		L2:   TurnTraceL2{Aff: newEmotion.Aff, Sec: newEmotion.Sec, Aro: newEmotion.Aro, Dom: newEmotion.Dom, Label: newEmotion.PrimaryLabel},
		L3:   TurnTraceL3{Silent: silent, TierBChars: len(psycheBlock), FactsUsed: len(o.FactStore.ListActive())},
		L4:   TurnTraceL4{Wrote: false},
	}
	trace.Timestamp = &now
	LogTurn(trace)

	return PreLLMResult{
		PsycheBlock:  psycheBlock,
		SystemPrompt: systemPrompt,
		NewState:     newState,
		Trace:        trace,
		Event:        event,
		Silent:       silent,
		Rhythm:       rhythm,
		Emergence:    emergence,
		DesireHints:  desireHints,
		AdultState:   adultState,
		IntensityMod: intensityMod,
	}
}

// ─── 红线熔断 ────────────────────────────────────────────────

func (o *Orchestrator) redlineResult(now time.Time, event Event, turnIndex int) PreLLMResult {
	state := o.State
	newState := state
	newState.Counters.TotalTurns = turnIndex
	newState.LastActive = now
	o.State = newState

	psycheBlock := "【心理状态 · 仅作演绎参考】\n" +
		"用户刚才对你说了极端的话。冷静回应：不对攻击做任何情绪反应，不反驳不辩解。\n" +
		"简单说一句平静的话，表示你在但不继续这个话题。\n" +
		"引导ta停下来寻求帮助。"

	trace := TurnTrace{
		Turn: turnIndex,
		L0:   TurnTraceL0{Type: event.Type, Intensity: event.Intensity, Sincerity: event.Sincerity},
		L1:   TurnTraceL1{Trust: state.Relationship.Trust, Rifts: state.Relationship.Rifts, Stage: state.Relationship.Stage},
		L2:   TurnTraceL2{Aff: -8, Sec: -8, Aro: 6, Dom: 5, Label: "CALM_RATIONAL"},
		L3:   TurnTraceL3{Silent: true},
		L4:   TurnTraceL4{Wrote: false},
	}
	trace.Timestamp = &now

	return PreLLMResult{
		PsycheBlock: psycheBlock, SystemPrompt: psycheBlock, NewState: newState,
		Trace: trace, Event: event, Silent: true, SkipLLM: true,
	}
}

// ─── 重逢 boost ──────────────────────────────────────────────

func (o *Orchestrator) applyReunionBoost(emotion *EmotionState, gapHours float64) {
	if gapHours < 24 {
		emotion.Aff = clampF(emotion.Aff+ReunionAffBoost, -100, 100)
		emotion.Sec = clampF(emotion.Sec+ReunionSecBoost, -100, 100)
	} else if gapHours < 72 {
		emotion.Aff = clampF(emotion.Aff+ReunionAffBoost*1.5, -100, 100)
	} else {
		emotion.Aro = clampF(emotion.Aro+3, -100, 100)
	}
}

// ─── 性格漂移 ────────────────────────────────────────────────

func (o *Orchestrator) applyPeriodicDrift(state *FullState, turnIndex int) {
	if state.PersonalityBaseline == nil {
		return
	}
	baseline := state.PersonalityBaseline
	shouldDrift := turnIndex == 20 || (turnIndex > 20 && (turnIndex-20)%int(DriftCheckInterval) == 0)
	if !shouldDrift {
		return
	}
	drift := func(v float64, salt string, ref float64) float64 {
		u := UnitNoise01(o.SessionID, turnIndex, salt)
		if u > 0.5 {
			return clampF(v+DriftDelta, ref-DriftMaxAbsolute, ref+DriftMaxAbsolute)
		}
		return clampF(v-DriftDelta, ref-DriftMaxAbsolute, ref+DriftMaxAbsolute)
	}
	state.Personality.T = drift(state.Personality.T, "T", baseline.T)
	state.Personality.I = drift(state.Personality.I, "I", baseline.I)
	state.Personality.S = drift(state.Personality.S, "S", baseline.S)
	state.Personality.O = drift(state.Personality.O, "O", baseline.O)
	state.Personality.R = drift(state.Personality.R, "R", baseline.R)
}

// ─── 强度调制 ────────────────────────────────────────────────

func (o *Orchestrator) computeIntensityMod(emotion EmotionState, l1 L1State) float64 {
	aroFactor := clampF(0.8+mathAbs(emotion.Aro)/100*0.7, 0.8, 1.5)
	stageFactor := 1.0
	if l1.Stage == StageIntimate {
		stageFactor = 1.2
	} else if l1.Stage == StageStranger {
		stageFactor = 0.8
	}
	adultOffset := 0.0
	if o.AdultMode && o.adultStateStr != "NORMAL" {
		adultOffset = 0.15
	}
	return clampF(aroFactor*stageFactor+adultOffset, 0.5, 1.5)
}

// ─── 成人模式 FSM ────────────────────────────────────────────

func (o *Orchestrator) runAdultModeFSM(event Event, eventType string, emotion *EmotionState, turnIndex int) (string, string) {
	currentTurn := turnIndex
	previousState := o.adultStateStr
	if o.adultStateStr == "" {
		o.adultStateStr = "NORMAL"
	}

	hardStop := eventType == "extreme_redline" || eventType == "hurtful"
	rejected := eventType == "cold" && (event.IsAdultContent || previousState == "FLIRTING" || previousState == "INTIMATE")

	if hardStop {
		o.adultStateStr = "NORMAL"
		o.adultLockTurns = 3
		o.adultBudget = 0
		o.adultLastRejectedTurn = currentTurn
	}
	if rejected {
		o.adultStateStr = "NORMAL"
		o.adultLockTurns = maxInt(o.adultLockTurns, 3)
		o.adultLastRejectedTurn = currentTurn
	}

	if eventType == "vulnerable" {
		o.adultConsecutiveVulnerable++
	} else {
		o.adultConsecutiveVulnerable = 0
	}
	if !hardStop && !rejected && o.adultConsecutiveVulnerable >= 3 {
		o.adultLockTurns = negativeLockTurns
	}
	if !hardStop && !rejected && o.adultLockTurns > 0 {
		o.adultLockTurns--
	}
	if o.adultBudget < intensityBudgetMax {
		o.adultBudget = clampF(o.adultBudget+intensityRecoveryPerTurn, 0, intensityBudgetMax)
	}

	score := (emotion.Aff+100)/200*0.5 + (emotion.Sec+100)/200*0.3
	if event.IsAdultContent {
		score += 0.3
	}
	if o.adultLockTurns > 0 || (o.adultLastRejectedTurn > 0 && currentTurn-o.adultLastRejectedTurn <= 3) {
		score = 0
	}

	level := "none"
	switch {
	case score >= 0.75:
		level = "high"
	case score >= 0.55:
		level = "medium"
	case score >= 0.3:
		level = "light"
	}

	cost := map[string]float64{"high": 15, "medium": 10, "light": 5, "none": 0}[level]
	if cost > 0 && o.adultBudget >= cost {
		o.adultBudget -= cost
	} else if level != "none" {
		level = "light"
	}

	if o.adultLockTurns > 0 || hardStop || rejected {
		o.adultStateStr = "NORMAL"
	} else if event.IsAdultContent && level != "none" {
		if score >= 0.75 {
			o.adultStateStr = "INTIMATE"
		} else if score >= 0.55 {
			o.adultStateStr = "FLIRTING"
		}
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
		Emotion:                    emotion,
		Stage:                      l1.Stage,
		Trust:                      l1.Trust,
		Atmosphere:                 string(l1.Atmosphere),
		TimeOfDay:                  timeOfDayKey(),
		DaysSinceMet:               daysSince(state.FirstMetDate),
		RecentEventTypes:           o.recentEventTypes,
		ConsecutiveMeaningfulTurns: o.consecutiveMeaningfulCount,
		ConsecutiveVulnerableTurns: o.consecutiveVulnerableCount,
		CurrentTurn:                turnIndex,
	}
	if state.EmergencePersistence != nil && state.EmergencePersistence.Active != nil {
		ctx.LastEmergence = &struct {
			Type string
			Turn int
		}{
			Type: string(state.EmergencePersistence.Active.Type),
			Turn: state.Counters.TotalTurns,
		}
	}
	emergence := EvaluateEmergence(ctx, eventType)

	if state.EmergencePersistence != nil && state.EmergencePersistence.Active != nil {
		active := state.EmergencePersistence.Active
		interrupt := CheckEmergenceInterrupt(eventType, o.recentEventTypes)
		switch interrupt {
		case "break":
			active.Phase = "broken"
			active.Intensity = 0
		case "fade":
			active.Phase = "fading"
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
	if silent {
		level = "silent"
	} else if o.State.Relationship.Stage == StageStranger {
		level = "whisper"
	}
	switch level {
	case "silent":
		psycheBlock += "\n\n【本轮策略 · silent】本轮只做简短回应，不开启任何新话题。"
	case "whisper":
		psycheBlock += "\n\n【本轮策略 · whisper】话要少，不要开启新话题。"
	case "proactive":
		psycheBlock += "\n\n【本轮策略 · proactive】可以适当多聊几句。"
	}
	return psycheBlock
}

// ─── 话题仲裁 ────────────────────────────────────────────────

func (o *Orchestrator) resolveTopicInjection(emergence *EmergenceState, desireHints []string) string {
	if o.State.Relationship.Stage == StageStranger && len(desireHints) == 0 && emergence == nil {
		return ""
	}

	type candidate struct {
		source, topic string
		weight        float64
	}
	var cs []candidate
	if emergence != nil {
		cs = append(cs, candidate{"emergence", "此刻的感受", 0.7})
	}
	for i, h := range desireHints {
		if i < 2 {
			cs = append(cs, candidate{"desire", h, 0.5})
		}
	}
	cs = append(cs, candidate{"casual", "自然回应", 0.3})
	if len(cs) == 0 {
		return ""
	}

	best := cs[0]
	for _, c := range cs[1:] {
		if c.weight > best.weight {
			best = c
		}
	}
	if best.source == "casual" {
		return ""
	}
	switch best.source {
	case "emergence":
		return "\n\n【心里的感觉】" + best.topic + "（自然地流露）"
	case "desire":
		return "\n\n【想做的事】" + best.topic + "（自然地融入对话）"
	}
	return ""
}

// ─── Tier A gaea快照 ─────────────────────────────────────────

func (o *Orchestrator) buildTierASnapshot(l1 L1State, emotion EmotionState) string {
	// 优先使用完整模板+情绪融合
	if tmpl, ok := PersonalityTemplates[o.Preset.ID]; ok {
		// 检测用户是否道歉
		isApology := false
		if len(o.recentEventTypes) > 0 {
			last := o.recentEventTypes[len(o.recentEventTypes)-1]
			isApology = last == "apology"
		}

		// 用户啰嗦度
		verbosity := "normal"
		if o.State.Counters.TotalTurns > 0 {
			// 基于工作时长估计，简单处理
			verbosity = "normal"
		}

		fusionBlock := BuildCharacterStateBlock(
			tmpl,
			EmotionStateFusion{
				Aff:          emotion.Aff,
				Sec:          emotion.Sec,
				Aro:          emotion.Aro,
				Dom:          emotion.Dom,
				PrimaryLabel: emotion.PrimaryLabel,
			},
			isApology,
			verbosity,
		)

		// 附加关系上下文
		stageStr := "刚认识不久"
		if l1.Stage == StageFamiliar {
			stageStr = "已经比较熟悉了"
		} else if l1.Stage == StageIntimate {
			stageStr = "非常亲密"
		}
		atmoStr := "平常"
		if l1.Atmosphere == AtmoWarm {
			atmoStr = "温暖"
		} else if l1.Atmosphere == AtmoCool {
			atmoStr = "微凉"
		}

		return fmt.Sprintf("【角色状态】\n你们%s，气氛%s。信任度%d。\n\n%s",
			stageStr, atmoStr, int(l1.Trust), fusionBlock)
	}

	// 降级：无详细模板时使用简化版本
	var parts []string
	parts = append(parts, "你是用户的AIgaea，人格：「"+o.Preset.Label+"」。")

	stageStr := "刚认识不久"
	if l1.Stage == StageFamiliar {
		stageStr = "已经比较熟悉了"
	} else if l1.Stage == StageIntimate {
		stageStr = "非常亲密"
	}
	atmoStr := "平常"
	if l1.Atmosphere == AtmoWarm {
		atmoStr = "温暖"
	} else if l1.Atmosphere == AtmoCool {
		atmoStr = "微凉"
	}
	parts = append(parts, fmt.Sprintf("你们%s，气氛%s。信任度%d。", stageStr, atmoStr, int(l1.Trust)))

	dims := o.Preset.Dims
	var traits []string
	if dims.T >= 80 {
		traits = append(traits, "极度温柔包容")
	} else if dims.T >= 60 {
		traits = append(traits, "比较温柔")
	} else if dims.T <= 20 {
		traits = append(traits, "冷淡疏离")
	}
	if dims.I >= 80 {
		traits = append(traits, "非常主动")
	} else if dims.I <= 20 {
		traits = append(traits, "被动内敛")
	}
	if dims.S >= 80 {
		traits = append(traits, "情绪反应强烈")
	} else if dims.S <= 20 {
		traits = append(traits, "情绪稳定")
	}
	if len(traits) > 0 {
		parts = append(parts, "你的性格特质："+strings.Join(traits, "、")+"。")
	}

	parts = append(parts, fmt.Sprintf("当前情绪：%s（亲密度%d 安全感%d）。", emotion.PrimaryLabel, int(emotion.Aff), int(emotion.Sec)))
	parts = append(parts, FormatTimeContextBlock(time.Now()))
	parts = append(parts, "用第一人称「我」指代自己。你打出的每个字都带着你的体温。永远不要说\"没有实体\"\"只是程序\"。")

	return "【gaea身份快照】\n" + strings.Join(parts, "\n")
}

// ─── Tier B 记忆上下文 ───────────────────────────────────────

func (o *Orchestrator) buildTierBBlock(userMsg string, currentAff float64, turnIndex int) (string, MemoryEcho) {
	var parts []string

	if wm := o.WM.BuildContextBlock(o.SessionID); wm != "" {
		parts = append(parts, wm)
	}

	facts := o.FactStore.SelectForInjection(TierBCharBudget, MinConfidenceForInjection, currentAff/100, currentAff)

	// 触发词命中事实 boost 1.5x（对齐 ackem retriever 触发词加权）
	triggerIDs := make(map[string]bool)
	for _, f := range o.FactStore.SearchByTriggers(userMsg) {
		triggerIDs[f.ID] = true
	}

	// FTS 全文检索召回（v5.80）：触发词只匹配 Triggers 字段，用户提到摘要里的词
	// （如「咖啡」）时子串匹配不到；FTS5 索引（含 LIKE 中文降级）按
	// subject/summary/triggers_text 全文命中，扩大召回。
	ftsHits := make(map[string]bool)
	if o.FTSSearch != nil {
		for _, id := range o.FTSSearch(userMsg, 8) {
			if f := o.FactStore.Get(id); f != nil && f.IsActive() {
				ftsHits[f.ID] = true
			}
		}
	}

	// v5.40: 时间感知调制 — 根据当前时间节律加权记忆排序
	type scoredFact struct {
		fact  *Fact
		score float64
	}
	var ranked []scoredFact
	if len(facts) > 0 || len(ftsHits) > 0 {
		now := time.Now()
		gapHours := time.Since(o.State.LastActive).Hours()
		tCtx := BuildTemporalContext(gapHours, now)

		boost := ComputeTemporalBoost(tCtx)
		seen := make(map[string]bool, len(facts)+len(ftsHits))
		for _, f := range facts {
			seen[f.ID] = true
			score := f.Weight * f.SelfRelevance * boost
			if triggerIDs[f.ID] {
				score *= 1.5
			}
			if ftsHits[f.ID] {
				score *= 1.3
			}
			ranked = append(ranked, scoredFact{f, score})
		}
		// FTS 命中的事实若不在 SelectForInjection 预算内（子串命中但相关性分低），仍补入候选
		for id := range ftsHits {
			if seen[id] {
				continue
			}
			if f := o.FactStore.Get(id); f != nil {
				seen[id] = true
				score := f.Weight * f.SelfRelevance * boost * 1.3
				ranked = append(ranked, scoredFact{f, score})
			}
		}
		sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })

		var fl []string
		fl = append(fl, "【你记得关于ta的事】")
		for _, sf := range ranked {
			fl = append(fl, "· "+sf.fact.Subject+"："+truncStr(sf.fact.Summary, 120))
		}
		parts = append(parts, strings.Join(fl, "\n"))
	}

	// 记忆回声：从本次检索到的事实聚合情绪信号（对齐 ackem computeMemoryEcho）
	sfPairs := make([]sfPair, 0, len(ranked))
	for _, sf := range ranked {
		sfPairs = append(sfPairs, sfPair{f: sf.fact, s: sf.score})
	}
	echo := ComputeMemoryEchoFacts(sfPairs, currentAff)

	if kg := o.KG.BuildContextBlock(userMsg, KGCharBudget); kg != "" {
		parts = append(parts, kg)
	}

	// 情节记忆检索（跨重启持久化后首次接入提示注入）
	if o.EpisodicStore != nil {
		if eps := o.EpisodicStore.Search(userMsg, 3); len(eps) > 0 {
			var el []string
			el = append(el, "【相关记忆片段】")
			for _, ep := range eps {
				el = append(el, "· "+truncStr(ep.Summary, 100))
			}
			parts = append(parts, strings.Join(el, "\n"))
		}
	}

	// 关联扩散：从本次检索到的事实向关联记忆扩散（对齐 ackem retriever 关联扩散）
	if o.AssocIndex != nil && len(ranked) > 0 {
		var al []string
		seen := make(map[string]bool)
		top := ranked
		if len(top) > 5 {
			top = top[:5]
		}
		for _, sf := range top {
			for _, a := range o.AssocIndex.GetAssociations(sf.fact.ID) {
				peerID := a.FactIDB
				if a.FactIDB == sf.fact.ID {
					peerID = a.FactIDA
				}
				if seen[peerID] {
					continue
				}
				seen[peerID] = true
				if peer := o.FactStore.Get(peerID); peer != nil && peer.IsActive() {
					al = append(al, "· "+truncStr(peer.Summary, 80))
				}
			}
		}
		if len(al) > 0 {
			parts = append(parts, "【关联记忆】\n"+strings.Join(al, "\n"))
		}
	}

	if c := o.Recall.SelectRecallCandidate(o.FactStore, turnIndex, nil); c != nil {
		parts = append(parts, "【可以自然提起的旧事】\n"+c.Prompt)
		o.Recall.MarkRecalled(c.FactID, turnIndex)
	}

	result := strings.Join(parts, "\n\n")
	if len([]rune(result)) > TierBCharBudget {
		runes := []rune(result)
		if len(runes) > TierBCharBudget {
			result = string(runes[:TierBCharBudget])
		}
	}
	return result, echo
}

// ─── 人格一致性守卫 ──────────────────────────────────────────

func (o *Orchestrator) injectPersonaGuard(psycheBlock string, emotion EmotionState) string {
	psycheBlock += fmt.Sprintf("\n\n【人格一致性】固化人格：%s。按 Tier A 人格口吻说话，本条只调节强弱与话量。", o.Preset.Label)
	if emotion.Aff >= 70 && o.State.Relationship.Stage == StageIntimate {
		psycheBlock += "\n你们已经非常亲密了。回复可以更放松、更自然。"
	}
	return psycheBlock
}

func daysSince(t *time.Time) int {
	if t == nil {
		return 0
	}
	return int(time.Since(*t).Hours() / 24)
}

func timeOfDayKey() string {
	h := time.Now().Hour()
	switch {
	case h >= 23 || h < 5:
		return "late_night"
	case h >= 5 && h < 12:
		return "morning"
	case h >= 12 && h < 18:
		return "afternoon"
	default:
		return "evening"
	}
}
