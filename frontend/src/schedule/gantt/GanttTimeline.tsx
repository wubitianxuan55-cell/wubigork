/**
 * schedule/gantt/GanttTimeline.tsx — 双标尺（斑马 ZPert8 复刻 W4）
 *
 * 逆向口径：`进度计划/gaea-schedule-zpert8-uiux-re-20261114.md` §5「双标尺：上方工程标尺
 * （年-月-日）、下方进度标尺（星期）」+ §18.2「画布双标尺（月 / W 周）」；
 * `...uiux-port-plan-20261114.md` §W4「双标尺：工程标尺（年月日）+ 进度标尺（周）」。
 *
 * 结构（两行常驻，总高 = `timescale.GANTT_RULER_H`，与 GanttView 表头/画布顶部偏移同源）：
 *  · 上行「工程标尺」= 年-月跨列段（`.sched-ts-month`）+ 日/周刻度线
 *    （`.sched-ruler-tick*`，疏密读 `project.networkStyle.tickLines`）；
 *  · 下行「进度标尺」= 周层（`.sched-ts-week`，周序号）或日层（`.sched-day-col`，日号 + 星期）。
 *
 * 降级阶梯（窄窗不横向溢出）：`dayW < NARROW_RULER_DAY_W(8)` → 上行只画月段文本、不画刻度线，
 * 下行只画周序号；`dayW < WEEK_TIER_BELOW(12)` → 下行由日层切周层（沿用既有阈值）。
 *
 * 本组件零计算：分段/刻度/今日线与竣工线的横坐标全部来自 `timescale.buildRulerModel`。
 * 今日/竣工竖线在此再画一遍 = 「穿透两行标尺」（画布覆盖层的那条被 sticky 表头挡住，
 * z-index 30 vs 10，只在行区可见）。
 */
import React from 'react'
import type { RulerModel } from '../timescale'
import './ruler.css'

/** 自然日列（`GanttOverlay` 也从这里取类型，保持既有导入面）。 */
export type GanttColDate = {
  d: Date
  iso: string
  offWork: boolean
}

export interface GanttTimelineProps {
  chartW: number
  /** 双标尺模型（`buildRulerModel` 产出，组件只渲染） */
  model: RulerModel
}

export const GanttTimeline: React.FC<GanttTimelineProps> = ({ chartW, model }) => {
  const { engineering, progress, narrow, lineDensity } = model
  const rowStyle = { height: model.rowHPx }
  return (
    <div
      className={`sched-gantt-datehead sched-ruler sched-ruler-density-${lineDensity}${narrow ? ' sched-ruler-narrow' : ''}`}
      style={{ width: chartW, height: model.heightPx }}
    >
      {/* 上行：工程标尺（年-月跨列段 + 日/周刻度线） */}
      {engineering.show && (
        <div className="sched-ruler-row sched-ruler-engineering" style={rowStyle}>
          <div className="sched-ts-months sched-ruler-band" style={rowStyle}>
            {engineering.segments.map((seg) => (
              <div key={seg.key} className="sched-ts-month sched-ruler-seg" style={{ width: seg.widthPx }} title={seg.title}>
                {seg.label}
              </div>
            ))}
          </div>
          {engineering.ticks.map((tick) => (
            <span
              key={tick.key}
              className={`sched-ruler-tick sched-ruler-tick-${tick.tier}`}
              style={{ left: tick.xPx }}
              aria-hidden
            />
          ))}
        </div>
      )}
      {/* 下行：进度标尺（周层=周序号 / 日层=日号 + 星期） */}
      {progress.show && (
        <div className={`sched-ts-days sched-ruler-row sched-ruler-progress sched-ruler-tier-${progress.tier}`} style={rowStyle}>
          {progress.tier === 'week'
            ? progress.weeks.map((w) => (
                <div key={w.key} className="sched-ts-week sched-ruler-seg" style={{ width: w.widthPx }} title={w.title}>
                  {w.label}
                </div>
              ))
            : progress.days.map((c) => (
                <div
                  key={c.key}
                  className={`sched-day-col${c.offWork ? ' sched-day-off' : ''}`}
                  style={{ width: c.widthPx }}
                  title={c.title}
                >
                  <div className="sched-day-cell">
                    {c.label}
                    {c.weekday && <span className="sched-ruler-weekday">{c.weekday}</span>}
                  </div>
                </div>
              ))}
        </div>
      )}
      {/* 今日/竣工竖线：穿透两行标尺（画布区的同名线在覆盖层内，被 sticky 表头遮住） */}
      {model.todayX !== null && <span className="sched-ruler-today" style={{ left: model.todayX }} aria-hidden />}
      {model.deadlineX !== null && <span className="sched-ruler-deadline" style={{ left: model.deadlineX }} aria-hidden />}
    </div>
  )
}
