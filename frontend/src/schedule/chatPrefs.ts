/**
 * chatPrefs.ts — 板块工作台偏好（左栏对话折叠/宽度 + 横道表格窗格宽度/列显隐，
 * 纯函数 + 可注入 storage，刀11；v4.131 刀C 增横道双栏字段；
 * W5 列筛持久化增 ganttColFilters）
 *
 * 键 gaea.schedule.chatPrefs：{"collapsed":boolean,"width":number,
 * "ganttTableW":number,"ganttHide":string[],
 * "ganttSort":{"field":string,"dir":"asc"|"desc"},"ganttGroup":string[],
 * "ganttViewId":string,"ganttFilterKind":string,
 * "ganttColFilters":{列键:{"kind":"text"|"number"|"date"|"enum",...}}}。
 * 宽度钳位 320~680（窄了输入框挤压、宽了横道图不够用）；表格窗格宽度钳位
 * [行号+名称, 可见列总宽]；W1 新建缺省=「计划编制」7 列，旧 ganttHide 原样
 * 保留并标记自定义视图；字段级容错——坏值回落缺省，单个字段坏不拖累另一个。
 * 列筛（W5）只存本机偏好：读走 `normalizeColFilters`（未知列键/脏值丢弃），
 * 写走 `serializeColFilters`（空条件剔除），缺省/旧 JSON 无该键 → `{}`；
 * 键名与版本不变，存量偏好不需要迁移。
 */
import { clampTableW, sanitizeHide, GANTT_LEFT_W_FULL } from './ganttCols'
import { normalizeColFilters, serializeColFilters } from './colFilterPrefs'
import type { ColFilters, GanttFilterKind } from './ganttFilter'
import { normalizePreset, type DisplayPreset } from './displayPreset'
import { GANTT_GROUP_OPTIONS, MAX_GANTT_GROUP_DEPTH } from './ganttGroup'
import {
  BUILTIN_GANTT_VIEWS,
  CUSTOM_GANTT_VIEW_ID,
  DEFAULT_GANTT_VIEW_ID,
  builtinViewOf,
  hideForBuiltinView,
  isBuiltinGanttViewId,
  widthForBuiltinView,
} from './viewPresets'

export interface ChatPrefs {
  collapsed: boolean
  width: number
  /** 横道表格窗格宽度（刀C 双栏；缺省=全列展开） */
  ganttTableW: number
  /** 横道隐藏列键集（缺省=进度列+自定义字段列收起，其余全显） */
  ganttHide: string[]
  /** 横道排序（v4.136；缺省=不排序按原顺序） */
  ganttSort: { field: string; dir: 'asc' | 'desc' }
  /** 横道分组字段（缺省=不分组；数组顺序即层级，至多三级） */
  ganttGroup: string[]
  /** 当前内置/自定义视图 id（W1；内置只存本机偏好，不写项目文件） */
  ganttViewId: string
  /** 当前筛选档（里程碑视图需要跨会话保持） */
  ganttFilterKind: GanttFilterKind
  /** 显示预设（W6 护眼/黑白）：只影响 `--sched-*` 领域色，不动主题令牌 */
  ganttDisplayPreset: DisplayPreset
  /** 横道列级筛选（W5；缺省 `{}` =无列筛；只存本机偏好，不进项目视图快照） */
  ganttColFilters: ColFilters
}

export const CHAT_PREFS_KEY = 'gaea.schedule.chatPrefs'
export const CHAT_WIDTH_MIN = 320
export const CHAT_WIDTH_MAX = 680
export const CHAT_WIDTH_DEFAULT = 420

const DEFAULT_VIEW = builtinViewOf(DEFAULT_GANTT_VIEW_ID) ?? BUILTIN_GANTT_VIEWS[0]
/** 缺省隐藏列来自「计划编制」预设；固定行号/名称仍由 visibleCols 兜底。 */
const DEFAULT_HIDE: string[] = hideForBuiltinView(DEFAULT_VIEW)
const DEFAULT_TABLE_W = widthForBuiltinView(DEFAULT_VIEW)

const SORT_FIELDS = ['none', 'name', 'duration', 'start', 'progress', 'tf']
const GROUP_FIELDS = GANTT_GROUP_OPTIONS.map((o) => o.value)
const FILTER_KINDS: GanttFilterKind[] = ['all', 'critical', 'manual', 'milestone']

const DEFAULTS: ChatPrefs = {
  // ZPert 式布局默认先给排程工作台全宽；AI 助手仍可从页头一键展开。
  collapsed: true,
  width: CHAT_WIDTH_DEFAULT,
  ganttTableW: DEFAULT_TABLE_W,
  ganttHide: DEFAULT_HIDE,
  ganttSort: { field: 'none', dir: 'asc' },
  ganttGroup: [],
  ganttViewId: DEFAULT_GANTT_VIEW_ID,
  ganttFilterKind: 'all',
  ganttDisplayPreset: 'normal',
  ganttColFilters: {},
}

/** 缺省副本（列筛是对象，逐次新建，避免调用方改到共享缺省值） */
function freshDefaults(): ChatPrefs {
  return { ...DEFAULTS, ganttColFilters: {} }
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
  const out: string[] = []
  for (const value of o) {
    if (typeof value !== 'string' || !GROUP_FIELDS.includes(value as (typeof GROUP_FIELDS)[number])) continue
    if (!out.includes(value)) out.push(value)
    if (out.length >= MAX_GANTT_GROUP_DEPTH) break
  }
  return out
}

function parseFilterKind(value: unknown): GanttFilterKind {
  return typeof value === 'string' && FILTER_KINDS.includes(value as GanttFilterKind)
    ? value as GanttFilterKind
    : 'all'
}

function parseViewId(value: unknown, hasLegacyHide: boolean): string {
  if (value === CUSTOM_GANTT_VIEW_ID) return CUSTOM_GANTT_VIEW_ID
  if (isBuiltinGanttViewId(value)) return value
  // 存量偏好已有 ganttHide 但尚无 viewId：尊重用户当前列集，不偷偷套新预设。
  return hasLegacyHide ? CUSTOM_GANTT_VIEW_ID : DEFAULT_GANTT_VIEW_ID
}

/** 宽度钳位：非有限值/越界回落缺省与边界 */
export function clampChatWidth(w: number): number {
  if (!Number.isFinite(w)) return CHAT_WIDTH_DEFAULT
  return Math.min(CHAT_WIDTH_MAX, Math.max(CHAT_WIDTH_MIN, Math.round(w)))
}

function parse(raw: string | null): ChatPrefs {
  if (!raw) return freshDefaults()
  try {
    const o = JSON.parse(raw) as Partial<ChatPrefs>
    const hasLegacyHide = Array.isArray((o as Record<string, unknown>).ganttHide)
    return {
      collapsed: typeof o.collapsed === 'boolean' ? o.collapsed : DEFAULTS.collapsed,
      width: clampChatWidth(typeof o.width === 'number' ? o.width : NaN),
      ganttTableW: clampTableW(
        typeof o.ganttTableW === 'number' ? o.ganttTableW : DEFAULT_TABLE_W,
        GANTT_LEFT_W_FULL,
      ),
      // 存量数据无 ganttHide 字段时回落计划编制；已有数组则逐项容错
      ganttHide: Array.isArray(o.ganttHide) ? sanitizeHide(o.ganttHide) : [...DEFAULT_HIDE],
      ganttSort: parseSort(o.ganttSort),
      ganttGroup: parseGroup(o.ganttGroup),
      ganttViewId: parseViewId(o.ganttViewId, hasLegacyHide),
      ganttFilterKind: parseFilterKind(o.ganttFilterKind),
      ganttDisplayPreset: normalizePreset((o as Record<string, unknown>).ganttDisplayPreset as string | undefined),
      // 旧 JSON 无该键 / 坏值 → 归一为空对象（列筛是会话态数据的落盘副本）
      ganttColFilters: normalizeColFilters((o as Record<string, unknown>).ganttColFilters),
    }
  } catch {
    return freshDefaults()
  }
}

/** 读取偏好（storage 可注入供测试；坏 JSON/坏字段逐项容错） */
export function loadChatPrefs(storage?: Pick<Storage, 'getItem'>): ChatPrefs {
  const s = storage ?? (typeof localStorage !== 'undefined' ? localStorage : undefined)
  if (!s) return freshDefaults()
  try {
    return parse(s.getItem(CHAT_PREFS_KEY))
  } catch {
    return freshDefaults()
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
    ganttViewId: patch.ganttViewId !== undefined ? parseViewId(patch.ganttViewId, false) : base.ganttViewId,
    ganttFilterKind: patch.ganttFilterKind !== undefined ? parseFilterKind(patch.ganttFilterKind) : base.ganttFilterKind,
    ganttDisplayPreset: patch.ganttDisplayPreset !== undefined
      ? normalizePreset(patch.ganttDisplayPreset)
      : base.ganttDisplayPreset,
    // 落盘前归一：空条件/未知列键不写进偏好（空对象就写 `{}`，不留垃圾键）
    ganttColFilters: patch.ganttColFilters !== undefined
      ? serializeColFilters(patch.ganttColFilters)
      : base.ganttColFilters,
  }
  const s = storage ?? (typeof localStorage !== 'undefined' ? localStorage : undefined)
  try {
    s?.setItem(CHAT_PREFS_KEY, JSON.stringify(next))
  } catch { /* 隐私模式等写失败忽略 */ }
  return next
}
