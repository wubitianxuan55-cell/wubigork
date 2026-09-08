/**
 * schedule/aoaRuler.ts — 双代号时标视图工程标尺数据（纯函数，v4.161）
 *
 * 对标上报件（重庆干休所网络计划）时间坐标框架：顶部=工程日刻度/月/日，
 * 底部=星期/工程周。口径与 networkExport 标尺一致：列=工作日（非工作日
 * 不占列）；刻度步进按总工期 5/10/20；月标签落本月首个工作日，月界竖线
 * 贯通图面；工程周=每 7 列一档（第 N 周，档内居中）。
 * 视图（AoaView，React 渲染）与导出（networkExport，字符串渲染）共用本
 * 数据，几何= x0 + 工作日×日宽（与节点 x 同一线性刻度）。
 */
import { wdToDate } from './calendar'
import type { SchedCalendar } from './types'

export interface AoaRulerSpec {
  /** 刻度步进（工作日） */
  step: number
  dayTicks: { x: number; label: string; total: boolean }[]
  monthLabels: { x: number; label: string }[]
  /** 月界竖线 x（贯通图面，不含 wd=0） */
  monthLines: { x: number }[]
  dayLabels: { x: number; label: string }[]
  weekdays: { x: number; label: string }[]
  /** 工程周序号（档内居中） */
  weeks: { x: number; label: string }[]
}

const WEEK = ['日', '一', '二', '三', '四', '五', '六']

export function buildAoaRuler(
  total: number,
  dayW: number,
  x0: number,
  startDate: string,
  calendar?: SchedCalendar,
): AoaRulerSpec {
  const step = total > 120 ? 20 : total > 60 ? 10 : 5
  const spec: AoaRulerSpec = { step, dayTicks: [], monthLabels: [], monthLines: [], dayLabels: [], weekdays: [], weeks: [] }
  let prev: Date | null = null
  for (let wd = 0; wd <= total; wd++) {
    const d = wdToDate(startDate, wd, calendar)
    const x = x0 + wd * dayW
    if (wd % step === 0 || wd === total) spec.dayTicks.push({ x, label: String(wd), total: wd === total })
    if (prev === null || d.getUTCDate() <= prev.getUTCDate()) {
      spec.monthLabels.push({ x: x + 2, label: `${d.getUTCFullYear()}.${d.getUTCMonth() + 1}` })
      if (prev !== null) spec.monthLines.push({ x })
    }
    spec.dayLabels.push({ x, label: String(d.getUTCDate()) })
    spec.weekdays.push({ x, label: WEEK[d.getUTCDay()] })
    if (wd % 7 === 0) spec.weeks.push({ x: x + 3.5 * dayW, label: String(wd / 7 + 1) })
    prev = d
  }
  return spec
}
