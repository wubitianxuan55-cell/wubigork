/**
 * schedule/blocks.ts — 斑马复刻新增区块的结构归一（检视 / 点-箭 / 图面参数 / 视图）
 *
 * Why：`reviews` / `aoa` / `networkStyle` / `views` 四块是"复刻斑马"的地基承载位
 * （见 docs/gaea-schedule-zpert8-network-re-20261114.md §3/§4/§5/§6）。v1 只要求
 * **能存能读不丢**，不接 UI、不进 CPM；但坏形数据必须在进板块前就被挡掉——否则
 * 一个手改坏的 .gsched.json 会让整个板块打不开（`normalizeProject` 是唯一入口）。
 *
 * How：每个 normalize 都是纯函数，风格与 `normalizeAoaLayout` / `normalizeCalendar`
 * 一致：非对象丢弃、逐条容错、超上限截断、无有效内容返回 `undefined`（保持
 * 「缺省=旧行为」语义，旧文件零迁移）。落盘的最后一道闸仍是 Go `Validate` fail-closed。
 */

import type {
  SchedAoaArrow,
  SchedAoaDot,
  SchedAoaGraph,
  SchedNetworkStyle,
  SchedRenderMode,
  SchedReview,
  SchedReviewActual,
  SchedView,
} from './types'

/** 检视序列上限（斑马无硬上限，取"一月一次约两年半"的量级做护栏） */
export const MAX_REVIEWS = 30
/** 视图上限（斑马内置 10 + 自定义若干，20 足以覆盖） */
export const MAX_VIEWS = 20
/** 点-箭图元上限（防手改文件把板块拖死） */
export const MAX_AOA_DOTS = 20000
export const MAX_AOA_ARROWS = 60000

const RENDER_MODES: SchedRenderMode[] = ['timescaled', 'timescaledLogic', 'logic', 'pdm', 'ganttNetwork']
const VIEW_KINDS: SchedView['kind'][] = ['gantt', 'pdm', 'aoa', 'usage', 'dcma', 'monte', 'evm']
const ARROW_KINDS: SchedAoaArrow['kind'][] = ['task', 'dummy', 'summary']

/** YYYY-MM-DD（只校验形状；真实日期合法性由 Go Validate 承担） */
export function isISODate(s: unknown): s is string {
  return typeof s === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(s)
}

function isNonNegInt(v: unknown): v is number {
  return typeof v === 'number' && Number.isInteger(v) && v >= 0
}

function clampInt(v: unknown, lo: number, hi: number): number | undefined {
  if (typeof v !== 'number' || !Number.isFinite(v)) return undefined
  const n = Math.round(v)
  return n < lo || n > hi ? undefined : n
}

function strArray(raw: unknown, maxLen = 64): string[] | undefined {
  if (!Array.isArray(raw)) return undefined
  const out: string[] = []
  for (const v of raw) {
    if (typeof v !== 'string' || v === '') continue
    out.push(v)
    if (out.length >= maxLen) break
  }
  return out
}

/** 逐任务实际数据：只保留至少有一项有效值的条目 */
function normalizeActual(raw: unknown): SchedReviewActual | undefined {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return undefined
  const r = raw as Record<string, unknown>
  const out: SchedReviewActual = {}
  if (isNonNegInt(r.start)) out.start = r.start
  if (isNonNegInt(r.finish)) out.finish = r.finish
  if (typeof r.percent === 'number' && Number.isFinite(r.percent)) {
    out.percent = Math.min(100, Math.max(0, Math.round(r.percent)))
  }
  return Object.keys(out).length > 0 ? out : undefined
}

/**
 * 检视序列归一：
 * - `at` 必须为非负整数、`date` 必须 YYYY-MM-DD，否则整条丢弃（跨口径的条目不可救）；
 * - 同一 `at` 只保留**最后**一条（后写覆盖，与斑马"新增的检视时间已存在"的反面口径一致：
 *   文件被手改成重复时以靠后的为准，不报错）；
 * - 按 `at` 升序重排（渲染与对比表都按时间轴读）；
 * - 超过 MAX_REVIEWS 时保留**最近的**若干条（历史头部可丢，近期不可丢）。
 */
export function normalizeReviews(raw: unknown): SchedReview[] | undefined {
  if (!Array.isArray(raw)) return undefined
  const byAt = new Map<number, SchedReview>()
  for (const item of raw) {
    if (!item || typeof item !== 'object' || Array.isArray(item)) continue
    const r = item as Record<string, unknown>
    if (!isNonNegInt(r.at) || !isISODate(r.date)) continue
    const actuals: Record<string, SchedReviewActual> = {}
    if (r.actuals && typeof r.actuals === 'object' && !Array.isArray(r.actuals)) {
      for (const [tid, av] of Object.entries(r.actuals as Record<string, unknown>)) {
        if (!tid) continue
        const a = normalizeActual(av)
        if (a) actuals[tid] = a
      }
    }
    const rec: SchedReview = { at: r.at, date: r.date, actuals }
    if (r.frozen === true) rec.frozen = true
    if (typeof r.note === 'string' && r.note !== '') rec.note = r.note
    byAt.set(r.at, rec)
  }
  if (byAt.size === 0) return undefined
  const list = [...byAt.values()].sort((a, b) => a.at - b.at)
  return list.length > MAX_REVIEWS ? list.slice(list.length - MAX_REVIEWS) : list
}

/**
 * 点-箭图归一：
 * - 点：`id` 非空且唯一、`num` 为正整数；重复 `id` 丢后一条；
 * - 箭：`from`/`to` 必须命中已保留的点、`kind` 合法、`id` 非空；否则丢该箭；
 * - 任一侧为空 → 返回 undefined（"有点无箭"在 v1 无意义，等价于未启用持久图元）。
 */
export function normalizeAoaGraph(raw: unknown): SchedAoaGraph | undefined {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return undefined
  const g = raw as Record<string, unknown>
  const dots: SchedAoaDot[] = []
  const seenDot = new Set<string>()
  if (Array.isArray(g.dots)) {
    for (const item of g.dots) {
      if (dots.length >= MAX_AOA_DOTS) break
      if (!item || typeof item !== 'object' || Array.isArray(item)) continue
      const d = item as Record<string, unknown>
      if (typeof d.id !== 'string' || d.id === '' || seenDot.has(d.id)) continue
      if (!isNonNegInt(d.num) || d.num < 1) continue
      const dot: SchedAoaDot = { id: d.id, num: d.num }
      const layer = clampInt(d.layer, 0, 10000)
      if (layer !== undefined) dot.layer = layer
      if (d.isBegin === true) dot.isBegin = true
      if (d.isEnd === true) dot.isEnd = true
      seenDot.add(d.id)
      dots.push(dot)
    }
  }
  const arrows: SchedAoaArrow[] = []
  const seenArrow = new Set<string>()
  if (Array.isArray(g.arrows)) {
    for (const item of g.arrows) {
      if (arrows.length >= MAX_AOA_ARROWS) break
      if (!item || typeof item !== 'object' || Array.isArray(item)) continue
      const a = item as Record<string, unknown>
      if (typeof a.id !== 'string' || a.id === '' || seenArrow.has(a.id)) continue
      if (typeof a.from !== 'string' || typeof a.to !== 'string') continue
      if (!seenDot.has(a.from) || !seenDot.has(a.to) || a.from === a.to) continue
      const kind = ARROW_KINDS.includes(a.kind as SchedAoaArrow['kind'])
        ? (a.kind as SchedAoaArrow['kind'])
        : undefined
      if (!kind) continue
      const arrow: SchedAoaArrow = { id: a.id, from: a.from, to: a.to, kind }
      if (typeof a.taskId === 'string' && a.taskId !== '') arrow.taskId = a.taskId
      // 时距：斑马闸门为 ±9999（见逆向报告 §3.1）
      const lag = clampInt(a.lag, -9999, 9999)
      if (lag !== undefined) arrow.lag = lag
      seenArrow.add(a.id)
      arrows.push(arrow)
    }
  }
  if (dots.length === 0 || arrows.length === 0) return undefined
  const out: SchedAoaGraph = { dots, arrows }
  if (typeof g.beginDotId === 'string' && seenDot.has(g.beginDotId)) out.beginDotId = g.beginDotId
  return out
}

/**
 * 图面参数归一：只认已知键；枚举非法值丢弃该键（不整块丢），数值按区间夹取。
 * 全部键都无效 → 返回 undefined（缺省=现有渲染口径）。
 */
export function normalizeNetworkStyle(raw: unknown): SchedNetworkStyle | undefined {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return undefined
  const r = raw as Record<string, unknown>
  const out: SchedNetworkStyle = {}
  if (RENDER_MODES.includes(r.mode as SchedRenderMode)) out.mode = r.mode as SchedRenderMode
  if (r.dotStyle === 'circle' || r.dotStyle === 'rect') out.dotStyle = r.dotStyle
  if (r.arrowStyle === 'solid' || r.arrowStyle === 'hollow') out.arrowStyle = r.arrowStyle
  if (r.namePos === 'auto' || r.namePos === 'horizontal' || r.namePos === 'vertical') out.namePos = r.namePos
  const nodeRadius = clampInt(r.nodeRadius, 4, 200)
  if (nodeRadius !== undefined) out.nodeRadius = nodeRadius
  const headHeight = clampInt(r.headHeight, 2, 60)
  if (headHeight !== undefined) out.headHeight = headHeight
  const lh1 = clampInt(r.layerHeight1, 20, 2000)
  if (lh1 !== undefined) out.layerHeight1 = lh1
  const lh2 = clampInt(r.layerHeight2, 20, 2000)
  if (lh2 !== undefined) out.layerHeight2 = lh2
  if (typeof r.stagger === 'boolean') out.stagger = r.stagger
  if (typeof r.hideUnimportantLink === 'boolean') out.hideUnimportantLink = r.hideUnimportantLink
  if (r.tickUnit === 'yearMonth' || r.tickUnit === 'monthDay' || r.tickUnit === 'dayHour') out.tickUnit = r.tickUnit
  if (r.tickLines === 'none' || r.tickLines === 'small' || r.tickLines === 'large') out.tickLines = r.tickLines
  if (r.density === 'compact' || r.density === 'normal' || r.density === 'loose') out.density = r.density
  if (typeof r.aspectMax === 'number' && Number.isFinite(r.aspectMax) && r.aspectMax >= 0.1 && r.aspectMax <= 20) {
    out.aspectMax = r.aspectMax
  }
  const maxPages = clampInt(r.maxPages, 1, 1000)
  if (maxPages !== undefined) out.maxPages = maxPages
  return Object.keys(out).length > 0 ? out : undefined
}

/**
 * 视图快照归一：`id`/`name` 非空、`kind` 合法、`id` 唯一；列/分组/排序按形状过滤。
 * 空列表 → undefined。
 */
export function normalizeViews(raw: unknown): SchedView[] | undefined {
  if (!Array.isArray(raw)) return undefined
  const out: SchedView[] = []
  const seen = new Set<string>()
  for (const item of raw) {
    if (out.length >= MAX_VIEWS) break
    if (!item || typeof item !== 'object' || Array.isArray(item)) continue
    const v = item as Record<string, unknown>
    if (typeof v.id !== 'string' || v.id === '' || seen.has(v.id)) continue
    if (typeof v.name !== 'string' || v.name.trim() === '') continue
    if (!VIEW_KINDS.includes(v.kind as SchedView['kind'])) continue
    const view: SchedView = { id: v.id, name: v.name.trim(), kind: v.kind as SchedView['kind'] }
    const cols = strArray(v.cols)
    if (cols && cols.length > 0) view.cols = cols
    const hide = strArray(v.hide)
    if (hide && hide.length > 0) view.hide = hide
    const groupBy = strArray(v.groupBy, 8)
    if (groupBy && groupBy.length > 0) view.groupBy = groupBy
    if (typeof v.filter === 'string' && v.filter !== '') view.filter = v.filter
    if (v.sort && typeof v.sort === 'object' && !Array.isArray(v.sort)) {
      const s = v.sort as Record<string, unknown>
      if (typeof s.field === 'string' && s.field !== '' && (s.dir === 'asc' || s.dir === 'desc')) {
        view.sort = { field: s.field, dir: s.dir }
      }
    }
    if (RENDER_MODES.includes(v.renderMode as SchedRenderMode)) {
      view.renderMode = v.renderMode as SchedRenderMode
    }
    seen.add(v.id)
    out.push(view)
  }
  return out.length > 0 ? out : undefined
}
