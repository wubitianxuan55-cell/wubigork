/**
 * schedule/PdmView.tsx — 单代号网络图（PDM/AON）视图
 *
 * 节点=工作（六格标注法：名称 + ES/D/EF + LS/TF/LF），箭线=搭接（FS/SS/FF/SF+时距）。
 * 关键工作红框，**绑定中**的临界搭接红色加粗——关键线路连续贯通可辨。
 *
 * v4.128 基本功刀F（单代号合规整改，走查用户实锤「连成一条直线/没有分支/看不出关键线路」）：
 *  - 布局改 **拓扑分层**（layerByTopology，列=最长路径层级，并行分支同列并列）——
 *    原按最早时间分层会把不同开工日的并行工作拉成一条直线，逻辑分支全灭；
 *  - 多起点/多终点增画 **虚拟起点/终点节点**（虚线框 S/T，JGJ/T 121 单代号口径）；
 *  - 关键线路贯通：搭接两端均关键且该搭接为「绑定约束」（后继日期恰由该搭接决定）时红色加粗。
 *  - v4.127 刀E：缩放/适配全图 + 图例吸顶。
 *  - v4.130 刀H G1：连线改段序列（aoaLayout.segs）共用过桥法——竖直段垂直
 *    穿越他边水平段处画半圆跨过（findBridgeArcs/segsToPath 与双代号同源）。
 */
import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Button } from 'antd'
import { AimOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons'
import type { CpmResult, LinkType, SchedProject } from './types'
import { layerByTopology } from './layout'
import { findBridgeArcs, segsToPath, type Seg } from './aoaLayout'
import { useScheduleStore } from './store'
import { useWheelZoom } from './wheelZoom'

const NODE_W = 150
const NODE_H = 92
const COL_GAP = 74
const ROW_GAP = 40
const MARGIN = 28

/** 该搭接当前是否为「绑定约束」：后继日期恰由这条搭接决定（两端关键+绑定 ⇒ 关键线路贯通段） */
function isLinkBinding(type: LinkType, lag: number, f: { es: number; ef: number }, t: { es: number; ef: number }): boolean {
  switch (type) {
    case 'FS': return t.es === f.ef + lag
    case 'SS': return t.es === f.es + lag
    case 'FF': return t.ef === f.ef + lag
    case 'SF': return t.ef === f.es + lag
  }
}

export const PdmView: React.FC<{ project: SchedProject; cpm: CpmResult }> = ({ project, cpm }) => {
  const selectedId = useScheduleStore((s) => s.selectedId)
  const select = useScheduleStore((s) => s.select)
  // v4.127 刀E：缩放/一图适配——链式计划整网一行展开时也能整屏读完
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const [zoom, setZoom] = useState(1)
  const zoomRef = useRef(1)
  const touchedRef = useRef(false)
  const applyZoom = (z: number) => {
    const c = Math.min(2, Math.max(0.2, Math.round(z * 100) / 100))
    zoomRef.current = c
    setZoom(c)
  }
  // 滚轮缩放（v4.159）：普通滚轮=以光标为锚缩放，Shift+滚轮=横向滚动
  useWheelZoom(scrollRef, zoomRef, applyZoom)

  const leaves = useMemo(() => project.tasks.filter((t) => t.level > 0), [project.tasks])
  /** 行号速查（Project 引用口径，节点角标） */
  const rowNo = useMemo(() => new Map(project.tasks.map((t, i) => [t.id, i + 1])), [project.tasks])

  const layout = useMemo(() => {
    if (leaves.length === 0 || !cpm.ok) return null
    const ids = new Set(leaves.map((t) => t.id))
    const links = project.links.filter((l) => ids.has(l.from) && ids.has(l.to))
    // 拓扑分层（刀F）：并行分支同列并列；起点/终点按「叶任务入边/出边」判定
    const pos = layerByTopology(leaves.map((t) => ({ id: t.id })), links.map((l) => ({ from: l.from, to: l.to })))
    const inIds = new Set(links.map((l) => l.to))
    const outIds = new Set(links.map((l) => l.from))
    const sources = leaves.filter((t) => !inIds.has(t.id)).map((t) => t.id)
    const sinks = leaves.filter((t) => !outIds.has(t.id)).map((t) => t.id)
    const hasStart = sources.length > 1
    const hasEnd = sinks.length > 1
    const colShift = hasStart ? 1 : 0
    const maxCol = Math.max(0, ...[...pos.values()].map((p) => p.col))
    const maxRow = Math.max(0, ...[...pos.values()].map((p) => p.row))
    const w = MARGIN * 2 + (maxCol + colShift + (hasEnd ? 1 : 0) + 1) * (NODE_W + COL_GAP)
    const h = MARGIN * 2 + (maxRow + 1) * (NODE_H + ROW_GAP)
    return {
      pos,
      sources,
      sinks,
      hasStart,
      hasEnd,
      colShift,
      maxCol,
      maxRow,
      w,
      h,
      nodeX: (id: string) => MARGIN + (pos.get(id)!.col + colShift) * (NODE_W + COL_GAP),
      nodeY: (id: string) => MARGIN + pos.get(id)!.row * (NODE_H + ROW_GAP),
      /** 虚拟节点（S/T）中线 y：全图行中线 */
      midY: MARGIN + (maxRow * (NODE_H + ROW_GAP)) / 2,
      startX: MARGIN,
      endX: MARGIN + (maxCol + colShift + 1) * (NODE_W + COL_GAP),
    }
  }, [leaves, project.links, cpm])

  // 布局尺寸变化（切工程/增删任务）且用户未手动缩放时，自动适配一次
  // （hooks 顺序纪律：必须在下方 early return 之前；clientWidth=0（jsdom/未布局）
  // 不误适配——v4.143 同款纪律，0 宽会让 min() 取负值把 zoom 压到下限）
  useEffect(() => {
    if (!layout || touchedRef.current || layout.w === 0 || layout.h === 0) return
    const el = scrollRef.current
    if (!el || el.clientWidth <= 0) return
    const z = Math.min(el.clientWidth / layout.w, (el.clientHeight - 40) / layout.h, 1)
    applyZoom(Math.max(0.2, Math.round(z * 100) / 100))
  }, [layout])

  if (!layout) {
    return <div className="sched-empty">暂无任务或计划存在循环依赖，无法绘制单代号网络图</div>
  }

  /** 适配：整网缩放到当前视口（手动缩放过后不再自动抢占） */
  const fitView = () => {
    const el = scrollRef.current
    if (!el || layout.w === 0 || layout.h === 0) return
    const z = Math.min(el.clientWidth / layout.w, (el.clientHeight - 40) / layout.h, 1)
    applyZoom(Math.max(0.2, Math.round(z * 100) / 100))
  }
  const stepZoom = (k: number) => {
    touchedRef.current = true
    applyZoom(zoomRef.current * k)
  }

  const links = project.links.filter((l) => layout.pos.has(l.from) && layout.pos.has(l.to))
  const critCount = leaves.filter((t) => cpm.rows[t.id]?.critical).length

  /** 搭接连线段：右缘出、直角拐、左缘入（与横道/双代号同口径的正交三段） */
  const linkSegs = (l: { from: string; to: string }): Seg[] => {
    const x1 = layout.nodeX(l.from) + NODE_W
    const y1 = layout.nodeY(l.from) + NODE_H / 2
    const x2 = layout.nodeX(l.to)
    const y2 = layout.nodeY(l.to) + NODE_H / 2
    const midX = Math.max(x1 + 8, (x1 + x2) / 2)
    return [
      { x1, y1, x2: midX, y2: y1 },
      { x1: midX, y1, x2: midX, y2 },
      { x1: midX, y1: y2, x2: x2 - 4, y2 },
    ]
  }
  const linkLabelXY = (l: { from: string; to: string }) => {
    const x1 = layout.nodeX(l.from) + NODE_W
    const y1 = layout.nodeY(l.from) + NODE_H / 2
    const x2 = layout.nodeX(l.to)
    const y2 = layout.nodeY(l.to) + NODE_H / 2
    const midX = Math.max(x1 + 8, (x1 + x2) / 2)
    return { x: midX, y: (y1 + y2) / 2 - 4 }
  }
  // G1 过桥法（v4.130 刀H）：全部连线（含虚拟 S/T 短桩）参与交叉检测，
  // 竖直段垂直穿越他边水平段处画半圆跨过
  const allSegs: Seg[][] = [
    ...links.map(linkSegs),
    // 虚拟起点桩：节点右缘 →8px→ 竖拐 → 目标左缘
    ...layout.sources.map((id) => {
      const yS = layout.midY + NODE_H / 2
      const yT = layout.nodeY(id) + NODE_H / 2
      const xV = layout.startX + NODE_W + 8
      return [
        { x1: layout.startX + NODE_W, y1: yS, x2: xV, y2: yS },
        { x1: xV, y1: yS, x2: xV, y2: yT },
        { x1: xV, y1: yT, x2: layout.nodeX(id) - 4, y2: yT },
      ]
    }),
    // 虚拟完成桩：节点右缘 → 长横线 → 竖拐 → 终点列
    ...layout.sinks.map((id) => {
      const yN = layout.nodeY(id) + NODE_H / 2
      const yT = layout.midY + NODE_H / 2
      const xV = layout.endX - 8
      return [
        { x1: layout.nodeX(id) + NODE_W, y1: yN, x2: xV, y2: yN },
        { x1: xV, y1: yN, x2: xV, y2: yT },
        { x1: xV, y1: yT, x2: layout.endX - 4, y2: yT },
      ]
    }),
  ]
  const bridges = findBridgeArcs(allSegs)

  return (
    <div className="sched-network-scroll" data-testid="sched-pdm" ref={scrollRef}>
      <div className="sched-network-legend">
        <span><i className="lg-dot lg-critical" />关键工作</span>
        <span><i className="lg-dot lg-normal" />非关键</span>
        <span>格：ES / 工期 / EF · LS / 总时差 / LF</span>
        <span>关键线路=红箭线贯通（绑定中的临界搭接）</span>
        <span>时间单位：工作日（按日历）；「(日历)」=日历天任务（自然日定时，等效工作日跨度=EF−ES）</span>
        <span style={{ display: 'inline-flex', gap: 4, alignItems: 'center' }}>
          <Button size="small" icon={<ZoomOutOutlined />} onClick={() => stepZoom(1 / 1.2)} title="缩小" />
          <span className="sched-pdm-zoom" data-testid="sched-pdm-zoom">{Math.round(zoom * 100)}%</span>
          <Button size="small" icon={<ZoomInOutlined />} onClick={() => stepZoom(1.2)} title="放大" />
          <Button size="small" icon={<AimOutlined />} onClick={() => { touchedRef.current = true; fitView() }} title="适配全图" data-testid="sched-pdm-fit" />
          <span className="sched-net-hint">滚轮缩放 · Shift+滚轮横移</span>
        </span>
        {/* 进度统计牌（斑马口径：红字大数字，挂图例行右端避免遮挡节点） */}
        <div className="sched-net-badge">
          <div className="nb-item">
            <span className="nb-num">{cpm.duration}</span>
            <span className="nb-label">总工期(日)</span>
          </div>
          <div className="nb-item">
            <span className="nb-num">{critCount}</span>
            <span className="nb-label">关键工作</span>
          </div>
          <div className="nb-item">
            <span className="nb-num">{leaves.length}</span>
            <span className="nb-label">工作总数</span>
          </div>
        </div>
      </div>
      <div style={{ width: layout.w * zoom, height: layout.h * zoom }}>
        <div
          className="sched-network-canvas"
          style={{ width: layout.w, height: layout.h, transform: `scale(${zoom})`, transformOrigin: '0 0' }}
        >
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
              const f = cpm.rows[l.from]
              const t = cpm.rows[l.to]
              const crit = !!(f && t && f.critical && t.critical && isLinkBinding(l.type, l.lag, f, t))
              const label = `${l.type}${l.lag !== 0 ? (l.lag > 0 ? '+' : '') + l.lag : ''}`
              const xy = linkLabelXY(l)
              return (
                <g key={i}>
                  <path
                    d={segsToPath(allSegs[i], null, bridges.get(i))}
                    className={`sched-net-link${crit ? ' sched-net-link-critical' : ''}`}
                    markerEnd={crit ? 'url(#pdm-arrow-crit)' : 'url(#pdm-arrow)'}
                  />
                  <text x={xy.x} y={xy.y} textAnchor="middle" className={`sched-net-label${crit ? ' sched-net-label-critical' : ''}`}>{label}</text>
                </g>
              )
            })}
            {/* 虚拟起点/终点（多起点/多终点时，JGJ/T 121 单代号口径）：虚线框 + 虚箭线 */}
            {layout.hasStart &&
              layout.sources.map((id, j) => (
                <path
                  key={`vs-${id}`}
                  d={segsToPath(allSegs[links.length + j], null, bridges.get(links.length + j))}
                  className="sched-net-link sched-net-link-virtual"
                  markerEnd="url(#pdm-arrow)"
                />
              ))}
            {layout.hasEnd &&
              layout.sinks.map((id, j) => (
                <path
                  key={`vt-${id}`}
                  d={segsToPath(allSegs[links.length + layout.sources.length + j], null, bridges.get(links.length + layout.sources.length + j))}
                  className="sched-net-link sched-net-link-virtual"
                  markerEnd="url(#pdm-arrow)"
                />
              ))}
          </svg>
          {layout.hasStart && (
            <div
              className="sched-pdm-node sched-pdm-virtual"
              data-testid="sched-pdm-virtual-start"
              style={{ left: layout.startX, top: layout.midY, width: NODE_W, height: NODE_H, display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700 }}
            >
              起点 S
            </div>
          )}
          {layout.hasEnd && (
            <div
              className="sched-pdm-node sched-pdm-virtual"
              data-testid="sched-pdm-virtual-end"
              style={{ left: layout.endX, top: layout.midY, width: NODE_W, height: NODE_H, display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700 }}
            >
              完成 T
            </div>
          )}
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
                title={`总时差 ${row.tf} 天 · 自由时差 ${row.ff} 天${crit ? ' · 关键工作' : ''}`}
              >
                <div className="sched-pdm-name">
                  <span className="sched-pdm-id">{rowNo.get(t.id)}</span>
                  {t.isMilestone ? <>◆ {t.name}</> : t.name}
                </div>
                <div className="sched-pdm-grid">
                  <span>{row.es}</span>
                  <span className="sched-dim">{t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))}{t.durationUnit === 'cd' ? '(日历)' : ''}</span>
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
    </div>
  )
}
