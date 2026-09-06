/**
 * schedule/AoaView.tsx — 双代号网络图（AOA）视图
 *
 * 圆圈节点=事件（圈内编号，上方最早时间、下方最迟时间），箭线=工作，
 * 虚箭线=虚工作（虚线）。关键线路（含关键虚工作）红色加粗。
 * 数据来自 buildAoa（虚工作自动插入 + i<j 编号 + 时标分层布局）。
 */
import React from 'react'
import type { AoaEdge, AoaGraph } from './aoa'
import { AOA_COL_W, AOA_MARGIN, AOA_ROW_H } from './aoa'
import { useScheduleStore } from './store'
import type { SchedTask } from './types'

const R = 16

function edgeGeom(e: AoaEdge, nodeById: Map<string, { x: number; y: number }>) {
  const a = nodeById.get(e.from)!
  const b = nodeById.get(e.to)!
  const dx = b.x - a.x
  const dy = b.y - a.y
  const len = Math.hypot(dx, dy) || 1
  const shrink = (R + 3) / len
  // 直连；同列（dx≈0）时的小偏移避免完全重合
  const bow = Math.abs(dx) < 2 ? 14 : 0
  return {
    x1: a.x + dx * shrink + (bow ? -6 : 0),
    y1: a.y + dy * shrink,
    x2: b.x - dx * shrink,
    y2: b.y - dy * shrink,
    bow,
    midX: (a.x + b.x) / 2 + (bow ? 16 : 0),
    midY: (a.y + b.y) / 2 - 5,
  }
}

export const AoaView: React.FC<{ graph: AoaGraph; tasks: SchedTask[] }> = ({ graph, tasks }) => {
  const select = useScheduleStore((s) => s.select)
  const selectedId = useScheduleStore((s) => s.selectedId)

  if (!graph.ok || graph.nodes.length === 0) {
    return <div className="sched-empty">暂无任务或计划存在循环依赖，无法绘制双代号网络图</div>
  }

  const nodeById = new Map(graph.nodes.map((n) => [n.id, n]))
  const selectedEdge = selectedId ? graph.edges.find((e) => e.id === graph.taskEdge[selectedId]) : undefined
  const w = Math.max(...graph.nodes.map((n) => n.x)) + AOA_MARGIN + AOA_COL_W / 2
  const h = Math.max(...graph.nodes.map((n) => n.y)) + AOA_MARGIN + AOA_ROW_H / 2
  const taskName = (id: string) => tasks.find((t) => t.id === id)?.name ?? ''
  const critCount = graph.edges.filter((e) => e.critical && e.kind === 'task').length
  const dummyCount = graph.edges.filter((e) => e.kind === 'dummy').length

  return (
    <div className="sched-network-scroll" data-testid="sched-aoa">
      <div className="sched-network-legend">
        <span><i className="lg-line lg-critical" />关键工作</span>
        <span><i className="lg-line lg-normal" />工作</span>
        <span><i className="lg-line lg-dummy" />虚工作</span>
        <span>节点：上=最早时间 · 下=最迟时间</span>
        <span>时间单位：工作日（按日历）</span>
        {/* 进度统计牌（斑马口径：红字大数字，挂图例行右端避免遮挡节点） */}
        <div className="sched-net-badge">
          <div className="nb-item">
            <span className="nb-num">{nodeFinish(graph)}</span>
            <span className="nb-label">总工期(日)</span>
          </div>
          <div className="nb-item">
            <span className="nb-num">{critCount}</span>
            <span className="nb-label">关键工作</span>
          </div>
          <div className="nb-item">
            <span className="nb-num">{dummyCount}</span>
            <span className="nb-label">虚工作</span>
          </div>
        </div>
      </div>
      <div className="sched-network-canvas" style={{ width: w, height: h }}>
        <svg width={w} height={h} style={{ position: 'absolute', inset: 0 }}>
          <defs>
            <marker id="aoa-arrow" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto">
              <path d="M0,0 L9,4.5 L0,9 z" fill="var(--sched-link, #94a3b8)" />
            </marker>
            <marker id="aoa-arrow-crit" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto">
              <path d="M0,0 L9,4.5 L0,9 z" fill="var(--sched-critical, #dc2626)" />
            </marker>
          </defs>
          {graph.edges.map((e) => {
            const g = edgeGeom(e, nodeById)
            const dummy = e.kind === 'dummy'
            const cls = `sched-aoa-link${e.critical ? ' sched-aoa-link-critical' : ''}${dummy ? ' sched-aoa-dummy' : ''}`
            return (
              <g key={e.id}>
                <path
                  d={`M ${g.x1} ${g.y1} L ${g.x2} ${g.y2}`}
                  className={cls}
                  markerEnd={e.critical ? 'url(#aoa-arrow-crit)' : 'url(#aoa-arrow)'}
                  onClick={() => e.taskId && select(e.taskId)}
                  style={{ pointerEvents: e.taskId ? 'stroke' : 'none', cursor: e.taskId ? 'pointer' : 'default' }}
                />
                {e.label && (
                  <text x={g.midX} y={g.midY} textAnchor="middle" className={`sched-net-label${e.critical ? ' sched-net-label-critical' : ''}`}>
                    {e.label}
                  </text>
                )}
                {e.taskId && (
                  <text x={g.midX} y={g.midY + 12} textAnchor="middle" className="sched-aoa-taskname">
                    {taskName(e.taskId)}
                  </text>
                )}
              </g>
            )
          })}
        </svg>
        {graph.nodes.map((n) => (
          <div key={n.id} className="sched-aoa-node-wrap" style={{ left: n.x - R, top: n.y - R }}>
            <div className={`sched-aoa-time sched-aoa-time-es${n.es === n.ls ? '' : ' sched-aoa-time-shift'}`}>{n.es}</div>
            <div className={`sched-aoa-node${selectedEdge && (selectedEdge.from === n.id || selectedEdge.to === n.id) ? ' sched-aoa-node-hl' : ''}`}>
              {n.num}
            </div>
            <div className="sched-aoa-time sched-aoa-time-ls">{n.ls}</div>
          </div>
        ))}
      </div>
    </div>
  )
}

/** 双代号总工期 = 终点事件最早时间（编号最大事件） */
function nodeFinish(graph: { nodes: { num: number; es: number }[] }): number {
  const last = graph.nodes.reduce((a, b) => (b.num > a.num ? b : a), graph.nodes[0])
  return last?.es ?? 0
}
