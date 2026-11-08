import React from 'react'
import { AudioOutlined, LoadingOutlined, SoundOutlined } from '@ant-design/icons'
import type { VoiceSessionState } from '../session/voiceSession'

const LazyVoiceMemoryBrain = React.lazy(() => import('./VoiceMemoryBrainLayer'))

type RGB = readonly [number, number, number]

const GOLDEN_ANGLE = Math.PI * (3 - Math.sqrt(5))
const USER_GOLD: RGB = [242, 184, 91]
const VOICE_VIOLET: RGB = [184, 102, 255]
const SIGNAL_CYAN: RGB = [94, 224, 248]
const STAR_WHITE: RGB = [244, 249, 255]
const SHELL_PER_LOBE = 520
const CLOUD_PER_LOBE = 150

interface StarParticle {
  x: number
  y: number
  z: number
  seed: number
  size: number
  side: -1 | 1
  shell: boolean
  node: boolean
}

interface StarLink {
  from: number
  to: number
  bend: number
  bright: boolean
}

interface StarSweep {
  from: number
  control: number
  to: number
}

interface ProjectedStar {
  x: number
  y: number
  depth: number
}

interface ParticleField {
  particles: StarParticle[]
  linksBySide: [StarLink[], StarLink[]]
  sweepsBySide: [StarSweep[], StarSweep[]]
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

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value))
}

function buildParticleField(): ParticleField {
  const particles: StarParticle[] = []
  const linksBySide: [StarLink[], StarLink[]] = [[], []]
  const sweepsBySide: [StarSweep[], StarSweep[]] = [[], []]

  for (const side of [-1, 1] as const) {
    const sideIndex = side === -1 ? 0 : 1
    const start = particles.length
    const phaseOffset = side === -1 ? 0 : 0.63

    for (let local = 0; local < SHELL_PER_LOBE; local += 1) {
      const index = start + local
      const u = (local + 0.5) / SHELL_PER_LOBE
      const y = (1 - u * 2) * 0.9
      const ring = Math.sqrt(Math.max(0, 1 - y * y))
      const theta = GOLDEN_ANGLE * local + phaseOffset
      const irregularity = 0.96 + seededNoise(index * 17 + 3) * 0.08
      const organicWarp = 1
        + Math.sin(y * 8.4 + phaseOffset) * 0.025
        + Math.sin(y * 3.1 - phaseOffset) * 0.018
      const x = side * (0.044 + Math.abs(Math.cos(theta)) * ring * irregularity * organicWarp)
      const z = Math.sin(theta) * ring * irregularity * 0.88 * organicWarp
      const seed = seededNoise(index * 11 + 7)

      particles.push({
        x,
        y: y + (seededNoise(index * 23 + 9) - 0.5) * 0.025,
        z,
        seed,
        size: 0.42 + seededNoise(index * 13 + 31) * 1.18,
        side,
        shell: true,
        node: local % 43 === 7 || seededNoise(index * 29 + 5) > 0.978,
      })
    }

    for (let local = 0; local < CLOUD_PER_LOBE; local += 1) {
      const index = start + SHELL_PER_LOBE + local
      const y = (seededNoise(index * 5 + 1) * 2 - 1) * 0.84
      const ring = Math.sqrt(Math.max(0, 1 - y * y))
      const theta = seededNoise(index * 7 + 11) * Math.PI * 2
      const volume = Math.cbrt(seededNoise(index * 11 + 23)) * 0.82
      const x = side * (0.035 + Math.abs(Math.cos(theta)) * ring * volume)
      const z = Math.sin(theta) * ring * volume * 0.9

      particles.push({
        x,
        y,
        z,
        seed: seededNoise(index * 19 + 13),
        size: 0.34 + seededNoise(index * 31 + 17) * 0.76,
        side,
        shell: false,
        node: false,
      })
    }

    for (let local = 0; local < SHELL_PER_LOBE; local += 1) {
      const from = start + local
      const addLink = (offset: number, bright: boolean) => {
        const to = start + ((local + offset) % SHELL_PER_LOBE)
        if (from === to) return
        linksBySide[sideIndex].push({
          from,
          to,
          bend: seededNoise(from * 37 + offset) * 2 - 1,
          bright,
        })
      }

      if (local % 2 === 0) addLink(3, local % 26 === 0)
      if (local % 5 === 0) addLink(17, local % 35 === 0)
      if (local % 11 === 0) addLink(47, true)
      if (local % 41 === 0) addLink(131, false)
    }

    for (let local = 0; local < SHELL_PER_LOBE; local += 19) {
      sweepsBySide[sideIndex].push({
        from: start + local,
        control: start + ((local + 31) % SHELL_PER_LOBE),
        to: start + ((local + 97) % SHELL_PER_LOBE),
      })
    }
  }

  return { particles, linksBySide, sweepsBySide }
}

export interface VoiceV2OrbProps {
  phase: VoiceSessionState['phase']
  active: boolean
  connected: boolean
  /** Live microphone / playback RMS in the 0..1 range (React-throttled). */
  level?: number
  /** 50Hz 采集包络 ref：画布逐帧读，避免 React 每帧重渲染。 */
  micLevelRef?: { current: number }
  /** 20Hz 声卡实际外放电平 ref：说话时和耳朵听到的声音同步。 */
  playLevelRef?: { current: number }
}

/** Shared visual core for the product console and the V2 protocol lab. */
export const VoiceV2Orb: React.FC<VoiceV2OrbProps> = ({
  phase, active, connected, level = 0, micLevelRef, playLevelRef,
}) => {
  const canvasRef = React.useRef<HTMLCanvasElement>(null)
  const propsRef = React.useRef({ phase, active, connected, level, micLevelRef, playLevelRef })
  const redrawRef = React.useRef<(() => void) | null>(null)
  const fallbackPausedRef = React.useRef(false)
  const [memoryBrainReady, setMemoryBrainReady] = React.useState(false)
  propsRef.current = { phase, active, connected, level, micLevelRef, playLevelRef }

  const handleMemoryBrainReady = React.useCallback(() => {
    fallbackPausedRef.current = true
    setMemoryBrainReady(true)
  }, [])

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

    const { particles, linksBySide, sweepsBySide } = buildParticleField()
    const projected: ProjectedStar[] = particles.map(() => ({ x: 0, y: 0, depth: 0 }))
    const dust = Array.from({ length: 82 }, (_, index) => ({
      x: seededNoise(index * 17 + 1) * 2 - 1,
      y: seededNoise(index * 29 + 11) * 2 - 1,
      alpha: 0.08 + seededNoise(index * 41 + 5) * 0.28,
      size: 0.35 + seededNoise(index * 59 + 23) * 0.8,
    }))
    const backgroundLayer = document.createElement('canvas')
    const userMistLayer = document.createElement('canvas')
    const voiceMistLayer = document.createElement('canvas')
    const dustLayer = document.createElement('canvas')

    let width = 1
    let height = 1
    let dpr = 1
    let raf = 0
    let rafPending = false
    let running = true
    let inViewport = true
    let start = performance.now()
    let lastPaint = 0
    let pausedAt = 0
    let smoothLevel = 0
    let peakLevel = 0

    const prepareLayer = (layer: HTMLCanvasElement) => {
      layer.width = Math.round(width * dpr)
      layer.height = Math.round(height * dpr)
      const layerContext = layer.getContext('2d')
      layerContext?.setTransform(dpr, 0, 0, dpr, 0, 0)
      layerContext?.clearRect(0, 0, width, height)
      return layerContext
    }

    const paintStaticLayers = () => {
      const background = prepareLayer(backgroundLayer)
      const userMist = prepareLayer(userMistLayer)
      const voiceMist = prepareLayer(voiceMistLayer)
      const dustContext = prepareLayer(dustLayer)
      if (!background || !userMist || !voiceMist || !dustContext) return

      const cx = width / 2
      const cy = height * 0.465
      const orbRadius = Math.min(width, height) * 0.43

      const backdrop = background.createRadialGradient(
        cx,
        cy,
        orbRadius * 0.06,
        cx,
        cy,
        orbRadius * 1.58,
      )
      backdrop.addColorStop(0, 'rgba(8, 21, 34, 0.32)')
      backdrop.addColorStop(0.54, 'rgba(4, 12, 22, 0.15)')
      backdrop.addColorStop(1, 'rgba(2, 7, 13, 0)')
      background.fillStyle = backdrop
      background.fillRect(0, 0, width, height)

      const userGlow = userMist.createRadialGradient(
        cx - orbRadius * 0.43,
        cy - orbRadius * 0.03,
        0,
        cx - orbRadius * 0.43,
        cy - orbRadius * 0.03,
        orbRadius * 0.92,
      )
      userGlow.addColorStop(0, rgba(USER_GOLD, 0.14))
      userGlow.addColorStop(0.58, rgba(USER_GOLD, 0.04))
      userGlow.addColorStop(1, rgba(USER_GOLD, 0))
      userMist.fillStyle = userGlow
      userMist.fillRect(0, 0, width, height)

      const voiceGlow = voiceMist.createRadialGradient(
        cx + orbRadius * 0.46,
        cy - orbRadius * 0.02,
        0,
        cx + orbRadius * 0.46,
        cy - orbRadius * 0.02,
        orbRadius * 0.94,
      )
      voiceGlow.addColorStop(0, rgba(VOICE_VIOLET, 0.15))
      voiceGlow.addColorStop(0.58, rgba(VOICE_VIOLET, 0.042))
      voiceGlow.addColorStop(1, rgba(VOICE_VIOLET, 0))
      voiceMist.fillStyle = voiceGlow
      voiceMist.fillRect(0, 0, width, height)

      dustContext.globalCompositeOperation = 'lighter'
      for (const star of dust) {
        const px = cx + star.x * orbRadius * 1.34
        const py = cy + star.y * orbRadius * 1.28
        dustContext.fillStyle = rgba(STAR_WHITE, star.alpha)
        dustContext.beginPath()
        dustContext.arc(px, py, star.size, 0, Math.PI * 2)
        dustContext.fill()
      }
    }

    const draw = (now: number, animate: boolean) => {
      if (document.hidden || width <= 1 || height <= 1) return
      const current = propsRef.current
      const rawMic = clamp(current.micLevelRef?.current ?? current.level, 0, 1)
      const rawPlay = clamp(current.playLevelRef?.current ?? current.level, 0, 1)
      const targetLevel = current.phase === 'speaking'
        ? Math.max(rawPlay, rawMic)
        : current.active ? rawMic : Math.max(rawMic, rawPlay)
      smoothLevel += (targetLevel - smoothLevel) * (targetLevel > smoothLevel ? 0.4 : 0.12)
      peakLevel = Math.max(targetLevel, peakLevel * 0.93)
      const live = clamp(smoothLevel, 0, 1)
      const peak = clamp(peakLevel, 0, 1)

      ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
      ctx.clearRect(0, 0, width, height)

      const elapsed = Math.max(0, now - start)
      const motion = animate ? elapsed : 2400
      const rotation = motion * (
        (current.phase === 'thinking' ? 0.00015 : current.phase === 'speaking' ? 0.00012 : 0.00008)
        + live * 0.000035
      )
      const ambientPulse = Math.sin(motion * 0.00125) * 0.008
      const pulse = ambientPulse + live * 0.075 + peak * 0.025
      const orbRadius = Math.min(width, height) * 0.43 * (1 + pulse)
      const cx = width / 2
      const cy = height * 0.465
      const connectedEnergy = current.connected ? 1 : 0.78
      const userEnergy = connectedEnergy * (
        current.phase === 'listening' ? 1.16 : current.phase === 'speaking' ? 0.82 : 0.92
      )
      const voiceEnergy = connectedEnergy * (
        current.phase === 'speaking' ? 1.18 : current.phase === 'thinking' ? 1.08 : 0.82
      )
      const signalEnergy = (current.phase === 'thinking' ? 1.08 : 0.9) * connectedEnergy
      const signalColor = current.phase === 'error'
        ? ([255, 96, 107] as RGB)
        : current.phase === 'interrupted'
          ? ([255, 178, 76] as RGB)
          : SIGNAL_CYAN

      ctx.drawImage(backgroundLayer, 0, 0, width, height)

      ctx.save()
      ctx.globalCompositeOperation = 'lighter'
      ctx.globalAlpha = clamp(userEnergy, 0, 1)
      ctx.drawImage(userMistLayer, 0, 0, width, height)
      ctx.globalAlpha = clamp(voiceEnergy, 0, 1)
      ctx.drawImage(voiceMistLayer, 0, 0, width, height)
      ctx.globalAlpha = clamp(connectedEnergy, 0, 1)
      ctx.drawImage(dustLayer, 0, 0, width, height)
      ctx.globalAlpha = 1

      const orbitAlpha = clamp(0.08 + live * 0.13, 0.08, 0.24) * connectedEnergy
      ctx.lineWidth = 1
      ctx.strokeStyle = rgba(signalColor, orbitAlpha * 0.82)
      ctx.beginPath()
      ctx.ellipse(cx, cy + orbRadius * 0.1, orbRadius * 1.19, orbRadius * 0.2, -0.035, 0, Math.PI * 2)
      ctx.stroke()
      ctx.strokeStyle = rgba(USER_GOLD, orbitAlpha * 0.52)
      ctx.beginPath()
      ctx.ellipse(cx, cy - orbRadius * 0.08, orbRadius * 1.05, orbRadius * 0.31, 0.08, 0, Math.PI * 2)
      ctx.stroke()
      ctx.strokeStyle = rgba(VOICE_VIOLET, orbitAlpha * 0.5)
      ctx.beginPath()
      ctx.ellipse(cx, cy + orbRadius * 0.2, orbRadius * 1.02, orbRadius * 0.15, -0.12, 0, Math.PI * 2)
      ctx.stroke()

      const tilt = Math.sin(motion * 0.00018) * 0.055 + 0.045
      const yaw = rotation * 0.72
      const sinTilt = Math.sin(tilt)
      const cosTilt = Math.cos(tilt)
      const cosYaw = Math.cos(yaw)
      const sinYaw = Math.sin(yaw)

      for (let index = 0; index < particles.length; index += 1) {
        const particle = particles[index]
        const y1 = particle.y * cosTilt - particle.z * sinTilt
        const z1 = particle.y * sinTilt + particle.z * cosTilt
        const x2 = particle.x * cosYaw + z1 * sinYaw
        const z2 = -particle.x * sinYaw + z1 * cosYaw
        const depth = clamp((z2 + 1) / 2, 0, 1)
        const perspective = 0.92 + z2 * 0.08
        const speakingWave = current.phase === 'speaking'
          ? Math.sin(motion * 0.0042 + (particle.y + 1) * 4.8 + particle.seed * 2.4)
            * 0.035 * depth * (0.45 + live * 1.5)
          : 0
        const listeningWave = current.phase === 'listening'
          ? Math.sin(motion * 0.0053 + particle.seed * Math.PI * 2 + particle.y * 3.2)
            * live * (0.025 + depth * 0.055)
          : 0
        const tactileWave = Math.sin(motion * 0.0105 + particle.seed * Math.PI * 2 + particle.y * 3.6)
          * (0.012 + depth * 0.04) * (live * 1.35 + peak * 0.5)
        const radialWave = 1 + speakingWave + listeningWave + tactileWave
        const point = projected[index]
        point.x = cx + x2 * orbRadius * perspective * radialWave
        point.y = cy + y1 * orbRadius * perspective * radialWave
        point.depth = depth
      }

      const strokeLinks = (sideIndex: 0 | 1, color: RGB, energy: number, brightPass: boolean) => {
        const links = linksBySide[sideIndex]
        ctx.beginPath()
        for (let index = 0; index < links.length; index += 1) {
          const link = links[index]
          if (brightPass && !link.bright) continue
          const from = projected[link.from]
          const to = projected[link.to]
          const dx = to.x - from.x
          const dy = to.y - from.y
          const distance = Math.hypot(dx, dy)
          if (distance < 0.35) continue
          const bow = link.bend * Math.min(18, distance * 0.13)
          const mx = (from.x + to.x) / 2 - (dy / distance) * bow
          const my = (from.y + to.y) / 2 + (dx / distance) * bow
          ctx.moveTo(from.x, from.y)
          ctx.quadraticCurveTo(mx, my, to.x, to.y)
        }
        ctx.strokeStyle = rgba(
          color,
          (brightPass ? 0.23 : 0.08) * energy * (0.82 + live * 0.5),
        )
        ctx.lineWidth = brightPass ? 0.9 : 0.55
        ctx.stroke()
      }

      strokeLinks(0, USER_GOLD, userEnergy, false)
      strokeLinks(1, VOICE_VIOLET, voiceEnergy, false)

      const strokeSweeps = (sideIndex: 0 | 1, color: RGB, energy: number) => {
        ctx.beginPath()
        for (const sweep of sweepsBySide[sideIndex]) {
          const from = projected[sweep.from]
          const control = projected[sweep.control]
          const to = projected[sweep.to]
          ctx.moveTo(from.x, from.y)
          ctx.quadraticCurveTo(control.x, control.y, to.x, to.y)
        }
        ctx.strokeStyle = rgba(color, 0.105 * energy * (0.84 + live * 0.42))
        ctx.lineWidth = 0.7
        ctx.stroke()
      }

      strokeSweeps(0, mix(USER_GOLD, STAR_WHITE, 0.18), userEnergy)
      strokeSweeps(1, mix(VOICE_VIOLET, STAR_WHITE, 0.16), voiceEnergy)
      strokeLinks(0, mix(USER_GOLD, STAR_WHITE, 0.32), userEnergy, true)
      strokeLinks(1, mix(VOICE_VIOLET, STAR_WHITE, 0.3), voiceEnergy, true)

      const drawLobeRim = (side: -1 | 1, color: RGB, energy: number) => {
        const centerX = cx + side * orbRadius * 0.43
        const startAngle = side === -1 ? Math.PI * 0.54 : -Math.PI * 0.46
        const endAngle = side === -1 ? Math.PI * 1.46 : Math.PI * 0.46
        const radiusX = orbRadius * (0.55 + live * 0.01)
        const radiusY = orbRadius * 0.73

        ctx.beginPath()
        for (let index = 0; index <= 36; index += 1) {
          const angle = startAngle + (endAngle - startAngle) * (index / 36)
          const organic = 1 + Math.sin(angle * 6.2 + motion * 0.0007 + side) * 0.012
          const px = centerX + Math.cos(angle) * radiusX * organic
          const py = cy + Math.sin(angle) * radiusY * organic
          if (index === 0) ctx.moveTo(px, py)
          else ctx.lineTo(px, py)
        }
        ctx.strokeStyle = rgba(color, 0.08 * energy * (0.8 + live * 0.45))
        ctx.lineWidth = 4.2
        ctx.stroke()
        ctx.strokeStyle = rgba(mix(color, STAR_WHITE, 0.28), 0.25 * energy)
        ctx.lineWidth = 0.8
        ctx.stroke()
      }

      drawLobeRim(-1, USER_GOLD, userEnergy)
      drawLobeRim(1, VOICE_VIOLET, voiceEnergy)

      const beamTop = cy - orbRadius * 1.53
      const beamBottom = cy + orbRadius * 1.2
      const beam = ctx.createLinearGradient(cx, beamTop, cx, beamBottom)
      beam.addColorStop(0, rgba(signalColor, 0))
      beam.addColorStop(0.1, rgba(signalColor, 0.2 * signalEnergy))
      beam.addColorStop(0.47, rgba(STAR_WHITE, 0.5 * signalEnergy))
      beam.addColorStop(0.76, rgba(signalColor, 0.3 * signalEnergy))
      beam.addColorStop(1, rgba(signalColor, 0))
      ctx.strokeStyle = beam
      ctx.lineWidth = 5
      ctx.beginPath()
      ctx.moveTo(cx, beamTop)
      ctx.lineTo(cx, beamBottom)
      ctx.stroke()
      ctx.lineWidth = 0.9
      ctx.beginPath()
      ctx.moveTo(cx, beamTop)
      ctx.lineTo(cx, beamBottom)
      ctx.stroke()

      const particleScale = clamp(orbRadius / 250, 0.72, 1.65)
      for (let index = 0; index < particles.length; index += 1) {
        const particle = particles[index]
        const point = projected[index]
        const depth = point.depth
        const base = particle.side === -1 ? USER_GOLD : VOICE_VIOLET
        const color = mix(base, STAR_WHITE, depth * 0.12)
        const energy = particle.side === -1 ? userEnergy : voiceEnergy
        const twinkle = 0.7 + Math.sin(motion * 0.0021 + particle.seed * Math.PI * 2) * 0.3
        const dot = particle.size * particleScale * (particle.shell ? 1.02 : 0.68)
          * (0.54 + depth * 0.86) * (1 + live * 0.48 + peak * 0.16)
        const alpha = clamp(
          (0.14 + depth * 0.84) * twinkle * energy * (0.88 + live * 0.32),
          0.12,
          1,
        )

        if (particle.node || dot > 1.35) {
          ctx.fillStyle = rgba(color, alpha * 0.16)
          ctx.beginPath()
          ctx.arc(point.x, point.y, dot * 4.8, 0, Math.PI * 2)
          ctx.fill()
        }

        ctx.fillStyle = rgba(color, alpha * 0.9)
        ctx.beginPath()
        ctx.arc(point.x, point.y, dot, 0, Math.PI * 2)
        ctx.fill()

        if (particle.node && depth > 0.5) {
          const flare = dot * 5.2
          ctx.strokeStyle = rgba(STAR_WHITE, alpha * 0.36)
          ctx.lineWidth = 0.65
          ctx.beginPath()
          ctx.moveTo(point.x - flare, point.y)
          ctx.lineTo(point.x + flare, point.y)
          ctx.moveTo(point.x, point.y - flare)
          ctx.lineTo(point.x, point.y + flare)
          ctx.stroke()
          ctx.fillStyle = rgba(STAR_WHITE, alpha * 0.8)
          ctx.beginPath()
          ctx.arc(point.x, point.y, Math.max(0.55, dot * 0.34), 0, Math.PI * 2)
          ctx.fill()
        }
      }

      if (current.phase === 'listening' || current.phase === 'speaking') {
        const pulseColor = current.phase === 'listening' ? USER_GOLD : VOICE_VIOLET
        const pulseEnergy = current.phase === 'listening' ? userEnergy : voiceEnergy
        const centerX = cx + (current.phase === 'listening' ? -1 : 1) * orbRadius * 0.43
        for (let ring = 0; ring < 2; ring += 1) {
          const progress = (motion * 0.00026 + ring * 0.5) % 1
          const scale = 1 + progress * 0.24
          ctx.beginPath()
          ctx.ellipse(
            centerX,
            cy,
            orbRadius * 0.56 * scale,
            orbRadius * 0.74 * scale,
            0,
            0,
            Math.PI * 2,
          )
          ctx.strokeStyle = rgba(
            pulseColor,
            (1 - progress) * (0.16 + live * 0.18) * pulseEnergy,
          )
          ctx.lineWidth = 0.9
          ctx.stroke()
        }
      }

      if (current.phase === 'thinking') {
        const arcStart = motion * 0.00062
        ctx.beginPath()
        ctx.ellipse(
          cx,
          cy,
          orbRadius * 1.02,
          orbRadius * 0.94,
          0,
          arcStart,
          arcStart + Math.PI * 0.58,
        )
        ctx.strokeStyle = rgba(signalColor, 0.42 * signalEnergy)
        ctx.lineWidth = 1.2
        ctx.stroke()
      }

      if (live > 0.045 || peak > 0.08) {
        const impact = clamp(0.1 + live * 0.26, 0.1, 0.38)
        ctx.beginPath()
        ctx.ellipse(
          cx,
          cy,
          orbRadius * (1.06 + peak * 0.06),
          orbRadius * (0.92 + peak * 0.04),
          0,
          0,
          Math.PI * 2,
        )
        ctx.strokeStyle = rgba(signalColor, impact * connectedEnergy)
        ctx.lineWidth = 1
        ctx.stroke()
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
      paintStaticLayers()
      redrawRef.current?.()
    }

    redrawRef.current = () => draw(performance.now(), false)

    const canAnimate = () => running
      && inViewport
      && !document.hidden
      && !fallbackPausedRef.current
      && !shouldReduce()

    const pause = () => {
      if (!pausedAt) pausedAt = performance.now()
      window.cancelAnimationFrame(raf)
      rafPending = false
    }

    const schedule = () => {
      if (rafPending || !canAnimate()) return
      if (pausedAt) {
        start += performance.now() - pausedAt
        pausedAt = 0
      }
      rafPending = true
      raf = window.requestAnimationFrame(tick)
    }

    const tick = (now: number) => {
      rafPending = false
      if (!canAnimate()) {
        pause()
        return
      }
      const frameInterval = width < 380 ? 42 : 32
      if (now - lastPaint >= frameInterval) {
        lastPaint = now
        draw(now, true)
      }
      schedule()
    }

    resize()
    draw(performance.now(), !shouldReduce())
    schedule()

    const observer = typeof ResizeObserver !== 'undefined'
      ? new ResizeObserver(resize)
      : null
    observer?.observe(host)
    if (!observer) window.addEventListener('resize', resize)

    const intersectionObserver = typeof IntersectionObserver !== 'undefined'
      ? new IntersectionObserver((entries) => {
        inViewport = Boolean(entries[0]?.isIntersecting)
        if (inViewport) {
          draw(performance.now(), true)
          schedule()
        } else {
          pause()
        }
      }, { threshold: 0.02 })
      : null
    intersectionObserver?.observe(host)

    const onVisibility = () => {
      if (document.hidden) {
        pause()
        return
      }
      draw(performance.now(), true)
      schedule()
    }
    document.addEventListener('visibilitychange', onVisibility)

    return () => {
      running = false
      window.cancelAnimationFrame(raf)
      rafPending = false
      observer?.disconnect()
      intersectionObserver?.disconnect()
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
        memoryBrainReady ? 'has-memory-brain' : '',
      ].filter(Boolean).join(' ')}
      style={{ '--vc2-live': Math.max(0, Math.min(1, level)).toFixed(3) } as React.CSSProperties}
      aria-hidden="true"
    >
      <canvas ref={canvasRef} className="vc2-orb-canvas" data-testid="vc2-particle-field" />
      <div className="vc2-memory-brain-layer" aria-hidden="true">
        <React.Suspense fallback={null}>
          <LazyVoiceMemoryBrain
            phase={phase}
            connected={connected}
            micLevelRef={micLevelRef}
            playLevelRef={playLevelRef}
            onReady={handleMemoryBrainReady}
          />
        </React.Suspense>
      </div>
      <span className="vc2-orb-orbit is-outer" />
      <span className="vc2-orb-orbit is-inner" />
      <span className="vc2-orb-glow" />
      {!memoryBrainReady && (
        <span className="vc2-orb-core">
          <PhaseIcon spin={phase === 'thinking'} />
        </span>
      )}
      <span className="vc2-orb-wave">
        {[0.42, 0.7, 1, 0.58, 0.82, 0.5, 0.92].map((height, index) => (
          <i key={index} style={{ '--vc2-bar': height } as React.CSSProperties} />
        ))}
      </span>
    </div>
  )
}
