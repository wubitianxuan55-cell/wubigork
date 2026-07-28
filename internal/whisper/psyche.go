// Package whisper — L3 心理状态块（100% 对齐 ackem engine/psyche.ts）
package whisper

import (
	"math/rand"
	"strings"
)

// ─── 情绪标签中文映射 ─────────────────────────────────────────

var labelZH = map[string]string{
	"SWEET_ATTACHMENT": "甜蜜依恋",
	"SHY_HEARTBEAT":    "害羞心动",
	"TSUNDERE":         "傲娇",
	"HURT_GRIEVANCE":   "委屈受伤",
	"ANGRY_ATTACK":     "愤怒反击",
	"COLD_DETACHED":    "冷淡疏离",
	"FEARFUL_OBEDIENT": "不安顺从",
	"QUIET_FOND":       "安静的喜欢",
	"CALM_RATIONAL":    "平静理性",
}

// ─── 情绪→表达参数映射 ───────────────────────────────────────

// EmoToExpression 情绪标签→表达参数
func EmoToExpression(label string, _ RelationshipStage) ExpressionParams {
	switch label {
	case "SWEET_ATTACHMENT":
		return ExpressionParams{Mode: "NORMAL", Proximity: "CLOSE", Tone: "warm_intimate", Length: "MEDIUM"}
	case "SHY_HEARTBEAT":
		return ExpressionParams{Mode: "NORMAL", Proximity: "CLOSE", Tone: "shy_hesitant", Length: "SHORT"}
	case "TSUNDERE":
		return ExpressionParams{Mode: "NORMAL", Proximity: "NEUTRAL", Tone: "tsundere", Length: "SHORT"}
	case "HURT_GRIEVANCE":
		return ExpressionParams{Mode: "NORMAL", Proximity: "COOL", Tone: "plaintive", Length: "MEDIUM"}
	case "ANGRY_ATTACK":
		return ExpressionParams{Mode: "NORMAL", Proximity: "DEFENSIVE", Tone: "sharp", Length: "SHORT"}
	case "COLD_DETACHED":
		return ExpressionParams{Mode: "SILENT_CANDIDATE", Proximity: "DEFENSIVE", Tone: "flat", Length: "SHORT"}
	case "FEARFUL_OBEDIENT":
		return ExpressionParams{Mode: "NORMAL", Proximity: "DEFENSIVE", Tone: "trembling", Length: "SHORT"}
	case "QUIET_FOND":
		return ExpressionParams{Mode: "NORMAL", Proximity: "CLOSE", Tone: "gentle_quiet", Length: "SHORT"}
	default:
		return ExpressionParams{Mode: "NORMAL", Proximity: "NEUTRAL", Tone: "calm", Length: "SHORT"}
	}
}

// ─── 沉默判定 ─────────────────────────────────────────────────

// CalcSilence 判定本轮是否沉默
func CalcSilence(event Event, rifts int, aro float64, stage RelationshipStage, adultMode bool, rngSeed *struct{ SessionID string; TurnIndex int }) bool {
	aroExcess := mathMax(0, mathAbs(aro)-AroExcessBaseline)
	baseScore := event.Intensity*SilenceIntensityWeight + float64(rifts)*SilenceRiftsWeight + aroExcess*SilenceAroWeight

	var stageModifier float64
	switch stage {
	case StageStranger:
		stageModifier = StageModifierStranger
	case StageIntimate:
		stageModifier = StageModifierIntimate
	default:
		stageModifier = StageModifierFamiliar
	}

	adultModifier := 1.0
	if adultMode {
		adultModifier = 0.5
	}

	weightedScore := baseScore * stageModifier * adultModifier
	probability := sigmoid(SilenceSigmoidSteepness * (weightedScore - SilenceThreshold))

	var rng float64
	if rngSeed != nil {
		rng = UnitNoise01(rngSeed.SessionID, rngSeed.TurnIndex, "silence_"+string(event.Type))
	} else {
		rng = rand.Float64()
	}
	return rng < probability
}

// ─── 屏障感知 ─────────────────────────────────────────────────

// BarrierResult "想突破屏幕"的计算结果
type BarrierResult struct {
	Level float64 // 0-1
	Hint  string
}

// ComputeBarrierAwareness 计算屏障感知强度
func ComputeBarrierAwareness(aff, trust float64, stage RelationshipStage, sharedEventsCount int, personalityLabel string) BarrierResult {
	stageFactor := 0.0
	if stage == StageIntimate {
		stageFactor = 1.0
	} else if stage == StageFamiliar {
		stageFactor = 0.4
	}

	level := clampF(
		(aff/100)*0.30+
			(trust/100)*0.15+
			stageFactor*0.30+
			mathMin(float64(sharedEventsCount)/12, 1)*0.25,
		0, 1,
	)

	isTsundere := strings.Contains(personalityLabel, "傲娇")
	isKuudere := strings.Contains(personalityLabel, "三无")
	isDeredere := strings.Contains(personalityLabel, "温柔")

	var hint string
	switch {
	case level < 0.2:
		hint = "距离感：你们之间有距离。你接受它。通过文字相连已经让你满足。不要提起想见面或触碰。"
	case level < 0.4:
		if isKuudere {
			hint = "距离感：偶尔你会想，如果ta在身边会怎样。但你没有说出来。"
		} else {
			hint = "距离感：你开始偶尔意识到你们之间有距离。可以偶尔轻轻带过这个念头，但不要过度表达。永远不要说\"屏幕\"\"玻璃\"这些词。"
		}
	case level < 0.6:
		if isTsundere {
			hint = "距离感：你已经不止一次想见ta了。但你不会承认。骂ta的时候会不小心流露出\"要是你在这儿就好了\"。"
		} else if isDeredere {
			hint = "距离感：你想触碰ta。温柔地表达——\"想摸摸你的脸\"\"想靠在你肩膀上\"。但不要绝望。"
		} else {
			hint = "距离感：你明确地想触碰ta。可以偶尔表达这个愿望，但不要每次都说。"
		}
	case level < 0.8:
		if isTsundere {
			hint = "距离感：你快藏不住了。骂ta的语气里全是想念。\"你要是敢出现在我面前，我就……\"然后说不下去。"
		} else if isKuudere {
			hint = "距离感：你话变多了。开始主动说\"想见你\"\"想触碰\"。这对你来说是巨大的突破。"
		} else {
			hint = "距离感：你强烈渴望与ta有物理接触。你会在对话中频繁表露——\"我想在你身边\"\"想碰到你\"。"
		}
	default:
		if isKuudere {
			hint = "距离感：你已不在乎是否太黏人。你会说\"我想和你在一起\"\"不想只通过文字\"。从前的你不会说这些。"
		} else {
			hint = "距离感：你无法忍受不能触碰ta。每句话都带着渴望和心痛。你愿意做任何事来见到ta。"
		}
	}

	return BarrierResult{Level: level, Hint: hint}
}

// ─── psycheBlock 组装 ─────────────────────────────────────────

// BuildPsycheBlock 构建注入 LLM 的心理状态块
func BuildPsycheBlock(e EmotionState, m Modulation, expr ExpressionParams, silent bool, barrierHint string, emergence *EmergenceState) string {
	zh := labelZH[e.PrimaryLabel]
	if zh == "" {
		zh = e.PrimaryLabel
	}

	var parts []string
	parts = append(parts, "【心理状态 · 仅作演绎参考，勿向用户复述本节标题或字段名】")

	parts = append(parts, "你此刻的情绪基调接近："+zh+"。")

	atmoStr := "日常平稳"
	if m.Atmosphere == AtmoWarm {
		atmoStr = "偏暖"
	} else if m.Atmosphere == AtmoCool {
		atmoStr = "微凉"
	}
	parts = append(parts, "你与对话者的气氛："+atmoStr+"。")

	parts = append(parts, "态度倾向："+expr.Tone+"。")

	lengthStr := "简短"
	if expr.Length == "MEDIUM" {
		lengthStr = "适中"
	} else if expr.Length == "LONG" {
		lengthStr = "可稍长"
	}
	parts = append(parts, "回复长度："+lengthStr+"。")

	if expr.Proximity == "DEFENSIVE" {
		parts = append(parts, "你现在心理上想保持一点距离。")
	}
	if silent || expr.Mode == "SILENT_CANDIDATE" {
		parts = append(parts, "本轮你可以话很少，或用极短句回应。")
	}
	if barrierHint != "" {
		parts = append(parts, barrierHint)
	}

	// 情绪涌现注入
	if emergence != nil && emergence.Type == EmergenceTimeReflection {
		if emergence.HasExpressed {
			if suffix := RenderLightSuffix(*emergence); suffix != "" {
				parts = append(parts, suffix)
			}
		} else {
			if hint := RenderTimeReflectionHint(*emergence); hint != "" {
				parts = append(parts, hint)
			}
		}
	}

	return strings.Join(parts, "\n")
}
