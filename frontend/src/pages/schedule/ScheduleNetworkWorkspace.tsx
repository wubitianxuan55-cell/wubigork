/**
 * ScheduleNetworkWorkspace — ZPert 网络图文档双栏。
 *
 * 左：计划表格只读镜像；右：现有 PDM/AOA 画布。默认 1:1，中间 2px 分隔条，
 * 与逆向报告 §15.1 的 ZSplitter 文档结构对齐。网络算法与现有视图保持零改动。
 */
import React from 'react'
import type { AoaGraph } from '../../schedule/aoa'
import type { CpmResult, SchedProject, SchedRenderMode, SchedTask } from '../../schedule/types'
import { wbsOf } from '../../schedule/ganttGroup'
import { fmtDate, noOf } from '../../schedule/gantt/ganttUtil'
import { fmtLinkRefs } from '../../schedule/ganttLinks'
import { useScheduleStore } from '../../schedule/store'
import { AoaView } from '../../schedule/AoaView'
import { PdmView } from '../../schedule/PdmView'

const ROMAN = ['I', 'II', 'III', 'IV', 'V', 'VI', 'VII', 'VIII', 'IX', 'X', 'XI', 'XII']
const COLS = [
  { key: 'no', label: '#', w: 40 },
  { key: 'partition', label: '分区', w: 40 },
  { key: 'marker', label: '标记', w: 40 },
  { key: 'name', label: '工作名称', w: 200 },
  { key: 'dur', label: '工期(工日)', w: 100 },
  { key: 'preds', label: '前置工作', w: 90 },
  { key: 'start', label: '计划开始', w: 110 },
  { key: 'finish', label: '计划完成', w: 110 },
] as const
const TABLE_W = COLS.reduce((sum, col) => sum + col.w, 0)

function partitionOf(wbs: string | undefined): string {
  const root = Number((wbs ?? '').split('.')[0])
  return Number.isInteger(root) && root > 0 ? (ROMAN[root - 1] ?? String(root)) : ''
}

function spanOf(tasks: SchedTask[], cpm: CpmResult, index: number): { es: number; ef: number } | null {
  if (tasks[index].level !== 0) return null
  let es = Number.POSITIVE_INFINITY
  let ef = Number.NEGATIVE_INFINITY
  for (let i = index + 1; i < tasks.length && tasks[i].level > tasks[index].level; i++) {
    const row = cpm.rows[tasks[i].id]
    if (!row) continue
    es = Math.min(es, row.es)
    ef = Math.max(ef, row.ef)
  }
  return Number.isFinite(es) ? { es, ef } : null
}

interface ScheduleNetworkWorkspaceProps {
  view: 'pdm' | 'aoa'
  project: SchedProject
  cpm: CpmResult
  graph: AoaGraph
  /** 网络显示模式（斑马 display_mode；默认 timescaled=现行口径） */
  renderMode?: SchedRenderMode
}

export const ScheduleNetworkWorkspace: React.FC<ScheduleNetworkWorkspaceProps> = ({ view, project, cpm, graph, renderMode }) => {
  const selectedId = useScheduleStore((state) => state.selectedId)
  const select = useScheduleStore((state) => state.select)
  const wbs = React.useMemo(() => wbsOf(project.tasks), [project.tasks])
  const no = React.useMemo(() => noOf(project), [project])
  let color = -1
  let groupColor = 0
  const rowColors = project.tasks.map((task) => {
    if (task.level === 0) {
      color = (color + 1) % 6
      groupColor = color
      return color
    }
    return Math.max(0, groupColor)
  })

  return (
    <div className="sched-network-workspace" data-testid="sched-network-workspace">
      <div className="sched-network-tablepane" data-testid="sched-network-table">
        <div className="sched-network-table-inner" style={{ width: TABLE_W }}>
          <div className="sched-network-table-row sched-network-table-head">
            {COLS.map((col) => (
              <div key={col.key} className="sched-network-table-cell" style={{ width: col.w }}>
                {col.label}
              </div>
            ))}
          </div>
          {project.tasks.map((task, index) => {
            const row = cpm.rows[task.id]
            const span = spanOf(project.tasks, cpm, index)
            const group = task.level === 0
            const predecessors = project.links
              .filter((link) => link.to === task.id)
              .map((link) => ({ id: link.from, type: link.type, lag: link.lag }))
            return (
              <button
                key={task.id}
                type="button"
                className={`sched-network-table-row${group ? ` sched-group-row sched-group-c${rowColors[index]}` : ''}${selectedId === task.id ? ' sched-row-selected' : ''}`}
                onClick={() => select(task.id)}
              >
                <div className="sched-network-table-cell sched-network-no">{index + 1}</div>
                <div className="sched-network-table-cell sched-network-center">{partitionOf(wbs[index])}</div>
                <div className="sched-network-table-cell sched-network-center">
                  {group ? <span className="sched-marker-star">★</span> : task.isMilestone ? <span className="sched-marker-mile">◆</span> : null}
                </div>
                <div className="sched-network-table-cell sched-network-name" style={{ paddingLeft: 8 + task.level * 14 }}>
                  {group ? <strong>▸ {task.name}</strong> : task.name}
                </div>
                <div className="sched-network-table-cell">{group ? '汇总' : row ? Math.max(0, Math.round(task.duration)) : ''}</div>
                <div className="sched-network-table-cell">{group ? '' : fmtLinkRefs(predecessors, no)}</div>
                <div className="sched-network-table-cell">
                  {group
                    ? span ? fmtDate(project.startDate, span.es, project.calendar) : ''
                    : row ? fmtDate(project.startDate, row.es, project.calendar) : ''}
                </div>
                <div className="sched-network-table-cell">
                  {group
                    ? span ? fmtDate(project.startDate, span.ef, project.calendar) : ''
                    : row ? fmtDate(project.startDate, row.ef, project.calendar) : ''}
                </div>
              </button>
            )
          })}
        </div>
      </div>
      <div className="sched-network-table-split" aria-hidden="true" />
      <div className="sched-network-graphpane">
        {view === 'pdm'
          ? <PdmView project={project} cpm={cpm} />
          : <AoaView graph={graph} tasks={project.tasks} renderMode={renderMode} />}
      </div>
    </div>
  )
}
