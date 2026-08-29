import React, { useRef, useEffect } from 'react'
import { C } from '../utils/theme'

/**
 * 语言粒子交互球（未来感语音核心）。
 *
 * 调研对齐主流 AI 语音界面范式（ChatGPT Advanced Voice / Siri / 豆包）：
 *   - 居中动态球体随音量呼吸、随状态变色，作为「AI 正在听/正在说」唯一信号
 *   - 增量叠加「粒子连线网络 + 环绕粒子轨道」体现语言粒子聚合与待机自转
 *
 * 三态配色：用户说话=暖橙红 / AI 回复=电光蓝 / 空闲=星云紫
 */

interface Particle {
  x: number; y: number; z: number
  vx: number; vy: number
  baseX: number; baseY: number; baseZ: number
  size: number
  alpha: number
}

interface OrbitParticle {
  angle: number
  speed: number
  ring: number      // 0 / 1 内外两圈
  radius: number    // 轨道半径比例
  yOff: number      // 垂直椭圆投影压缩
  size: number
  alpha: number
}

interface Props {
  volume: number
  listening: boolean
  speaking: boolean
  aiSpeaking: boolean
  transcript: string
  size?: number
}

const N = 90          // 球面粒子数
const R = 0.62        // 球面半径比例
const LINK_MAX = 74   // 连线距离阈值（像素）
const ORBIT_N = 26    // 每圈轨道粒子数

const VoiceChatOrb: React.FC<Props> = ({
  volume, listening, speaking, aiSpeaking, transcript, size = 360,
}) => {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const particlesRef = useRef<Particle[]>([])
  const orbitsRef = useRef<OrbitParticle[]>([])
  const animRef = useRef<number>(0)
  const tmRef = useRef(0)
  const propsRef = useRef({ volume, listening, speaking, aiSpeaking })
  propsRef.current = { volume, listening, speaking, aiSpeaking }

  useEffect(() => {
    const cvs = canvasRef.current
    if (!cvs) return
    const ctx = cvs.getContext('2d')!
    const dpr = Math.min(window.devicePixelRatio || 1, 2)
    cvs.width = size * dpr
    cvs.height = size * dpr
    cvs.style.width = `${size}px`
    cvs.style.height = `${size}px`
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)

    const cx = size / 2
    const cy = size / 2

    if (particlesRef.current.length === 0) {
      const rr = size * R * 0.5
      const ps: Particle[] = []
      for (let i = 0; i < N; i++) {
        const theta = Math.random() * Math.PI * 2
        const phi = Math.acos(2 * Math.random() - 1)
        const bx = Math.sin(phi) * Math.cos(theta) * rr
        const by = Math.sin(phi) * Math.sin(theta) * rr
        const bz = Math.cos(phi)
        ps.push({ x: cx + bx, y: cy + by, z: bz, vx: 0, vy: 0, baseX: bx, baseY: by, baseZ: bz, size: 1.5 + Math.random() * 2, alpha: 0.5 + Math.random() * 0.5 })
      }
      particlesRef.current = ps
    }

    if (orbitsRef.current.length === 0) {
      const orbs: OrbitParticle[] = []
      for (let ring = 0; ring < 2; ring++) {
        for (let i = 0; i < ORBIT_N; i++) {
          orbs.push({
            angle: (i / ORBIT_N) * Math.PI * 2 + ring * 0.7,
            speed: (ring === 0 ? 0.28 : -0.2) * (0.9 + Math.random() * 0.25),
            ring,
            radius: (ring === 0 ? 0.78 : 0.95),
            yOff: ring === 0 ? 0.32 : 0.22,
            size: ring === 0 ? 1.4 + Math.random() * 1.2 : 1 + Math.random() * 1,
            alpha: 0.25 + Math.random() * 0.3,
          })
        }
      }
      orbitsRef.current = orbs
    }

    const sv = { v: 0 }

    const frame = () => {
      tmRef.current += 0.016
      const t = tmRef.current
      const { volume: vr, listening: lis, speaking: spk, aiSpeaking: ai } = propsRef.current
      ctx.clearRect(0, 0, size, size)

      sv.v += (vr - sv.v) * 0.15
      const vol = Math.max(sv.v, 0.01)
      const act = lis || ai

      const c1 = spk ? [255, 140, 80] : ai ? [100, 200, 255] : [120, 140, 220]
      const c2 = spk ? [255, 80, 120] : ai ? [80, 160, 240] : [80, 100, 200]

      // 光晕
      const g = ctx.createRadialGradient(cx, cy, 0, cx, cy, size * 0.3 * (1 + vol))
      const ga = act ? 0.12 + vol * 0.15 : 0.04
      g.addColorStop(0, `rgba(${c1.join(',')},${ga * 1.5})`)
      g.addColorStop(0.6, `rgba(${c1.join(',')},${ga * 0.3})`)
      g.addColorStop(1, 'rgba(0,0,0,0)')
      ctx.fillStyle = g
      ctx.fillRect(0, 0, size, size)

      const ef = act ? 1 + vol * 0.5 : 1
      const ps = particlesRef.current

      // ── 更新粒子位置 ──
      for (let i = 0; i < ps.length; i++) {
        const p = ps[i]
        const tx = cx + p.baseX * ef + Math.sin(t * 3 + i * 0.3) * (act ? 3 + vol * 8 : 1.5)
        const ty = cy + p.baseY * ef + Math.cos(t * 2.7 + i * 0.5) * (act ? 2 + vol * 6 : 1)
        p.vx += (tx - p.x) * 0.08
        p.vy += (ty - p.y) * 0.08
        p.vx *= 0.85
        p.vy *= 0.85
        p.x += p.vx
        p.y += p.vy
      }

      // ── 粒子连线网络（距离阈值内画线，亮度随音量）──
      const linkBase = act ? 0.28 + vol * 0.35 : 0.12
      ctx.lineWidth = 1
      for (let i = 0; i < ps.length; i++) {
        for (let j = i + 1; j < ps.length; j++) {
          const dx = ps[i].x - ps[j].x
          const dy = ps[i].y - ps[j].y
          const d2 = dx * dx + dy * dy
          if (d2 < LINK_MAX * LINK_MAX) {
            const d = Math.sqrt(d2)
            const a = (1 - d / LINK_MAX) * linkBase
            if (a > 0.02) {
              ctx.strokeStyle = `rgba(${c1.join(',')},${a})`
              ctx.beginPath()
              ctx.moveTo(ps[i].x, ps[i].y)
              ctx.lineTo(ps[j].x, ps[j].y)
              ctx.stroke()
            }
          }
        }
      }

      // ── 球面粒子 ──
      for (let i = 0; i < ps.length; i++) {
        const p = ps[i]
        const tc = (p.baseZ + 1) / 2
        const rr = Math.round(c1[0] + (c2[0] - c1[0]) * tc)
        const gg = Math.round(c1[1] + (c2[1] - c1[1]) * tc)
        const bb = Math.round(c1[2] + (c2[2] - c1[2]) * tc)
        const a = act ? p.alpha * (0.6 + vol * 0.4) : p.alpha * 0.45

        ctx.beginPath()
        ctx.arc(p.x, p.y, p.size * (act ? 1 + vol * 0.4 : 1), 0, Math.PI * 2)
        ctx.fillStyle = `rgb(${rr},${gg},${bb})`
        ctx.globalAlpha = a
        ctx.fill()
      }

      // ── 环绕粒子轨道（音量膨胀，静默慢自转）──
      const orbitR = size * 0.46 * (1 + vol * (act ? 0.16 : 0.04))
      for (const o of orbitsRef.current) {
        o.angle += o.speed * (act ? 1 + vol * 1.5 : 1) * 0.016
        const r = orbitR * o.radius
        const ox = cx + Math.cos(o.angle) * r
        const oy = cy + Math.sin(o.angle) * r * o.yOff
        const glow = act ? 0.5 + vol * 0.5 : 0.3
        ctx.beginPath()
        ctx.arc(ox, oy, o.size, 0, Math.PI * 2)
        ctx.fillStyle = `rgb(${c1.join(',')})`
        ctx.globalAlpha = o.alpha * glow
        ctx.fill()
      }
      ctx.globalAlpha = 1

      // 核心光
      if (act) {
        const cg = ctx.createRadialGradient(cx, cy, 0, cx, cy, 10 + vol * 14)
        cg.addColorStop(0, `rgba(255,255,255,${0.2 + vol * 0.4})`)
        cg.addColorStop(1, 'rgba(0,0,0,0)')
        ctx.fillStyle = cg
        ctx.fillRect(cx - 20, cy - 20, 40, 40)
      }

      animRef.current = requestAnimationFrame(frame)
    }

    animRef.current = requestAnimationFrame(frame)
    return () => { if (animRef.current) cancelAnimationFrame(animRef.current) }
  }, [size])

  return (
    <div style={{ position: 'relative', width: size, height: size, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <canvas ref={canvasRef} style={{ borderRadius: '50%' }} />

      {transcript && (
        <div style={{ position: 'absolute', bottom: -10, left: '50%', transform: 'translateX(-50%)', maxWidth: size + 80, textAlign: 'center', color: C('color-text'), fontSize: 14, opacity: 0.8, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
          {transcript}
        </div>
      )}

      <div style={{ position: 'absolute', top: -10, left: '50%', transform: 'translateX(-50%)', color: aiSpeaking ? '#64b5f6' : speaking ? '#ff8a65' : C('color-text-secondary'), fontSize: 12, fontWeight: 500, opacity: listening || aiSpeaking ? 0.9 : 0, transition: 'opacity 0.3s' }}> {/* hex-exempt 语音双方品牌识别色 */}
        {aiSpeaking ? 'AI 回复中...' : speaking ? '正在聆听...' : listening ? '准备聆听' : ''}
      </div>
    </div>
  )
}

export default VoiceChatOrb
