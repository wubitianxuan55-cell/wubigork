import React from 'react'
import { AudioOutlined, LoadingOutlined, SoundOutlined } from '@ant-design/icons'
import type { VoiceSessionState } from '../session/voiceSession'

type RGB = readonly [number, number, number]

const GOLDEN_ANGLE = Math.PI * (3 - Math.sqrt(5))

function parseColor(value: string, fallback: RGB): RGB {
  const hex = value.match(/^#([0-9a-f]{3,8})$/i)
  if (hex) {
    const raw = hex[1].length === 3
      ? hex[1].split('').map((channel) => channel + channel).join('')
      : hex[1].slice(0, 6)
    return [
      parseInt(raw.slice(0, 2), 16),
      parseInt(raw.slice(2, 4), 16),
      parseInt(raw.slice(4, 6), 16),
    ]
  }
  const channels = value.match(/[\d.]+/g)?.map(Number)
  if (!channels || channels.length < 3) return fallback
  const scale = value.startsWith('color(') ? 255 : 1
  return [channels[0] * scale, channels[1] * scale, channels[2] * scale]
}

function mix(a: RGB, b: RGB, amount: number): RGB {
  return [
    a[0] + (b[0] - a[0]) * amount,
    a[1] + (b[1] - a[1]) * amount,
    a[2] + (b[2] - a[2]) * amount,
  ]
}

function rgba(color: RGB, alpha: number): string {
  return `rgba(${Math.round(color[0])}, ${Math.round(color[1])}, ${Math.round(color[2])}, ${alpha})`
}

function seededNoise(seed: number): number {
  const value = Math.sin(seed * 12.9898) * 43758.5453
  return value - Math.floor(value)
}

export interface VoiceV2OrbProps {
  phase: VoiceSessionState['phase']
  active: boolean
  connected: boolean
  /** Live microphone RMS in the 0..1 range. */
  level?: number
}

/** Shared visual core for the product console and the V2 protocol lab. */
export const VoiceV2Orb: React.FC<VoiceV2OrbProps> = ({ phase, active, connected, level = 0 }) => {
  const canvasRef = React.useRef<HTMLCanvasElement>(null)
  const propsRef = React.useRef({ phase, active, connected, level })
  const redrawRef = React.useRef<(() => void) | null>(null)
  propsRef.current = { phase, active, connected, level }

  const PhaseIcon = phase === 'thinking'
    ? LoadingOutlined
    : phase === 'speaking'
      ? SoundOutlined
      : AudioOutlined

  React.useEffect(() => {
    const canvas = canvasRef.current
    const host = canvas?.parentElement
    const ctx = canvas?.getContext('2d')
    if (!canvas || !host || !ctx) return

    const shouldReduce = () => window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
      || Boolean(host.closest('.ui-reduced-motion'))

    const shellCount = 320
    const particles = Array.from({ length: 448 }, (_, index) => {
      const onShell = index < shellCount
      const localIndex = onShell ? index : index - shellCount
      const localCount = onShell ? shellCount : 448 - shellCount
      const seed = seededNoise(index + 3)
      let x: number
      let y: number
      let z: number

      if (onShell) {
        y = 1 - (localIndex / Math.max(1, localCount - 1)) * 2
        const ringRadius = Math.sqrt(Math.max(0, 1 - y * y))
        const theta = GOLDEN_ANGLE * localIndex
        const surfaceJitter = 0.88 + seededNoise(index + 17) * 0.12
        x = Math.cos(theta) * ringRadius * surfaceJitter
        z = Math.sin(theta) * ringRadius * surfaceJitter
      } else {
        y = seededNoise(index * 5 + 1) * 2 - 1
        const ringRadius = Math.sqrt(Math.max(0, 1 - y * y))
        const theta = seededNoise(index * 7 + 11) * Math.PI * 2
        const volumeRadius = Math.cbrt(seededNoise(index * 11 + 23)) * 0.76
        x = Math.cos(theta) * ringRadius * volumeRadius
        z = Math.sin(theta) * ringRadius * volumeRadius
      }

      return {
        x,
        y,
        z,
        seed,
        size: 0.46 + seededNoise(index * 13 + 31) * 1.38,
        onShell,
      }
    })

    let width = 1
    let height = 1
    let dpr = 1
    let raf = 0
    let running = true
    let start = performance.now()
    let lastPaint = 0
    let cachedAccent: RGB = [86, 205, 190]
    let cachedInk: RGB = [235, 246, 244]
    let lastPaletteRead = 0

    const draw = (now: number, animate: boolean) => {
      if (document.hidden || width <= 1 || height <= 1) return
      const current = propsRef.current
      const live = Math.max(0, Math.min(1, current.level))

      if (now - lastPaletteRead > 500) {
        cachedAccent = parseColor(getComputedStyle(host).color, cachedAccent)
        const core = host.querySelector<HTMLElement>('.vc2-orb-core')
        cachedInk = parseColor(core ? getComputedStyle(core).color : getComputedStyle(host).color, cachedInk)
        lastPaletteRead = now
      }

      ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
      ctx.clearRect(0, 0, width, height)

      const elapsed = Math.max(0, now - start)
      const motion = animate ? elapsed : 2400
      const speed = current.phase === 'thinking' ? 0.0005 : current.phase === 'speaking' ? 0.00038 : 0.0002
      const rotation = motion * speed
      const ambientPulse = Math.sin(motion * 0.0014) * 0.012
      const pulse = ambientPulse + live * 0.065 + (current.phase === 'thinking' ? Math.sin(motion * 0.0032) * 0.018 : 0)
      const orbRadius = Math.min(width, height) * 0.365 * (1 + pulse)
      const cx = width / 2
      const cy = height / 2
      const energy = current.connected ? 1 : 0.72
      const light = mix(cachedAccent, cachedInk, 0.56)

      ctx.save()
      ctx.globalCompositeOperation = 'lighter'

      const halo = ctx.createRadialGradient(cx, cy, orbRadius * 0.24, cx, cy, orbRadius * 1.78)
      halo.addColorStop(0, rgba(light, (0.12 + live * 0.08) * energy))
      halo.addColorStop(0.42, rgba(cachedAccent, (0.08 + live * 0.04) * energy))
      halo.addColorStop(0.72, rgba(cachedAccent, 0.025 * energy))
      halo.addColorStop(1, rgba(cachedAccent, 0))
      ctx.fillStyle = halo
      ctx.fillRect(0, 0, width, height)

      const bloom = ctx.createRadialGradient(
        cx - orbRadius * 0.14,
        cy - orbRadius * 0.18,
        orbRadius * 0.03,
        cx,
        cy,
        orbRadius * 1.08,
      )
      bloom.addColorStop(0, rgba(light, 0.35 * energy))
      bloom.addColorStop(0.2, rgba(cachedAccent, 0.2 * energy))
      bloom.addColorStop(0.62, rgba(cachedAccent, 0.065 * energy))
      bloom.addColorStop(1, rgba(cachedAccent, 0))
      ctx.beginPath()
      ctx.arc(cx, cy, orbRadius * 1.2, 0, Math.PI * 2)
      ctx.fillStyle = bloom
      ctx.fill()

      const tilt = rotation * 0.62
      const sinTilt = Math.sin(tilt)
      const cosTilt = Math.cos(tilt)
      const cosYaw = Math.cos(rotation)
      const sinYaw = Math.sin(rotation)
      const particleScale = Math.max(0.82, Math.min(1.55, orbRadius / 132))

      for (const particle of particles) {
        const y1 = particle.y * cosTilt - particle.z * sinTilt
        const z1 = particle.y * sinTilt + particle.z * cosTilt
        const x2 = particle.x * cosYaw + z1 * sinYaw
        const z2 = -particle.x * sinYaw + z1 * cosYaw
        const perspective = 0.9 + z2 * 0.1
        const depth = Math.max(0.08, (z2 + 1) / 2)
        const speakingWave = current.phase === 'speaking'
          ? 1 + Math.sin(motion * 0.0045 + (particle.y + 1) * 4.2) * 0.045 * depth
          : 1
        const listeningWave = current.phase === 'listening'
          ? 1 + Math.sin(motion * 0.0058 + particle.seed * Math.PI * 2 + particle.y * 2.8)
            * live * (0.025 + depth * 0.035)
          : 1
        const radialWave = speakingWave * listeningWave
        const px = cx + x2 * orbRadius * perspective * radialWave
        const py = cy + y1 * orbRadius * perspective * radialWave
        const twinkle = 0.74 + Math.sin(motion * 0.0024 + particle.seed * Math.PI * 2) * 0.26
        const typeScale = particle.onShell ? 1 : 0.72
        const dot = particle.size * particleScale * typeScale * (0.66 + depth * 0.88) * (1 + live * 0.3)
        const alpha = Math.min(0.94, (0.12 + depth * 0.78) * twinkle * Math.max(0.74, energy))

        ctx.fillStyle = rgba(light, alpha * 0.14)
        ctx.beginPath()
        ctx.arc(px, py, dot * 3.1, 0, Math.PI * 2)
        ctx.fill()

        ctx.fillStyle = rgba(depth > 0.55 ? light : cachedAccent, alpha)
        ctx.beginPath()
        ctx.arc(px, py, dot, 0, Math.PI * 2)
        ctx.fill()

        if (particle.size > 1.5 && depth > 0.58) {
          ctx.fillStyle = rgba(cachedInk, alpha * 0.72)
          ctx.beginPath()
          ctx.arc(px, py, Math.max(0.65, dot * 0.42), 0, Math.PI * 2)
          ctx.fill()
        }
      }

      const nucleus = ctx.createRadialGradient(cx, cy, 0, cx, cy, orbRadius * 0.58)
      nucleus.addColorStop(0, rgba(light, 0.22 * energy))
      nucleus.addColorStop(0.34, rgba(cachedAccent, 0.12 * energy))
      nucleus.addColorStop(1, rgba(cachedAccent, 0))
      ctx.beginPath()
      ctx.arc(cx, cy, orbRadius * 0.58, 0, Math.PI * 2)
      ctx.fillStyle = nucleus
      ctx.fill()

      if (current.phase === 'thinking') {
        const arcStart = rotation * 2.1
        ctx.beginPath()
        ctx.arc(cx, cy, orbRadius * 1.18, arcStart, arcStart + Math.PI * 0.62)
        ctx.strokeStyle = rgba(cachedAccent, 0.66)
        ctx.lineWidth = 2
        ctx.lineCap = 'round'
        ctx.stroke()
      }

      if (current.phase === 'speaking') {
        for (let ring = 0; ring < 2; ring += 1) {
          const ringPulse = ((motion * 0.00035 + ring * 0.5) % 1)
          ctx.beginPath()
          ctx.arc(cx, cy, orbRadius * (1.05 + ringPulse * 0.34), 0, Math.PI * 2)
          ctx.strokeStyle = rgba(cachedAccent, (1 - ringPulse) * 0.2)
          ctx.lineWidth = 1
          ctx.stroke()
        }
      }

      ctx.restore()
    }

    const resize = () => {
      const rect = host.getBoundingClientRect()
      width = Math.max(1, rect.width)
      height = Math.max(1, rect.height)
      dpr = Math.min(2, window.devicePixelRatio || 1)
      canvas.width = Math.round(width * dpr)
      canvas.height = Math.round(height * dpr)
      canvas.style.width = `${width}px`
      canvas.style.height = `${height}px`
      redrawRef.current?.()
    }

    redrawRef.current = () => draw(performance.now(), false)

    const tick = (now: number) => {
      if (!running) return
      if (now - lastPaint >= 30) {
        lastPaint = now
        draw(now, true)
      }
      if (!shouldReduce()) raf = window.requestAnimationFrame(tick)
    }

    resize()
    draw(performance.now(), !shouldReduce())
    if (!shouldReduce()) raf = window.requestAnimationFrame(tick)

    const observer = typeof ResizeObserver !== 'undefined'
      ? new ResizeObserver(resize)
      : null
    observer?.observe(host)
    if (!observer) window.addEventListener('resize', resize)

    const onVisibility = () => {
      if (document.hidden) {
        window.cancelAnimationFrame(raf)
        return
      }
      start = performance.now()
      if (!shouldReduce()) raf = window.requestAnimationFrame(tick)
    }
    document.addEventListener('visibilitychange', onVisibility)

    return () => {
      running = false
      window.cancelAnimationFrame(raf)
      observer?.disconnect()
      if (!observer) window.removeEventListener('resize', resize)
      document.removeEventListener('visibilitychange', onVisibility)
      redrawRef.current = null
    }
  }, [])

  React.useEffect(() => {
    if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) {
      redrawRef.current?.()
    }
  }, [phase, active, connected, level])

  return (
    <div
      className={[
        'vc2-orb',
        `is-${phase}`,
        active ? 'is-active' : '',
        connected ? 'is-connected' : 'is-disconnected',
      ].filter(Boolean).join(' ')}
      style={{ '--vc2-live': Math.max(0, Math.min(1, level)).toFixed(3) } as React.CSSProperties}
      aria-hidden="true"
    >
      <canvas ref={canvasRef} className="vc2-orb-canvas" />
      <span className="vc2-orb-orbit is-outer" />
      <span className="vc2-orb-orbit is-inner" />
      <span className="vc2-orb-glow" />
      <span className="vc2-orb-core">
        <PhaseIcon spin={phase === 'thinking'} />
      </span>
      <span className="vc2-orb-wave">
        {[0.42, 0.7, 1, 0.58, 0.82, 0.5, 0.92].map((height, index) => (
          <i key={index} style={{ '--vc2-bar': height } as React.CSSProperties} />
        ))}
      </span>
    </div>
  )
}
