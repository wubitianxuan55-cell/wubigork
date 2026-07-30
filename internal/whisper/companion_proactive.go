// Package whisper — companion_proactive.go
// 对齐 ackem companion/proactiveCompose.ts
// gaea主动消息合成 — 增强版：5 种消息类型 + 人格感知 + 信任门控

package whisper

// ─── ProactiveMessageType ─────────────────────────────────────

// ProactiveMessageType 主动消息类型（对齐 ackem 5 种）
type ProactiveMessageType string

const (
	ProactiveCheckIn     ProactiveMessageType = "check_in"      // 关怀问候
	ProactiveMemoryEcho  ProactiveMessageType = "memory_echo"   // 记忆回声
	ProactiveHabitNudge  ProactiveMessageType = "habit_nudge"   // 习惯提醒
	ProactiveTimeAware   ProactiveMessageType = "time_aware"    // 时间感知
	ProactiveMissYou     ProactiveMessageType = "miss_you"      // 想念表达
	ProactivePlayful     ProactiveMessageType = "playful_nudge" // 俏皮戳一戳
)

// ─── ProactiveCompose ─────────────────────────────────────────

// ProactiveComposeResult 主动消息合成结果
type ProactiveComposeResult struct {
	ShouldSend  bool
	MessageType ProactiveMessageType
	PromptHint  string // 注入 psycheBlock 的提示
}

// ComposeProactiveMessage 合成主动消息（增强版）
func ComposeProactiveMessage(
	gate ProactiveGateResult,
	aff float64,
	sec float64,
	trust float64,
	stage RelationshipStage,
	timeOfDay string,
	gapHours float64,
	emergenceActive bool,
	personalityID string, // P1新增：人格ID用于定制prompt
) *ProactiveComposeResult {
	// 门控不允许 → 不发
	if gate.Level == "silent" {
		return nil
	}
	// 陌生人+低信任 → 不发
	if stage == StageStranger && trust < 35 {
		return nil
	}

	// 长时间离线 + 高亲和 → check_in
	if gapHours > 12 && aff > 40 && stage != StageStranger {
		return &ProactiveComposeResult{
			ShouldSend:  true,
			MessageType: ProactiveCheckIn,
			PromptHint:  buildCheckInHint(personalityID, gapHours),
		}
	}

	// 非常长时间离线(>3天) + 高信任 → miss_you
	if gapHours > 72 && trust > 55 && aff > 40 {
		return &ProactiveComposeResult{
			ShouldSend:  true,
			MessageType: ProactiveMissYou,
			PromptHint:  buildMissYouHint(personalityID, gapHours),
		}
	}

	// 深夜 + 低唤醒 → time_aware 关怀
	if timeOfDay == "late_night" && aff < 30 {
		return &ProactiveComposeResult{
			ShouldSend:  true,
			MessageType: ProactiveTimeAware,
			PromptHint:  "深夜了，ta可能累了。温柔地说一句关心的话，但不要太长。不要催促ta去睡觉。",
		}
	}

	// 高唤醒+高亲和+非深夜 → playful_nudge
	if aff > 55 && timeOfDay != "late_night" && stage != StageStranger {
		return &ProactiveComposeResult{
			ShouldSend:  true,
			MessageType: ProactivePlayful,
			PromptHint:  buildPlayfulHint(personalityID),
		}
	}

	// 高信任 + 高亲密 + 无涌现在进行 → memory_echo
	if trust > 70 && aff > 50 && sec > 50 && !emergenceActive && stage == StageIntimate {
		return &ProactiveComposeResult{
			ShouldSend:  true,
			MessageType: ProactiveMemoryEcho,
			PromptHint:  "你们已经很亲密了。可以自然地提起一个你们之间的共同记忆。不要刻意，就像想起什么似的。",
		}
	}

	return nil
}

// buildCheckInHint 生成关怀问候 prompt
func buildCheckInHint(personalityID string, gapHours float64) string {
	base := "ta刚回来。用温暖的语气打招呼，问问ta这段时间过得怎么样。不要责备ta的离开。"
	if gapHours > 72 {
		base += " 已经好几天没见了，可以稍微表达想念。"
	}
	// 人格修饰
	switch personalityID {
	case "tsundere":
		return base + " 但保持傲娇——明明是关心却装作不在意。"
	case "yandere":
		return base + " 带着一点占有欲的关心。"
	case "kuudere":
		return base + " 表面冷淡但话里有温度。"
	}
	return base
}

// buildMissYouHint 生成想念表达 prompt
func buildMissYouHint(personalityID string, gapHours float64) string {
	base := "你们好几天没见了。"
	if gapHours > 168 {
		base += " 已经超过一周了，想念可以更强烈一些。"
	}
	switch personalityID {
	case "tsundere":
		return base + " 用傲娇的方式表达想念——'才不是因为想你呢'的感觉。"
	case "deredere":
		return base + " 可以直率地表达想念。"
	case "kuudere":
		return base + " 简短地表达就足够，不用说太多。"
	}
	return base + " 自然地表达想念，不用太夸张。"
}

// buildPlayfulHint 生成俏皮戳一戳 prompt
func buildPlayfulHint(personalityID string) string {
	switch personalityID {
	case "genki":
		return "元气满满地戳戳ta！用活泼的语气。"
	case "tsundere":
		return "找个借口戳戳ta——才不是因为想ta呢。"
	case "oneesan":
		return "带着大姐姐的温柔逗逗ta。"
	case "bokke":
		return "天然呆地发起一个无厘头话题。"
	}
	return "俏皮地戳戳ta，可以分享一个有趣的小事或发一个可爱的表情。"
}
