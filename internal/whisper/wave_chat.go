// Package whisper — wave_chat.go
// 100% 对齐 ackem shared/wavePlan.ts + chat/waveChat.ts
// WavePlan 构建 + 多波决策 + 编排逻辑

package whisper

import (
	"math/rand"
)

// WaveCount 波数（1-4）
type WaveCount int

// WaveEmotionHint 波数决策所需的情绪提示
type WaveEmotionHint struct {
	Aro       float64 `json:"aro"`
	Aff       float64 `json:"aff"`
	Intensity float64 `json:"intensity"`
	Sincerity float64 `json:"sincerity"`
}

// WavePlan 波次计划
type WavePlan struct {
	WaveCount  WaveCount  `json:"waveCount"`
	Waves      []WaveSpec `json:"waves"`
	RhythmMode RhythmMode `json:"rhythmMode"`
}

// SkipWavesInput 是否跳过 Wave 的判断输入
type SkipWavesInput struct {
	AsyncMultiMsgEnabled *bool  `json:"asyncMultiMessageEnabled,omitempty"`
	KnowledgeTopic       string `json:"knowledgeTopic,omitempty"`
	PlanDocumentTopic    string `json:"planDocumentTopic,omitempty"`
	ForcedWebSearchQuery string `json:"forcedWebSearchQuery,omitempty"`
	DispatchDecision     string `json:"dispatchDecision,omitempty"`
	EnterPlanMode        bool   `json:"enterPlanMode,omitempty"`
	SkipLLM              bool   `json:"skipLlm,omitempty"`
	RequiresToolTurn     bool   `json:"requiresToolTurn,omitempty"`
}

// ─── 波数决策 ──────────────────────────────────────────────────

// resolveAsyncWaveCount 根据节奏和情绪决定异步多波轮数
// 100% 对齐 ackem shared/wavePlan.ts resolveAsyncWaveCount
func resolveAsyncWaveCount(rhythm RhythmDecision, emotion *WaveEmotionHint, rng func() float64) WaveCount {
	if rhythm.Mode == RhythmMonologue {
		return 1
	}

	if rng == nil {
		rng = rand.Float64
	}

	aro := 0.0
	aff := 0.0
	intensity := 0.0
	sincerity := 0.0
	if emotion != nil {
		aro = emotion.Aro
		aff = emotion.Aff
		intensity = emotion.Intensity
		sincerity = emotion.Sincerity
	}

	emotional := aro > 10 || aff > 12 || intensity > 0.4 || sincerity > 0.6
	highEmotional := aro > 18 || aff > 22 || intensity > 0.55 || (sincerity > 0.7 && aff > 8)

	if highEmotional && rng() < 0.32 {
		return 4
	}
	if emotional || rhythm.Mode == RhythmChatter {
		if rng() < 0.55 {
			return 3
		}
		return 2
	}
	if rng() < 0.28 {
		return 3
	}
	return 2
}

// ─── 波次系统增量 ──────────────────────────────────────────────

// waveSystemDelta 按波次返回 system 增量提示
// 100% 对齐 ackem shared/wavePlan.ts waveSystemDelta
func waveSystemDelta(waveIndex int, waveCount WaveCount, locale string) string {
	singleBubbleRule := "【单条】本条=微信里的一条消息：只写1句，不换行，不要第二句，不要括号内心独白。"

	switch {
	case waveIndex == 0:
		return singleBubbleRule + "\n【快反应】先接话：共鸣/短问/短表态，≤25字。禁止排期、店名、外卖、记笔记等具体安排。"
	case waveIndex == 1 && waveCount >= 2:
		return singleBubbleRule + "\n【续聊】假定你已短答过。只加一个新信息：一个具体建议（时间/店/做法三选一），≤35字。禁止在线确认（在/在呢）。"
	case waveIndex == 2 && waveCount >= 3:
		return singleBubbleRule + "\n【细节】若有共同记忆可带一句；没有就一句关心或补充，≤35字。禁止重复前面任何意思。"
	case waveIndex >= 3 && waveCount >= 4:
		return singleBubbleRule + "\n【收尾】一句关心或轻松收束，≤30字。不要重复前文。"
	case waveIndex == int(waveCount)-1 && waveCount >= 2:
		return singleBubbleRule + "\n【收尾】最后一句稍暖，与前文衔接但不重复，≤30字。"
	default:
		return ""
	}
}

// ─── BuildWavePlan ─────────────────────────────────────────────

// BuildWavePlan 根据节奏决策构建 WavePlan
// 100% 对齐 ackem shared/wavePlan.ts buildWavePlan
func BuildWavePlan(rhythm RhythmDecision, locale string, emotion *WaveEmotionHint) WavePlan {
	waveCount := resolveAsyncWaveCount(rhythm, emotion, nil)
	if rhythm.Mode == RhythmMonologue {
		waveCount = 1
	}

	maxCharsPerMsg := rhythm.MaxCharsPerMsg
	if maxCharsPerMsg <= 0 {
		maxCharsPerMsg = 40
	}

	waves := make([]WaveSpec, 0, waveCount)
	for i := 0; i < int(waveCount); i++ {
		maxChars := maxCharsPerMsg
		if i == 0 {
			if maxChars > 40 {
				maxChars = 40
			}
		} else if rhythm.Mode == RhythmMonologue {
			maxChars = 200
		}

		waves = append(waves, WaveSpec{
			WaveIndex:   i,
			MaxChars:    maxChars,
			SystemDelta: waveSystemDelta(i, waveCount, locale),
		})
	}

	return WavePlan{
		WaveCount:  waveCount,
		Waves:      waves,
		RhythmMode: rhythm.Mode,
	}
}

// ─── Wave Chat 开关逻辑 ────────────────────────────────────────

// ShouldUseWaveChat 是否使用异步多波聊天
// 100% 对齐 ackem shared/wavePlan.ts shouldUseWaveChat
func ShouldUseWaveChat(input SkipWavesInput) bool {
	if input.AsyncMultiMsgEnabled != nil && !*input.AsyncMultiMsgEnabled {
		return false
	}
	return !skipWaves(input)
}

// skipWaves 强制单轮路径
// 100% 对齐 ackem shared/wavePlan.ts skipWaves
func skipWaves(input SkipWavesInput) bool {
	if input.SkipLLM {
		return true
	}
	if input.EnterPlanMode {
		return true
	}
	if input.KnowledgeTopic != "" {
		return true
	}
	if input.PlanDocumentTopic != "" {
		return true
	}
	if input.ForcedWebSearchQuery != "" {
		return true
	}
	if input.RequiresToolTurn {
		return true
	}

	d := input.DispatchDecision
	switch d {
	case "evolve", "open_surface", "invoke_surface", "ask_invoke", "ask_plan", "plan":
		return true
	}
	return false
}

// RequiresToolTurn 任务型单轮（表格/对比/联网搜等）不走 wave
// 100% 对齐 ackem shared/wavePlan.ts requiresToolTurn
func RequiresToolTurn(needsSearch bool, goal string, delivery string) bool {
	if needsSearch {
		return true
	}
	if goal != "casual" {
		return true
	}
	if delivery != "prose" {
		return true
	}
	return false
}
