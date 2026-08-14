// ChatPage 拆分产物：纯工具函数（行为零变化，T6-10.1）
import * as App from '../../wailsjsCompat'
import {
  STORAGE_KEY, LEGACY_STORAGE_KEY, WHISPER_TOPICS_KEY, LEGACY_WHISPER_TOPICS_KEY,
  PERSONALITY_KEY, LEGACY_PERSONALITY_KEY, COMPANION_SETTINGS_KEY, LEGACY_COMPANION_SETTINGS_KEY,
  MIGRATION_KEY,
} from './constants'
import type { LegacyMsg, LegacyTopic } from './types'

let msgSeq = 0
export function nextMsgKey(): string { msgSeq++; return `m_${msgSeq}_${Date.now()}` }
export function nowStr(): string { return new Date().toISOString() }
export function navigateToCharacterLib(): void {
  window.dispatchEvent(new CustomEvent('navigate', { detail: { page: 'characterlib' } }))
}
export function parseExtra(raw?: string): Record<string, unknown> | undefined {
  if (!raw) return undefined
  try { const o = JSON.parse(raw); return typeof o === 'object' && o ? o : undefined } catch (_) { return undefined }
}

/** 话题最近活跃时间（优先 updated_at，缺失回退 created_at）。 */
export function toUpdatedAt(t: { updated_at?: string; created_at?: string }): number {
  const raw = t?.updated_at || t?.created_at
  const ms = raw ? new Date(String(raw)).getTime() : 0
  return Number.isFinite(ms) && ms > 0 ? ms : Date.now()
}

export function loadPersonality(): string {
  try {
    return (localStorage.getItem(PERSONALITY_KEY) ?? localStorage.getItem(LEGACY_PERSONALITY_KEY)) || 'gaea'
  } catch (_) { return 'gaea' }
}

export function loadCompanionName(personalityLabel: string): string {
  try {
    const raw = localStorage.getItem(COMPANION_SETTINGS_KEY) ?? localStorage.getItem(LEGACY_COMPANION_SETTINGS_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (parsed?.companionName) return parsed.companionName
    }
  } catch (_) {}
  return personalityLabel || 'gaea'
}

// T6-3：前端错误统一上报 gaea.log（GaeaLogFrontendError，T6-1.2 通道）。
// 日志通道自身异常时静默降级，绝不掩盖原始错误。
export function logFrontendError(message: string): void {
  try {
    App.GaeaLogFrontendError?.(message)?.catch(() => {})
  } catch (_) {
    // 日志通道未注入（dev 等）时忽略
  }
}

/** 旧 localStorage 话题 → chat.db（一次性；成功后清理本地键并写迁移标记）。 */
export async function migrateLegacyTopics(): Promise<boolean> {
  // T6-3.4：已迁移过（持久化标记）→ 直接跳过，避免每次初始化都重复执行。
  try { if (localStorage.getItem(MIGRATION_KEY) === '1') return false } catch (_) {}
  const buckets: Array<{ title: string; mode: string; messages: LegacyMsg[] }> = []
  const chatRaw = localStorage.getItem(STORAGE_KEY) ?? localStorage.getItem(LEGACY_STORAGE_KEY)
  if (chatRaw) {
    try {
      const p = JSON.parse(chatRaw)
      if (Array.isArray(p)) p.forEach((t: LegacyTopic) => buckets.push({ title: t.title || '新对话', mode: 'plain', messages: t.messages || [] }))
    } catch (_) {}
  }
  const whisperRaw = localStorage.getItem(WHISPER_TOPICS_KEY) ?? localStorage.getItem(LEGACY_WHISPER_TOPICS_KEY)
  if (whisperRaw) {
    try {
      const p = JSON.parse(whisperRaw)
      if (Array.isArray(p)) p.forEach((t: LegacyTopic) => buckets.push({ title: t.title || '新对话', mode: loadPersonality(), messages: t.messages || [] }))
    } catch (_) {}
  }
  if (buckets.length === 0) return false
  let failed = false
  for (const t of buckets) {
    const msgs = (t.messages || [])
      .filter(m => m.role === 'user' || m.role === 'assistant')
      .map(m => ({ Role: m.role, Content: typeof m.content === 'string' ? m.content : '', Extra: '' }))
    try {
      await App.ChatImportTopic(t.title, t.mode, msgs)
    } catch (err: unknown) {
      // T6-3.4：导入失败不再静默吞掉——记录日志；失败不写迁移标记
      // （下次启动可重试）。本次会话内仅尝试一次（initRef 守卫），不无限重试。
      failed = true
      const errText = err instanceof Error ? err.message : String(err)
      console.error('[Chat] 旧话题导入失败（' + t.title + '）:', errText)
      logFrontendError('旧话题导入失败（' + t.title + '）: ' + errText)
    }
  }
  if (failed) return false // 有失败：不写标记、不清理本地键（保留数据供下次重试）
  try {
    localStorage.removeItem(STORAGE_KEY); localStorage.removeItem(LEGACY_STORAGE_KEY)
    localStorage.removeItem(WHISPER_TOPICS_KEY); localStorage.removeItem(LEGACY_WHISPER_TOPICS_KEY)
    localStorage.setItem(MIGRATION_KEY, '1')
  } catch (_) {}
  return true
}
