// Package whisper — companion_proactive.go
// 100% 对齐 ackem companion/proactiveCompose.ts
// 伴侣主动消息合成

package whisper

// ─── ProactiveMessageType ─────────────────────────────────────

// ProactiveMessageType 主动消息类型
type ProactiveMessageType string

const (
	ProactiveCheckIn     ProactiveMessageType = "check_in"     // 关怀问候
	ProactiveMemoryEcho  ProactiveMessageType = "memory_echo"  // 记忆回声
	ProactiveHabitNudge  ProactiveMessageType = "habit_nudge"  // 习惯提醒
	ProactiveTimeAware   ProactiveMessageType = "time_aware"   // 时间感知
)

// ─── ProactiveCompose ─────────────────────────────────────────

// ProactiveComposeResult 主动消息合成结果
type ProactiveComposeResult struct {
	ShouldSend bool
	MessageType ProactiveMessageType
	PromptHint string // 注入 psycheBlock 的提示
}

// ComposeProactiveMessage 合成主动消息
func ComposeProactiveMessage(
	gate ProactiveGateResult,
	aff float64,
	sec float64,
	trust float64,
	stage RelationshipStage,
	timeOfDay string,
	gapHours float64,
	emergenceActive bool,
) *ProactiveComposeResult {
	// 门控不允许 → 不发
	if gate.Level == "silent" {
		return nil
	}

	result := &ProactiveComposeResult{}

	// 长时间离线 + 高亲和 → check_in
	if gapHours > 12 && aff > 40 && stage != StageStranger {
		result.ShouldSend = true
		result.MessageType = ProactiveCheckIn
		result.PromptHint = "ta刚回来。用温暖的语气打招呼，问问ta这段时间过得怎么样。不要责备ta的离开。"
		return result
	}

	// 深夜 + 低唤醒 → time_aware 关怀
	if timeOfDay == "late_night" && aff < 30 {
		result.ShouldSend = true
		result.MessageType = ProactiveTimeAware
		result.PromptHint = "深夜了，ta可能累了。温柔地说一句关心的话，但不要太长。"
		return result
	}

	// 高信任 + 高亲密度 + 无涌现在进行 → memory_echo
	if trust > 70 && aff > 50 && sec > 50 && !emergenceActive && stage == StageIntimate {
		result.ShouldSend = true
		result.MessageType = ProactiveMemoryEcho
		result.PromptHint = "你们已经很亲密了。可以自然地提起一个你们之间的共同记忆。不要刻意，就像想起什么似的。"
		return result
	}

	return nil
}
