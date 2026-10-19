import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Button, Table, Tooltip } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { getRuntimeMonitor } from '../api/engines'
import { usePollingGate } from '../hooks/usePollingGate'
import './runtime.css'

// RuntimePage — 运行监测板块（2026-10-19，参照 Strata Monitor 仪表盘）。
// 数据源 App.RuntimeMonitor 聚合快照（2s 轮询，页面不可见停拉）：
//   · 指标卡：引擎 /metrics 遥测（CPU/内存/磁盘/GPU 系列——Windows AMD 下
//     GPU 系列为 null=诚实显示「无遥测」）+ 系统统计兜底 + 生成速度 tok/s；
//     迷你趋势为前端滚动窗口（60 点 ≈ 2 分钟）；
//   · 引擎状态条：模型中心引擎表 + Strata 系引擎 live 状态（空闲/读入中/
//     生成中、队列、prompt 进度）；
//   · 最近请求表：gaea 侧全引擎（含云端）调用记账（模型中心同源数据）。

/** /metrics 宽松契约（Strata 系服务端字段演进不编进前端硬类型，按需取用） */
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
  engine?: { model?: string }
  hardware?: Record<string, number | string | null>
  hardware_static?: Record<string, number | string | null>
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
  engines?: EngineRow[]
  strata?: EngineMetrics | null
  tianxuan?: EngineMetrics | null
  recent_calls?: RecentCall[]
  totals?: {
    total_calls?: number
    success_calls?: number
    fail_calls?: number
    total_tokens?: number
    total_cost?: number
    since?: string
  }
}

/** 数值宽容取用：null/undefined/非数 → null（「无遥测」口径） */
function num(v: unknown): number | null {
  return typeof v === 'number' && Number.isFinite(v) ? v : null
}

function fmtInt(v: number | null): string {
  return v == null ? '—' : Math.round(v).toLocaleString()
}

const LIVE_STATE_LABEL: Record<string, string> = {
  idle: '空闲',
  reading: '读入中',
  generating: '生成中',
  unloaded: '未加载',
}

/** 运行中的 Strata 系引擎遥测（先 tianxuan 后 strata；均停机为 null） */
function pickEngineMetrics(snap: MonitorSnapshot): EngineMetrics | null {
  const cand = [snap.tianxuan, snap.strata].find((m): m is EngineMetrics => !!m && !!m.live)
  return cand ?? null
}

/** 全局状态条：任一引擎生成中/读入中优先，否则有引擎在线=空闲 */
function overallState(snap: MonitorSnapshot): { label: string; tone: 'ok' | 'busy' | 'idle' | 'off'; tokS: number | null } {
  for (const m of [snap.tianxuan, snap.strata]) {
    if (m?.live?.state === 'generating' || m?.live?.state === 'reading') {
      return { label: LIVE_STATE_LABEL[m.live.state] ?? m.live.state, tone: 'busy', tokS: num(m.live.tok_s) }
    }
  }
  const online = (snap.engines ?? []).some((e) => e.enabled && e.connected)
  if (online) return { label: '空闲', tone: 'idle', tokS: null }
  return { label: '无引擎在线', tone: 'off', tokS: null }
}

/** 迷你趋势折线（手卷 SVG，100×26；数据不足两点画空轨） */
function Sparkline({ data, active }: { data: number[]; active: boolean }) {
  if (data.length < 2) {
    return (
      <svg className="rt-spark" viewBox="0 0 100 26" preserveAspectRatio="none" aria-hidden>
        <line className="rt-spark-baseline" x1="0" y1="24" x2="100" y2="24" />
      </svg>
    )
  }
  const min = Math.min(...data)
  const max = Math.max(...data)
  const span = max - min || 1
  const pts = data
    .map((v, i) => `${(i / (data.length - 1)) * 100},${24 - ((v - min) / span) * 20}`)
    .join(' ')
  return (
    <svg className={`rt-spark${active ? ' rt-spark-active' : ''}`} viewBox="0 0 100 26" preserveAspectRatio="none" aria-hidden>
      <polyline points={pts} fill="none" />
    </svg>
  )
}

const SERIES_WINDOW = 60 // 每序列滚动窗口（2s 一拍 ≈ 2 分钟）

function MetricCard({ label, value, unit, series }: { label: string; value: string; unit: string; series: number[] }) {
  const hasValue = value !== '—'
  return (
    <div className={`rt-card${hasValue ? '' : ' rt-card-empty'}`}>
      <div className="rt-card-label">{label}</div>
      <div className="rt-card-value">
        {value}
        {hasValue && unit && <span className="rt-card-unit"> {unit}</span>}
      </div>
      <Sparkline data={series} active={hasValue} />
    </div>
  )
}

export default function RuntimePage() {
  const [snap, setSnap] = useState<MonitorSnapshot | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState('')
  const pollable = usePollingGate()
  /** 趋势滚动窗口（键 → 最近 SERIES_WINDOW 个样本；ref 不触发重渲染，随快照一拍一画） */
  const seriesRef = useRef<Record<string, number[]>>({})

  const pushSeries = useCallback((key: string, v: number | null) => {
    if (v == null) return
    const arr = seriesRef.current[key] ?? []
    arr.push(v)
    if (arr.length > SERIES_WINDOW) arr.shift()
    seriesRef.current[key] = arr
  }, [])

  const load = useCallback(async () => {
    try {
      const m = (await getRuntimeMonitor()) as MonitorSnapshot
      setSnap(m)
      setError(null)
      setLastUpdated(new Date().toLocaleTimeString())
      // 指标卡序列：引擎遥测优先，系统统计兜底（无遥测序列不推进=空轨）
      const em = pickEngineMetrics(m)
      const hw = em?.hardware ?? {}
      const sys = m.system ?? {}
      pushSeries('tok_s', num(em?.live?.tok_s))
      pushSeries('gpu_util', num(hw.gpu_util))
      pushSeries('gpu_temp', num(hw.gpu_temp))
      pushSeries('gpu_power', num(hw.gpu_power))
      pushSeries('cpu', num(hw.cpu) ?? num(sys.cpu))
      pushSeries('ram_used', num(hw.ram_used) ?? num(sys.memUsed))
      pushSeries('vram_used', num(sys.vramUsed))
      pushSeries('disk_read', num(hw.disk_read_mb))
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '读取监测快照失败')
    }
  }, [pushSeries])

  useEffect(() => {
    const tick = () => { if (!pollable) return; void load() }
    tick()
    const t = window.setInterval(tick, 2000)
    return () => window.clearInterval(t)
  }, [load, pollable])

  const em = useMemo(() => (snap ? pickEngineMetrics(snap) : null), [snap])
  const overall = useMemo(() => (snap ? overallState(snap) : null), [snap])

  const metricCards = useMemo(() => {
    if (!snap) return []
    const hw = em?.hardware ?? {}
    const sys = snap.system ?? {}
    const ramUsed = num(hw.ram_used) ?? num(sys.memUsed)
    const ramTotal = num(hw.ram_total) ?? num(sys.memTotal)
    const vramUsed = num(sys.vramUsed)
    const vramTotal = num(sys.vramTotal)
    const tokS = num(em?.live?.tok_s)
    return [
      { label: '生成速度', value: tokS == null ? null : tokS.toFixed(1), unit: 'tok/s', series: 'tok_s' },
      { label: 'GPU 负载', value: num(hw.gpu_util)?.toFixed(0), unit: '%', series: 'gpu_util' },
      { label: 'GPU 温度', value: num(hw.gpu_temp)?.toFixed(0), unit: '°C', series: 'gpu_temp' },
      { label: 'GPU 功率', value: num(hw.gpu_power)?.toFixed(0), unit: 'W', series: 'gpu_power' },
      { label: 'CPU', value: (num(hw.cpu) ?? num(sys.cpu))?.toFixed(0), unit: '%', series: 'cpu' },
      { label: '内存', value: ramUsed == null ? null : `${ramUsed.toFixed(1)}/${ramTotal ?? '?'}`, unit: 'GB', series: 'ram_used' },
      { label: '显存', value: vramUsed == null || vramUsed <= 0 ? null : `${vramUsed.toFixed(1)}/${vramTotal ?? '?'}`, unit: 'GB', series: 'vram_used' },
      { label: '磁盘读', value: num(hw.disk_read_mb)?.toFixed(1), unit: 'MB/s', series: 'disk_read' },
    ]
  }, [snap, em])

  const enginesById = useMemo(() => {
    const map = new Map<string, EngineRow>()
    for (const e of snap?.engines ?? []) map.set(e.id, e)
    return map
  }, [snap])

  const localEngineCards = useMemo(
    () => ['herdsman', 'tianxuan', 'strata'].map((id) => ({ id, row: enginesById.get(id) })),
    [enginesById],
  )

  const cloudEngines = useMemo(
    () => (snap?.engines ?? []).filter((e) => e.enabled && !e.local),
    [snap],
  )

  const requestsColumns = useMemo(
    () => [
      { title: '时间', dataIndex: 'finished_at', key: 'time', width: 90, ellipsis: true },
      { title: '引擎', dataIndex: 'engine_id', key: 'engine', width: 90, ellipsis: true },
      { title: '模型', dataIndex: 'model', key: 'model', ellipsis: true },
      { title: '域', dataIndex: 'feature', key: 'feature', width: 70, render: (v: string) => v || '—' },
      {
        title: '入/出 tok',
        key: 'tokens',
        width: 110,
        render: (_: unknown, r: RecentCall) => `${fmtInt(r.input_tokens)} / ${fmtInt(r.output_tokens)}`,
      },
      {
        title: '缓存命中',
        dataIndex: 'cache_hit_tokens',
        key: 'cache',
        width: 80,
        render: (v: number) => (v ? fmtInt(v) : '—'),
      },
      {
        title: '时长',
        dataIndex: 'duration_ms',
        key: 'dur',
        width: 70,
        render: (v: number) => (v >= 1000 ? `${(v / 1000).toFixed(1)}s` : `${Math.round(v)}ms`),
      },
      {
        title: '出速',
        key: 'tps',
        width: 70,
        render: (_: unknown, r: RecentCall) =>
          r.duration_ms > 0 && r.output_tokens > 0 ? `${((r.output_tokens / r.duration_ms) * 1000).toFixed(1)}` : '—',
      },
      {
        title: '状态',
        dataIndex: 'success',
        key: 'ok',
        width: 70,
        render: (ok: boolean, r: RecentCall) =>
          ok ? <span className="rt-ok">成功</span> : (
            <Tooltip title={r.error_message || '失败'}><span className="rt-fail">失败</span></Tooltip>
          ),
      },
    ],
    [],
  )

  const totals = snap?.totals ?? {}
  const successRate = totals.total_calls
    ? `${((totals.success_calls ?? 0) / totals.total_calls) * 100 > 99.95 ? '100' : (((totals.success_calls ?? 0) / totals.total_calls) * 100).toFixed(1)}%`
    : '—'

  return (
    <div className="rt-page">
      <div className="rt-header">
        <div className={`rt-state rt-state-${overall?.tone ?? 'off'}`}>
          <span className="rt-state-dot" />
          {overall?.label ?? '加载中'}
          {overall?.tokS != null && <span className="rt-state-toks"> · {overall.tokS.toFixed(1)} tok/s</span>}
        </div>
        <span className="rt-header-meta">
          累计 {fmtInt(totals.total_calls ?? null)} 次调用 · 成功率 {successRate} · {fmtInt(totals.total_tokens ?? null)} tokens
          {totals.total_cost ? ` · ¥${totals.total_cost.toFixed(2)}` : ''}
        </span>
        <span style={{ flex: 1 }} />
        {lastUpdated && <span className="rt-updated">{lastUpdated}</span>}
        <Tooltip title="立即刷新">
          <Button size="small" type="text" icon={<ReloadOutlined />} aria-label="立即刷新" onClick={() => void load()} />
        </Tooltip>
      </div>
      {error && <div className="rt-error">刷新失败：{error}</div>}

      <div className="rt-cards">
        {metricCards.map((c) => (
          <MetricCard key={c.label} label={c.label} value={c.value ?? '—'} unit={c.unit} series={seriesRef.current[c.series] ?? []} />
        ))}
        {metricCards.length === 0 && <div className="rt-loading">正在读取监测数据…</div>}
      </div>

      <div className="rt-engines">
        {localEngineCards.map(({ id, row }) => {
          const metrics = id === 'tianxuan' ? snap?.tianxuan : id === 'strata' ? snap?.strata : null
          const live = metrics?.live
          const stateLabel = live ? (LIVE_STATE_LABEL[live.state ?? ''] ?? live.state) : null
          return (
            <div key={id} className={`rt-engine${row?.enabled ? '' : ' rt-engine-disabled'}`}>
              <div className="rt-engine-head">
                <span className={`rt-engine-dot${row?.connected ? ' on' : ''}`} />
                <span className="rt-engine-name">{row?.name ?? id}</span>
                {stateLabel && <span className={`rt-engine-state${live?.state === 'generating' ? ' busy' : ''}`}>{stateLabel}</span>}
                {row?.latency_ms != null && row.latency_ms > 0 && <span className="rt-engine-lat">{row.latency_ms}ms</span>}
              </div>
              <div className="rt-engine-model">
                {metrics?.engine?.model ?? row?.default_model ?? '未设置默认模型'}
                {live?.queued ? <span className="rt-engine-queued"> · 队列 {live.queued}</span> : null}
                {live?.prompt_total ? (
                  <span className="rt-engine-prompt"> · prompt {fmtInt(live.prompt_read ?? 0)}/{fmtInt(live.prompt_total)}</span>
                ) : null}
                {live?.elapsed_s != null ? <span className="rt-engine-elapsed"> · {live.elapsed_s}s</span> : null}
              </div>
              <div className="rt-engine-models">
                {(row?.models ?? [])
                  .filter((m) => m.kind === 'llm' || m.status === 'running')
                  .slice(0, 4)
                  .map((m) => (
                    <span key={m.id} className={`rt-chip${m.status === 'running' ? ' on' : ''}`} title={m.id}>{shortModel(m.id)}</span>
                  ))}
                {(row?.models ?? []).length > 4 && (
                  <span className="rt-chip rt-chip-more">+{(row?.models ?? []).length - 4}</span>
                )}
              </div>
            </div>
          )
        })}
        {cloudEngines.length > 0 && (
          <div className="rt-engine rt-engine-cloud">
            <div className="rt-engine-head">
              <span className="rt-engine-cloud-label">云端引擎</span>
            </div>
            <div className="rt-engine-models">
              {cloudEngines.map((e) => (
                <Tooltip key={e.id} title={`${e.name}${e.connected ? ' · 已连通' : ''}${e.default_model ? ` · ${e.default_model}` : ''}`}>
                  <span className={`rt-chip${e.connected ? ' on' : ''}`}>{e.name}</span>
                </Tooltip>
              ))}
            </div>
          </div>
        )}
      </div>

      <div className="rt-requests">
        <div className="rt-requests-title">最近请求（gaea 侧全引擎记账）</div>
        <Table<RecentCall>
          size="small"
          rowKey={(r, i) => `${r.finished_at}-${r.engine_id}-${i}`}
          columns={requestsColumns}
          dataSource={snap?.recent_calls ?? []}
          pagination={{ pageSize: 10, hideOnSinglePage: true, size: 'small' }}
          locale={{ emptyText: '暂无调用记录' }}
        />
      </div>
    </div>
  )
}

/** 模型名截短（chip 位宽有限；保留尾部档位段更有辨识度） */
function shortModel(id: string): string {
  if (id.length <= 28) return id
  const tail = id.slice(-24)
  return `…${tail}`
}
