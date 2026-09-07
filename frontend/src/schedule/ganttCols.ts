/**
 * ganttCols.ts — 横道表格列定义与显隐纯函数（v4.131.0 刀C：工作台双栏）
 *
 * 口径：行号/任务名称为不可隐藏的最小可用集；其余列可按需收纳，
 * 让位后面的时间画布。表格窗格宽度独立可拖（分隔条），拖窄=收纳表格、
 * 拖宽=展开；列显隐与窗格宽度均持久化（chatPrefs）。
 * 全部纯函数零 DOM。
 */

export interface GanttCol {
  key: string
  label: string
  w: number
}

export const GANTT_COLS: GanttCol[] = [
  { key: 'no', label: '', w: 34 },
  { key: 'name', label: '任务名称', w: 160 },
  { key: 'wbs', label: 'WBS', w: 44 },
  { key: 'dur', label: '工期', w: 48 },
  { key: 'progress', label: '进度', w: 56 },
  { key: 'start', label: '开始', w: 66 },
  { key: 'finish', label: '完成', w: 66 },
  { key: 'ls', label: '最迟开始', w: 66 },
  { key: 'lf', label: '最迟完成', w: 66 },
  { key: 'tf', label: '总时差', w: 44 },
  { key: 'ff', label: '自由时差', w: 58 },
  { key: 'mode', label: '模式', w: 48 },
  { key: 'preds', label: '前置', w: 68 },
  { key: 'succ', label: '后续', w: 68 },
  { key: 'cost', label: '成本', w: 68 },
]

/** 不可隐藏列：行号+任务名称（表格最小可用集） */
export const GANTT_FIXED_KEYS: string[] = ['no', 'name']

/** 可隐藏列键（显隐菜单口径，键集校验用） */
export const GANTT_HIDEABLE_KEYS: string[] = GANTT_COLS.map((c) => c.key).filter((k) => !GANTT_FIXED_KEYS.includes(k))

/** 全列展开的表格总宽（分隔条双击复位目标） */
export const GANTT_LEFT_W_FULL: number = GANTT_COLS.reduce((s, c) => s + c.w, 0)

/** 表格窗格最小宽：行号+名称（拖到最窄=收纳表格只留身份列，画布全屏） */
export const GANTT_TABLE_W_MIN: number = GANTT_COLS.filter((c) => GANTT_FIXED_KEYS.includes(c.key)).reduce((s, c) => s + c.w, 0)

/** 上报 6 列键序（刀D1 导出图面预设；对标 MS Project 上报件：标识号/任务名称/工期/开始/完成/前置） */
export const GANTT_REPORT_KEYS: string[] = ['no', 'name', 'dur', 'start', 'finish', 'preds']

/** 按键集取列定义（保持 GANTT_COLS 既有顺序） */
export function colsByKeys(keys: string[]): GanttCol[] {
  const on = new Set(keys)
  return GANTT_COLS.filter((c) => on.has(c.key))
}

/** 可见列序列：固定列（行号/名称）恒保留，其余滤除 hide（未知键在 sanitizeHide 已滤，此处再防御） */
export function visibleCols(hide: string[]): GanttCol[] {
  const off = new Set(hide)
  return GANTT_COLS.filter((c) => GANTT_FIXED_KEYS.includes(c.key) || !off.has(c.key))
}

/** 可见列总宽（=表格内容宽度） */
export function visibleLeftW(hide: string[]): number {
  return visibleCols(hide).reduce((s, c) => s + c.w, 0)
}

/** 表格窗格宽度钳位：非有限值回落全量；钳在 [最小宽, 可见列总宽] */
export function clampTableW(w: number, leftW: number): number {
  if (!Number.isFinite(w)) return leftW
  return Math.min(leftW, Math.max(GANTT_TABLE_W_MIN, Math.round(w)))
}

/** 显隐键集归一：只留可隐藏合法键、去重、上限 32（脏数据/agent 手写兜底） */
export function sanitizeHide(raw: unknown): string[] {
  if (!Array.isArray(raw)) return []
  const out: string[] = []
  for (const v of raw) {
    if (typeof v !== 'string' || !GANTT_HIDEABLE_KEYS.includes(v)) continue
    if (!out.includes(v)) out.push(v)
    if (out.length >= 32) break
  }
  return out
}
