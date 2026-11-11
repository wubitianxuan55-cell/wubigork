import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Button, Input, Segmented, Table, Tooltip } from 'antd'
import {
  ApiOutlined,
  CheckCircleOutlined,
  CloudOutlined,
  DatabaseOutlined,
  MonitorOutlined,
  PayCircleOutlined,
  ReloadOutlined,
  SearchOutlined,
  ThunderboltOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import { getRuntimeMonitor } from '../api/engines'
import type { ModelUsageStats } from '../api/engines'
import { usePollingGate } from '../hooks/usePollingGate'
import { useStatsState } from './modelcenter/hooks/useStatsState'
import { type TrendDatum, type TrendRange } from './modelcenter/charts'
import {
  costToCNY,
  engineColor,
  engineLabel,
  fmtCompact,
  fmtCost,
  isLocalEngine,
  USD_TO_CNY,
} from './modelcenter/utils'
import './runtime.css'

// RuntimePage — 运行监测。
//
// Information architecture (2026-11-03 redesign):
//   1. compact live health summary
//   2. call analytics first: period KPIs, trends, outcomes, latency and errors
//   3. searchable / sortable per-model ledger
//   4. recent request ledger
//   5. engine and hardware detail below the operational answer
//
// Data contracts are unchanged:
//   - App.RuntimeMonitor: 2s snapshot for live engines and recent calls
//   - App.GetModelCallStats via useStatsState: 30s aggregate stats

interface EngineRequest {
  time?: number
  duration_s?: number
  finish?: string
  prompt_tokens?: number | null
  prompt_total?: number | null
  prompt_read?: number | null
  reused?: number | null
  output_tokens?: number | null
  prompt_ms?: number | null
  decode_tok_s?: number | null
  hit_rate?: number | null
}

interface EngineMetrics {
  live?: {
    state?: string
    queued?: number
    tok_s?: number | null
    tok_s_mean?: number | null
    prefill_tok_s_mean?: number | null
    prompt_read?: number | null
    prompt_total?: number | null
    generated?: number | null
    max_tokens?: number | null
    elapsed_s?: number | null
  }
  engine?: { model?: string; max_context?: number | null }
  hardware?: Record<string, number | string | null>
  hardware_static?: Record<string, number | string | null>
  requests?: EngineRequest[]
  conversation_cache?: {
    enabled?: boolean
    budget_mib?: number | null
    slots?: number | null
    requests?: number
    requests_reused?: number
    reused_tokens?: number
    prompt_tokens?: number
  }
  totals?: Record<string, number>
}

interface EngineRow {
  id: string
  name: string
  type: string
  local: boolean
  enabled: boolean
  connected: boolean
  latency_ms?: number
  error?: string
  default_model?: string
  models: { id: string; kind?: string; status?: string }[]
}

interface RecentCall {
  engine_id: string
  model: string
  feature?: string
  input_tokens: number
  output_tokens: number
  cache_hit_tokens?: number
  duration_ms: number
  success: boolean
  error_message?: string
  finished_at?: string
}

interface MonitorSnapshot {
  time?: string
  system?: Record<string, number | string>
  comfy_running?: boolean
  engines?: EngineRow[]
  strata?: EngineMetrics | null
  recent_calls?: RecentCall[]
  totals?: {
    total_calls?: number
    success_calls?: number
    fail_calls?: number
    total_tokens?: number
    input_tokens?: number
    output_tokens?: number
    total_cost?: number
    since?: string
  }
}

interface TrendSums {
  calls: number
  success: number
  fail: number
  input: number
  output: number
  cost: number
}

type ModelScope = 'all' | 'attention' | 'local' | 'cloud'
type RecentScope = 'all' | 'failed'

interface ModelRow extends ModelUsageStats {
  engineName: string
  local: boolean
  avgMs: number
  successRate: number
  costCny: number
  attention: boolean
}

const SERIES_WINDOW = 60

const LIVE_STATE_LABEL: Record<string, string> = {
  idle: '空闲',
  reading: '读入中',
  generating: '生成中',
  unloaded: '未加载',
}

const FINISH_LABEL: Record<string, string> = {
  stop: '正常',
  length: '截断',
  error: '错误',
}

function num(value: unknown): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null
}

function fmtInt(value: number | null | undefined): string {
  return value == null || !Number.isFinite(value) ? '暂无' : Math.round(value).toLocaleString()
}

function fmtToken(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0'
  if (value >= 1e9) return `${(value / 1e9).toFixed(1)}B`
  if (value >= 1e6) return `${(value / 1e6).toFixed(1)}M`
  if (value >= 1e3) return `${(value / 1e3).toFixed(1)}K`
  return Math.round(value).toLocaleString()
}

function fmtDuration(ms: number | null | undefined): string {
  if (ms == null || !Number.isFinite(ms) || ms < 0) return '暂无'
  if (ms < 1000) return `${Math.round(ms)}ms`
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`
  const minutes = Math.floor(ms / 60_000)
  const seconds = Math.round((ms % 60_000) / 1000)
  return `${minutes}m ${seconds}s`
}

function fmtClock(epoch?: number): string {
  if (!epoch) return '暂无'
  const d = new Date(epoch * 1000)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}

function pct(value: number): string {
  if (!Number.isFinite(value)) return '暂无'
  return `${value.toFixed(value >= 99.95 ? 0 : 1)}%`
}

function pickEngineMetrics(snap: MonitorSnapshot): EngineMetrics | null {
  return snap.strata?.live ? snap.strata : null
}

function overallState(snap: MonitorSnapshot): {
  label: string
  tone: 'ok' | 'busy' | 'off'
  tokS: number | null
} {
  for (const metrics of [snap.strata]) {
    if (metrics?.live?.state === 'generating' || metrics?.live?.state === 'reading') {
      return {
        label: LIVE_STATE_LABEL[metrics.live.state] ?? metrics.live.state,
        tone: 'busy',
        tokS: num(metrics.live.tok_s),
      }
    }
  }
  if ((snap.engines ?? []).some((engine) => engine.enabled && engine.connected)) {
    return { label: '空闲', tone: 'ok', tokS: null }
  }
  return { label: '无引擎在线', tone: 'off', tokS: null }
}

function shortModel(id: string): string {
  if (id.length <= 30) return id
  return `…${id.slice(-27)}`
}

/** A restrained request trend. Failure buckets use a ring plus textual titles,
 * so the chart remains understandable without relying on color alone. */
function RequestTrendChart({ data }: { data: TrendDatum[] }) {
  const W = 760
  const H = 228
  const padL = 48
  const padR = 18
  const padT = 18
  const padB = 34
  const plotW = W - padL - padR
  const plotH = H - padT - padB
  const maxValue = Math.max(...data.map((item) => item.calls), 1)
  const axisMax = maxValue <= 4 ? 4 : Math.ceil(maxValue / 4) * 4
  const x = (index: number) => (
    data.length === 1 ? padL + plotW / 2 : padL + (index / (data.length - 1)) * plotW
  )
  const y = (value: number) => padT + plotH * (1 - value / axisMax)
  const points = data.map((item, index) => `${x(index).toFixed(1)},${y(item.calls).toFixed(1)}`)
  const line = points.map((point, index) => `${index === 0 ? 'M' : 'L'}${point}`).join(' ')
  const area = `${line} L${x(data.length - 1).toFixed(1)},${padT + plotH} L${x(0).toFixed(1)},${padT + plotH} Z`
  const labelStep = Math.max(1, Math.ceil(data.length / 7))

  return (
    <div className="rt-chart-scroll">
      <svg className="rt-chart-svg" viewBox={`0 0 ${W} ${H}`} role="img" aria-label="调用趋势图">
        <title>所选时间范围内的调用趋势，红点表示该时段有失败调用</title>
        <defs>
          <linearGradient id="rt-request-area" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="var(--rt-accent)" stopOpacity="0.22" />
            <stop offset="100%" stopColor="var(--rt-accent)" stopOpacity="0.01" />
          </linearGradient>
        </defs>
        {[0, 0.25, 0.5, 0.75, 1].map((fraction) => {
          const tickValue = axisMax * fraction
          return (
            <g key={fraction}>
              <line
                className="rt-chart-grid"
                x1={padL}
                x2={W - padR}
                y1={y(tickValue)}
                y2={y(tickValue)}
              />
              <text className="rt-chart-axis" x={padL - 8} y={y(tickValue) + 4} textAnchor="end">
                {fmtCompact(tickValue)}
              </text>
            </g>
          )
        })}
        <path d={area} fill="url(#rt-request-area)" />
        <path d={line} className="rt-chart-line" />
        {data.map((item, index) => (
          <g key={item.key}>
            {item.failCalls > 0 && (
              <circle
                className="rt-chart-failure"
                cx={x(index)}
                cy={y(item.calls)}
                r={4}
              >
                <title>{`${item.label}，调用 ${item.calls} 次，失败 ${item.failCalls} 次`}</title>
              </circle>
            )}
            <circle cx={x(index)} cy={y(item.calls)} r={8} fill="transparent">
              <title>
                {`${item.label}，调用 ${item.calls} 次，成功 ${item.successCalls} 次，失败 ${item.failCalls} 次`}
              </title>
            </circle>
            {index === data.length - 1 && (
              <circle className="rt-chart-now" cx={x(index)} cy={y(item.calls)} r={3} />
            )}
            {(index % labelStep === 0 || index === data.length - 1) && (
              <text className="rt-chart-axis" x={x(index)} y={H - 10} textAnchor="middle">
                {item.label}
              </text>
            )}
          </g>
        ))}
      </svg>
    </div>
  )
}

function TokenTrendChart({ data }: { data: TrendDatum[] }) {
  const W = 760
  const H = 228
  const padL = 48
  const padR = 18
  const padT = 18
  const padB = 34
  const plotW = W - padL - padR
  const plotH = H - padT - padB
  const maxValue = Math.max(...data.map((item) => item.inputTokens + item.outputTokens), 1)
  const axisMax = maxValue <= 10 ? 10 : Math.ceil(maxValue / 4) * 4
  const slot = plotW / data.length
  const barWidth = Math.max(4, Math.min(34, slot * 0.58))
  const base = padT + plotH
  const labelStep = Math.max(1, Math.ceil(data.length / 7))

  return (
    <div className="rt-chart-scroll">
      <svg className="rt-chart-svg" viewBox={`0 0 ${W} ${H}`} role="img" aria-label="Token 用量趋势图">
        <title>输入与输出 Token 的分段用量趋势</title>
        {[0, 0.25, 0.5, 0.75, 1].map((fraction) => {
          const tickValue = axisMax * fraction
          const tickY = padT + plotH * (1 - fraction)
          return (
            <g key={fraction}>
              <line className="rt-chart-grid" x1={padL} x2={W - padR} y1={tickY} y2={tickY} />
              <text className="rt-chart-axis" x={padL - 8} y={tickY + 4} textAnchor="end">
                {fmtToken(tickValue)}
              </text>
            </g>
          )
        })}
        {data.map((item, index) => {
          const centerX = padL + (index + 0.5) * slot
          const inputHeight = (item.inputTokens / axisMax) * plotH
          const outputHeight = (item.outputTokens / axisMax) * plotH
          const inputY = base - inputHeight
          const outputY = inputY - outputHeight
          return (
            <g key={item.key}>
              <rect
                className="rt-token-input"
                x={centerX - barWidth / 2}
                y={inputY}
                width={barWidth}
                height={Math.max(inputHeight, 0.5)}
                rx={2}
              >
                <title>{`${item.label}，输入 ${item.inputTokens.toLocaleString()} Token`}</title>
              </rect>
              <rect
                className="rt-token-output"
                x={centerX - barWidth / 2}
                y={outputY}
                width={barWidth}
                height={Math.max(outputHeight, 0.5)}
                rx={2}
              >
                <title>{`${item.label}，输出 ${item.outputTokens.toLocaleString()} Token`}</title>
              </rect>
              {(index % labelStep === 0 || index === data.length - 1) && (
                <text className="rt-chart-axis" x={centerX} y={H - 10} textAnchor="middle">
                  {item.label}
                </text>
              )}
            </g>
          )
        })}
      </svg>
    </div>
  )
}

function TrendDataTable({ data }: { data: TrendDatum[] }) {
  return (
    <details className="rt-chart-data">
      <summary>查看图表明细数据</summary>
      <div className="rt-table-scroll">
        <table>
          <thead>
            <tr>
              <th>时间</th>
              <th>调用</th>
              <th>成功</th>
              <th>失败</th>
              <th>输入 Token</th>
              <th>输出 Token</th>
              <th>费用</th>
            </tr>
          </thead>
          <tbody>
            {data.map((item) => (
              <tr key={item.key}>
                <td>{item.label}</td>
                <td>{item.calls.toLocaleString()}</td>
                <td>{item.successCalls.toLocaleString()}</td>
                <td>{item.failCalls.toLocaleString()}</td>
                <td>{item.inputTokens.toLocaleString()}</td>
                <td>{item.outputTokens.toLocaleString()}</td>
                <td>{fmtCost(item.cost, 'CNY') || '¥0.00'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </details>
  )
}

function MetricTile({
  icon,
  label,
  value,
  detail,
  tone = 'neutral',
  testId,
  children,
}: {
  icon: ReactNode
  label: string
  value: string
  detail: ReactNode
  tone?: 'neutral' | 'positive' | 'danger'
  testId?: string
  children?: ReactNode
}) {
  return (
    <article className={`rt-kpi rt-kpi-${tone}`} data-testid={testId}>
      <div className="rt-kpi-label">
        <span className="rt-kpi-icon" aria-hidden="true">{icon}</span>
        <span>{label}</span>
      </div>
      <div className="rt-kpi-value">{value}</div>
      <div className="rt-kpi-detail">{detail}</div>
      {children}
    </article>
  )
}

function HeroWave({ data }: { data: number[] }) {
  const W = 360
  const H = 62
  const pad = 4
  if (data.length < 2) {
    return <div className="rt-wave-empty">等待连续吞吐采样</div>
  }
  const min = Math.min(...data)
  const max = Math.max(...data)
  const span = max - min || 1
  const points = data.map((value, index) => {
    const x = (index / (data.length - 1)) * W
    const y = pad + (H - pad * 2) * (1 - (value - min) / span)
    return `${x.toFixed(1)},${y.toFixed(1)}`
  })
  const line = points.map((point, index) => `${index === 0 ? 'M' : 'L'}${point}`).join(' ')
  const area = `${line} L${W},${H} L0,${H} Z`
  return (
    <svg className="rt-live-wave" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" aria-hidden="true">
      <defs>
        <linearGradient id="rt-live-wave-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--rt-accent)" stopOpacity="0.2" />
          <stop offset="100%" stopColor="var(--rt-accent)" stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d={area} fill="url(#rt-live-wave-fill)" />
      <path d={line} className="rt-live-wave-line" />
    </svg>
  )
}

function ResourceMeter({
  label,
  value,
  unit,
  fraction,
  title,
}: {
  label: string
  value: string
  unit: string
  fraction: number | null
  title?: string
}) {
  const width = fraction == null ? 0 : Math.max(0, Math.min(fraction * 100, 100))
  const pressure = width >= 90 ? 'danger' : width >= 75 ? 'warning' : 'normal'
  return (
    <div className="rt-meter" title={title}>
      <div className="rt-meter-head">
        <span>{label}</span>
        <strong>{value}{value !== '暂无' && unit ? ` ${unit}` : ''}</strong>
      </div>
      <div className="rt-meter-track" aria-hidden="true">
        <span className={`rt-meter-fill rt-meter-${pressure}`} style={{ width: `${width}%` }} />
      </div>
    </div>
  )
}

function ContextDonut({ used, max }: { used: number | null; max: number | null }) {
  const fraction = used != null && max != null && max > 0 ? Math.min(used / max, 1) : null
  const radius = 38
  const circumference = 2 * Math.PI * radius
  return (
    <svg className="rt-context-ring" viewBox="0 0 100 100" role="img" aria-label="上下文用量">
      <circle className="rt-context-track" cx="50" cy="50" r={radius} fill="none" strokeWidth="8" />
      {fraction != null && (
        <circle
          className={`rt-context-value${fraction >= 0.9 ? ' is-hot' : ''}`}
          cx="50"
          cy="50"
          r={radius}
          fill="none"
          strokeWidth="8"
          strokeDasharray={`${fraction * circumference} ${circumference}`}
          transform="rotate(-90 50 50)"
        />
      )}
      <text x="50" y="49" textAnchor="middle" className="rt-context-number">
        {fraction == null ? '暂无' : `${Math.round(fraction * 100)}%`}
      </text>
      <text x="50" y="63" textAnchor="middle" className="rt-context-caption">上下文</text>
    </svg>
  )
}

function CallQualityPanel({
  sums,
  recentCalls,
}: {
  sums: TrendSums
  recentCalls: RecentCall[]
}) {
  const successRateValue = sums.calls > 0 ? (sums.success / sums.calls) * 100 : 0
  const successWidth = sums.calls > 0 ? Math.max(0, Math.min(successRateValue, 100)) : 0
  const durations = recentCalls
    .map((call) => call.duration_ms)
    .filter((value) => Number.isFinite(value) && value >= 0)
    .sort((a, b) => a - b)
  const averageDuration = durations.length
    ? durations.reduce((sum, value) => sum + value, 0) / durations.length
    : null
  const p95Duration = durations.length
    ? durations[Math.min(durations.length - 1, Math.ceil(durations.length * 0.95) - 1)]
    : null
  const buckets = [
    { label: '1 秒内', count: durations.filter((value) => value < 1000).length },
    { label: '1 至 3 秒', count: durations.filter((value) => value >= 1000 && value < 3000).length },
    { label: '3 至 10 秒', count: durations.filter((value) => value >= 3000 && value < 10_000).length },
    { label: '10 秒以上', count: durations.filter((value) => value >= 10_000).length },
  ]
  const bucketMax = Math.max(...buckets.map((bucket) => bucket.count), 1)
  const errors = new Map<string, number>()
  for (const call of recentCalls) {
    if (call.success) continue
    const message = call.error_message?.trim() || '未提供错误信息'
    errors.set(message, (errors.get(message) ?? 0) + 1)
  }
  const topErrors = Array.from(errors.entries())
    .sort((a, b) => b[1] - a[1])
    .slice(0, 3)

  return (
    <div className="rt-quality">
      <section className="rt-quality-block">
        <div className="rt-subhead">
          <h3>调用结果</h3>
          <span>{sums.calls.toLocaleString()} 次样本</span>
        </div>
        <div
          className="rt-outcome-bar"
          role="img"
          aria-label={`成功 ${sums.success} 次，失败 ${sums.fail} 次`}
        >
          <span className="rt-outcome-success" style={{ width: `${successWidth}%` }} />
        </div>
        <div className="rt-outcome-legend">
          <span><i className="is-success" />成功 {sums.success.toLocaleString()}</span>
          <span><i className="is-failure" />失败 {sums.fail.toLocaleString()}</span>
          <strong>{sums.calls > 0 ? pct(successRateValue) : '暂无样本'}</strong>
        </div>
      </section>

      <section className="rt-quality-block">
        <div className="rt-subhead">
          <h3>近期请求耗时</h3>
          <span>{durations.length} 条样本</span>
        </div>
        <div className="rt-latency-summary">
          <span>平均 <strong>{fmtDuration(averageDuration)}</strong></span>
          <span>P95 <strong>{fmtDuration(p95Duration)}</strong></span>
        </div>
        <div className="rt-latency-buckets">
          {buckets.map((bucket) => (
            <div className="rt-latency-row" key={bucket.label}>
              <span>{bucket.label}</span>
              <span className="rt-latency-track" aria-hidden="true">
                <i style={{ width: `${Math.max((bucket.count / bucketMax) * 100, bucket.count ? 4 : 0)}%` }} />
              </span>
              <strong>{bucket.count}</strong>
            </div>
          ))}
        </div>
      </section>

      <section className="rt-quality-block">
        <div className="rt-subhead">
          <h3>近期错误</h3>
          <span>{errors.size ? `${errors.size} 类` : '无失败'}</span>
        </div>
        {topErrors.length ? (
          <ul className="rt-error-list">
            {topErrors.map(([message, count]) => (
              <li key={message}>
                <WarningOutlined aria-hidden="true" />
                <span title={message}>{message}</span>
                <strong>{count} 次</strong>
              </li>
            ))}
          </ul>
        ) : (
          <p className="rt-quality-empty"><CheckCircleOutlined aria-hidden="true" /> 最近记录中没有失败调用</p>
        )}
      </section>
    </div>
  )
}

function EngineCard({
  id,
  row,
  metrics,
}: {
  id: string
  row?: EngineRow
  metrics?: EngineMetrics | null
}) {
  const live = metrics?.live
  const stateLabel = live ? (LIVE_STATE_LABEL[live.state ?? ''] ?? live.state) : null
  const enabled = !!row?.enabled
  const connected = !!row?.connected
  return (
    <article className={`rt-engine-card${enabled ? '' : ' is-disabled'}${connected ? ' is-online' : ''}`}>
      <div className="rt-engine-head">
        <span className={`rt-status-dot ${connected ? 'is-ok' : 'is-off'}`} aria-hidden="true" />
        <strong>{row?.name ?? id}</strong>
        <span className={`rt-engine-state${live?.state === 'generating' ? ' is-busy' : ''}`}>
          {stateLabel ?? (row ? (enabled ? (connected ? '已连接' : '未连接') : '已停用') : '未配置')}
        </span>
        {row?.latency_ms != null && row.latency_ms > 0 && (
          <span className="rt-engine-latency">{row.latency_ms}ms</span>
        )}
      </div>
      <div className="rt-engine-model" title={metrics?.engine?.model ?? row?.default_model}>
        {metrics?.engine?.model ?? row?.default_model ?? '未设置默认模型'}
      </div>
      <div className="rt-engine-meta">
        {live?.queued ? <span>队列 {live.queued}</span> : <span>队列空闲</span>}
        {live?.prompt_total ? <span>Prompt {fmtInt(live.prompt_read)} / {fmtInt(live.prompt_total)}</span> : null}
        {live?.elapsed_s != null ? <span>本轮 {live.elapsed_s}s</span> : null}
      </div>
      <div className="rt-engine-models">
        {(row?.models ?? [])
          .filter((model) => model.kind === 'llm' || model.status === 'running')
          .slice(0, 4)
          .map((model) => (
            <span key={model.id} className={`rt-model-chip${model.status === 'running' ? ' is-running' : ''}`} title={model.id}>
              {shortModel(model.id)}
            </span>
          ))}
        {(row?.models ?? []).length > 4 && (
          <span className="rt-model-chip">+{(row?.models ?? []).length - 4}</span>
        )}
        {!row?.models?.length && <span className="rt-engine-empty">暂无模型信息</span>}
      </div>
      {row?.error && (
        <Tooltip title={row.error}>
          <span className="rt-engine-error">连接异常</span>
        </Tooltip>
      )}
    </article>
  )
}

export default function RuntimePage() {
  const [snap, setSnap] = useState<MonitorSnapshot | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState('')
  const [modelQuery, setModelQuery] = useState('')
  const [modelScope, setModelScope] = useState<ModelScope>('all')
  const [recentScope, setRecentScope] = useState<RecentScope>('all')
  const pollable = usePollingGate()
  const seriesRef = useRef<Record<string, number[]>>({})
  const stats = useStatsState(false)

  const pushSeries = useCallback((key: string, value: number | null) => {
    if (value == null) return
    const values = seriesRef.current[key] ?? []
    values.push(value)
    if (values.length > SERIES_WINDOW) values.shift()
    seriesRef.current[key] = values
  }, [])

  const load = useCallback(async () => {
    try {
      const snapshot = (await getRuntimeMonitor()) as MonitorSnapshot
      setSnap(snapshot)
      setError(null)
      setLastUpdated(new Date().toLocaleTimeString())
      const metrics = pickEngineMetrics(snapshot)
      pushSeries('tok_s', num(metrics?.live?.tok_s))
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '读取监测快照失败')
    }
  }, [pushSeries])

  useEffect(() => {
    const tick = () => {
      if (pollable) void load()
    }
    tick()
    const timer = window.setInterval(tick, 2000)
    return () => window.clearInterval(timer)
  }, [load, pollable])

  const em = useMemo(() => (snap ? pickEngineMetrics(snap) : null), [snap])
  const overall = useMemo(() => (snap ? overallState(snap) : null), [snap])
  const recentCalls = useMemo(() => snap?.recent_calls ?? [], [snap])
  const engines = useMemo(() => snap?.engines ?? [], [snap])
  const activeEngines = engines.filter((engine) => engine.enabled)
  const connectedEngines = activeEngines.filter((engine) => engine.connected)
  const engineRequests = useMemo(() => (em?.requests ?? []).slice(0, 12), [em])
  const ctxMax = num(em?.engine?.max_context)
  const latestEngineRequest = engineRequests[0]
  const ctxUsed = latestEngineRequest
    ? (num(latestEngineRequest.prompt_total) ?? num(latestEngineRequest.prompt_tokens) ?? 0)
      + (num(latestEngineRequest.output_tokens) ?? 0)
    : null
  const ctxPercent = ctxUsed != null && ctxMax ? Math.round(Math.min(ctxUsed / ctxMax, 1) * 100) : null

  const totals = snap?.totals ?? {}
  const lifetimeSuccessRate = totals.total_calls
    ? ((totals.success_calls ?? 0) / totals.total_calls) * 100
    : null

  const trendPeak = useMemo(
    () => stats.trendData.reduce((max, item) => Math.max(max, item.calls), 0),
    [stats.trendData],
  )
  const trendSums = useMemo<TrendSums>(
    () => stats.trendData.reduce(
      (sum, item) => ({
        calls: sum.calls + item.calls,
        success: sum.success + item.successCalls,
        fail: sum.fail + item.failCalls,
        input: sum.input + item.inputTokens,
        output: sum.output + item.outputTokens,
        cost: sum.cost + item.cost,
      }),
      { calls: 0, success: 0, fail: 0, input: 0, output: 0, cost: 0 },
    ),
    [stats.trendData],
  )
  const rangeSuccessRate = trendSums.calls > 0 ? (trendSums.success / trendSums.calls) * 100 : null
  const localEngineIds = useMemo(
    () => new Set(engines.filter((engine) => engine.local).map((engine) => engine.id)),
    [engines],
  )

  const modelRows = useMemo<ModelRow[]>(() => {
    const usdToCny = stats.callStats?.usd_to_cny ?? USD_TO_CNY
    return (stats.callStats?.per_model ?? []).map((row) => {
      const successRate = row.call_count > 0 ? (row.success_count / row.call_count) * 100 : 0
      const avgMs = row.call_count > 0 ? row.total_duration_ms / row.call_count : 0
      return {
        ...row,
        engineName: engineLabel({ engineId: row.engine_id }),
        local: localEngineIds.has(row.engine_id) || isLocalEngine(row.engine_id),
        avgMs,
        successRate,
        costCny: costToCNY(row.estimated_cost ?? 0, row.currency, usdToCny),
        attention: row.fail_count > 0 || (row.call_count >= 5 && successRate < 95),
      }
    })
  }, [localEngineIds, stats.callStats])

  const filteredModelRows = useMemo(() => {
    const query = modelQuery.trim().toLowerCase()
    return modelRows.filter((row) => {
      if (query && !row.model.toLowerCase().includes(query) && !row.engineName.toLowerCase().includes(query)) {
        return false
      }
      if (modelScope === 'attention') return row.attention
      if (modelScope === 'local') return row.local
      if (modelScope === 'cloud') return !row.local
      return true
    })
  }, [modelQuery, modelRows, modelScope])

  const modelMaxCalls = Math.max(...filteredModelRows.map((row) => row.call_count), 1)

  const resources = useMemo(() => {
    const hardware = em?.hardware ?? {}
    const system = snap?.system ?? {}
    const gpuName = typeof system.gpuName === 'string' ? system.gpuName : undefined
    const ramUsed = num(hardware.ram_used) ?? num(system.memUsed)
    const ramTotal = num(hardware.ram_total) ?? num(system.memTotal)
    const vramUsed = num(system.vramUsed)
    const vramTotal = num(system.vramTotal)
    const cpu = num(hardware.cpu) ?? num(system.cpu)
    const gpu = num(hardware.gpu_util) ?? num(system.gpuUsage)
    const diskRead = num(hardware.disk_read_mb)
    const diskWrite = num(hardware.disk_write_mb)
    const ratio = (used: number | null, total: number | null) => (
      used != null && total != null && total > 0 ? used / total : null
    )
    return {
      cpu,
      gpu,
      gpuName,
      ramUsed,
      ramTotal,
      vramUsed,
      vramTotal,
      ramFraction: ratio(ramUsed, ramTotal),
      gpuFraction: gpu == null ? null : gpu / 100,
      vramFraction: ratio(vramUsed, vramTotal),
      temperature: num(hardware.gpu_temp),
      power: num(hardware.gpu_power),
      diskRead,
      diskWrite,
    }
  }, [em, snap])

  const recentRows = useMemo(
    () => recentScope === 'failed' ? recentCalls.filter((call) => !call.success) : recentCalls,
    [recentCalls, recentScope],
  )

  const modelColumns = useMemo(() => [
    {
      title: '模型',
      key: 'model',
      width: 300,
      fixed: 'left' as const,
      render: (_: unknown, row: ModelRow) => (
        <div className="rt-model-cell">
          <strong title={row.model}>{row.model}</strong>
          <span>
            <i className="rt-engine-mark" style={{ background: engineColor({ engineId: row.engine_id }) }} />
            {row.engineName}
            {!row.engineName.endsWith(row.local ? '本地' : '云端') && (
              <em>{row.local ? '本地' : '云端'}</em>
            )}
          </span>
        </div>
      ),
    },
    {
      title: '调用量',
      dataIndex: 'call_count',
      key: 'calls',
      width: 150,
      sorter: (a: ModelRow, b: ModelRow) => a.call_count - b.call_count,
      render: (value: number, row: ModelRow) => (
        <div className="rt-call-volume">
          <strong>{value.toLocaleString()}</strong>
          <span><i style={{ width: `${Math.max((value / modelMaxCalls) * 100, 3)}%` }} /></span>
          {row.fail_count > 0 && <small>失败 {row.fail_count}</small>}
        </div>
      ),
    },
    {
      title: '成功率',
      dataIndex: 'successRate',
      key: 'success',
      width: 120,
      sorter: (a: ModelRow, b: ModelRow) => a.successRate - b.successRate,
      render: (value: number, row: ModelRow) => (
        <div className={`rt-table-rate${value < 95 ? ' is-warning' : ''}`}>
          <strong>{pct(value)}</strong>
          <span>{row.success_count.toLocaleString()} 成功</span>
        </div>
      ),
    },
    {
      title: 'Token 输入 / 输出',
      dataIndex: 'total_tokens',
      key: 'tokens',
      width: 170,
      sorter: (a: ModelRow, b: ModelRow) => a.total_tokens - b.total_tokens,
      render: (_: unknown, row: ModelRow) => (
        <div className="rt-token-cell">
          <strong>{fmtToken(row.input_tokens)} / {fmtToken(row.output_tokens)}</strong>
          <span>合计 {fmtToken(row.total_tokens)}</span>
        </div>
      ),
    },
    {
      title: '平均耗时',
      dataIndex: 'avgMs',
      key: 'latency',
      width: 110,
      sorter: (a: ModelRow, b: ModelRow) => a.avgMs - b.avgMs,
      render: (value: number) => <span className="rt-mono">{fmtDuration(value)}</span>,
    },
    {
      title: '估算费用',
      dataIndex: 'costCny',
      key: 'cost',
      width: 110,
      sorter: (a: ModelRow, b: ModelRow) => a.costCny - b.costCny,
      render: (_: unknown, row: ModelRow) => (
        <span className="rt-mono">
          {row.local ? '免费' : row.costCny > 0 ? fmtCost(row.costCny, 'CNY') : '未计价'}
        </span>
      ),
    },
    {
      title: '状态',
      key: 'status',
      width: 110,
      render: (_: unknown, row: ModelRow) => (
        <Tooltip title={row.last_error || (row.attention ? '存在失败调用，建议检查错误与路由' : '近期无失败记录')}>
          <span className={`rt-table-status${row.attention ? ' is-warning' : ' is-good'}`}>
            {row.attention ? <WarningOutlined aria-hidden="true" /> : <CheckCircleOutlined aria-hidden="true" />}
            {row.attention ? '需关注' : '正常'}
          </span>
        </Tooltip>
      ),
    },
  ], [modelMaxCalls])

  const recentColumns = useMemo(() => [
    {
      title: '时间',
      dataIndex: 'finished_at',
      key: 'time',
      width: 142,
      render: (value?: string) => <span className="rt-mono">{value || '暂无'}</span>,
    },
    {
      title: '模型',
      key: 'model',
      width: 260,
      render: (_: unknown, row: RecentCall) => (
        <div className="rt-model-cell">
          <strong title={row.model}>{row.model}</strong>
          <span>{engineLabel({ engineId: row.engine_id })}<em>{row.feature || '未标记功能'}</em></span>
        </div>
      ),
    },
    {
      title: 'Token 输入 / 输出',
      key: 'tokens',
      width: 160,
      render: (_: unknown, row: RecentCall) => (
        <span className="rt-mono">{fmtInt(row.input_tokens)} / {fmtInt(row.output_tokens)}</span>
      ),
    },
    {
      title: '缓存命中',
      dataIndex: 'cache_hit_tokens',
      key: 'cache',
      width: 110,
      render: (value?: number) => <span className="rt-mono">{value ? fmtInt(value) : '暂无'}</span>,
    },
    {
      title: '耗时',
      dataIndex: 'duration_ms',
      key: 'duration',
      width: 100,
      render: (value: number) => <span className="rt-mono">{fmtDuration(value)}</span>,
    },
    {
      title: '输出速度',
      key: 'throughput',
      width: 105,
      render: (_: unknown, row: RecentCall) => (
        <span className="rt-mono">
          {row.duration_ms > 0 && row.output_tokens > 0
            ? `${((row.output_tokens / row.duration_ms) * 1000).toFixed(1)} t/s`
            : '暂无'}
        </span>
      ),
    },
    {
      title: '结果',
      dataIndex: 'success',
      key: 'success',
      width: 160,
      render: (success: boolean, row: RecentCall) => (
        <div className={`rt-request-result${success ? ' is-good' : ' is-bad'}`}>
          <strong>
            {success ? <CheckCircleOutlined aria-hidden="true" /> : <WarningOutlined aria-hidden="true" />}
            {success ? '成功' : '失败'}
          </strong>
          {!success && <span title={row.error_message}>{row.error_message || '未提供错误信息'}</span>}
        </div>
      ),
    },
  ], [])

  const engineRequestColumns = useMemo(() => [
    {
      title: '时间',
      key: 'time',
      width: 88,
      render: (_: unknown, row: EngineRequest) => fmtClock(row.time),
    },
    {
      title: '结果',
      key: 'finish',
      width: 78,
      render: (_: unknown, row: EngineRequest) => (
        <span className={row.finish === 'error' ? 'rt-inline-bad' : 'rt-inline-good'}>
          {FINISH_LABEL[row.finish ?? ''] ?? row.finish ?? '暂无'}
        </span>
      ),
    },
    {
      title: 'Prompt',
      key: 'prompt',
      width: 132,
      render: (_: unknown, row: EngineRequest) => (
        <span>
          {fmtToken(num(row.prompt_tokens) ?? 0)}
          {row.reused ? <small className="rt-cell-note">复用 {fmtToken(num(row.reused) ?? 0)}</small> : null}
        </span>
      ),
    },
    {
      title: '输出',
      key: 'output',
      width: 84,
      render: (_: unknown, row: EngineRequest) => fmtToken(num(row.output_tokens) ?? 0),
    },
    {
      title: 'Prefill',
      key: 'prefill',
      width: 95,
      render: (_: unknown, row: EngineRequest) => {
        const ms = num(row.prompt_ms)
        const tokens = num(row.prompt_read) ?? num(row.prompt_tokens)
        return ms && ms > 0 && tokens ? `${((tokens / ms) * 1000).toFixed(0)} t/s` : '暂无'
      },
    },
    {
      title: 'Decode',
      key: 'decode',
      width: 95,
      render: (_: unknown, row: EngineRequest) => {
        const value = num(row.decode_tok_s)
        return value != null ? `${value.toFixed(1)} t/s` : '暂无'
      },
    },
    {
      title: '缓存命中',
      key: 'hit',
      width: 95,
      render: (_: unknown, row: EngineRequest) => {
        const value = num(row.hit_rate)
        return value != null ? `${Math.round(value * 100)}%` : '暂无'
      },
    },
    {
      title: '耗时',
      key: 'duration',
      width: 84,
      render: (_: unknown, row: EngineRequest) => fmtDuration(
        num(row.duration_s) == null ? null : (num(row.duration_s) as number) * 1000,
      ),
    },
  ], [])

  const localEngineCards = useMemo(
    () => ['herdsman', 'strata', 'breeze'].map((id) => ({
      id,
      row: engines.find((engine) => engine.id === id),
      metrics: id === 'strata' ? snap?.strata : null,
    })),
    [engines, snap],
  )
  const cloudEngines = engines.filter((engine) => engine.enabled && !engine.local)
  const cache = em?.conversation_cache

  return (
    <div className="rt-page">
      <header className="rt-page-header">
        <div className="rt-title-group">
          <span className="rt-title-icon" aria-hidden="true"><MonitorOutlined /></span>
          <div>
            <h1>运行监测</h1>
            <p>
              查看模型调用质量、推理状态与资源占用。
              {totals.total_calls != null && (
                <span className="rt-header-lifetime">
                  累计 {fmtInt(totals.total_calls)} 次调用
                  {lifetimeSuccessRate != null ? `，成功率 ${pct(lifetimeSuccessRate)}` : ''}
                </span>
              )}
            </p>
          </div>
        </div>
        <div className="rt-header-actions">
          <span className={`rt-live-pill${pollable ? '' : ' is-paused'}`}>
            <i aria-hidden="true" />
            {pollable ? '实时更新' : '已暂停'}
          </span>
          <span className="rt-updated">
            {lastUpdated ? `更新于 ${lastUpdated}` : '等待首轮同步'}
          </span>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              void load()
              void stats.loadCallStats()
            }}
          >
            刷新全部
          </Button>
        </div>
      </header>

      {error && (
        <div className="rt-alert" role="alert">
          <WarningOutlined aria-hidden="true" />
          <span>实时快照刷新失败：{error}</span>
          <Button size="small" onClick={() => void load()}>重试</Button>
        </div>
      )}

      <section className="rt-health-strip" aria-label="运行状态摘要">
        <div className="rt-health-item">
          <span className="rt-health-label">服务状态</span>
          <div className={`rt-health-value is-${overall?.tone ?? 'off'}`}>
            <i className="rt-status-dot" aria-hidden="true" />
            <strong>{overall?.label ?? '加载中'}</strong>
          </div>
          <span className="rt-health-note">
            {overall?.tokS != null ? `${overall.tokS.toFixed(1)} tok/s` : '当前没有生成任务'}
          </span>
        </div>
        <div className="rt-health-item">
          <span className="rt-health-label">已连接引擎</span>
          <div className="rt-health-value">
            <strong>{snap ? `${connectedEngines.length} / ${activeEngines.length}` : '加载中'}</strong>
          </div>
          <span className="rt-health-note">
            {snap
              ? `${connectedEngines.filter((engine) => engine.local).length} 个本地，${connectedEngines.filter((engine) => !engine.local).length} 个云端`
              : '正在检查连接'}
          </span>
        </div>
        <div className="rt-health-item">
          <span className="rt-health-label">当前生成速度</span>
          <div className="rt-health-value">
            <strong>{overall?.tokS != null ? `${overall.tokS.toFixed(1)} tok/s` : '暂无'}</strong>
          </div>
          <span className="rt-health-note">{em?.engine?.model ? shortModel(em.engine.model) : '等待本机推理遥测'}</span>
        </div>
        <div className="rt-health-item">
          <span className="rt-health-label">上下文压力</span>
          <div className="rt-health-value">
            <strong>{ctxPercent != null ? `${ctxPercent}%` : '暂无遥测'}</strong>
          </div>
          <span className="rt-health-note">
            {ctxUsed != null && ctxMax != null ? `${fmtToken(ctxUsed)} / ${fmtToken(ctxMax)} Token` : '最近一次请求尚未上报'}
          </span>
        </div>
      </section>

      <section className="rt-section rt-call-section" aria-labelledby="rt-call-title">
        <div className="rt-section-head">
          <div>
            <h2 id="rt-call-title">调用统计</h2>
            <p>先看所选时间范围内的调用量、成功率和成本，再下钻到模型与失败请求。</p>
          </div>
          <div className="rt-section-actions">
            <Segmented
              value={stats.trendRange}
              onChange={(value) => stats.setTrendRange(value as TrendRange)}
              options={[
                { value: 'today', label: '今日' },
                { value: '7d', label: '最近 7 天' },
                { value: '30d', label: '最近 30 天' },
              ]}
            />
            <Tooltip title="刷新调用统计">
              <Button
                type="text"
                icon={<ReloadOutlined />}
                aria-label="刷新调用统计"
                onClick={() => void stats.loadCallStats()}
              />
            </Tooltip>
          </div>
        </div>

        {stats.loadError ? (
          <div className="rt-alert" role="alert">
            <WarningOutlined aria-hidden="true" />
            <span>调用统计加载失败：{stats.loadError}</span>
            <Button size="small" onClick={() => void stats.loadCallStats()}>重试</Button>
          </div>
        ) : !stats.callStats ? (
          <div className="rt-stats-loading" aria-busy="true" aria-label="正在读取调用统计">
            {[0, 1, 2, 3].map((item) => <span key={item} />)}
          </div>
        ) : stats.trendData.length === 0 ? (
          <div className="rt-empty-state">
            <ThunderboltOutlined aria-hidden="true" />
            <strong>所选范围暂无调用记录</strong>
            <span>对话、语音和办公模块调用模型后，这里会自动生成趋势与质量分析。</span>
          </div>
        ) : (
          <div aria-live="polite">
            <div className="rt-kpi-grid">
              <MetricTile
                icon={<ApiOutlined />}
                label="调用次数"
                value={trendSums.calls.toLocaleString()}
                detail={`成功 ${trendSums.success.toLocaleString()}，失败 ${trendSums.fail.toLocaleString()}`}
                testId="metric-calls"
              />
              <MetricTile
                icon={<CheckCircleOutlined />}
                label="成功率"
                value={rangeSuccessRate == null ? '暂无' : pct(rangeSuccessRate)}
                detail={rangeSuccessRate != null && rangeSuccessRate < 95 ? '低于 95% 健康线' : '调用结果稳定'}
                tone={rangeSuccessRate != null && rangeSuccessRate < 95 ? 'danger' : 'positive'}
                testId="metric-success"
              />
              <MetricTile
                icon={<DatabaseOutlined />}
                label="Token 用量"
                value={fmtToken(trendSums.input + trendSums.output)}
                detail={`输入 ${fmtToken(trendSums.input)}，输出 ${fmtToken(trendSums.output)}`}
                testId="metric-tokens"
              >
                <div className="rt-token-split" aria-hidden="true">
                  <i style={{ width: `${trendSums.input + trendSums.output > 0 ? (trendSums.input / (trendSums.input + trendSums.output)) * 100 : 0}%` }} />
                </div>
              </MetricTile>
              <MetricTile
                icon={<PayCircleOutlined />}
                label="估算费用"
                value={fmtCost(trendSums.cost, 'CNY') || '¥0.00'}
                detail="按当前价格目录估算"
                testId="metric-cost"
              />
            </div>

            <div className="rt-analytics-grid">
              <article className="rt-chart-card">
                <div className="rt-chart-head">
                  <div>
                    <h3>调用趋势</h3>
                    <p>柱点按当前时间粒度聚合，红点标记有失败的时间段。</p>
                  </div>
                  <div className="rt-chart-facts">
                    <span>峰值 <strong>{trendPeak.toLocaleString()}</strong></span>
                    <span className="rt-chart-legend">
                      <i className="is-line" aria-hidden="true" />调用
                      <i className="is-fail" aria-hidden="true" />失败
                    </span>
                  </div>
                </div>
                <RequestTrendChart data={stats.trendData} />
              </article>
              <CallQualityPanel sums={trendSums} recentCalls={recentCalls} />
            </div>

            <article className="rt-chart-card rt-token-chart-card">
              <div className="rt-chart-head">
                <div>
                  <h3>Token 用量趋势</h3>
                  <p>仅比较输入与输出规模，费用请看上方所选范围汇总。</p>
                </div>
                <div className="rt-chart-facts">
                  <span>合计 <strong>{fmtToken(trendSums.input + trendSums.output)}</strong></span>
                  <span className="rt-chart-legend">
                    <i className="is-input" aria-hidden="true" />输入
                    <i className="is-output" aria-hidden="true" />输出
                  </span>
                </div>
              </div>
              <TokenTrendChart data={stats.trendData} />
            </article>
            <TrendDataTable data={stats.trendData} />
          </div>
        )}
      </section>

      <section className="rt-section" aria-labelledby="rt-model-title">
        <div className="rt-section-head">
          <div>
            <h2 id="rt-model-title">模型调用明细</h2>
            <p>累计口径。可搜索模型或引擎，按调用量、失败、Token、耗时和费用排序。</p>
          </div>
          <span className="rt-section-count">
            {filteredModelRows.length} / {modelRows.length} 个模型
          </span>
        </div>
        <div className="rt-table-toolbar">
          <Input
            allowClear
            value={modelQuery}
            onChange={(event) => setModelQuery(event.target.value)}
            prefix={<SearchOutlined aria-hidden="true" />}
            placeholder="搜索模型或引擎"
            aria-label="搜索模型或引擎"
          />
          <Segmented
            size="small"
            value={modelScope}
            onChange={(value) => setModelScope(value as ModelScope)}
            options={[
              { value: 'all', label: '全部' },
              { value: 'attention', label: '需关注' },
              { value: 'local', label: '本地' },
              { value: 'cloud', label: '云端' },
            ]}
          />
        </div>
        {!stats.callStats ? (
          <div className="rt-table-loading">正在读取模型调用明细…</div>
        ) : (
          <Table<ModelRow>
            className="rt-data-table"
            size="small"
            rowKey={(row) => `${row.engine_id}-${row.model}`}
            columns={modelColumns}
            dataSource={filteredModelRows}
            rowClassName={(row) => row.attention ? 'rt-row-attention' : ''}
            scroll={{ x: 1150 }}
            pagination={{
              pageSize: 8,
              hideOnSinglePage: true,
              showSizeChanger: false,
              showTotal: (total) => `共 ${total} 个模型`,
            }}
            locale={{
              emptyText: modelQuery || modelScope !== 'all'
                ? '没有符合条件的模型'
                : '暂无模型调用记录，使用模型后会自动累计',
            }}
          />
        )}
      </section>

      <section className="rt-section" aria-labelledby="rt-operations-title">
        <div className="rt-section-head">
          <div>
            <h2 id="rt-operations-title">运行状态</h2>
            <p>本机生成速度、上下文、会话缓存与硬件资源。</p>
          </div>
          <span className={`rt-health-badge is-${overall?.tone ?? 'off'}`}>
            <i aria-hidden="true" />
            {overall?.label ?? '加载中'}
          </span>
        </div>
        {!snap ? (
          <div className="rt-empty-state rt-empty-compact">
            <MonitorOutlined aria-hidden="true" />
            <strong>正在读取监测数据…</strong>
            <span>同步引擎状态与系统遥测。</span>
          </div>
        ) : (
          <>
            <div className="rt-ops-grid">
              <article className="rt-runtime-card">
                <div className="rt-runtime-head">
                  <div>
                    <span className="rt-eyebrow">当前推理</span>
                    <strong>{em?.engine?.model ?? '暂无活跃推理模型'}</strong>
                  </div>
                  <span className={`rt-status-label${overall?.tone === 'busy' ? ' is-busy' : ''}`}>
                    {overall?.label ?? '加载中'}
                  </span>
                </div>
                <div className="rt-runtime-body">
                  <div className="rt-throughput">
                    <span>生成速度</span>
                    <div>
                      <strong>{overall?.tokS != null ? overall.tokS.toFixed(1) : '暂无'}</strong>
                      {overall?.tokS != null && <em>tok/s</em>}
                    </div>
                    <HeroWave data={seriesRef.current.tok_s ?? []} />
                  </div>
                  <div className="rt-context-block">
                    <ContextDonut used={ctxUsed} max={ctxMax} />
                    <div>
                      <strong>{ctxUsed != null ? fmtToken(ctxUsed) : '暂无'}</strong>
                      <span>最近请求上下文</span>
                      <small>{ctxMax != null ? `上限 ${fmtToken(ctxMax)} Token` : '引擎未上报上限'}</small>
                    </div>
                  </div>
                </div>
                <div className="rt-cache-line">
                  <span>会话缓存</span>
                  {cache ? (
                    <>
                      <strong>{cache.enabled ? `${cache.requests_reused ?? 0} / ${cache.requests ?? 0}` : '未启用'}</strong>
                      <span>请求复用</span>
                      {!!cache.prompt_tokens && (
                        <span>
                          Prompt 复用 {Math.round(((cache.reused_tokens ?? 0) / cache.prompt_tokens) * 100)}%
                        </span>
                      )}
                      <span>{cache.slots ?? 0} 槽，预算 {cache.budget_mib ?? 0} MiB</span>
                    </>
                  ) : (
                    <span>当前引擎未上报缓存遥测</span>
                  )}
                </div>
              </article>

              <article className="rt-resource-card">
                <div className="rt-chart-head">
                  <div>
                    <h3>硬件资源</h3>
                    <p>{resources.gpuName || '未上报 GPU 名称'}</p>
                  </div>
                  <span className="rt-section-count">实时快照</span>
                </div>
                <div className="rt-resource-grid">
                  <ResourceMeter
                    label="CPU"
                    value={resources.cpu == null ? '暂无' : resources.cpu.toFixed(0)}
                    unit="%"
                    fraction={resources.cpu == null ? null : resources.cpu / 100}
                  />
                  <ResourceMeter
                    label="GPU"
                    value={resources.gpu == null ? '暂无' : resources.gpu.toFixed(0)}
                    unit="%"
                    fraction={resources.gpuFraction}
                    title={resources.gpuName}
                  />
                  <ResourceMeter
                    label="内存"
                    value={resources.ramUsed == null
                      ? '暂无'
                      : `${resources.ramUsed.toFixed(1)} / ${resources.ramTotal?.toFixed(1) ?? '?'}`}
                    unit="GB"
                    fraction={resources.ramFraction}
                  />
                  <ResourceMeter
                    label="显存"
                    value={resources.vramUsed == null
                      ? '暂无'
                      : `${resources.vramUsed.toFixed(1)} / ${resources.vramTotal?.toFixed(1) ?? '?'}`}
                    unit="GB"
                    fraction={resources.vramFraction}
                    title={resources.gpuName}
                  />
                </div>
                <div className="rt-secondary-readouts">
                  <span>GPU 温度 <strong>{resources.temperature == null ? '暂无' : `${resources.temperature.toFixed(0)} °C`}</strong></span>
                  <span>GPU 功率 <strong>{resources.power == null ? '暂无' : `${resources.power.toFixed(0)} W`}</strong></span>
                  <span>
                    磁盘读写{' '}
                    <strong>
                      {resources.diskRead == null && resources.diskWrite == null
                        ? '暂无'
                        : `${resources.diskRead?.toFixed(1) ?? '0.0'} / ${resources.diskWrite?.toFixed(1) ?? '0.0'} MB/s`}
                    </strong>
                  </span>
                </div>
              </article>
            </div>

            <div className="rt-engine-grid">
              {localEngineCards.filter(({ row }) => !!row).map(({ id, row, metrics }) => (
                <EngineCard key={id} id={id} row={row} metrics={metrics} />
              ))}
              {cloudEngines.length > 0 && (
                <article className="rt-engine-card rt-cloud-card">
                  <div className="rt-engine-head">
                    <CloudOutlined aria-hidden="true" />
                    <strong>云端引擎</strong>
                    <span className="rt-engine-state">{cloudEngines.length} 个</span>
                  </div>
                  <div className="rt-engine-models">
                    {cloudEngines.map((engine) => (
                      <Tooltip
                        key={engine.id}
                        title={`${engine.name}${engine.connected ? '，已连接' : '，未连接'}${engine.default_model ? `，${engine.default_model}` : ''}`}
                      >
                        <span className={`rt-model-chip${engine.connected ? ' is-running' : ''}`}>{engine.name}</span>
                      </Tooltip>
                    ))}
                  </div>
                </article>
              )}
              <article className={`rt-engine-card${snap.comfy_running ? ' is-online' : ' is-disabled'}`}>
                <div className="rt-engine-head">
                  <span className={`rt-status-dot ${snap.comfy_running ? 'is-ok' : 'is-off'}`} aria-hidden="true" />
                  <strong>ComfyUI</strong>
                  <span className="rt-engine-state">{snap.comfy_running ? '运行中' : '未运行'}</span>
                </div>
                <div className="rt-engine-model">图像工作流引擎</div>
                <div className="rt-engine-meta">
                  <span>{snap.comfy_running ? '画室生成队列可用' : '启动 ComfyUI 后可接收图像任务'}</span>
                </div>
              </article>
            </div>
          </>
        )}
      </section>

      <section className="rt-section" aria-labelledby="rt-recent-title">
        <div className="rt-section-head">
          <div>
            <h2 id="rt-recent-title">最近请求</h2>
            <p>来自 Gaea 全引擎调用记账的最近记录，可直接查看错误与缓存命中。</p>
          </div>
          <Segmented
            size="small"
            value={recentScope}
            onChange={(value) => setRecentScope(value as RecentScope)}
            options={[
              { value: 'all', label: `全部 ${recentCalls.length}` },
              { value: 'failed', label: `失败 ${recentCalls.filter((call) => !call.success).length}` },
            ]}
          />
        </div>
        <Table<RecentCall>
          className="rt-data-table"
          size="small"
          rowKey={(row) => `${row.finished_at ?? 'unknown'}-${row.engine_id}-${row.model}-${row.duration_ms}`}
          columns={recentColumns}
          dataSource={recentRows}
          scroll={{ x: 1080 }}
          pagination={{
            pageSize: 8,
            hideOnSinglePage: true,
            showSizeChanger: false,
            showTotal: (total) => `共 ${total} 条请求`,
          }}
          locale={{
            emptyText: recentScope === 'failed' ? '最近记录中没有失败请求' : '暂无调用记录',
          }}
        />
      </section>

      <section className="rt-section" aria-labelledby="rt-engine-request-title">
        <div className="rt-section-head">
          <div>
            <h2 id="rt-engine-request-title">引擎侧请求</h2>
            <p>
              {em
                ? 'Strata 引擎 /metrics 的最近完成请求，用于核对 Prompt、PreFill、Decode 与缓存。'
                : '当前没有可用的引擎侧请求遥测。'}
            </p>
          </div>
          <span className="rt-section-count">{engineRequests.length} 条</span>
        </div>
        <Table<EngineRequest>
          className="rt-data-table"
          size="small"
          rowKey={(row) => `${row.time ?? 'unknown'}-${row.finish ?? 'unknown'}-${row.prompt_tokens ?? 0}-${row.duration_s ?? 0}`}
          columns={engineRequestColumns}
          dataSource={engineRequests}
          scroll={{ x: 760 }}
          pagination={false}
          locale={{ emptyText: '引擎暂无已完成请求' }}
        />
      </section>
    </div>
  )
}
