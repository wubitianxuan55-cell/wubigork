import type { FC } from 'react'
import { C } from '../../utils/theme'
import { fmtCompact } from './utils'

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
// 视口 540 宽（抽屉双列 ~380px 栏按 0.7 缩放渲染，轴字有效 ~7.7px；
// 旧 720 宽压到同栏轴字仅 ~5px 过小——检查器已改用下方迷你图组件）。
export const RequestsTrendChart: FC<{ data: TrendDatum[]; color: string }> = ({ data, color }) => {
  const W = 540, H = 200, padL = 44, padR = 16, padT = 16, padB = 28
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
          <text x={padL - 6} y={y(t) + 3} textAnchor="end" fontSize={11} style={{ fill: C('color-text-secondary') }}>{fmtCompact(t)}</text>
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
          <text key={i} x={x(i) + dx} y={H - 8} textAnchor={anchor} fontSize={11} style={{ fill: C('color-text-secondary') }}>{data[i].label}</text>
        )
      })}
    </svg>
  )
}

// TokenTrendChart Token 堆叠柱状图（入蓝/出绿）+ 费用红线（右侧轴）。视口 540（同上缩放口径）。
export const TokenTrendChart: FC<{ data: TrendDatum[] }> = ({ data }) => {
  const W = 540, H = 200, padL = 48, padR = 64, padT = 16, padB = 28
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
          <text x={padL - 6} y={yT(t) + 3} textAnchor="end" fontSize={11} style={{ fill: C('color-text-secondary') }}>{fmtCompact(t)}</text>
        </g>
      ))}
      {hasCost && (
        <>
          {[0.5, 1].map(r => (
            <text key={r} x={W - padR + 8} y={yC(maxCost * r) + 3} fontSize={11} style={{ fill: 'var(--color-destructive)' }}>{fmtAxis(maxCost * r)}</text>
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
          <text key={i} x={x(i) + dx} y={H - 8} textAnchor={anchor} fontSize={11} style={{ fill: C('color-text-secondary') }}>{data[i].label}</text>
        )
      })}
    </svg>
  )
}

// ── 检查器窄柱迷你图（右栏 ~276px 宽专用；viewBox 按真实宽度设计，
//    字标/描边按 1:1 渲染不缩放——全尺寸图压进窄柱会让 10px 字标缩成 ~4px 不可读）──

/** RequestsSpark 请求迷你趋势：面积 + 折线 + 失败红点 + 末点标记，无坐标轴（峰值在标题行） */
export const RequestsSpark: FC<{ data: TrendDatum[]; color: string }> = ({ data, color }) => {
  const W = 260, H = 56, padT = 5, padB = 5, padL = 2, padR = 4
  const plotW = W - padL - padR
  const plotH = H - padT - padB
  const maxV = Math.max(...data.map(d => d.calls), 1)
  const x = (i: number) => (data.length === 1 ? padL + plotW / 2 : padL + (i / (data.length - 1)) * plotW)
  const y = (v: number) => padT + plotH * (1 - v / maxV)
  const line = data.map((d, i) => `${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(d.calls).toFixed(1)}`).join(' ')
  const area = `${line} L${x(data.length - 1).toFixed(1)},${(padT + plotH).toFixed(1)} L${x(0).toFixed(1)},${(padT + plotH).toFixed(1)} Z`
  const last = data.length - 1
  return (
    <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: 'auto', display: 'block' }} role="img" aria-label="请求趋势迷你图">
      <path d={area} style={{ fill: `color-mix(in srgb, ${color} 18%, transparent)` }} />
      <path d={line} fill="none" style={{ stroke: color }} strokeWidth={1.6} strokeLinejoin="round" strokeLinecap="round" />
      {data.map((d, i) => (
        <g key={d.key}>
          {d.failCalls > 0 && (
            <circle cx={x(i)} cy={y(d.calls)} r={2.4} style={{ fill: 'var(--color-destructive)' }}>
              <title>{`${d.label} · ${d.calls} 次（失败 ${d.failCalls}）`}</title>
            </circle>
          )}
          <circle cx={x(i)} cy={y(d.calls)} r={0}>
            <title>{`${d.label} · ${d.calls} 次 · 成功 ${d.successCalls} · 失败 ${d.failCalls}`}</title>
          </circle>
        </g>
      ))}
      <circle cx={x(last)} cy={y(data[last].calls)} r={2.6} style={{ fill: color, stroke: 'var(--mc-surface)', strokeWidth: 1.5 }}>
        <title>{`${data[last].label} · ${data[last].calls} 次`}</title>
      </circle>
    </svg>
  )
}

/** TokenMiniBars Token 迷你堆叠条：入（主色，下）/ 出（成功色，上），无坐标轴（总计在标题行） */
export const TokenMiniBars: FC<{ data: TrendDatum[] }> = ({ data }) => {
  const W = 260, H = 48, padT = 3, padB = 3, padL = 2, padR = 2
  const plotW = W - padL - padR
  const plotH = H - padT - padB
  const maxTok = Math.max(...data.map(d => d.inputTokens + d.outputTokens), 1)
  const x = (i: number) => padL + (i + 0.5) * (plotW / data.length)
  const barW = Math.max(2, Math.min(16, (plotW / data.length) * 0.6))
  return (
    <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: 'auto', display: 'block' }} role="img" aria-label="Token 分布迷你图">
      {data.map((d, i) => {
        const x0 = x(i) - barW / 2
        const hIn = (d.inputTokens / maxTok) * plotH
        const hOut = (d.outputTokens / maxTok) * plotH
        const yIn = padT + plotH - hIn
        const yOut = yIn - hOut
        return (
          <g key={d.key}>
            <rect x={x0} y={yIn} width={barW} height={Math.max(0, hIn)} rx={1.5} style={{ fill: 'var(--color-primary)', opacity: 0.75 }}>
              <title>{`${d.label} · 输入 ${d.inputTokens.toLocaleString()} token`}</title>
            </rect>
            <rect x={x0} y={yOut} width={barW} height={Math.max(0, hOut)} rx={1.5} style={{ fill: 'var(--color-success)', opacity: 0.85 }}>
              <title>{`${d.label} · 输出 ${d.outputTokens.toLocaleString()} token`}</title>
            </rect>
          </g>
        )
      })}
    </svg>
  )
}
