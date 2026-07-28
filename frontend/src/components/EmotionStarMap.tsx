import React from 'react'
import { C } from '../utils/theme'

// ─── 四维情绪星图（对齐 ackem EmotionStarMap） ─────────────────

interface Props {
  aff: number; sec: number; aro: number; dom: number
  primaryLabel: string; size?: number
}

function emotionColor(label: string): string {
  const map: Record<string, string> = {
    SWEET_ATTACHMENT: '#f472b6', SHY_HEARTBEAT: '#fb7185',
    TSUNDERE: '#f59e0b', HURT_GRIEVANCE: '#a78bfa',
    ANGRY_ATTACK: '#ef4444', COLD_DETACHED: '#94a3b8',
    FEARFUL_OBEDIENT: '#c084fc', QUIET_FOND: '#fbbf24',
    CALM_RATIONAL: '#60a5fa',
  }
  return map[label] || '#60a5fa'
}

export const EmotionStarMap: React.FC<Props> = ({ aff, sec, aro, dom, primaryLabel, size = 160 }) => {
  const fillColor = emotionColor(primaryLabel)
  const cx = size / 2; const cy = size / 2; const r = size * 0.36

  const toAngle = (i: number) => (i * Math.PI * 2) / 4 - Math.PI / 2
  const toPoint = (i: number, val: number) => {
    const v = (val + 100) / 200
    const a = toAngle(i)
    return { x: cx + Math.cos(a) * r * v, y: cy + Math.sin(a) * r * v }
  }

  const dims = [
    { name: '亲密', val: aff, i: 0 },
    { name: '安全', val: sec, i: 1 },
    { name: '支配', val: dom, i: 2 },
    { name: '唤醒', val: aro, i: 3 },
  ]

  const gridRings = [0.25, 0.5, 0.75, 1]
  const dataPts = dims.map(d => toPoint(d.i, d.val))
  const dataPath = dataPts.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x.toFixed(1)} ${p.y.toFixed(1)}`).join(' ') + ' Z'

  // 残影帧
  const ghostFrames = [0.88, 0.76, 0.64].map(scale => {
    const pts = dims.map(d => { const v = ((d.val + 100) / 200) * scale; const a = toAngle(d.i); return { x: cx + Math.cos(a) * r * v, y: cy + Math.sin(a) * r * v } })
    return pts.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x.toFixed(1)} ${p.y.toFixed(1)}`).join(' ') + ' Z'
  })

  return (
    <svg viewBox={`0 0 ${size} ${size}`} style={{ width: '100%', height: 'auto' }} aria-hidden>
      {gridRings.map((s, ri) => {
        const pts = dims.map(d => toPoint(d.i, s * 200 - 100))
        return <path key={`r-${ri}`} d={pts.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ') + ' Z'} fill="none" stroke={fillColor} strokeWidth="0.5" opacity={0.06 + ri * 0.01} />
      })}
      {dims.map(d => { const p = toPoint(d.i, 100); return <line key={`ax-${d.i}`} x1={cx} y1={cy} x2={p.x} y2={p.y} stroke={fillColor} strokeWidth="0.5" opacity={0.08} /> })}
      {ghostFrames.map((path, i) => <path key={`gh-${i}`} d={path} fill={fillColor} fillOpacity={0.04 + i * 0.02} stroke="none" />)}
      <path d={dataPath} fill={fillColor} fillOpacity={0.35} stroke={fillColor} strokeWidth={1} strokeOpacity={0.7} style={{ filter: `drop-shadow(0 0 4px ${fillColor})` }} />
      {dataPts.map((p, i) => <circle key={i} cx={p.x} cy={p.y} r="2.5" fill={fillColor} fillOpacity={0.9} />)}
      {dims.map(d => { const p = toPoint(d.i, 118); return <text key={`l-${d.i}`} x={p.x} y={p.y} textAnchor="middle" dominantBaseline="middle" fill="var(--md-sys-color-text-secondary)" style={{ fontSize: '8px' }}>{d.name}</text> })}
    </svg>
  )
}
