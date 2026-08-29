// Package voice — 情感→TTS 语音参数映射
//
// 100% 对齐 Ackem emotionVoiceMap.ts 的设计：
// 将 whisper L2 情绪标签映射为 TTS 引擎可用的语音指令/参数。
//
// 支持三种引擎的映射：
//   - Herdsman qwen3-tts-voicedesign: voice_description（中文自然语言指令）
//   - Edge TTS: rate/pitch 百分比偏移
//   - WinTTS SAPI: 无情感控制（仅做标记）
package voice

import "github.com/gaea/gaea/internal/tts"

// EmotionVoiceParams 情感→语音参数
type EmotionVoiceParams struct {
	// 中文自然语言指令（用于 qwen3-tts-voicedesign / CosyVoice 风格）
	VoiceDescription string `json:"voiceDescription"`
	// Edge TTS 语速偏移（百分比，如 "-10%"）
	EdgeRate string `json:"edgeRate"`
	// Edge TTS 音高偏移（百分比，如 "+5Hz" → 实际用百分比）
	EdgePitch string `json:"edgePitch"`
	// WinTTS 备注（无实际参数控制）
	WinTTSNote string `json:"winTTSNote"`
}

// EmotionVoiceMap 情绪标签 → 语音参数映射表
//
// 对齐 Ackem emotionVoiceMap.ts 的 9 种 L2 情感 + 人格修饰逻辑
var EmotionVoiceMap = map[string]EmotionVoiceParams{
	// ── 积极/温暖类 ──
	"SWEET_ATTACHMENT": {
		VoiceDescription: "用温柔甜蜜的语气说",
		EdgeRate:         "-10%",
		EdgePitch:        "+5Hz",
		WinTTSNote:       "甜美温柔",
	},
	"SHY_HEARTBEAT": {
		VoiceDescription: "用害羞带点紧张的语气说",
		EdgeRate:         "-5%",
		EdgePitch:        "+3Hz",
		WinTTSNote:       "害羞紧张",
	},
	"QUIET_FOND": {
		VoiceDescription: "用安静温柔的语气说",
		EdgeRate:         "-15%",
		EdgePitch:        "-2Hz",
		WinTTSNote:       "安静温柔",
	},
	"TSUNDERE": {
		VoiceDescription: "用傲娇又害羞的语气说",
		EdgeRate:         "+5%",
		EdgePitch:        "+5Hz",
		WinTTSNote:       "傲娇",
	},

	// ── 中性/理性类 ──
	"CALM_RATIONAL": {
		VoiceDescription: "用冷静平淡的语气说",
		EdgeRate:         "0%",
		EdgePitch:        "0Hz",
		WinTTSNote:       "冷静理性",
	},

	// ── 消极/疏离类 ──
	"COLD_DETACHED": {
		VoiceDescription: "用冷淡疏离的语气说",
		EdgeRate:         "0%",
		EdgePitch:        "-5Hz",
		WinTTSNote:       "冷淡疏离",
	},
	"HURT_GRIEVANCE": {
		VoiceDescription: "用委屈受伤的语气说",
		EdgeRate:         "-10%",
		EdgePitch:        "-5Hz",
		WinTTSNote:       "委屈受伤",
	},
	"FEARFUL_OBEDIENT": {
		VoiceDescription: "用不安顺从的语气说",
		EdgeRate:         "-5%",
		EdgePitch:        "-3Hz",
		WinTTSNote:       "不安顺从",
	},
	"ANGRY_ATTACK": {
		VoiceDescription: "用愤怒尖锐的语气说",
		EdgeRate:         "+10%",
		EdgePitch:        "+10Hz",
		WinTTSNote:       "愤怒激动",
	},
}

// GetEmotionVoiceLegacyParams 获取情绪对应的语音参数（旧版，返回描述型参数）
// 对齐 Ackem getEmotionInstruction() 逻辑；v4.3d 起结构化参数见
// GetEmotionVoiceParams（返回 EmotionTTSParams），本函数仅保留给
// VoiceDescription/Edge 字符串参数路径使用。
func GetEmotionVoiceLegacyParams(emotionLabel string) EmotionVoiceParams {
	if params, ok := EmotionVoiceMap[emotionLabel]; ok {
		return params
	}
	// 默认：平静理性
	return EmotionVoiceMap["CALM_RATIONAL"]
}

// GetVoiceDescription 获取中文自然语言指令（用于 qwen3-tts-voicedesign）
func GetVoiceDescription(emotionLabel string) string {
	return GetEmotionVoiceLegacyParams(emotionLabel).VoiceDescription
}

// GetEdgeTTSParams 获取 Edge TTS 的 rate/pitch 参数
func GetEdgeTTSParams(emotionLabel string) (rate string, pitch string) {
	p := GetEmotionVoiceLegacyParams(emotionLabel)
	return p.EdgeRate, p.EdgePitch
}

// ── 结构化 TTS 参数（v4.3d 新增） ──

// EmotionTTSParamsMap 情绪标签 → 结构化 TTS 参数默认值（v4.3d）。
//
// 映射依据：结合设计文档 §5.3「每情绪标签 → TTSParams 默认值」与本任务契约的
// 取值约定——仅强情绪标签给非中性参数，其余标签一律中性默认（Speed/Pitch=0，
// 即引擎默认语速/音高，Style/Emotion 为空），未知/空标签返回零值。
// 具体数值来源：参考现有 EdgeRate/EdgePitch 数据的情感方向——
//   - ANGRY_ATTACK（愤怒，EdgeRate "+10%"/EdgePitch "+10Hz"）：加速 + 提音，
//     Speed 1.1（≈+10% 语速）、Pitch +2 半音、Emotion "angry"（透传引擎情绪）；
//   - CALM_RATIONAL（冷静，EdgeRate "0%"/EdgePitch "0Hz"）：放缓 + 微降调，
//     Speed 0.9、Pitch -1 半音、Emotion "calm"。
// 其余 7 个标签（SWEET_ATTACHMENT/SHY_HEARTBEAT/QUIET_FOND/TSUNDERE/
// COLD_DETACHED/HURT_GRIEVANCE/FEARFUL_OBEDIENT）暂不指定结构化参数，
// 走引擎中性默认，避免过度调参；需要时按同一方向增量补表。
var EmotionTTSParamsMap = map[string]tts.TTSParams{
	"ANGRY_ATTACK":  {Speed: 1.1, Pitch: 2, Emotion: "angry"},
	"CALM_RATIONAL": {Speed: 0.9, Pitch: -1, Emotion: "calm"},
}

// GetEmotionVoiceParams 获取情绪标签对应的结构化 TTS 参数（v4.3d）。
// emotion 为空或未知（不在 EmotionTTSParamsMap）→ 零值 TTSParams
// （Speed/Pitch=0 表示引擎默认，Style/Emotion 为空串，语义即「中性默认」）。
func GetEmotionVoiceParams(emotion string) tts.TTSParams {
	return EmotionTTSParamsMap[emotion]
}

// ── 人格修饰 ──

// PersonalityVoiceModifier 人格→语音修饰词
// 对齐 Ackem emotionVoiceMap.ts 中的人格修饰逻辑
// 例如：tsundere 人格会插入 "傲娇又"
var PersonalityVoiceModifier = map[string]string{
	"gaea":      "温厚又",
	"tsundere":  "傲娇又",
	"yandere":   "病态又",
	"kuudere":   "更",
	"deredere":  "超级",
	"himedere":  "高傲又",
	"dandere":   "更",
	"kamidere":  "傲慢又",
	"mayadere":  "更",
	"darudere":  "懒洋洋又",
	"hinedere":  "更",
	"sadodere":  "更",
	"bakadere":  "更天真又",
	"goudere":   "更强势又",
	"shundere":  "更悲伤又",
	"biridere":  "更紧张又",
	"nyandere":  "更猫猫又",
	"kanedere":  "更拜金又",
	"oji-dere":  "更宠溺又",
	"ero-dere":  "更色气又",
	"oni-dere":  "更强势又",
	"zettai-dere": "绝对",
	"hajidere":  "更害羞又",
	"megadere":  "更狂热又",
	"utsudere":  "更忧郁又",
	"undere":    "更",
	"bokodere":  "更暴力又",
	"otokodere": "更男子气又",
	"tennodere": "更天使又",
	"shindere":  "更温柔又",
}

// ModifyWithPersonality 用人格修饰情绪指令
// 对齐 Ackem 的 tsundere → "用傲娇又甜蜜的语气说"
//
// 以 tsundere 为例：原始 "用温柔甜蜜的语气说" → "用傲娇又温柔甜蜜的语气说"
func ModifyWithPersonality(description string, personalityID string) string {
	modifier, ok := PersonalityVoiceModifier[personalityID]
	if !ok || modifier == "" {
		return description
	}
	// 在 "用" 和后续描述之间插入人格修饰词
	// "用温柔甜蜜的语气说" → "用傲娇又温柔甜蜜的语气说"
	if len(description) > 3 && description[:3] == "用" {
		return "用" + modifier + description[3:]
	}
	return description
}

// GetVoiceDescriptionWithPersonality 获取带人格修饰的语音指令
func GetVoiceDescriptionWithPersonality(emotionLabel string, personalityID string) string {
	desc := GetVoiceDescription(emotionLabel)
	return ModifyWithPersonality(desc, personalityID)
}
