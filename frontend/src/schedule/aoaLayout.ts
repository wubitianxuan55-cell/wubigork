/**
 * aoaLayout.ts — 双代号手动布局纯函数（v4.123.0 AOA 刀1）
 *
 * 设计口径（docs/gaea-schedule-aoa-manual-layout-design-2026-09.md §4.2/§4.3，
 * 「混合锚定」：pin 按事件锚点键匹配——命中→手动位，未命中/失配→自动落位；
 * buildAoa 始终裁决事件身份与时间参数，pin 通路不存在任何能让图变非法的输入）。
 * 全部纯函数零 DOM；坐标为整数像素（与 AOA_COL_W/ROW_H 同一坐标系）。
 */
import type { AoaGraph } from './aoa'
import { AOA_COL_W, AOA_ROW_H } from './aoa'
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
