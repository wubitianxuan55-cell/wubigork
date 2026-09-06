/**
 * schedule/calendar.ts — 工作日历工具（纯函数，v4.111.0 刀2）
 *
 * 对齐 Project/斑马工作制口径：CPM 与工期全程用**工作日整数**，
 * 本模块负责「工作日序号 ↔ 日历日期」双向换算（周末按工作制集合、
 * 节假日例外命中即跳过）。扫描上限 3650 天（≈10 年）防呆。
 */
import type { SchedCalendar } from './types'

/** 默认基准日历：周一~周五为工作日（JS getDay 口径 0=周日..6=周六） */
export const DEFAULT_CALENDAR: SchedCalendar = { workweek: [1, 2, 3, 4, 5], holidays: [] }

const SCAN_LIMIT = 3650

export function normalizeCalendar(cal?: SchedCalendar): SchedCalendar {
  if (!cal || !Array.isArray(cal.workweek) || cal.workweek.length === 0) return DEFAULT_CALENDAR
  return { workweek: [...new Set(cal.workweek)].filter((n) => Number.isInteger(n) && n >= 0 && n <= 6), holidays: Array.isArray(cal.holidays) ? cal.holidays.filter((d) => /^\d{4}-\d{2}-\d{2}$/.test(d)) : [] }
}

function parseISO(iso: string): Date {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(Date.UTC(y || 1970, (m || 1) - 1, d || 1))
}

export function isoOf(d: Date): string {
  return d.toISOString().slice(0, 10)
}

export function isWorkingDate(d: Date, cal: SchedCalendar): boolean {
  if (!cal.workweek.includes(d.getUTCDay())) return false
  return !cal.holidays.includes(isoOf(d))
}

/**
 * 开工日（startISO）起第 idx 个工作日（idx=0 即开工日当日；若开工日恰为非工作日，
 * 自动顺延到首个工作日）。idx<0 返回开工日顺延前的原日期。
 */
export function wdToDate(startISO: string, idx: number, calInput?: SchedCalendar): Date {
  const cal = normalizeCalendar(calInput)
  const start = parseISO(startISO)
  if (!Number.isFinite(idx) || idx < 0) return start
  const found: Date[] = []
  let cur = new Date(start)
  // 若开工日本身非工作日，先顺延到首个工作日（它即第 0 个工作日）
  while (!isWorkingDate(cur, cal) && found.length < SCAN_LIMIT) {
    cur = new Date(cur.getTime() + 86400000)
  }
  found.push(cur)
  while (found.length < idx + 1 && found.length < SCAN_LIMIT) {
    cur = new Date(cur.getTime() + 86400000)
    if (isWorkingDate(cur, cal)) found.push(cur)
  }
  return found[Math.min(Math.max(0, idx), found.length - 1)]
}

/** 日期 → 工作日序号；非工作日（周末/节假日）或早于开工日返回 null */
export function dateToWd(startISO: string, dateISO: string, calInput?: SchedCalendar): number | null {
  const cal = normalizeCalendar(calInput)
  const start = parseISO(startISO)
  const target = parseISO(dateISO)
  if (target.getTime() < start.getTime()) return null
  let cur = new Date(start)
  let idx = -1
  while (cur.getTime() <= target.getTime() && idx < SCAN_LIMIT) {
    if (isWorkingDate(cur, cal)) {
      idx++
      if (cur.getTime() === target.getTime()) return idx
    }
    cur = new Date(cur.getTime() + 86400000)
  }
  return null
}
