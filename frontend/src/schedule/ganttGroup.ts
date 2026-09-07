/**
 * ganttGroup.ts — 横道行排序与多级分组（纯函数）
 *
 * 对标 MS Project 的排序/分组：把「用户排序 + 多级分组 + 折叠」合成为新的
 * 可见行序列，视图只按该序列渲染。蒸馏自 ProjectLibre 的两个机制：
 * 1. 可见行序列唯一状态源——本函数不改 tasks 原数组，task 行只携带原索引
 *    idx，视图按 idx 回查任务数据（行号列、条形、依赖线锚点共用）；
 * 2. 合成组头行机制——分组组头不是 tasks 里的行，而是按可见叶聚合出的合成
 *    行（key/label/count/汇总条 min-max），折叠=命中 key 的组头裁掉子树。
 *
 * 与 filterGanttRows 衔接：visibleIdx 传筛选后的可见叶原索引子集（只含叶），
 * 组头统计只计可见叶、空分支不产出组头；不分组模式传 visibleIdx 时，
 * 无可见直接子叶的原分组行同样不保留（与筛选「无可见子孙的分组行不保留」同口径）。
 */
import type { CpmResult, SchedTask, TaskCpm } from './types'

export type GanttSortField = 'none' | 'name' | 'duration' | 'start' | 'progress' | 'tf'

/** 排序参数：field 'none'=不排序（等于原顺序）；desc 反转主键、同值仍保原相对顺序 */
export interface GanttSort {
  field: GanttSortField
  dir: 'asc' | 'desc'
}

/** 分组字段：关键性（CPM）/ 排程模式 / 里程碑 */
export type GanttGroupField = 'critical' | 'mode' | 'milestone'

export interface GanttGroupOptions {
  /** 缺省=原顺序（field 'none' 同义） */
  sort?: GanttSort
  /** 缺省=[]=不分组；多级分组=数组顺序即层级；启用后原 WBS 分组行不输出（分组接管层级） */
  groupBy?: GanttGroupField[]
  /** 已折叠组头 key 集合：命中组头的全部子孙（子组头+叶）不输出，组头自身保留；缺省不折叠 */
  collapsed?: Set<string>
  /** 筛选后的可见叶原索引子集（只含叶，不含分组行）；缺省=全部叶 */
  visibleIdx?: number[]
}

/** 输出行：task=原数组第 idx 个任务（level=组深+原 level）；group=合成组头 */
export type GanttRow =
  | { kind: 'task'; idx: number; level: number }
  | {
      kind: 'group'
      /** 组头唯一键：不分组=`wbs:原索引`；分组=`父key/字段:值`（根父 key 为空串，字段用英文 id） */
      key: string
      /** 不分组=分组行任务名；分组=`字段中文名: 值` */
      label: string
      level: number
      /** 可见直接子叶数（不分组）/ 可见叶数（分组） */
      count: number
      /** 可见叶的 min(es) / max(ef)：缺 CPM 行的叶不计入，全缺则 0 */
      minEs: number
      maxEf: number
    }

/** 分组字段中文名（组头 label 前缀） */
const GROUP_FIELD_ZH: Record<GanttGroupField, string> = { critical: '关键', mode: '模式', milestone: '里程碑' }

/** WBS 编码：分组 n、叶 n.k（与 mspdi 导出同构；横道表格行号旁展示 + 检查器取码共用） */
export function wbsOf(tasks: SchedTask[]): string[] {
  const out: string[] = []
  let g = 0
  let k = 0
  for (const t of tasks) {
    if (t.level === 0) { g++; k = 0; out.push(`${g}`) } else { k++; out.push(`${g}.${k}`) }
  }
  return out
}

/** 取 CPM 行：ok=false 一律视为无数据（关键=false、es/ef/tf 按 0），缺行同 */
function cpmRowOf(cpm: CpmResult, id: string): TaskCpm | undefined {
  return cpm.ok ? cpm.rows[id] : undefined
}

/** 分组值：critical 取 CPM 关键（缺行=false）；mode 缺省 auto；milestone 取 isMilestone */
function groupValueOf(field: GanttGroupField, t: SchedTask, cpm: CpmResult): string {
  if (field === 'critical') return cpmRowOf(cpm, t.id)?.critical ? '是' : '否'
  if (field === 'mode') return t.mode === 'manual' ? '手动' : '自动'
  return t.isMilestone ? '是' : '否'
}

/** 可见叶的 min(es)/max(ef)：缺 CPM 行的叶不计入，全缺则 0（ok=false 时恒 0） */
function minMaxOf(cpm: CpmResult, tasks: SchedTask[], idxs: number[]): { minEs: number; maxEf: number } {
  let minEs = Number.POSITIVE_INFINITY
  let maxEf = Number.NEGATIVE_INFINITY
  for (const i of idxs) {
    const row = cpmRowOf(cpm, tasks[i].id)
    if (!row) continue
    if (row.es < minEs) minEs = row.es
    if (row.ef > maxEf) maxEf = row.ef
  }
  return minEs === Number.POSITIVE_INFINITY ? { minEs: 0, maxEf: 0 } : { minEs, maxEf }
}

/** 数值排序键：duration/progress 取任务字段；start/tf 取 CPM（缺行=0） */
function sortNumKey(field: GanttSortField, t: SchedTask, cpm: CpmResult): number {
  switch (field) {
    case 'duration':
      return t.duration
    case 'progress':
      return t.progress
    case 'start':
      return cpmRowOf(cpm, t.id)?.es ?? 0
    case 'tf':
      return cpmRowOf(cpm, t.id)?.tf ?? 0
    default:
      return 0
  }
}

/**
 * 排序比较器（原索引入参）：主键定序、同值用原索引做次序键（稳定）；desc 只
 * 反转主键不反转次序键。field 'none' 返回 null=不排序。
 */
function comparerOf(
  tasks: SchedTask[],
  cpm: CpmResult,
  sort: GanttSort,
): ((a: number, b: number) => number) | null {
  if (sort.field === 'none') return null
  const desc = sort.dir === 'desc'
  return (a, b) => {
    let primary: number
    if (sort.field === 'name') primary = tasks[a].name.localeCompare(tasks[b].name, 'zh')
    else {
      const va = sortNumKey(sort.field, tasks[a], cpm)
      const vb = sortNumKey(sort.field, tasks[b], cpm)
      primary = va < vb ? -1 : va > vb ? 1 : 0
    }
    if (primary !== 0) return desc ? -primary : primary
    return a - b
  }
}

/**
 * 逐级产出合成组头：按当前字段把已排序的叶序列稳定切片（分支按首次出现顺序
 * 产出），每支先落组头（命中折叠=只落组头），再递归下一字段或落叶。
 * 组头 level=组深度（0 起）；叶 level=父组头组深+原 level。
 */
function emitGroups(
  out: GanttRow[],
  tasks: SchedTask[],
  cpm: CpmResult,
  leaves: number[],
  fields: GanttGroupField[],
  depth: number,
  parentKey: string,
  collapsed: Set<string> | undefined,
): void {
  const field = fields[0]
  const branchOf = new Map<string, number[]>()
  const order: string[] = []
  for (const i of leaves) {
    const v = groupValueOf(field, tasks[i], cpm)
    const arr = branchOf.get(v)
    if (arr) arr.push(i)
    else {
      branchOf.set(v, [i])
      order.push(v)
    }
  }
  for (const v of order) {
    const idxs = branchOf.get(v)
    if (!idxs) continue
    const key = `${parentKey}/${field}:${v}`
    const stats = minMaxOf(cpm, tasks, idxs)
    out.push({
      kind: 'group',
      key,
      label: `${GROUP_FIELD_ZH[field]}: ${v}`,
      level: depth,
      count: idxs.length,
      minEs: stats.minEs,
      maxEf: stats.maxEf,
    })
    if (collapsed?.has(key)) continue
    const rest = fields.slice(1)
    if (rest.length > 0) emitGroups(out, tasks, cpm, idxs, rest, depth + 1, key, collapsed)
    else for (const i of idxs) out.push({ kind: 'task', idx: i, level: depth + tasks[i].level })
  }
}

/**
 * 不分组模式：保留原数组行序与原 WBS 分组行（合成 kind=group：key=`wbs:原索引`、
 * label=任务名、统计=可见直接子叶）。排序施加在每个分组行的直接子叶上（组头
 * 位置不动，MS Project 同口径）；传入 visibleIdx（hasFilter）时无可见直接子叶
 * 的分组行不保留；折叠命中 wbs key 只裁该组直接子叶。
 */
function identityRows(
  tasks: SchedTask[],
  cpm: CpmResult,
  cmp: ((a: number, b: number) => number) | null,
  isVisible: (i: number) => boolean,
  hasFilter: boolean,
  collapsed: Set<string> | undefined,
): GanttRow[] {
  const out: GanttRow[] = []
  let pending: number[] = [] // 首个分组行之前的顶层直出叶（遇分组行统一落场）
  let hideLeaves = false // 折叠组头的子叶到下一分组行前一律不输出
  const flushPending = (): void => {
    if (cmp) pending.sort(cmp)
    for (const i of pending) out.push({ kind: 'task', idx: i, level: tasks[i].level })
    pending = []
  }
  for (let i = 0; i < tasks.length; i++) {
    const t = tasks[i]
    if (t.level === 0) {
      flushPending()
      hideLeaves = false
      const kids: number[] = []
      for (let j = i + 1; j < tasks.length && tasks[j].level > t.level; j++) {
        if (tasks[j].level === t.level + 1 && isVisible(j)) kids.push(j)
      }
      if (hasFilter && kids.length === 0) continue
      const stats = minMaxOf(cpm, tasks, kids)
      const key = `wbs:${i}`
      out.push({
        kind: 'group',
        key,
        label: t.name,
        level: t.level,
        count: kids.length,
        minEs: stats.minEs,
        maxEf: stats.maxEf,
      })
      hideLeaves = collapsed?.has(key) === true
      if (hideLeaves) continue
      if (cmp) kids.sort(cmp)
      for (const j of kids) out.push({ kind: 'task', idx: j, level: tasks[j].level })
      hideLeaves = true // 该组子叶已落场：组范围内不再进 pending（防重复输出）
      continue
    }
    if (!isVisible(i) || hideLeaves) continue
    pending.push(i)
  }
  flushPending()
  return out
}

/**
 * 合成横道可见行序列（唯一状态源）：排序/分组/折叠/筛选全部只体现在返回序列
 * 上，不改 tasks、不抛错（cpm.ok=false 时 CPM 值一律按 0/否 参与统计与排序）。
 *
 * 三条正交机制：
 * - 排序：施加在叶序列上——不分组=各 WBS 分组行内直接子叶重排（组头不动）；
 *   分组=全局排一次序再稳定切片（组内顺序=全局排序的分片，同值保原相对顺序）。
 * - 分组：groupBy 逐字段合成组头，key=`父key/字段:值`（根父 key 为空串），
 *   空分支不产出；原 level 0 的 WBS 行不输出（分组接管层级），子叶照常参与。
 * - 折叠：collapsed 命中组头 key → 该组头保留、全部子孙不输出。
 */
export function buildGanttRows(tasks: SchedTask[], cpm: CpmResult, opts?: GanttGroupOptions): GanttRow[] {
  const collapsed = opts?.collapsed
  const visSet = opts?.visibleIdx ? new Set(opts.visibleIdx) : null
  const isVisible = (i: number): boolean => !visSet || visSet.has(i)
  const cmp = opts?.sort ? comparerOf(tasks, cpm, opts.sort) : null
  const groupBy = opts?.groupBy ?? []
  if (groupBy.length > 0) {
    // 可见叶（原顺序）全局排一次序——组内顺序=全局排序的稳定分片
    const leaves: number[] = []
    for (let i = 0; i < tasks.length; i++) {
      if (tasks[i].level !== 0 && isVisible(i)) leaves.push(i)
    }
    if (cmp) leaves.sort(cmp)
    const out: GanttRow[] = []
    emitGroups(out, tasks, cpm, leaves, groupBy, 0, '', collapsed)
    return out
  }
  return identityRows(tasks, cpm, cmp, isVisible, visSet !== null, collapsed)
}
