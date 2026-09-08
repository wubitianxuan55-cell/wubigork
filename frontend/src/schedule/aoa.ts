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
 *  - 分级衔接（v4.142）：一级（分组）工作=一条汇总箭线，从其二级子网络的开始
 *    界点事件连到完成界点事件——共享事件、嵌入同一张图；汇总箭线不参与正逆推
 *    （时间由二级工作决定）、恒非关键。分组行绝不单独成图。
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
   * 分级横幅序号（v4.160，对标上报件画法）：一级分组行=一条横幅，组内事件的
   * 归属带（共享事件取成员最小 band——跨组共用界点归前组，后组竖线衔接）。
   * 无分组行=单一横幅（行为与旧版一致）。
   */
  band: number
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
  /**
   * 'task'=实工作箭线；'dummy'=虚工作（虚线）；'summary'=分级汇总箭线（v4.142）：
   * 一级（分组）工作从其二级子网络的开始界点事件连到完成界点事件——同一批事件、
   * 同一张图（分级衔接口径），不参与正逆推（时间由二级工作决定），恒非关键。
   */
  kind: 'task' | 'dummy' | 'summary'
  /** kind='task'/'summary' 时对应的任务 id（summary=分组行 id） */
  taskId?: string
  /** 箭线时长（虚工作=0 或时距；汇总箭线=界点事件时间差） */
  dur: number
  critical: boolean
  /** 箭线标注：实工作=工期，虚工作=时距（0 不标），汇总=界点时间差 */
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

/**
 * 布局常量（视图层按比例消费）；AOA_R=事件圆半径（渲染/文字避让共用）。
 * AOA_ROW_H=基础行距（节点含上下时间标注占 ~56px，74px 时只剩 18px 空隙
 * ——用户实测「全部挤在一起」，2026-09 放宽到 112）；且它只是下限：buildAoa
 * 按整图纵横比自适应放大（AOA_ASPECT_MAX），长计划不再「又矮又长」。
 */
export const AOA_COL_W = 110
export const AOA_ROW_H = 112
export const AOA_MARGIN = 40
export const AOA_R = 16
/** 行距自适应的目标纵横比上限（宽/高）：超过则放大行距，上限 2×AOA_ROW_H */
export const AOA_ASPECT_MAX = 7

function effDur(t: SchedTask): number {
  return t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
}

export function buildAoa(tasks: SchedTask[], links: SchedLink[], opts?: { planFinish?: number | null }): AoaGraph {
  // 分组行（level 0，汇总口径）不是工作，不进双代号网络图（JGJ/T 121：网络图
  // 只画实际工作）。分组行入图会变成挂在 START/END 上的零工期假箭线，与二级
  // 实际工作网络割裂成「两张图」（v4.142 修正，用户实测反馈）。
  const acts = tasks.filter((t) => t.level > 0)
  if (acts.length === 0) return { ok: true, nodes: [], edges: [], taskEdge: {} }
  const byId = new Map(acts.map((t) => [t.id, t]))
  const valid = links.filter((l) => l.from !== l.to && byId.has(l.from) && byId.has(l.to))

  // 搭接索引：按 to 分组（约束入度）、FF/SF 指入记录（判断完成事件是否被污染）
  const inOf = new Map<string, SchedLink[]>()
  const outOf = new Map<string, SchedLink[]>()
  for (const t of acts) { inOf.set(t.id, []); outOf.set(t.id, []) }
  for (const l of valid) {
    inOf.get(l.to)!.push(l)
    outOf.get(l.from)!.push(l)
  }

  // PDM 层拓扑序（Kahn）；不完整 = 有环。事件分配必须按拓扑序进行，
  // 否则前向引用前置任务的 startOf/endOf 会取到 undefined。
  const indeg = new Map(acts.map((t) => [t.id, inOf.get(t.id)!.length]))
  const queue = acts.filter((t) => indeg.get(t.id) === 0).map((t) => t.id)
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
  if (order.length !== acts.length) {
    const inOrder = new Set(order)
    const names = acts.filter((t) => !inOrder.has(t.id)).map((t) => t.name)
    return { ok: false, error: `存在循环依赖：${names.join(' → ')}`, nodes: [], edges: [], taskEdge: {} }
  }

  // ── 分级横幅归属（v4.160，需在事件分配前算好：合并事件只许同带）──
  // 一级分组行（level 0）=一条横幅：分部汇总箭线走横幅顶部通长线（分部名
  // 标在线上），二级事件只在横幅内排布——一级/二级图面分区，不再按全局
  // 重心混排（用户实测「一级、二级没有分级展示，比较混乱」，对标重庆干休所
  // 网络计划上报件）。组前未分组叶任务并入头部合成横幅；组后叶任务并入
  // 最后一条分组横幅（扁平大纲轮廓语义=最后分组行拥有其后的全部非分组行，
  // 与 groupKids 同口径）；全无分组行=单一横幅（退化=旧行为）。
  const bandLabels: string[] = []
  const taskBand = new Map<string, number>()
  let curBand = -1
  for (const t of tasks) {
    if (t.level === 0) {
      curBand = bandLabels.length
      bandLabels.push(t.name)
    } else if (byId.has(t.id)) {
      if (curBand === -1) {
        curBand = bandLabels.length
        bandLabels.push('') // 组前未分组合成横幅
      }
      taskBand.set(t.id, curBand)
    }
  }

  // ── 事件分配 ──────────────────────────────────────────────
  const START = 'S'
  const END = 'T'
  // 起点终点事件预先注册（nodeIds 的事件注册表即 nodeEs 键集）
  const nodeEs = new Map<string, number>([[START, 0], [END, 0]])
  const startOf = new Map<string, string>()
  const endOf = new Map<string, string>()
  /** 非首横幅的无前置任务：自设开始事件（虚工作从 START 衔入，任务边不出带） */
  const ownStart = new Set<string>()
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
  // 跨横幅不合并事件（v4.160）：后学分部的工作不挂前学分部的事件上——
  // 否则任务边跨带穿行、标注压字；跨分部改由虚工作竖向衔接（上报件画法）。
  const sameBand = (a: string, b: string) => taskBand.get(a) === taskBand.get(b)

  for (const id of order) {
    const t = byId.get(id)!
    const preds = inOf.get(t.id)!
    if (preds.length === 0) {
      if ((taskBand.get(t.id) ?? 0) > 0) {
        startOf.set(t.id, nextNode(0))
        ownStart.add(t.id)
      } else {
        startOf.set(t.id, START)
      }
    } else if (
      preds.length === 1 && preds[0].type === 'FS' && preds[0].lag === 0 &&
      finishClean(preds[0].from) && sameBand(preds[0].from, t.id)
    ) {
      startOf.set(t.id, endOf.get(preds[0].from)!)
    } else if (
      preds.length === 1 && preds[0].type === 'SS' && preds[0].lag === 0 &&
      startClean(preds[0].from) && sameBand(preds[0].from, t.id)
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
    if (ownStart.has(t.id)) {
      const key = `${START}>${startOf.get(t.id)!}`
      if (!dummySeen.has(key)) { dummySeen.add(key); raws.push({ from: START, to: startOf.get(t.id)!, dur: 0, kind: 'dummy' }) }
    }
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
  // 逆推锚点（v4.129 刀G G3）：目标竣工换算的计划工期 < 计算工期时按计划工期逆推。
  // v4.133 修正：total 必须在正推**之后**取（事件注册时的初始 es=单任务工期，
  // 长链下远小于真实计算工期——逆推整体平移出假负时差，且 planFinish 锚点失真）。
  for (const id of eventOrder) {
    let es = 0
    for (const e of inEdges.get(id) ?? []) es = Math.max(es, nodeEs.get(e.from)! + e.dur)
    nodeEs.set(id, es)
  }
  const total = Math.max(0, ...nodeIds.map((id) => nodeEs.get(id)!))
  const planFinish = opts?.planFinish
  const finish = planFinish != null && planFinish < total ? planFinish : total
  for (let i = eventOrder.length - 1; i >= 0; i--) {
    const id = eventOrder[i]
    let ls = finish
    for (const e of outEdges.get(id) ?? []) ls = Math.min(ls, nodeLs.get(e.to)! - e.dur)
    nodeLs.set(id, ls)
  }

  // ── 编号（拓扑序 1..N）与箭线关键标记 ─────────────────────
  const num = new Map(eventOrder.map((id, i) => [id, i + 1]))
  // cd（日历天）任务标注（v4.152 刀3）：AOA 网格恒工作日刻度，cd 箭线工期数
  // 仍是自然日数（本图计算不换算），标「(日历)」防与工作日刻度混淆。
  const cdTasks = new Set(tasks.filter((t) => t.durationUnit === 'cd').map((t) => t.id))
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
      label: r.kind === 'task'
        ? `${r.dur}d${r.taskId && cdTasks.has(r.taskId) ? '(日历)' : ''}`
        : (r.lag && r.lag !== 0 ? `${r.lag > 0 ? '+' : ''}${r.lag}d` : ''),
    }
  })
  for (const e of edges) {
    if (e.taskId) taskEdge[e.taskId] = e.id
  }

  // ── 分级汇总箭线（一级工作界点衔接二级子网络）──────────────
  // 分组行轮廓配对（扁平大纲）：level 0 行的子级=其后连续的非分组行。界点=
  // 子网络最早开始事件 / 最迟完成事件（按正推后事件时间）。汇总箭线只进展示
  // 边集、不进 raws——正逆推/浮时/编号全部只由二级工作决定，一级不扰动计算。
  const groupKids: { taskId: string; childIds: string[] }[] = []
  {
    let cur: { taskId: string; childIds: string[] } | null = null
    for (const t of tasks) {
      if (t.level === 0) {
        if (cur) groupKids.push(cur)
        cur = { taskId: t.id, childIds: [] }
      } else if (cur) {
        cur.childIds.push(t.id)
      }
    }
    if (cur) groupKids.push(cur)
  }
  for (const g of groupKids) {
    const starts = g.childIds.map((id) => startOf.get(id)).filter((v): v is string => !!v)
    const ends = g.childIds.map((id) => endOf.get(id)).filter((v): v is string => !!v)
    if (starts.length === 0 || ends.length === 0) continue
    // 界点：最早开始事件 / 最迟完成事件（时间并列取编号小者，稳定）
    let bStart = starts[0]
    let bEnd = ends[0]
    for (const s of starts) {
      if (nodeEs.get(s)! < nodeEs.get(bStart)! || (nodeEs.get(s) === nodeEs.get(bStart)! && num.get(s)! < num.get(bStart)!)) bStart = s
    }
    for (const e of ends) {
      if (nodeEs.get(e)! > nodeEs.get(bEnd)! || (nodeEs.get(e) === nodeEs.get(bEnd)! && num.get(e)! < num.get(bEnd)!)) bEnd = e
    }
    if (bStart === bEnd) continue // 退化（全零工期子网）无箭线可画
    const dur = nodeEs.get(bEnd)! - nodeEs.get(bStart)!
    const edge: AoaEdge = {
      id: `s${edges.length}`,
      from: bStart,
      to: bEnd,
      kind: 'summary',
      taskId: g.taskId,
      dur,
      critical: false, // 汇总箭线不参与关键判定（红色留给二级关键线路）
      label: `${dur}d`,
    }
    edges.push(edge)
    taskEdge[g.taskId] = edge.id
  }

  // ── 布局（真时标：x=最早时间×列宽，1 天=AOA_COL_W px）──────────
  // v4.130 刀H G4 前提：时标网络图的水平距离必须与时间成线性（旧版按 es 排名
  // 分列，es 有空洞时跨边距离不可比、波形线读不出真实自由时差）。
  // 行序仍由 layerByTime 重心松弛决定（同列/邻列交错最少），列坐标弃用。
  const pos = layerByTime(
    nodeIds.map((id) => ({ id, es: nodeEs.get(id)! })),
    raws.map((r) => ({ from: r.from, to: r.to })),
  )
  // ── 分级横幅布局（v4.160，归属带已在事件分配前算好=taskBand/bandLabels）──
  // 事件归属带=成员任务最小 band（跨组共用界点归前组，后组经衔接线接入）
  const nodeBand = new Map<string, number>()
  const noteBand = (nid: string, b: number) => {
    const prev = nodeBand.get(nid)
    if (prev === undefined || b < prev) nodeBand.set(nid, b)
  }
  for (const t of acts) {
    const b = taskBand.get(t.id) ?? 0
    noteBand(startOf.get(t.id)!, b)
    noteBand(endOf.get(t.id)!, b)
  }
  // band 内保序打包：沿用全局重心行序为 band 内相对序（同列节点行号必不同，
  // 打包不产生同行冲突），band 间留 1 行空档容纳下一横幅的顶部汇总线；
  // y 首行再预留半个行距（band 0 顶汇总线不越界）。
  const bandRowCount = new Map<number, number>()
  const ordered = nodeIds
    .map((id) => ({ id, band: nodeBand.get(id) ?? 0, gRow: pos.get(id)!.row, col: pos.get(id)!.col }))
    .sort((p, q) => p.band - q.band || p.gRow - q.gRow || p.col - q.col)
  for (const it of ordered) bandRowCount.set(it.band, (bandRowCount.get(it.band) ?? 0) + 1)
  const bandBase = new Map<number, number>()
  {
    let acc = 0
    for (let b = 0; b < bandLabels.length; b++) {
      bandBase.set(b, acc)
      acc += (bandRowCount.get(b) ?? 0) + 1 // +1=横幅间空档
    }
  }
  const rowOf = new Map<string, number>()
  {
    const packCursor = new Map<number, number>()
    for (const it of ordered) {
      const r = packCursor.get(it.band) ?? 0
      packCursor.set(it.band, r + 1)
      rowOf.set(it.id, (bandBase.get(it.band) ?? 0) + r)
    }
  }
  const rowsTotal = Math.max(1, ...[...rowOf.values()].map((r) => r + 1))
  // 行距自适应（比例协调）：时标图宽度被工期锁定，低并行度的长计划行数少，
  // 天然「又矮又长」——宽高比超 AOA_ASPECT_MAX 时按比例放大行距（上限 2×
  // 基础行距，行间呼吸与整图比例两顾；y 仍与时间无耦合，波形语义不受影响）。
  const w = AOA_MARGIN * 2 + total * AOA_COL_W
  const hBase = AOA_MARGIN * 2 + rowsTotal * AOA_ROW_H
  const rowH = w <= hBase * AOA_ASPECT_MAX
    ? AOA_ROW_H
    : Math.min(AOA_ROW_H * 2, Math.floor((w / AOA_ASPECT_MAX - AOA_MARGIN * 2) / rowsTotal))
  // 锚点键：事件成员关系求逆（task → start/end 两成员键），取字典序最小者为代表
  const nodeMembers = new Map<string, string[]>(nodeIds.map((id) => [id, []]))
  for (const t of acts) {
    nodeMembers.get(startOf.get(t.id)!)!.push(`start:${t.id}`)
    nodeMembers.get(endOf.get(t.id)!)!.push(`end:${t.id}`)
  }
  const nodes: AoaNode[] = nodeIds.map((id) => {
    return {
      id,
      num: num.get(id)!,
      es: nodeEs.get(id)!,
      ls: nodeLs.get(id)!,
      x: AOA_MARGIN + nodeEs.get(id)! * AOA_COL_W,
      y: AOA_MARGIN + rowH / 2 + (rowOf.get(id) ?? 0) * rowH,
      band: nodeBand.get(id) ?? 0,
      anchor: id === START ? 'S' : id === END ? 'T' : (nodeMembers.get(id)!.sort()[0] ?? id),
    }
  })

  return { ok: true, nodes, edges, taskEdge }
}
