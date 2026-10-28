import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Button, Segmented, Table, Tooltip } from 'antd'
import { MonitorOutlined, ReloadOutlined } from '@ant-design/icons'
import { getRuntimeMonitor } from '../api/engines'
import type { ModelUsageStats } from '../api/engines'
import { usePollingGate } from '../hooks/usePollingGate'
import { useStatsState } from './modelcenter/hooks/useStatsState'
import { type TrendDatum, type TrendRange } from './modelcenter/charts'
import { costToCNY, engineColor, engineLabel, fmtCost, USD_TO_CNY } from './modelcenter/utils'
import './runtime.css'

// RuntimePage — 运行监测板块（2026-10-19，参照 Strata Monitor 仪表盘）。
// 数据源 App.RuntimeMonitor 聚合快照（2s 轮询，页面不可见停拉）：
//   · 指标卡：引擎 /metrics 遥测（CPU/内存/磁盘/GPU 系列——Windows AMD 下
//     GPU 系列为 null=诚实显示「无遥测」）+ 系统统计兜底 + 生成速度 tok/s；
//     迷你趋势为前端滚动窗口（60 点 ≈ 2 分钟）；
//   · 引擎状态条：模型中心引擎表 + Strata 系引擎 live 状态（空闲/读入中/
//     生成中、队列、prompt 进度）+ ComfyUI 运行标记；
//   · 按模型明细（2026-10-27 拍板常驻平铺）：per_model 累计口径按引擎分组
//     全量平铺，不折叠不跳转（跳钮保留=抽屉还有 KPI/分流带全量统计）；
//   · 调用统计趋势：今日/7天/30天（useStatsState 同源 getModelCallStats；
//     2026-10 模型中心右侧统计检查器并入本板块——总计/成功率/费用在状态条，
//     趋势与入/出拆分在此）；
//   · 最近请求表：gaea 侧全引擎（含云端）调用记账（模型中心同源数据）。
// 视觉（2026-10-27 舰桥 HUD 全面重设计，用户判旧版「太low」；ui-ux-pro-max
// 技能 HUD/Sci-Fi FUI 样式卡处方）：指挥条 LIVE 徽标（轮询暂停诚实转「已暂停」/
// FUI 规约）+ 遥测主轴（核心速度大数+平滑波形+四弹道仪表+副读数行，替代八卡
// 网格）+ FUI 细线框（1px glow 边+角标）+ 终端化表格（mono/虚线行）+ CSS 计数
// 器编号节标——全部令牌驱动（--gaea-glow/--gaea-glass-bg/--v3-*），亮暗随主题；
// reduced-motion 扫描线/光晕停摆。可及性（技能 a11y 高风险项）：状态不只靠色
// （点+文字）、对比走令牌、焦点可见。

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

/** 主轴波形（HUD 重设计：tok/s 滚动窗口平滑面积波，宽幅细条） */
function HeroWave({ data }: { data: number[] }) {
  const W = 360, H = 46, padT = 6, padB = 4
  if (data.length < 2) {
    return <div className="rt-hero-wave-empty" aria-hidden="true">waiting for stream…</div>
  }
  const min = Math.min(...data)
  const max = Math.max(...data)
  const span = max - min || 1
  const pts = data.map((v, i) => [ (i / (data.length - 1)) * W, padT + (H - padT - padB) * (1 - (v - min) / span) ] as [number, number])
  const line = smoothPath(pts, padT, H - padB)
  const area = `${line} L${W},${H} L0,${H} Z`
  return (
    <svg className="rt-hero-wave" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" aria-hidden="true">
      <defs>
        <linearGradient id="rt-wave-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--gaea-glow)" stopOpacity="0.34" />
          <stop offset="100%" stopColor="var(--gaea-glow)" stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d={area} fill="url(#rt-wave-fill)" />
      <path d={line} fill="none" className="rt-hero-wave-line" strokeWidth={1.6} vectorEffect="non-scaling-stroke" />
    </svg>
  )
}

/** 弹道仪表（bullet gauge：值条 + 80% 阈值刻度；FUI 控制台签名件）。
 *  value 为展示串（'—'=无遥测诚实空轨），frac 为 0~1 满条比例。 */
function BulletGauge({ label, value, unit, frac }: { label: string; value: string; unit: string; frac: number | null }) {
  const has = value !== '—' && frac != null
  return (
    <div className={`rt-gauge${has ? '' : ' rt-gauge-empty'}`}>
      <div className="rt-gauge-head">
        <span className="rt-gauge-label">{label}</span>
        <span className="rt-gauge-val">
          {value}
          {has && unit && <i> {unit}</i>}
        </span>
      </div>
      <div className="rt-gauge-track">
        <span className="rt-gauge-fill" style={{ width: has ? `${Math.min(frac * 100, 100)}%` : '0%' }} />
        <span className="rt-gauge-tick" aria-hidden="true" title="负载阈值 80%" />
      </div>
    </div>
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

/** Catmull-Rom → 三次贝塞尔平滑路径（控制点钳回绘图区防尖峰过冲穿底） */
function smoothPath(pts: Array<[number, number]>, yMin: number, yMax: number): string {
  const clampY = (v: number) => Math.min(yMax, Math.max(yMin, v))
  if (pts.length < 3) return pts.map((p, i) => `${i ? 'L' : 'M'}${p[0].toFixed(1)},${clampY(p[1]).toFixed(1)}`).join(' ')
  let d = `M${pts[0][0].toFixed(1)},${clampY(pts[0][1]).toFixed(1)}`
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[i - 1] ?? pts[i]
    const p1 = pts[i]
    const p2 = pts[i + 1]
    const p3 = pts[i + 2] ?? p2
    const c1x = p1[0] + (p2[0] - p0[0]) / 6
    const c1y = clampY(p1[1] + (p2[1] - p0[1]) / 6)
    const c2x = p2[0] - (p3[0] - p1[0]) / 6
    const c2y = clampY(p2[1] - (p3[1] - p1[1]) / 6)
    d += ` C${c1x.toFixed(1)},${c1y.toFixed(1)} ${c2x.toFixed(1)},${c2y.toFixed(1)} ${p2[0].toFixed(1)},${clampY(p2[1]).toFixed(1)}`
  }
  return d
}

/** 请求趋势 HUD 图（2026-10-27 用户判旧迷你图「太low」重画）：平滑曲线 + 渐变
 *  辉光 + 虚线网格 + 峰值轴标 + 失败红点光晕 + 末点呼吸标记。数据/口径与旧
 *  RequestsSpark 完全一致（档位内 trendData），只换画法。 */
function RequestsHudChart({ data }: { data: TrendDatum[] }) {
  const W = 300, H = 112, padT = 14, padB = 12, padL = 6, padR = 10
  const plotW = W - padL - padR
  const plotH = H - padT - padB
  const maxV = Math.max(...data.map((d) => d.calls), 1)
  const x = (i: number) => (data.length === 1 ? padL + plotW / 2 : padL + (i / (data.length - 1)) * plotW)
  const y = (v: number) => padT + plotH * (1 - v / maxV)
  const pts = data.map((d, i) => [x(i), y(d.calls)] as [number, number])
  const line = smoothPath(pts, padT, padT + plotH)
  const base = padT + plotH
  const area = `${line} L${x(data.length - 1).toFixed(1)},${base} L${x(0).toFixed(1)},${base} Z`
  const last = data.length - 1
  const peakIdx = data.reduce((a, d, i) => (d.calls > data[a].calls ? i : a), 0)
  return (
    <svg className="rt-hchart" viewBox={`0 0 ${W} ${H}`} role="img" aria-label="请求趋势图">
      <defs>
        <linearGradient id="rt-req-stroke" x1="0" y1="0" x2="1" y2="0">
          <stop offset="0%" stopColor="var(--gaea-glow)" stopOpacity="0.25" />
          <stop offset="55%" stopColor="var(--gaea-glow)" stopOpacity="0.95" />
          <stop offset="100%" stopColor="var(--gaea-glow)" />
        </linearGradient>
        <linearGradient id="rt-req-area" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--gaea-glow)" stopOpacity="0.3" />
          <stop offset="100%" stopColor="var(--gaea-glow)" stopOpacity="0" />
        </linearGradient>
      </defs>
      {/* 虚线网格 + 轴标（mono 小字，max/半程） */}
      {[1, 0.5].map((f) => (
        <line key={f} className="rt-hchart-grid" x1={padL} x2={W - padR} y1={padT + plotH * (1 - f)} y2={padT + plotH * (1 - f)} />
      ))}
      <text className="rt-hchart-axis" x={padL + 2} y={padT - 4}>{maxV}</text>
      <text className="rt-hchart-axis" x={padL + 2} y={base - 3}>{Math.round(maxV / 2)}</text>
      <text className="rt-hchart-axis rt-hchart-axis-r" x={W - padR} y={base - 3}>0</text>
      <path d={area} fill="url(#rt-req-area)" />
      <path d={line} fill="none" stroke="url(#rt-req-stroke)" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" className="rt-hchart-line" />
      {/* 峰值刻度：短竖线 + 点 */}
      <line className="rt-hchart-peak" x1={x(peakIdx)} x2={x(peakIdx)} y1={y(data[peakIdx].calls) - 6} y2={y(data[peakIdx].calls) + 3} />
      {data.map((d, i) => (
        <g key={d.key}>
          {d.failCalls > 0 && (
            <circle className="rt-hchart-fail" cx={x(i)} cy={y(d.calls)} r={2.6}>
              <title>{`${d.label} · ${d.calls} 次（失败 ${d.failCalls}）`}</title>
            </circle>
          )}
          <circle cx={x(i)} cy={y(d.calls)} r={0}>
            <title>{`${d.label} · ${d.calls} 次 · 成功 ${d.successCalls} · 失败 ${d.failCalls}`}</title>
          </circle>
        </g>
      ))}
      {/* 末点：光晕环 + 实心点 */}
      <circle className="rt-hchart-last-halo" cx={x(last)} cy={y(data[last].calls)} r={6} />
      <circle className="rt-hchart-last" cx={x(last)} cy={y(data[last].calls)} r={2.8}>
        <title>{`${data[last].label} · ${data[last].calls} 次`}</title>
      </circle>
    </svg>
  )
}

/** Token 分布 HUD 堆叠柱（入=primary 下段 / 出=success 上段）：渐变柱 + 虚线
 *  网格 + 总量轴标 + 末柱点亮辉光。数据口径与旧 TokenMiniBars 一致。 */
function TokenHudBars({ data }: { data: TrendDatum[] }) {
  const W = 300, H = 112, padT = 14, padB = 12, padL = 6, padR = 10
  const plotW = W - padL - padR
  const plotH = H - padT - padB
  const maxTok = Math.max(...data.map((d) => d.inputTokens + d.outputTokens), 1)
  const slot = plotW / data.length
  const barW = Math.max(3, Math.min(26, slot * 0.62))
  const base = padT + plotH
  return (
    <svg className="rt-hchart" viewBox={`0 0 ${W} ${H}`} role="img" aria-label="Token 分布图">
      <defs>
        <linearGradient id="rt-tok-in" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--color-primary)" stopOpacity="0.55" />
          <stop offset="100%" stopColor="var(--color-primary)" stopOpacity="0.9" />
        </linearGradient>
        <linearGradient id="rt-tok-out" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--color-success)" stopOpacity="0.95" />
          <stop offset="100%" stopColor="var(--color-success)" stopOpacity="0.6" />
        </linearGradient>
      </defs>
      {[1, 0.5].map((f) => (
        <line key={f} className="rt-hchart-grid" x1={padL} x2={W - padR} y1={padT + plotH * (1 - f)} y2={padT + plotH * (1 - f)} />
      ))}
      <text className="rt-hchart-axis" x={padL + 2} y={padT - 4}>{fmtKTok(maxTok)}</text>
      <text className="rt-hchart-axis" x={padL + 2} y={base - 3}>{fmtKTok(maxTok / 2)}</text>
      <text className="rt-hchart-axis rt-hchart-axis-r" x={W - padR} y={base - 3}>0</text>
      {data.map((d, i) => {
        const x0 = padL + (i + 0.5) * slot - barW / 2
        const hIn = (d.inputTokens / maxTok) * plotH
        const hOut = (d.outputTokens / maxTok) * plotH
        const yIn = base - hIn
        const yOut = yIn - hOut
        const isLast = i === data.length - 1
        return (
          <g key={d.key} className={isLast ? 'rt-tok-last' : undefined}>
            <rect className="rt-tok-bar rt-tok-in" x={x0} y={yIn} width={barW} height={Math.max(0, hIn)} rx={Math.min(2.5, barW / 3)}>
              <title>{`${d.label} · 输入 ${d.inputTokens.toLocaleString()} token`}</title>
            </rect>
            <rect className="rt-tok-bar rt-tok-out" x={x0} y={yOut} width={barW} height={Math.max(0, hOut)} rx={Math.min(2.5, barW / 3)}>
              <title>{`${d.label} · 输出 ${d.outputTokens.toLocaleString()} token`}</title>
            </rect>
          </g>
        )
      })}
    </svg>
  )
}

/** Context 用量环（最近一次请求的 prompt_total+输出 vs 引擎 max_context） */
function ContextDonut({ used, max }: { used: number | null; max: number | null }) {  const pct = used != null && max && max > 0 ? Math.min(used / max, 1) : null
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

/** 按模型明细常驻板（2026-10-27 用户拍板「不要折叠隐藏」）：原「按模型明细 →」
 *  跳钮指向的明细本体（模型中心抽屉同源 getModelCallStats）平铺进本页，
 *  按引擎分组、全部模型行常驻（无折叠/展开/弹层）。口径=**累计**——per_model
 *  后端全量聚合，不随右列 今日/7天/30天 档位过滤（档位只管趋势图，两套口径
 *  在节标与右列文案里各自明示，防误读）。跳钮已随拍板移除（抽屉取消=本页即
 *  明细唯一入口；模型中心自有「详细统计」钮不动）。 */
function PerModelBoard({ rows, usdToCny }: { rows: ModelUsageStats[]; usdToCny?: number }) {
  const groups = new Map<string, ModelUsageStats[]>()
  for (const r of rows) {
    const list = groups.get(r.engine_id)
    if (list) list.push(r)
    else groups.set(r.engine_id, [r])
  }
  return (
    <div className="rt-mb" aria-label="按模型明细">
      <div className="rt-mb-caption" aria-hidden="true">
        <span>模型</span><span>调用</span><span>入 / 出 Token</span><span>成功率</span><span>费用</span><span>失败</span>
      </div>
      {Array.from(groups.entries()).map(([engineId, list]) => {
        const color = engineColor({ engineId })
        const calls = list.reduce((a, b) => a + b.call_count, 0)
        const succ = list.reduce((a, b) => a + b.success_count, 0)
        const tokens = list.reduce((a, b) => a + b.total_tokens, 0)
        const cost = list.reduce((a, b) => a + costToCNY(b.estimated_cost ?? 0, b.currency, usdToCny ?? USD_TO_CNY), 0)
        const rate = calls > 0 ? (succ / calls) * 100 : 0
        const maxCalls = Math.max(...list.map((s) => s.call_count), 1)
        return (
          <section key={engineId} className="rt-mb-group">
            <div className="rt-mb-head">
              <span className="rt-mb-dot" style={{ background: color }} />
              <span className="rt-mb-engine">{engineLabel({ engineId })}</span>
              <span className="rt-mb-sub">{list.length} 个模型</span>
              <span className="rt-mb-sum">
                调用 {calls.toLocaleString()} · {fmtKTok(tokens)} tok · {fmtCost(cost, 'CNY')} ·{' '}
                <span className={rate >= 99 ? 'rt-ok' : 'rt-fail'}>{rate.toFixed(0)}%</span>
              </span>
            </div>
            {list.map((r) => {
              const share = Math.max((r.call_count / maxCalls) * 100, 2)
              const rRate = r.call_count > 0 ? (r.success_count / r.call_count) * 100 : 0
              return (
                <div key={`${r.engine_id}-${r.model}`} className="rt-mb-row">
                  <span className="rt-mb-name" title={r.model}>{r.model}</span>
                  <span className="rt-mb-calls">
                    <span className="rt-mb-bar" style={{ width: `${share}%` }} />
                    <span className="rt-mb-calls-num">{r.call_count.toLocaleString()}</span>
                  </span>
                  <span className="rt-mb-tok">{fmtKTok(r.input_tokens)} / {fmtKTok(r.output_tokens)}</span>
                  <span className={`rt-mb-rate ${rRate >= 99 ? 'rt-ok' : 'rt-fail'}`}>{rRate.toFixed(0)}%</span>
                  <span className="rt-mb-cost">{fmtCost(costToCNY(r.estimated_cost ?? 0, r.currency, usdToCny ?? USD_TO_CNY), 'CNY')}</span>
                  {r.fail_count > 0 ? (
                    <Tooltip title={r.last_error || `累计失败 ${r.fail_count} 次`}>
                      <span className="rt-mb-fail">败 {r.fail_count}</span>
                    </Tooltip>
                  ) : (
                    <span className="rt-mb-fail rt-mb-fail-none" aria-hidden="true">—</span>
                  )}
                </div>
              )
            })}
          </section>
        )
      })}
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

  /** 遥测主轴数据（HUD 重设计：核心速度 + 四弹道仪表 + 副读数行；替代旧八卡网格）。
   *  展示串逐项保持旧口径（测试锁值：62/31/12.2/96/12.5 / 3.2 等），只换容器。 */
  const hero = useMemo(() => {
    if (!snap) return null
    const hw = em?.hardware ?? {}
    const sys = snap.system ?? {}
    const gpuName = typeof sys.gpuName === 'string' ? sys.gpuName : ''
    const gpuHint = gpuName || undefined
    const ramUsed = num(hw.ram_used) ?? num(sys.memUsed)
    const ramTotal = num(hw.ram_total) ?? num(sys.memTotal)
    const vramUsed = num(sys.vramUsed)
    const vramTotal = num(sys.vramTotal)
    const tokS = num(em?.live?.tok_s)
    const gpuUtil = num(hw.gpu_util) ?? num(sys.gpuUsage)
    const cpu = num(hw.cpu) ?? num(sys.cpu)
    const diskRead = num(hw.disk_read_mb)
    const diskWrite = num(hw.disk_write_mb)
    const diskValue =
      diskRead == null && diskWrite == null
        ? null
        : `${diskRead?.toFixed(1) ?? '—'} / ${diskWrite?.toFixed(1) ?? '—'}`
    const frac = (used: number | null, total: number | null) =>
      used != null && total != null && total > 0 ? Math.min(used / total, 1) : null
    return {
      tokS,
      gpuHint,
      gauges: [
        { label: 'GPU 负载', value: gpuUtil?.toFixed(0) ?? '—', unit: '%', frac: gpuUtil != null ? gpuUtil / 100 : null, hint: gpuHint },
        { label: 'CPU', value: cpu?.toFixed(0) ?? '—', unit: '%', frac: cpu != null ? cpu / 100 : null },
        { label: '内存', value: ramUsed == null ? '—' : `${ramUsed.toFixed(1)}/${ramTotal?.toFixed(1) ?? '?'}`, unit: 'GB', frac: frac(ramUsed, ramTotal) },
        { label: '显存', value: vramUsed == null || vramUsed <= 0 ? '—' : `${vramUsed.toFixed(1)}/${vramTotal?.toFixed(1) ?? '?'}`, unit: 'GB', frac: frac(vramUsed, vramTotal), hint: gpuHint },
      ] as Array<{ label: string; value: string; unit: string; frac: number | null; hint?: string }>,
      chips: [
        { label: 'GPU 温度', value: num(hw.gpu_temp)?.toFixed(0) ?? '—', unit: '°C', hint: gpuHint },
        { label: 'GPU 功率', value: num(hw.gpu_power)?.toFixed(0) ?? '—', unit: 'W', hint: gpuHint },
        { label: '磁盘 读/写', value: diskValue ?? '—', unit: 'MB/s' },
      ] as Array<{ label: string; value: string; unit: string; hint?: string }>,
    }
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
  // （getModelCallStats，30s 轮询走 usePollingGate）。头部 totals 行=累计口径
  // （2s 快照）；本面板数字条与双图均为**档位内汇总**——今日/7天/30天切换时
  // 调用数、Token 入/出、费用随 trendData 重算，不再混累计值。
  const stats = useStatsState(false)
  const trendPeak = useMemo(() => stats.trendData.reduce((a, b) => Math.max(a, b.calls), 0), [stats.trendData])
  const trendSums = useMemo(
    () =>
      stats.trendData.reduce(
        (a, b) => ({
          calls: a.calls + b.calls,
          in: a.in + b.inputTokens,
          out: a.out + b.outputTokens,
          cost: a.cost + b.cost,
        }),
        { calls: 0, in: 0, out: 0, cost: 0 },
      ),
    [stats.trendData],
  )

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
      {/* HUD 氛围层（2026-10-27 舰桥改版：星云底 + 网格 + 扫描线；pointer-events
          全关不挡交互，reduced-motion 下扫描线停摆——见 runtime.css 降级段） */}
      <div className="rt-hud-bg" aria-hidden="true" />
      <div className="rt-hud-grid" aria-hidden="true" />
      <div className="rt-hud-scanline" aria-hidden="true" />
      {/* D1 指挥条（HUD 重设计）：图标章+工作区名+LIVE 徽标（轮询门控暂停时
          诚实转「已暂停」——FUI 规约：遥测只在有活数据源时标 LIVE）+实时状态+
          累计口径+动作区 */}
      <header className="rt-ph">
        <span className="rt-ph-seat" aria-hidden="true"><MonitorOutlined /></span>
        <span className="rt-ph-name">运行监测</span>
        <span className={`rt-live${pollable ? '' : ' rt-live-stale'}`}>
          <span className="rt-live-dot" aria-hidden="true" />
          {pollable ? 'LIVE' : '已暂停'}
        </span>
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

      {/* 左主列（实时面）：遥测主轴 → 引擎阵列 → 模型调用 → 请求流水 */}
      <main className="rt-main" aria-label="实时监测">
        <div className="rt-eyebrow">硬件遥测</div>
        {!hero ? (
          <div className="rt-loading">正在读取监测数据…</div>
        ) : (
          <section className="rt-hero" aria-label="遥测主轴">
            <div className="rt-hero-core">
              <div className="rt-hero-label">
                生成速度
                {overall && overall.tone !== 'off' && (
                  <span className={`rt-hero-state rt-hero-state-${overall.tone}`}>{overall.label}</span>
                )}
              </div>
              <div className="rt-hero-value">
                {hero.tokS == null ? '—' : hero.tokS.toFixed(1)}
                {hero.tokS != null && <span className="rt-hero-unit"> tok/s</span>}
              </div>
              <HeroWave data={seriesRef.current['tok_s'] ?? []} />
            </div>
            <div className="rt-hero-gauges">
              {hero.gauges.map((g) => (
                <BulletGauge key={g.label} label={g.label} value={g.value} unit={g.unit} frac={g.frac} />
              ))}
            </div>
            <div className="rt-hero-chips">
              {hero.chips.map((c) => (
                <span key={c.label} className="rt-readout" title={c.hint}>
                  <span className="rt-readout-label">{c.label}</span>
                  <span className="rt-readout-val">
                    {c.value}
                    {c.value !== '—' && <i> {c.unit}</i>}
                  </span>
                </span>
              ))}
            </div>
          </section>
        )}

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

        <div className="rt-eyebrow">按模型明细<span className="rt-eyebrow-note">累计口径 · 全引擎平铺</span></div>
        <div className="rt-panel rt-models-panel">
          {!stats.callStats ? (
            <div className="rt-stats-empty">明细加载中…</div>
          ) : (stats.callStats.per_model ?? []).length === 0 ? (
            /* per_model 后端 nil 切片 → JSON null（Wails 系统性契约）：一律 ?? [] */
            <div className="rt-stats-empty">暂无模型调用记录（对话、语音、办公等模块调用模型后自动累计）</div>
          ) : (
            <PerModelBoard rows={stats.callStats.per_model ?? []} usdToCny={stats.callStats.usd_to_cny} />
          )}
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
          </div>
          {stats.loadError ? (
            <div className="rt-error rt-stats-error">
              <span>统计加载失败：{stats.loadError}</span>
              <Button size="small" type="primary" ghost onClick={() => void stats.loadCallStats()}>重试</Button>
            </div>
          ) : !stats.callStats ? (
            <div className="rt-stats-empty">正在读取统计…</div>
          ) : stats.trendData.length === 0 ? (
            <div className="rt-stats-empty">暂无趋势数据（对话、语音、办公等模块调用模型后自动累计）</div>
          ) : (
            <>
              {/* 档位内汇总读数（随今日/7天/30天重算；累计口径在头部 totals 行） */}
              <div className="rt-stats-strip">
                <span>调用 {trendSums.calls.toLocaleString()}</span>
                <span className="rt-stats-sep" />
                <span>Token {fmtKTok(trendSums.in + trendSums.out)} · 入 {fmtKTok(trendSums.in)} / 出 {fmtKTok(trendSums.out)}</span>
                <span className="rt-stats-sep" />
                <span>¥{trendSums.cost.toFixed(2)}</span>
              </div>
              <div className="rt-stats-charts">
                <div className="rt-stats-chart">
                  <div className="rt-stats-chart-head">
                    <span>请求趋势</span>
                    <span>峰值 {trendPeak.toLocaleString()}</span>
                  </div>
                  <RequestsHudChart data={stats.trendData} />
                </div>
                <div className="rt-stats-chart">
                  <div className="rt-stats-chart-head">
                    <span>Token 分布</span>
                    <span className="rt-tok-legend">
                      <i className="rt-tok-dot rt-tok-dot-in" />入
                      <i className="rt-tok-dot rt-tok-dot-out" />出
                    </span>
                  </div>
                  <TokenHudBars data={stats.trendData} />
                </div>
              </div>
            </>
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
