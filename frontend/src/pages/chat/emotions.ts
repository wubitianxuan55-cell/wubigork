// 朗读情绪目录（v4.3d 前端收尾）
//
// 前端硬编码的情绪清单，与后端情绪→语音映射保持一致：
//   - 标签集合 = internal/voice/emotion_voice_map.go 的 EmotionVoiceMap（9 种
//     whisper L2 情绪，结构化参数表 EmotionTTSParamsMap 是其子集）；
//   - 中文名对齐 internal/whisper/psyche.go 的 labelZH；
//   - 情绪色复用本目录 constants.ts 的 EMO_COLORS（分类色板）。
//
// 朗读时把标签作为 TTSParams.Emotion 透传给 TTSSpeakBase64WithParams，
// 后端按标签解析语速/音高/风格（未列入结构化表的标签走引擎中性默认）。
import { EMO_COLORS } from './constants'

/** 情绪选项 value 的空值语义：跟随会话最近一轮情绪（自动）。 */
export const SPEAK_EMOTION_AUTO = ''

export interface SpeakEmotionOption {
  /** whisper L2 情绪标签（与后端 EmotionVoiceMap 键一致） */
  value: string
  /** 中文显示名（对齐 whisper labelZH） */
  label: string
  /** 情绪分类色（EMO_COLORS） */
  color: string
}

export const SPEAK_EMOTIONS: SpeakEmotionOption[] = [
  { value: 'SWEET_ATTACHMENT', label: '甜蜜依恋', color: EMO_COLORS.SWEET_ATTACHMENT },
  { value: 'SHY_HEARTBEAT', label: '害羞心动', color: EMO_COLORS.SHY_HEARTBEAT },
  { value: 'QUIET_FOND', label: '安静的喜欢', color: EMO_COLORS.QUIET_FOND },
  { value: 'TSUNDERE', label: '傲娇', color: EMO_COLORS.TSUNDERE },
  { value: 'CALM_RATIONAL', label: '平静理性', color: EMO_COLORS.CALM_RATIONAL },
  { value: 'COLD_DETACHED', label: '冷淡疏离', color: EMO_COLORS.COLD_DETACHED },
  { value: 'HURT_GRIEVANCE', label: '委屈受伤', color: EMO_COLORS.HURT_GRIEVANCE },
  { value: 'FEARFUL_OBEDIENT', label: '不安顺从', color: EMO_COLORS.FEARFUL_OBEDIENT },
  { value: 'ANGRY_ATTACK', label: '愤怒反击', color: EMO_COLORS.ANGRY_ATTACK },
]

/** 情绪标签 → 中文名；未知标签返回 undefined（上层按原文展示）。 */
export function emotionLabel(value: string): string | undefined {
  return SPEAK_EMOTIONS.find(o => o.value === value)?.label
}

/** 情绪标签 → 分类色；未知标签返回 undefined。 */
export function emotionColor(value: string): string | undefined {
  return SPEAK_EMOTIONS.find(o => o.value === value)?.color
}
