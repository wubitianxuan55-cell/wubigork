// emotionColors.ts — 轻语情绪→识别色映射（情绪星图 EmotionStarMap / 情绪面板
// WhisperEmotionPanel 共用，v4.361 令牌卫生：此前两处逐字重复，改色必漏一处）。
// hex-exempt：情绪识别色是跨主题恒定的语义色（同 canvas 图表色豁免，不随主题令牌）。
export const EMOTION_COLORS: Record<string, string> = {
  SWEET_ATTACHMENT: '#f472b6', SHY_HEARTBEAT: '#fb7185',
  TSUNDERE: '#f59e0b', HURT_GRIEVANCE: '#a78bfa',
  ANGRY_ATTACK: '#ef4444', COLD_DETACHED: '#94a3b8',
  FEARFUL_OBEDIENT: '#c084fc', QUIET_FOND: '#fbbf24',
  CALM_RATIONAL: '#60a5fa',
}

/** 未知情绪回退冷静蓝 */
export function emotionColor(label: string): string {
  return EMOTION_COLORS[label] || '#60a5fa'
}
