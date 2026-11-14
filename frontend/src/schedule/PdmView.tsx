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
 *  - W6 收尾：**消费 `nodeRadius`**——单代号的节点是矩形盒（不是圆点），
 *    所以半径在这里是「尺度系数」而不是绝对 px：k = nodeRadius/16，
 *    盒宽高 / 列距 / 行距 / 连线端点全部同源缩放（唯一入口
 *    `networkLayout.resolvePdmNodeMetrics`）。缺省 16 ⇒ k=1 ⇒ 输出逐字不变。
 */
import React, { useMemo, useRef } from 'react'
/* eslint-disable react-refresh/only-export-components -- isLinkBinding 是 FE7-06
   单源收敛的零依赖纯函数（networkExport 构图同口径复用），随唯一语义宿主导出；
   与 genui/markdownFence.tsx 同款先例 */
import { Button } from 'antd'
import { AimOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons'
import type { CpmResult, LinkType, SchedProject } from './types'
import { layerByTopology } from './layout'
import { findBridgeArcs, segsToPath, type Seg } from './aoaLayout'
import { fmtDate } from './gantt/ganttUtil'
import { useAutoFitZoom, useFitView } from './gantt/useFitView'
import { NETWORK_GEOMETRY_DEFAULTS, resolveNetworkDisplay, resolveNetworkGeometry, sixParaLines } from './networkDisplay'
import { resolveNetworkLayout, resolvePdmNodeMetrics } from './networkLayout'
import { useScheduleStore } from './store'
import { useWheelZoom } from './wheelZoom'
import './network-display.css'

/**
 * 画布外边距（px）：**不参与**半径缩放（留白是画布边距，不是节点间隙）。
 * 节点盒宽高 / 列距 / 行距的基准值统一在 `networkLayout.resolvePdmNodeMetrics`
 * 里按 nodeRadius 出尺度（W6 收尾），本文件不再写第二份 150/92/74/40。
 */
const MARGIN = 28

/** 该搭接当前是否为「绑定约束」：后继日期恰由这条搭接决定（两端关键+绑定 ⇒ 关键线路贯通段）。
 * 单源导出：networkExport 构图同口径复用（FE7-06 消除本地同构副本）；零依赖纯函数。 */
export function isLinkBinding(type: LinkType, lag: number, f: { es: number; ef: number }, t: { es: number; ef: number }): boolean {
  switch (type) {
    case 'FS': return t.es === f.ef + lag
    case 'SS': return t.es === f.es + lag
    case 'FF': return t.ef === f.ef + lag
    case 'SF': return t.ef === f.es + lag
  }
}

/**
 * 箭头标记（W4 几何参数）：尺寸由 headHeight 派生、实心/空心由 arrowStyle 决定。
 * h=8（单代号缺省口径）时产出与原手写标记完全一致的 8×8 / refX 7 / refY 4。
 */
function arrowMarker(id: string, color: string, h: number, hollow: boolean) {
  return (
    <marker key={id} id={id} markerWidth={h} markerHeight={h} refX={h - 1} refY={h / 2} orient="auto">
      <path
        d={`M0,0 L${h},${h / 2} L0,${h} z`}
        fill={hollow ? 'none' : color}
        stroke={hollow ? color : undefined}
        strokeWidth={hollow ? 1.2 : undefined}
      />
    </marker>
  )
}

export const PdmView: React.FC<{ project: SchedProject; cpm: CpmResult }> = ({ project, cpm }) => {
  const selectedId = useScheduleStore((s) => s.selectedId)
  const select = useScheduleStore((s) => s.select)
  /**
   * 图面样式参数与图上内容四开关（W4）：与双代号同一份解析入口
   * （`networkDisplay.resolveNetworkGeometry/Display`）——两视图不再各写缺省。
   */
  const geom = useMemo(() => resolveNetworkGeometry(project.networkStyle), [project.networkStyle])
  const flags = useMemo(() => resolveNetworkDisplay(project.networkStyle), [project.networkStyle])
  /**
   * 布图参数（W4 收口，唯一解析入口 networkLayout.ts）：density 只调列距/行距系数、
   * namePos 控节点名称排布、hideUnimportantLink 收掉非关键搭接线。缺省全缺省时
   * 系数=1、namePos=auto、hide=false ⇒ 本视图输出与改动前逐字一致。
   */
  const nlayout = useMemo(() => resolveNetworkLayout(project.networkStyle), [project.networkStyle])
  /**
   * 节点尺度（W6 收尾）：单代号里 `nodeRadius` 是**尺度系数**不是绝对 px ——
   * k = nodeRadius / 16，节点盒宽高、列距、行距、连线端点全部按同一 k 缩放，
   * 盒子与间隙同比例，布局不会重叠。缺省 16 ⇒ k=1 ⇒ 与改动前逐字一致。
   */
  const metric = useMemo(() => resolvePdmNodeMetrics(geom.nodeRadius), [geom.nodeRadius])
  // v4.127 刀E：缩放/一图适配——链式计划整网一行展开时也能整屏读完。
  // 外壳单源 useFitView（FE7-12）：PDM 钳域 [0.2,2]、适配下限 0.2/上限 1、
  // 高度项 clientHeight-40——参数刻意不与 AoaView 统一（AOA 更密）。
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const { zoom, zoomRef, touchedRef, applyZoom, stepZoom, fitView } = useFitView(scrollRef, 0.2, 2)
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
    // 紧凑度系数（斑马 layout_density）：**只**调列距/行距比例——节点尺寸、拓扑、
    // 层级都不动；系数=1 时与旧版同一条算式（缺省零回归）。
    // W6 收尾：基准列距/行距来自 metric（随 nodeRadius 同源缩放），密度系数再乘上去。
    const colGap = metric.colGap * nlayout.colScale
    const rowGap = metric.rowGap * nlayout.rowScale
    const maxCol = Math.max(0, ...[...pos.values()].map((p) => p.col))
    const maxRow = Math.max(0, ...[...pos.values()].map((p) => p.row))
    const w = MARGIN * 2 + (maxCol + colShift + (hasEnd ? 1 : 0) + 1) * (metric.nodeW + colGap)
    const h = MARGIN * 2 + (maxRow + 1) * (metric.nodeH + rowGap)
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
      nodeX: (id: string) => MARGIN + (pos.get(id)!.col + colShift) * (metric.nodeW + colGap),
      nodeY: (id: string) => MARGIN + pos.get(id)!.row * (metric.nodeH + rowGap),
      /** 虚拟节点（S/T）中线 y：全图行中线 */
      midY: MARGIN + (maxRow * (metric.nodeH + rowGap)) / 2,
      startX: MARGIN,
      endX: MARGIN + (maxCol + colShift + 1) * (metric.nodeW + colGap),
    }
  }, [leaves, project.links, cpm, nlayout.colScale, nlayout.rowScale, metric])

  // 布局尺寸变化（切工程/增删任务）且用户未手动缩放时，自动适配一次
  // （hooks 顺序纪律：必须在下方 early return 之前；clientWidth=0（jsdom/未布局）
  // 不误适配——v4.143 同款纪律，0 宽会让 min() 取负值把 zoom 压到下限）
  useAutoFitZoom(
    scrollRef,
    touchedRef,
    applyZoom,
    (el) => {
      if (!layout || layout.w === 0 || layout.h === 0 || el.clientWidth <= 0) return null
      const z = Math.min(el.clientWidth / layout.w, (el.clientHeight - 40) / layout.h, 1)
      return Math.max(0.2, Math.round(z * 100) / 100)
    },
    layout,
    !!layout && layout.w > 0 && layout.h > 0,
  )

  if (!layout) {
    return <div className="sched-empty">暂无任务或计划存在循环依赖，无法绘制单代号网络图</div>
  }

  /** 适配：整网缩放到当前视口（手动缩放过后不再自动抢占；touched 由按钮先置） */
  const fitPdm = () => {
    fitView((el) => {
      if (layout.w === 0 || layout.h === 0) return null
      const z = Math.min(el.clientWidth / layout.w, (el.clientHeight - 40) / layout.h, 1)
      return Math.max(0.2, Math.round(z * 100) / 100)
    })
  }

  const links = project.links.filter((l) => layout.pos.has(l.from) && layout.pos.has(l.to))
  /** 关键搭接（绑定中的临界搭接=关键线路贯通段）：渲染与「隐藏非重要关系线」同一判据 */
  const isCritLink = (l: { from: string; to: string; type: LinkType; lag: number }): boolean => {
    const f = cpm.rows[l.from]
    const t = cpm.rows[l.to]
    return !!(f && t && f.critical && t.critical && isLinkBinding(l.type, l.lag, f, t))
  }
  /**
   * 隐藏非重要关系线（斑马 is_hide_unimportant_link）：单代号的「非重要」= 非关键线路上的
   * 搭接（含时距标注整条收掉）；**虚拟起点/终点桩是拓扑兜底**，不受该开关影响。
   */
  const hiddenLinkCount = nlayout.hideUnimportantLink ? links.filter((l) => !isCritLink(l)).length : 0
  const critCount = leaves.filter((t) => cpm.rows[t.id]?.critical).length
  /**
   * 箭头与线宽（W4 几何参数）：headHeight 只在**显式非缺省**时生效——缺省保留单代号
   * 既有 8×8 / refX 7 / refY 4 口径；线宽同理，缺省不内联，避免压掉关键搭接（2px）
   * 与虚拟桩虚线（5 4）各自的 CSS 档位。旧观感零回归靠这两条保持不变式。
   */
  const headH = geom.headHeight === NETWORK_GEOMETRY_DEFAULTS.headHeight ? 8 : geom.headHeight
  const linkStyle = geom.lineWidth === NETWORK_GEOMETRY_DEFAULTS.lineWidth ? undefined : { strokeWidth: geom.lineWidth }
  /** 节点框内的附加行（六参数 / 起止时间）——只在对应开关打开时出现 */
  const nodeExtra = flags.showSixPara || flags.showStartStopTime

  /** 搭接连线段：右缘出、直角拐、左缘入（与横道/双代号同口径的正交三段）。
   *  端点跟着节点盒走（W6 收尾）：盒宽高已按 nodeRadius 尺度缩放，
   *  所以 x1/x2/y1/y2 必须取同一份 metric，否则连线会脱开盒边。
   *  +8 / −4 是箭头头部让位（固定 px），不随半径缩放，保证极小半径下拐点仍为正。 */
  const linkSegs = (l: { from: string; to: string }): Seg[] => {
    const x1 = layout.nodeX(l.from) + metric.nodeW
    const y1 = layout.nodeY(l.from) + metric.nodeH / 2
    const x2 = layout.nodeX(l.to)
    const y2 = layout.nodeY(l.to) + metric.nodeH / 2
    const midX = Math.max(x1 + 8, (x1 + x2) / 2)
    return [
      { x1, y1, x2: midX, y2: y1 },
      { x1: midX, y1, x2: midX, y2 },
      { x1: midX, y1: y2, x2: x2 - 4, y2 },
    ]
  }
  const linkLabelXY = (l: { from: string; to: string }) => {
    const x1 = layout.nodeX(l.from) + metric.nodeW
    const y1 = layout.nodeY(l.from) + metric.nodeH / 2
    const x2 = layout.nodeX(l.to)
    const y2 = layout.nodeY(l.to) + metric.nodeH / 2
    const midX = Math.max(x1 + 8, (x1 + x2) / 2)
    return { x: midX, y: (y1 + y2) / 2 - 4 }
  }
  // G1 过桥法（v4.130 刀H）：全部连线（含虚拟 S/T 短桩）参与交叉检测，
  // 竖直段垂直穿越他边水平段处画半圆跨过
  const allSegs: Seg[][] = [
    ...links.map(linkSegs),
    // 虚拟起点桩：节点右缘 →8px→ 竖拐 → 目标左缘
    ...layout.sources.map((id) => {
      const yS = layout.midY + metric.nodeH / 2
      const yT = layout.nodeY(id) + metric.nodeH / 2
      const xV = layout.startX + metric.nodeW + 8
      return [
        { x1: layout.startX + metric.nodeW, y1: yS, x2: xV, y2: yS },
        { x1: xV, y1: yS, x2: xV, y2: yT },
        { x1: xV, y1: yT, x2: layout.nodeX(id) - 4, y2: yT },
      ]
    }),
    // 虚拟完成桩：节点右缘 → 长横线 → 竖拐 → 终点列
    ...layout.sinks.map((id) => {
      const yN = layout.nodeY(id) + metric.nodeH / 2
      const yT = layout.midY + metric.nodeH / 2
      const xV = layout.endX - 8
      return [
        { x1: layout.nodeX(id) + metric.nodeW, y1: yN, x2: xV, y2: yN },
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
        {nlayout.hideUnimportantLink && (
          <span className="sched-net-hide-hint" data-testid="sched-pdm-hide-hint">
            已隐藏 {hiddenLinkCount} 条非重要关系线（非关键搭接；虚拟起点/终点桩保留）
          </span>
        )}
        <span>时间单位：工作日（按日历）；「(日历)」=日历天任务（自然日定时，等效工作日跨度=EF−ES）</span>
        <span style={{ display: 'inline-flex', gap: 4, alignItems: 'center' }}>
          <Button size="small" icon={<ZoomOutOutlined />} onClick={() => stepZoom(1 / 1.2)} title="缩小" />
          <span className="sched-pdm-zoom" data-testid="sched-pdm-zoom">{Math.round(zoom * 100)}%</span>
          <Button size="small" icon={<ZoomInOutlined />} onClick={() => stepZoom(1.2)} title="放大" />
          <Button size="small" icon={<AimOutlined />} onClick={() => { touchedRef.current = true; fitPdm() }} title="适配全图" data-testid="sched-pdm-fit" />
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
              {/* 箭头几何（W4）：尺寸=headHeight、填充=arrowStyle；缺省产出与原 8×8 标记一致 */}
              {arrowMarker('pdm-arrow', 'var(--sched-link, #94a3b8)', headH, geom.arrowStyle === 'hollow')}
              {arrowMarker('pdm-arrow-crit', 'var(--sched-critical, #e02020)', headH, geom.arrowStyle === 'hollow')}
            </defs>
            {links.map((l, i) => {
              const crit = isCritLink(l)
              // 隐藏非重要关系线：整条（线+箭头+搭接标注）不画；allSegs 序号不动，
              // 否则过桥弧/虚拟桩的索引会错位
              if (nlayout.hideUnimportantLink && !crit) return null
              const label = `${l.type}${l.lag !== 0 ? (l.lag > 0 ? '+' : '') + l.lag : ''}`
              const xy = linkLabelXY(l)
              return (
                <g key={i}>
                  <path
                    d={segsToPath(allSegs[i], null, bridges.get(i))}
                    className={`sched-net-link${crit ? ' sched-net-link-critical' : ''}`}
                    markerEnd={crit ? 'url(#pdm-arrow-crit)' : 'url(#pdm-arrow)'}
                    style={linkStyle}
                  />
                  {/* 搭接标注（类型+时距）：W4 开关「工期」关掉时整条不画 */}
                  {flags.showDuration && (
                    <text x={xy.x} y={xy.y} textAnchor="middle" className={`sched-net-label${crit ? ' sched-net-label-critical' : ''}`}>{label}</text>
                  )}
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
              style={{ left: layout.startX, top: layout.midY, width: metric.nodeW, height: metric.nodeH, display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700 }}
            >
              起点 S
            </div>
          )}
          {layout.hasEnd && (
            <div
              className="sched-pdm-node sched-pdm-virtual"
              data-testid="sched-pdm-virtual-end"
              style={{ left: layout.endX, top: layout.midY, width: metric.nodeW, height: metric.nodeH, display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700 }}
            >
              完成 T
            </div>
          )}
          {leaves.map((t) => {
            const row = cpm.rows[t.id]
            if (!row) return null
            const crit = row.critical
            /* 盒尺寸随半径尺度缩放（W6 收尾）：宽度恒内联；**高度只在非缺省时内联**——
               缺省 k=1 时沿用 CSS 内容撑高（与改动前逐字一致，零回归），
               非缺省才显式给等比高度，让盒子与间距同比例、内容不会互相压住。 */
            return (
              <div
                key={t.id}
                className={`sched-pdm-node${crit ? ' sched-pdm-node-critical' : ''}${selectedId === t.id ? ' sched-pdm-node-selected' : ''}${geom.dotStyle === 'rect' ? ' sched-net-pdm-rect' : ''}${nodeExtra ? ' sched-net-pdm-extra' : ''}`}
                style={{
                  left: layout.nodeX(t.id),
                  top: layout.nodeY(t.id),
                  width: metric.nodeW,
                  ...(metric.isDefault ? {} : { height: metric.nodeH }),
                }}
                onClick={() => select(t.id)}
                title={`总时差 ${row.tf} 天 · 自由时差 ${row.ff} 天${crit ? ' · 关键工作' : ''}`}
              >
                {/* 名称排布（斑马 name_pos）：auto=现行；横向=强制单行横排；纵向=竖排（writing-mode） */}
                <div
                  className={`sched-pdm-name${nlayout.namePos === 'vertical' ? ' sched-net-name-vertical' : nlayout.namePos === 'horizontal' ? ' sched-net-name-horizontal' : ''}`}
                >
                  <span className="sched-pdm-id">{rowNo.get(t.id)}</span>
                  {t.isMilestone ? <>◆ {t.name}</> : t.name}
                </div>
                <div className="sched-pdm-grid">
                  {/* 节点时间（W4 开关）：最早开始/最早完成两格；工期格由「工期」开关管 */}
                  {flags.showDotTime && <span>{row.es}</span>}
                  {flags.showDuration && (
                    <span className="sched-dim">{t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))}{t.durationUnit === 'cd' ? '(日历)' : ''}</span>
                  )}
                  {flags.showDotTime && <span>{row.ef}</span>}
                  <span>{row.ls}</span>
                  <span className={crit ? 'sched-critical-text' : 'sched-dim'}>{row.tf}</span>
                  <span>{row.lf}</span>
                </div>
                {/* 六参数（显式六行）/ 起止时间：追加在六格下方，不改布局尺寸（绝对定位于框底） */}
                {nodeExtra && (
                  <div className="sched-net-pdm-extra-body">
                    {flags.showSixPara && (
                      <div className="sched-net-sixpara sched-net-sixpara-pdm" data-testid={`sched-net-pdm-sixpara-${t.id}`}>
                        {sixParaLines({
                          es: String(row.es),
                          ef: String(row.ef),
                          ls: String(row.ls),
                          lf: String(row.lf),
                          tf: String(row.tf),
                          ff: String(row.ff),
                        }).map((line) => (
                          <span key={line} className="sched-net-sixpara-line">
                            {line}
                          </span>
                        ))}
                      </div>
                    )}
                    {flags.showStartStopTime && (
                      <div className="sched-net-startstop sched-net-startstop-pdm" data-testid={`sched-net-pdm-startstop-${t.id}`}>
                        {fmtDate(project.startDate, row.es, project.calendar)} → {fmtDate(project.startDate, row.ef, project.calendar)}
                      </div>
                    )}
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
