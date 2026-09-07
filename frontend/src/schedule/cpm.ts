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
import type { CpmResult, SchedCalendar, SchedLink, SchedTask, TaskCpm } from './types'
import { cdLatestStart, cdToEf, normalizeCalendar } from './calendar'

/** 有效工期：里程碑为 0，其余取非负整数（cost.ts 成本工期同口径，单一来源） */
export function effDur(t: SchedTask): number {
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

/**
 * 单条搭接对前置任务自由时差的贡献（搭接口径 LAG_i-j，v4.129 刀G 修 E1）：
 * FF/FS 锚点=前置 EF；SS/SF 锚点=前置 ES（旧版漏减 ES_from 致 SS/SF 下 FF 虚高，
 * 违反「TF=0 ⇒ FF=0」定理）。导出供镜像测试直接钉死四型公式。
 */
export function freeFloatPart(link: SchedLink, to: TaskCpm, efFrom: number, esFrom: number): number {
  switch (link.type) {
    case 'FS': return to.es - link.lag - efFrom
    case 'SS': return to.es - link.lag - esFrom
    case 'FF': return to.ef - link.lag - efFrom
    case 'SF': return to.ef - link.lag - esFrom
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
 *
 * v4.129 刀G（G3）：opts.planFinish=计划工期锚点（工作日边界，来自目标竣工换算）。
 * 规程口径：计划工期 Tp 小于计算工期 Tc 时逆推从 Tp 起——绑定链出现**负时差**，
 * 关键工作=总时差最小者（不再恒为 TF=0）；planFinish 缺省/≥Tc 时与旧口径完全一致。
 */
export interface CpmOptions {
  /** 计划工期锚点（工作日边界索引；null/undefined=无目标竣工，按计算工期逆推） */
  planFinish?: number | null
  /** 日历换算上下文（v4.150 双工期刀1）：存在 cd（日历天）任务时的换算锚点。
   * 任务表无 cd 任务时本组字段不被读取（快路径，行为与缺省逐位一致）；
   * 有 cd 任务而 startDate 缺失 → fail-closed（日历缺省回落周一~五，与全部
   * wdToDate 读点同口径）。 */
  calendar?: SchedCalendar
  startDate?: string
}

/** 任务是否按日历天（cd）工期（v4.150 双工期刀1） */
export function isCd(t: Pick<SchedTask, 'durationUnit'>): boolean {
  return t.durationUnit === 'cd'
}

export function computeCpm(tasks: SchedTask[], links: SchedLink[], opts?: CpmOptions): CpmResult {
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

  // 双工期口径（v4.150 刀1）：存在 cd（日历天）任务时启用日历换算分支；
  // 无 cd 任务走快路径（下方分支全部短路，结果与旧口径逐位一致）。
  // fail-closed：缺开工日期无法换算（日历缺省回落周一~五，与全部 wdToDate
  // 读点同口径）；cd 涉非 FS 搭接会引入对 dur 的不动点迭代，v1 引擎拒绝。
  const hasCd = tasks.some(isCd)
  let cal: SchedCalendar | undefined
  let startISO = ''
  if (hasCd) {
    if (!opts?.startDate) {
      return { ok: false, rows, duration: 0, error: '计划缺开工日期，无法计算日历天（cd）任务' }
    }
    cal = normalizeCalendar(opts.calendar)
    startISO = opts.startDate
    for (const l of valid) {
      if ((isCd(byId.get(l.from)!) || isCd(byId.get(l.to)!)) && l.type !== 'FS') {
        return {
          ok: false, rows, duration: 0,
          error: `日历天（cd）任务仅支持 FS 搭接（${byId.get(l.from)!.name} → ${byId.get(l.to)!.name} 为 ${l.type}）`,
        }
      }
    }
  }

  // 正推：ES = max(各搭接下界)，EF = ES + 工期；manual 任务锁定开始、忽略入边
  for (const id of topo.order) {
    const t = byId.get(id)!
    const dur = effDur(t)
    const row = rows[id]
    if (t.mode === 'manual') {
      const fixed = Math.max(0, Math.round(t.manualStart ?? 0))
      row.es = fixed
      row.ef = hasCd && isCd(t) ? cdToEf(startISO, fixed, dur, cal) : fixed + dur
      continue
    }
    let es = 0
    for (const l of inLinks.get(id) ?? []) {
      es = Math.max(es, forwardBound(l, rows[l.from], dur))
    }
    row.es = es
    row.ef = hasCd && isCd(t) ? cdToEf(startISO, es, dur, cal) : es + dur
  }
  const duration = Math.max(0, ...tasks.map((t) => rows[t.id].ef))

  // 逆推锚点（G3）：计划工期 < 计算工期时从计划工期逆推（负时差诚实呈现）；
  // 其余（无目标竣工/压线/富余）从计算工期逆推，与旧口径零差异。
  const planFinish = opts?.planFinish
  const anchor = planFinish != null && planFinish < duration ? planFinish : duration

  // 逆推：LF = min(各搭接上界 / 计划工期)；manual 任务 LF=EF（锁定，不回传约束）
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
    let lf = anchor
    for (const l of outLinks.get(id) ?? []) {
      if (byId.get(l.to)!.mode === 'manual') continue // 手动后继不约束前置
      // FS 上界=后继最迟开始−lag：wd 后继 to.ls=to.lf−durTo 与旧式逐位一致；
      // cd 后继的 ls 已按日历换算（≠lf−dur），必须用 to.ls 才诚实。
      if (hasCd && l.type === 'FS') {
        lf = Math.min(lf, rows[l.to].ls - l.lag)
      } else {
        lf = Math.min(lf, backwardBound(l, rows[l.to], dur, effDur(byId.get(l.to)!)))
      }
    }
    row.lf = lf
    row.ls = hasCd && isCd(t) ? cdLatestStart(startISO, lf, dur, cal) : lf - dur
  }

  // 时差与关键标记（manual 任务不标关键）：关键工作=总时差最小者
  // （规程口径：Tp=Tc 时最小值为 0，Tp<Tc 时为负——不再恒为 TF===0）。
  // 仅在 deadline 真正收紧（anchor<duration）时启用 minTF：manual 驱动总工期、
  // 非手动链无 TF=0 的场景维持经典口径（避免把带浮动的任务误标关键）。
  let minTf = Number.POSITIVE_INFINITY
  for (const t of tasks) {
    if (t.mode === 'manual') continue
    minTf = Math.min(minTf, rows[t.id].ls - rows[t.id].es)
  }
  if (!Number.isFinite(minTf)) minTf = 0
  const critTf = anchor < duration ? minTf : 0
  for (const t of tasks) {
    const row = rows[t.id]
    row.tf = row.ls - row.es
    let ff = anchor - row.ef
    for (const l of outLinks.get(t.id) ?? []) {
      if (byId.get(l.to)!.mode === 'manual') continue
      ff = Math.min(ff, freeFloatPart(l, rows[l.to], row.ef, row.es))
    }
    row.ff = ff
    row.critical = t.mode !== 'manual' && row.tf === critTf
  }

  return { ok: true, rows, duration }
}
