import React, { useMemo, useState, useRef, useEffect, useCallback } from 'react'
import {
  forceSimulation, forceLink, forceManyBody, forceCenter, forceCollide,
} from 'd3-force'
import { ROLE_LABELS, RELATION_LABELS } from '../utils/theme'

interface CharNode {
  id: string; name: string; role_type: string; color: string
}
interface OrgNode {
  id: string; name: string; color: string
}
interface RelEdge {
  from: string; to: string; type: string; color: string; label: string
}

interface RelationGraphProps {
  characters: { id: string; name: string; role_type: string }[]
  organizations: { id: string; name: string }[]
  relationships: { from_id: string; to_id: string; relation_type: string }[]
}

// ── 颜色常量（令牌派生：canvas 不解析 var()，挂载时经 getComputedStyle 解析为具体色；
//    解析失败回退到原 hex，保证可用性） ──
function resolveCSSColor(v: string): string {
  if (!v.includes('var(') || typeof document === 'undefined') return v
  try {
    const el = document.createElement('span')
    el.style.color = v
    document.body.appendChild(el)
    const c = getComputedStyle(el).color
    el.remove()
    return c || v
  } catch { return v }
}

const roleColorTokens: Record<string, string> = {
  protagonist: 'var(--color-warning)', antagonist: 'var(--color-destructive)',
  supporting: 'var(--color-primary)', minor: 'var(--color-text-secondary)',
}
const orgColorToken = 'var(--gaea-glow)'
const relColorTokens: Record<string, string> = {
  friend: 'var(--color-success)', enemy: 'var(--color-destructive)',
  family: 'var(--color-primary)', mentor: 'color-mix(in srgb, var(--color-primary) 55%, var(--color-text))',
  rival: 'var(--color-warning)', lover: 'color-mix(in srgb, var(--color-destructive) 45%, var(--color-primary))',
  member: 'var(--color-text-secondary)', leader: 'var(--gaea-glow)',
}
const roleColors: Record<string, string> = Object.fromEntries(
  Object.entries(roleColorTokens).map(([k, v]) => [k, resolveCSSColor(v)]),
)
const orgColor = resolveCSSColor(orgColorToken)
const relColors: Record<string, string> = Object.fromEntries(
  Object.entries(relColorTokens).map(([k, v]) => [k, resolveCSSColor(v)]),
)
const roleCN = ROLE_LABELS
const relCN = RELATION_LABELS
const BG = resolveCSSColor('color-mix(in srgb, var(--color-surface, #0f0f0f) 92%, #000)')

// ── d3-force 二维仿真 ──
function runSimulation(
  charNodes: CharNode[], orgNodes: OrgNode[], edges: RelEdge[],
): Map<string, { x: number; y: number }> {
  const N = charNodes.length + orgNodes.length
  if (N === 0) return new Map()
  const linkDist = Math.min(200, 80 + 30 * Math.log10(N + 1))
  const repel = -Math.min(3000, 500 + 200 * Math.log10(N + 1))
  const jitter = () => (Math.random() - 0.5) * 2
  interface SimNode { id: string; x: number; y: number }
  interface SimLink { source: string; target: string }
  const nodes: SimNode[] = [
    ...charNodes.map((c) => ({ id: c.id, x: jitter(), y: jitter() })),
    ...orgNodes.map((o) => ({ id: o.id, x: jitter(), y: jitter() })),
  ]
  const links: SimLink[] = edges.map((e) => ({ source: e.from, target: e.to }))

  const sim = forceSimulation<SimNode>(nodes)
    .force('link', forceLink<SimNode, SimLink>(links).id((d) => d.id).distance(linkDist).strength(0.3))
    .force('charge', forceManyBody().strength(repel).distanceMax(linkDist * 3))
    .force('center', forceCenter(0, 0))
    .force('collide', forceCollide(30))
    .stop()

  sim.tick(300)
  const result = new Map<string, { x: number; y: number }>()
  for (const n of nodes) result.set(n.id, { x: n.x, y: n.y })
  return result
}

// ── 主组件 ──
const RelationGraph: React.FC<RelationGraphProps> = ({
  characters, organizations, relationships,
}) => {
  const containerRef = useRef<HTMLDivElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [size, setSize] = useState({ w: 700, h: 500 })
  const [hovered, setHovered] = useState<string | null>(null)
  const [scale, setScale] = useState(1)
  const [offset, setOffset] = useState({ x: 0, y: 0 })
  const dragRef = useRef({ dragging: false, startX: 0, startY: 0, ox: 0, oy: 0, panning: false })

  // ResizeObserver
  useEffect(() => {
    const el = containerRef.current
    if (!el) return
    const ro = new ResizeObserver(([entry]) => {
      const { width, height } = entry.contentRect
      if (width > 0 && height > 0) setSize({ w: width, h: height })
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  // 构建节点和边
  const charNodes: CharNode[] = useMemo(() =>
    characters.map((c) => ({
      id: c.id, name: c.name, role_type: c.role_type,
      color: roleColors[c.role_type] || '#6b7280',
    })),
  [characters])
  const orgNodes: OrgNode[] = useMemo(() =>
    organizations.map((o) => ({ id: o.id, name: o.name, color: orgColor })),
  [organizations])

  const edges: RelEdge[] = useMemo(() => {
    const edgeSet = new Set<string>()
    const result: RelEdge[] = []
    for (const r of relationships) {
      const key = [r.from_id, r.to_id].sort().join('-')
      if (edgeSet.has(key)) continue
      edgeSet.add(key)
      result.push({
        from: r.from_id, to: r.to_id, type: r.relation_type,
        color: relColors[r.relation_type] || '#6b7280',
        label: relCN[r.relation_type] || r.relation_type,
      })
    }
    return result
  }, [relationships])

  // 力导向布局
  const positions = useMemo(() => runSimulation(charNodes, orgNodes, edges), [charNodes, orgNodes, edges])

  // 悬停关联节点集
  const hoverRelated = useMemo(() => {
    if (!hovered) return new Set<string>()
    const related = new Set<string>([hovered])
    for (const e of edges) {
      if (e.from === hovered) related.add(e.to)
      if (e.to === hovered) related.add(e.from)
    }
    return related
  }, [hovered, edges])

  const allNodes = useMemo(() => [...charNodes, ...orgNodes], [charNodes, orgNodes])
  const nodeRadiusScale = 8 + 2 * Math.log10(allNodes.length + 1)

  // 检测鼠标下的节点（基于前一帧的变换）
  const hitTest = useCallback((mx: number, my: number): string | null => {
    const px = (mx - size.w / 2 - offset.x) / scale
    const py = (my - size.h / 2 - offset.y) / scale
    let closest: string | null = null
    let closestDist = 40
    for (const n of allNodes) {
      const p = positions.get(n.id)
      if (!p) continue
      const dx = p.x - px, dy = p.y - py
      const d = Math.sqrt(dx * dx + dy * dy)
      const isOrg = orgNodes.some((o) => o.id === n.id)
      const r = isOrg ? nodeRadiusScale * 1.6 : nodeRadiusScale
      if (d < r + 6 && d < closestDist) {
        closest = n.id
        closestDist = d
      }
    }
    return closest
  }, [allNodes, positions, size, offset, scale, nodeRadiusScale, orgNodes])

  // Canvas 渲染循环
  const rafRef = useRef<number>(0)
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const render = () => {
      const { w, h } = size
      canvas!.width = w
      canvas!.height = h
      ctx!.clearRect(0, 0, w, h)

      // 背景
      ctx!.fillStyle = BG
      ctx!.fillRect(0, 0, w, h)

      ctx!.save()
      ctx!.translate(w / 2 + offset.x, h / 2 + offset.y)
      ctx!.scale(scale, scale)

      // ── 连线 ──
      for (const e of edges) {
        const from = positions.get(e.from), to = positions.get(e.to)
        if (!from || !to) continue
        const dim = hovered !== null && !(e.from === hovered || e.to === hovered)
        ctx!.beginPath()
        const mx = (from.x + to.x) / 2
        const my = (from.y + to.y) / 2 - 10
        ctx!.moveTo(from.x, from.y)
        ctx!.quadraticCurveTo(mx, my, to.x, to.y)
        ctx!.strokeStyle = dim ? '#333' : e.color
        ctx!.lineWidth = dim ? 0.5 : 1.5
        ctx!.globalAlpha = dim ? 0.15 : 0.7
        ctx!.stroke()

        // 中点标签
        if (!dim) {
          ctx!.fillStyle = e.color
          ctx!.font = '10px sans-serif'
          ctx!.textAlign = 'center'
          ctx!.textBaseline = 'bottom'
          ctx!.fillText(e.label, mx, my - 4)
        }
        ctx!.globalAlpha = 1
      }

      // ── 节点 ──
      for (const n of allNodes) {
        const p = positions.get(n.id)
        if (!p) continue
        const isOrg = orgNodes.some((o) => o.id === n.id)
        const r = isOrg ? nodeRadiusScale * 1.6 : nodeRadiusScale
        const dim = hovered !== null && !hoverRelated.has(n.id)
        const color = dim ? '#444' : n.color
        const alpha = dim ? 0.3 : 1

        // 发光效果
        if (!dim) {
          const glow = ctx!.createRadialGradient(p.x, p.y, 0, p.x, p.y, r * 3)
          glow.addColorStop(0, n.color + '44')
          glow.addColorStop(1, 'transparent')
          ctx!.fillStyle = glow
          ctx!.beginPath()
          ctx!.arc(p.x, p.y, r * 3, 0, Math.PI * 2)
          ctx!.fill()
        }

        // 节点圆
        const grad = ctx!.createRadialGradient(p.x - r * 0.3, p.y - r * 0.3, 0, p.x, p.y, r)
        grad.addColorStop(0, lighten(color, 30))
        grad.addColorStop(1, color)
        ctx!.fillStyle = grad
        ctx!.globalAlpha = alpha
        ctx!.beginPath()
        ctx!.arc(p.x, p.y, r, 0, Math.PI * 2)
        ctx!.fill()
        ctx!.globalAlpha = 1

        // 描边
        ctx!.strokeStyle = color
        ctx!.lineWidth = 1.5
        ctx!.globalAlpha = alpha
        ctx!.beginPath()
        ctx!.arc(p.x, p.y, r, 0, Math.PI * 2)
        ctx!.stroke()
        ctx!.globalAlpha = 1

        // 名称标签
        ctx!.fillStyle = dim ? '#555' : '#ddd'
        ctx!.font = `${10 + Math.min(2, Math.log10(allNodes.length + 1))}px sans-serif`
        ctx!.textAlign = 'center'
        ctx!.textBaseline = 'top'
        const label = n.name.length > 6 ? n.name.slice(0, 5) + '…' : n.name
        ctx!.fillText(label, p.x, p.y + r + 3)
      }

      ctx!.restore()
      rafRef.current = requestAnimationFrame(render)
    }

    rafRef.current = requestAnimationFrame(render)
    return () => cancelAnimationFrame(rafRef.current)
  }, [size, positions, edges, allNodes, hovered, hoverRelated, scale, offset, nodeRadiusScale, orgNodes])

  // ── 交互事件 ──
  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault()
    const delta = e.deltaY > 0 ? 0.9 : 1.1
    setScale((s) => Math.max(0.1, Math.min(5, s * delta)))
  }, [])

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    const rect = canvasRef.current?.getBoundingClientRect()
    if (!rect) return
    const mx = e.clientX - rect.left, my = e.clientY - rect.top
    const hit = hitTest(mx, my)
    if (hit) {
      // 点击节点：不拖拽
      dragRef.current.dragging = false
      return
    }
    dragRef.current = {
      dragging: true, startX: e.clientX, startY: e.clientY,
      ox: offset.x, oy: offset.y, panning: false,
    }
  }, [offset, hitTest])

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    const rect = canvasRef.current?.getBoundingClientRect()
    if (!rect) return
    const mx = e.clientX - rect.left, my = e.clientY - rect.top

    if (dragRef.current.dragging) {
      const dx = e.clientX - dragRef.current.startX
      const dy = e.clientY - dragRef.current.startY
      if (Math.abs(dx) > 3 || Math.abs(dy) > 3) {
        dragRef.current.panning = true
      }
      if (dragRef.current.panning) {
        setOffset({ x: dragRef.current.ox + dx, y: dragRef.current.oy + dy })
      }
      return
    }

    // 悬停检测
    const hit = hitTest(mx, my)
    setHovered(hit)
  }, [hitTest])

  const handleMouseUp = useCallback(() => {
    dragRef.current.dragging = false
  }, [])

  const handleMouseLeave = useCallback(() => {
    setHovered(null)
    dragRef.current.dragging = false
  }, [])

  const handleReset = useCallback(() => {
    setScale(1)
    setOffset({ x: 0, y: 0 })
  }, [])

  if (charNodes.length === 0 && orgNodes.length === 0) {
    return (
      <div style={{ width: '100%', height: 300, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#9ca3af', fontSize: 13 }}>
        无数据
      </div>
    )
  }

  return (
    <div
      ref={containerRef}
      style={{ width: '100%', height: '100%', position: 'relative', borderRadius: 8, overflow: 'hidden', background: BG }}
    >
      <canvas
        ref={canvasRef}
        onWheel={handleWheel}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseLeave}
        style={{ display: 'block', cursor: hovered ? 'pointer' : 'grab' }}
      />
      {/* 缩放控制 */}
      <div style={{
        position: 'absolute', top: 8, left: 12,
        display: 'flex', gap: 6, alignItems: 'center',
      }}>
        <span style={{ fontSize: 10, color: '#555', pointerEvents: 'none' }}>
          {Math.round(scale * 100)}%
        </span>
        {scale !== 1 && (
          <button
            onClick={handleReset}
            style={{
              background: 'rgba(255,255,255,0.08)', border: '1px solid #333',
              borderRadius: 4, color: '#aaa', fontSize: 10, cursor: 'pointer',
              padding: '2px 6px',
            }}
          >
            重置
          </button>
        )}
      </div>
      <div style={{ position: 'absolute', top: 8, right: 12, fontSize: 10, color: '#555', pointerEvents: 'none' }}>
        🖱 拖拽平移 · 滚轮缩放 · 悬停高亮
      </div>
      {/* 图例 */}
      <Legend />
    </div>
  )
}

// ── 颜色工具 ──
function lighten(hex: string, percent: number): string {
  const num = parseInt(hex.replace('#', ''), 16)
  const r = Math.min(255, (num >> 16) + Math.round(255 * percent / 100))
  const g = Math.min(255, ((num >> 8) & 0x00FF) + Math.round(255 * percent / 100))
  const b = Math.min(255, (num & 0x0000FF) + Math.round(255 * percent / 100))
  return `rgb(${r},${g},${b})`
}

// ── 图例 ──
const Legend: React.FC = () => (
  <div style={{
    position: 'absolute', bottom: 12, right: 12,
    background: 'rgba(10,10,10,0.85)', borderRadius: 8,
    border: '1px solid #333', padding: '10px 14px',
    fontSize: 11, color: '#9ca3af', fontFamily: 'sans-serif',
    pointerEvents: 'none', userSelect: 'none',
    display: 'flex', gap: 24,
  }}>
    <div>
      <div style={{ color: '#ddd', fontWeight: 600, marginBottom: 4, fontSize: 11 }}>角色</div>
      {Object.entries(roleCN).map(([k, v]) => (
        <div key={k} style={{ display: 'flex', alignItems: 'center', gap: 6, lineHeight: '18px' }}>
          <span style={{ width: 8, height: 8, borderRadius: '50%', background: roleColors[k], flexShrink: 0 }} />
          <span>{v}</span>
        </div>
      ))}
    </div>
    <div>
      <div style={{ color: '#ddd', fontWeight: 600, marginBottom: 4, fontSize: 11 }}>组织</div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, lineHeight: '18px' }}>
        <span style={{ width: 10, height: 10, borderRadius: '50%', background: orgColor, flexShrink: 0 }} />
        <span>势力/组织</span>
      </div>
      <div style={{ marginTop: 6, color: '#ddd', fontWeight: 600, marginBottom: 4 }}>关系</div>
      {Object.entries(relCN).map(([k, v]) => (
        <div key={k} style={{ display: 'flex', alignItems: 'center', gap: 6, lineHeight: '18px' }}>
          <span style={{ width: 12, height: 2, background: relColors[k], flexShrink: 0, borderRadius: 1 }} />
          <span>{v}</span>
        </div>
      ))}
    </div>
  </div>
)

export default RelationGraph
