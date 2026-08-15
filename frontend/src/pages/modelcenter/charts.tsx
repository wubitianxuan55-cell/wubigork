import type { FC } from 'react'
import { C } from '../../utils/theme'

export type StatsSort = 'calls' | 'tokens' | 'cost'
export type TrendRange = 'today' | '7d' | '30d'

export interface TrendDatum {
  key: string
  label: string
  calls: number
  successCalls: number
  failCalls: number
  inputTokens: number
  outputTokens: number
  cost: number
}

const fmtCompact = (v: number): string => {
  if (v >= 1e6) return `${(v / 1e6).toFixed(1)}M`
  if (v >= 1e3) return `${(v / 1e3).toFixed(1)}k`
  return `${Math.round(v)}`
}

// niceMax 把最大值规整为 1/2/2.5/5×10^n，让坐标轴刻度好看。
function niceMax(v: number): number {
  if (v <= 0) return 1
  const exp = Math.floor(Math.log10(v))
  const base = Math.pow(10, exp)
  const f = v / base
  const nf = f <= 1 ? 1 : f <= 2 ? 2 : f <= 2.5 ? 2.5 : f <= 5 ? 5 : 10
  return nf * base
}

const trendXTicks = (n: number): number[] => {
  if (n <= 1) return [0]
  const count = Math.min(6, n)
  return Array.from({ length: count }, (_, i) => Math.round((i * (n - 1)) / (count - 1)))
}

// RequestsTrendChart 请求趋势折线图（红点标注失败调用）。
export const RequestsTrendChart: FC<{ data: TrendDatum[]; color: string }> = ({ data, color }) => {
  const W = 720, H = 200, padL = 44, padR = 16, padT = 16, padB = 28
  const plotW = W - padL - padR
  const plotH = H - padT - padB
  const maxV = niceMax(Math.max(...data.map(d => d.calls), 1))
  const x = (i: number) => (data.length === 1 ? padL + plotW / 2 : padL + (i / (data.length - 1)) * plotW)
  const y = (v: number) => padT + plotH * (1 - v / maxV)
  const line = data.map((d, i) => `${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(d.calls).toFixed(1)}`).join(' ')
  const area = `${line} L${x(data.length - 1).toFixed(1)},${(padT + plotH).toFixed(1)} L${x(0).toFixed(1)},${(padT + plotH).toFixed(1)} Z`
  const ticks = Array.from({ length: 5 }, (_, i) => (maxV * i) / 4)
  const labelIdx = trendXTicks(data.length)
  return (
    <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: 'auto', display: 'block' }}>
      {ticks.map((t, i) => (
        <g key={i}>
          <line x1={padL} y1={y(t)} x2={W - padR} y2={y(t)} style={{ stroke: 'color-mix(in srgb, var(--color-text) 10%, transparent)' }} strokeWidth={1} />
          <text x={padL - 6} y={y(t) + 3} textAnchor="end" fontSize={10} style={{ fill: C('color-text-secondary') }}>{fmtCompact(t)}</text>
        </g>
      ))}
      <path d={area} style={{ fill: `color-mix(in srgb, ${color} 20%, transparent)` }} />
      <path d={line} fill="none" style={{ stroke: color }} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" />
      {data.map((d, i) => (
        <g key={d.key}>
          {d.failCalls > 0 && (
            <circle cx={x(i)} cy={y(d.calls)} r={3} style={{ fill: 'var(--color-destructive)', stroke: 'var(--color-bg-container)' }} strokeWidth={1}>
              <title>{`${d.label} · ${d.calls} 次（失败 ${d.failCalls}）`}</title>
            </circle>
          )}
          <circle cx={x(i)} cy={y(d.calls)} r={0}>
            <title>{`${d.label} · ${d.calls} 次 · 成功 ${d.successCalls} · 失败 ${d.failCalls}`}</title>
          </circle>
        </g>
      ))}
      {labelIdx.map(i => {
        const anchor = i === 0 ? 'start' : i === labelIdx[labelIdx.length - 1] ? 'end' : 'middle'
        const dx = i === 0 ? 2 : i === labelIdx[labelIdx.length - 1] ? -2 : 0
        return (
          <text key={i} x={x(i) + dx} y={H - 8} textAnchor={anchor} fontSize={10} style={{ fill: C('color-text-secondary') }}>{data[i].label}</text>
        )
      })}
    </svg>
  )
}

// TokenTrendChart Token 堆叠柱状图（入蓝/出绿）+ 费用红线（右侧轴）。
export const TokenTrendChart: FC<{ data: TrendDatum[] }> = ({ data }) => {
  const W = 720, H = 200, padL = 48, padR = 64, padT = 16, padB = 28
  const plotW = W - padL - padR
  const plotH = H - padT - padB
  const maxTok = niceMax(Math.max(...data.map(d => d.inputTokens + d.outputTokens), 1))
  const maxCost = niceMax(Math.max(...data.map(d => d.cost), 0))
  const hasCost = data.some(d => d.cost > 0)
  const x = (i: number) => padL + (i + 0.5) * (plotW / data.length)
  const yT = (v: number) => padT + plotH * (1 - v / maxTok)
  const yC = (v: number) => padT + plotH * (1 - v / maxCost)
  const barW = Math.max(2, Math.min(28, (plotW / data.length) * 0.55))
  const costLine = data.map((d, i) => `${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${yC(d.cost).toFixed(1)}`).join(' ')
  const fmtAxis = (v: number) => (maxCost >= 0.01 ? `¥${v.toFixed(2)}` : `¥${v.toFixed(3)}`)
  const ticks = Array.from({ length: 5 }, (_, i) => (maxTok * i) / 4)
  const labelIdx = trendXTicks(data.length)
  return (
    <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: 'auto', display: 'block' }}>
      {ticks.map((t, i) => (
        <g key={i}>
          <line x1={padL} y1={yT(t)} x2={W - padR} y2={yT(t)} style={{ stroke: 'color-mix(in srgb, var(--color-text) 10%, transparent)' }} strokeWidth={1} />
          <text x={padL - 6} y={yT(t) + 3} textAnchor="end" fontSize={10} style={{ fill: C('color-text-secondary') }}>{fmtCompact(t)}</text>
        </g>
      ))}
      {hasCost && (
        <>
          {[0.5, 1].map(r => (
            <text key={r} x={W - padR + 8} y={yC(maxCost * r) + 3} fontSize={10} style={{ fill: 'var(--color-destructive)' }}>{fmtAxis(maxCost * r)}</text>
          ))}
          <path d={costLine} fill="none" style={{ stroke: 'var(--color-destructive)' }} strokeWidth={1.6} strokeDasharray="4 3" />
        </>
      )}
      {data.map((d, i) => {
        const x0 = x(i) - barW / 2
        const hIn = (d.inputTokens / maxTok) * plotH
        const hOut = (d.outputTokens / maxTok) * plotH
        const yIn = padT + plotH - hIn
        const yOut = yIn - hOut
        return (
          <g key={d.key}>
            <rect x={x0} y={yIn} width={barW} height={Math.max(0, hIn)} style={{ fill: 'var(--color-primary)' }}>
              <title>{`${d.label} · 输入 ${d.inputTokens} Token`}</title>
            </rect>
            <rect x={x0} y={yOut} width={barW} height={Math.max(0, hOut)} style={{ fill: 'var(--color-success)' }}>
              <title>{`${d.label} · 输出 ${d.outputTokens} Token`}</title>
            </rect>
            {hasCost && d.cost > 0 && (
              <circle cx={x(i)} cy={yC(d.cost)} r={2} style={{ fill: 'var(--color-destructive)' }}>
                <title>{`${d.label} · 费用 ${fmtAxis(d.cost)}`}</title>
              </circle>
            )}
          </g>
        )
      })}
      {labelIdx.map(i => {
        const anchor = i === 0 ? 'start' : i === labelIdx[labelIdx.length - 1] ? 'end' : 'middle'
        const dx = i === 0 ? 2 : i === labelIdx[labelIdx.length - 1] ? -2 : 0
        return (
          <text key={i} x={x(i) + dx} y={H - 8} textAnchor={anchor} fontSize={10} style={{ fill: C('color-text-secondary') }}>{data[i].label}</text>
        )
      })}
    </svg>
  )
}
