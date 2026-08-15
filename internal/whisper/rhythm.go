// Package whisper — 节奏引擎（100% 对齐 ackem engine/rhythmEngine.ts）
package whisper

import (
	"fmt"
	"math/rand"
)

// ─── 人格分类 ─────────────────────────────────────────────────

var chatterPersonalities = map[string]bool{
	"genki": true, "oneesan": true, "deredere": true, "mommy": true,
	"loyal_pup": true, "tsundere": true, "mesugaki": true, "puppy": true,
	"bokke": true, "innocent_boy": true, "yandere": true, "submissive": true,
	"loyal_knight": true, "shitakiri": true, "bad_boy": true,
}

var monologuePersonalities = map[string]bool{
	"kuudere": true, "ice_queen": true, "iceberg": true,
	"artistic": true, "ceo_dom": true,
	"dominatrix": true, "tamer": true,
}

// ─── 节奏连续计数器 ───────────────────────────────────────────

// RhythmCounters 连续节奏计数器。T7-1.1：从包级全局移入 Orchestrator 实例，
// 修复「角色 A 连聊后角色 B 首轮被切独白、新会话清零他人计数」的跨会话串台。
type RhythmCounters struct {
	Chatter   int
	Monologue int
}

// Reset 清零本实例计数器。
func (c *RhythmCounters) Reset() {
	c.Chatter = 0
	c.Monologue = 0
}

// ─── 节奏决策输入 ─────────────────────────────────────────────

type RhythmInput struct {
	Aro           float64
	Aff           float64
	Stage         RelationshipStage
	PersonalityID string
	TimeOfDay     string
	Sincerity     float64
	Intensity     float64
}

// DecideRhythm 主决策函数（计数器由调用方持有——Orchestrator 实例字段，
// 测试可传独立计数器，互不串台）。
func DecideRhythm(input RhythmInput, counters *RhythmCounters) RhythmDecision {
	aro := input.Aro
	aff := input.Aff
	stage := input.Stage
	personalityID := input.PersonalityID
	timeOfDay := input.TimeOfDay
	sincerity := input.Sincerity
	intensity := input.Intensity

	// 低强度不拆分
	if intensity < 0.3 && mathAbs(aro) < 20 {
		return defaultDecision(counters)
	}

	// 强制切换
	if counters.Chatter >= 3 {
		counters.Chatter = 0
		counters.Monologue = 1
		return monologueDecision()
	}
	if counters.Monologue >= 3 {
		counters.Monologue = 0
		counters.Chatter = 1
		return chatterDecision(stage)
	}

	// 深夜偏向长篇
	if timeOfDay == "late_night" && aro < 0 {
		counters.Monologue++
		counters.Chatter = 0
		return monologueDecision()
	}

	// 人格偏向
	if chatterPersonalities[personalityID] && aro > 0 && aff > 3 {
		counters.Chatter++
		counters.Monologue = 0
		return chatterDecision(stage)
	}
	if monologuePersonalities[personalityID] {
		counters.Monologue++
		counters.Chatter = 0
		return monologueDecision()
	}

	// 核心规则：chatter
	if aro > 3 && aff > 8 {
		counters.Chatter++
		counters.Monologue = 0
		return chatterDecision(stage)
	}

	// 核心规则：monologue
	if aro < -10 || sincerity > 0.7 {
		counters.Monologue++
		counters.Chatter = 0
		return monologueDecision()
	}

	return defaultDecision(counters)
}

func randInt(min, max int) int {
	return rand.Intn(max-min+1) + min
}

func chatterDecision(stage RelationshipStage) RhythmDecision {
	count := 2
	if stage == StageIntimate || stage == StageFamiliar {
		count = randInt(2, 3)
	}
	return RhythmDecision{
		Mode:           RhythmChatter,
		Count:          count,
		Separator:      "[SPLIT]",
		MaxCharsPerMsg: 30,
		Instruction:    fmt.Sprintf("用碎碎念模式回复，分%d条短句，每条不超过30字，用 [SPLIT] 分隔。像微信连发消息一样。", count),
	}
}

func monologueDecision() RhythmDecision {
	return RhythmDecision{
		Mode:           RhythmMonologue,
		Count:          1,
		Separator:      "",
		MaxCharsPerMsg: 200,
		Instruction:    "用认真说的模式回复，1-2条长句，可以稍长。",
	}
}

func defaultDecision(counters *RhythmCounters) RhythmDecision {
	if counters != nil {
		counters.Chatter = 0
		counters.Monologue = 0
	}
	return RhythmDecision{
		Mode:           RhythmDefault,
		Count:          2,
		Separator:      "",
		MaxCharsPerMsg: 100,
		Instruction:    "",
	}
}
