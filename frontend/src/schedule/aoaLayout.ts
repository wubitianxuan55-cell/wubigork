/**
 * aoaLayout.ts — 双代号手动布局纯函数（v4.123.0 AOA 刀1）
 *
 * 设计口径（docs/gaea-schedule-aoa-manual-layout-design-2026-09.md §4.2/§4.3，
 * 「混合锚定」：pin 按事件锚点键匹配——命中→手动位，未命中/失配→自动落位；
 * buildAoa 始终裁决事件身份与时间参数，pin 通路不存在任何能让图变非法的输入）。
 * 全部纯函数零 DOM；坐标为整数像素（与 AOA_COL_W/ROW_H 同一坐标系）。
 *
 * v4.130 刀H（图例语言）：G2 标注归位（名称在箭线上、工期在下，JGJ/T 121
 * 标注口径）→ edgeSegs；G4 波形线（时标网络自由时差可视化）→ segsToPath
 * 水平段按实体工期终点切波形；G1 过桥法（箭线交叉处竖线画半圆跨过）→
 * findBridgeArcs。单代号（PdmView）共用 segsToPath/findBridgeArcs（无波形）。
 */
import type { AoaEdge, AoaGraph } from './aoa'
import { AOA_COL_W, AOA_R, AOA_ROW_H } from './aoa'
import type { AoaLayout, AoaPin } from './types'

/** 防御性键数上限（agent 手写 JSON / 脏数据兜底） */
export const AOA_PINS_MAX = 5000

/** 半格吸附（55×37）：拖拽中预览即吸附位，所见即所得杜绝落点跳格 */
export function snapPt(x: number, y: number): AoaPin {
  const gx = AOA_COL_W / 2
  const gy = AOA_ROW_H / 2
  return { x: Math.round(x / gx) * gx, y: Math.round(y / gy) * gy }
}

/** 结构归一：非对象/键非有限整数坐标 → 丢弃该键；键数截断至上限 */
export function normalizeAoaLayout(raw: unknown): AoaLayout {
  const pins: Record<string, AoaPin> = {}
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
    const rawPins = (raw as { pins?: unknown }).pins
    if (rawPins && typeof rawPins === 'object' && !Array.isArray(rawPins)) {
      for (const [key, v] of Object.entries(rawPins as Record<string, unknown>)) {
        if (Object.keys(pins).length >= AOA_PINS_MAX) break
        if (!key || typeof v !== 'object' || v === null) continue
        const { x, y } = v as Record<string, unknown>
        if (typeof x !== 'number' || typeof y !== 'number' || !Number.isFinite(x) || !Number.isFinite(y)) continue
        pins[key] = { x: Math.round(x), y: Math.round(y) }
      }
    }
  }
  return { pins }
}

/**
 * pin 覆盖：命中锚点的事件用手动位，未命中事件保留 buildAoa 自动位（混合共存）。
 * 返回新 graph（节点克隆），edges/taskEdge/时间参数原样——时间参数仍全量显示，
 * 只是手动模式下 x 不再编码时间轴（自由坐标口径，工具栏明示）。
 */
export function applyPins(graph: AoaGraph, pins: Record<string, AoaPin>): AoaGraph {
  if (Object.keys(pins).length === 0) return graph
  return {
    ...graph,
    nodes: graph.nodes.map((n) => {
      const pin = pins[n.anchor]
      return pin ? { ...n, x: pin.x, y: pin.y } : n
    }),
  }
}

/** 提交时剪枝：以当前图锚点集为准清理失配键（pins 不无限膨胀） */
export function prunePins(pins: Record<string, AoaPin>, graph: AoaGraph): Record<string, AoaPin> {
  const anchors = new Set(graph.nodes.map((n) => n.anchor))
  const out: Record<string, AoaPin> = {}
  for (const [key, pin] of Object.entries(pins)) {
    if (anchors.has(key)) out[key] = pin
  }
  return out
}

// ── 刀H 图例语言（G1/G2/G4）──────────────────────────────────

/** 正交线段（水平：y1=y2；竖直：x1=x2） */
export interface Seg {
  x1: number
  y1: number
  x2: number
  y2: number
}

/** 箭线文字标注位（anchor 与 SVG textAnchor 对应） */
export interface SegLabel {
  x: number
  y: number
  anchor: 'start' | 'middle'
}

export interface EdgeGeom {
  segs: Seg[]
  /** 工作名称（G2：箭线上方） */
  name: SegLabel
  /** 持续时间/时距（G2：箭线下方） */
  dur: SegLabel
}

/**
 * 箭线正交几何（自 AoaView.edgeGeom 迁入并改产出段序列，供过桥/波形二次加工）。
 * 前进边：右缘出、直角拐、左缘入；同列边：竖直直连；后退边（手动布局可能产生）：
 * 底缘出、下方通道绕行、底缘入。同节点对多条平行边按序错开通道防重叠。
 * waveFromX=实体工期终点 x（时标 auto 模式）：水平标注居中于实体段而非波形段；
 * 传 null（手动布局）退化为按全段居中。
 */
export function edgeSegs(
  e: AoaEdge,
  nodeById: Map<string, { x: number; y: number }>,
  lane: { idx: number; cnt: number },
  waveFromX?: number | null,
): EdgeGeom {
  const a = nodeById.get(e.from)!
  const b = nodeById.get(e.to)!
  const dx = b.x - a.x
  const dy = b.y - a.y
  // 平行边错位：同 (from,to) 多条边（如虚工作+实工作）各让 10px
  const off = lane.cnt > 1 ? (lane.idx - (lane.cnt - 1) / 2) * 10 : 0
  if (Math.abs(dx) < 2) {
    // 同列：竖直边（虚工作常态）；并行时横向微移防重合；文字右移竖排堆叠（名称上/工期下）
    const x = a.x + off
    const up = b.y < a.y
    const y1 = up ? a.y - AOA_R : a.y + AOA_R
    const y2 = up ? b.y + AOA_R : b.y - AOA_R
    const my = (y1 + y2) / 2
    return {
      segs: [{ x1: x, y1, x2: x, y2 }],
      name: { x: x + 7, y: my, anchor: 'start' },
      dur: { x: x + 7, y: my + 12, anchor: 'start' },
    }
  }
  if (dx > 0) {
    const x1 = a.x + AOA_R
    const x2 = b.x - AOA_R
    if (Math.abs(dy) < 2) {
      const y = a.y + off
      // 波形线存在时标注居中于实体段（工期段），不落在波形上
      const solidEnd = waveFromX != null ? Math.min(x2, Math.max(x1, waveFromX)) : x2
      const mid = (x1 + solidEnd) / 2
      return {
        segs: [{ x1, y1: y, x2, y2: y }],
        name: { x: mid, y: y - 5, anchor: 'middle' },
        dur: { x: mid, y: y + 7, anchor: 'middle' },
      }
    }
    // H-V-H：通道 x 错开并行边；标注贴第一段水平线，过短则贴末段
    const mid = (x1 + x2) / 2 + off
    const firstLen = mid - x1
    const onFirst = firstLen >= 36
    const lx = onFirst ? (x1 + mid) / 2 : (mid + x2) / 2
    const ly = (onFirst ? a.y : b.y) - 5
    return {
      segs: [
        { x1, y1: a.y, x2: mid, y2: a.y },
        { x1: mid, y1: a.y, x2: mid, y2: b.y },
        { x1: mid, y1: b.y, x2, y2: b.y },
      ],
      name: { x: lx, y: ly, anchor: 'middle' },
      dur: { x: lx, y: ly + 12, anchor: 'middle' },
    }
  }
  // 后退边（手动布局）：底缘出 → 下方通道 → 底缘入（箭头朝上）
  const laneY = Math.max(a.y, b.y) + AOA_R + 14 + off
  const mx = (a.x + b.x) / 2
  return {
    segs: [
      { x1: a.x, y1: a.y + AOA_R, x2: a.x, y2: laneY },
      { x1: a.x, y1: laneY, x2: b.x, y2: laneY },
      { x1: b.x, y1: laneY, x2: b.x, y2: b.y + AOA_R },
    ],
    name: { x: mx, y: laneY - 4, anchor: 'middle' },
    dur: { x: mx, y: laneY + 10, anchor: 'middle' },
  }
}

/** G1 过桥法半圆半径（px） */
export const AOA_BRIDGE_R = 5

/**
 * G1 过桥法桥位检测：竖直段垂直穿越「他边」水平段处竖线画半圆跨过
 * （JGJ/T 121：交叉宜避免，不可避免用过桥法/指向法）。水平段=时标轴不断，
 * 一律竖线让桥；H×H/V×V 平行重合不属过桥范畴。严格内交（端点相触=拐角/入出
 * 节点不算）；同段多桥按 y 升序、间距 <2R+4 丢弃后者；距段端 <R+1 不设桥
 * （半圆须完整落在段内）。返回 边序号 → (段序号 → 升序桥位 y)。
 */
export function findBridgeArcs(segsByEdge: Seg[][]): Map<number, Map<number, number[]>> {
  const hs: { e: number; y: number; x1: number; x2: number }[] = []
  segsByEdge.forEach((segs, e) =>
    segs.forEach((s) => {
      if (s.y1 === s.y2) hs.push({ e, y: s.y1, x1: Math.min(s.x1, s.x2), x2: Math.max(s.x1, s.x2) })
    }),
  )
  const out = new Map<number, Map<number, number[]>>()
  segsByEdge.forEach((segs, e) =>
    segs.forEach((s, si) => {
      if (s.x1 !== s.x2) return
      const yTop = Math.min(s.y1, s.y2)
      const yBot = Math.max(s.y1, s.y2)
      const ys: number[] = []
      for (const h of hs) {
        if (h.e === e) continue
        if (h.y > yTop && h.y < yBot && s.x1 > h.x1 && s.x1 < h.x2) ys.push(h.y)
      }
      ys.sort((p, q) => p - q)
      const kept: number[] = []
      for (const y of ys) {
        if (y - yTop < AOA_BRIDGE_R + 1 || yBot - y < AOA_BRIDGE_R + 1) continue
        if (kept.length > 0 && y - kept[kept.length - 1] < AOA_BRIDGE_R * 2 + 4) continue
        kept.push(y)
      }
      if (kept.length > 0) {
        let inner = out.get(e)
        if (!inner) { inner = new Map(); out.set(e, inner) }
        inner.set(si, kept)
      }
    }),
  )
  return out
}

/**
 * 单边段序列 → path d。G4 波形线（时标网络图例语言：自由时差以波形线表示，
 * 虚工作有时差时加波形线）：右行水平段超出 waveFromX（实体工期终点 x）的
 * 尾段画波形，<6px 不足半个波则直连；waveFromX=null（手动布局，x 与时间
 * 解耦）不画波形。G1 过桥：bridges=本边 段序号→桥位 y（findBridgeArcs 内层），
 * 竖段在桥位画半圆（下行 sweep=1、上行 sweep=0，统一右凸）。
 */
export function segsToPath(segs: Seg[], waveFromX: number | null, bridges?: Map<number, number[]>): string {
  const parts: string[] = [`M ${segs[0].x1} ${segs[0].y1}`]
  segs.forEach((s, si) => {
    if (s.x1 === s.x2) {
      const down = s.y2 >= s.y1
      let cur = s.y1
      for (const yc of bridges?.get(si) ?? []) {
        const enter = down ? yc - AOA_BRIDGE_R : yc + AOA_BRIDGE_R
        const leave = down ? yc + AOA_BRIDGE_R : yc - AOA_BRIDGE_R
        parts.push(`L ${s.x1} ${enter}`)
        parts.push(`A ${AOA_BRIDGE_R} ${AOA_BRIDGE_R} 0 0 ${down ? 1 : 0} ${s.x1} ${leave}`)
        cur = leave
      }
      parts.push(`L ${s.x1} ${cur}`)
      parts.push(`L ${s.x1} ${s.y2}`)
    } else if (waveFromX != null && s.x2 > s.x1 && s.x2 > waveFromX) {
      // 水平段切波形：实体段直连到 waveFromX，其后尾段波形（自由时差）
      const solidEnd = Math.max(s.x1, Math.min(waveFromX, s.x2))
      if (solidEnd > s.x1) parts.push(`L ${solidEnd} ${s.y1}`)
      const w = s.x2 - solidEnd
      if (w >= 6) parts.push(waveD(w))
      parts.push(`L ${s.x2} ${s.y2}`)
    } else {
      parts.push(`L ${s.x1} ${s.y1}`)
      parts.push(`L ${s.x2} ${s.y2}`)
    }
  })
  return parts.join(' ')
}

/** 波形线 d：等宽半波交替成 ⌒⌓⌒⌓ 正弦形，相对坐标接续当前点、终点恰落段尾 */
function waveD(len: number): string {
  const pairs = Math.max(1, Math.round(len / 16))
  const half = len / (pairs * 2)
  let d = ''
  for (let i = 0; i < pairs * 2; i++) {
    const w = Math.round(half * 50) / 100
    d += `q ${w} ${i % 2 === 0 ? -4 : 4} ${w} 0 `
  }
  return d.trimEnd()
}
