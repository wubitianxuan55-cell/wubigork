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

/**
 * 目标竣工日期 → 目标总工期（工作日）：[开工日, 竣工日] 内的工作日计数
 * （竣工日恰为非工作日时自然回落到此前最近工作日，即「第 count 个工作日竣工」）。
 * 竣工早于开工 → 0（必然不可达，由倒排校核裁决报告）。
 */
export function deadlineWorkdays(startISO: string, deadlineISO: string, calInput?: SchedCalendar): number {
  const cal = normalizeCalendar(calInput)
  const start = parseISO(startISO).getTime()
  const dl = parseISO(deadlineISO).getTime()
  if (dl < start) return 0
  let count = 0
  for (let t = start; t <= dl && count < SCAN_LIMIT; t += 86400000) {
    if (isWorkingDate(new Date(t), cal)) count++
  }
  return count
}

/**
 * cd（日历天）任务正推换算（v4.150 双工期刀1）：完成边界 = 首个 ≥（es 所在
 * 工作日 + cd 自然日）的工作日序号（ceil 吸附：边界落在周末/节假日即顺延到
 * 其后第一个工作日；边界恰为工作日即当日；cd≤0 → es）。扫描上限与 wdToDate
 * 同源（3650 天防呆，越界截断）。与 Go 侧 calendar.go CdToEf 互为镜像。
 */
export function cdToEf(startISO: string, es: number, cd: number, calInput?: SchedCalendar): number {
  const cal = normalizeCalendar(calInput)
  const es0 = Number.isFinite(es) && es > 0 ? Math.round(es) : 0
  const days = Number.isFinite(cd) && cd > 0 ? Math.round(cd) : 0
  const anchor = wdToDate(startISO, es0, cal).getTime()
  const boundary = anchor + days * 86400000
  let idx = es0
  let cur = anchor
  for (let steps = 0; steps < SCAN_LIMIT && (cur < boundary || !isWorkingDate(new Date(cur), cal)); steps++) {
    cur += 86400000
    if (isWorkingDate(new Date(cur), cal)) idx++
  }
  return idx
}

/**
 * cd（日历天）任务逆推换算：最大的工作日序号 s 使 fwd(s) ≤ lf（fwd=cdToEf，
 * 单调不减且有平段）。镜像回退：lf 所在工作日回退 cd 自然日、向下吸附到
 * 工作日，再有界双侧校正至满足性质 fwd(s) ≤ lf < fwd(s+1)；下限截到 0
 * （负时差极端场景，两侧镜像一致）。与 Go 侧 calendar.go CdLatestStart 互为镜像。
 */
export function cdLatestStart(startISO: string, lf: number, cd: number, calInput?: SchedCalendar): number {
  const cal = normalizeCalendar(calInput)
  const days = Number.isFinite(cd) && cd > 0 ? Math.round(cd) : 0
  const lf0 = Number.isFinite(lf) && lf > 0 ? Math.round(lf) : 0
  const start = parseISO(startISO).getTime()
  const boundary = wdToDate(startISO, lf0, cal).getTime() - days * 86400000
  let cur = boundary
  while (cur > start && !isWorkingDate(new Date(cur), cal)) cur -= 86400000
  const idx = dateToWd(startISO, isoOf(new Date(cur)), cal)
  let s = idx != null && idx > 0 ? idx : 0
  for (let i = 0; i < SCAN_LIMIT && s > 0 && cdToEf(startISO, s, days, cal) > lf0; i++) s--
  for (let i = 0; i < SCAN_LIMIT && cdToEf(startISO, s + 1, days, cal) <= lf0; i++) s++
  return s
}

/**
 * cd（日历天）任务正推单调逆查（v4.155 双工期全搭接放开）：最小工作日序号 s
 * 使 fwd(s) ≥ targetEf（fwd=cdToEf，正推单调不减且有平段；FF/SF-to-cd 的最早
 * 开工换算用）。口径：正推单调逆查，与 cdToEf/cdLatestStart 互为镜像三件套；
 * Go 侧 calendar.go CdEarliestStart 互为镜像。实现镜像 cdLatestStart 风格：
 * 估锚 s0 = max(0, targetEf − cd)，再有界向下/向上双侧校正至满足性质
 * fwd(s) ≥ targetEf 且（s=0 或 fwd(s−1) < targetEf）；targetEf ≤ 0 直接返回 0。
 */
export function cdEarliestStart(startISO: string, targetEf: number, cd: number, calInput?: SchedCalendar): number {
  const cal = normalizeCalendar(calInput)
  const target = Number.isFinite(targetEf) && targetEf > 0 ? Math.round(targetEf) : 0
  if (target <= 0) return 0
  const days = Number.isFinite(cd) && cd > 0 ? Math.round(cd) : 0
  let s = Math.max(0, target - days)
  for (let i = 0; i < SCAN_LIMIT && s > 0 && cdToEf(startISO, s - 1, days, cal) >= target; i++) s--
  for (let i = 0; i < SCAN_LIMIT && cdToEf(startISO, s, days, cal) < target; i++) s++
  return s
}
