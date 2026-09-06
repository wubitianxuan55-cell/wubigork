/**
 * schedule/ganttLinks.ts — 横道依赖线几何与前置/后续引用文本（纯函数）
 *
 * v4.127 基本功刀B：
 *  - 依赖线按四种搭接取各自锚点（FS=前完成→后开始、SS=起→起、FF=完→完、
 *    SF=前开始→后完成），不再一律画成 EF→ES 的 FS 形状；
 *  - 前置/后续列按 Project 口径格式化「行号+类型+时距」（如 3FS+2,1SS-1）。
 */
import type { LinkType } from './types'

/** 依赖线两端候选锚点（图区像素）：x1s/x1f=前置的开始/完成列，x2s/x2f=后续的开始/完成列 */
export interface LinkPts {
  x1s: number
  x1f: number
  y1: number
  x2s: number
  x2f: number
  y2: number
}

/** 按搭接类型取锚点并生成正交折线 d（横平竖直；水平不足时短桩绕行） */
export function ganttLinkPath(type: LinkType, p: LinkPts): string {
  const a = type === 'SS' || type === 'SF' ? p.x1s : p.x1f
  const b = type === 'SS' || type === 'FS' ? p.x2s : p.x2f
  return orthPath(a, p.y1, b, p.y2)
}

function orthPath(x1: number, y1: number, x2: number, y2: number): string {
  const dir = x2 >= x1 ? 1 : -1
  const end = x2 - dir * 3 // 末尾留箭头长度
  if (dir === 1 && x2 - x1 >= 20) {
    const mid = Math.max(x1 + 5, (x1 + x2) / 2)
    return `M ${x1} ${y1} H ${mid} V ${y2} H ${end}`
  }
  if (dir === -1 && x1 - x2 >= 20) {
    const mid = Math.min(x1 - 5, (x1 + x2) / 2)
    return `M ${x1} ${y1} H ${mid} V ${y2} H ${end}`
  }
  // 水平空间不足：出条短桩 → 半行错位 → 平移到箭头列 → 竖直入行（箭头朝下/上）
  const jog = y2 >= y1 ? y2 - 8 : y2 + 8
  return `M ${x1} ${y1} h ${dir * 6} V ${jog} H ${end} V ${y2}`
}

/** 前置/后续引用文本（Project 口径）：行号+类型+时距，lag=0 省略时距，如「3FS+2,1SS」 */
export function fmtLinkRefs(
  refs: { id: string; type: LinkType; lag: number }[],
  noOf: (id: string) => number,
): string {
  return refs
    .map((r) => `${noOf(r.id)}${r.type}${r.lag !== 0 ? (r.lag > 0 ? '+' : '') + r.lag : ''}`)
    .join(',')
}
