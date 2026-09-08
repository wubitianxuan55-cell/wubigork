/**
 * schedule/AoaView.tsx — 双代号网络图（AOA）视图
 *
 * 圈节点=事件（圈内编号，上方最早时间、下方最迟时间），箭线=工作，
 * 虚箭线=虚工作（虚线）。关键线路（含关键虚工作）红色加粗。
 * 数据来自 buildAoa（虚工作自动插入 + i<j 编号 + 真时标布局：x=最早时间）。
 *
 * v4.123 AOA 刀1：手动布局双模式——「自动」=时标布局（现状零改动）；
 * 「手动」= pins 覆盖（applyPins 混合锚定：命中手动位/新事件自动落位），
 * x 与时间解耦（时间参数仍全量显示）。拖拽布点对齐横道刀9 纪律：
 * window 监听三件套、半格吸附纯函数（aoaLayout.ts snapPt）、未移动不提交、
 * Esc 取消、自动模式拖拽=自动转手动（「不骗人」先例）、提交时剪枝失配键。
 *
 * v4.127 基本功刀A：箭线正交画法合规——横平竖直、直角拐弯、禁斜线
 * （JGJ/T 121 绘图口径）；平行边错位通道。
 *
 * v4.130 基本功刀H（图例语言）：G2 标注归位（工作名称在箭线上、持续时间
 * 在下）；G4 波形线——auto 时标模式下水平段超出实体工期终点（x=事件 x+R+
 * 工期×列宽）的尾段画波形线=自由时差（虚工作有时差时加波形线），手动布局
 * x 与时间解耦不画波形；G1 过桥法——竖直段垂直穿越他边水平段处画半圆跨过
 * （几何纯函数在 aoaLayout.ts，单代号视图共用）。
 */
import React, { useRef, useState } from 'react'
import type { AoaGraph } from './aoa'
import { AOA_COL_W, AOA_MARGIN, AOA_R, AOA_ROW_H } from './aoa'
import { applyPins, assignChannels, edgeSegs, findBridgeArcs, prunePins, segsToPath, snapPt, summarySegs } from './aoaLayout'
import { useScheduleStore } from './store'
import type { AoaPin, SchedTask } from './types'
import { useWheelZoom } from './wheelZoom'

const R = AOA_R

/**
 * 正交箭线几何已迁 aoaLayout.edgeSegs（v4.130 刀H：纯函数可测 + 段序列
 * 供波形/过桥二次加工）。
 */

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
  // 图面尺寸（提前派生供自动适配用）
  const shownEarly = mode === 'manual' ? applyPins(graph, pins) : graph
  const wEarly = shownEarly.nodes.length > 0 ? Math.max(...shownEarly.nodes.map((n) => n.x)) + AOA_MARGIN + AOA_COL_W / 2 : 0
  const hEarly = shownEarly.nodes.length > 0 ? Math.max(...shownEarly.nodes.map((n) => n.y)) + AOA_MARGIN + AOA_ROW_H / 2 : 0
  // 缩放/全览（v4.142，对齐单代号刀E 范式）：长计划 110px/天 展开上万像素，
  // 没有缩放根本读不了——首帧自动适配一次，用户手动缩放后不再抢占
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const [zoom, setZoom] = useState(1)
  const zoomRef = useRef(1)
  const touchedRef = useRef(false)
  const applyZoom = (z: number) => {
    const c = Math.min(2, Math.max(0.1, Math.round(z * 100) / 100))
    zoomRef.current = c
    setZoom(c)
  }
  const stepZoom = (f: number) => {
    touchedRef.current = true
    applyZoom(zoomRef.current * f)
  }
  const fitView = () => {
    touchedRef.current = true
    const el = scrollRef.current
    if (!el || el.clientWidth <= 0) return
    // 全览下限 0.3：整图入窗也不缩成蚂蚁（再小读不出字）
    applyZoom(Math.max(0.3, Math.min(el.clientWidth / wEarly, Math.max(160, el.clientHeight - 24) / hEarly, 1.5)))
  }
  // 滚轮缩放（v4.159）：普通滚轮=以光标为锚缩放，Shift+滚轮=横向滚动；
  // 与按钮/全览共用 touchedRef（手动缩放后首帧自动适配不再抢占）
  useWheelZoom(scrollRef, zoomRef, applyZoom)
  // 图面尺寸变化（切工程/增删任务）且用户未手动缩放时，自动适配一次。
  // 下限 0.5=可读优先：装不下就横向滚动（参考斑马/Project 显示口径），
  // 绝不把整图无脑缩到看不清
  React.useEffect(() => {
    if (touchedRef.current) return
    const el = scrollRef.current
    if (!el || wEarly <= 0 || el.clientWidth <= 0) return // clientWidth=0（jsdom/未布局）不误适配
    const z = Math.max(0.5, Math.min(el.clientWidth / wEarly, Math.max(160, el.clientHeight - 24) / hEarly, 1.5))
    if (Number.isFinite(z) && z > 0) applyZoom(z)
  }, [wEarly, hEarly])

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
      const k = zoomRef.current || 1 // 缩放下拖拽：屏幕位移折算回图面坐标
      const snapped = snapPt(d.orig.x + (ev.clientX - d.startX) / k, d.orig.y + (ev.clientY - d.startY) / k)
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
    <div className="sched-network-scroll" data-testid="sched-aoa" ref={scrollRef}>
      <div className="sched-network-legend">
        <span><i className="lg-line lg-critical" />关键工作</span>
        <span><i className="lg-line lg-normal" />工作</span>
        <span><i className="lg-line lg-dummy" />虚工作</span>
        <span>
          <svg width="18" height="8" style={{ marginRight: 4, verticalAlign: 'middle' }} aria-hidden>
            <path d="M0 4 q 2.25 -5 4.5 0 t 4.5 0 t 4.5 0 t 4.5 0" fill="none" stroke="var(--sched-link, #94a3b8)" strokeWidth="1.4" />
          </svg>
          自由时差（波形线）
        </span>
        <span>节点：上=最早时间 · 下=最迟时间</span>
        <span>标注：箭线上=工作名称 · 下=工期（「(日历)」=日历天任务）</span>
        <span><i className="lg-line sched-aoa-summary-legend" />一级汇总线（分组横幅顶部通长线，衔接二级子网络）</span>
        {/* 布局开关（视图态，不入文件）：手动=自由坐标，时间参数不受影响 */}
        <span className="sched-aoa-mode" data-testid="sched-aoa-mode">
          <button
            type="button"
            className={`sched-aoa-mode-btn${!manual ? ' sched-aoa-mode-on' : ''}`}
            onClick={() => setMode('auto')}
            title="时标布局：水平位置=最早时间（1 格=1 天），波形线=自由时差"
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
        {/* 缩放/全览（v4.142）：长计划时标展开上万像素，首帧自动适配一次 */}
        <span style={{ display: 'inline-flex', gap: 4, alignItems: 'center' }}>
          <button type="button" className="sched-aoa-mode-btn" data-testid="sched-aoa-zoomout" title="缩小" onClick={() => stepZoom(1 / 1.2)}>−</button>
          <span className="sched-pdm-zoom" data-testid="sched-aoa-zoom">{Math.round(zoom * 100)}%</span>
          <button type="button" className="sched-aoa-mode-btn" data-testid="sched-aoa-zoomin" title="放大" onClick={() => stepZoom(1.2)}>＋</button>
          <button type="button" className="sched-aoa-mode-btn" data-testid="sched-aoa-fit" title="全览：整网适配当前视口" onClick={() => { touchedRef.current = true; fitView() }}>
            全览
          </button>
          <span className="sched-net-hint">滚轮缩放 · Shift+滚轮横移</span>
        </span>
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
      <div className="sched-network-canvas" style={{ width: w * zoom, height: h * zoom }}>
        <div style={{ position: 'absolute', inset: 0, width: w, height: h, transform: `scale(${zoom})`, transformOrigin: '0 0' }}>
        <svg width={w} height={h} style={{ position: 'absolute', inset: 0 }}>
          <defs>
            <marker id="aoa-arrow" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto">
              <path d="M0,0 L9,4.5 L0,9 z" fill="var(--sched-link, #94a3b8)" />
            </marker>
            <marker id="aoa-arrow-crit" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto">
              <path d="M0,0 L9,4.5 L0,9 z" fill="var(--sched-critical, #dc2626)" />
            </marker>
            <marker id="aoa-arrow-summary" markerWidth="9" markerHeight="9" refX="8" refY="4.5" orient="auto">
              <path d="M0,0 L9,4.5 L0,9 z" fill="var(--sched-group, #1f2937)" />
            </marker>
          </defs>
          {(() => {
            // 分级横幅底纹（v4.160，对标上报件交替底色）：每 band 一块浅色
            // 矩形垫底，一级/二级分区一眼可辨；band 顶=带内最小 y−半个行距
            // （汇总线行），band 底=最大 y+半个行距
            const bandRange = new Map<number, { top: number; bottom: number }>()
            for (const n of shown.nodes) {
              const r = bandRange.get(n.band)
              if (!r) bandRange.set(n.band, { top: n.y, bottom: n.y })
              else {
                r.top = Math.min(r.top, n.y)
                r.bottom = Math.max(r.bottom, n.y)
              }
            }
            const bandTopY = (b: number): number => (bandRange.get(b)?.top ?? 0) - AOA_ROW_H / 2
            const tints = [...bandRange.entries()].map(([b, r]) => (
              <rect
                key={`band-${b}`}
                data-testid={`sched-aoa-band-${b}`}
                x={0}
                y={r.top - AOA_ROW_H / 2}
                width={w}
                height={r.bottom - r.top + AOA_ROW_H}
                fill={b % 2 === 0 ? 'rgba(148,163,184,0.06)' : 'rgba(96,165,250,0.08)'}
              />
            ))
            // 同 (from,to) 平行边计数（错位通道用）
            const pairCnt = new Map<string, number>()
            for (const e of shown.edges) {
              const k = `${e.from}>${e.to}`
              pairCnt.set(k, (pairCnt.get(k) ?? 0) + 1)
            }
            const pairSeen = new Map<string, number>()
            // 同行长边通道分配（v4.143 分行）：跨多列的同行边让到行间通道，
            // x 区间重叠者分层，杜绝长线沿节点中心线重合/横穿节点
            const channels = assignChannels(shown.edges, nodeById)
            // G4 波形线（auto=时标）：实体工期终点 x=源事件 x+R+工期×列宽；
            // 手动布局 x 与时间解耦，波形失义不画；走通道的边无波形
            const geoms = shown.edges.map((e, i) => {
              const k = `${e.from}>${e.to}`
              const idx = pairSeen.get(k) ?? 0
              pairSeen.set(k, idx + 1)
              // 波形切点只对实/虚工作有意义；汇总箭线=横幅顶通长线（v4.160），无波形
              const chY = channels.get(e.id)
              const waveFromX = !manual && e.kind !== 'summary' ? nodeById.get(e.from)!.x + R + e.dur * AOA_COL_W : null
              // 汇总箭线走横幅顶（归属带取两界点的较大 band=被汇总的分部）
              const g = e.kind === 'summary'
                ? summarySegs(nodeById.get(e.from)!, nodeById.get(e.to)!, bandTopY(Math.max(nodeById.get(e.from)!.band, nodeById.get(e.to)!.band)))
                : edgeSegs(e, nodeById, { idx, cnt: pairCnt.get(k)! }, waveFromX, chY)
              return { e, i, waveFromX, g }
            })
            // G1 过桥法：竖段垂直穿越他边横段处画半圆（水平段=时标轴不断）
            const bridges = findBridgeArcs(geoms.map((it) => it.g.segs))
            return (
              <>
                {tints}
                {geoms.map(({ e, i, waveFromX, g }) => {
                  const dummy = e.kind === 'dummy'
                  const summary = e.kind === 'summary'
                  const cls = `sched-aoa-link${e.critical ? ' sched-aoa-link-critical' : ''}${dummy ? ' sched-aoa-dummy' : ''}${summary ? ' sched-aoa-summary' : ''}`
                  return (
                    <g key={e.id}>
                      <path
                        d={segsToPath(g.segs, waveFromX, bridges.get(i))}
                        className={cls}
                        markerEnd={e.critical ? 'url(#aoa-arrow-crit)' : summary ? 'url(#aoa-arrow-summary)' : 'url(#aoa-arrow)'}
                        onClick={() => e.taskId && select(e.taskId)}
                        style={{ pointerEvents: e.taskId ? 'stroke' : 'none', cursor: e.taskId ? 'pointer' : 'default' }}
                      />
                      {e.label && (
                        <text x={g.dur.x} y={g.dur.y} textAnchor={g.dur.anchor} className={`sched-net-label${e.critical ? ' sched-net-label-critical' : ''}`}>
                          {e.label}
                        </text>
                      )}
                      {e.taskId && (
                        <text x={g.name.x} y={g.name.y} textAnchor={g.name.anchor} className={`sched-aoa-taskname${summary ? ' sched-aoa-taskname-summary' : ''}`}>
                          {taskName(e.taskId)}
                        </text>
                      )}
                    </g>
                  )
                })}
              </>
            )
          })()}
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
    </div>
  )
}

/** 双代号总工期 = 终点事件最早时间（编号最大事件） */
function nodeFinish(graph: { nodes: { num: number; es: number }[] }): number {
  const last = graph.nodes.reduce((a, b) => (b.num > a.num ? b : a), graph.nodes[0])
  return last?.es ?? 0
}
