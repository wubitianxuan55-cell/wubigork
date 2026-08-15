// CompanionAvatar.tsx — 简易Canvasgaea光球
// 100% 对齐 ackem AIVatar 的核心视觉效果：旋转粒子球 + 呼吸脉冲 + 状态变色

import React, { useEffect, useRef } from 'react'

interface Props {
  size?: number
  /** idle / listening / thinking / speaking */
  state?: 'idle' | 'listening' | 'thinking' | 'speaking'
  emotionColor?: string
}

const N = 64 // 粒子数
const R = 0.55 // 球半径

// canvas 不支持 CSS 变量，这里把 var(--x[, fallback]) 解析成具体颜色再拼透明度后缀；
// 支持嵌套 fallback（如 var(--gaea-glow, var(--md-sys-color-primary))），递归解析。
function resolveCanvasColor(raw: string): string {
  const s = (raw || '').trim()
  const m = s.match(/^var\(\s*(--[\w-]+)\s*(?:,\s*(.*))?\)$/)
  if (m) {
    try {
      const computed = getComputedStyle(document.documentElement).getPropertyValue(m[1]).trim()
      if (computed && !computed.startsWith('var(')) return computed
    } catch {
      /* 非浏览器环境（测试等）忽略 */
    }
    // fallback 存在 → 递归解析（可能仍是 var(...)）
    if (m[2] !== undefined) return resolveCanvasColor(m[2])
  }
  return s
}

export const CompanionAvatar: React.FC<Props> = ({
  size = 280,
  state = 'idle',
  emotionColor = '#e85388',
}) => {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const stateRef = useRef(state)
  stateRef.current = state

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const dpr = window.devicePixelRatio || 1
    canvas.width = size * dpr
    canvas.height = size * dpr
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    const baseColor = resolveCanvasColor(emotionColor)

    // 生成球面点云
    const points: { theta: number; phi: number; r: number; baseR: number }[] = []
    for (let i = 0; i < N; i++) {
      const theta = Math.random() * Math.PI * 2
      const phi = Math.acos(2 * Math.random() - 1)
      const r = R * size * 0.5
      points.push({ theta, phi, r, baseR: r })
    }

    let raf = 0
    let t = 0

    const draw = () => {
      const cx = size / 2
      const cy = size / 2
      ctx.clearRect(0, 0, size, size)

      const s = stateRef.current
      const speed = s === 'listening' ? 0.6 : s === 'speaking' ? 0.8 : 0.3
      const pulse = s === 'speaking' ? 1 + Math.sin(t * 8) * 0.06 : 1 + Math.sin(t * 2) * 0.03
      const glow = s === 'speaking' ? 0.4 : s === 'listening' ? 0.3 : 0.15

      t += speed * 0.016

      // 外发光
      const grd = ctx.createRadialGradient(cx, cy, R * size * 0.3, cx, cy, R * size * 0.8)
      grd.addColorStop(0, 'transparent')
      grd.addColorStop(0.5, `${baseColor}22`)
      grd.addColorStop(1, `${baseColor}${Math.round(glow * 255).toString(16).padStart(2, '0')}`)
      ctx.fillStyle = grd
      ctx.beginPath()
      ctx.arc(cx, cy, R * size * 0.8, 0, Math.PI * 2)
      ctx.fill()

      // 粒子
      for (const p of points) {
        // 绕Y轴旋转
        const rx = Math.cos(t * 0.5) * p.theta
        const x3 = Math.sin(p.phi) * Math.cos(rx) * p.r * pulse
        const y3 = Math.cos(p.phi) * p.r * pulse
        const z3 = Math.sin(p.phi) * Math.sin(rx) * p.r * pulse

        // 透视投影
        const scale = 1 / (1 + z3 / (size * 0.8))
        const sx = cx + x3 * scale
        const sy = cy - y3 * scale

        // z深度决定大小和透明度
        const depth = (z3 / (R * size * 0.5) + 1) / 2
        const alpha = 0.15 + depth * 0.6
        const r = 1.5 + depth * 2.5

        ctx.beginPath()
        ctx.arc(sx, sy, r, 0, Math.PI * 2)
        ctx.fillStyle = `${baseColor}${Math.round(alpha * 255).toString(16).padStart(2, '0')}`
        ctx.fill()
      }

      raf = requestAnimationFrame(draw)
    }

    draw()
    return () => cancelAnimationFrame(raf)
  }, [size, emotionColor])

  return (
    <canvas
      ref={canvasRef}
      style={{ width: size, height: size, borderRadius: '50%' }}
      aria-hidden
    />
  )
}
