// Package whisper — L2 情绪引擎（100% 对齐 ackem engine/emotion.ts）
package whisper

import (
	"fmt"
	"hash/fnv"
	"math"
)

// ─── BASE_STIMULUS 刺激值表 ───────────────────────────────────

type stimulusVector struct{ Aff, Sec, Aro, Dom float64 }

var baseStimulus = map[EventType]stimulusVector{
	EvtPraise:          {7.0, 4.5, 5.0, -2.0},
	EvtTease:           {4.5, 2.0, 7.0, 2.0},
	EvtCasualChat:      {0.8, 0.5, 1.5, 0},
	EvtCold:            {-5.0, -6.5, -1.5, -2.0},
	EvtHurtful:         {-10.0, -11.0, 7.5, 5.5},
	EvtApology:         {4.5, 6.5, -2.0, -3.5},
	EvtVulnerable:      {10.0, -2.0, -1.0, -5.0},
	EvtQuestion:        {0.8, 0.8, 2.0, 0},
	EvtAdultFlirt:      {3.5, 2.0, 5.0, 1.0},
	EvtAdultDominant:   {2.5, 0.5, 6.0, 5.0},
	EvtAdultSubmissive: {4.5, 3.0, 3.0, -5.0},
	EvtAdultExplicit:   {5.5, 1.0, 7.5, 2.0},
}

// ─── 情绪标签映射 ─────────────────────────────────────────────

// MapEmotionLabel 四维情绪 → 标签
func MapEmotionLabel(e Emotion4D) string {
	// 负向标签
	if e.Aff < -18 && e.Sec < -25 && e.Aro > 40 && e.Dom > 30 {
		return "ANGRY_ATTACK"
	}
	if e.Aff >= 8 && e.Aff <= 55 && e.Sec < -55 && e.Aro > 45 && e.Dom < -45 {
		return "FEARFUL_OBEDIENT"
	}

	// 傲娇：dom > 18 是关键区分
	if e.Aff >= 15 && e.Aff <= 75 && e.Sec >= -10 && e.Sec <= 45 &&
		e.Aro >= 15 && e.Aro <= 75 && e.Dom > 18 {
		return "TSUNDERE"
	}

	// 委屈受伤
	if e.Aff >= 15 && e.Aff <= 55 && e.Sec >= -55 && e.Sec <= -12 &&
		e.Aro >= 15 && e.Aro <= 55 && e.Dom < -18 {
		return "HURT_GRIEVANCE"
	}

	// 甜蜜依恋
	if e.Aff > 25 && e.Sec > 10 && e.Aro > 20 && e.Aro <= 70 &&
		e.Dom >= -25 && e.Dom <= 25 {
		return "SWEET_ATTACHMENT"
	}

	// 安静的喜欢
	if e.Aff > 20 && e.Aro < 25 && e.Dom >= -25 && e.Dom <= 25 {
		return "QUIET_FOND"
	}

	// 害羞心动
	if e.Aff > 15 && e.Aff <= 65 && e.Sec >= -25 && e.Sec <= 35 &&
		e.Aro >= 15 && e.Aro <= 75 && e.Dom < 0 {
		return "SHY_HEARTBEAT"
	}

	// 冷淡疏离
	if e.Aff < -3 && e.Sec >= -35 && e.Sec <= 25 && e.Aro < -3 &&
		e.Dom >= -5 && e.Dom <= 35 {
		return "COLD_DETACHED"
	}

	return "CALM_RATIONAL"
}

// checkLock 检查是否锁定
func checkLock(e Emotion4D) bool {
	return e.Aff > LockAffHigh || e.Aff < LockAffLow || e.Sec < LockSecLow
}

// ─── D/s 情感反转 ─────────────────────────────────────────────

// ApplyDsReversal D/s 臣服情感反转（18+优化）
func ApplyDsReversal(delta stimulusVector, event Event, sensitivity float64, personalityTags []string) stimulusVector {
	if !event.IsAdultContent {
		return delta
	}

	isDs := sensitivity <= 15
	isMesugaki := containsTag(personalityTags, "provoke-submit")
	if !isDs && !isMesugaki {
		return delta
	}

	result := delta

	if (isDs || isMesugaki) && event.AdultSubtype == "dominant" {
		result.Sec = math.Abs(delta.Sec) * 0.6
		result.Dom = -math.Abs(delta.Dom) * 0.8
		result.Aff = delta.Aff * 0.8
		if isMesugaki {
			result.Aro = delta.Aro * 1.3
			result.Aff = delta.Aff * 0.5
			result.Sec = math.Abs(delta.Sec) * 1.0
		}
	}

	if isDs && event.AdultSubtype == "submissive" {
		result.Dom = math.Abs(delta.Dom) * 0.7
		result.Aff = delta.Aff * 1.2
		result.Sec = math.Abs(delta.Sec) * 0.5
	}

	if event.AdultSubtype == "explicit" || event.AdultSubtype == "romantic" {
		result.Aff = delta.Aff * 1.15
		result.Sec = math.Abs(delta.Sec) * 0.7
	}

	return result
}

// ─── 噪声生成 ─────────────────────────────────────────────────

// UnitNoise01 确定性 [0,1) 噪声（FNV-1a hash，对齐 ackem 十进制字符串编码）
func UnitNoise01(sessionID string, turnIndex int, salt string) float64 {
	h := fnv.New32a()
	h.Write([]byte(sessionID))
	h.Write([]byte{0})
	tiStr := fmt.Sprintf("%d", turnIndex)
	h.Write([]byte(tiStr))
	h.Write([]byte{0})
	h.Write([]byte(salt))
	return float64(h.Sum32()) / float64(^uint32(0))
}

// ─── EmotionStep L2 情绪递推 ──────────────────────────────────

// EmotionStepOpts 情绪递推选项
type EmotionStepOpts struct {
	SessionID       string
	TurnIndex       int
	DecayMultiplier float64
	Sensitivity     float64
	PersonalityTags []string
}

// EmotionStep L2 主入口：事件 + 调制 → 新 EmotionState
func EmotionStep(event Event, modulation Modulation, prev EmotionState, opts *EmotionStepOpts) EmotionState {
	if event.Type == EvtExtremeRedline {
		return prev
	}

	S, ok := baseStimulus[event.Type]
	if !ok {
		return prev
	}

	deltaRaw := stimulusVector{
		Aff: S.Aff * modulation.TrustMod * modulation.StageWeight * event.Intensity * event.Sincerity,
		Sec: S.Sec * modulation.TrustMod * event.Intensity * event.Sincerity,
		Aro: S.Aro * modulation.StageWeight * event.Intensity,
		Dom: S.Dom * modulation.StageWeight * event.Intensity,
	}

	capScale := func(absVal float64) float64 {
		return math.Max(0.1, 1-math.Abs(absVal)/EmotionCapDenom)
	}

	deltaCap := stimulusVector{
		Aff: deltaRaw.Aff * capScale(prev.Aff),
		Sec: deltaRaw.Sec * capScale(prev.Sec),
		Aro: deltaRaw.Aro * capScale(prev.Aro),
		Dom: deltaRaw.Dom * capScale(prev.Dom),
	}

	delta := stimulusVector{
		Aff: clamp10(deltaCap.Aff),
		Sec: clamp10(deltaCap.Sec),
		Aro: clamp10(deltaCap.Aro),
		Dom: clamp10(deltaCap.Dom),
	}

	if delta.Aff > 0 {
		delta.Aff *= modulation.RiftMod
	}
	if delta.Sec > 0 {
		delta.Sec *= modulation.RiftMod
	}

	// 锁定区保护
	if prev.Aff > LockAffHigh && delta.Aff < 0 {
		delta.Aff *= LockAffHighReduceNeg
	}
	if prev.Aff < LockAffLow && delta.Aff > 0 {
		delta.Aff *= LockAffLowReducePos
	}
	if prev.Sec < LockSecLow && delta.Sec > 0 {
		delta.Sec *= LockSecLowReducePos
	}

	// 气氛调制
	if modulation.Atmosphere == AtmoWarm {
		delta.Aff *= 1.15
		delta.Sec *= 1.1
	} else if modulation.Atmosphere == AtmoCool {
		delta.Aff *= 0.7
		delta.Sec *= 0.8
	}

	// D/s 情感反转
	if event.IsAdultContent && opts != nil && opts.Sensitivity != 0 {
		reversed := ApplyDsReversal(delta, event, opts.Sensitivity, opts.PersonalityTags)
		delta = reversed
	}

	decayMul := 1.0
	if opts != nil && opts.DecayMultiplier != 0 {
		decayMul = opts.DecayMultiplier
	}
	decay := EmotionDecay * decayMul

	next := Emotion4D{
		Aff: prev.Aff*(1-decay) + delta.Aff,
		Sec: prev.Sec*(1-decay) + delta.Sec,
		Aro: prev.Aro*(1-decay) + delta.Aro,
		Dom: prev.Dom*(1-decay) + delta.Dom,
	}

	// 噪声
	sid := "default"
	tid := 0
	if opts != nil {
		sid = opts.SessionID
		tid = opts.TurnIndex
	}
	addNoise := func(v float64, salt string) float64 {
		if math.Abs(v) > NoiseThresholdAbs {
			u := UnitNoise01(sid, tid, salt)
			return v + (u-0.5)*2*NoiseMax
		}
		return v
	}
	next.Aff = addNoise(next.Aff, "aff")
	next.Sec = addNoise(next.Sec, "sec")
	next.Aro = addNoise(next.Aro, "aro")
	next.Dom = addNoise(next.Dom, "dom")

	next.Aff = clampF(next.Aff, -100, 100)
	next.Sec = clampF(next.Sec, -100, 100)
	next.Aro = clampF(next.Aro, -100, 100)
	next.Dom = clampF(next.Dom, -100, 100)

	label := MapEmotionLabel(next)
	locked := checkLock(next)
	return EmotionState{
		Aff: next.Aff, Sec: next.Sec, Aro: next.Aro, Dom: next.Dom,
		PrimaryLabel: label, IsLocked: locked,
		Mood: stepMood(prev.Mood, next),
	}
}

// ─── 长期心境(Mood) ───────────────────────────────────────────

// stepMood 长期心境 EWMA 递推：Mood = Mood*(1-α) + Emotion*α（α=MoodAlpha）。
// 无历史（prevMood 全 0，即新会话/未播种）时直接以即时情绪首值播种，
// 避免冷启动从 0 每轮仅挪 1% 的慢爬；未被识别的事件在 EmotionStep 入口
// 提前返回，天然保持 Mood 不变（与 Emotion 行为一致）。
func stepMood(prevMood [4]float64, emotion Emotion4D) [4]float64 {
	vec := [4]float64{emotion.Aff, emotion.Sec, emotion.Aro, emotion.Dom}
	if prevMood == ([4]float64{}) {
		return vec
	}
	var out [4]float64
	for i := 0; i < len(out); i++ {
		out[i] = prevMood[i]*(1-MoodAlpha) + vec[i]*MoodAlpha
	}
	return out
}

// ApplyMemoryEcho 记忆回声叠加到情绪
func ApplyMemoryEcho(l2 EmotionState, echo MemoryEcho) EmotionState {
	l2.Aff = clampF(l2.Aff+echo.Aff, -100, 100)
	l2.Sec = clampF(l2.Sec+echo.Sec, -100, 100)
	l2.Aro = clampF(l2.Aro+echo.Aro, -100, 100)
	l2.Dom = clampF(l2.Dom+echo.Dom, -100, 100)
	return l2
}
