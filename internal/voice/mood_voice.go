package voice

import "strings"

// ─── 长期心境 → 连续韵律（v4.6 Mood→TTS 闭环）────────────────────
//
// 审计口径（docs/audit-2026-08-30-v4-execution-review.md §C）：「长期心境
// Mood 只存不用——无任何 TTS 路径读 Mood，『听得出她今天低落』是原料非成品；
// 映射为离散标签→静态预设，非连续韵律」。本文件补上读路径：把 whisper 的
// 4D 慢速 EWMA（Aff 喜爱 / Sec 安全感 / Aro 唤醒 / Dom 掌控，-100..100）
// 映射为中文连续韵律指令，供语音对话自动 TTS 使用。
//
// 语义约定（基线色调，非逐轮覆盖）：
//   - 即时情绪标签（9 标签静态预设）代表「本轮反应」，强情绪轮次仍由标签主导；
//   - Mood 代表「今天/最近的整体基调」——中性轮次（CALM_RATIONAL/空标签）由
//     长期心境主导韵律：低落几天后连冷静回答都带着低气压（"低沉平缓"），
//     兴奋期连普通闲聊都轻快。
//   - 全维低于阈值（心境未播种/长期中性）返回 ""，调用方回退标签静态预设。

// moodVoiceThresholds 各维生效阈值：|值| 低于阈值视为中性（不参与韵律调制）。
// 与 MapEmotionLabel 的标签区间同量纲（-100..100），阈值取标签区间的低端
// （如 COLD_DETACHED 需 Aff < -3、QUIET_FOND 需 Aro < 25），保证「标签已识别
// 的情绪」与「韵律调制」一致，不出现标签中性但韵律剧烈的空转。
type moodVoiceThresholds struct {
	AffLow, AffHigh float64 // 喜爱：<低=低沉，>高=温暖
	SecLow          float64 // 安全感：<低=带着一丝不安
	AroLow, AroHigh float64 // 唤醒：<低=平缓，>高=轻快
	DomLow, DomHigh float64 // 掌控：<低=柔和顺从，>高=略带强势
}

var moodVoiceThresholdsDefault = moodVoiceThresholds{
	AffLow: -15, AffHigh: 20,
	SecLow: -20,
	AroLow: -12, AroHigh: 18,
	DomLow: -18, DomHigh: 18,
}

// MoodToVoiceDescription 把长期心境 4D EWMA 映射为中文连续韵律指令。
// 全维中性（未播种/长期平稳）返回 ""——调用方继续用情绪标签静态预设，
// 行为与 v4.3d 及以前完全一致（增量不破坏）。
func MoodToVoiceDescription(mood [4]float64) string {
	return moodToVoiceDescription(mood, moodVoiceThresholdsDefault)
}

func moodToVoiceDescription(mood [4]float64, t moodVoiceThresholds) string {
	aff, sec, aro, dom := mood[0], mood[1], mood[2], mood[3]
	mods := make([]string, 0, 4)
	switch {
	case aff < t.AffLow:
		mods = append(mods, "低沉")
	case aff > t.AffHigh:
		mods = append(mods, "温暖")
	}
	if sec < t.SecLow {
		mods = append(mods, "带着一丝不安")
	}
	switch {
	case aro < t.AroLow:
		mods = append(mods, "平缓")
	case aro > t.AroHigh:
		mods = append(mods, "轻快")
	}
	switch {
	case dom < t.DomLow:
		mods = append(mods, "柔和顺从")
	case dom > t.DomHigh:
		mods = append(mods, "略带强势")
	}
	if len(mods) == 0 {
		return ""
	}
	return "用" + strings.Join(mods, "") + "的语气说"
}
