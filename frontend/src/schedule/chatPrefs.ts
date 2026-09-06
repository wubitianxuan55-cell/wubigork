/**
 * chatPrefs.ts — 左栏对话偏好（折叠态/宽度，纯函数 + 可注入 storage，刀11）
 *
 * 键 gaea.schedule.chatPrefs：{"collapsed":boolean,"width":number}。
 * 宽度钳位 320~680（窄了输入框挤压、宽了横道图不够用）；字段级容错——
 * 坏值回落缺省，单个字段坏不拖累另一个。
 */
export interface ChatPrefs {
  collapsed: boolean
  width: number
}

export const CHAT_PREFS_KEY = 'gaea.schedule.chatPrefs'
export const CHAT_WIDTH_MIN = 320
export const CHAT_WIDTH_MAX = 680
export const CHAT_WIDTH_DEFAULT = 420

const DEFAULTS: ChatPrefs = { collapsed: false, width: CHAT_WIDTH_DEFAULT }

/** 宽度钳位：非有限值/越界回落缺省与边界 */
export function clampChatWidth(w: number): number {
  if (!Number.isFinite(w)) return CHAT_WIDTH_DEFAULT
  return Math.min(CHAT_WIDTH_MAX, Math.max(CHAT_WIDTH_MIN, Math.round(w)))
}

function parse(raw: string | null): ChatPrefs {
  if (!raw) return { ...DEFAULTS }
  try {
    const o = JSON.parse(raw) as Partial<ChatPrefs>
    return {
      collapsed: typeof o.collapsed === 'boolean' ? o.collapsed : DEFAULTS.collapsed,
      width: clampChatWidth(typeof o.width === 'number' ? o.width : NaN),
    }
  } catch {
    return { ...DEFAULTS }
  }
}

/** 读取偏好（storage 可注入供测试；坏 JSON/坏字段逐项容错） */
export function loadChatPrefs(storage?: Pick<Storage, 'getItem'>): ChatPrefs {
  const s = storage ?? (typeof localStorage !== 'undefined' ? localStorage : undefined)
  if (!s) return { ...DEFAULTS }
  try {
    return parse(s.getItem(CHAT_PREFS_KEY))
  } catch {
    return { ...DEFAULTS }
  }
}

/** 保存偏好（部分字段合并；storage 不可写静默失败——偏好属锦上添花） */
export function saveChatPrefs(patch: Partial<ChatPrefs>, storage?: Pick<Storage, 'getItem' | 'setItem'>): ChatPrefs {
  const next: ChatPrefs = {
    collapsed: typeof patch.collapsed === 'boolean' ? patch.collapsed : loadChatPrefs(storage).collapsed,
    width: patch.width !== undefined ? clampChatWidth(patch.width) : loadChatPrefs(storage).width,
  }
  const s = storage ?? (typeof localStorage !== 'undefined' ? localStorage : undefined)
  try {
    s?.setItem(CHAT_PREFS_KEY, JSON.stringify(next))
  } catch { /* 隐私模式等写失败忽略 */ }
  return next
}
