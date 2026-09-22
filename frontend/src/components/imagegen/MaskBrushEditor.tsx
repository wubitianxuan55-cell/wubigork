// MaskBrushEditor.tsx — 蒙版笔刷编辑器（绘梦阶段二刀 B，规格
// 进度计划/gaea-mask-inpaint-20260923.md）：原图上涂抹圈选「要改的区域」，
// 导出灰度 PNG data URL（白=重绘区、黑=保留区——gaea 统一蒙版契约，
// Go 侧按后端口径各自适配）。strokes 状态驱动：显示层半透明红笔刷、
// 导出层黑底白笔刷同坐标重放；CSS 缩放显示不影响精度（指针坐标按
// getBoundingClientRect 比例映射到画布自然尺寸）。jsdom 无 2d ctx 时导出
// 守卫返回 null（调用方按「未涂抹」处理，不崩）。
import { softTextStyle } from '../../utils/uiStyles'
import React, { useCallback, useRef, useState } from 'react'
import { Button, Slider, Typography } from 'antd'
import { renderMaskDataURL, type MaskStroke } from './ui'

export default function MaskBrushEditor({ src, onMaskChange }: {
  /** 编辑源图（与原图同坐标） */
  src: string
  /** 涂抹/清除后回调：灰度蒙版 data URL；不可用或无笔画时 null */
  onMaskChange: (mask: string | null) => void
}) {
  const [strokes, setStrokes] = useState<MaskStroke[]>([])
  const [brush, setBrush] = useState(36)
  const drawing = useRef(false)
  const current = useRef<MaskStroke | null>(null)
  const wrapRef = useRef<HTMLDivElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const imgRef = useRef<HTMLImageElement>(null)

  const emit = useCallback((next: MaskStroke[]) => {
    const img = imgRef.current
    const w = img?.naturalWidth || 0
    const h = img?.naturalHeight || 0
    onMaskChange(w > 0 && h > 0 ? renderMaskDataURL(next, w, h) : null)
  }, [onMaskChange])

  // 显示层重放：半透明红笔刷（纯视觉，导出走 renderMaskDataURL）
  const repaint = useCallback((all: MaskStroke[], live: MaskStroke | null) => {
    const canvas = canvasRef.current
    const img = imgRef.current
    if (!canvas || !img) return
    canvas.width = img.naturalWidth
    canvas.height = img.naturalHeight
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, canvas.width, canvas.height)
    ctx.fillStyle = 'rgba(239,68,68,0.45)'
    for (const s of [...all, ...(live ? [live] : [])]) {
      for (const [x, y] of s.points) {
        ctx.beginPath()
        ctx.arc(x, y, s.size / 2, 0, Math.PI * 2)
        ctx.fill()
      }
    }
  }, [])

  // 指针坐标 → 画布自然尺寸（CSS 缩放显示下的比例映射）
  const toCanvasPoint = (e: React.PointerEvent): [number, number] | null => {
    const canvas = canvasRef.current
    const img = imgRef.current
    if (!canvas || !img || !img.naturalWidth) return null
    const rect = canvas.getBoundingClientRect()
    if (!rect.width || !rect.height) return null
    const x = ((e.clientX - rect.left) / rect.width) * img.naturalWidth
    const y = ((e.clientY - rect.top) / rect.height) * img.naturalHeight
    return [Math.round(x), Math.round(y)]
  }

  const onPointerDown = (e: React.PointerEvent) => {
    e.preventDefault()
    const p = toCanvasPoint(e)
    if (!p) return
    drawing.current = true
    current.current = { points: [p], size: brush }
    repaint(strokes, current.current)
  }
  const onPointerMove = (e: React.PointerEvent) => {
    if (!drawing.current || !current.current) return
    const p = toCanvasPoint(e)
    if (!p) return
    current.current.points.push(p)
    repaint(strokes, current.current)
  }
  const finishStroke = () => {
    if (!drawing.current || !current.current) return
    drawing.current = false
    const next = [...strokes, current.current]
    current.current = null
    setStrokes(next)
    repaint(next, null)
    emit(next)
  }

  const clear = () => {
    setStrokes([])
    current.current = null
    repaint([], null)
    onMaskChange(null)
  }

  return (
    <div data-testid="mask-brush-editor" style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      <div ref={wrapRef} style={{ position: 'relative', lineHeight: 0, borderRadius: 8, overflow: 'hidden' }}>
        <img
          ref={imgRef} src={src} alt="蒙版底图" crossOrigin="anonymous"
          style={{ maxWidth: '100%', maxHeight: 260, objectFit: 'contain', display: 'block', margin: '0 auto' }} />
        <canvas
          ref={canvasRef} data-testid="mask-brush-canvas"
          style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', cursor: 'crosshair', touchAction: 'none' }}
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={finishStroke}
          onPointerLeave={finishStroke} />
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>笔刷</Typography.Text>
        <span data-testid="mask-brush-size" style={{ flex: 1 }}>
          <Slider style={{ margin: 0 }} min={8} max={120} value={brush}
            onChange={v => setBrush(typeof v === 'number' ? v : brush)} />
        </span>
        <Button size="small" data-testid="mask-brush-clear" onClick={clear}>清除</Button>
      </div>
      <div style={{ ...softTextStyle, fontSize: 12 }} data-testid="mask-brush-status">
        {strokes.length > 0 ? `已涂选 ${strokes.length} 笔（只改涂红区域，其余保持原样）` : '在图上涂抹要修改的区域'}
      </div>
    </div>
  )
}
