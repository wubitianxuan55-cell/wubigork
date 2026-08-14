import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button, Input, Popconfirm, message } from 'antd'
import { AppstoreOutlined, BarChartOutlined, DatabaseOutlined, HddOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import {
  downloadHerdsmanModel,
  getHerdsmanCatalog,
  getHerdsmanLaunchPresets,
  getHerdsmanModelStats,
  startHerdsmanModel,
  stopHerdsmanModel,
  uninstallHerdsmanModel,
  type HerdsmanCatalogModel,
  type HerdsmanLaunchPreset,
  type HerdsmanModelStat,
  type HerdsmanModelStats,
  type HerdsmanOpResult,
} from '../../api/engines'
import { EmptyState, KpiTile, ModelCard, SectionHead, StatusChip, type StatusTone } from './ui'

type StatusFilter = 'all' | 'installed' | 'running' | 'uninstalled'

const STATUS_FILTERS: { key: StatusFilter; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'installed', label: '已安装' },
  { key: 'running', label: '运行中' },
  { key: 'uninstalled', label: '未安装' },
]

const TYPE_TONE: Record<string, StatusTone> = {
  'text-generation': 'accent',
  multimodal: 'accent',
  embedding: 'ok',
  reranking: 'warn',
  ocr: 'ok',
  asr: 'neutral',
  tts: 'neutral',
  'image-generation': 'accent',
  'audio-generation': 'neutral',
}

function fmtSize(n?: number): string {
  if (!n || n <= 0) return ''
  const mb = n / 1024 / 1024
  if (mb < 1024) return `${mb.toFixed(1)} MB`
  const gb = mb / 1024
  if (gb < 1024) return `${gb.toFixed(1)} GB`
  return `${(gb / 1024).toFixed(1)} TB`
}

function fmtParams(m: HerdsmanCatalogModel): string {
  const p = m.parameter_count ?? 0
  if (p <= 0) return ''
  const active = m.is_moe && m.active_parameters ? `（激活 ${m.active_parameters}B）` : ''
  return `${p}B${active}`
}

function fmtMs(ms: number): string {
  if (!ms || ms <= 0) return '—'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function fmtTps(v?: number): string {
  if (!v || v <= 0) return '—'
  return v.toFixed(1)
}

/** 模型中心「模型库」：Herdsman 完整模型目录（90 个已知模型，含能力/状态/量化/变体）。 */
export function HerdsmanCatalogSection() {
  const [models, setModels] = useState<HerdsmanCatalogModel[]>([])
  const [total, setTotal] = useState(0)
  const [installed, setInstalled] = useState(0)
  const [running, setRunning] = useState(0)
  const [disk, setDisk] = useState<{ installedBytes: number; total: number; free: number; error?: string } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [busy, setBusy] = useState<Set<string>>(new Set())
  const [presets, setPresets] = useState<Record<string, HerdsmanLaunchPreset>>({})
  const [stats, setStats] = useState<HerdsmanModelStats | null>(null)
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<StatusFilter>('all')
  const [type, setType] = useState<string>('all')

  const loadPresets = useCallback(async () => {
    try {
      const list = await getHerdsmanLaunchPresets()
      const map: Record<string, HerdsmanLaunchPreset> = {}
      for (const p of list) map[p.model] = p
      setPresets(map)
    } catch {
      setPresets({})
    }
  }, [])

  const loadStats = useCallback(async () => {
    try {
      const s = await getHerdsmanModelStats()
      setStats(s)
    } catch {
      setStats(null)
    }
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [c] = await Promise.all([getHerdsmanCatalog(), loadPresets(), loadStats()])
      setModels(c.models ?? [])
      setTotal(c.total ?? c.models?.length ?? 0)
      setInstalled(c.installed ?? 0)
      setRunning(c.running ?? 0)
      setDisk({
        installedBytes: c.installed_bytes ?? 0,
        total: c.disk_total ?? 0,
        free: c.disk_free ?? 0,
        error: c.disk_error,
      })
      if (c.error) setError(c.error)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e))
      setModels([])
      setTotal(0)
      setInstalled(0)
      setRunning(0)
    } finally {
      setLoading(false)
    }
  }, [loadPresets, loadStats])

  useEffect(() => { void load() }, [load])

  const types = useMemo(() => {
    const s = new Set(models.map(m => m.type).filter(Boolean))
    return Array.from(s).sort()
  }, [models])

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    return models.filter(m => {
      if (status === 'installed' && !m.installed) return false
      if (status === 'running' && !m.running) return false
      if (status === 'uninstalled' && m.installed) return false
      if (type !== 'all' && m.type !== type) return false
      if (!q) return true
      const caps = (m.capabilities ?? []).join(' ')
      return m.name.toLowerCase().includes(q)
        || m.display_name.toLowerCase().includes(q)
        || caps.toLowerCase().includes(q)
    })
  }, [models, search, status, type])

  const runOp = useCallback(async (m: HerdsmanCatalogModel, label: string, op: () => Promise<HerdsmanOpResult>) => {
    setBusy(prev => new Set(prev).add(m.name))
    try {
      const r = await op()
      if (!r.ok) throw new Error(r.message || '操作失败')
      message.success(`${m.display_name || m.name} ${label}成功`)
      await load()
    } catch (e: unknown) {
      message.error(`${m.display_name || m.name} ${label}失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setBusy(prev => {
        const next = new Set(prev)
        next.delete(m.name)
        return next
      })
    }
  }, [load])

  const cardAction = (m: HerdsmanCatalogModel) => {
    const isBusy = busy.has(m.name)
    if (m.running) {
      return (
        <Button size="small" danger loading={isBusy} onClick={() => void runOp(m, '停止', () => stopHerdsmanModel(m.name))}>
          停止
        </Button>
      )
    }
    if (m.installed) {
      return (
        <span className="mc-catalog-actions">
          <Button size="small" type="primary" loading={isBusy} onClick={() => void runOp(m, '启动', () => startHerdsmanModel(m.name))}>
            启动
          </Button>
          <Popconfirm
            title={`卸载 ${m.display_name || m.name}？`}
            description="将删除已下载的模型文件"
            okText="卸载"
            okButtonProps={{ danger: true }}
            cancelText="取消"
            onConfirm={() => void runOp(m, '卸载', () => uninstallHerdsmanModel(m.name))}
          >
            <Button size="small" type="text" loading={isBusy}>卸载</Button>
          </Popconfirm>
        </span>
      )
    }
    return (
      <Button size="small" loading={isBusy} onClick={() => void runOp(m, '下载', () => downloadHerdsmanModel(m.name))}>
        下载
      </Button>
    )
  }

  const presetChip = (m: HerdsmanCatalogModel) => {
    const p = presets[m.name]
    if (!p) return null
    const detail = Object.entries(p.options)
      .map(([k, v]) => `${k}=${String(v)}`)
      .join(' · ')
    return (
      <StatusChip key="preset" tone="accent" title={`本机实测启动参数：${detail}`}>
        启动预设
      </StatusChip>
    )
  }

  return (
    <section className="mc-section">
      <SectionHead
        icon={<AppstoreOutlined />}
        title="Herdsman 模型库"
        desc="来自 herdsman.exe 的完整本地模型目录（90 个已知模型）：能力、安装/运行状态、量化与变体一目了然，可先安装再在引擎管理中启用"
        extra={
          <Button size="small" icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>
            刷新目录
          </Button>
        }
      />

      <div className="mc-catalog-kpis">
        <KpiTile icon={<AppstoreOutlined />} label="已知模型" value={error ? '—' : total} hint="可安装/已安装合计" />
        <KpiTile icon={<SearchOutlined />} label="已安装" value={installed} hint="本机已下载" />
        <KpiTile icon={<ReloadOutlined />} label="运行中" value={running} hint="当前加载服务" />
        <KpiTile
          icon={<DatabaseOutlined />}
          label="已装空间"
          value={error ? '—' : disk && disk.installedBytes > 0 ? fmtSize(disk.installedBytes) : '—'}
          hint="已下载模型文件占用（不含未装）"
        />
        <KpiTile
          icon={<HddOutlined />}
          label="磁盘余量"
          value={error || !disk || disk.total <= 0 ? '—' : `${fmtSize(disk.free)} / ${fmtSize(disk.total)}`}
          hint={disk?.error ? `探测失败：${disk.error}` : '模型数据目录所在卷'}
        />
      </div>

      {error && (
        <EmptyState
          icon={<ReloadOutlined />}
          title="模型目录不可用"
          hint={`${error}。请确认 Herdsman 桌面端已启动，或设置 HERDSMAN_EXE 环境变量后重试。`}
        />
      )}

      {!error && (
        <>
          <div className="mc-catalog-filters">
            <Input.Search
              allowClear
              placeholder="搜索模型名称 / 能力（如 translation、voice-clone）"
              value={search}
              onChange={e => setSearch(e.target.value)}
              style={{ maxWidth: 360 }}
            />
            <div className="mc-catalog-filters-right">
              <select
                className="mc-catalog-type"
                value={type}
                onChange={e => setType(e.target.value)}
                title="按类型过滤"
              >
                <option value="all">全部类型</option>
                {types.map(t => <option key={t} value={t}>{t}</option>)}
              </select>
              <div className="mc-catalog-status">
                {STATUS_FILTERS.map(f => (
                  <button
                    key={f.key}
                    type="button"
                    className={`mc-catalog-status-btn${status === f.key ? ' is-active' : ''}`}
                    onClick={() => setStatus(f.key)}
                  >
                    {f.label}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {filtered.length === 0 ? (
            <EmptyState
              compact
              icon={<SearchOutlined />}
              title="没有匹配的模型"
              hint="换个关键词或调整过滤条件"
            />
          ) : (
            <div className="mc-grid">
              {filtered.map(m => {
                const chips: React.ReactNode[] = []
                for (const cap of m.capabilities ?? []) {
                  chips.push(<StatusChip key={`cap-${cap}`} tone="neutral">{cap}</StatusChip>)
                }
                const q = m.quantization
                if (q) chips.push(<StatusChip key="q" tone="neutral">{q}</StatusChip>)
                const size = fmtSize(m.file_size)
                if (size) chips.push(<StatusChip key="size" tone="neutral">{size}</StatusChip>)
                const params = fmtParams(m)
                if (params) chips.push(<StatusChip key="params" tone="neutral">{params}</StatusChip>)
                if (m.is_moe) chips.push(<StatusChip key="moe" tone="accent">MoE</StatusChip>)
                const preset = presetChip(m)
                if (preset) chips.push(preset)
                return (
                  <ModelCard
                    key={m.name}
                    name={m.display_name || m.name}
                    engineId="herdsman"
                    engineName="Herdsman 本地"
                    desc={m.hint}
                    kindChip={
                      <StatusChip tone={TYPE_TONE[m.type] ?? 'neutral'}>
                        {m.type}
                      </StatusChip>
                    }
                    chips={chips}
                    status={{
                      tone: m.running ? 'ok' : m.installed ? 'warn' : 'neutral',
                      text: m.running ? '运行中' : m.installed ? '已安装 · 未运行' : '未安装',
                    }}
                    action={cardAction(m)}
                  />
                )
              })}
            </div>
          )}

          {stats && stats.total > 0 && !stats.error && (
            <div className="mc-panel mc-catalog-stats">
              <div className="mc-catalog-stats-head">
                <span className="mc-panel-title">
                  <BarChartOutlined /> Herdsman 本地调用统计
                </span>
                <span className="mc-catalog-stats-meta">
                  来自 model_stats/events.jsonl{stats.since ? ` · 自 ${stats.since}` : ''}
                </span>
              </div>
              <div className="mc-overview-grid">
                <KpiTile label="模型数" value={stats.total} />
                <KpiTile
                  label="总调用"
                  value={stats.per_model.reduce((n, s) => n + s.calls, 0).toLocaleString()}
                  hint={`成功 ${stats.per_model.reduce((n, s) => n + s.succeeded, 0)} · 失败 ${stats.per_model.reduce((n, s) => n + s.failed, 0)}`}
                />
                <KpiTile
                  label="Token 用量"
                  value={(stats.per_model.reduce((n, s) => n + s.input_tokens + s.output_tokens, 0)).toLocaleString()}
                />
              </div>
              <table className="mc-catalog-stats-table">
                <thead>
                  <tr>
                    <th>模型</th>
                    <th>类型</th>
                    <th>调用</th>
                    <th>成功/失败</th>
                    <th>Token(入/出)</th>
                    <th>平均耗时</th>
                    <th>TTFT</th>
                    <th>Prompt/预测 TPS</th>
                  </tr>
                </thead>
                <tbody>
                  {stats.per_model.map(s => (
                    <tr key={s.model}>
                      <td className="mc-catalog-stats-model" title={s.model}>{s.model}</td>
                      <td>{s.type || '—'}</td>
                      <td>{s.calls}</td>
                      <td>{s.succeeded}/{s.failed}</td>
                      <td>{s.input_tokens}/{s.output_tokens}</td>
                      <td>{fmtMs(s.avg_duration_ms)}</td>
                      <td>{fmtMs(Math.round(s.avg_ttft_ms))}</td>
                      <td>{fmtTps(s.avg_prompt_tps)} / {fmtTps(s.avg_predicted_tps)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </section>
  )
}
