/**
 * schedule/networkLayout.ts — 布图参数（错层/层高/紧凑度/名称排布/非重要连线）的唯一解析与变换
 *
 * 背景（ZPert W4 收口）：`SchedNetworkStyle` 里 `namePos / hideUnimportantLink /
 * layerHeight1-2 / stagger / density` 五个键此前已落盘但**无人消费**——面板能写、
 * 画面不动，等于假开关。本文件把它们接进渲染，且只走一份解析（与
 * `networkDisplay.ts` 的四开关/几何参数同一纪律：视图层禁止写第二份缺省）。
 *
 * 三条硬约束（缺省零回归）：
 *  1. 全部缺省（无 networkStyle / 键全缺省）时 `resolveNetworkLayout` 产出 `isDefault=true`，
 *     `applyNetworkLayout` **原样返回同一个 graph 引用**——节点坐标数组逐字相等，
 *     渲染输出与改动前逐字一致；
 *  2. 变换只动**坐标**（x/y），绝不动 es/ls/num/band/anchor/edges/taskEdge——
 *     拓扑与 CPM 参数零影响（布图是显示层动作，不是计算层动作）；
 *  3. 消费不了的键就**不消费**：宁可缺，不造假（见 §「未消费」注释）。
 *
 * 斑马口径对齐（进度计划/gaea-schedule-zpert8-network-re-20261114.md §4.1–§4.2）：
 *  - `is_stagger`=智能位错（相邻事件纵向错开，减少箭头交叉）；
 *  - `layer_height1/2`=两层高度（一级横幅带高 / 二级行高）；
 *  - `layout_density`=布图紧凑度（更紧凑/常规/更疏）；
 *  - `name_pos`=工作名称排布（自动/横向/纵向）；
 *  - `is_hide_unimportant_link`=隐藏非重要关系线。
 */
import { AOA_COL_W, AOA_MARGIN, AOA_ROW_H, type AoaEdge, type AoaGraph } from './aoa'
import type { Seg, SegLabel } from './aoaLayout'
import { NETWORK_GEOMETRY_DEFAULTS } from './networkDisplay'
import type { SchedNetworkStyle } from './types'

/** 名称排布（斑马 name_pos） */
export type NamePos = 'auto' | 'horizontal' | 'vertical'

/** 布图紧凑度（斑马 layout_density） */
export type LayoutDensity = 'compact' | 'normal' | 'loose'

/** 行距绝对安全域（px）：layerHeight1/2 只是上下限约束，不能把行距压成负数或撑爆画布 */
export const AOA_ROW_MIN_H = 56
export const AOA_ROW_MAX_H = 320
/** 横幅（band）带高下限（px）：与 AOA_ROW_H=112 同一坐标系的「一行留白」量级 */
export const AOA_BAND_MIN_H = 96

/**
 * 紧凑度系数：**只**调列距/行距的整体比例（不改拓扑、不改 CPM、不改时标线性——
 * 列宽系数同时喂给工程标尺与波形切点，时标仍与时间成线性，见 AoaView）。
 */
export const DENSITY_SCALE: Record<LayoutDensity, number> = {
  compact: 0.85,
  normal: 1,
  loose: 1.15,
}

/** 布图参数（缺省值=现行 AOA 布图口径，改这里即改全部视图） */
export interface NetworkLayoutParams {
  density: LayoutDensity
  /** 列距系数（1=现行 AOA_COL_W/天） */
  colScale: number
  /** 行距系数（1=buildAoa 现行 rowH） */
  rowScale: number
  /** 有效列宽（px/天）= AOA_COL_W × colScale：标尺/波形切点共用，保证时标线性 */
  colW: number
  /** 行距下限（px）：来自 layerHeight2，未给则 AOA_ROW_MIN_H */
  rowMin: number
  /** 行距上限（px）：来自 layerHeight1（一级层高不得小于二级层高），未给则 AOA_ROW_MAX_H */
  rowMax: number
  /** 横幅带高下限（px）：来自 layerHeight1（一级层高），未给则 AOA_BAND_MIN_H */
  bandMin: number
  /** 错层排布（斑马 is_stagger）：相邻行纵向错开半个行距 */
  stagger: boolean
  /** 名称排布（斑马 name_pos）：auto=现行按边方向自动 */
  namePos: NamePos
  /** 隐藏非重要关系线（斑马 is_hide_unimportant_link） */
  hideUnimportantLink: boolean
  /**
   * 几何是否全缺省：true ⇒ 消费方必须走「不变换」零回归路径
   * （applyNetworkLayout 直接返回原 graph 引用，坐标数组逐字相等）。
   * 注意：namePos/hideUnimportantLink 是**显示开关**不是几何量，不参与本标志。
   */
  isDefault: boolean
}

export const NETWORK_LAYOUT_DEFAULTS: NetworkLayoutParams = {
  density: 'normal',
  colScale: 1,
  rowScale: 1,
  colW: AOA_COL_W,
  rowMin: AOA_ROW_MIN_H,
  rowMax: AOA_ROW_MAX_H,
  bandMin: AOA_BAND_MIN_H,
  stagger: false,
  namePos: 'auto',
  hideUnimportantLink: false,
  isDefault: true,
}

/** 有限数值 + 区间夹取；越界/非数一律回落 null（落盘侧已 normalize，这里兜运行期内存值） */
function pxIn(v: unknown, lo: number, hi: number): number | null {
  return typeof v === 'number' && Number.isFinite(v) && v >= lo && v <= hi ? v : null
}

/**
 * 五键解析（唯一入口）。
 * - `density`：只认三档字面量，其余回落 normal；
 * - `layerHeight1`：一级横幅层高 → 横幅带高下限 + 行距上限（[40,600] 外回落）；
 * - `layerHeight2`：二级行高 → 行距下限（[24,400] 外回落）；
 * - `layerHeight1 < layerHeight2`（运行期脏值）：上限抬到下限，不允许负区间；
 * - `stagger`：只认显式 true（缺省/脏值=false，旧文件零变化）；
 * - `namePos`：只认三档，其余回落 auto；
 * - `hideUnimportantLink`：只认显式 true。
 */
export function resolveNetworkLayout(style?: SchedNetworkStyle | null): NetworkLayoutParams {
  const density: LayoutDensity =
    style?.density === 'compact' || style?.density === 'loose' || style?.density === 'normal'
      ? style.density
      : NETWORK_LAYOUT_DEFAULTS.density
  const scale = DENSITY_SCALE[density]
  const h1 = pxIn(style?.layerHeight1, 40, 600)
  const h2 = pxIn(style?.layerHeight2, 24, 400)
  const rowMin = h2 ?? AOA_ROW_MIN_H
  const rowMax = Math.max(h1 ?? AOA_ROW_MAX_H, rowMin)
  const namePos: NamePos =
    style?.namePos === 'vertical' || style?.namePos === 'horizontal' || style?.namePos === 'auto'
      ? style.namePos
      : NETWORK_LAYOUT_DEFAULTS.namePos
  const stagger = style?.stagger === true
  const hideUnimportantLink = style?.hideUnimportantLink === true
  return {
    density,
    colScale: scale,
    rowScale: scale,
    colW: AOA_COL_W * scale,
    rowMin,
    rowMax,
    bandMin: h1 ?? AOA_BAND_MIN_H,
    stagger,
    namePos,
    hideUnimportantLink,
    isDefault: density === 'normal' && !stagger && h1 === null && h2 === null,
  }
}

/**
 * 基准行距：buildAoa 的 rowH 是**全局统一**的（含纵横比自适应的放大值），
 * 但横幅（band）之间留了空档，跨 band 的相邻行差是 rowH 的整数倍——
 * 所以取「**带内**相邻行最小正差」（带内行号连续，差恰为 1 个行距）；
 * 每个 band 都只有一行时无从推断，回落 fallback（缺省 AOA_ROW_H）。
 */
export function deriveRowPitch(nodes: { y: number; band?: number }[], fallback: number = AOA_ROW_H): number {
  const bands = new Map<number, Set<number>>()
  for (const n of nodes) {
    const b = n.band ?? 0
    const set = bands.get(b)
    if (set) set.add(n.y)
    else bands.set(b, new Set([n.y]))
  }
  let best = Number.POSITIVE_INFINITY
  for (const set of bands.values()) {
    const ys = [...set].sort((a, b) => a - b)
    for (let i = 1; i < ys.length; i++) {
      const d = ys[i] - ys[i - 1]
      if (d > 0 && d < best) best = d
    }
  }
  return Number.isFinite(best) ? best : fallback
}

/**
 * 布图变换：把 density（列距/行距系数）、layerHeight1/2（行距上下限 + 横幅带高下限）、
 * stagger（错层）落到节点坐标上。
 *
 * 语义：
 *  - 列：`x' = AOA_MARGIN + (x − AOA_MARGIN) × colScale`——以左边距为原点线性压缩，
 *    时标线性不破（标尺 dayW 同步取 params.colW，1 天仍是 colW 像素）；
 *  - 行：`y' = AOA_MARGIN + 有效行距/2 + 行号 × 有效行距`（与 buildAoa 的
 *    `AOA_MARGIN + rowH/2 + row × rowH` 同一条算式），有效行距 =
 *    clamp(基准行距 × rowScale, rowMin, rowMax)；
 *  - 横幅带高下限：某 band 的竖向跨度（含上下半行留白）< bandMin 时，**只在它之后的
 *    band** 整体下推补差（band 0 恒贴顶，不产生顶部空档）；
 *  - 错层：`stagger` 且**未手动 pin**（opts.manual=false）时，奇数行再抬半个有效行距——
 *    相邻事件纵向错开，减少平行箭线共线/交叉；
 *  - 全缺省（isDefault）直接返回**同一引用**（坐标逐字相等，零回归锚点）。
 */
export function applyNetworkLayout(
  graph: AoaGraph,
  params: NetworkLayoutParams,
  opts?: { manual?: boolean },
): AoaGraph {
  if (!graph.ok || graph.nodes.length === 0) return graph
  if (params.isDefault) return graph
  const stagger = params.stagger && !opts?.manual
  const base = deriveRowPitch(graph.nodes, AOA_ROW_H)
  const pitch = Math.min(params.rowMax, Math.max(params.rowMin, base * params.rowScale))
  const ys = [...new Set(graph.nodes.map((n) => n.y))].sort((a, b) => a - b)
  const rowIndexOf = new Map(ys.map((y, i) => [y, i]))
  // 行 → 归属横幅（buildAoa 的打包保证同行的 band 相同；防御性取最小 band）
  const bandOfRow = new Map<number, number>()
  for (const n of graph.nodes) {
    const r = rowIndexOf.get(n.y)!
    const prev = bandOfRow.get(r)
    bandOfRow.set(r, prev === undefined ? n.band : Math.min(prev, n.band))
  }
  const rowsOfBand = new Map<number, number[]>()
  for (const [r, b] of bandOfRow) {
    const arr = rowsOfBand.get(b)
    if (arr) arr.push(r)
    else rowsOfBand.set(b, [r])
  }
  // 逐 band 累积下推量（band 0 不补差=贴顶）
  const extra = new Map<number, number>()
  let shift = 0
  for (const rows of [...rowsOfBand.entries()].sort((p, q) => p[0] - q[0]).map(([, r]) => r)) {
    rows.sort((p, q) => p - q)
    for (const r of rows) extra.set(r, shift)
    const span = (rows[rows.length - 1] - rows[0]) * pitch + pitch // +半个行距×2 = 整带留白
    shift += Math.max(0, params.bandMin - span)
  }
  const yOf = (r: number): number =>
    AOA_MARGIN + pitch / 2 + r * pitch + (extra.get(r) ?? 0) + (stagger && r % 2 === 1 ? pitch / 2 : 0)
  const xOf = (x: number): number => AOA_MARGIN + (x - AOA_MARGIN) * params.colScale
  return {
    ...graph,
    nodes: graph.nodes.map((n) => ({ ...n, x: xOf(n.x), y: yOf(rowIndexOf.get(n.y)!) })),
  }
}

/**
 * 非重要关系线判定（斑马 is_hide_unimportant_link 的 gaea 等价物）：
 * 「非关键 + 非虚工作」的**实工作箭线**才隐藏——关键线路（含关键虚工作）与
 * 虚工作拓扑、一级汇总线全部保留，图不会因为隐藏而看不懂逻辑。
 */
export function isUnimportantLink(e: Pick<AoaEdge, 'kind' | 'critical'>): boolean {
  return e.kind === 'task' && !e.critical
}

/** 段序列包围盒（min/max），供标注落位共用 */
function segBBox(segs: Seg[]): { minX: number; maxX: number; minY: number; maxY: number } | null {
  if (segs.length === 0) return null
  let minX = Number.POSITIVE_INFINITY
  let maxX = Number.NEGATIVE_INFINITY
  let minY = Number.POSITIVE_INFINITY
  let maxY = Number.NEGATIVE_INFINITY
  for (const s of segs) {
    minX = Math.min(minX, s.x1, s.x2)
    maxX = Math.max(maxX, s.x1, s.x2)
    minY = Math.min(minY, s.y1, s.y2)
    maxY = Math.max(maxY, s.y1, s.y2)
  }
  return { minX, maxX, minY, maxY }
}

/** 箭线几何中心（标注落位用）：包围盒中心；纯竖/纯横段退化为该段中点 */
export function edgeMidpoint(segs: Seg[]): { x: number; y: number } {
  const bb = segBBox(segs)
  if (!bb) return { x: 0, y: 0 }
  return { x: (bb.minX + bb.maxX) / 2, y: (bb.minY + bb.maxY) / 2 }
}

/** 名称落位结果（rotate 缺省=不旋转；SVG 侧 `rotate(${rotate} ${x} ${y})`） */
export interface NamePlacement {
  x: number
  y: number
  anchor: 'start' | 'middle'
  rotate?: number
}

/**
 * 名称排布（斑马 name_pos）：**只**管工作名称，工期/时距标注不动。
 *  - `auto`：逐字沿用 edgeSegs 的现行落位（水平边贴线上方居中、竖边贴右侧上对齐）
 *    ——**零回归锚点**，返回的 x/y/anchor 与入参完全相等且不带 rotate；
 *  - `horizontal`：名称一律横排、以箭线几何中心水平居中（竖边取右侧 +7px 的居中位，
 *    避免文字压在竖线上）；
 *  - `vertical`：名称旋转 −90°（自下而上竖排）落在箭线几何中心——竖段/纵向排布。
 */
export function resolveNamePlacement(label: SegLabel, segs: Seg[], namePos: NamePos): NamePlacement {
  if (namePos === 'auto') return { x: label.x, y: label.y, anchor: label.anchor }
  const bb = segBBox(segs)
  if (!bb) return { x: label.x, y: label.y, anchor: label.anchor }
  const mid = edgeMidpoint(segs)
  const verticalEdge = bb.maxX - bb.minX < 2
  const x = verticalEdge ? mid.x + 7 : mid.x
  if (namePos === 'vertical') return { x, y: label.y, anchor: 'middle', rotate: -90 }
  return { x, y: label.y, anchor: 'middle' }
}

/* ── 单代号（PDM）节点尺度：nodeRadius 的唯一消费入口（W6 收尾）─────────────
 *
 * 缺口：面板能改 `nodeRadius`，但单代号视图**不消费**它（节点尺寸不动），
 * 与双代号（AoaView 的 nodeRadius=节点半径 px）口径不一致。
 *
 * 单代号的节点是**矩形盒**（不是圆点），把半径当绝对 px 会直接压坏六格标注法。
 * 所以这里把 `nodeRadius` 定义成**尺度系数**而不是绝对尺寸：
 *
 *     k = nodeRadius / 16      （16 = networkDisplay 缺省半径，k=1 ⇒ 现行视觉）
 *
 * 盒宽高、列距、行距**同源同系数**缩放 —— 只缩盒子不动间距会让节点重叠，
 * 只缩间距不动盒子会留大洞，两者必须成比例。缩放域由半径夹取域决定
 * （[4,200] ⇒ k ∈ [0.25,12.5]），因此任何输入下间距都恒为正，不会出现负值。
 *
 * 缺省零回归锚点：k=1 时四个量逐字等于基准常量（乘法 ×1 在 IEEE754 下精确），
 * 视图侧连内联样式键都不多写一个。
 * ─────────────────────────────────────────────────────────────────────── */

/** 单代号节点盒基准宽（px，k=1 时的现行视觉） */
export const PDM_NODE_W = 150
/** 单代号节点盒基准高（px） */
export const PDM_NODE_H = 92
/** 单代号列距基准（px，同列不同行/不同列之间的净空隙） */
export const PDM_COL_GAP = 74
/** 单代号行距基准（px） */
export const PDM_ROW_GAP = 40
/** 尺度基准半径：与 `networkDisplay.NETWORK_GEOMETRY_DEFAULTS.nodeRadius` 同一来源 */
export const PDM_RADIUS_BASE = NETWORK_GEOMETRY_DEFAULTS.nodeRadius
/** 半径夹取域：与 `blocks.normalizeNetworkStyle` 的 [4,200] 同口径（兜运行期内存脏值） */
export const PDM_RADIUS_MIN = 4
export const PDM_RADIUS_MAX = 200

/** 单代号布图尺度（缺省 k=1 ⇒ 基准常量原值） */
export interface PdmNodeMetrics {
  /** 尺度系数 = nodeRadius / PDM_RADIUS_BASE */
  k: number
  /** 节点盒宽（px） */
  nodeW: number
  /** 节点盒高（px） */
  nodeH: number
  /** 列距（px）：视图侧再乘 `density` 的 colScale */
  colGap: number
  /** 行距（px）：视图侧再乘 `density` 的 rowScale */
  rowGap: number
  /**
   * 是否**缺省尺度**（k=1）：true ⇒ 视图侧按现行输出渲染，不新增任何内联尺寸键
   * （节点高由 CSS 内容撑开，与改动前逐字一致）。
   */
  isDefault: boolean
}

/**
 * 半径 → 单代号布图尺度（唯一入口，视图层禁止再写第二份 150/92/74/40）。
 * 非有限值/越界一律夹进 [PDM_RADIUS_MIN, PDM_RADIUS_MAX]——宁可回到下限，
 * 也不让脏值把间距算成 0 或负数。
 */
export function resolvePdmNodeMetrics(nodeRadius?: number | null): PdmNodeMetrics {
  const r =
    typeof nodeRadius === 'number' && Number.isFinite(nodeRadius)
      ? Math.min(PDM_RADIUS_MAX, Math.max(PDM_RADIUS_MIN, nodeRadius))
      : PDM_RADIUS_BASE
  const k = r / PDM_RADIUS_BASE
  return {
    k,
    nodeW: PDM_NODE_W * k,
    nodeH: PDM_NODE_H * k,
    colGap: PDM_COL_GAP * k,
    rowGap: PDM_ROW_GAP * k,
    isDefault: k === 1,
  }
}
