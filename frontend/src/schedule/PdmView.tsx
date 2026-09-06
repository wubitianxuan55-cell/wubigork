/**
 * schedule/PdmView.tsx — 单代号网络图（PDM/AON）视图
 *
 * 节点=工作（六格图：名称 + ES/D/EF + LS/TF/LF），箭线=搭接（FS/SS/FF/SF+时距）。
 * 布局复用时标分层（layerByTime）；关键工作与关键箭线红色。
 */
import React, { useMemo } from 'react'
import type { CpmResult, SchedProject } from './types'
import { layerByTime } from './layout'
import { useScheduleStore } from './store'

const NODE_W = 150
const NODE_H = 92
const COL_GAP = 74
const ROW_GAP = 40
const MARGIN = 28

export const PdmView: React.FC<{ project: SchedProject; cpm: CpmResult }> = ({ project, cpm }) => {
  const selectedId = useScheduleStore((s) => s.selectedId)
  const select = useScheduleStore((s) => s.select)

  const leaves = useMemo(() => project.tasks.filter((t) => t.level > 0), [project.tasks])

  const layout = useMemo(() => {
    if (leaves.length === 0 || !cpm.ok) return null
    const pos = layerByTime(
      leaves.map((t) => ({ id: t.id, es: cpm.rows[t.id]?.es ?? 0 })),
      project.links.map((l) => ({ from: l.from, to: l.to })),
    )
    const maxCol = Math.max(0, ...[...pos.values()].map((p) => p.col))
    const maxRow = Math.max(0, ...[...pos.values()].map((p) => p.row))
    return {
      pos,
      w: MARGIN * 2 + (maxCol + 1) * (NODE_W + COL_GAP),
      h: MARGIN * 2 + (maxRow + 1) * (NODE_H + ROW_GAP),
      nodeX: (id: string) => MARGIN + pos.get(id)!.col * (NODE_W + COL_GAP),
      nodeY: (id: string) => MARGIN + pos.get(id)!.row * (NODE_H + ROW_GAP),
    }
  }, [leaves, project.links, cpm])

  if (!layout) {
    return <div className="sched-empty">暂无任务或计划存在循环依赖，无法绘制单代号网络图</div>
  }

  const links = project.links.filter((l) => layout.pos.has(l.from) && layout.pos.has(l.to))

  return (
    <div className="sched-network-scroll" data-testid="sched-pdm">
      <div className="sched-network-legend">
        <span><i className="lg-dot lg-critical" />关键工作</span>
        <span><i className="lg-dot lg-normal" />非关键</span>
        <span>格：ES / 工期 / EF · LS / 总时差 / LF</span>
      </div>
      <div className="sched-network-canvas" style={{ width: layout.w, height: layout.h }}>
        <svg width={layout.w} height={layout.h} style={{ position: 'absolute', inset: 0 }}>
          <defs>
            <marker id="pdm-arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
              <path d="M0,0 L8,4 L0,8 z" fill="var(--sched-link, #94a3b8)" />
            </marker>
            <marker id="pdm-arrow-crit" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
              <path d="M0,0 L8,4 L0,8 z" fill="var(--sched-critical, #dc2626)" />
            </marker>
          </defs>
          {links.map((l, i) => {
            const crit = cpm.rows[l.from]?.critical && cpm.rows[l.to]?.critical && l.type === 'FS' && l.lag === 0
            const x1 = layout.nodeX(l.from) + NODE_W
            const y1 = layout.nodeY(l.from) + NODE_H / 2
            const x2 = layout.nodeX(l.to)
            const y2 = layout.nodeY(l.to) + NODE_H / 2
            const midX = Math.max(x1 + 8, (x1 + x2) / 2)
            const d = `M ${x1} ${y1} H ${midX} V ${y2} H ${x2 - 4}`
            const label = `${l.type}${l.lag !== 0 ? (l.lag > 0 ? '+' : '') + l.lag : ''}`
            return (
              <g key={i}>
                <path d={d} className={`sched-net-link${crit ? ' sched-net-link-critical' : ''}`} markerEnd={crit ? 'url(#pdm-arrow-crit)' : 'url(#pdm-arrow)'} />
                <text x={midX} y={(y1 + y2) / 2 - 4} textAnchor="middle" className={`sched-net-label${crit ? ' sched-net-label-critical' : ''}`}>{label}</text>
              </g>
            )
          })}
        </svg>
        {leaves.map((t) => {
          const row = cpm.rows[t.id]
          if (!row) return null
          const crit = row.critical
          return (
            <div
              key={t.id}
              className={`sched-pdm-node${crit ? ' sched-pdm-node-critical' : ''}${selectedId === t.id ? ' sched-pdm-node-selected' : ''}`}
              style={{ left: layout.nodeX(t.id), top: layout.nodeY(t.id), width: NODE_W }}
              onClick={() => select(t.id)}
              title={`总时差 ${row.tf} 天 · 自由时差 ${row.ff} 天`}
            >
              <div className="sched-pdm-name">{t.isMilestone ? <>◆ {t.name}</> : t.name}</div>
              <div className="sched-pdm-grid">
                <span>{row.es}</span>
                <span className="sched-dim">{t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))}</span>
                <span>{row.ef}</span>
                <span>{row.ls}</span>
                <span className={crit ? 'sched-critical-text' : 'sched-dim'}>{row.tf}</span>
                <span>{row.lf}</span>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
