/**
 * chatPrefs.ts — 板块工作台偏好（左栏对话折叠/宽度 + 横道表格窗格宽度/列显隐，
 * 纯函数 + 可注入 storage，刀11；v4.131 刀C 增横道双栏字段）
 *
 * 键 gaea.schedule.chatPrefs：{"collapsed":boolean,"width":number,
 * "ganttTableW":number,"ganttHide":string[],
 * "ganttSort":{"field":string,"dir":"asc"|"desc"},"ganttGroup":string[]}。
 * 宽度钳位 320~680（窄了输入框挤压、宽了横道图不够用）；表格窗格宽度钳位
 * [行号+名称, 可见列总宽]；ganttHide 缺省=进度列收起（v4.135）+ 自定义字段列
 * 全部收起（v4.138 #14，让位画布；列菜单可开）；字段级容错——坏值回落缺省，
 * 单个字段坏不拖累另一个。
 */
import { clampTableW, sanitizeHide, GANTT_CUSTOM_KEYS, GANTT_LEFT_W_FULL } from './ganttCols'

export interface ChatPrefs {
  collapsed: boolean
  width: number
  /** 横道表格窗格宽度（刀C 双栏；缺省=全列展开） */
  ganttTableW: number
  /** 横道隐藏列键集（缺省=进度列+自定义字段列收起，其余全显） */
  ganttHide: string[]
  /** 横道排序（v4.136；缺省=不排序按原顺序） */
  ganttSort: { field: string; dir: 'asc' | 'desc' }
  /** 横道分组字段（v4.136；缺省=不分组；多级=数组顺序即层级，至多两级） */
  ganttGroup: string[]
}

export const CHAT_PREFS_KEY = 'gaea.schedule.chatPrefs'
export const CHAT_WIDTH_MIN = 320
export const CHAT_WIDTH_MAX = 680
export const CHAT_WIDTH_DEFAULT = 420

/** 缺省隐藏列：进度列（v4.135）+ 全部自定义字段列（v4.138 #14）缺省收起（让位画布；列菜单可开） */
const DEFAULT_HIDE: string[] = ['progress', ...GANTT_CUSTOM_KEYS]

const SORT_FIELDS = ['none', 'name', 'duration', 'start', 'progress', 'tf']
const GROUP_FIELDS = ['critical', 'mode', 'milestone']

const DEFAULTS: ChatPrefs = {
  collapsed: false,
  width: CHAT_WIDTH_DEFAULT,
  ganttTableW: GANTT_LEFT_W_FULL,
  ganttHide: DEFAULT_HIDE,
  ganttSort: { field: 'none', dir: 'asc' },
  ganttGroup: [],
}

function parseSort(o: unknown): ChatPrefs['ganttSort'] {
  if (!o || typeof o !== 'object') return { ...DEFAULTS.ganttSort }
  const s = o as { field?: unknown; dir?: unknown }
  const field = typeof s.field === 'string' && SORT_FIELDS.includes(s.field) ? s.field : 'none'
  const dir = s.dir === 'desc' ? 'desc' : 'asc'
  return { field, dir }
}

function parseGroup(o: unknown): string[] {
  if (!Array.isArray(o)) return []
  return o.filter((v): v is string => typeof v === 'string' && GROUP_FIELDS.includes(v)).slice(0, 2)
}

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
      ganttTableW: clampTableW(typeof o.ganttTableW === 'number' ? o.ganttTableW : NaN, GANTT_LEFT_W_FULL),
      // 存量数据无 ganttHide 字段时回落缺省隐藏集；已有数组则逐项容错
      ganttHide: Array.isArray(o.ganttHide) ? sanitizeHide(o.ganttHide) : [...DEFAULT_HIDE],
      ganttSort: parseSort(o.ganttSort),
      ganttGroup: parseGroup(o.ganttGroup),
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
  const base = loadChatPrefs(storage)
  const next: ChatPrefs = {
    collapsed: typeof patch.collapsed === 'boolean' ? patch.collapsed : base.collapsed,
    width: patch.width !== undefined ? clampChatWidth(patch.width) : base.width,
    ganttTableW: patch.ganttTableW !== undefined ? clampTableW(patch.ganttTableW, GANTT_LEFT_W_FULL) : base.ganttTableW,
    ganttHide: patch.ganttHide !== undefined ? sanitizeHide(patch.ganttHide) : base.ganttHide,
    ganttSort: patch.ganttSort !== undefined ? parseSort(patch.ganttSort) : base.ganttSort,
    ganttGroup: patch.ganttGroup !== undefined ? parseGroup(patch.ganttGroup) : base.ganttGroup,
  }
  const s = storage ?? (typeof localStorage !== 'undefined' ? localStorage : undefined)
  try {
    s?.setItem(CHAT_PREFS_KEY, JSON.stringify(next))
  } catch { /* 隐私模式等写失败忽略 */ }
  return next
}
