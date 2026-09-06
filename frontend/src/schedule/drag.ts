/**
 * drag.ts — 横道拖拽的落点换算（纯函数，v4.118.0 刀9）
 *
 * 语义（与 Project 拖动加约束同源，诚实口径）：
 *  - 拖移（move）：auto 任务=转 manual 并锁定开始到落点（manual 任务=更新
 *    manualStart）。auto 的位置由搭接决定，拖完会被 CPM 拉回——转手动是
 *    唯一不骗人的做法；
 *  - 缩放（resize）：右缘拖动改工期（es 不动），里程碑不可缩放（工期恒 0）。
 * 落点全部按**最近工作日**吸附（自然日像素 → 工作日序号），跨周末/节假日
 * 自动吸附到相邻工作日（等距取左）。
 */
import type { SchedCalendar } from './types'
import { normalizeCalendar, wdToDate } from './calendar'

/**
 * 工作日序号 → 自然日偏移 反查表（升序单调）：开工日起逐个工作日，
 * 值=该工作日在自然日轴上的列偏移。横道像素位移靠它换算回工作日。
 */
export function workdayOffsets(startISO: string, count: number, calInput?: SchedCalendar): number[] {
  const cal = normalizeCalendar(calInput)
  const out: number[] = []
  for (let wd = 0; wd < count; wd++) {
    const d = wdToDate(startISO, wd, cal)
    out.push(Math.round((d.getTime() - Date.parse(`${startISO}T00:00:00Z`)) / 86400000))
  }
  return out
}

/** 自然日偏移 → 最近工作日序号（钳位 0..last；空表返回 0；等距取左） */
export function nearestWd(offsets: number[], naturalOffset: number): number {
  if (offsets.length === 0) return 0
  if (naturalOffset <= offsets[0]) return 0
  if (naturalOffset >= offsets[offsets.length - 1]) return offsets.length - 1
  let lo = 0
  let hi = offsets.length - 1
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (offsets[mid] < naturalOffset) lo = mid + 1
    else hi = mid
  }
  if (offsets[lo] === naturalOffset) return lo
  const left = offsets[lo - 1]
  const right = offsets[lo]
  return naturalOffset - left <= right - naturalOffset ? lo - 1 : lo
}

/** 拖移落点：条形左缘当前自然日偏移 + 像素位移 → 目标 manualStart（工作日） */
export function dropToWd(offsets: number[], startNaturalOffset: number, dxPx: number, dayW: number): number {
  return nearestWd(offsets, startNaturalOffset + Math.round(dxPx / dayW))
}

/** 缩放落点：右缘自然日偏移 + 像素位移 → 新工期（es 固定，钳位 ≥0） */
export function resizeToDuration(offsets: number[], esWd: number, rightNaturalOffset: number, dxPx: number, dayW: number): number {
  const newEf = nearestWd(offsets, rightNaturalOffset + Math.round(dxPx / dayW))
  return Math.max(0, newEf - esWd)
}
