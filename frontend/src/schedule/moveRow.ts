/**
 * moveRow.ts — 大纲树形拖动的纯计划器（斑马「树形拖动」对标）
 *
 * 两级大纲（level 0=分区、1=工作）下的三种落点：
 *  - `before` / `after`：与目标行同级，插到其前/后；
 *  - `inside`：成为目标分区的子工作（只有分区能接收）。
 *
 * 约束（拒绝时给原因，调用方直接展示，不落库）：
 *  1. 不能拖到自己身上，也不能落到自己的子孙里（会把子树弄丢）；
 *  2. 分区不能塞进分区（两级大纲）；
 *  3. 分区只能与分区同级移动，工作只能与工作同级移动；
 *  4. `inside` 只对分区目标有效。
 *
 * 拖分区时**整棵子树一起搬**（紧邻的连续 level 1 行），搬完子行仍是 level 1。
 * 纯函数：入参不被修改，返回新数组。
 */
import type { SchedTask } from './types'

export type DropZone = 'before' | 'after' | 'inside'

export interface RowMovePlan {
  ok: boolean
  tasks?: SchedTask[]
  reason?: string
}

/** 取某行的子树范围：分区=其后连续的 level 1 行；工作=只有自己 */
function subtreeRange(tasks: SchedTask[], idx: number): { start: number; end: number } {
  const t = tasks[idx]
  if (!t || t.level !== 0) return { start: idx, end: idx + 1 }
  let end = idx + 1
  while (end < tasks.length && tasks[end].level === 1) end++
  return { start: idx, end }
}

export function planRowMove(
  tasks: SchedTask[],
  dragId: string,
  targetId: string,
  zone: DropZone,
): RowMovePlan {
  const from = tasks.findIndex((t) => t.id === dragId)
  const to = tasks.findIndex((t) => t.id === targetId)
  if (from < 0 || to < 0) return { ok: false, reason: '拖动行或目标行不存在' }
  if (from === to) return { ok: false, reason: '原地未移动' }

  const drag = tasks[from]
  const target = tasks[to]
  const range = subtreeRange(tasks, from)
  if (to >= range.start && to < range.end) {
    return { ok: false, reason: '不能把分区拖进它自己的子工作里' }
  }

  if (zone === 'inside') {
    if (target.level !== 0) return { ok: false, reason: '只有分区行可以接收子工作' }
    if (drag.level === 0) return { ok: false, reason: '分区不能塞进分区（两级大纲）' }
  } else {
    if (drag.level !== target.level) {
      return {
        ok: false,
        reason: drag.level === 0 ? '分区只能与分区同级移动' : '工作只能与工作同级移动（要换分区请拖到分区行中段）',
      }
    }
  }

  const moving = tasks.slice(range.start, range.end).map((t) => ({ ...t }))
  const rest = tasks.filter((_, i) => i < range.start || i >= range.end)
  const targetIdx = rest.findIndex((t) => t.id === targetId)
  if (targetIdx < 0) return { ok: false, reason: '目标行已被一并移走' }

  // 「之后」按目标的**子树末尾**算：拖到分组行下方时，要落在它现有子工作之后，
  // 而不是插到分区行与它的子工作之间（那样会把目标分区分家）
  const targetRange = subtreeRange(rest, targetIdx)
  let insertAt: number
  if (zone === 'before') insertAt = targetIdx
  else if (zone === 'after') insertAt = targetRange.end
  else insertAt = targetRange.end // inside：成为目标分区最后一个子工作
  const nextLevel = zone === 'inside' ? 1 : moving[0].level
  const placed = moving.map((t, i) => ({ ...t, level: i === 0 ? nextLevel : nextLevel + 1 }))
  const out = [...rest.slice(0, insertAt), ...placed, ...rest.slice(insertAt)]
  return { ok: true, tasks: out }
}
