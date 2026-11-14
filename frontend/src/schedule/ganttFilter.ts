/**
 * ganttFilter.ts — 横道任务行筛选（纯函数，v4.135.0 刀C 余项 + W5 列级筛选）
 *
 * 四种筛选模（全部/只看关键/只看手动/只看里程碑）+ 名称文本包含（不区分大小写）
 * + **五类列过滤器**（W5）：文本包含 / 数值比较（≥ ≤ =）/ 日期区间 / 枚举集合 / 空值口径。
 * 返回可见行的**原索引**序列（表格/条形/依赖线共用同一可见集，行号列仍按
 * 全表序号显示）。分组行保留规则：至少一个可见子孙才保留——分组行自身
 * 不是关键/手动，不能按叶条件误杀。网络图视图不参与筛选（逻辑图语义）。
 *
 * 列筛口径（W5，与 `inplaceEdit` 的四类就地编辑同族）：
 *  - `GANTT_FILTER_DEFAULT` 与既有 `filterGanttRows` 三参调用**逐字不变**
 *    （`colFilters` 缺省=无列筛=行为与拆分前完全一致）；
 *  - 列筛是**硬约束**：文本强制集（分组名命中→整支可见）也要过列筛；
 *  - 未知列键 / 无法取值的列筛 **fail-open**（不误杀行）：日期列筛在缺
 *    `ctx.startDate` 时直接跳过（不参与），不给"全表被清空"的假象。
 */
import type { CpmResult, SchedCalendar, SchedTask } from './types'
import { descendantIds } from './store'
import { wdToDate, isoOf } from './calendar'

export type GanttFilterKind = 'all' | 'critical' | 'manual' | 'milestone'
/** 筛选档全集（views.ts/chatPrefs 共用单源；chatPrefs 本地副本待其刀收编） */
export const FILTER_KINDS: GanttFilterKind[] = ['all', 'critical', 'manual', 'milestone']

/** 列筛编辑器类型（文本/数值/日期/枚举；空值口径是独立开关，不占一类） */
export type ColFilterKind = 'text' | 'number' | 'date' | 'enum'

/** 单列筛选值（未设置的字段=不约束；`empty` 与其它条件为 AND 关系） */
export interface ColFilterValue {
  kind: ColFilterKind
  /** text：子串包含（不区分大小写；trim 后空串=该列不约束） */
  text?: string
  /** number：比较符（缺省 `ge`） */
  op?: 'ge' | 'le' | 'eq'
  /** number：阈值（非有限值=不约束） */
  num?: number
  /** date：区间左界 YYYY-MM-DD（空=不限） */
  from?: string
  /** date：区间右界 YYYY-MM-DD（空=不限） */
  to?: string
  /** enum：允许值集合（空集=不约束） */
  set?: string[]
  /** 空值口径（缺省 any）：只看空 / 只看有值 */
  empty?: 'any' | 'empty' | 'nonEmpty'
}

/** 列键 → 列筛值（缺键=不筛） */
export type ColFilters = Record<string, ColFilterValue>

export interface GanttFilter {
  kind: GanttFilterKind
  /** 名称包含文本（trim 后空串=不过滤；不区分大小写） */
  text: string
  /** 列级筛选（W5，可缺省；缺省/空对象=行为与拆分前一致） */
  colFilters?: ColFilters
}

export const GANTT_FILTER_DEFAULT: GanttFilter = { kind: 'all', text: '' }

/** 日期列口径需要的上下文（开工日 + 日历；缺 startDate 时日期列筛跳过） */
export interface GanttFilterCtx {
  startDate?: string
  calendar?: SchedCalendar
}

/** 列键 → 列筛编辑器类型（表头筛选按钮按此选编辑器；未登记列=不可筛） */
export const COL_FILTER_KIND: Record<string, ColFilterKind> = {
  name: 'text',
  dur: 'number',
  progress: 'number',
  start: 'date',
  finish: 'date',
  ls: 'date',
  lf: 'date',
  tf: 'number',
  ff: 'number',
  mode: 'enum',
  marker: 'enum',
  text1: 'text',
  text2: 'text',
  text3: 'text',
  num1: 'number',
  num2: 'number',
}

/** 枚举列的候选值（表头筛选器给勾选项；与取值口径同源） */
export const COL_FILTER_ENUM_OPTIONS: Record<string, { value: string; label: string }[]> = {
  mode: [
    { value: 'auto', label: '自动' },
    { value: 'manual', label: '手动' },
  ],
  marker: [
    { value: 'normal', label: '普通' },
    { value: 'milestone', label: '里程碑' },
    { value: 'group', label: '汇总' },
  ],
}

function leafVisible(t: SchedTask, f: GanttFilter, cpm: CpmResult): boolean {
  if (f.kind === 'critical') return cpm.rows[t.id]?.critical === true
  if (f.kind === 'manual') return t.mode === 'manual'
  if (f.kind === 'milestone') return t.isMilestone === true
  return true
}

/** 列筛是否生效（表头按钮激活态/计数共用口径） */
export function isColFilterActive(v: ColFilterValue | undefined): boolean {
  if (!v) return false
  if (v.empty !== undefined && v.empty !== 'any') return true
  if (v.kind === 'text') return (v.text ?? '').trim() !== ''
  if (v.kind === 'number') return typeof v.num === 'number' && Number.isFinite(v.num)
  if (v.kind === 'date') return (v.from ?? '') !== '' || (v.to ?? '') !== ''
  if (v.kind === 'enum') return (v.set?.length ?? 0) > 0
  return false
}

/** 生效列筛条数（表头/工具栏徽标） */
export function activeColFilterCount(map: ColFilters | undefined): number {
  if (!map) return 0
  return Object.values(map).filter((v) => isColFilterActive(v)).length
}

/** 写入一列筛选（不可变；失效条件=删键，不留空壳） */
export function setColFilter(map: ColFilters | undefined, key: string, v: ColFilterValue): ColFilters {
  const next: ColFilters = { ...(map ?? {}) }
  if (isColFilterActive(v)) next[key] = v
  else delete next[key]
  return next
}

/** 清除一列筛选（不可变） */
export function clearColFilter(map: ColFilters | undefined, key: string): ColFilters {
  const next: ColFilters = { ...(map ?? {}) }
  delete next[key]
  return next
}

/** 列键 → 单元格可比值（数值列=number、日期列=ISO 字符串、枚举=机器码、其余=文本；取不到=null） */
export function colValueOf(
  t: SchedTask,
  key: string,
  cpm: CpmResult,
  ctx?: GanttFilterCtx,
): string | number | null {
  const row = cpm.rows[t.id]
  switch (key) {
    case 'name':
      return t.name
    case 'dur':
      return t.duration
    case 'progress':
      return t.progress
    case 'start':
      return isoCol(ctx, row?.es)
    case 'finish':
      return isoCol(ctx, row?.ef)
    case 'ls':
      return isoCol(ctx, row?.ls)
    case 'lf':
      return isoCol(ctx, row?.lf)
    case 'tf':
      return row ? row.tf : null
    case 'ff':
      return row ? row.ff : null
    case 'mode':
      return t.mode ?? 'auto'
    case 'marker':
      return t.isMilestone ? 'milestone' : t.level === 0 ? 'group' : 'normal'
    default: {
      const v = t.custom?.[key]
      if (typeof v === 'number') return Number.isFinite(v) ? v : null
      if (typeof v === 'string') return v
      return null
    }
  }
}

/** 工作日序号 → ISO 日期（无开工日/无行=null：日期列筛随之跳过，不误杀） */
function isoCol(ctx: GanttFilterCtx | undefined, wd: number | undefined): string | null {
  if (!ctx?.startDate || wd === undefined || !Number.isFinite(wd)) return null
  return isoOf(wdToDate(ctx.startDate, wd, ctx.calendar))
}

/** 单列条件判定（fail-open：条件不足以筛选=通过） */
function passColFilter(
  t: SchedTask,
  key: string,
  v: ColFilterValue,
  cpm: CpmResult,
  ctx?: GanttFilterCtx,
): boolean {
  const raw = colValueOf(t, key, cpm, ctx)
  const isEmpty = raw === null || (typeof raw === 'string' && raw.trim() === '')
  // 空值口径是独立闸门（与下面按类型的条件 AND）
  if (v.empty === 'empty' && !isEmpty) return false
  if (v.empty === 'nonEmpty' && isEmpty) return false
  if (v.kind === 'text') {
    const needle = (v.text ?? '').trim().toLowerCase()
    if (!needle) return true
    return typeof raw === 'string' && raw.toLowerCase().includes(needle)
  }
  if (v.kind === 'number') {
    if (typeof v.num !== 'number' || !Number.isFinite(v.num)) return true
    if (typeof raw !== 'number') return false
    const op = v.op ?? 'ge'
    if (op === 'le') return raw <= v.num
    if (op === 'eq') return raw === v.num
    return raw >= v.num
  }
  if (v.kind === 'date') {
    if (typeof raw !== 'string' || raw === '') return false
    if ((v.from ?? '') !== '' && raw < v.from!) return false
    if ((v.to ?? '') !== '' && raw > v.to!) return false
    return true
  }
  if (v.kind === 'enum') {
    const set = v.set ?? []
    if (set.length === 0) return true
    return typeof raw === 'string' && set.includes(raw)
  }
  return true
}

/**
 * 行筛选两遍法：
 * 1. 强制集=分组名命中文本的分组及其全部子孙（「搜支名看整支」的树搜索直觉）；
 * 2. 主遍历：叶=（模条件+文本+列筛）命中或被强制；分组=任一直接子孙可见或被强制。
 * 返回可见行原索引（行号列仍按全表序号显示）。
 */
export function filterGanttRows(
  tasks: SchedTask[],
  f: GanttFilter,
  cpm: CpmResult,
  ctx?: GanttFilterCtx,
): number[] {
  const needle = f.text.trim().toLowerCase()
  // 日期列筛在没有开工日上下文时无法换算 → 跳过（fail-open，不整表清空）
  const colPairs = Object.entries(f.colFilters ?? {}).filter(
    ([, v]) => isColFilterActive(v) && (v.kind !== 'date' || ctx?.startDate !== undefined),
  )
  const colOk = (t: SchedTask): boolean => colPairs.every(([k, v]) => passColFilter(t, k, v, cpm, ctx))
  const leafOk = (t: SchedTask): boolean =>
    leafVisible(t, f, cpm) && (!needle || t.name.toLowerCase().includes(needle))
  const forced = new Set<number>()
  /** 行可见性：列筛是硬约束，文本强制集不能绕过它 */
  const shown = (i: number): boolean => (forced.has(i) || leafOk(tasks[i])) && colOk(tasks[i])
  for (let i = 0; i < tasks.length; i++) {
    const t = tasks[i]
    if (t.level === 0 && needle && t.name.toLowerCase().includes(needle)) {
      for (const id of descendantIds(tasks, t.id)) {
        const j = tasks.findIndex((x) => x.id === id)
        if (j > i) forced.add(j)
      }
    }
  }
  const out: number[] = []
  for (let i = 0; i < tasks.length; i++) {
    const t = tasks[i]
    if (t.level === 0) {
      const kids = descendantIds(tasks, t.id)
      let anyKid = false
      for (const id of kids) {
        const j = tasks.findIndex((x) => x.id === id)
        if (j > i && tasks[j].level > 0 && shown(j)) {
          anyKid = true
          break
        }
      }
      if (anyKid) out.push(i)
      continue
    }
    if (shown(i)) out.push(i)
  }
  return out
}
