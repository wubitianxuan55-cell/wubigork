/**
 * schedule/PdmView.tsx — 单代号网络图（PDM/AON）视图
 *
 * 节点=工作（六格图：名称 + ES/D/EF + LS/TF/LF），箭线=搭接（FS/SS/FF/SF+时距）。
 * 布局复用时标分层（layerByTime）；关键工作与关键箭线红色。
 * v4.127 基本功刀E：缩放/适配全图 + 图例吸顶（长链计划一屏可读）。
 */
import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Button } from 'antd'
import { AimOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons'
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
  // v4.127 刀E：缩放/一图适配——链式计划整网一行展开时也能整屏读完
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const [zoom, setZoom] = useState(1)
  const touchedRef = useRef(false)

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

  // 布局尺寸变化（切工程/增删任务）且用户未手动缩放时，自动适配一次
  // （hooks 顺序纪律：必须在下方 early return 之前）
  useEffect(() => {
    if (!layout || touchedRef.current || layout.w === 0 || layout.h === 0) return
    const el = scrollRef.current
    if (!el) return
    const z = Math.min(el.clientWidth / layout.w, (el.clientHeight - 40) / layout.h, 1)
    setZoom(Math.max(0.2, Math.round(z * 100) / 100))
  }, [layout])

  if (!layout) {
    return <div className="sched-empty">暂无任务或计划存在循环依赖，无法绘制单代号网络图</div>
  }

  /** 适配：整网缩放到当前视口（手动缩放过后不再自动抢占） */
  const fitView = () => {
    const el = scrollRef.current
    if (!el || layout.w === 0 || layout.h === 0) return
    const z = Math.min(el.clientWidth / layout.w, (el.clientHeight - 40) / layout.h, 1)
    setZoom(Math.max(0.2, Math.round(z * 100) / 100))
  }
  const stepZoom = (k: number) => {
    touchedRef.current = true
    setZoom((z) => Math.round(Math.min(2, Math.max(0.2, z * k)) * 100) / 100)
  }

  const links = project.links.filter((l) => layout.pos.has(l.from) && layout.pos.has(l.to))
  const critCount = leaves.filter((t) => cpm.rows[t.id]?.critical).length

  return (
    <div className="sched-network-scroll" data-testid="sched-pdm" ref={scrollRef}>
      <div className="sched-network-legend">
        <span><i className="lg-dot lg-critical" />关键工作</span>
        <span><i className="lg-dot lg-normal" />非关键</span>
        <span>格：ES / 工期 / EF · LS / 总时差 / LF</span>
        <span>时间单位：工作日（按日历）</span>
        <span style={{ display: 'inline-flex', gap: 4 }}>
          <Button size="small" icon={<ZoomOutOutlined />} onClick={() => stepZoom(1 / 1.2)} title="缩小" />
          <span className="sched-pdm-zoom" data-testid="sched-pdm-zoom">{Math.round(zoom * 100)}%</span>
          <Button size="small" icon={<ZoomInOutlined />} onClick={() => stepZoom(1.2)} title="放大" />
          <Button size="small" icon={<AimOutlined />} onClick={() => { touchedRef.current = true; fitView() }} title="适配全图" data-testid="sched-pdm-fit" />
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
    </div>
  )
}
