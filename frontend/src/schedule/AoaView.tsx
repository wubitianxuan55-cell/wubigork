/**
 * schedule/AoaView.tsx — 双代号网络图（AOA）视图
 *
 * 圆圈节点=事件（圈内编号，上方最早时间、下方最迟时间），箭线=工作，
 * 虚箭线=虚工作（虚线）。关键线路（含关键虚工作）红色加粗。
 * 数据来自 buildAoa（虚工作自动插入 + i<j 编号 + 时标分层布局）。
 *
 * v4.123 AOA 刀1：手动布局双模式——「自动」=时标分层（现状零改动）；
 * 「手动」= pins 覆盖（applyPins 混合锚定：命中手动位/新事件自动落位），
 * x 与时间解耦（时间参数仍全量显示）。拖拽布点对齐横道刀9 纪律：
 * window 监听三件套、半格吸附纯函数（aoaLayout.ts snapPt）、未移动不提交、
 * Esc 取消、自动模式拖拽=自动转手动（「不骗人」先例）、提交时剪枝失配键。
 */
import React, { useRef, useState } from 'react'
import type { AoaEdge, AoaGraph } from './aoa'
import { AOA_COL_W, AOA_MARGIN, AOA_ROW_H } from './aoa'
import { applyPins, prunePins, snapPt } from './aoaLayout'
import { useScheduleStore } from './store'
import type { AoaPin, SchedTask } from './types'

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
  const aoaLayout = useScheduleStore((s) => s.project.aoaLayout)
  const setAoaPins = useScheduleStore((s) => s.setAoaPins)
  const pins = aoaLayout?.pins ?? {} // 派生放渲染体：selector 必须返回稳定引用（getSnapshot 缓存纪律）
  // 布局开关是视图态（mode 不入文件）：默认 pins 非空→手动
  const [mode, setMode] = useState<'auto' | 'manual'>(() => (Object.keys(pins).length > 0 ? 'manual' : 'auto'))
  // 拖拽中状态：只动本体（节点跟随+吸附位 tip），提交在 mouseup
  const [drag, setDrag] = useState<{ anchor: string; x: number; y: number; moved: boolean } | null>(null)
  const dragRef = useRef<{ anchor: string; startX: number; startY: number; orig: { x: number; y: number }; x: number; y: number; moved: boolean } | null>(null)

  if (!graph.ok || graph.nodes.length === 0) {
    return <div className="sched-empty">暂无任务或计划存在循环依赖，无法绘制双代号网络图</div>
  }

  const manual = mode === 'manual'
  const shown = manual ? applyPins(graph, pins) : graph
  const nodeById = new Map(shown.nodes.map((n) => [n.id, n]))
  const anchorById = new Map(graph.nodes.map((n) => [n.id, n.anchor]))
  const selectedEdge = selectedId ? graph.edges.find((e) => e.id === graph.taskEdge[selectedId]) : undefined
  const w = Math.max(...shown.nodes.map((n) => n.x)) + AOA_MARGIN + AOA_COL_W / 2
  const h = Math.max(...shown.nodes.map((n) => n.y)) + AOA_MARGIN + AOA_ROW_H / 2
  const taskName = (id: string) => tasks.find((t) => t.id === id)?.name ?? ''
  const critCount = graph.edges.filter((e) => e.critical && e.kind === 'task').length
  const dummyCount = graph.edges.filter((e) => e.kind === 'dummy').length

  /** 拖拽提交：吸附落点写 pin + 剪枝失配键；自动模式拖拽=转手动（不骗人先例） */
  const commitPin = (anchor: string, pin: AoaPin) => {
    const next = { ...pins, [anchor]: pin }
    setAoaPins(prunePins(next, graph))
    if (!manual) setMode('manual')
  }

  const beginDrag = (e: React.MouseEvent, nodeId: string) => {
    const node = nodeById.get(nodeId)
    if (!node) return
    e.preventDefault()
    e.stopPropagation()
    const anchor = anchorById.get(nodeId)!
    dragRef.current = { anchor, startX: e.clientX, startY: e.clientY, orig: { x: node.x, y: node.y }, x: node.x, y: node.y, moved: false }
    const onMove = (ev: MouseEvent) => {
      const d = dragRef.current
      if (!d) return
      const snapped = snapPt(d.orig.x + (ev.clientX - d.startX), d.orig.y + (ev.clientY - d.startY))
      const moved = d.orig.x !== snapped.x || d.orig.y !== snapped.y
      dragRef.current = { ...d, x: snapped.x, y: snapped.y, moved }
      setDrag({ anchor, x: snapped.x, y: snapped.y, moved })
    }
    const onUp = () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      window.removeEventListener('keydown', onKey)
      const d = dragRef.current
      dragRef.current = null
      setDrag(null)
      if (d?.moved) commitPin(d.anchor, { x: d.x, y: d.y }) // 提交在渲染期外（updater 不得带副作用）
    }
    const onKey = (ev: KeyboardEvent) => {
      if (ev.key === 'Escape') {
        // Esc 取消：不提交，回原位（回调内自清理，避免与 onUp 双跑）
        window.removeEventListener('mousemove', onMove)
        window.removeEventListener('mouseup', onUp)
        window.removeEventListener('keydown', onKey)
        dragRef.current = null
        setDrag(null)
      }
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    window.addEventListener('keydown', onKey)
  }

  return (
    <div className="sched-network-scroll" data-testid="sched-aoa">
      <div className="sched-network-legend">
        <span><i className="lg-line lg-critical" />关键工作</span>
        <span><i className="lg-line lg-normal" />工作</span>
        <span><i className="lg-line lg-dummy" />虚工作</span>
        <span>节点：上=最早时间 · 下=最迟时间</span>
        {/* 布局开关（视图态，不入文件）：手动=自由坐标，时间参数不受影响 */}
        <span className="sched-aoa-mode" data-testid="sched-aoa-mode">
          <button
            type="button"
            className={`sched-aoa-mode-btn${!manual ? ' sched-aoa-mode-on' : ''}`}
            onClick={() => setMode('auto')}
            title="时标分层布局（列=最早时间）"
          >
            自动
          </button>
          <button
            type="button"
            className={`sched-aoa-mode-btn${manual ? ' sched-aoa-mode-on' : ''}`}
            onClick={() => setMode('manual')}
            title="手动布局：自由坐标，时间参数不受影响（拖动节点布点，半格吸附）"
          >
            手动
          </button>
        </span>
        {manual && (
          <button
            type="button"
            className="sched-aoa-mode-btn"
            data-testid="sched-aoa-reset"
            onClick={() => { setAoaPins({}); setMode('auto') }}
            title="清除全部手动布点，回到自动布局"
          >
            重置布局
          </button>
        )}
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
          {shown.edges.map((e) => {
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
        {shown.nodes.map((n) => {
          const isDragging = drag?.anchor === n.anchor
          const pos = isDragging && drag ? { x: drag.x, y: drag.y } : { x: n.x, y: n.y }
          return (
            <div
              key={n.id}
              data-anchor={n.anchor}
              className="sched-aoa-node-wrap"
              style={{ left: pos.x - R, top: pos.y - R, cursor: 'grab', opacity: isDragging ? 0.85 : 1 }}
              onMouseDown={(e) => beginDrag(e, n.id)}
              title={manual ? '拖动布点（半格吸附）' : '拖动即转入手动布局'}
            >
              <div className={`sched-aoa-time sched-aoa-time-es${n.es === n.ls ? '' : ' sched-aoa-time-shift'}`}>{n.es}</div>
              <div className={`sched-aoa-node${selectedEdge && (selectedEdge.from === n.id || selectedEdge.to === n.id) ? ' sched-aoa-node-hl' : ''}`}>
                {n.num}
              </div>
              <div className="sched-aoa-time sched-aoa-time-ls">{n.ls}</div>
              {isDragging && drag && (
                <div className="sched-aoa-drag-tip" data-testid="sched-aoa-drag-tip">
                  {drag.x},{drag.y}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

/** 双代号总工期 = 终点事件最早时间（编号最大事件） */
function nodeFinish(graph: { nodes: { num: number; es: number }[] }): number {
  const last = graph.nodes.reduce((a, b) => (b.num > a.num ? b : a), graph.nodes[0])
  return last?.es ?? 0
}
