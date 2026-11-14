/**
 * schedule/inplaceEdit.ts — 就地编辑弹层族的值语义（纯函数，W5）
 *
 * 对标斑马 `ZF*` 统一弹层族（日期 / 日期区间 / 数值 / 文本，一套构造函数）：
 * 解析、校验、夹取、清空口径全部收敛到本模块，组件（InplaceEditField）只做
 * 「取草稿 → parseInplace → 写回 or 原位提示」，不各写一套校验。
 *
 * 四类值的提交语义（口径即契约，测试锁定）：
 *  - date      ：仅 `YYYY-MM-DD` 且日历存在（2026-02-30 非法）；空串=清空（null）；
 *  - dateRange ：`{from,to}`，两边都填时 from<=to；单边空=开区间（该侧 null）；
 *                两边都空=整区间清空；
 *  - number    ：min/max 夹取 + step 对齐（先对齐后夹取）；NaN/非有限值不提交；
 *                空串默认=清空（null）——工期类要落 0 必须显式声明 emptyAs:'zero'，
 *                绝不隐式把空当成 0（空工期≠零工期，是两种业务含义）；
 *  - text      ：trim；超 maxLen 截断（与数值夹取同口径，不抛错）；空串=清空（null）。
 *
 * 返回值一律 discriminated union：`{ok:true,value,...}` / `{ok:false,reason}`，
 * 调用方按 `ok` 收窄即可，reason 为机器码、message 为原位提示文案。
 * 全部纯函数零 DOM、零 React，可单测（node 环境）。
 */
import type { CustomFieldType } from './customFields'

/** 就地编辑四类值（=斑马 ZFDate/ZFDateRange/ZFNumberical/ZFText） */
export type InplaceKind = 'date' | 'dateRange' | 'number' | 'text'

export const INPLACE_KINDS: InplaceKind[] = ['date', 'dateRange', 'number', 'text']

/** 校验失败机器码（UI 原位提示按此取文案，文案本身在 message 里给全） */
export type InplaceReason =
  /** 非 `YYYY-MM-DD` 或日历上不存在 */
  | 'invalidDate'
  /** 区间单侧格式非法 */
  | 'invalidRange'
  /** from > to */
  | 'rangeOrder'
  /** 非数值 / NaN / Infinity */
  | 'notNumber'
  /** 该列禁止为空（required 声明） */
  | 'emptyForbidden'

/** 按期种类生效的规格（无关字段忽略：date/dateRange 不看 min/max 等） */
export interface InplaceSpec {
  /** text：最大长度（超出截断） */
  maxLen?: number
  /** number：下限 / 上限（夹取） */
  min?: number
  max?: number
  /** number：步长（对齐；缺省不对齐） */
  step?: number
  /**
   * number 空串语义：缺省 `'clear'` → 提交 null（清空该槽）；
   * `'zero'` → 落 0。工期/进度这类「0 是合法业务值」的列须显式声明，
   * 否则清空不会被悄悄改写成 0。
   */
  emptyAs?: 'clear' | 'zero'
  /** text/number：禁止为空（必填列 → emptyForbidden） */
  required?: boolean
}

/** 日期区间值（null=该侧开区间） */
export interface InplaceDateRangeValue {
  from: string | null
  to: string | null
}

/** 校验通过：value 已规范化（null=清空） */
export interface InplaceOkBase {
  ok: true
  /** 是否清空（date/text/number 的 null；区间的两边皆空） */
  cleared: boolean
  /** 是否被夹取/对齐/截断（提示用户"我改了你的输入"） */
  adjusted: boolean
}

export interface InplaceFail {
  ok: false
  kind: InplaceKind
  reason: InplaceReason
  /** 原位提示文案（中文，直接显示在弹层内） */
  message: string
}

export type InplaceResult =
  | (InplaceOkBase & { kind: 'date'; value: string | null })
  | (InplaceOkBase & { kind: 'dateRange'; value: InplaceDateRangeValue })
  | (InplaceOkBase & { kind: 'number'; value: number | null })
  | (InplaceOkBase & { kind: 'text'; value: string | null })
  | InplaceFail

/** 弹层草稿（组件按此形态提交，kind 决定读哪些字段） */
export type InplaceRaw =
  | { kind: 'date'; text: string }
  | { kind: 'dateRange'; from: string; to: string }
  | { kind: 'number'; text: string }
  | { kind: 'text'; text: string }

const REASON_MESSAGE: Record<InplaceReason, string> = {
  invalidDate: '日期需为 YYYY-MM-DD 格式且日历存在',
  invalidRange: '区间日期需为 YYYY-MM-DD 格式（留空表示不限）',
  rangeOrder: '开始日期不能晚于结束日期',
  notNumber: '请输入有效数值',
  emptyForbidden: '该列不允许为空',
}

function fail(kind: InplaceKind, reason: InplaceReason): InplaceFail {
  return { ok: false, kind, reason, message: REASON_MESSAGE[reason] }
}

/** `YYYY-MM-DD` 严格判定：格式 + 真实日历（UTC 往返，防 2026-02-30 之类的"看起来合法"） */
export function isIsoDate(text: string): boolean {
  const s = text.trim()
  if (!/^\d{4}-\d{2}-\d{2}$/.test(s)) return false
  const ms = Date.parse(`${s}T00:00:00Z`)
  if (!Number.isFinite(ms)) return false
  return new Date(ms).toISOString().slice(0, 10) === s
}

/** 日期：`YYYY-MM-DD`，空串=清空；非法不提交 */
export function parseDateText(text: string, spec: InplaceSpec = {}): InplaceResult {
  const s = text.trim()
  if (s === '') {
    if (spec.required) return fail('date', 'emptyForbidden')
    return { ok: true, kind: 'date', value: null, cleared: true, adjusted: false }
  }
  if (!isIsoDate(s)) return fail('date', 'invalidDate')
  return { ok: true, kind: 'date', value: s, cleared: false, adjusted: false }
}

/** 日期区间：单边空=开区间，两边空=清空；两边都填时 from<=to */
export function parseDateRangeText(from: string, to: string, spec: InplaceSpec = {}): InplaceResult {
  const f = from.trim()
  const t = to.trim()
  if (f === '' && t === '') {
    if (spec.required) return fail('dateRange', 'emptyForbidden')
    return {
      ok: true,
      kind: 'dateRange',
      value: { from: null, to: null },
      cleared: true,
      adjusted: false,
    }
  }
  if ((f !== '' && !isIsoDate(f)) || (t !== '' && !isIsoDate(t))) return fail('dateRange', 'invalidRange')
  // ISO 定长零填充：字符串序即时间序，无需 Date 比较
  if (f !== '' && t !== '' && f > t) return fail('dateRange', 'rangeOrder')
  return {
    ok: true,
    kind: 'dateRange',
    value: { from: f === '' ? null : f, to: t === '' ? null : t },
    cleared: false,
    adjusted: false,
  }
}

/** 数值：NaN 不提交；空=清空（emptyAs:'zero' 才落 0）；step 对齐后夹取 */
export function parseNumberText(text: string, spec: InplaceSpec = {}): InplaceResult {
  const s = text.trim()
  if (s === '') {
    if (spec.required) return fail('number', 'emptyForbidden')
    if (spec.emptyAs === 'zero') {
      return { ok: true, kind: 'number', value: 0, cleared: false, adjusted: false }
    }
    return { ok: true, kind: 'number', value: null, cleared: true, adjusted: false }
  }
  const n = Number(s)
  if (!Number.isFinite(n)) return fail('number', 'notNumber')
  let v = n
  let adjusted = false
  const step = spec.step
  if (step !== undefined && step > 0) {
    const base = spec.min ?? 0
    const snapped = base + Math.round((n - base) / step) * step
    // 浮点步长（0.1 之类）对齐后去噪：6 位小数足够工程口径
    const clean = Number(snapped.toFixed(6))
    if (clean !== n) adjusted = true
    v = clean
  }
  if (spec.min !== undefined && v < spec.min) {
    v = spec.min
    adjusted = true
  }
  if (spec.max !== undefined && v > spec.max) {
    v = spec.max
    adjusted = true
  }
  return { ok: true, kind: 'number', value: v, cleared: false, adjusted }
}

/** 文本：trim、超长截断、空串=清空 */
export function parseTextInput(text: string, spec: InplaceSpec = {}): InplaceResult {
  const s = text.trim()
  if (s === '') {
    if (spec.required) return fail('text', 'emptyForbidden')
    return { ok: true, kind: 'text', value: null, cleared: true, adjusted: false }
  }
  if (spec.maxLen !== undefined && s.length > spec.maxLen) {
    return { ok: true, kind: 'text', value: s.slice(0, spec.maxLen), cleared: false, adjusted: true }
  }
  return { ok: true, kind: 'text', value: s, cleared: false, adjusted: false }
}

/** 统一入口：按草稿 kind 分派（组件只需调这一个） */
export function parseInplace(raw: InplaceRaw, spec: InplaceSpec = {}): InplaceResult {
  switch (raw.kind) {
    case 'date':
      return parseDateText(raw.text, spec)
    case 'dateRange':
      return parseDateRangeText(raw.from, raw.to, spec)
    case 'number':
      return parseNumberText(raw.text, spec)
    case 'text':
      return parseTextInput(raw.text, spec)
  }
}

/** 列表头/单元格共用：列键 → 就地编辑种类（未知列=null，不开弹层） */
export const INPLACE_COL_KIND: Record<string, InplaceKind> = {
  name: 'text',
  dur: 'number',
  progress: 'number',
  start: 'date',
  finish: 'date',
  text1: 'text',
  text2: 'text',
  text3: 'text',
  num1: 'number',
  num2: 'number',
}

/** 列键 → 就地编辑种类；自定义列按注册表类型（text/number）落定 */
export function inplaceKindOfCol(colKey: string, customType?: CustomFieldType): InplaceKind | null {
  if (customType) return customType === 'number' ? 'number' : 'text'
  return INPLACE_COL_KIND[colKey] ?? null
}

/** 列键 → 缺省规格（工期/进度「0 是合法业务值」→ 显式 emptyAs:'zero'） */
export const INPLACE_COL_SPEC: Record<string, InplaceSpec> = {
  name: { maxLen: 200 },
  dur: { min: 0, max: 3650, step: 1, emptyAs: 'zero' },
  progress: { min: 0, max: 100, step: 1, emptyAs: 'zero' },
  text1: { maxLen: 200 },
  text2: { maxLen: 200 },
  text3: { maxLen: 200 },
}

/** 列键 → 规格（未登记列给空规格=不夹取） */
export function inplaceSpecOfCol(colKey: string): InplaceSpec {
  return INPLACE_COL_SPEC[colKey] ?? {}
}
