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

// 流式对话的真实静默超时：收到 delta / reasoning / 搜索进度才重置。旧实现
// 从 runID 起只计一次总时长，会把仍在搜索或慢解码的成功回合误判为失败。
export const STREAM_INACTIVITY_TIMEOUT_MS = 30_000
// 兼容既有测试/导出名；语义已更正为“无真实进度超时”。
export const STREAM_SILENCE_TIMEOUT_MS = STREAM_INACTIVITY_TIMEOUT_MS
// 绝对硬上限：即使持续有进度，也不允许单回合无限挂起。
export const STREAM_HARD_TIMEOUT_MS = 120_000

// 绑定本地引擎可能需要在首个 runID 返回前完成 35B 冷启动。该窗口与后端
// qwen35Ensure（300s）同量级，给后端 20s 收尾余量；runID 一到即切回上面的
// 30s 流式无帧超时，不延长已开始生成的容忍时间。
export const CHAT_ENGINE_START_TIMEOUT_MS = 320_000

// 情绪类别专用色：已收敛到 utils/emotionColors.ts 单源（2026-10-04 第三轮审计 §3.2，
// 此前两处 9 色逐字全等、改色必漏一处）。保留 EMO_COLORS 域名：chat 域消费方
// （emotions.ts / ChatPage.tsx）导入路径零改动。hex 豁免理由见单源文件头注释。
export { EMOTION_COLORS as EMO_COLORS } from '../../utils/emotionColors'
