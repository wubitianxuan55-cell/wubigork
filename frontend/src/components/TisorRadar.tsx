// TisorRadar.tsx — 五维环形雷达图 (T/I/S/O/R)
// Liquid Glass 风格 SVG 实现
import React, { useState } from 'react'

interface Props {
  dims: { T: number; I: number; S: number; O: number; R: number }
  size?: number       // SVG 画布尺寸，默认 200
  color?: string      // 数据区域主色，默认粉色
  showLabels?: boolean
}

const LABELS = ['Tend', 'Inde', 'Sens', 'Open', 'Rati'] as const
const FULL_LABELS: Record<string, string> = { T: 'Tender 温顺', I: 'Independent 独立', S: 'Sensitive 感性', O: 'Open 开放', R: 'Rational 理性' }
const KEYS = ['T', 'I', 'S', 'O', 'R'] as const

/** 将极坐标转为笛卡尔坐标（0° = 顶部） */
function polar(cx: number, cy: number, r: number, angleDeg: number): [number, number] {
  const rad = ((angleDeg - 90) * Math.PI) / 180
  return [cx + r * Math.cos(rad), cy + r * Math.sin(rad)]
}

export default function TisorRadar({ dims, size = 200, color = '#e85388', showLabels = true }: Props) {
  const [hovered, setHovered] = useState<string | null>(null)
  const cx = size / 2, cy = size / 2
  const radius = size * 0.34
  const levels = [0.2, 0.4, 0.6, 0.8, 1.0]

  // 五个轴的角度（从顶部顺时针：T/0°, I/72°, S/144°, O/216°, R/288°）
  const angles = KEYS.map((_, i) => i * 72)

  // 参考网格五边形
  const gridPolygons = levels.map(level => {
    const pts = angles.map(a => polar(cx, cy, radius * level, a))
    return pts.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' ')
  })

  // 数据多边形
  const dataPts = KEYS.map((k, i) => polar(cx, cy, radius * (dims[k] / 100), angles[i]))

  // 数据点位置
  const dotPositions = dataPts.map(([x, y], i) => ({ x, y, key: KEYS[i], value: dims[KEYS[i]] }))

  // 平滑曲线路径（二次贝塞尔逼近）
  const dataPath = dataPts.map(([x, y], i) => {
    const next = dataPts[(i + 1) % dataPts.length]
    const mx = (x + next[0]) / 2, my = (y + next[1]) / 2
    return i === 0 ? `M${x},${y}` : `Q${x},${y} ${mx},${my}`
  }).join(' ') + ' Z'

  // 标签位置（在最大半径外面一点）
  const labelPositions = angles.map(a => polar(cx, cy, radius * 1.18, a))

  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} style={{ overflow: 'visible' }}>
      <defs>
        {/* 数据区域渐变 */}
        <radialGradient id="tisor-fill-grad" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stopColor={color} stopOpacity={0.35} />
          <stop offset="70%" stopColor={color} stopOpacity={0.12} />
          <stop offset="100%" stopColor={color} stopOpacity={0.03} />
        </radialGradient>
        {/* 虹彩光泽 */}
        <linearGradient id="tisor-iridescent" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor={color} stopOpacity={0.15} />
          <stop offset="33%" stopColor="#a78bfa" stopOpacity={0.08} />
          <stop offset="66%" stopColor="#60a5fa" stopOpacity={0.06} />
          <stop offset="100%" stopColor={color} stopOpacity={0.12} />
        </linearGradient>
        {/* 光晕滤镜 */}
        <filter id="tisor-glow">
          <feGaussianBlur stdDeviation="1.5" result="blur" />
          <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
        </filter>
      </defs>

      {/* 参考网格 */}
      {gridPolygons.map((pts, i) => (
        <polygon key={`grid-${i}`} points={pts} fill="none"
          stroke="rgba(255,255,255,0.06)" strokeWidth={i === 4 ? 1 : 0.5}
          strokeDasharray={i === 4 ? 'none' : '3 3'} />
      ))}

      {/* 轴线 */}
      {angles.map((a, i) => {
        const [x, y] = polar(cx, cy, radius, a)
        return <line key={`axis-${i}`} x1={cx} y1={cy} x2={x} y2={y}
          stroke="rgba(255,255,255,0.04)" strokeWidth={0.5} />
      })}

      {/* 数据区域填充 — 底层虹彩 */}
      <path d={dataPath} fill="url(#tisor-iridescent)" stroke="none" />

      {/* 数据区域填充 — 上层主色 */}
      <path d={dataPath} fill="url(#tisor-fill-grad)" stroke={color} strokeWidth={1.5}
        strokeOpacity={0.5} filter="url(#tisor-glow)" />

      {/* 数据点 */}
      {dotPositions.map(({ x, y, key, value }) => (
        <g key={`dot-${key}`} onMouseEnter={() => setHovered(key)} onMouseLeave={() => setHovered(null)}
          style={{ cursor: 'pointer' }}>
          {/* 外圈光晕 */}
          <circle cx={x} cy={y} r={5} fill="none" stroke={color} strokeWidth={1} strokeOpacity={0.3} />
          {/* 内部圆点 */}
          <circle cx={x} cy={y} r={3} fill={color} opacity={hovered === key ? 1 : 0.85}
            style={{ transition: 'opacity 200ms ease' }} />
        </g>
      ))}

      {/* Tooltip */}
      {hovered && (() => {
        const d = dotPositions.find(p => p.key === hovered)!
        return (
          <g>
            <rect x={d.x - 28} y={d.y - 32} width={56} height={22} rx={6}
              fill="rgba(10,10,20,0.9)" stroke="rgba(255,255,255,0.1)" strokeWidth={0.5} />
            <text x={d.x} y={d.y - 15} textAnchor="middle" fill="#fff"
              fontSize={10} fontWeight={600}>{FULL_LABELS[hovered]}</text>
            <text x={d.x} y={d.y - 4} textAnchor="middle" fill={color}
              fontSize={10} fontWeight={700}>{d.value}</text>
          </g>
        )
      })()}

      {/* 轴标签 */}
      {showLabels && labelPositions.map(([x, y], i) => (
        <text key={`label-${i}`} x={x} y={y} textAnchor="middle" dominantBaseline="middle"
          fill="rgba(255,255,255,0.35)" fontSize={9} fontWeight={500}
          style={{ letterSpacing: '0.5px' }}>
          {LABELS[i]}
        </text>
      ))}
    </svg>
  )
}
