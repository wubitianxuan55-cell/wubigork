// ChatPage 拆分产物：常量（行为零变化，T6-10.1）
// ── 存储键（旧 localStorage 话题导入 chat.db 后清理） ──
export const STORAGE_KEY = 'gaea_chat_topics'
export const LEGACY_STORAGE_KEY = 'wubigrok_chat_topics'
export const WHISPER_TOPICS_KEY = 'gaea_whisper_topics'
export const LEGACY_WHISPER_TOPICS_KEY = 'wubigrok_whisper_topics'
export const PERSONALITY_KEY = 'gaea_whisper_personality'
export const LEGACY_PERSONALITY_KEY = 'wubigrok_whisper_personality'
export const COMPANION_SETTINGS_KEY = 'gaea_whisper_companion_settings'
export const LEGACY_COMPANION_SETTINGS_KEY = 'wubigrok_whisper_companion_settings'
export const ACTIVE_TOPIC_KEY = 'gaea_chat_active_topic'
export const CHAT_SIDEBAR_KEY = 'gaea.chatSidebarCollapsed'
// T6-3.4：旧 localStorage 话题迁移「已完成」持久化标记（版本化键）。
// 迁移成功才写入；失败不写（下次启动可重试），本次会话内仅尝试一次（initRef 守卫）。
export const MIGRATION_KEY = 'gaea_chat_migration_v1'

// T6-3.1：流式对话无帧超时（30s 无任何帧即视为失败）。导出为常量便于测试
// （vitest fake timers 推进同一阈值）；后端正常完成必 emit done、失败必 emit error。
export const STREAM_SILENCE_TIMEOUT_MS = 30_000

// 情绪类别专用色（9 种情绪各一色，属分类色板而非状态色：映射到成功/警告/次要文字会丢失区分度，
// 故保留专用 hex，UI 走令牌时以 CompanionAvatar emotionColor 独立使用，不随 12 主题）。
export const EMO_COLORS: Record<string, string> = {
  SWEET_ATTACHMENT: '#f472b6', SHY_HEARTBEAT: '#fb7185', TSUNDERE: '#f59e0b',
  HURT_GRIEVANCE: '#a78bfa', ANGRY_ATTACK: '#ef4444', COLD_DETACHED: '#94a3b8',
  FEARFUL_OBEDIENT: '#c084fc', QUIET_FOND: '#fbbf24', CALM_RATIONAL: '#60a5fa',
}
