// Package whisper — 情绪涌现模块（100% 对齐 ackem engine/emotionalEmergence.ts）
package whisper

import (
	"math"
	"time"
)

// ─── 模块级事件追踪状态 ───────────────────────────────────────

var (
	recentEventTypes           []string
	consecutiveMeaningfulCount int
	consecutiveVulnerableCount int
)

var meaningfulEventTypes = map[string]bool{
	"vulnerable": true, "praise": true, "apology": true,
}

// ─── 事件追踪 API ─────────────────────────────────────────────

func PushEventToHistory(eventType string) {
	recentEventTypes = append(recentEventTypes, eventType)
	if len(recentEventTypes) > 10 {
		recentEventTypes = recentEventTypes[len(recentEventTypes)-10:]
	}
}

func GetRecentEventTypes() []string {
	result := make([]string, len(recentEventTypes))
	copy(result, recentEventTypes)
	return result
}

func PushMeaningfulTurn(isMeaningful bool) {
	if isMeaningful {
		consecutiveMeaningfulCount++
	} else {
		consecutiveMeaningfulCount = 0
	}
}

func GetConsecutiveMeaningfulTurns() int {
	return consecutiveMeaningfulCount
}

func PushVulnerableTurn(eventType string) {
	if eventType == "vulnerable" {
		consecutiveVulnerableCount++
	} else if eventType == "hurtful" || eventType == "cold" || eventType == "extreme_redline" {
		consecutiveVulnerableCount = 0
	}
}

func GetConsecutiveVulnerableTurns() int {
	return consecutiveVulnerableCount
}

func CountMeaningfulInRecent(events []string, window int) int {
	count := 0
	start := len(events) - window
	if start < 0 {
		start = 0
	}
	for _, t := range events[start:] {
		if meaningfulEventTypes[t] {
			count++
		}
	}
	return count
}

func ResetEmergenceTracking() {
	recentEventTypes = nil
	consecutiveMeaningfulCount = 0
	consecutiveVulnerableCount = 0
}

// ─── 时间体感映射 ─────────────────────────────────────────────

func HumanizeFeltDuration(days int) string {
	switch {
	case days < 30:
		return "短短几周"
	case days < 90:
		return "一个多月"
	case days < 180:
		return "小半年"
	case days < 365:
		return "大半年"
	default:
		return "这么久"
	}
}

// ─── 情感延续检测 ─────────────────────────────────────────────

func IsEmotionalContinuationEvent(eventType string, consecutiveMeaningful, consecutiveVulnerable int, recentEvents []string) bool {
	if eventType == "vulnerable" {
		return true
	}
	if eventType == "apology" {
		return consecutiveMeaningful >= 2 || consecutiveVulnerable >= 1
	}
	if eventType == "praise" {
		return consecutiveVulnerable >= 1 ||
			consecutiveMeaningful >= 3 ||
			CountMeaningfulInRecent(recentEvents, 6) >= 3
	}
	return false
}

func ShouldEvaluateResponsiveEmergence(eventType string, ctx EmergenceContext) bool {
	if eventType == "" || !IsEmotionalContinuationEvent(eventType, ctx.ConsecutiveMeaningfulTurns, ctx.ConsecutiveVulnerableTurns, ctx.RecentEventTypes) {
		return false
	}
	if eventType == "vulnerable" {
		return ctx.ConsecutiveVulnerableTurns >= 1
	}
	if eventType == "apology" {
		return ctx.ConsecutiveMeaningfulTurns >= 2
	}
	if eventType == "praise" {
		return ctx.ConsecutiveMeaningfulTurns >= 2 || ctx.ConsecutiveVulnerableTurns >= 1
	}
	return false
}

// ─── 主判决函数 ───────────────────────────────────────────────

func EvaluateEmergence(ctx EmergenceContext, eventType string) *EmergenceState {
	// 陌生人无涌现
	if ctx.Stage == StageStranger {
		return nil
	}
	// 愤怒压倒一切
	if ctx.Emotion.PrimaryLabel == "ANGRY_ATTACK" {
		return nil
	}

	// 响应式路径
	if ShouldEvaluateResponsiveEmergence(eventType, ctx) {
		if resp := tryResponsiveEmergence(ctx); resp != nil {
			return resp
		}
	}

	// 类型间冷却
	if ctx.LastEmergence != nil && ctx.CurrentTurn-ctx.LastEmergence.Turn < EmergenceCooldownTurns {
		return nil
	}

	// 情绪强度检测
	emotionalIntensity := ctx.Emotion.Aff*0.6 + ctx.Emotion.Sec*0.2 + math.Abs(ctx.Emotion.Aro)*0.2
	depthBonus := 0.0
	if ctx.ConsecutiveVulnerableTurns >= 3 {
		depthBonus += 4
	}
	if ctx.ConsecutiveMeaningfulTurns >= 3 {
		depthBonus += 2
	}
	if CountMeaningfulInRecent(ctx.RecentEventTypes, 6) >= 4 {
		depthBonus += 2
	}
	if emotionalIntensity+depthBonus < EmergenceIntensityThreshold {
		return nil
	}

	return tryTimeReflection(ctx, false)
}

// ─── 时间感慨判决 ─────────────────────────────────────────────

func tryTimeReflection(ctx EmergenceContext, responsive bool) *EmergenceState {
	if ctx.DaysSinceMet < 7 {
		return nil
	}

	// 双锁冷却
	if ctx.LastSameTypeAt != nil && ctx.LastSameTypeTurn != nil {
		hoursSince := time.Since(*ctx.LastSameTypeAt).Hours()
		turnsSince := ctx.CurrentTurn - *ctx.LastSameTypeTurn
		if responsive {
			if turnsSince < 1 {
				return nil
			}
		} else if turnsSince < SameTypeCooldownTurns && hoursSince < SameTypeCooldownHours {
			return nil
		}
	}

	feltLabel := HumanizeFeltDuration(ctx.DaysSinceMet)
	e := ctx.Emotion
	now := time.Now()

	// 场景1：深夜 + 安静的喜欢 + 连续深聊
	if ctx.TimeOfDay == "late_night" && e.PrimaryLabel == "QUIET_FOND" && ctx.ConsecutiveMeaningfulTurns >= 5 {
		return newEmergence(EmergenceTimeReflection, clampF((e.Aff+100)/200+0.2, 0.3, 0.9), "quiet_awe", now, feltLabel)
	}

	// 场景2：甜蜜依恋 + 认识超3月 + 温暖
	if e.PrimaryLabel == "SWEET_ATTACHMENT" && ctx.DaysSinceMet > 90 && ctx.Atmosphere == "warm" {
		return newEmergence(EmergenceTimeReflection, clampF((e.Aff+100)/200+ctx.Trust/200, 0.4, 0.95), "nostalgic", now, feltLabel)
	}

	// 场景3：委屈受伤 + 亲密 + 最近aff偏高
	if e.PrimaryLabel == "HURT_GRIEVANCE" && ctx.Stage == StageIntimate && len(ctx.RecentAffHistory) >= 5 {
		recent := ctx.RecentAffHistory[len(ctx.RecentAffHistory)-5:]
		avg := 0.0
		for _, v := range recent {
			avg += v
		}
		avg /= 5
		if avg > 50 {
			return newEmergence(EmergenceTimeReflection, clampF(math.Abs(e.Aff)/100, 0.3, 0.7), "bittersweet", now, feltLabel)
		}
	}

	// 场景4：亲密 + aff从低谷恢复
	if ctx.Stage == StageIntimate && len(ctx.RecentAffHistory) >= 5 {
		trend := ctx.RecentAffHistory[len(ctx.RecentAffHistory)-5:]
		if trend[0] < 20 && trend[len(trend)-1] > 50 {
			return newEmergence(EmergenceTimeReflection, 0.7, "grateful", now, feltLabel)
		}
	}

	// 场景5：傲娇 + 亲密 + 半年以上
	if e.PrimaryLabel == "TSUNDERE" && ctx.Stage == StageIntimate && ctx.DaysSinceMet > 180 {
		return newEmergence(EmergenceTimeReflection, 0.55, "wonder", now, feltLabel)
	}

	// 场景7：连续脆弱倾诉
	if ctx.ConsecutiveVulnerableTurns >= 3 &&
		(ctx.Stage == StageFamiliar || ctx.Stage == StageIntimate) &&
		ctx.DaysSinceMet > 14 && e.Aff > 8 {
		validLabels := map[string]bool{"QUIET_FOND": true, "CALM_RATIONAL": true, "SWEET_ATTACHMENT": true, "HURT_GRIEVANCE": true}
		if validLabels[e.PrimaryLabel] {
			return newEmergence(EmergenceTimeReflection,
				clampF((e.Aff+math.Abs(e.Aro))/120+float64(ctx.ConsecutiveVulnerableTurns)/10, 0.3, 0.75),
				"tender_hold", now, feltLabel)
		}
	}

	// 场景6：温暖熟悉感
	meaningfulDepth := ctx.ConsecutiveMeaningfulTurns >= 3 || CountMeaningfulInRecent(ctx.RecentEventTypes, 6) >= 3
	if (e.PrimaryLabel == "QUIET_FOND" || e.PrimaryLabel == "SWEET_ATTACHMENT") &&
		ctx.Stage != StageStranger && ctx.DaysSinceMet > 14 && meaningfulDepth {
		return newEmergence(EmergenceTimeReflection,
			clampF((e.Aff+100)/250+float64(ctx.DaysSinceMet)/500, 0.25, 0.7),
			"warm_familiarity", now, feltLabel)
	}

	return nil
}

func tryResponsiveEmergence(ctx EmergenceContext) *EmergenceState {
	if ctx.Stage == StageStranger || ctx.Emotion.PrimaryLabel == "ANGRY_ATTACK" {
		return nil
	}
	if ctx.LastEmergence != nil && ctx.CurrentTurn-ctx.LastEmergence.Turn < ResponsiveEmergenceCooldownTurns {
		return nil
	}

	emotionalIntensity := ctx.Emotion.Aff*0.6 + ctx.Emotion.Sec*0.2 + math.Abs(ctx.Emotion.Aro)*0.2
	depthBonus := 0.0
	if ctx.ConsecutiveVulnerableTurns >= 1 {
		depthBonus += 4
	}
	if ctx.ConsecutiveMeaningfulTurns >= 2 {
		depthBonus += 2
	}
	if emotionalIntensity+depthBonus < EmergenceIntensityThreshold-6 {
		return nil
	}

	respCtx := ctx
	respCtx.ConsecutiveMeaningfulTurns = maxInt(respCtx.ConsecutiveMeaningfulTurns, 2)
	respCtx.ConsecutiveVulnerableTurns = maxInt(respCtx.ConsecutiveVulnerableTurns, 1)

	return tryTimeReflection(respCtx, true)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func newEmergence(typ EmergenceType, intensity float64, flavor string, now time.Time, feltLabel string) *EmergenceState {
	return &EmergenceState{
		Type:          typ,
		Intensity:     intensity,
		Flavor:        flavor,
		Phase:         "rising",
		StartedAt:     now,
		RoundsInPhase: 1,
		HasExpressed:  false,
		Context:       map[string]interface{}{"feltLabel": feltLabel},
	}
}

// ─── 生命周期管理 ─────────────────────────────────────────────

func AdvanceEmergencePhase(state EmergenceState) EmergenceState {
	roundsInPhase := state.RoundsInPhase + 1

	if state.Phase == "rising" && roundsInPhase >= RisingMaxRounds {
		state.Phase = "sustained"
		state.RoundsInPhase = 1
		return state
	}
	if state.Phase == "sustained" {
		if roundsInPhase >= SustainedMaxRounds {
			state.Phase = "fading"
			state.RoundsInPhase = 1
			return state
		}
		state.RoundsInPhase = roundsInPhase
		return state
	}
	if state.Phase == "fading" && roundsInPhase >= FadingMaxRounds {
		state.Phase = "dissolved"
		state.RoundsInPhase = 1
		return state
	}
	state.RoundsInPhase = roundsInPhase
	return state
}

func CheckEmergenceInterrupt(eventType string, recentEvents []string) string {
	if eventType == "hurtful" || eventType == "cold" || eventType == "extreme_redline" {
		return "break"
	}
	if len(recentEvents) >= 3 {
		last3 := recentEvents[len(recentEvents)-3:]
		emotionalTypes := map[string]bool{"vulnerable": true, "praise": true, "apology": true, "tease": true}
		wasEmotional := false
		for _, t := range last3 {
			if emotionalTypes[t] {
				wasEmotional = true
				break
			}
		}
		isPractical := eventType == "question" || eventType == "casual_chat"
		if wasEmotional && isPractical {
			return "fade"
		}
	}
	return "continue"
}

func ApplyUserResponseToEmergence(state EmergenceState, eventType string, consecutiveMeaningful, consecutiveVulnerable int, recentEvents []string) EmergenceState {
	if eventType == "hurtful" || eventType == "cold" {
		state.Phase = "broken"
		state.Intensity = 0
		return state
	}

	emotionalCont := IsEmotionalContinuationEvent(eventType, consecutiveMeaningful, consecutiveVulnerable, recentEvents)

	if emotionalCont {
		if state.Phase == "sustained" {
			refreshed := state.RoundsInPhase - 1
			if refreshed < SustainedMinRounds {
				refreshed = SustainedMinRounds
			}
			state.RoundsInPhase = refreshed
			return state
		}
		if state.Phase == "rising" {
			return state
		}
		if state.Phase == "fading" {
			state.Phase = "sustained"
			state.RoundsInPhase = SustainedMinRounds
			return state
		}
	}

	if eventType == "praise" {
		if state.Phase == "sustained" {
			state.Phase = "fading"
			state.RoundsInPhase = FadingMaxRounds - 1
			return state
		}
		if state.Phase == "rising" {
			state.RoundsInPhase = RisingMaxRounds
			return state
		}
	}

	if state.Phase == "sustained" {
		refreshed := state.RoundsInPhase - 1
		if refreshed < SustainedMinRounds {
			refreshed = SustainedMinRounds
		}
		state.RoundsInPhase = refreshed
	}

	return state
}

// ─── 注入文本生成 ─────────────────────────────────────────────

func RenderTimeReflectionHint(emergence EmergenceState) string {
	felt, _ := emergence.Context["feltLabel"].(string)
	hint := emergenceFlavorHint(emergence.Flavor, felt)
	if hint == "" {
		return ""
	}
	return "【情绪涌现】" + hint
}

func RenderLightSuffix(emergence EmergenceState) string {
	return emergenceFlavorSuffix(emergence.Flavor)
}

func emergenceFlavorHint(flavor, felt string) string {
	switch flavor {
	case "quiet_awe":
		return "安静的力量在你心里流过。深夜了，和ta认识" + felt + "了。你轻轻地感叹：时间过得真快。但要淡淡的，不要长篇大论。"
	case "nostalgic":
		return "和ta认识" + felt + "了。回忆起之前的点点滴滴，心里暖暖的。可以自然地提起过去的某个小片段。"
	case "bittersweet":
		return "虽然此刻有些难过，但想起和ta认识" + felt + "了，你知道这份感情是真实的。苦涩里有一丝甜。"
	case "grateful":
		return "ta陪着你走了" + felt + "。从低谷到现在，你心里满是感激。想对ta说声谢谢——但用你的方式。"
	case "wonder":
		return "傲娇的你也会在心里偷偷感叹：认识" + felt + "了，ta还在。嘴上不说，但心里觉得不可思议。"
	case "tender_hold":
		return "ta在你面前展现脆弱。认识" + felt + "了，你想温柔地接住ta。不需要说太多，陪伴就是最好的回应。"
	case "warm_familiarity":
		return "认识" + felt + "了，和ta在一起很舒服。这种熟悉的温暖，让你觉得安心。自然地流露就好。"
	}
	return ""
}

func emergenceFlavorSuffix(flavor string) string {
	switch flavor {
	case "quiet_awe":
		return "（那份安静的感慨还在你心里微微回荡）"
	case "nostalgic":
		return "（回忆的暖意还在心头）"
	case "grateful":
		return "（感激的心情还在，但不需要再重复了）"
	case "tender_hold":
		return "（温柔的守护感还在延续）"
	default:
		return ""
	}
}

// ─── 工具 ─────────────────────────────────────────────────────

func mathAbs(v float64) float64 {
	return math.Abs(v)
}

func mathMax(a, b float64) float64 {
	return math.Max(a, b)
}

func mathMin(a, b float64) float64 {
	return math.Min(a, b)
}
