/**
 * networkExport.ts — 网络图上报图面构建器（纯函数 → SVG 字符串，v4.133.0 刀D2 /
 * v4.137.0 刀D4 增里程碑旗标）
 *
 * 对标 .gzp 时标网络上报件（重庆干休所件口径）：双代号=时标网络+工程标尺
 * （工程日/月/日/星期四行，月界竖线贯通图面）；单代号=拓扑分层+六格节点
 * （逻辑关系图不带时标，JGJ/T 121 口径）。
 * 几何复用刀H 纯函数：AOA 箭线=edgeSegs/findBridgeArcs/segsToPath（自定义
 * 日宽——图面按计划长度自适应 36~110px/天，长计划不爆画布）；PDM 分层=
 * layerByTopology、盒尺寸与连线三段式同 PdmView 口径（isLinkBinding 本地
 * 同构副本，判定口径：后继日期恰由该搭接决定）。
 * 标题带/图签/调色板/编制说明注共用 ganttExport 导出件；全部纯函数零 DOM。
 * v4.137.0 刀D4：里程碑节点加小旗标——单代号六格盒右上角、双代号里程碑
 * 完成事件圈右上角（与横道菱形旗同款形状，本文件内重复实现，不建第三文件）。
 */
import type { AoaGraph } from './aoa'
import { AOA_R, AOA_ROW_H } from './aoa'
import { edgeSegs, findBridgeArcs, segsToPath, assignChannels } from './aoaLayout'
import { layerByTopology } from './layout'
import { wdToDate } from './calendar'
import type { CpmResult, LinkType, SchedProject } from './types'
import {
  EXP_COLORS, EXP_FONT, EXP_MARGIN, EXP_TITLE_H, esc, exportNotesBlockH, exportNotesSvg,
  fitText, exportSignSvg, exportTitleSvg, type ExportMeta,
} from './ganttExport'

const C = EXP_COLORS
const FONT = EXP_FONT
const M = EXP_MARGIN
const TITLE_H = EXP_TITLE_H

export interface NetworkExportSvg {
  svg: string
  w: number
  h: number
}

/** AOA 左侧留白：工程标尺行名（工程日/月/日/星期）右对齐落位 */
const AOA_LGUT = 40

/**
 * 里程碑小旗（与 ganttExport 内 expFlagSvg 同款形状的本文件副本）：旗杆竖线 +
 * 三角旗面，data-exp-flag 供导出图面测试断言。(x,y)=旗杆顶点，poleH=杆高，
 * fw/fh=旗面宽高（旗面自杆顶向右）；颜色传 EXP_COLORS 现成色（关键红）。
 */
function expFlagSvg(x: number, y: number, poleH: number, fw: number, fh: number, color: string): string {
  return (
    `<g class="sched-exp-flag" data-exp-flag="1">` +
    `<line x1="${x}" y1="${y}" x2="${x}" y2="${y + poleH}" stroke="${color}" stroke-width="1.5"/>` +
    `<polygon points="${x},${y} ${x + fw},${y + fh / 2} ${x},${y + fh}" fill="${color}"/>` +
    `</g>`
  )
}

/** PDM 节点盒尺寸（与 PdmView 同口径：六格标注法 150×92） */
const PDM_NODE_W = 150
const PDM_NODE_H = 92
const PDM_COL_GAP = 74
const PDM_ROW_GAP = 40
const PDM_MARGIN = 28

/** 该搭接当前是否为「绑定约束」（PdmView.isLinkBinding 同构副本；四型判定穷尽） */
function isLinkBinding(type: LinkType, lag: number, f: { es: number; ef: number }, t: { es: number; ef: number }): boolean {
  switch (type) {
    case 'FS': return t.es === f.ef + lag
    case 'SS': return t.es === f.es + lag
    case 'FF': return t.ef === f.ef + lag
    case 'SF': return t.ef === f.es + lag
  }
}

/**
 * 双代号时标网络上报件。graph 未过（循环依赖）或空图抛错。
 * 水平轴=最早时间（1 格=1 天，发布物恒取时标口径，手动布点不入图面）；
 * 工程标尺四行：工程日（步长按总工期 5/10/20 自适应）、月、日、星期；
 * 月界竖线贯通图面。
 */
export function buildAoaExportSvg(project: SchedProject, graph: AoaGraph, meta: ExportMeta = {}): NetworkExportSvg {
  if (!graph.ok) throw new Error(graph.error ?? '计划存在循环依赖，导出前先修正搭接')
  if (graph.nodes.length === 0) throw new Error('暂无任务，无法绘制网络图')
  const total = Math.max(...graph.nodes.map((n) => n.es))
  // 图面日宽自适应：长计划压窄不爆画布（34 天=110 上限，200 天≈59，400 天=36 下限）
  const dayW = Math.max(36, Math.min(110, Math.floor(12000 / (total + 2))))
  // 节点位：x=最早时间×日宽（发布物时标口径），y 沿用引擎重心行布局
  const nodeById = new Map(graph.nodes.map((n) => [n.id, { ...n, x: AOA_LGUT + n.es * dayW }]))
  const netH = Math.max(...graph.nodes.map((n) => n.y)) + AOA_ROW_H / 2 + 46
  const chartW = AOA_LGUT + (total + 1) * dayW + M
  const RULER_H = 62
  const notesH = exportNotesBlockH(meta.notes) // 编制说明块占高（无 notes=0，布局不变）
  const footY = TITLE_H + netH + RULER_H + 24 + notesH // 图脚整体下移说明块高度，块不压图面
  const totalH = footY + 44
  const dateAt = (wd: number): Date => wdToDate(project.startDate, wd, project.calendar)

  // 箭线：平行边通道 + 波形切点（实体工期终点）+ 过桥
  const pairCnt = new Map<string, number>()
  for (const e of graph.edges) {
    const k = `${e.from}>${e.to}`
    pairCnt.set(k, (pairCnt.get(k) ?? 0) + 1)
  }
  const pairSeen = new Map<string, number>()
  // 同行长边通道分配（与视图同口径：长线走行间通道不重合）
  const channels = assignChannels(graph.edges, nodeById)
  const geoms = graph.edges.map((e, i) => {
    const k = `${e.from}>${e.to}`
    const idx = pairSeen.get(k) ?? 0
    pairSeen.set(k, idx + 1)
    const a = nodeById.get(e.from)!
    // 汇总箭线横跨子网络界点（时间由二级决定）、走通道的边：波形切点无意义不画
    const chY = channels.get(e.id)
    const waveFromX = e.kind === 'summary' ? null : a.x + AOA_R + e.dur * dayW
    return { e, i, waveFromX, g: edgeSegs(e, nodeById, { idx, cnt: pairCnt.get(k)! }, waveFromX, chY) }
  })
  const bridges = findBridgeArcs(geoms.map((it) => it.g.segs))
  const taskName = (id: string): string => project.tasks.find((t) => t.id === id)?.name ?? ''

  let net = `<defs>` +
    `<marker id="exp-aoa-arrow" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto"><path d="M0,0 L9,4.5 L0,9 z" fill="${C.link}"/></marker>` +
    `<marker id="exp-aoa-arrow-crit" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto"><path d="M0,0 L9,4.5 L0,9 z" fill="${C.critical}"/></marker>` +
    `<marker id="exp-aoa-arrow-summary" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto"><path d="M0,0 L9,4.5 L0,9 z" fill="${C.ink}"/></marker>` +
    `</defs>`
  // 月界竖线（浅，贯通图面+标尺）
  let lastMonth = `${dateAt(0).getUTCFullYear()}-${dateAt(0).getUTCMonth()}`
  for (let wd = 1; wd <= total; wd++) {
    const d = dateAt(wd)
    const mk = `${d.getUTCFullYear()}-${d.getUTCMonth()}`
    if (mk !== lastMonth) {
      lastMonth = mk
      net += `<line x1="${AOA_LGUT + wd * dayW}" y1="0" x2="${AOA_LGUT + wd * dayW}" y2="${netH + RULER_H}" stroke="${C.grid}" stroke-width="1"/>`
    }
  }
  for (const { e, i, waveFromX, g } of geoms) {
    const crit = e.critical
    const summary = e.kind === 'summary'
    const cls = `sched-exp-aoa${crit ? ' sched-exp-aoa-critical' : ''}${e.kind === 'dummy' ? ' sched-exp-aoa-dummy' : ''}${summary ? ' sched-exp-aoa-summary' : ''}`
    const strokeC = summary ? C.ink : crit ? C.critical : C.link
    net += `<path class="${cls}" d="${segsToPath(g.segs, waveFromX, bridges.get(i))}" fill="none" stroke="${strokeC}" stroke-width="${summary ? 2.6 : crit ? 2.4 : 1.2}"${e.kind === 'dummy' ? ' stroke-dasharray="6 4"' : ''} marker-end="url(#exp-aoa-arrow${crit ? '-crit' : summary ? '-summary' : ''})"/>`
    const nameText = e.taskId ? taskName(e.taskId) : ''
    if (nameText) {
      net += `<text class="sched-exp-aoa-name${summary ? ' sched-exp-aoa-name-summary' : ''}" x="${g.name.x}" y="${g.name.y}" text-anchor="${g.name.anchor}" font-family="${FONT}" font-size="11" font-weight="${summary ? 700 : 400}" fill="${crit ? C.critical : C.ink}">${esc(nameText)}</text>`
    }
    const durText = e.kind === 'task' ? String(e.dur) : e.label
    if (durText) {
      net += `<text class="sched-exp-aoa-dur" x="${g.dur.x}" y="${g.dur.y}" text-anchor="${g.dur.anchor}" font-family="${FONT}" font-size="10" fill="${crit ? C.critical : C.dim}">${esc(durText)}</text>`
    }
  }
  // 事件节点：圈+编号，上=最早时间、下=最迟时间
  for (const n of nodeById.values()) {
    net += `<circle class="sched-exp-aoa-node" cx="${n.x}" cy="${n.y}" r="${AOA_R}" fill="white" stroke="${C.border}" stroke-width="1.4"/>`
    net += `<text x="${n.x}" y="${n.y + 4}" text-anchor="middle" font-family="${FONT}" font-size="11" font-weight="600" fill="${C.ink}">${n.num}</text>`
    net += `<text x="${n.x}" y="${n.y - AOA_R - 6}" text-anchor="middle" font-family="${FONT}" font-size="10" fill="${C.dim}">${n.es}</text>`
    net += `<text x="${n.x}" y="${n.y + AOA_R + 14}" text-anchor="middle" font-family="${FONT}" font-size="10" fill="${C.dim}">${n.ls}</text>`
  }
  // 里程碑旗标：里程碑实工作箭线的完成事件圈右上角加缩小版小旗（杆高 10，
  // 让开圈上方最早时间文本）；多里程碑共事件按事件去重只画一面
  const mileEnds = new Set<string>()
  for (const e of graph.edges) {
    if (e.kind !== 'task' || !e.taskId) continue
    const task = project.tasks.find((x) => x.id === e.taskId)
    if (task?.isMilestone) mileEnds.add(e.to)
  }
  for (const nid of mileEnds) {
    const n = nodeById.get(nid)
    if (n) net += expFlagSvg(n.x + AOA_R + 3, n.y - AOA_R - 2, 10, 6, 5, C.critical)
  }

  // 工程标尺四行（列=工作日；非工作日不占列，与引擎时标口径一致）
  const step = total > 120 ? 20 : total > 60 ? 10 : 5
  const WEEK = ['日', '一', '二', '三', '四', '五', '六']
  const yTick = 14
  const yMon = 30
  const yDay = 45
  const yWk = 59
  let ruler = `<line x1="${AOA_LGUT}" y1="0" x2="${chartW - M}" y2="0" stroke="${C.border}"/>`
  ruler += `<text x="${AOA_LGUT - 6}" y="${yTick}" text-anchor="end" font-family="${FONT}" font-size="9" fill="${C.dim}">工程日</text>`
  ruler += `<text x="${AOA_LGUT - 6}" y="${yMon}" text-anchor="end" font-family="${FONT}" font-size="9" fill="${C.dim}">月</text>`
  ruler += `<text x="${AOA_LGUT - 6}" y="${yDay}" text-anchor="end" font-family="${FONT}" font-size="9" fill="${C.dim}">日</text>`
  ruler += `<text x="${AOA_LGUT - 6}" y="${yWk}" text-anchor="end" font-family="${FONT}" font-size="9" fill="${C.dim}">星期</text>`
  for (let wd = 0; wd <= total; wd++) {
    const x = AOA_LGUT + wd * dayW
    const d = dateAt(wd)
    if (wd % step === 0 || wd === total) {
      ruler += `<line x1="${x}" y1="0" x2="${x}" y2="10" stroke="${C.border}"/>`
      ruler += `<text x="${x}" y="${yTick}" text-anchor="middle" font-family="${FONT}" font-size="9" fill="${wd === total ? C.critical : C.dim}" font-weight="${wd === total ? '600' : '400'}">${wd}</text>`
    }
    // 月标签：本月首个工作日（日号不增即跨月）
    if (wd === 0 || d.getUTCDate() <= dateAt(wd - 1).getUTCDate()) {
      ruler += `<text x="${x + 2}" y="${yMon}" font-family="${FONT}" font-size="9" fill="${C.ink}">${d.getUTCFullYear()}.${d.getUTCMonth() + 1}</text>`
    }
    ruler += `<text x="${x}" y="${yDay}" text-anchor="middle" font-family="${FONT}" font-size="9" fill="${C.ink}">${d.getUTCDate()}</text>`
    ruler += `<text x="${x}" y="${yWk}" text-anchor="middle" font-family="${FONT}" font-size="9" fill="${C.dim}">${WEEK[d.getUTCDay()]}</text>`
  }

  // 图例 + 口径注
  const legendY = footY + 4
  let foot = ''
  let lx = M
  const items: { draw: string; label: string; lw: number }[] = [
    { draw: `<line x1="${lx}" y1="${legendY}" x2="${lx + 20}" y2="${legendY}" stroke="${C.critical}" stroke-width="2.4"/>`, label: '关键工作', lw: 20 },
    { draw: `<line x1="${lx}" y1="${legendY}" x2="${lx + 20}" y2="${legendY}" stroke="${C.link}" stroke-width="1.2"/>`, label: '工作', lw: 20 },
    { draw: `<line x1="${lx}" y1="${legendY}" x2="${lx + 20}" y2="${legendY}" stroke="${C.link}" stroke-width="1.2" stroke-dasharray="6 4"/>`, label: '虚工作', lw: 20 },
    { draw: `<line x1="${lx}" y1="${legendY}" x2="${lx + 20}" y2="${legendY}" stroke="${C.ink}" stroke-width="2.6"/>`, label: '一级汇总（界点衔接二级）', lw: 20 },
    { draw: `<path d="M${lx} ${legendY} q 2.5 -5 5 0 t 5 0 t 5 0 t 5 0" fill="none" stroke="${C.link}" stroke-width="1.2"/>`, label: '自由时差（波形线）', lw: 22 },
  ]
  for (const it of items) {
    foot += it.draw
    foot += `<text x="${lx + it.lw + 6}" y="${legendY + 4}" font-family="${FONT}" font-size="11" fill="${C.ink}">${it.label}</text>`
    lx += it.lw + 6 + it.label.length * 11 + 18
  }
  foot += `<text x="${lx}" y="${legendY + 4}" font-family="${FONT}" font-size="10" fill="${C.dim}">节点：上=最早时间 · 下=最迟时间\u3000标注：箭线上=工作名称 · 下=工期</text>`

  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" width="${chartW}" height="${totalH}" viewBox="0 0 ${chartW} ${totalH}" class="sched-exp-aoa">` +
    `<rect width="${chartW}" height="${totalH}" fill="white"/>` +
    exportTitleSvg(meta, chartW, `${project.name || '进度计划'}\u3000施工进度计划（双代号时标网络图）`) +
    `<g transform="translate(0,${TITLE_H})">` + net + `</g>` +
    `<g transform="translate(0,${TITLE_H + netH})">` + ruler + `</g>` +
    foot +
    exportNotesSvg(meta, footY - 8, chartW) + // 块底=图签顶(footY+4-8)上浮 4px
    exportSignSvg(meta, footY + 4, chartW) +
    `</svg>`
  return { svg, w: chartW, h: totalH }
}

/**
 * 单代号网络图上报件（逻辑关系图，不带时标——JGJ/T 121 单代号的绘图口径）。
 * 布局与节点盒同 PdmView 口径：拓扑分层（并行分支同列并列）、多起点/终点
 * 增虚拟 S/T（虚线框）、六格标注（上 ES/D/EF、下 LS/TF/LF）、绑定中的临界
 * 搭接红色贯通。
 */
export function buildPdmExportSvg(project: SchedProject, cpm: CpmResult, meta: ExportMeta = {}): NetworkExportSvg {
  if (!cpm.ok) throw new Error(cpm.error ?? '计划存在循环依赖，导出前先修正搭接')
  const leaves = project.tasks.filter((t) => t.level > 0)
  if (leaves.length === 0) throw new Error('暂无任务，无法绘制网络图')
  const ids = new Set(leaves.map((t) => t.id))
  const links = project.links.filter((l) => ids.has(l.from) && ids.has(l.to))
  const pos = layerByTopology(leaves.map((t) => ({ id: t.id })), links.map((l) => ({ from: l.from, to: l.to })))
  const inIds = new Set(links.map((l) => l.to))
  const outIds = new Set(links.map((l) => l.from))
  const sources = leaves.filter((t) => !inIds.has(t.id)).map((t) => t.id)
  const sinks = leaves.filter((t) => !outIds.has(t.id)).map((t) => t.id)
  const hasStart = sources.length > 1
  const hasEnd = sinks.length > 1
  const colShift = hasStart ? 1 : 0
  const maxCol = Math.max(0, ...[...pos.values()].map((p) => p.col))
  const maxRow = Math.max(0, ...[...pos.values()].map((p) => p.row))
  const w = PDM_MARGIN * 2 + (maxCol + colShift + (hasEnd ? 1 : 0) + 1) * (PDM_NODE_W + PDM_COL_GAP)
  const netH = PDM_MARGIN * 2 + (maxRow + 1) * (PDM_NODE_H + PDM_ROW_GAP)
  const nodeX = (id: string): number => PDM_MARGIN + (pos.get(id)!.col + colShift) * (PDM_NODE_W + PDM_COL_GAP)
  const nodeY = (id: string): number => PDM_MARGIN + pos.get(id)!.row * (PDM_NODE_H + PDM_ROW_GAP)
  const midY = PDM_MARGIN + (maxRow * (PDM_NODE_H + PDM_ROW_GAP)) / 2
  const startX = PDM_MARGIN
  const endX = PDM_MARGIN + (maxCol + colShift + 1) * (PDM_NODE_W + PDM_COL_GAP)
  const rowNo = new Map(project.tasks.map((t, i) => [t.id, i + 1]))

  const totalW = w + M * 2
  const notesH = exportNotesBlockH(meta.notes) // 编制说明块占高（无 notes=0，布局不变）
  const footY = TITLE_H + netH + 24 + notesH // 图脚整体下移说明块高度，块不压图面
  const totalH = footY + 44

  /** 连线三段式：右缘出、直角拐、左缘入（与 PdmView 同口径） */
  const linkSegs = (l: { from: string; to: string }): { segs: { x1: number; y1: number; x2: number; y2: number }[]; labelXY: { x: number; y: number } } => {
    const x1 = nodeX(l.from) + PDM_NODE_W
    const y1 = nodeY(l.from) + PDM_NODE_H / 2
    const x2 = nodeX(l.to)
    const y2 = nodeY(l.to) + PDM_NODE_H / 2
    const midX = Math.max(x1 + 8, (x1 + x2) / 2)
    return {
      segs: [
        { x1, y1, x2: midX, y2: y1 },
        { x1: midX, y1, x2: midX, y2 },
        { x1: midX, y1: y2, x2: x2 - 4, y2 },
      ],
      labelXY: { x: midX, y: (y1 + y2) / 2 - 4 },
    }
  }
  const segLists = links.map((l) => linkSegs(l).segs)
  // 虚拟起点/终点短桩
  const stubStart = hasStart ? sources.map((id) => {
    const yS = midY + PDM_NODE_H / 2
    const yT = nodeY(id) + PDM_NODE_H / 2
    const xV = startX + PDM_NODE_W + 8
    return [
      { x1: startX + PDM_NODE_W, y1: yS, x2: xV, y2: yS },
      { x1: xV, y1: yS, x2: xV, y2: yT },
      { x1: xV, y1: yT, x2: nodeX(id) - 4, y2: yT },
    ]
  }) : []
  const stubEnd = hasEnd ? sinks.map((id) => {
    const yN = nodeY(id) + PDM_NODE_H / 2
    const yT = midY + PDM_NODE_H / 2
    const xV = endX - 8
    return [
      { x1: nodeX(id) + PDM_NODE_W, y1: yN, x2: xV, y2: yN },
      { x1: xV, y1: yN, x2: xV, y2: yT },
      { x1: xV, y1: yT, x2: endX - 4, y2: yT },
    ]
  }) : []
  const bridges = findBridgeArcs([...segLists, ...stubStart, ...stubEnd])

  let net = `<defs>` +
    `<marker id="exp-pdm-arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="${C.link}"/></marker>` +
    `<marker id="exp-pdm-arrow-crit" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="${C.critical}"/></marker>` +
    `</defs>`
  links.forEach((l, i) => {
    const f = cpm.rows[l.from]
    const t = cpm.rows[l.to]
    const crit = !!(f && t && f.critical && t.critical && isLinkBinding(l.type, l.lag, f, t))
    const { segs, labelXY } = linkSegs(l)
    net += `<path class="sched-exp-pdm-link${crit ? ' sched-exp-pdm-link-critical' : ''}" d="${segsToPath(segs, null, bridges.get(i))}" fill="none" stroke="${crit ? C.critical : C.link}" stroke-width="${crit ? 2 : 1.2}" marker-end="url(#exp-pdm-arrow${crit ? '-crit' : ''})"/>`
    const label = `${l.type}${l.lag !== 0 ? (l.lag > 0 ? '+' : '') + l.lag : ''}`
    net += `<text x="${labelXY.x}" y="${labelXY.y}" text-anchor="middle" font-family="${FONT}" font-size="10" fill="${crit ? C.critical : C.dim}">${esc(label)}</text>`
  })
  stubStart.forEach((segs, j) => {
    net += `<path class="sched-exp-pdm-link sched-exp-pdm-link-virtual" d="${segsToPath(segs, null, bridges.get(segLists.length + j))}" fill="none" stroke="${C.link}" stroke-width="1.2" stroke-dasharray="5 4" marker-end="url(#exp-pdm-arrow)"/>`
  })
  stubEnd.forEach((segs, j) => {
    net += `<path class="sched-exp-pdm-link sched-exp-pdm-link-virtual" d="${segsToPath(segs, null, bridges.get(segLists.length + stubStart.length + j))}" fill="none" stroke="${C.link}" stroke-width="1.2" stroke-dasharray="5 4" marker-end="url(#exp-pdm-arrow)"/>`
  })

  // 六格节点盒
  const box = (id: string): string => {
    const t = project.tasks.find((x) => x.id === id)!
    const row = cpm.rows[id]
    if (!row) return ''
    const x = nodeX(id)
    const y = nodeY(id)
    const crit = !!row.critical
    const nameH = 26
    const cellW = PDM_NODE_W / 3
    const cellH = (PDM_NODE_H - nameH) / 2
    let out = `<rect class="sched-exp-pdm-node${crit ? ' sched-exp-pdm-node-critical' : ''}" x="${x}" y="${y}" width="${PDM_NODE_W}" height="${PDM_NODE_H}" rx="4" fill="white" stroke="${crit ? C.critical : C.border}" stroke-width="${crit ? 1.8 : 1.2}"/>`
    out += `<text x="${x + 7}" y="${y + 17}" font-family="${FONT}" font-size="9" fill="${C.dim}">${rowNo.get(id) ?? ''}</text>`
    // 里程碑名 fit 宽度收窄 14px：给右上角小旗让位，旗不压文字
    out += `<text x="${x + 20}" y="${y + 17}" font-family="${FONT}" font-size="10.5" font-weight="600" fill="${C.ink}">${esc(fitText(`${t.isMilestone ? '◆ ' : ''}${t.name}`, PDM_NODE_W - 26 - (t.isMilestone ? 14 : 0), 10.5))}</text>`
    out += `<line x1="${x}" y1="${y + nameH}" x2="${x + PDM_NODE_W}" y2="${y + nameH}" stroke="${C.grid}"/>`
    out += `<line x1="${x}" y1="${y + nameH + cellH}" x2="${x + PDM_NODE_W}" y2="${y + nameH + cellH}" stroke="${C.grid}"/>`
    for (let i = 1; i < 3; i++) {
      out += `<line x1="${x + i * cellW}" y1="${y + nameH}" x2="${x + i * cellW}" y2="${y + PDM_NODE_H}" stroke="${C.grid}"/>`
    }
    const cells: { v: string; c: number; r: number; critMark?: boolean }[] = [
      { v: String(row.es), c: 0, r: 0 }, { v: String(t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))), c: 1, r: 0 }, { v: String(row.ef), c: 2, r: 0 },
      { v: String(row.ls), c: 0, r: 1 }, { v: String(row.tf), c: 1, r: 1, critMark: true }, { v: String(row.lf), c: 2, r: 1 },
    ]
    for (const cell of cells) {
      out += `<text x="${x + cell.c * cellW + cellW / 2}" y="${y + nameH + cell.r * cellH + cellH / 2 + 4}" text-anchor="middle" font-family="${FONT}" font-size="11" fill="${cell.critMark && crit ? C.critical : C.ink}">${esc(cell.v)}</text>`
    }
    // 里程碑旗标（上报口径恒带）：六格盒右上角缩小版小旗（杆高 10，关键红）
    if (t.isMilestone) {
      out += expFlagSvg(x + PDM_NODE_W - 8, y + 4, 10, 6, 5, C.critical)
    }
    return out
  }
  for (const t of leaves) net += box(t.id)
  if (hasStart) {
    net += `<rect class="sched-exp-pdm-node sched-exp-pdm-node-virtual" x="${startX}" y="${midY}" width="${PDM_NODE_W}" height="${PDM_NODE_H}" rx="4" fill="white" stroke="${C.border}" stroke-width="1.2" stroke-dasharray="6 4"/>`
    net += `<text x="${startX + PDM_NODE_W / 2}" y="${midY + PDM_NODE_H / 2 + 4}" text-anchor="middle" font-family="${FONT}" font-size="12" font-weight="700" fill="${C.ink}">起点 S</text>`
  }
  if (hasEnd) {
    net += `<rect class="sched-exp-pdm-node sched-exp-pdm-node-virtual" x="${endX}" y="${midY}" width="${PDM_NODE_W}" height="${PDM_NODE_H}" rx="4" fill="white" stroke="${C.border}" stroke-width="1.2" stroke-dasharray="6 4"/>`
    net += `<text x="${endX + PDM_NODE_W / 2}" y="${midY + PDM_NODE_H / 2 + 4}" text-anchor="middle" font-family="${FONT}" font-size="12" font-weight="700" fill="${C.ink}">完成 T</text>`
  }

  // 图例 + 口径注
  const legendY = footY + 4
  let foot = ''
  let lx = M
  const items: { draw: string; label: string; lw: number }[] = [
    { draw: `<rect x="${lx}" y="${legendY - 8}" width="14" height="12" fill="none" stroke="${C.critical}" stroke-width="1.8"/>`, label: '关键工作', lw: 14 },
    { draw: `<rect x="${lx}" y="${legendY - 8}" width="14" height="12" fill="none" stroke="${C.border}" stroke-width="1.2"/>`, label: '非关键', lw: 14 },
    { draw: `<rect x="${lx}" y="${legendY - 8}" width="14" height="12" fill="none" stroke="${C.link}" stroke-width="1.2" stroke-dasharray="4 3"/>`, label: '虚拟节点', lw: 14 },
  ]
  for (const it of items) {
    foot += it.draw
    foot += `<text x="${lx + it.lw + 6}" y="${legendY + 4}" font-family="${FONT}" font-size="11" fill="${C.ink}">${it.label}</text>`
    lx += it.lw + 6 + it.label.length * 11 + 18
  }
  foot += `<text x="${lx}" y="${legendY + 4}" font-family="${FONT}" font-size="10" fill="${C.dim}">格：ES / 工期 / EF · LS / 总时差 / LF\u3000关键线路=红箭线贯通（绑定中的临界搭接）</text>`

  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" width="${totalW}" height="${totalH}" viewBox="0 0 ${totalW} ${totalH}" class="sched-exp-pdm">` +
    `<rect width="${totalW}" height="${totalH}" fill="white"/>` +
    exportTitleSvg(meta, totalW, `${project.name || '进度计划'}\u3000施工进度计划（单代号网络图）`) +
    `<g transform="translate(${M},${TITLE_H})">` + net + `</g>` +
    foot +
    exportNotesSvg(meta, footY - 8, totalW) + // 块底=图签顶(footY+4-8)上浮 4px
    exportSignSvg(meta, footY + 4, totalW) +
    `</svg>`
  return { svg, w: totalW, h: totalH }
}
