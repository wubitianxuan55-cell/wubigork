/**
 * schedule/gantt/GanttOverlay.tsx — 画布覆盖层（自 GanttView.tsx 拆分，行为零变化）
 *
 * 非工作日底纹 + 搭接箭线 + 今日线 + 目标竣工线 + 前锋线（竖向检查线/折线/圆点）。
 * 所有几何量由调用方按当前缩放/行序算好传入，本组件零计算纯渲染。
 */
import React from 'react'
import type { GanttColDate } from './GanttTimeline'

export interface GanttOverlayProps {
  chartW: number
  colDates: GanttColDate[]
  dayW: number
  totalH: number
  linkPaths: { d: string; key: string; lk: string }[]
  chainLinks: Set<string> | null
  todayCol: number | null
  deadlineCol: number | null
  deadline: string | null | undefined
  frontLine: { pts: { x: number; y: number }[]; checkX: number } | null
}

export const GanttOverlay: React.FC<GanttOverlayProps> = ({ chartW, colDates, dayW, totalH, linkPaths, chainLinks, todayCol, deadlineCol, deadline, frontLine }) => {
  return (
    <div className="sched-overlay" style={{ left: 0, width: chartW, height: totalH + 40 }}>
      <svg width={chartW} height={totalH + 40} style={{ position: 'absolute', inset: 0, pointerEvents: 'none' }}>
        <defs>
          <marker id="sched-arrow" markerWidth="7" markerHeight="7" refX="6" refY="3.5" orient="auto">
            <path d="M0,0 L7,3.5 L0,7 z" fill="var(--sched-link, #94a3b8)" />
          </marker>
        </defs>
        {colDates.map((c, i) => c.offWork && (
          <rect key={`off${i}`} x={i * dayW} y={0} width={dayW} height={totalH + 40} className="sched-weekend" />
        ))}
        {linkPaths.map((p) => (
          <path
            key={p.key}
            d={p.d}
            className={`sched-link-line${chainLinks ? (chainLinks.has(p.lk) ? ' sched-link-chain' : ' sched-link-dim') : ''}`}
            markerEnd="url(#sched-arrow)"
          />
        ))}
        {todayCol !== null && <line x1={todayCol * dayW} y1={0} x2={todayCol * dayW} y2={totalH + 40} className="sched-today-line" />}
        {deadlineCol !== null && (
          <line x1={deadlineCol * dayW + dayW} y1={0} x2={deadlineCol * dayW + dayW} y2={totalH + 40} className="sched-deadline-line">
            <title>{`目标竣工 ${deadline}（倒排校核线）`}</title>
          </line>
        )}
        {frontLine && (
          <>
            <line x1={frontLine.checkX} y1={0} x2={frontLine.checkX} y2={totalH + 40} className="sched-front-check" />
            <polyline points={frontLine.pts.map((p) => `${p.x},${p.y}`).join(' ')} className="sched-front-line" />
            {frontLine.pts.map((p, i) => <circle key={i} cx={p.x} cy={p.y} r={2.4} className="sched-front-dot" />)}
          </>
        )}
      </svg>
    </div>
  )
}