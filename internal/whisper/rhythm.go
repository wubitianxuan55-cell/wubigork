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

// ─── 模块级连续计数器 ─────────────────────────────────────────

var (
	consecutiveChatter   int
	consecutiveMonologue int
)

func ResetRhythmState() {
	consecutiveChatter = 0
	consecutiveMonologue = 0
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

// DecideRhythm 主决策函数
func DecideRhythm(input RhythmInput) RhythmDecision {
	aro := input.Aro
	aff := input.Aff
	stage := input.Stage
	personalityID := input.PersonalityID
	timeOfDay := input.TimeOfDay
	sincerity := input.Sincerity
	intensity := input.Intensity

	// 低强度不拆分
	if intensity < 0.3 && mathAbs(aro) < 20 {
		return defaultDecision()
	}

	// 强制切换
	if consecutiveChatter >= 3 {
		consecutiveChatter = 0
		consecutiveMonologue = 1
		return monologueDecision()
	}
	if consecutiveMonologue >= 3 {
		consecutiveMonologue = 0
		consecutiveChatter = 1
		return chatterDecision(stage)
	}

	// 深夜偏向长篇
	if timeOfDay == "late_night" && aro < 0 {
		consecutiveMonologue++
		consecutiveChatter = 0
		return monologueDecision()
	}

	// 人格偏向
	if chatterPersonalities[personalityID] && aro > 0 && aff > 3 {
		consecutiveChatter++
		consecutiveMonologue = 0
		return chatterDecision(stage)
	}
	if monologuePersonalities[personalityID] {
		consecutiveMonologue++
		consecutiveChatter = 0
		return monologueDecision()
	}

	// 核心规则：chatter
	if aro > 3 && aff > 8 {
		consecutiveChatter++
		consecutiveMonologue = 0
		return chatterDecision(stage)
	}

	// 核心规则：monologue
	if aro < -10 || sincerity > 0.7 {
		consecutiveMonologue++
		consecutiveChatter = 0
		return monologueDecision()
	}

	return defaultDecision()
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
		Mode:          RhythmChatter,
		Count:         count,
		Separator:     "[SPLIT]",
		MaxCharsPerMsg: 30,
		Instruction:   fmt.Sprintf("用碎碎念模式回复，分%d条短句，每条不超过30字，用 [SPLIT] 分隔。像微信连发消息一样。", count),
	}
}

func monologueDecision() RhythmDecision {
	return RhythmDecision{
		Mode:          RhythmMonologue,
		Count:         1,
		Separator:     "",
		MaxCharsPerMsg: 200,
		Instruction:   "用认真说的模式回复，1-2条长句，可以稍长。",
	}
}

func defaultDecision() RhythmDecision {
	consecutiveChatter = 0
	consecutiveMonologue = 0
	return RhythmDecision{
		Mode:          RhythmDefault,
		Count:         2,
		Separator:     "",
		MaxCharsPerMsg: 100,
		Instruction:   "",
	}
}
