/**
 * schedule/gantt/GanttBars.tsx — 画布行（条形/基线/汇总条/拖拽层，自 GanttView.tsx 拆分，行为零变化）
 *
 * 逐可见行渲染：合成组头行=组汇总条（点击折叠/展开）；任务行=基线灰条、
 * 里程碑黑菱形、关键/手动/进度覆盖条形、拖拽期手势预览、右缘缩放把手、
 * 总时差尾巴与拖拽提示。几何量（dayNo/以像素为单位）由调用方注入，本组件纯渲染。
 */
import React from 'react'
import { Dropdown, type MenuProps } from 'antd'
import type { CpmResult, SchedProject, SchedTask } from '../types'
import { isCd } from '../cpm'
import { ROW_H, fmtDate, type GanttDragState } from './ganttUtil'
import type { GanttRow } from '../ganttGroup'

interface GanttBarsProps {
  rows: GanttRow[]
  project: SchedProject
  cpm: CpmResult
  chartW: number
  dayW: number
  dayNo: (wd: number) => number
  selectedId: string | null
  showBase: boolean
  drag: GanttDragState | null
  groupColorSeq: number[]
  rowDim: (id: string) => boolean
  onSelect: (id: string) => void
  onToggleCollapse: (key: string) => void
  rowMenu: (t: SchedTask, group: boolean) => MenuProps['items']
  onRowMenuClick: (key: string, t: SchedTask) => void
  beginDrag: (e: React.MouseEvent, t: SchedTask, kind: 'move' | 'resize') => void
}

export const GanttBars: React.FC<GanttBarsProps> = ({
  rows,
  project,
  cpm,
  chartW,
  dayW,
  dayNo,
  selectedId,
  showBase,
  drag,
  groupColorSeq,
  rowDim,
  onSelect,
  onToggleCollapse,
  rowMenu,
  onRowMenuClick,
  beginDrag,
}) => {
  const cal = project.calendar
  return (
    <>
      {(() => {
      let synSeq = -1
      return rows.map((r) => {
      if (r.kind === 'group' && !r.key.startsWith('wbs:')) {
        synSeq++
        return (
          <div
            key={r.key}
            className={`sched-gantt-row sched-group-row sched-group-c${synSeq % 6}`}
            style={{ height: ROW_H }}
            onClick={() => onToggleCollapse(r.key)}
          >
            <div className="sched-bar-lane" style={{ width: chartW }}>
              {cpm.ok && (
                <div
                  className="sched-summary-bar"
                  style={{ left: dayNo(r.minEs) * dayW, width: Math.max((dayNo(r.maxEf) - dayNo(r.minEs)) * dayW, 8) }}
                  title={`${r.label}：${r.count} 项（组汇总条）`}
                />
              )}
            </div>
          </div>
        )
      }
      const i = r.kind === 'task' ? r.idx : Number(r.key.slice(4))
      const t = project.tasks[i]
      const row = cpm.rows[t.id]
      const group = r.kind === 'group'
      const span = group ? { es: r.minEs, ef: r.maxEf } : null
      const crit = !group && t.mode !== 'manual' && row?.critical
      const manual = t.mode === 'manual'
      const dur = t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
      const colorCls = group ? ` sched-group-c${groupColorSeq[i]}` : ''
      const dimCls = !group && rowDim(t.id) ? ' sched-row-dim' : ''
      const baseRow = showBase && !group ? project.baseline?.rows[t.id] : undefined
      const isDrag = drag?.id === t.id
      const movePreview = isDrag && drag!.kind === 'move' ? drag!.preview : null
      const resizePreview = isDrag && drag!.kind === 'resize' ? drag!.preview : null
      const cdRow = isCd(t)
      // 条形宽（刀3）：cd 行=真实日历跨度 dayNo(ef)−dayNo(es)（此前把自然日
      // 数当工作日序号加会画过宽）；仅 resize 拖拽中按预览自然日直绘。
      const barWidthPx = cdRow && resizePreview != null
        ? Math.max(resizePreview * dayW, 6)
        : Math.max(
            (dayNo((movePreview ?? row.es) + (cdRow ? row.ef - row.es : (resizePreview ?? dur))) - dayNo(movePreview ?? row.es)) * dayW,
            6,
          )
      return (
        <Dropdown key={t.id} trigger={['contextMenu']} menu={{ items: rowMenu(t, group), onClick: (e) => onRowMenuClick(e.key, t) }}>
        <div
          className={`sched-gantt-row${group ? ' sched-group-row' : ''}${colorCls}${dimCls}${selectedId === t.id ? ' sched-row-selected' : ''}`}
          style={{ height: ROW_H }}
          onClick={() => onSelect(t.id)}
        >
          <div className="sched-bar-lane" style={{ width: chartW }}>
            {baseRow && (
              <div
                className="sched-baseline-bar"
                style={{ left: dayNo(baseRow.es) * dayW, width: Math.max((dayNo(baseRow.ef) - dayNo(baseRow.es)) * dayW, 5) }}
                title={row && (row.es !== baseRow.es || row.ef !== baseRow.ef)
                  ? `基线：第 ${baseRow.es}~${baseRow.ef} 工作日（当前第 ${row.es}~${row.ef}）`
                  : `基线：第 ${baseRow.es}~${baseRow.ef} 工作日（与当前一致）`}
              />
            )}
            {group ? (
              span && (
                <div
                  className="sched-summary-bar"
                  style={{ left: dayNo(span.es) * dayW, width: Math.max((dayNo(span.ef) - dayNo(span.es)) * dayW, 8) }}
                  title={`${t.name}：${span.es} ~ ${span.ef} 工作日`}
                />
              )
            ) : t.isMilestone ? (
              row && (
                <div
                  className={`sched-milestone${isDrag && drag!.kind === 'move' ? ' sched-bar-dragging' : ''}`}
                  style={{ left: dayNo(movePreview ?? row.es) * dayW - 7 }}
                  title={`${t.name}（里程碑，拖动可定位）`}
                  onMouseDown={(e) => beginDrag(e, t, 'move')}
                />
              )
            ) : (
              row && (
                <>
                  <div
                    className={`sched-bar${crit ? ' sched-bar-critical' : ''}${manual ? ' sched-bar-manual' : ''}${isDrag ? ' sched-bar-dragging' : ''}`}
                    style={{
                      left: dayNo(movePreview ?? row.es) * dayW,
                      width: barWidthPx,
                    }}
                    title={
                      cdRow
                        ? `${t.name}：日历天 ${dur}（等效 ${row.ef - row.es} 工作日，${fmtDate(project.startDate, row.es, cal)} ~ ${fmtDate(project.startDate, row.ef, cal)}）${manual ? '（手动锁定，拖动移位）' : crit ? '（关键）' : ''}`
                        : `${t.name}：第 ${row.es}~${row.ef} 工作日${manual ? '（手动锁定，拖动移位）' : crit ? '（关键，拖动=转手动锁定）' : `，总时差 ${row.tf} 天，拖动=转手动锁定`}`
                    }
                    onMouseDown={(e) => beginDrag(e, t, 'move')}
                  >
                    {t.progress > 0 && (
                      <div className="sched-bar-progress" style={{ width: `${Math.min(100, t.progress)}%` }} />
                    )}
                    <div className="sched-resize-handle" onMouseDown={(e) => beginDrag(e, t, 'resize')} title="拖动改工期" />
                  </div>
                  {!manual && !isDrag && row.tf > 0 && (
                    <div className="sched-float" style={{ left: dayNo(row.ef) * dayW, width: row.tf * dayW }} title={`总时差 ${row.tf} 天`} />
                  )}
                  {isDrag && (
                    <div
                      className="sched-drag-tip"
                      style={{ left: (cdRow && drag!.kind === 'resize' ? dayNo(row.es) + drag!.preview : dayNo(drag!.kind === 'move' ? drag!.preview : row.es + drag!.preview)) * dayW }}
                    >
                      {drag!.kind === 'move'
                        ? `第 ${row.es} → ${drag!.preview} 工作日${manual ? '' : '（转手动）'}`
                        : cdRow
                          ? `日历 ${dur} → ${drag!.preview} 天（自然日）`
                          : `工期 ${dur} → ${drag!.preview} 天`}
                    </div>
                  )}
                </>
              )
            )}
          </div>
        </div>
        </Dropdown>
      )
      }
    )
    })()}
    </>
  )
}