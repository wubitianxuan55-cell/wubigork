/**
 * schedule/gantt/GanttTimeline.tsx — 双层时标（自 GanttView.tsx 拆分，行为零变化）
 *
 * MS Project 口径，自左向右：上行=月份跨列段；下行随缩放切换刻度——
 * dayW≥12 逐日号，<12 自然周（周一起始，跨月不断开）。日期格 nowrap，
 * 窄刻度（<10px）只画格不写数。
 */
import React from 'react'
import { WEEKDAY_LABELS } from './ganttUtil'

export interface GanttColDate {
  d: Date
  iso: string
  offWork: boolean
}

export interface GanttTimelineProps {
  chartW: number
  monthRuns: { y: number; m: number; count: number }[]
  useWeekTier: boolean
  weekRuns: { startIdx: number; count: number; label: string; title: string }[]
  dayW: number
  colDates: GanttColDate[]
}

export const GanttTimeline: React.FC<GanttTimelineProps> = ({ chartW, monthRuns, useWeekTier, weekRuns, dayW, colDates }) => {
  return (
    <div className="sched-gantt-datehead" style={{ width: chartW }}>
      <div className="sched-ts-months">
        {monthRuns.map((r, i) => {
          const w = r.count * dayW
          return (
            <div key={i} className="sched-ts-month" style={{ width: w }} title={`${r.y}年${r.m + 1}月`}>
              {w >= 76 ? `${r.y}年${r.m + 1}月` : w >= 44 ? `${r.m + 1}月` : w >= 14 ? r.m + 1 : ''}
            </div>
          )
        })}
      </div>
      {useWeekTier ? (
        <div className="sched-ts-days">
          {weekRuns.map((r, i) => (
            <div key={i} className="sched-ts-week" style={{ width: r.count * dayW }} title={r.title}>
              {r.label}
            </div>
          ))}
        </div>
      ) : (
        <div className="sched-ts-days">
          {colDates.map((c, i) => (
            <div key={i} className={`sched-day-col${c.offWork ? ' sched-day-off' : ''}`} style={{ width: dayW }} title={c.offWork ? '非工作日' : `周${WEEKDAY_LABELS[c.d.getUTCDay()]}`}>
              <div className="sched-day-cell">{dayW >= 14 || (dayW >= 10 && i % 2 === 0) ? c.d.getUTCDate() : ''}</div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}