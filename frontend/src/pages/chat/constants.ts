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
// T6-3.4：旧 localStorage 话题迁移「已完成」持久化标记（版本化键）。
// 迁移成功才写入；失败不写（下次启动可重试），本次会话内仅尝试一次（initRef 守卫）。
export const MIGRATION_KEY = 'gaea_chat_migration_v1'

// T6-3.1：流式对话无帧超时（30s 无任何帧即视为失败）。导出为常量便于测试
// （vitest fake timers 推进同一阈值）；后端正常完成必 emit done、失败必 emit error。
export const STREAM_SILENCE_TIMEOUT_MS = 30_000

// 情绪类别专用色：已收敛到 utils/emotionColors.ts 单源（2026-10-04 第三轮审计 §3.2，
// 此前两处 9 色逐字全等、改色必漏一处）。保留 EMO_COLORS 域名：chat 域消费方
// （emotions.ts / ChatPage.tsx）导入路径零改动。hex 豁免理由见单源文件头注释。
export { EMOTION_COLORS as EMO_COLORS } from '../../utils/emotionColors'
