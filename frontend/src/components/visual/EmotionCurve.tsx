import React, { useEffect, useRef, useState, useCallback } from 'react'
import { Typography, Spin, Empty, Segmented } from 'antd'

/**
 * EmotionCurve — 章节情绪曲线
 *
 * Canvas 折线图展示各章的紧张度(valence)和正负情感(tension)
 * 支持切换 tension/valence 视图
 */
interface EmotionPoint {
  chapter_num: number
  label: string
  emotion: string
  tension: number
  valence: number
  word_count: number
}

const EmotionCurve: React.FC = () => {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [points, setPoints] = useState<EmotionPoint[]>([])
  const [loading, setLoading] = useState(true)
  const [mode, setMode] = useState<string>('tension')
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null)

  useEffect(() => {
    const load = async () => {
      setLoading(true)
      try {
        // @ts-ignore
        const result = await window.go.app.App.ExtractEmotionCurve()
        setPoints(Array.isArray(result) ? result : [])
      } catch (_) {
        setPoints([])
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  // 渲染曲线
  useEffect(() => {
    if (!canvasRef.current || points.length === 0) return

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
    const pad = { top: 30, right: 30, bottom: 40, left: 40 }
    const chartW = W - pad.left - pad.right
    const chartH = H - pad.top - pad.bottom

    // 清屏
    ctx.clearRect(0, 0, W, H)

    // 背景
    ctx.fillStyle = 'var(--bg-deep)'
    ctx.fillRect(0, 0, W, H)

    // 网格线
    ctx.strokeStyle = 'rgba(255,255,255,0.05)'
    ctx.lineWidth = 1
    for (let i = 0; i <= 5; i++) {
      const y = pad.top + (chartH / 5) * i
      ctx.beginPath()
      ctx.moveTo(pad.left, y)
      ctx.lineTo(W - pad.right, y)
      ctx.stroke()

      // Y 轴标签
      const val = mode === 'tension' ? 10 - i * 2 : 5 - i * 2
      ctx.fillStyle = 'rgba(255,255,255,0.3)'
      ctx.font = '10px Inter, sans-serif'
      ctx.textAlign = 'right'
      ctx.fillText(String(val), pad.left - 6, y + 3)
    }

    if (points.length < 2) return

    // 数据点坐标
    const getX = (i: number) => pad.left + (chartW / (points.length - 1)) * i
    const getY = (val: number) => {
      const max = mode === 'tension' ? 10 : 5
      const min = mode === 'tension' ? 0 : -5
      const ratio = (val - min) / (max - min)
      return pad.top + chartH - ratio * chartH
    }

    const coords = points.map((p, i) => {
      const val = mode === 'tension' ? p.tension : p.valence
      return { x: getX(i), y: getY(val), ...p }
    })

    // 填充渐变
    const gradient = ctx.createLinearGradient(0, pad.top, 0, H - pad.bottom)
    if (mode === 'tension') {
      gradient.addColorStop(0, 'rgba(248, 113, 113, 0.3)')
      gradient.addColorStop(1, 'rgba(248, 113, 113, 0.02)')
    } else {
      gradient.addColorStop(0, 'rgba(96, 165, 250, 0.3)')
      gradient.addColorStop(1, 'rgba(96, 165, 250, 0.02)')
    }

    ctx.beginPath()
    ctx.moveTo(coords[0].x, pad.top + chartH)
    for (const c of coords) {
      ctx.lineTo(c.x, c.y)
    }
    ctx.lineTo(coords[coords.length - 1].x, pad.top + chartH)
    ctx.closePath()
    ctx.fillStyle = gradient
    ctx.fill()

    // 折线
    const lineColor = mode === 'tension' ? '#f87171' : '#60a5fa'
    ctx.beginPath()
    ctx.moveTo(coords[0].x, coords[0].y)
    for (let i = 1; i < coords.length; i++) {
      const cp1x = (coords[i - 1].x + coords[i].x) / 2
      const cp1y = coords[i - 1].y
      const cp2x = (coords[i - 1].x + coords[i].x) / 2
      const cp2y = coords[i].y
      ctx.bezierCurveTo(cp1x, cp1y, cp2x, cp2y, coords[i].x, coords[i].y)
    }
    ctx.strokeStyle = lineColor
    ctx.lineWidth = 2.5
    ctx.stroke()

    // 数据点
    coords.forEach((c, i) => {
      const isHovered = i === hoveredIdx
      const r = isHovered ? 6 : 3

      ctx.beginPath()
      ctx.arc(c.x, c.y, r + 3, 0, Math.PI * 2)
      ctx.fillStyle = lineColor + '30'
      ctx.fill()

      ctx.beginPath()
      ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
      ctx.fillStyle = lineColor
      ctx.fill()
      ctx.strokeStyle = 'var(--bg-deep)'
      ctx.lineWidth = 1.5
      ctx.stroke()

      // X 轴标签
      ctx.fillStyle = 'rgba(255,255,255,0.4)'
      ctx.font = '9px Inter, sans-serif'
      ctx.textAlign = 'center'
      ctx.fillText(`Ch${c.chapter_num}`, c.x, H - pad.bottom + 16)
    })

    // 悬停 tooltip
    if (hoveredIdx !== null && hoveredIdx < coords.length) {
      const c = coords[hoveredIdx]
      ctx.fillStyle = 'var(--bg-elevated)'
      ctx.strokeStyle = lineColor
      ctx.lineWidth = 1
      const tw = 120
      const th = 40
      const tx = Math.min(c.x - tw / 2, W - tw - 10)
      const ty = c.y - th - 12
      ctx.beginPath()
      ctx.roundRect(Math.max(10, tx), Math.max(0, ty), tw, th, 6)
      ctx.fill()
      ctx.stroke()

      ctx.fillStyle = 'var(--color-text)'
      ctx.font = '10px Inter, sans-serif'
      ctx.textAlign = 'center'
      ctx.fillText(
        `${c.emotion || '—'} | ${mode === 'tension' ? '紧张度' : '情感'}: ${(mode === 'tension' ? c.tension : c.valence).toFixed(1)}`,
        Math.max(10, tx) + tw / 2,
        Math.max(0, ty) + 22,
      )
    }
  }, [points, mode, hoveredIdx])

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    if (!canvasRef.current || points.length < 2) return
    const rect = canvasRef.current.getBoundingClientRect()
    const mx = e.clientX - rect.left
    const pad = 40
    const chartW = rect.width - pad - 30

    const idx = Math.round(((mx - pad) / chartW) * (points.length - 1))
    if (idx >= 0 && idx < points.length) {
      setHoveredIdx(idx)
    } else {
      setHoveredIdx(null)
    }
  }, [points])

  if (loading) return <div style={{ textAlign: 'center', padding: 40 }}><Spin tip="加载情绪曲线..." /></div>
  if (points.length === 0) return <Empty description="暂无章节数据" />

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Typography.Text strong style={{ fontSize: 13 }}>📈 情绪曲线</Typography.Text>
        <Segmented
          size="small"
          value={mode}
          onChange={val => setMode(val as string)}
          options={[
            { label: '紧张度', value: 'tension' },
            { label: '正负情感', value: 'valence' },
          ]}
        />
      </div>
      <canvas
        ref={canvasRef}
        style={{ width: '100%', height: 220, borderRadius: 'var(--radius-lg)', cursor: 'crosshair' }}
        onMouseMove={handleMouseMove}
        onMouseLeave={() => setHoveredIdx(null)}
      />
    </div>
  )
}

export default EmotionCurve
