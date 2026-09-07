/**
 * schedule/aoa.ts — 双代号网络图（AOA，Activity-On-Arrow）转换引擎（纯函数）
 *
 * PDM 任务表 + 搭接 → 箭线图：节点=事件（圆圈编号 i<j），箭线=工作，虚箭线=虚工作。
 * 转换规则（对齐教材口径）：
 *  - 无前置任务 → 汇入唯一起点事件 START；
 *  - 唯一 FS(lag=0) 前置且该前置完成事件无「进入箭线」（无 FF/SF 搭接指入）→
 *    直接共用完成事件（ textbook 链式画法，免虚工作）；
 *  - 唯一 SS(lag=0) 前置且该前置开始事件无进入箭线 → 共用开始事件；
 *  - 其余情况每任务独立开始/完成事件，按搭接类型补虚箭线（FS/SS/FF/SF）；
 *  - 无后续任务 → 虚箭线汇入唯一终点事件 END（一始一终规则）；
 *  - 虚箭线去重（同 from+to 只留一条）。
 * 事件时间沿箭线网络正逆推（虚工作工期=0 或时距），关键线路 = 时差为 0 的箭线
 * （虚工作也可能是关键线路的一段，红色虚线显示）。
 */
import type { SchedLink, SchedTask } from './types'
import { layerByTime } from './layout'

export interface AoaNode {
  id: string
  /** 节点编号（1 起，拓扑序，保证所有箭线 i<j） */
  num: number
  /** 事件最早/最迟时间（天） */
  es: number
  ls: number
  x: number
  y: number
  /**
   * 锚点键（v4.123 AOA 刀1）：事件业务身份，跨拓扑变更稳定（手动布局 pin 的匹配键）。
   * START→"S"、END→"T"；其余=全部成员键（end:<taskId>|start:<taskId>）的字典序最小者
   * （end: < start: 保证合并事件代表=先构造者）。推导规则单测钉死（aoaLayout.test.ts）。
   */
  anchor: string
}

export interface AoaEdge {
  id: string
  from: string
  to: string
  /** 'task' = 实工作箭线；'dummy' = 虚工作（虚线） */
  kind: 'task' | 'dummy'
  /** kind='task' 时对应的任务 id */
  taskId?: string
  /** 箭线时长（虚工作=0 或时距） */
  dur: number
  critical: boolean
  /** 箭线标注：实工作=工期，虚工作=时距（0 不标） */
  label: string
}

export interface AoaGraph {
  ok: boolean
  error?: string
  nodes: AoaNode[]
  edges: AoaEdge[]
  /** taskId → 实工作箭线 id */
  taskEdge: Record<string, string>
}

/** 布局常量（视图层按比例消费）；AOA_R=事件圆半径（渲染/文字避让共用） */
export const AOA_COL_W = 110
export const AOA_ROW_H = 74
export const AOA_MARGIN = 40
export const AOA_R = 16

function effDur(t: SchedTask): number {
  return t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
}

export function buildAoa(tasks: SchedTask[], links: SchedLink[], opts?: { planFinish?: number | null }): AoaGraph {
  if (tasks.length === 0) return { ok: true, nodes: [], edges: [], taskEdge: {} }
  const byId = new Map(tasks.map((t) => [t.id, t]))
  const valid = links.filter((l) => l.from !== l.to && byId.has(l.from) && byId.has(l.to))

  // 搭接索引：按 to 分组（约束入度）、FF/SF 指入记录（判断完成事件是否被污染）
  const inOf = new Map<string, SchedLink[]>()
  const outOf = new Map<string, SchedLink[]>()
  for (const t of tasks) { inOf.set(t.id, []); outOf.set(t.id, []) }
  for (const l of valid) {
    inOf.get(l.to)!.push(l)
    outOf.get(l.from)!.push(l)
  }

  // PDM 层拓扑序（Kahn）；不完整 = 有环。事件分配必须按拓扑序进行，
  // 否则前向引用前置任务的 startOf/endOf 会取到 undefined。
  const indeg = new Map(tasks.map((t) => [t.id, inOf.get(t.id)!.length]))
  const queue = tasks.filter((t) => indeg.get(t.id) === 0).map((t) => t.id)
  const order: string[] = []
  while (queue.length > 0) {
    const id = queue.shift()!
    order.push(id)
    for (const l of outOf.get(id) ?? []) {
      const v = (indeg.get(l.to) ?? 0) - 1
      indeg.set(l.to, v)
      if (v === 0) queue.push(l.to)
    }
  }
  if (order.length !== tasks.length) {
    const inOrder = new Set(order)
    const names = tasks.filter((t) => !inOrder.has(t.id)).map((t) => t.name)
    return { ok: false, error: `存在循环依赖：${names.join(' → ')}`, nodes: [], edges: [], taskEdge: {} }
  }

  // ── 事件分配 ──────────────────────────────────────────────
  const START = 'S'
  const END = 'T'
  // 起点终点事件预先注册（nodeIds 的事件注册表即 nodeEs 键集）
  const nodeEs = new Map<string, number>([[START, 0], [END, 0]])
  const startOf = new Map<string, string>()
  const endOf = new Map<string, string>()
  let seq = 0
  const nextNode = (es: number): string => {
    const id = `n${seq++}`
    nodeEs.set(id, es)
    return id
  }
  // 完成事件无进入箭线的判断：无 FF/SF 搭接指入
  const finishClean = (p: string) => !inOf.get(p)!.some((l) => l.type === 'FF' || l.type === 'SF')
  // 开始事件无进入箭线的判断：无 FS/SS 搭接指入
  const startClean = (p: string) => !inOf.get(p)!.some((l) => l.type === 'FS' || l.type === 'SS')

  for (const id of order) {
    const t = byId.get(id)!
    const preds = inOf.get(t.id)!
    if (preds.length === 0) {
      startOf.set(t.id, START)
    } else if (
      preds.length === 1 && preds[0].type === 'FS' && preds[0].lag === 0 &&
      finishClean(preds[0].from)
    ) {
      startOf.set(t.id, endOf.get(preds[0].from)!)
    } else if (
      preds.length === 1 && preds[0].type === 'SS' && preds[0].lag === 0 &&
      startClean(preds[0].from)
    ) {
      startOf.set(t.id, startOf.get(preds[0].from)!)
    } else {
      startOf.set(t.id, nextNode(0))
    }
    endOf.set(t.id, nextNode(effDur(t)))
  }

  // ── 箭线生成 ──────────────────────────────────────────────
  interface RawEdge { from: string; to: string; dur: number; kind: 'task' | 'dummy'; taskId?: string; lag?: number }
  const raws: RawEdge[] = []
  const taskEdge: Record<string, string> = {}
  const dummySeen = new Set<string>()

  for (const id of order) {
    const t = byId.get(id)!
    const dur = effDur(t)
    raws.push({ from: startOf.get(t.id)!, to: endOf.get(t.id)!, dur, kind: 'task', taskId: t.id })
    const succs = outOf.get(t.id)!
    if (succs.length === 0) {
      const key = `${endOf.get(t.id)!}>${END}`
      if (!dummySeen.has(key)) { dummySeen.add(key); raws.push({ from: endOf.get(t.id)!, to: END, dur: 0, kind: 'dummy' }) }
    }
  }
  for (const l of valid) {
    const tailStart = startOf.get(l.from)!
    const tailEnd = endOf.get(l.from)!
    const headStart = startOf.get(l.to)!
    const headEnd = endOf.get(l.to)!
    let from: string | undefined
    let to: string | undefined
    switch (l.type) {
      case 'FS': if (tailEnd !== headStart) { from = tailEnd; to = headStart } break
      case 'SS': if (tailStart !== headStart) { from = tailStart; to = headStart } break
      case 'FF': if (tailEnd !== headEnd) { from = tailEnd; to = headEnd } break
      case 'SF': if (tailStart !== headEnd) { from = tailStart; to = headEnd } break
    }
    if (!from || !to) continue // 已被合并/同事件，无需箭线
    const key = `${from}>${to}`
    if (dummySeen.has(key)) continue
    dummySeen.add(key)
    raws.push({ from, to, dur: l.lag, kind: 'dummy', lag: l.lag })
  }

  // ── 事件时间正逆推（沿箭线网络）───────────────────────────
  const outEdges = new Map<string, RawEdge[]>()
  const inEdges = new Map<string, RawEdge[]>()
  const nodeIds = [...nodeEs.keys()]
  for (const id of nodeIds) { outEdges.set(id, []); inEdges.set(id, []) }
  for (const r of raws) {
    outEdges.get(r.from)!.push(r)
    inEdges.get(r.to)!.push(r)
  }

  // 拓扑序（事件网络天然无环：箭线方向即时间方向；防御性再检一次）
  const eventOrder: string[] = []
  {
    const d = new Map<string, number>(nodeIds.map((id) => [id, inEdges.get(id)!.length]))
    const q = nodeIds.filter((id) => d.get(id) === 0)
    while (q.length > 0) {
      const id = q.shift()!
      eventOrder.push(id)
      for (const e of outEdges.get(id) ?? []) {
        const v = (d.get(e.to) ?? 0) - 1
        d.set(e.to, v)
        if (v === 0) q.push(e.to)
      }
    }
    if (eventOrder.length !== nodeIds.length) {
      return { ok: false, error: '网络图存在异常回路', nodes: [], edges: [], taskEdge: {} }
    }
  }

  const nodeLs = new Map<string, number>()
  // 逆推锚点（v4.129 刀G G3）：目标竣工换算的计划工期 < 计算工期时按计划工期逆推
  const total = Math.max(0, ...nodeIds.map((id) => nodeEs.get(id)!))
  const planFinish = opts?.planFinish
  const finish = planFinish != null && planFinish < total ? planFinish : total
  for (const id of eventOrder) {
    let es = 0
    for (const e of inEdges.get(id) ?? []) es = Math.max(es, nodeEs.get(e.from)! + e.dur)
    nodeEs.set(id, es)
  }
  for (let i = eventOrder.length - 1; i >= 0; i--) {
    const id = eventOrder[i]
    let ls = finish
    for (const e of outEdges.get(id) ?? []) ls = Math.min(ls, nodeLs.get(e.to)! - e.dur)
    nodeLs.set(id, ls)
  }

  // ── 编号（拓扑序 1..N）与箭线关键标记 ─────────────────────
  const num = new Map(eventOrder.map((id, i) => [id, i + 1]))
  // 关键箭线=浮时最小（规程口径：Tp<Tc 时最小浮时为负，不再恒为 0）
  const floats = raws.map((r) => nodeLs.get(r.to)! - r.dur - nodeEs.get(r.from)!)
  const minFloat = Math.min(0, ...floats)
  const edges: AoaEdge[] = raws.map((r, i) => {
    return {
      id: `e${i}`,
      from: r.from,
      to: r.to,
      kind: r.kind,
      taskId: r.taskId,
      dur: r.dur,
      critical: floats[i] === minFloat,
      label: r.kind === 'task' ? `${r.dur}d` : (r.lag && r.lag !== 0 ? `${r.lag > 0 ? '+' : ''}${r.lag}d` : ''),
    }
  })
  for (const e of edges) {
    if (e.taskId) taskEdge[e.taskId] = e.id
  }

  // ── 布局（真时标：x=最早时间×列宽，1 天=AOA_COL_W px）──────────
  // v4.130 刀H G4 前提：时标网络图的水平距离必须与时间成线性（旧版按 es 排名
  // 分列，es 有空洞时跨边距离不可比、波形线读不出真实自由时差）。
  // 行序仍由 layerByTime 重心松弛决定（同列/邻列交错最少），列坐标弃用。
  const pos = layerByTime(
    nodeIds.map((id) => ({ id, es: nodeEs.get(id)! })),
    raws.map((r) => ({ from: r.from, to: r.to })),
  )
  // 锚点键：事件成员关系求逆（task → start/end 两成员键），取字典序最小者为代表
  const nodeMembers = new Map<string, string[]>(nodeIds.map((id) => [id, []]))
  for (const t of tasks) {
    nodeMembers.get(startOf.get(t.id)!)!.push(`start:${t.id}`)
    nodeMembers.get(endOf.get(t.id)!)!.push(`end:${t.id}`)
  }
  const nodes: AoaNode[] = nodeIds.map((id) => {
    const p = pos.get(id) ?? { col: 0, row: 0 }
    return {
      id,
      num: num.get(id)!,
      es: nodeEs.get(id)!,
      ls: nodeLs.get(id)!,
      x: AOA_MARGIN + nodeEs.get(id)! * AOA_COL_W,
      y: AOA_MARGIN + p.row * AOA_ROW_H,
      anchor: id === START ? 'S' : id === END ? 'T' : (nodeMembers.get(id)!.sort()[0] ?? id),
    }
  })

  return { ok: true, nodes, edges, taskEdge }
}
