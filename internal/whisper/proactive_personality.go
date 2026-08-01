// Package whisper — proactive_personality.go
// 100% 对齐 ackem companion/proactivePersonalityContext.ts
// gaea主动消息：人格上下文构建 + 消息类型选择 + 骚扰间隔

package whisper

import "math/rand"

// ─── ProactiveMessageKind ────────────────────────────────────

type ProactiveMessageKind string

const (
	KindCheckIn      ProactiveMessageKind = "check_in"
	KindMemoryEcho   ProactiveMessageKind = "memory_echo"
	KindTimeGreet    ProactiveMessageKind = "time_greet"
	KindMissYou      ProactiveMessageKind = "miss_you"
	KindPlayfulNudge ProactiveMessageKind = "playful_nudge"
)

// ─── BuildProactivePersonalityBlock ──────────────────────────

// BuildProactivePersonalityBlock 构建主动消息的人格上下文块
func BuildProactivePersonalityBlock(presetID string, aff float64, adultMode, harass bool) string {
	preset := GetPreset(presetID)
	if preset == nil {
		return ""
	}
	tmpl := PersonalityTemplates[presetID]

	voiceGuide := BuildPresetVoiceGuide(*preset, adultMode)
	stage := inferStageFromAff(aff)

	var lines []string

	if tmpl.ID != "" {
		lines = append(lines, BuildPersonalitySection(presetID, stage))
	}
	if voiceGuide != "" {
		lines = append(lines, "【口吻演绎】"+voiceGuide)
	}
	if tmpl.ID != "" && len(tmpl.Prohibitions) > 0 {
		n := len(tmpl.Prohibitions)
		if n > 6 {
			n = 6
		}
		lines = append(lines, "【禁止】"+joinStrings(tmpl.Prohibitions[:n], "、"))
	}

	examples := selectExamplesByAff(tmpl, aff, 3)
	if len(examples) > 0 {
		lines = append(lines, "【示例】"+joinStrings(examples, " / "))
	}

	label := preset.Label
	if tmpl.Label != "" {
		label = tmpl.Label
	}
	lines = append(lines, "【主动消息】用户暂时没回，你要主动发一条短消息。必须像「"+label+"」本人说话，禁止通用温柔助手/客服腔。")

	if harass {
		I := preset.I
		switch {
		case I < 35:
			lines = append(lines, "【骚扰模式】低主动人格也要用极短、克制的方式表达在意，不要突然变话痨撒娇。")
		case I >= 70:
			lines = append(lines, "【骚扰模式】高主动人格可以更直接地黏人、调侃或表达想念。")
		default:
			lines = append(lines, "【骚扰模式】按人设自然程度主动，不要脱离语癖与说话方式。")
		}
	}

	return joinNonEmpty(lines, "\n")
}

// ─── PickPersonalityProactiveFallback ────────────────────────

// PickPersonalityProactiveFallback 从人格示例中选取 fallback 消息
func PickPersonalityProactiveFallback(presetID string, aff float64, harass bool) string {
	tmpl := PersonalityTemplates[presetID]
	if tmpl.ID == "" {
		return "在吗？"
	}
	effectiveAff := aff
	if harass && effectiveAff < 55 {
		effectiveAff = 55
	}
	examples := selectExamplesByAff(tmpl, effectiveAff, 6)
	if len(examples) > 0 {
		return examples[rand.Intn(len(examples))]
	}
	if len(tmpl.ExamplesMedium) > 0 {
		return tmpl.ExamplesMedium[0]
	}
	return "在吗？"
}

// ─── ShouldHarassTick ────────────────────────────────────────

// ShouldHarassTickForPersonality 低主动人格跳过骚扰 tick
func ShouldHarassTickForPersonality(presetID string) bool {
	I := 50.0
	if p := GetPreset(presetID); p != nil {
		I = p.I
	}
	r := rand.Float64()
	switch {
	case I >= 70:
		return true
	case I >= 50:
		return r < 0.85
	case I >= 30:
		return r < 0.55
	default:
		return r < 0.25
	}
}

// ─── PickCompanionProactiveKind ──────────────────────────────

// PickCompanionProactiveKind 选择主动消息类型
func PickCompanionProactiveKind(presetID string, factExists bool, aff float64, stage RelationshipStage, harass bool) ProactiveMessageKind {
	I := 50.0
	if p := GetPreset(presetID); p != nil {
		I = p.I
	}

	if harass {
		var pool []ProactiveMessageKind
		switch {
		case I >= 70:
			pool = []ProactiveMessageKind{KindPlayfulNudge, KindPlayfulNudge, KindMissYou, KindMissYou}
		case I >= 40:
			pool = []ProactiveMessageKind{KindPlayfulNudge, KindMissYou, KindCheckIn}
		default:
			pool = []ProactiveMessageKind{KindCheckIn, KindMemoryEcho}
			if I >= 25 {
				pool = append(pool, KindPlayfulNudge)
			}
		}
		if factExists {
			pool = append(pool, KindMemoryEcho)
		}
		if aff > 20 && stage != StageStranger && I >= 35 {
			pool = append(pool, KindMissYou)
		}
		return pool[rand.Intn(len(pool))]
	}

	var pool []ProactiveMessageKind
	switch {
	case I >= 60:
		pool = []ProactiveMessageKind{KindCheckIn, KindTimeGreet, KindPlayfulNudge, KindPlayfulNudge}
	case I >= 35:
		pool = []ProactiveMessageKind{KindCheckIn, KindTimeGreet, KindPlayfulNudge}
	default:
		pool = []ProactiveMessageKind{KindCheckIn, KindMemoryEcho, KindTimeGreet}
	}
	if factExists {
		pool = append(pool, KindMemoryEcho, KindMemoryEcho)
	}
	if aff > 25 && stage != StageStranger && I >= 40 {
		pool = append(pool, KindMissYou, KindMissYou)
	}
	return pool[rand.Intn(len(pool))]
}

// ─── 辅助 ────────────────────────────────────────────────────

func inferStageFromAff(aff float64) RelationshipStage {
	if aff > 50 {
		return StageIntimate
	}
	if aff > 20 {
		return StageFamiliar
	}
	return StageStranger
}

func selectExamplesByAff(tmpl PersonalityTemplate, aff float64, count int) []string {
	var examples []string
	switch {
	case aff > 50:
		examples = tmpl.ExamplesHigh
	case aff > 20:
		examples = tmpl.ExamplesMedium
	default:
		examples = tmpl.ExamplesLow
	}
	if len(examples) > count {
		// 随机选取 count 个
		perm := rand.Perm(len(examples))
		var picked []string
		for i := 0; i < count; i++ {
			picked = append(picked, examples[perm[i]])
		}
		return picked
	}
	return examples
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for i := 1; i < len(ss); i++ {
		result += sep + ss[i]
	}
	return result
}
