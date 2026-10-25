import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Button, Segmented, Table, Tooltip } from 'antd'
import { MonitorOutlined, ReloadOutlined } from '@ant-design/icons'
import { getRuntimeMonitor } from '../api/engines'
import { emitFrontendEvent, FRONTEND_EVENTS } from '../events'
import { usePollingGate } from '../hooks/usePollingGate'
import { useStatsState } from './modelcenter/hooks/useStatsState'
import { RequestsSpark, TokenMiniBars, type TrendRange } from './modelcenter/charts'
import './runtime.css'

// RuntimePage — 运行监测板块（2026-10-19，参照 Strata Monitor 仪表盘）。
// 数据源 App.RuntimeMonitor 聚合快照（2s 轮询，页面不可见停拉）：
//   · 指标卡：引擎 /metrics 遥测（CPU/内存/磁盘/GPU 系列——Windows AMD 下
//     GPU 系列为 null=诚实显示「无遥测」）+ 系统统计兜底 + 生成速度 tok/s；
//     迷你趋势为前端滚动窗口（60 点 ≈ 2 分钟）；
//   · 引擎状态条：模型中心引擎表 + Strata 系引擎 live 状态（空闲/读入中/
//     生成中、队列、prompt 进度）+ ComfyUI 运行标记；
//   · 调用统计趋势：今日/7天/30天（useStatsState 同源 getModelCallStats；
//     2026-10 模型中心右侧统计检查器并入本板块——总计/成功率/费用在状态条，
//     趋势与入/出拆分在此）；
//   · 最近请求表：gaea 侧全引擎（含云端）调用记账（模型中心同源数据）。

/** /metrics 宽松契约（Strata 系服务端字段演进不编进前端硬类型，按需取用） */
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
  tianxuan?: EngineMetrics | null
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

/** 引擎侧请求的 finish 状态文案（Strata：stop/length/error） */
const FINISH_LABEL: Record<string, string> = {
  stop: '正常',
  length: '截断',
  error: '错误',
}

function fmtClock(epoch?: number): string {
  if (!epoch) return '—'
  const d = new Date(epoch * 1000)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`
}

function fmtKTok(v: number | null): string {
  if (v == null) return '—'
  return v >= 1000 ? `${(v / 1000).toFixed(1)}K` : String(Math.round(v))
}

/** Context 用量环（最近一次请求的 prompt_total+输出 vs 引擎 max_context） */
function ContextDonut({ used, max }: { used: number | null; max: number | null }) {
  const pct = used != null && max && max > 0 ? Math.min(used / max, 1) : null
  const R = 42
  const C = 2 * Math.PI * R
  return (
    <svg viewBox="0 0 100 100" className="rt-donut" role="img" aria-label="上下文用量">
      <circle className="rt-donut-track" cx="50" cy="50" r={R} fill="none" strokeWidth="10" />
      {pct != null && (
        <circle
          className={`rt-donut-fill${pct > 0.9 ? ' rt-donut-hot' : ''}`}
          cx="50" cy="50" r={R} fill="none" strokeWidth="10"
          strokeDasharray={`${pct * C} ${C}`} transform="rotate(-90 50 50)"
        />
      )}
      <text x="50" y="49" textAnchor="middle" className="rt-donut-num">
        {pct == null ? '—' : `${Math.round(pct * 100)}%`}
      </text>
      <text x="50" y="64" textAnchor="middle" className="rt-donut-label">Context</text>
    </svg>
  )
}

function MetricCard({ label, value, unit, series, hint }: { label: string; value: string; unit: string; series: number[]; hint?: string }) {
  const hasValue = value !== '—'
  return (
    <div className={`rt-card${hasValue ? '' : ' rt-card-empty'}`} title={hint}>
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
      // GPU 负载：引擎遥测优先，系统 PDH 采集兜底（AMD Windows 下引擎侧恒
      // null，系统侧常年有数——2026-10-19 轮二补齐）。
      pushSeries('gpu_util', num(hw.gpu_util) ?? num(sys.gpuUsage))
      pushSeries('gpu_temp', num(hw.gpu_temp))
      pushSeries('gpu_power', num(hw.gpu_power))
      pushSeries('cpu', num(hw.cpu) ?? num(sys.cpu))
      pushSeries('ram_used', num(hw.ram_used) ?? num(sys.memUsed))
      pushSeries('vram_used', num(sys.vramUsed))
      pushSeries('disk_read', num(hw.disk_read_mb))
      pushSeries('disk_write', num(hw.disk_write_mb))
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

  const metricCards = useMemo<{ label: string; value?: string | null; unit: string; series: string; hint?: string }[]>(() => {
    if (!snap) return []
    const hw = em?.hardware ?? {}
    const sys = snap.system ?? {}
    const gpuName = typeof sys.gpuName === 'string' ? sys.gpuName : ''
    const gpuHint = gpuName || undefined
    const ramUsed = num(hw.ram_used) ?? num(sys.memUsed)
    const ramTotal = num(hw.ram_total) ?? num(sys.memTotal)
    const vramUsed = num(sys.vramUsed)
    const vramTotal = num(sys.vramTotal)
    const tokS = num(em?.live?.tok_s)
    const diskRead = num(hw.disk_read_mb)
    const diskWrite = num(hw.disk_write_mb)
    const diskValue =
      diskRead == null && diskWrite == null
        ? null
        : `${diskRead?.toFixed(1) ?? '—'} / ${diskWrite?.toFixed(1) ?? '—'}`
    // 占用百分比（原模型中心资源条的 CPU/内存/显存 百分比读数并入：总量卡
    // 尾缀 pct，有 used+total 才给）
    const pct = (used: number | null, total: number | null) =>
      used != null && total != null && total > 0 ? ` · ${Math.round((used / total) * 100)}%` : ''
    return [
      { label: '生成速度', value: tokS == null ? null : tokS.toFixed(1), unit: 'tok/s', series: 'tok_s' },
      { label: 'GPU 负载', value: (num(hw.gpu_util) ?? num(sys.gpuUsage))?.toFixed(0), unit: '%', series: 'gpu_util', hint: gpuHint },
      { label: 'GPU 温度', value: num(hw.gpu_temp)?.toFixed(0), unit: '°C', series: 'gpu_temp', hint: gpuHint },
      { label: 'GPU 功率', value: num(hw.gpu_power)?.toFixed(0), unit: 'W', series: 'gpu_power', hint: gpuHint },
      { label: 'CPU', value: (num(hw.cpu) ?? num(sys.cpu))?.toFixed(0), unit: '%', series: 'cpu' },
      { label: '内存', value: ramUsed == null ? null : `${ramUsed.toFixed(1)}/${ramTotal ?? '?'}`, unit: `GB${pct(ramUsed, ramTotal)}`, series: 'ram_used' },
      { label: '显存', value: vramUsed == null || vramUsed <= 0 ? null : `${vramUsed.toFixed(1)}/${vramTotal ?? '?'}`, unit: `GB${pct(vramUsed, vramTotal)}`, series: 'vram_used', hint: gpuHint },
      { label: '磁盘 读/写', value: diskValue, unit: 'MB/s', series: 'disk_read' },
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

  // 调用统计趋势（原模型中心右侧统计检查器并入）：与「详细统计」抽屉同源
  // （getModelCallStats，30s 轮询走 usePollingGate）；总计/成功率/费用已由
  // 状态条 totals 行承载（2s 快照口径），此处只补趋势与入/出拆分。
  const stats = useStatsState(false)
  const trendPeak = useMemo(() => stats.trendData.reduce((a, b) => Math.max(a, b.calls), 0), [stats.trendData])

  // ── 引擎侧遥测（/metrics：Context 环 + 会话缓存 + 引擎请求表）──
  const engineRequests = useMemo(() => (em?.requests ?? []).slice(0, 12), [em])
  const ctxMax = num(em?.engine?.max_context)
  const lastReq = engineRequests[0]
  const ctxUsed = lastReq
    ? (num(lastReq.prompt_total) ?? num(lastReq.prompt_tokens) ?? 0) + (num(lastReq.output_tokens) ?? 0)
    : null
  const cache = em?.conversation_cache
  const engineReqColumns = useMemo(
    () => [
      { title: '时间', key: 'time', width: 80, render: (_: unknown, r: EngineRequest) => fmtClock(r.time) },
      {
        title: '状态', key: 'finish', width: 56,
        render: (_: unknown, r: EngineRequest) => {
          const ok = r.finish !== 'error'
          return ok
            ? <span className="rt-ok">{FINISH_LABEL[r.finish ?? ''] ?? r.finish ?? '—'}</span>
            : <span className="rt-fail">错误</span>
        },
      },
      {
        title: 'Prompt', key: 'prompt', width: 90,
        render: (_: unknown, r: EngineRequest) => (
          <span>
            {fmtKTok(num(r.prompt_tokens))}
            {r.reused ? <span className="rt-req-sub"> 复用 {fmtKTok(num(r.reused))}</span> : null}
          </span>
        ),
      },
      { title: '输出', key: 'out', width: 70, render: (_: unknown, r: EngineRequest) => fmtKTok(num(r.output_tokens)) },
      {
        title: 'prefill t/s', key: 'prefill', width: 90,
        render: (_: unknown, r: EngineRequest) => {
          const ms = num(r.prompt_ms)
          const toks = num(r.prompt_read) ?? num(r.prompt_tokens)
          return ms && ms > 0 && toks ? ((toks / ms) * 1000).toFixed(0) : '—'
        },
      },
      {
        title: 'decode t/s', key: 'decode', width: 90,
        render: (_: unknown, r: EngineRequest) => (num(r.decode_tok_s) != null ? num(r.decode_tok_s)!.toFixed(1) : '—'),
      },
      {
        title: 'VRAM 命中', key: 'hit', width: 80,
        render: (_: unknown, r: EngineRequest) => (num(r.hit_rate) != null ? `${Math.round(num(r.hit_rate)! * 100)}%` : '—'),
      },
      {
        title: '时长', key: 'dur', width: 64,
        render: (_: unknown, r: EngineRequest) => (num(r.duration_s) != null ? `${num(r.duration_s)!.toFixed(1)}s` : '—'),
      },
    ],
    [],
  )

  return (
    <div className="rt-page">
      {/* D1 统一板块头部（40px 紧凑档·运行监测样板间）：图标章+工作区名+
          实时状态+累计口径+动作区——板块名已在壳层面包屑，不再另起大标题 */}
      <header className="rt-ph">
        <span className="rt-ph-seat" aria-hidden="true"><MonitorOutlined /></span>
        <span className="rt-ph-name">运行监测</span>
        <span className={`rt-state rt-state-${overall?.tone ?? 'off'}`}>
          <span className="rt-state-dot" />
          {overall?.label ?? '加载中'}
          {overall?.tokS != null && <span className="rt-state-toks"> · {overall.tokS.toFixed(1)} tok/s</span>}
        </span>
        <span className="rt-header-meta">
          累计 {fmtInt(totals.total_calls ?? null)} 次调用 · 成功率 {successRate} · {fmtInt(totals.total_tokens ?? null)} tokens
          {totals.total_cost ? ` · ¥${totals.total_cost.toFixed(2)}` : ''}
        </span>
        <div className="rt-ph-actions">
          {lastUpdated && <span className="rt-updated">{lastUpdated}</span>}
          <Tooltip title="立即刷新">
            <Button size="small" type="text" icon={<ReloadOutlined />} aria-label="立即刷新" onClick={() => void load()} />
          </Tooltip>
        </div>
      </header>
      {error && <div className="rt-error">刷新失败：{error}</div>}

      {/* 左主列（实时面）：硬件遥测 → 引擎 → gaea 记账明细 */}
      <main className="rt-main" aria-label="实时监测">
        <div className="rt-eyebrow">硬件遥测</div>
        <div className="rt-cards">
          {metricCards.map((c) => (
            <MetricCard key={c.label} label={c.label} value={c.value ?? '—'} unit={c.unit} series={seriesRef.current[c.series] ?? []} hint={c.hint} />
          ))}
          {metricCards.length === 0 && <div className="rt-loading">正在读取监测数据…</div>}
        </div>

        <div className="rt-eyebrow">引擎</div>
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
          {/* ComfyUI：图像工作流引擎，非 engineMgr 引擎表成员（原模型中心资源
              检查器「ComfyUI 运行中」chip 并入，恒出卡、点灯随 comfy_running） */}
          <div className={`rt-engine${snap?.comfy_running ? '' : ' rt-engine-disabled'}`}>
            <div className="rt-engine-head">
              <span className={`rt-engine-dot${snap?.comfy_running ? ' on' : ''}`} />
              <span className="rt-engine-name">ComfyUI</span>
              <span className="rt-engine-state">图像工作流</span>
            </div>
            <div className="rt-engine-model">{snap?.comfy_running ? '运行中 · 画室生成队列可用' : '未运行'}</div>
          </div>
        </div>

        <div className="rt-eyebrow">最近请求（gaea 侧全引擎记账）</div>
        <div className="rt-panel">
          <Table<RecentCall>
            size="small"
            rowKey={(r, i) => `${r.finished_at}-${r.engine_id}-${i}`}
            columns={requestsColumns}
            dataSource={snap?.recent_calls ?? []}
            pagination={{ pageSize: 10, hideOnSinglePage: true, size: 'small' }}
            locale={{ emptyText: '暂无调用记录' }}
          />
        </div>
      </main>

      {/* 右检查列（统计面）：上下文 → 调用统计趋势 → 引擎侧请求。
          两路信息的「结合」：实时遥测在左一眼全貌，统计/上下文在右常驻对照 */}
      <aside className="rt-side" aria-label="统计与上下文">
        {em && (engineRequests.length > 0 || ctxMax != null) && (
          <div className="rt-panel rt-ctx-panel">
            <div className="rt-side-title">上下文 · 会话缓存</div>
            <div className="rt-ctx-donut">
              <ContextDonut used={ctxUsed} max={ctxMax} />
              <div className="rt-ctx-meta">
                <div className="rt-ctx-tokens">
                  {ctxUsed != null ? `${fmtKTok(ctxUsed)} / ${fmtKTok(ctxMax)} tok` : '等待首个请求'}
                </div>
                <div className="rt-ctx-model">{em.engine?.model ?? ''}</div>
              </div>
            </div>
            <div className="rt-cache">
              <div className="rt-cache-title">会话缓存</div>
              {cache ? (
                <div className="rt-cache-rows">
                  <span>{cache.enabled ? `预算 ${cache.budget_mib ?? '?'} MiB · ${cache.slots ?? '?'} 槽` : '未启用'}</span>
                  <span>近 {cache.requests ?? 0} 请求 · 复用 {cache.requests_reused ?? 0} 次</span>
                  {cache.prompt_tokens
                    ? <span>Prompt 复用 {fmtKTok(cache.reused_tokens ?? 0)} / {fmtKTok(cache.prompt_tokens)} tok（{Math.round(((cache.reused_tokens ?? 0) / cache.prompt_tokens) * 100)}%）</span>
                    : null}
                </div>
              ) : (
                <div className="rt-cache-rows">无数据</div>
              )}
            </div>
          </div>
        )}

        <div className="rt-panel rt-stats">
          <div className="rt-stats-head">
            <span className="rt-side-title">调用统计</span>
            <span style={{ flex: 1 }} />
            <Tooltip title="刷新统计">
              <Button size="small" type="text" icon={<ReloadOutlined />} aria-label="刷新统计" onClick={() => void stats.loadCallStats()} />
            </Tooltip>
          </div>
          <div className="rt-stats-sub">
            <Segmented
              size="small"
              value={stats.trendRange}
              onChange={(v) => stats.setTrendRange(v as TrendRange)}
              options={[
                { value: 'today', label: '今日' },
                { value: '7d', label: '7天' },
                { value: '30d', label: '30天' },
              ]}
            />
            <Button
              size="small"
              type="text"
              className="rt-stats-link"
              onClick={() => emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: 'modelcenter', crossSpace: true })}
            >
              按模型明细 →
            </Button>
          </div>
          <div className="rt-stats-strip">
            Token {fmtInt(totals.total_tokens ?? null)}
            <span className="rt-stats-sep" />
            入 {fmtInt(totals.input_tokens ?? null)} / 出 {fmtInt(totals.output_tokens ?? null)}
          </div>
          {stats.loadError ? (
            <div className="rt-error">统计加载失败：{stats.loadError}</div>
          ) : stats.trendData.length === 0 ? (
            <div className="rt-stats-empty">暂无趋势数据（对话、语音、办公等模块调用模型后自动累计）</div>
          ) : (
            <div className="rt-stats-charts">
              <div className="rt-stats-chart">
                <div className="rt-stats-chart-head">
                  <span>请求趋势</span>
                  <span>峰值 {trendPeak.toLocaleString()}</span>
                </div>
                <RequestsSpark data={stats.trendData} color="var(--color-primary)" />
              </div>
              <div className="rt-stats-chart">
                <div className="rt-stats-chart-head">
                  <span>Token 分布</span>
                  <span>入 / 出 堆叠</span>
                </div>
                <TokenMiniBars data={stats.trendData} />
              </div>
            </div>
          )}
        </div>

        {em && (engineRequests.length > 0 || ctxMax != null) && (
          <div className="rt-panel rt-engine-req">
            <div className="rt-side-title">引擎侧请求（{em === snap?.tianxuan ? '天璇' : 'Strata'} /metrics）</div>
            <Table<EngineRequest>
              size="small"
              rowKey={(r, i) => `${r.time}-${i}`}
              columns={engineReqColumns}
              dataSource={engineRequests}
              pagination={false}
              scroll={{ x: 620, y: 260 }}
              locale={{ emptyText: '引擎暂无已完成请求' }}
            />
          </div>
        )}
      </aside>
    </div>
  )
}

/** 模型名截短（chip 位宽有限；保留尾部档位段更有辨识度） */
function shortModel(id: string): string {
  if (id.length <= 28) return id
  const tail = id.slice(-24)
  return `…${tail}`
}
