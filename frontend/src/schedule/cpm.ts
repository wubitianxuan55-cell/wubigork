/**
 * cpm.ts — 关键路径法（CPM）计算引擎（纯函数，v4.110.0 刀1 / v4.111.0 刀2）
 *
 * 输入任务表 + 搭接关系（FS/SS/FF/SF + 时距），输出正推（ES/EF）、
 * 逆推（LS/LF）、总时差（TF）、自由时差（FF）与关键工作标记。
 * 四种搭接均以 from → to 为拓扑方向（计算 from 先于 to）；
 * 检测到循环依赖时 fail-closed 返回 ok=false（UI 提示，不静默出错误条）。
 *
 * 手动/自动双模式（对齐 Project 任务模式）：
 *  - auto：按搭接逻辑排程（标准 CPM）；
 *  - manual：正推忽略其入边、ES 锁定为 manualStart（不动）；
 *    逆推不回传约束（前置任务不因手动任务收 LF）；自身 TF/FF=0、
 *    不标关键；其后继仍以 manual 的 EF 为正向约束。
 */
import type { CpmResult, SchedLink, SchedTask, TaskCpm } from './types'

/** 有效工期：里程碑为 0，其余取非负整数 */
function effDur(t: SchedTask): number {
  return t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
}

/** 单条搭接对后续任务 ES 的正推下界 */
function forwardBound(link: SchedLink, from: TaskCpm, durTo: number): number {
  switch (link.type) {
    case 'FS': return from.ef + link.lag
    case 'SS': return from.es + link.lag
    case 'FF': return from.ef + link.lag - durTo
    case 'SF': return from.es + link.lag - durTo
  }
}

/** 单条搭接对前置任务 LF 的逆推上界（durFrom/durTo 分别为前置/后续任务工期） */
function backwardBound(link: SchedLink, to: TaskCpm, durFrom: number, durTo: number): number {
  switch (link.type) {
    case 'FS': return to.lf - durTo - link.lag
    case 'SS': return to.ls - link.lag + durFrom
    case 'FF': return to.lf - link.lag
    case 'SF': return to.lf - link.lag + durFrom
  }
}

/** 单条搭接对前置任务自由时差的贡献（相对 to 的最早时间；FS 按完成对齐再减 EF_from） */
function freeFloatPart(link: SchedLink, to: TaskCpm, efFrom: number): number {
  switch (link.type) {
    case 'FS': return to.es - link.lag - efFrom
    case 'SS': return to.es - link.lag
    case 'FF': return to.ef - link.lag - efFrom
    case 'SF': return to.ef - link.lag
  }
}

/** Kahn 拓扑排序；返回 null 表示存在环（residual 为环上任务） */
function topoOrder(ids: string[], out: Map<string, string[]>, indeg: Map<string, number>): { order: string[]; cycle: string[] } | null {
  const deg = new Map(indeg)
  const queue = ids.filter((id) => (deg.get(id) ?? 0) === 0)
  const order: string[] = []
  while (queue.length > 0) {
    const id = queue.shift()!
    order.push(id)
    for (const nxt of out.get(id) ?? []) {
      const d = (deg.get(nxt) ?? 0) - 1
      deg.set(nxt, d)
      if (d === 0) queue.push(nxt)
    }
  }
  if (order.length !== ids.length) {
    return { order, cycle: ids.filter((id) => !order.includes(id)) }
  }
  return { order, cycle: [] }
}

/**
 * CPM 主计算。任务表全量参与（分组行 duration=0、无搭接时退化为零工期点，
 * 汇总条由视图层按子项滚动，不经引擎）。
 */
export function computeCpm(tasks: SchedTask[], links: SchedLink[]): CpmResult {
  const byId = new Map(tasks.map((t) => [t.id, t]))
  const ids = tasks.map((t) => t.id)
  const valid = links.filter((l) => l.from !== l.to && byId.has(l.from) && byId.has(l.to))

  const out = new Map<string, string[]>()
  const outLinks = new Map<string, SchedLink[]>()
  const inLinks = new Map<string, SchedLink[]>()
  const indeg = new Map<string, number>()
  for (const id of ids) {
    out.set(id, []); outLinks.set(id, []); inLinks.set(id, []); indeg.set(id, 0)
  }
  for (const l of valid) {
    out.get(l.from)!.push(l.to)
    outLinks.get(l.from)!.push(l)
    inLinks.get(l.to)!.push(l)
    indeg.set(l.to, (indeg.get(l.to) ?? 0) + 1)
  }

  const rows: Record<string, TaskCpm> = {}
  for (const t of tasks) {
    rows[t.id] = { es: 0, ef: 0, ls: 0, lf: 0, tf: 0, ff: 0, critical: false }
  }
  if (ids.length === 0) return { ok: true, rows, duration: 0 }

  const topo = topoOrder(ids, out, indeg)
  if (!topo) return { ok: false, rows, duration: 0 }
  if (topo.cycle.length > 0) {
    const names = topo.cycle.map((id) => byId.get(id)?.name ?? id)
    return { ok: false, rows, duration: 0, error: `存在循环依赖：${names.join(' → ')}`, cycle: topo.cycle }
  }

  // 正推：ES = max(各搭接下界)，EF = ES + 工期；manual 任务锁定开始、忽略入边
  for (const id of topo.order) {
    const t = byId.get(id)!
    const dur = effDur(t)
    const row = rows[id]
    if (t.mode === 'manual') {
      const fixed = Math.max(0, Math.round(t.manualStart ?? 0))
      row.es = fixed
      row.ef = fixed + dur
      continue
    }
    let es = 0
    for (const l of inLinks.get(id) ?? []) {
      es = Math.max(es, forwardBound(l, rows[l.from], dur))
    }
    row.es = es
    row.ef = es + dur
  }
  const duration = Math.max(0, ...tasks.map((t) => rows[t.id].ef))

  // 逆推：LF = min(各搭接上界 / 总工期)；manual 任务 LF=EF（锁定，不回传约束）
  for (let i = topo.order.length - 1; i >= 0; i--) {
    const id = topo.order[i]
    const t = byId.get(id)!
    const dur = effDur(t)
    const row = rows[id]
    if (t.mode === 'manual') {
      row.lf = row.ef
      row.ls = row.es
      continue
    }
    let lf = duration
    for (const l of outLinks.get(id) ?? []) {
      if (byId.get(l.to)!.mode === 'manual') continue // 手动后继不约束前置
      lf = Math.min(lf, backwardBound(l, rows[l.to], dur, effDur(byId.get(l.to)!)))
    }
    row.lf = lf
    row.ls = lf - dur
  }

  // 时差与关键标记（manual 任务时差为 0 但不标关键）
  for (const t of tasks) {
    const row = rows[t.id]
    row.tf = row.ls - row.es
    let ff = duration - row.ef
    for (const l of outLinks.get(t.id) ?? []) {
      if (byId.get(l.to)!.mode === 'manual') continue
      ff = Math.min(ff, freeFloatPart(l, rows[l.to], row.ef))
    }
    row.ff = ff
    row.critical = t.mode !== 'manual' && row.tf === 0
  }

  return { ok: true, rows, duration }
}
