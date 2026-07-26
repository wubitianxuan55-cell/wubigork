import React, { useRef, useEffect, useState, useMemo } from 'react'
import { Typography, Tag, Spin, Empty } from 'antd'

/**
 * StoryGraph — 2D 故事知识图谱可视化
 *
 * 使用 Canvas 实现简单力导向布局，展示角色/地点/组织/物品之间的关系网络
 * 节点颜色按实体类型区分，悬停高亮，点击查看详情
 *
 * Props:
 *   nodes — 图节点列表
 *   edges — 图边列表
 *   onNodeClick — 点击节点回调
 *   loading — 加载状态
 */
interface GraphNode {
  id: string
  name: string
  type: string
  group: number
}

interface GraphEdge {
  from: string
  to: string
  type: string
  label?: string
}

interface StoryGraphProps {
  nodes: GraphNode[]
  edges: GraphEdge[]
  onNodeClick?: (node: GraphNode) => void
  loading?: boolean
}

const TYPE_COLORS: Record<string, string> = {
  character: '#4ade80',
  organization: '#60a5fa',
  location: '#f59e0b',
  item: '#c084fc',
  event: '#f87171',
  concept: '#9ca3af',
}

const StoryGraph: React.FC<StoryGraphProps> = ({ nodes, edges, onNodeClick, loading }) => {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [hoveredNode, setHoveredNode] = useState<GraphNode | null>(null)

  // ── 力导向布局（共享计算，桌面 Canvas 和移动 DOM 都使用）──
  const layout = useMemo(() => {
    if (nodes.length === 0) return { simNodes: [], nodeMap: new Map() }

    const W = 800
    const H = 400
    const centerX = W / 2
    const centerY = H / 2

    interface SimNode {
      id: string
      name: string
      type: string
      group: number
      x: number
      y: number
      vx: number
      vy: number
    }

    const simNodes: SimNode[] = nodes.map((n) => ({
      ...n,
      x: centerX + (Math.random() - 0.5) * W * 0.6,
      y: centerY + (Math.random() - 0.5) * H * 0.6,
      vx: 0,
      vy: 0,
    }))

    const nodeMap = new Map(simNodes.map(n => [n.id, n]))

    const ITERATIONS = 200
    for (let iter = 0; iter < ITERATIONS; iter++) {
      const alpha = 1 - iter / ITERATIONS

      for (let i = 0; i < simNodes.length; i++) {
        for (let j = i + 1; j < simNodes.length; j++) {
          const dx = simNodes[j].x - simNodes[i].x
          const dy = simNodes[j].y - simNodes[i].y
          const dist = Math.sqrt(dx * dx + dy * dy) || 1
          const force = (80 * 80) / dist * alpha * 0.02
          const fx = (dx / dist) * force
          const fy = (dy / dist) * force
          simNodes[i].vx -= fx
          simNodes[i].vy -= fy
          simNodes[j].vx += fx
          simNodes[j].vy += fy
        }
      }

      for (const edge of edges) {
        const from = nodeMap.get(edge.from)
        const to = nodeMap.get(edge.to)
        if (!from || !to) continue
        const dx = to.x - from.x
        const dy = to.y - from.y
        const dist = Math.sqrt(dx * dx + dy * dy) || 1
        const force = (dist - 100) * alpha * 0.005
        const fx = (dx / dist) * force
        const fy = (dy / dist) * force
        from.vx += fx
        from.vy += fy
        to.vx -= fx
        to.vy -= fy
      }

      for (const n of simNodes) {
        n.vx += (centerX - n.x) * alpha * 0.001
        n.vy += (centerY - n.y) * alpha * 0.001
      }

      for (const n of simNodes) {
        n.x += n.vx
        n.y += n.vy
        n.vx *= 0.6
        n.vy *= 0.6
        n.x = Math.max(40, Math.min(W - 40, n.x))
        n.y = Math.max(40, Math.min(H - 40, n.y))
      }
    }

    return { simNodes, nodeMap }
  }, [nodes, edges])

  const { simNodes, nodeMap } = layout

  // ── 桌面 Canvas 渲染 ──
  useEffect(() => {
    if (!canvasRef.current || nodes.length === 0) return

    const canvas = canvasRef.current
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const dpr = window.devicePixelRatio || 1
    const rect = canvas.getBoundingClientRect()
    canvas.width = rect.width * dpr
    canvas.height = rect.height * dpr
    ctx.scale(dpr, dpr)

    const W = rect.width
    const H = rect.height

    // 缩放坐标到实际画布尺寸
    const scaleX = W / 800
    const scaleY = H / 400

    const render = () => {
      ctx.clearRect(0, 0, W, H)

      for (const edge of edges) {
        const from = nodeMap.get(edge.from)
        const to = nodeMap.get(edge.to)
        if (!from || !to) continue

        ctx.beginPath()
        ctx.moveTo(from.x * scaleX, from.y * scaleY)
        ctx.lineTo(to.x * scaleX, to.y * scaleY)
        ctx.strokeStyle = 'rgba(255,255,255,0.1)'
        ctx.lineWidth = 1
        ctx.stroke()
      }

      for (const n of simNodes) {
        const color = TYPE_COLORS[n.type] || '#9ca3af'
        const isHovered = hoveredNode?.id === n.id
        const r = isHovered ? 24 : 18
        const nx = n.x * scaleX
        const ny = n.y * scaleY

        if (isHovered) {
          ctx.beginPath()
          ctx.arc(nx, ny, r + 6, 0, Math.PI * 2)
          ctx.fillStyle = color + '30'
          ctx.fill()
        }

        ctx.beginPath()
        ctx.arc(nx, ny, r, 0, Math.PI * 2)
        ctx.fillStyle = color
        ctx.fill()
        ctx.strokeStyle = 'var(--bg-deep)'
        ctx.lineWidth = 2
        ctx.stroke()

        ctx.font = `${isHovered ? 12 : 10}px Inter, sans-serif`
        ctx.fillStyle = 'var(--color-text)'
        ctx.textAlign = 'center'
        ctx.fillText(n.name, nx, ny + r + 14)
      }
    }

    render()

    const handleMouseMove = (e: MouseEvent) => {
      const mx = e.clientX - rect.left
      const my = e.clientY - rect.top

      let found: GraphNode | null = null
      for (const n of simNodes) {
        const nx = n.x * scaleX
        const ny = n.y * scaleY
        const dx = nx - mx
        const dy = ny - my
        if (dx * dx + dy * dy < 24 * 24) {
          found = { id: n.id, name: n.name, type: n.type, group: n.group }
          break
        }
      }
      setHoveredNode(found)
      canvas.style.cursor = found ? 'pointer' : 'default'
    }

    const handleClick = (e: MouseEvent) => {
      const mx = e.clientX - rect.left
      const my = e.clientY - rect.top

      for (const n of simNodes) {
        const nx = n.x * scaleX
        const ny = n.y * scaleY
        const dx = nx - mx
        const dy = ny - my
        if (dx * dx + dy * dy < 24 * 24) {
          onNodeClick?.({ id: n.id, name: n.name, type: n.type, group: n.group })
          break
        }
      }
    }

    canvas.addEventListener('mousemove', handleMouseMove)
    canvas.addEventListener('click', handleClick)

    return () => {
      canvas.removeEventListener('mousemove', handleMouseMove)
      canvas.removeEventListener('click', handleClick)
    }
  }, [nodes, edges, hoveredNode, onNodeClick, simNodes, nodeMap])

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 400 }}>
        <Spin tip="构建知识图谱..." />
      </div>
    )
  }

  if (nodes.length === 0) {
    return <Empty description="暂无实体数据，请先同步实体数据库" />
  }

  return (
    <div style={{ position: 'relative' }}>
      /* ── Desktop: Canvas 渲染 ── */
      <>
        <canvas
          ref={canvasRef}
          style={{ width: '100%', height: 400, borderRadius: 'var(--radius-lg)', background: 'var(--bg-deep)' }}
        />
        {hoveredNode && (
          <div
            style={{
              position: 'absolute',
              top: 8,
              right: 8,
              background: 'var(--bg-elevated)',
              borderRadius: 'var(--radius-md)',
              padding: '6px 12px',
              border: '1px solid var(--border-subtle)',
              fontSize: 12,
            }}
          >
            <Typography.Text strong>{hoveredNode.name}</Typography.Text>
            <Tag style={{ marginLeft: 6, fontSize: 10 }}>{hoveredNode.type}</Tag>
          </div>
        )}
      </>
      {/* 图例 */}
      <div style={{ display: 'flex', gap: 8, padding: '8px 0', flexWrap: 'wrap' }}>
        {Object.entries(TYPE_COLORS).map(([type, color]) => (
          <Tag key={type} color={color} style={{ fontSize: 11, margin: 0 }}>
            {type}
          </Tag>
        ))}
      </div>
    </div>
  )
}

export default StoryGraph
export type { StoryGraphProps, GraphNode, GraphEdge }
