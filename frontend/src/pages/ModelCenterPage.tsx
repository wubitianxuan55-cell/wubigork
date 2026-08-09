import React, { useState, useEffect, useCallback, useMemo } from 'react'
import { Typography, Card, Switch, Button, Input, Space, Tag, message, Spin, Collapse, Select, Segmented } from 'antd'
import {
  CloudOutlined, CheckCircleOutlined,
  CloseCircleOutlined, ReloadOutlined, ThunderboltOutlined,
  DesktopOutlined, RocketOutlined, PictureOutlined, SoundOutlined, AudioOutlined,
  CaretRightOutlined, SettingOutlined, LoginOutlined, LogoutOutlined, KeyOutlined, GlobalOutlined,
  LinkOutlined,
} from '@ant-design/icons'
import { useAppStore } from '../stores/appStore'
import SettingField from '../components/SettingField'
import { C } from '../utils/theme'
import * as App from '../../wailsjs/go/app/App'
import {
  getEngines, saveEngine, testEngineConnection,
  refreshEngineModels, setEngineDefaultModel,
  setActiveEngine, getActiveEngine, setDeepseekKey, getDeepseekKeyStatus,
  setOpencodeGoKey, getOpencodeGoKeyStatus,
  setOpencodeZenKey, getOpencodeZenKeyStatus,
  getModelCallStats, resetModelCallStats,
  type EngineConfig, type ModelInfo, type EngineStatus, type ModelStatsSummary,
} from '../api/engines'
import {
  getConfig, saveConfig,
  getImageBackendInfo, setImageBackend as setImageBackendAPI,
} from '../api/settings'
import { getPortraitConfig, setPortraitConfig } from '../api/image'
import { startComfyUI, stopComfyUI, getComfyUIStatus } from '../api/image'

type Category = 'llm' | 'image' | 'tts' | 'engine' | 'bind' | 'stats'

interface ModelCardData {
  modelId: string; modelName: string
  engineId: string; engineName: string
  engineType: string; engineEnabled: boolean
  status: string
}

type ModelKind = 'llm' | 'tts' | 'stt' | 'image'

function classifyModel(id: string): ModelKind {
  const lid = id.toLowerCase()
  if (lid.includes('tts') || lid.includes('voice') || lid.includes('edge')) return 'tts'
  if (lid.includes('sherpa') || lid.includes('whisper') || lid.includes('zipformer') || lid.includes('asr')) return 'stt'
  if (lid.includes('image') || lid.includes('zimage') || lid.includes('flux') || lid.includes('turbo') || lid.includes('sd') || lid.includes('dalle') || lid.includes('krea')) return 'image'
  return 'llm'
}

const engineIcons: Record<string, React.ReactNode> = {
  xai: <CloudOutlined />, ollama: <DesktopOutlined />, herdsman: <RocketOutlined />, deepseek: <KeyOutlined />, cosyvoice: <RocketOutlined />, 'opencode-go': <GlobalOutlined />, 'opencode-zen': <GlobalOutlined />,
}
const engineColors: Record<string, string> = {
  xai: '#60a5fa', ollama: '#f59e0b', herdsman: '#84cc16', deepseek: '#8b5cf6', cosyvoice: '#f472b6', 'opencode-go': '#22d3ee', 'opencode-zen': '#a78bfa',
}
const engineLabels: Record<string, string> = {
  xai: 'xAI 云端', ollama: 'Ollama 本地', herdsman: 'Herdsman 本地', deepseek: 'DeepSeek 云端', cosyvoice: 'CosyVoice2 本地', 'opencode-go': 'OpenCode Go 云端', 'opencode-zen': 'OpenCode Zen 云端',
}

// xAI Grok TTS 音色（与设置面板一致，模型中心绑定卡内可直接选择）
const XAI_VOICES = [
  { value: 'eve', label: 'Eve（默认）' },
  { value: 'ara', label: 'Ara（温暖友好）' },
  { value: 'rex', label: 'Rex（自信清晰）' },
  { value: 'sal', label: 'Sal（平滑均衡）' },
  { value: 'leo', label: 'Leo（权威）' },
  { value: 'lumen', label: 'Lumen' },
  { value: 'castor', label: 'Castor' },
  { value: 'naksh', label: 'Naksh' },
  { value: 'atlas', label: 'Atlas' },
  { value: 'carina', label: 'Carina' },
  { value: 'zagan', label: 'Zagan' },
  { value: 'helix', label: 'Helix' },
  { value: 'orion', label: 'Orion' },
  { value: 'luna', label: 'Luna' },
  { value: 'celeste', label: 'Celeste' },
  { value: 'cosmo', label: 'Cosmo' },
  { value: 'helios', label: 'Helios' },
  { value: 'iris', label: 'Iris' },
  { value: 'kepler', label: 'Kepler' },
  { value: 'lux', label: 'Lux' },
  { value: 'perseus', label: 'Perseus' },
  { value: 'rigel', label: 'Rigel' },
  { value: 'sirius', label: 'Sirius' },
  { value: 'ursa', label: 'Ursa' },
  { value: 'zenith', label: 'Zenith' },
  { value: 'altair', label: 'Altair' },
]

// CosyVoice2 内置音色兜底（服务端 /v1/audio/info 查询失败时）
const COSYVOICE_FALLBACK_VOICES = ['中文女', '中文男', '英文女', '英文男']


// ── 费用展示工具（与后端 usdToCny=7.2 保持一致） ────────────────
const USD_TO_CNY = 7.2
const isLocalEngine = (id: string) => id === 'ollama' || id === 'herdsman' || id === 'cosyvoice'

// 本地 TTS 引擎的服务端音色兜底：CosyVoice2 4 个内置音色（中文女/男、英文女/男）
const localTTSFallbackVoices = (model: string): string[] => {
  const l = model.toLowerCase()
  if (l.includes('cosyvoice')) return COSYVOICE_FALLBACK_VOICES
  return []
}
const localTTSDefaultVoice = (model: string): string | undefined => {
  const l = model.toLowerCase()
  return l.includes('cosyvoice') ? '中文女' : undefined
}
const costToCNY = (cost: number, currency?: string) => (currency === 'USD' ? cost * USD_TO_CNY : cost)
const fmtCost = (cost?: number, currency?: string): string => {
  const c = cost ?? 0
  if (currency === 'USD') return `$${c.toFixed(2)}`
  if (currency === 'CNY') return `¥${c.toFixed(2)}`
  return ''
}

type StatsSort = 'calls' | 'tokens' | 'cost'
type TrendRange = 'today' | '7d' | '30d'

interface TrendDatum {
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
const RequestsTrendChart: React.FC<{ data: TrendDatum[]; color: string }> = ({ data, color }) => {
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
          <line x1={padL} y1={y(t)} x2={W - padR} y2={y(t)} stroke="rgba(128,128,128,0.14)" strokeWidth={1} />
          <text x={padL - 6} y={y(t) + 3} textAnchor="end" fontSize={10} style={{ fill: C('color-text-secondary') }}>{fmtCompact(t)}</text>
        </g>
      ))}
      <path d={area} fill={`${color}20`} />
      <path d={line} fill="none" stroke={color} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" />
      {data.map((d, i) => (
        <g key={d.key}>
          {d.failCalls > 0 && (
            <circle cx={x(i)} cy={y(d.calls)} r={3} fill="#f87171" stroke="#1e1e2e" strokeWidth={1}>
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
const TokenTrendChart: React.FC<{ data: TrendDatum[] }> = ({ data }) => {
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
          <line x1={padL} y1={yT(t)} x2={W - padR} y2={yT(t)} stroke="rgba(128,128,128,0.14)" strokeWidth={1} />
          <text x={padL - 6} y={yT(t) + 3} textAnchor="end" fontSize={10} style={{ fill: C('color-text-secondary') }}>{fmtCompact(t)}</text>
        </g>
      ))}
      {hasCost && (
        <>
          {[0.5, 1].map(r => (
            <text key={r} x={W - padR + 8} y={yC(maxCost * r) + 3} fontSize={10} style={{ fill: '#f87171' }}>{fmtAxis(maxCost * r)}</text>
          ))}
          <path d={costLine} fill="none" stroke="#f87171" strokeWidth={1.6} strokeDasharray="4 3" />
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
            <rect x={x0} y={yIn} width={barW} height={Math.max(0, hIn)} fill="#60a5fa">
              <title>{`${d.label} · 输入 ${d.inputTokens} Token`}</title>
            </rect>
            <rect x={x0} y={yOut} width={barW} height={Math.max(0, hOut)} fill="#34d399">
              <title>{`${d.label} · 输出 ${d.outputTokens} Token`}</title>
            </rect>
            {hasCost && d.cost > 0 && (
              <circle cx={x(i)} cy={yC(d.cost)} r={2} fill="#f87171">
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

const ModelCenterPage: React.FC = () => {
  const { loggedIn, login, logout } = useAppStore()
  const [category, setCategory] = useState<Category>('llm')
  const [engines, setEngines] = useState<EngineConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [activeEngine, setActiveEngineState] = useState('xai')
  const [activeModel, setActiveModel] = useState('')
  const [testingEngine, setTestingEngine] = useState<string | null>(null)
  const [editingURLs, setEditingURLs] = useState<Record<string, string>>({})
  const [savingEngine, setSavingEngine] = useState<string | null>(null)
  const [engineStatuses, setEngineStatuses] = useState<Record<string, EngineStatus>>({})
  const [deepseekKey, setDeepseekKeyState] = useState('')
  const [deepseekKeyMasked, setDeepseekKeyMasked] = useState('')
  const [opencodeGoKey, setOpencodeGoKeyState] = useState('')
  const [opencodeGoKeyMasked, setOpencodeGoKeyMasked] = useState('')
  const [opencodeZenKey, setOpencodeZenKeyState] = useState('')
  const [opencodeZenKeyMasked, setOpencodeZenKeyMasked] = useState('')
  const [callStats, setCallStats] = useState<ModelStatsSummary | null>(null)
  const [statsSort, setStatsSort] = useState<StatsSort>('calls')
  const [trendRange, setTrendRange] = useState<TrendRange>('7d')
  const [loggingIn, setLoggingIn] = useState(false)
  const [imageBackend, setImageBackend] = useState('xai')
  const [comfyUIURL, setComfyUIURL] = useState('http://127.0.0.1:8188')
  const [imageSaveDir, setImageSaveDir] = useState('')
  const [imageModel, setImageModel] = useState('krea2')
  const [comfyUIPath, setComfyUIPath] = useState('')
  const [comfyUIPythonPath, setComfyUIPythonPath] = useState('')
  const [imageBackendSaving, setImageBackendSaving] = useState(false)
  const [comfyStatus, setComfyStatus] = useState<{ running: boolean; port: number }>({ running: false, port: 0 })
  const [comfyBusy, setComfyBusy] = useState(false)

  // 语音管道三段激活模型（STT/LLM/TTS，来自模型中心选择）
  const [voiceCfg, setVoiceCfg] = useState<{ stt: { engine: string; model: string }; llm: { engine: string; model: string }; tts: { engine: string; model: string; voice: string } }>({ stt: { engine: '', model: '' }, llm: { engine: '', model: '' }, tts: { engine: '', model: '', voice: '' } })
  // 功能绑定：聊天语音合成（优先于全局 TTS，便于后续扩展更多语音绑定）
  const [chatVoiceCfg, setChatVoiceCfg] = useState<{ engine: string; model: string }>({ engine: '', model: '' })
  const [chatVoiceDraft, setChatVoiceDraft] = useState<{ engine: string; model: string }>({ engine: '', model: '' })
  const [chatVoiceSaving, setChatVoiceSaving] = useState(false)
  const [chatVoiceSpeakers, setChatVoiceSpeakers] = useState<string[]>([])

  const loadAll = useCallback(async () => {
    try {
      const list = await getEngines(); setEngines(list)
      // 引擎最近连接状态缓存（后端随 engines.json 持久化，重启后仍可见）
      const st: Record<string, EngineStatus> = {}
      list.forEach(e => { if (e.status) st[e.id] = e.status })
      setEngineStatuses(st)
      const urls: Record<string, string> = {}
      list.forEach(e => { urls[e.id] = e.base_url || '' })
      try { const ae = await getActiveEngine(); if (ae) setActiveEngineState(ae) } catch (_) {}
      try {
        const am = await App.GetActiveModel()
        if (am) setActiveModel(String(am))
      } catch (_) {}
      try {
        const ks = await getDeepseekKeyStatus()
        if (ks) { setDeepseekKeyMasked(ks.masked || '') }
      } catch (_) {}
      try {
        const ks = await getOpencodeGoKeyStatus()
        if (ks) { setOpencodeGoKeyMasked(ks.masked || '') }
      } catch (_) {}
      try {
        const ks = await getOpencodeZenKeyStatus()
        if (ks) { setOpencodeZenKeyMasked(ks.masked || '') }
      } catch (_) {}
    } catch (_) {}
    finally { setLoading(false) }
  }, [])

  const loadCallStats = useCallback(async () => {
    try {
      const s = await getModelCallStats()
      if (s) setCallStats(s)
    } catch (_) {}
  }, [])

  // 调用统计页定时刷新
  useEffect(() => {
    if (category !== 'stats') return
    loadCallStats()
    const timer = window.setInterval(loadCallStats, 15000)
    return () => window.clearInterval(timer)
  }, [category, loadCallStats])

  const handleResetCallStats = async () => {
    try {
      await resetModelCallStats()
      setCallStats(null)
      message.success('模型调用统计已清空')
      loadCallStats()
    } catch (err: any) {
      message.error(err?.message || '重置失败')
    }
  }

  const loadImageBackend = useCallback(async () => {
    try {
      const cfg: any = await getImageBackendInfo()
      if (cfg?.backend) setImageBackend(cfg.backend)
      if (cfg?.image_model) setImageModel(cfg.image_model)
      if (cfg?.comfyui_url) setComfyUIURL(cfg.comfyui_url)
      if (cfg?.image_save_dir) setImageSaveDir(cfg.image_save_dir)
      if (cfg?.comfyui_path) setComfyUIPath(cfg.comfyui_path)
      if (cfg?.comfyui_python_path) setComfyUIPythonPath(cfg.comfyui_python_path)
    } catch (_) {}
    try {
      const st: any = await getComfyUIStatus()
      if (st) setComfyStatus({ running: !!st.running, port: st.port || 0 })
    } catch (_) {}
  }, [])

  const handleToggleComfy = async () => {
    setComfyBusy(true)
    try {
      if (comfyStatus.running) { await stopComfyUI(); setComfyStatus({ running: false, port: 0 }) }
      else { await startComfyUI(); setComfyStatus({ running: true, port: 8188 }) }
      message.success(comfyStatus.running ? 'ComfyUI 已停止' : 'ComfyUI 已启动')
    } catch (err: any) { message.error(err?.message || '操作失败') }
    finally { setComfyBusy(false) }
  }

  // 加载语音管道三段激活模型
  const loadVoiceCfg = useCallback(async () => {
    try {
      const cfg = await App.GetVoicePipelineConfig()
      if (cfg) {
        setVoiceCfg({
          stt: { engine: cfg.stt?.engine || '', model: cfg.stt?.model || '' },
          llm: { engine: cfg.llm?.engine || '', model: cfg.llm?.model || '' },
          tts: { engine: cfg.tts?.engine || '', model: cfg.tts?.model || '', voice: cfg.tts?.voice || '' },
        })
        setChatVoiceCfg({ engine: cfg.chatTts?.engine || '', model: cfg.chatTts?.model || '' })
        setChatVoiceDraft({ engine: cfg.chatTts?.engine || '', model: cfg.chatTts?.model || '' })
      }
    } catch (_) {}
  }, [])


  // ComfyUI 本地图片模型（本机硬编码：ComfyUI 非 LLM 引擎，模型不入引擎列表）
  const COMFY_IMAGES = [
    { modelId: 'krea2', modelName: 'Krea2 Turbo', engineId: 'comfyui', engineName: 'ComfyUI', status: 'running' },
    { modelId: 'z-image-turbo', modelName: 'Z-Image-Turbo', engineId: 'comfyui', engineName: 'ComfyUI', status: 'running' },
  ]

  // ── 功能模型绑定（聊天/小说/办公/角色库 各自独立 LLM，持久化重启不丢）──
  // 聊天已合并轻语（后端 whisper 为 chat 别名）；办公合并通用办公+方案编写（mergeKeys 一并写入）。
  const FEATURES: { key: string; label: string; icon: string; mergeKeys?: string[] }[] = [
    { key: 'chat', label: '聊天', icon: '💬' },
    { key: 'novel', label: '小说', icon: '📖' },
    { key: 'office', label: '办公', icon: '🛠️', mergeKeys: ['gaea'] },
    { key: 'characterlib', label: '角色库', icon: '🎭' },
  ]
  const [featureCfg, setFeatureCfg] = useState<Record<string, { engine: string; model: string }>>({})
  const [featureDraft, setFeatureDraft] = useState<Record<string, { engine: string; model: string }>>({})
  const [featureEnabled, setFeatureEnabled] = useState<Record<string, boolean>>({})
  const [modelRoutes, setModelRoutes] = useState<Record<string, { engine: string; model: string; source: string }>>({})
  const [portraitCfg, setPortraitCfg] = useState<{ backend: string; model: string }>({ backend: '', model: '' })
  const [portraitDraft, setPortraitDraft] = useState<{ backend: string; model: string }>({ backend: '', model: '' })
  const [portraitSaving, setPortraitSaving] = useState(false)

  // 当前生效路由（后端 routeModel 降级链结果：feature / global / fallback）
  const refreshRoutes = useCallback(async () => {
    const bind = (window as any).go?.app?.App
    if (!bind?.GetModelRoute) return
    const next: Record<string, { engine: string; model: string; source: string }> = {}
    for (const key of ['chat', 'novel', 'office', 'gaea', 'characterlib']) {
      try {
        next[key] = JSON.parse(await bind.GetModelRoute(key))
      } catch { /* 单功能失败忽略 */ }
    }
    setModelRoutes(next)
  }, [])
  useEffect(() => { refreshRoutes() }, [refreshRoutes])

  // 角色库剧照独立后端/模型选项（空 = 跟随绘梦）
  const portraitModelOptions = useMemo(() => {
    const b = portraitDraft.backend
    if (!b) return [{ label: '跟随绘梦', value: '' }]
    if (b === 'comfyui') {
      return [
        { label: '跟随绘梦', value: '' },
        { label: 'Krea2 Turbo', value: 'krea2' },
        { label: 'Z-Image-Turbo', value: 'z-image-turbo' },
      ]
    }
    const eng = engines.find(e => e.id === b)
    const imgs = (eng?.models || []).filter(m => classifyModel(m.id) === 'image')
    return [
      { label: '跟随绘梦', value: '' },
      ...imgs.map(m => ({ label: m.id, value: m.id })),
    ]
  }, [portraitDraft.backend, engines])

  // 趋势数据：后端按小时返回，按当前范围聚合为小时或天粒度。
  const trendData = useMemo<TrendDatum[]>(() => {
    if (!callStats?.trend?.length) return []
    const agg = new Map<string, TrendDatum>()
    for (const p of callStats.trend) {
      const hourly = trendRange === 'today'
      const key = hourly ? p.time : p.time.slice(0, 10)
      const label = hourly ? p.time.slice(5, 16).replace('T', ' ') : p.time.slice(5)
      const cur = agg.get(key)
      if (cur) {
        cur.calls += p.calls
        cur.successCalls += p.success_calls
        cur.failCalls += p.fail_calls
        cur.inputTokens += p.input_tokens
        cur.outputTokens += p.output_tokens
        cur.cost += p.cost
      } else {
        agg.set(key, {
          key,
          label,
          calls: p.calls,
          successCalls: p.success_calls,
          failCalls: p.fail_calls,
          inputTokens: p.input_tokens,
          outputTokens: p.output_tokens,
          cost: p.cost,
        })
      }
    }
    const list = Array.from(agg.values()).sort((a, b) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0))
    const limit = trendRange === 'today' ? 24 : trendRange === '7d' ? 7 : 30
    return list.slice(-limit)
  }, [callStats, trendRange])

  const loadFeatureCfg = useCallback(async () => {
    try {
      const cfg: Record<string, { engine: string; model: string }> = {}
      const en: Record<string, boolean> = {}
      for (const f of FEATURES) {
        const keys = [f.key, ...(f.mergeKeys || [])]
        let engine = ''
        let model = ''
        let on = true
        for (const k of keys) {
          const r: any = await App.GetFeatureModel(k)
          if (!engine && r?.engine) { engine = r.engine; model = r.model || '' }
          let e = true
          try { e = !!(await App.GetFeatureModelEnabled(k)) } catch (_) { e = true }
          on = on && e
        }
        cfg[f.key] = { engine, model }
        en[f.key] = on
      }
      setFeatureCfg(cfg)
      setFeatureEnabled(en)
      setFeatureDraft(JSON.parse(JSON.stringify(cfg)))
    } catch (_) {}
  }, [])

  // 同步：其他页面（FeatureModelBar 等）修改绑定后，本面板即时刷新
  useEffect(() => {
    const reload = () => { loadFeatureCfg(); refreshRoutes() }
    let unsub: any
    try {
      unsub = (window as any).runtime?.EventsOn?.('feature-model-changed', reload)
    } catch (_) {}
    return () => {
      try { if (typeof unsub === 'function') unsub() } catch (_) {}
    }
  }, [loadFeatureCfg, refreshRoutes])

  const handleSaveFeature = async (key: string) => {
    const d = featureDraft[key]
    if (!d?.engine || !d?.model) { message.warning('请先选择引擎和模型'); return }
    const f = FEATURES.find(x => x.key === key)
    try {
      await App.SetFeatureModel(key, d.engine, d.model)
      for (const k of f?.mergeKeys || []) {
        await App.SetFeatureModel(k, d.engine, d.model)
      }
      message.success(`${f?.label || key}模型已绑定并持久化`)
      loadFeatureCfg()
      refreshRoutes()
    } catch (err: any) {
      message.error(err?.message || '保存失败')
    }
  }

  const handleToggleFeatureEnabled = async (key: string, enabled: boolean) => {
    const f = FEATURES.find(x => x.key === key)
    try {
      await App.SetFeatureModelEnabled(key, enabled)
      for (const k of f?.mergeKeys || []) {
        await App.SetFeatureModelEnabled(k, enabled)
      }
      message.success(`${f?.label || key}功能模型已${enabled ? '启用' : '停用'}`)
      setFeatureEnabled(prev => ({ ...prev, [key]: enabled }))
      loadFeatureCfg()
    } catch (err: any) {
      message.error(err?.message || '操作失败')
    }
  }

  const handleSavePortrait = async () => {
    setPortraitSaving(true)
    try {
      await setPortraitConfig(portraitDraft.backend, portraitDraft.model)
      setPortraitCfg({ ...portraitDraft })
      message.success(
        portraitDraft.backend
          ? `角色库剧照已绑定：${portraitDraft.backend} / ${portraitDraft.model || '跟随绘梦'}`
          : '角色库剧照已恢复为跟随绘梦',
      )
    } catch (err: any) {
      message.error(err?.message || '保存失败')
    } finally {
      setPortraitSaving(false)
    }
  }

  useEffect(() => { loadAll(); loadImageBackend(); loadVoiceCfg(); loadFeatureCfg() }, [loadVoiceCfg, loadFeatureCfg])
  useEffect(() => {
    (async () => {
      const p = await getPortraitConfig()
      setPortraitCfg(p)
      setPortraitDraft(p)
    })()
  }, [])

  // 设为语音识别/合成（模型中心 → 语音管道）
  const handleSetVoiceModel = async (kind: 'asr' | 'tts', engineId: string, modelId: string) => {
    try {
      if (kind === 'asr') await App.SetActiveASRModel(engineId, modelId)
      else await App.SetActiveTTSModel(engineId, modelId)
      message.success(`已设为${kind === 'asr' ? '语音识别' : '语音合成'}：${modelId}`)
      loadVoiceCfg()
    } catch (err: any) {
      message.error(err?.message || '设置失败')
    }
  }

  // 保存功能绑定「聊天语音」（功能绑定 → 语音管道，空=清除绑定回退全局 TTS）
  const handleSaveChatVoice = async () => {
    const d = chatVoiceDraft
    if (!d.engine || !d.model) {
      message.warning('请选择引擎和语音模型（清除绑定请用右侧「清除」按钮）')
      return
    }
    setChatVoiceSaving(true)
    try {
      await App.SetChatVoiceModel(d.engine, d.model)
      message.success(`聊天语音已绑定：${d.model}`)
      loadVoiceCfg()
    } catch (err: any) {
      message.error(err?.message || '绑定失败')
    }
    setChatVoiceSaving(false)
  }

  // 清除功能绑定「聊天语音」
  const handleClearChatVoice = async () => {
    setChatVoiceSaving(true)
    try {
      await App.SetChatVoiceModel('', '')
      message.success('已清除聊天语音绑定（回退全局 TTS）')
      setChatVoiceDraft({ engine: '', model: '' })
      loadVoiceCfg()
    } catch (err: any) {
      message.error(err?.message || '清除失败')
    }
    setChatVoiceSaving(false)
  }

  // 聊天语音绑定卡：非 xAI 引擎时拉取服务端音色列表
  useEffect(() => {
    const { engine, model } = chatVoiceDraft
    if (!engine || !model || engine === 'xai') {
      setChatVoiceSpeakers([])
      return
    }
    ;(App as any).GetTTSSpeakers?.(model)
      .then((sp: any) => setChatVoiceSpeakers(Array.isArray(sp) ? sp : []))
      .catch(() => setChatVoiceSpeakers([]))
  }, [chatVoiceDraft])

  // 聊天语音绑定卡的音色选项：xAI → 固定列表；其他 → 服务端列表 / 兜底
  const chatVoiceOptions = useMemo(() => {
    const { engine, model } = chatVoiceDraft
    if (!engine || !model) return []
    if (engine === 'xai') return XAI_VOICES
    const list = chatVoiceSpeakers.length > 0
      ? chatVoiceSpeakers
      : localTTSFallbackVoices(model)
    return list.map(v => ({ value: v, label: v }))
  }, [chatVoiceDraft, chatVoiceSpeakers])

  const chatVoiceValue = useMemo(() => {
    const { engine, model } = chatVoiceDraft
    if (!engine || !model) return undefined
    const cur = voiceCfg.tts.voice || ''
    if (engine === 'xai') return XAI_VOICES.some(v => v.value === cur) ? cur : 'eve'
    const list = chatVoiceSpeakers.length > 0
      ? chatVoiceSpeakers
      : localTTSFallbackVoices(model)
    if (list.includes(cur)) return cur
    return localTTSDefaultVoice(model)
  }, [chatVoiceDraft, chatVoiceSpeakers, voiceCfg.tts.voice])

  const handleStartModel = async (card: ModelCardData) => {
    const kind = classifyModel(card.modelId)
    if (kind === 'tts') {
      // 本地 TTS 模型：启动对应本地服务（CosyVoice2 8010，幂等）
      try {
        const res = await App.StartLocalTTSService(card.engineId)
        if (res?.ready) message.success(`本地 TTS 服务已就绪：${card.engineName}`)
        else message.success(`正在启动本地 TTS 服务：${card.engineName}（首次约 1–2 分钟）`)
      } catch (err: any) { message.error(`启动失败：${err.message || err}`) }
      return
    }
    if (kind !== 'llm') return
    try {
      if (activeEngine !== card.engineId) { await setActiveEngine(card.engineId); setActiveEngineState(card.engineId) }
      await setEngineDefaultModel(card.engineId, card.modelId)
      setEngines(prev => prev.map(e => e.id === card.engineId ? { ...e, default_model: card.modelId } : e))
      setActiveModel(card.modelId)
      message.success(`已启动 ${card.modelName}`)
    } catch (err: any) { message.error(`启动失败：${err.message || err}`) }
  }

  const isModelActive = (card: ModelCardData) => activeEngine === card.engineId && activeModel === card.modelId

  const handleTestConnection = async (id: string) => {
    setTestingEngine(id)
    try {
      const st = await testEngineConnection(id)
      if (st) setEngineStatuses(prev => ({ ...prev, [id]: st }))
      await loadAll()
    } catch (err: any) { message.error(err.message) }
    finally { setTestingEngine(null) }
  }

  const handleRefreshModels = async (id: string) => {
    setTestingEngine(id)
    try { await refreshEngineModels(id); await loadAll() } catch (err: any) { message.error(err.message) }
    finally { setTestingEngine(null) }
  }

  const handleSaveURL = async (engine: EngineConfig) => {
    setSavingEngine(engine.id)
    try { await saveEngine({ id: engine.id, base_url: editingURLs[engine.id] || '', enabled: engine.enabled } as any); message.success('已保存') }
    catch (err: any) { message.error(err.message) }
    finally { setSavingEngine(null) }
  }

  const handleToggleEngine = async (engine: EngineConfig, enabled: boolean) => {
    try { await saveEngine({ id: engine.id, base_url: engine.base_url || '', enabled } as any); await loadAll() }
    catch (err: any) { message.error(err.message) }
  }

  const handleSaveImageBackend = async () => {
    setImageBackendSaving(true)
    try { await setImageBackendAPI(imageBackend, comfyUIURL, imageModel, imageSaveDir); message.success('已保存') }
    catch (err: any) { message.error(err.message) }
    finally { setImageBackendSaving(false) }
  }

  const handleSaveDeepseekKey = async () => {
    if (!deepseekKey.trim()) { message.warning('请输入 API Key'); return }
    try {
      await setDeepseekKey(deepseekKey.trim())
      message.success('DeepSeek Key 已保存')
      const ks = await getDeepseekKeyStatus()
      if (ks) setDeepseekKeyMasked(ks.masked || '')
      setDeepseekKeyState('')
    } catch (err: any) { message.error(err.message) }
  }

  const handleSaveOpencodeGoKey = async () => {
    if (!opencodeGoKey.trim()) { message.warning('请输入 API Key'); return }
    try {
      await setOpencodeGoKey(opencodeGoKey.trim())
      message.success('OpenCode Go Key 已保存')
      const ks = await getOpencodeGoKeyStatus()
      if (ks) setOpencodeGoKeyMasked(ks.masked || '')
      setOpencodeGoKeyState('')
    } catch (err: any) { message.error(err.message) }
  }

  const handleSaveOpencodeZenKey = async () => {
    if (!opencodeZenKey.trim()) { message.warning('请输入 API Key'); return }
    try {
      await setOpencodeZenKey(opencodeZenKey.trim())
      message.success('OpenCode Zen Key 已保存')
      const ks = await getOpencodeZenKeyStatus()
      if (ks) setOpencodeZenKeyMasked(ks.masked || '')
      setOpencodeZenKeyState('')
    } catch (err: any) { message.error(err.message) }
  }

  const makeModels = (engine: EngineConfig): ModelCardData[] =>
    (engine.models || []).map(m => ({ modelId: m.id, modelName: m.id, engineId: engine.id, engineName: engine.name, engineType: engine.type, engineEnabled: engine.enabled, status: m.status || 'running' }))

  const allModels = engines.filter(e => e.enabled).flatMap(e => makeModels(e))
  const llmModels = allModels.filter(m => classifyModel(m.modelId) === 'llm')
  const ttsModels = allModels.filter(m => classifyModel(m.modelId) === 'tts')
  const sttModels = allModels.filter(m => classifyModel(m.modelId) === 'stt')
  const imageModels = allModels.filter(m => classifyModel(m.modelId) === 'image')

  const sidebarBtn = (key: Category, icon: React.ReactNode, label: string) => (
    <Button type={category === key ? 'primary' : 'text'} icon={icon as any}
      onClick={() => setCategory(key)}
      style={{ justifyContent: 'flex-start', textAlign: 'left', borderRadius: 8, color: category === key ? '#fff' : C('color-text-secondary'), background: category === key ? C('color-primary') : 'transparent', fontWeight: category === key ? 500 : 400, padding: '8px 14px', height: 38 }}>
      {label}
    </Button>
  )

  if (loading) return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}><Spin size="large" /></div>

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Typography.Title level={4} style={{ color: C('color-text'), marginBottom: 16 }}>
        <ThunderboltOutlined style={{ marginRight: 8 }} />模型引擎中心
      </Typography.Title>
      <div style={{ flex: 1, display: 'flex', gap: 20, minHeight: 0 }}>
        <div style={{ width: 140, flexShrink: 0, display: 'flex', flexDirection: 'column', gap: 4 }}>
          {sidebarBtn('llm', <ThunderboltOutlined />, '语言模型')}
          {sidebarBtn('image', <PictureOutlined />, '图片生成')}
          {sidebarBtn('tts', <SoundOutlined />, '语音模型')}
          {sidebarBtn('engine', <SettingOutlined />, '引擎管理')}
          {sidebarBtn('bind', <LinkOutlined />, '功能绑定')}
          {sidebarBtn('stats', <ThunderboltOutlined />, '调用统计')}
        </div>

        <div style={{ flex: 1, overflow: 'auto', minWidth: 0 }}>
          {/* XAI 账户卡片 */}
          {category !== 'engine' && (
            <Card style={{ marginBottom: 20, background: loggedIn ? 'linear-gradient(135deg, rgba(52,211,153,0.06), rgba(16,185,129,0.03))' : 'linear-gradient(135deg, rgba(99,102,241,0.08), rgba(37,99,235,0.04))', border: loggedIn ? '1px solid rgba(52,211,153,0.25)' : '1px solid rgba(99,102,241,0.2)', borderRadius: 12 }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Space size={12}>
                  <div style={{ width: 36, height: 36, borderRadius: 10, background: loggedIn ? 'linear-gradient(135deg, #34d399, #10b981)' : 'linear-gradient(135deg, #6366f1, #2563eb)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    {loggedIn ? <CheckCircleOutlined style={{ fontSize: 18, color: '#fff' }} /> : <CloudOutlined style={{ fontSize: 18, color: '#fff' }} />}
                  </div>
                  <div>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>{loggedIn ? 'xAI 已连接' : 'xAI 账户'}</Typography.Text><br />
                    <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>{loggedIn ? 'Grok 模型已就绪' : '登录以使用云端模型'}</Typography.Text>
                  </div>
                </Space>
                {loggedIn ? <Button icon={<LogoutOutlined />} onClick={() => logout()} style={{ color: C('color-text-secondary'), fontSize: 12 }}>退出登录</Button>
                  : <Button type="primary" icon={<LoginOutlined />} loading={loggingIn}
                    onClick={async () => {
                      setLoggingIn(true)
                      try {
                        await login()
                        message.success('xAI 登录成功！')
                        await loadAll()
                      } catch (err: any) {
                        message.error('登录失败：' + (err?.message || err || '未知错误，请检查浏览器是否完成了 xAI 授权'))
                      } finally {
                        setLoggingIn(false)
                      }
                    }}
                    style={{ background: 'linear-gradient(135deg, #6366f1, #2563eb)', border: 'none', borderRadius: 8, fontWeight: 500 }}>登录 xAI</Button>}
              </div>
            </Card>
          )}

          {/* LLM */}
          {category === 'llm' && (
            <>
              {llmModels.length === 0 && (
                <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, textAlign: 'center', padding: 40, marginBottom: 16 }}>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14 }}>未发现语言模型。请在「引擎管理」中启用引擎并刷新模型。</Typography.Text>
                </Card>
              )}
              {engines.filter(e => e.enabled).map(engine => {
                const engineModels = llmModels.filter(m => m.engineId === engine.id)
                if (engineModels.length === 0) return null
                const color = engineColors[engine.id] || '#888'
                return (
                  <div key={engine.id} style={{ marginBottom: 24 }}>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, paddingBottom: 8, borderBottom: `1px solid ${color}30` }}>
                      <Space size={8}>
                        <span style={{ fontSize: 18, color }}>{engineIcons[engine.id]}</span>
                        <Typography.Text strong style={{ color: C('color-text'), fontSize: 15 }}>{engine.name}</Typography.Text>
                        <Tag color={color} style={{ fontSize: 10 }}>{engineLabels[engine.id]}</Tag>
                        <Tag style={{ fontSize: 10 }}>{engineModels.length} 个</Tag>
                        {engineStatuses[engine.id] && (
                          <Tag color={engineStatuses[engine.id].connected ? 'green' : 'red'} style={{ fontSize: 10 }}>
                            {engineStatuses[engine.id].connected ? '● 已连接' : '✗ 连接失败'}
                          </Tag>
                        )}
                      </Space>
                      <Space size={4}>
                        <Button size="small" onClick={() => handleTestConnection(engine.id)} loading={testingEngine === engine.id} style={{ fontSize: 11 }}>测试连接</Button>
                        <Button size="small" onClick={() => handleRefreshModels(engine.id)} loading={testingEngine === engine.id} style={{ fontSize: 11 }}>刷新模型</Button>
                      </Space>
                    </div>
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 10 }}>
                      {engineModels.map(card => {
                        const active = isModelActive(card)
                        return (
                          <Card key={card.modelId} size="small" style={{ background: active ? `linear-gradient(135deg, ${color}18, ${color}08)` : 'var(--bg-glass)', border: active ? `2px solid ${color}` : '1px solid var(--border-subtle)', borderRadius: 10 }}>
                            <Typography.Text strong style={{ color: active ? color : C('color-text'), fontSize: 13, display: 'block', marginBottom: 6, wordBreak: 'break-all' }}>{card.modelName}</Typography.Text>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                              <Tag color={color} style={{ fontSize: 10, margin: 0 }}>{engineLabels[card.engineId]}</Tag>
                              {active ? (
                                <Tag color="green" style={{ fontSize: 10, margin: 0 }}>● 运行中</Tag>
                              ) : (
                                <Tag color={card.status === 'stopped' ? 'default' : 'blue'} style={{ fontSize: 10, margin: 0 }}>{card.status === 'stopped' ? '○ 已停止' : '○ 就绪'}</Tag>
                              )}
                              <Button type={active ? 'default' : 'primary'} size="small" icon={active ? <CheckCircleOutlined /> : <CaretRightOutlined />} onClick={() => handleStartModel(card)} disabled={active} style={{ borderRadius: 8, fontSize: 11, marginLeft: 'auto' }}>{active ? '已启动' : '启动'}</Button>
                            </div>
                          </Card>
                        )
                      })}
                    </div>
                  </div>
                )
              })}
            </>
          )}

          {/* Image */}
          {category === 'bind' && (
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 14, flexWrap: 'wrap' }}>
                <LinkOutlined style={{ color: C('color-text-secondary') }} />
                <Typography.Text strong style={{ color: C('color-text'), fontSize: 15 }}>功能模型绑定</Typography.Text>
                <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
                  各功能板块独立模型，设置后持久化（重启不丢）
                </Typography.Text>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 14 }}>
                {FEATURES.map(f => {
                  const cur = featureCfg[f.key]
                  const draft = featureDraft[f.key] || { engine: '', model: '' }
                  const engineModels = draft.engine ? llmModels.filter(m => m.engineId === draft.engine) : []
                  const bound = !!cur?.engine && !!cur?.model
                  return (
                    <Card key={f.key} size="small" style={{ background: 'var(--bg-glass)', border: bound ? '1px solid rgba(34,197,94,0.35)' : '1px solid var(--border-subtle)', borderRadius: 12 }}>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, gap: 8, flexWrap: 'wrap' }}>
                        <Space size={6}>
                          <span style={{ fontSize: 16 }}>{f.icon}</span>
                          <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>{f.label}</Typography.Text>
                          {f.key === 'chat' && (
                            <>
                              <Tag color="purple" style={{ fontSize: 9, margin: 0 }}>TTS {chatVoiceCfg.model || voiceCfg.tts.model || '自动'}</Tag>
                              <Tag color="blue" style={{ fontSize: 9, margin: 0 }}>STT {voiceCfg.stt.model || '自动'}</Tag>
                            </>
                          )}
                          {f.key === 'office' && <Tag color="cyan" style={{ fontSize: 9, margin: 0 }}>通用 + 方案 + 知识库</Tag>}
                          {f.key === 'characterlib' && <Tag color="geekblue" style={{ fontSize: 9, margin: 0 }}>生成 / 补全</Tag>}
                        </Space>
                        <Tag color={bound ? 'green' : 'default'} style={{ fontSize: 10, margin: 0 }}>{bound ? '已绑定' : '未绑定'}</Tag>
                      </div>
                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 10, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {bound ? `当前：${cur!.engine} / ${cur!.model}` : '尚未绑定，选择引擎和模型后点绑定'}
                      </Typography.Text>
                      {modelRoutes[f.key] && (
                        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block' }}>
                          当前生效：{modelRoutes[f.key].engine || '-'} / {modelRoutes[f.key].model || '-'}（{modelRoutes[f.key].source || '-'}）
                        </Typography.Text>
                      )}
                      {f.key === 'office' && modelRoutes['gaea'] && (
                        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginTop: 2 }}>
                          通用办公 / 知识库路由：{modelRoutes['gaea'].engine || '-'} / {modelRoutes['gaea'].model || '-'}（{modelRoutes['gaea'].source || '-'}）
                        </Typography.Text>
                      )}
                      <div style={{ display: 'flex', gap: 8 }}>
                        <Select size="small" placeholder="引擎" value={draft.engine || undefined}
                          onChange={(v: string) => setFeatureDraft(p => ({ ...p, [f.key]: { engine: v, model: '' } }))}
                          style={{ flex: 1, minWidth: 0 }}
                          options={engines.filter(e => e.enabled).map(e => ({ value: e.id, label: engineLabels[e.id] || e.id }))} />
                        <Select size="small" placeholder="模型" value={draft.model || undefined}
                          onChange={(v: string) => setFeatureDraft(p => ({ ...p, [f.key]: { engine: p[f.key]?.engine || '', model: v } }))}
                          style={{ flex: 1, minWidth: 0 }}
                          options={engineModels.map(m => ({ value: m.modelId, label: m.modelName }))} />
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: 10, gap: 8 }}>
                        <span style={{ fontSize: 11, color: C('color-text-secondary') }}>功能启用（停用后回退全局模型）</span>
                        <Switch size="small" checked={featureEnabled[f.key] !== false} onChange={(v: boolean) => handleToggleFeatureEnabled(f.key, v)} />
                      </div>
                      <Button size="small" type={bound ? 'primary' : 'default'} block onClick={() => handleSaveFeature(f.key)} style={{ marginTop: 8, fontSize: 11 }}>
                        {bound ? '更新绑定' : '绑定'}
                      </Button>
                    </Card>
                  )
                })}
                {/* 功能绑定：聊天语音（优先于全局 TTS，模型列表随引擎自动刷新） */}
                <Card size="small" style={{ background: 'var(--bg-glass)', border: chatVoiceCfg.model ? '1px solid rgba(168,85,247,0.35)' : '1px dashed var(--border-subtle)', borderRadius: 12 }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, gap: 8, flexWrap: 'wrap' }}>
                    <Space size={6}>
                      <span style={{ fontSize: 16 }}>🗣️</span>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>聊天语音</Typography.Text>
                      <Tag color="purple" style={{ fontSize: 9, margin: 0 }}>TTS</Tag>
                      {chatVoiceCfg.model && <Tag color="geekblue" style={{ fontSize: 9, margin: 0 }}>功能绑定</Tag>}
                    </Space>
                    <Tag color={chatVoiceCfg.model ? 'green' : 'default'} style={{ fontSize: 10, margin: 0 }}>
                      {chatVoiceCfg.model ? '已绑定' : '未绑定'}
                    </Tag>
                  </div>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 8, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {chatVoiceCfg.model ? `当前：${chatVoiceCfg.engine} / ${chatVoiceCfg.model}` : '未绑定：语音对话使用全局 TTS（语音模型页）'}
                  </Typography.Text>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10, display: 'block', marginBottom: 8 }}>
                    优先于全局 TTS；列表随引擎模型自动刷新，后续新增语音模型无需改代码
                  </Typography.Text>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <Select size="small" placeholder="引擎" value={chatVoiceDraft.engine || undefined}
                      onChange={(v: string) => setChatVoiceDraft({ engine: v, model: '' })}
                      style={{ flex: 1, minWidth: 0 }}
                      options={engines.filter(e => e.enabled && ttsModels.some(m => m.engineId === e.id)).map(e => ({ value: e.id, label: engineLabels[e.id] || e.id }))} />
                    <Select size="small" placeholder="语音模型" value={chatVoiceDraft.model || undefined}
                      onChange={(v: string) => setChatVoiceDraft(p => ({ ...p, model: v }))}
                      style={{ flex: 1, minWidth: 0 }}
                      options={ttsModels.filter(m => m.engineId === chatVoiceDraft.engine).map(m => ({ value: m.modelId, label: m.modelName }))} />
                  </div>
                  {chatVoiceDraft.engine && chatVoiceDraft.model && chatVoiceOptions.length > 0 && (
                    <div style={{ display: 'flex', gap: 8, marginTop: 8, alignItems: 'center' }}>
                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, whiteSpace: 'nowrap' }}>音色</Typography.Text>
                      <Select size="small" value={chatVoiceValue} placeholder="选择音色"
                        onChange={async (v: string) => {
                          try {
                            await (App as any).VoiceApplySettings?.({ ttsVoice: v })
                            message.success('音色已更新：' + v)
                          } catch (err: any) {
                            message.error(err?.message || '音色更新失败')
                          }
                          setVoiceCfg(p => ({ ...p, tts: { ...p.tts, voice: v } }))
                        }}
                        style={{ flex: 1, minWidth: 0 }}
                        options={chatVoiceOptions} />
                    </div>
                  )}
                  <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
                    <Button size="small" type={chatVoiceCfg.model ? 'primary' : 'default'} block loading={chatVoiceSaving} onClick={handleSaveChatVoice} style={{ fontSize: 11 }}>
                      {chatVoiceCfg.model ? '更新绑定' : '绑定聊天语音'}
                    </Button>
                    {chatVoiceCfg.model && (
                      <Button size="small" danger onClick={handleClearChatVoice} loading={chatVoiceSaving} style={{ fontSize: 11 }}>清除</Button>
                    )}
                  </div>
                </Card>
                {/* 绘梦：自身界面选择 */}
                <Card size="small" style={{ background: 'rgba(255,255,255,0.02)', border: '1px dashed var(--border-subtle)', borderRadius: 12 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{ fontSize: 16 }}>🎨</span>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>绘梦</Typography.Text>
                  </div>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginTop: 8, lineHeight: 1.6 }}>
                    图片模型在绘梦界面内选择（后端 / 模型 / ComfyUI 启停），无需在此重复设置
                  </Typography.Text>
                </Card>

                {/* 角色库剧照：独立图片后端/模型（空 = 跟随绘梦） */}
                <Card size="small" style={{ background: portraitCfg.backend ? 'var(--bg-glass)' : 'rgba(255,255,255,0.02)', border: portraitCfg.backend ? '1px solid rgba(96,165,250,0.35)' : '1px dashed var(--border-subtle)', borderRadius: 12 }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, gap: 8, flexWrap: 'wrap' }}>
                    <Space size={6}>
                      <span style={{ fontSize: 16 }}>🖼️</span>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>角色库剧照</Typography.Text>
                      <Tag color="blue" style={{ fontSize: 9, margin: 0 }}>图片</Tag>
                    </Space>
                    <Tag color={portraitCfg.backend ? 'blue' : 'default'} style={{ fontSize: 10, margin: 0 }}>
                      {portraitCfg.backend ? `${portraitCfg.backend} / ${portraitCfg.model || '跟随绘梦'}` : '跟随绘梦'}
                    </Tag>
                  </div>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 10, lineHeight: 1.6 }}>
                    角色卡「生成剧照」使用独立后端；留空则跟随绘梦页当前选择
                  </Typography.Text>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <Select
                      size="small"
                      placeholder="后端"
                      value={portraitDraft.backend || undefined}
                      onChange={(v: string) => setPortraitDraft({ backend: v, model: '' })}
                      style={{ flex: 1, minWidth: 0 }}
                      options={[
                        { value: '', label: '跟随绘梦' },
                        { value: 'xai', label: 'xAI 云端' },
                        { value: 'comfyui', label: 'ComfyUI 本地' },
                        { value: 'herdsman', label: 'Herdsman 本地' },
                        { value: 'ollama', label: 'Ollama 本地' },
                      ]}
                    />
                    <Select
                      size="small"
                      placeholder="模型"
                      value={portraitDraft.model || undefined}
                      onChange={(v: string) => setPortraitDraft(p => ({ ...p, model: v }))}
                      style={{ flex: 1, minWidth: 0 }}
                      options={portraitModelOptions}
                    />
                  </div>
                  <Button size="small" type={portraitCfg.backend ? 'primary' : 'default'} block
                    loading={portraitSaving} onClick={handleSavePortrait}
                    style={{ marginTop: 10, fontSize: 11 }}>
                    {portraitCfg.backend ? '更新剧照绑定' : '绑定剧照后端'}
                  </Button>
                </Card>
              </div>
            </div>
          )}

          {/* 模型调用统计（独立标签页，参考 CCSwitch 的按提供商分组展示） */}
          {category === 'stats' && (
            <div>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14, flexWrap: 'wrap', gap: 8 }}>
                <Space size={8}>
                  <ThunderboltOutlined style={{ color: '#fbbf24' }} />
                  <Typography.Text strong style={{ color: C('color-text'), fontSize: 15 }}>模型调用统计</Typography.Text>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
                    {callStats?.since ? `统计自 ${callStats.since}` : '按引擎 / 模型维度统计调用情况与估算费用'}
                  </Typography.Text>
                </Space>
                <Space size={4}>
                  <Segmented
                    size="small"
                    value={statsSort}
                    onChange={(v) => setStatsSort(v as StatsSort)}
                    options={[
                      { value: 'calls', label: '调用最多' },
                      { value: 'tokens', label: 'Token 最多' },
                      { value: 'cost', label: '费用最高' },
                    ]}
                    style={{ fontSize: 11 }}
                  />
                  <Button size="small" icon={<ReloadOutlined />} onClick={loadCallStats} style={{ fontSize: 11 }}>刷新</Button>
                  <Button size="small" danger onClick={handleResetCallStats} style={{ fontSize: 11 }}>清空统计</Button>
                </Space>
              </div>

              {!callStats || callStats.total_calls === 0 ? (
                <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, textAlign: 'center', padding: 48 }}>
                  <ThunderboltOutlined style={{ fontSize: 30, color: C('color-text-secondary'), marginBottom: 12 }} />
                  <Typography.Text style={{ color: C('color-text'), fontSize: 14, display: 'block' }}>暂无调用记录</Typography.Text>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, display: 'block', marginTop: 6 }}>
                    对话、语音、办公等模块调用模型后，这里会自动统计次数、Token、耗时与估算费用
                  </Typography.Text>
                </Card>
              ) : (
                <>
                  {/* 全局指标 */}
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 10, marginBottom: 16 }}>
                    <div style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, padding: '12px 14px' }}>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginBottom: 4 }}>总调用</div>
                      <div style={{ color: C('color-text'), fontSize: 22, fontWeight: 700 }}>{callStats.total_calls}</div>
                      <div style={{ color: '#34d399', fontSize: 11, marginTop: 2 }}>成功 {callStats.success_calls} · 失败 {callStats.fail_calls}</div>
                    </div>
                    <div style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, padding: '12px 14px' }}>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginBottom: 4 }}>Token 用量</div>
                      <div style={{ color: C('color-text'), fontSize: 22, fontWeight: 700 }}>{callStats.total_tokens.toLocaleString()}</div>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 2 }}>入 {callStats.input_tokens.toLocaleString()} / 出 {callStats.output_tokens.toLocaleString()}</div>
                    </div>
                    <div style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, padding: '12px 14px' }}>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginBottom: 4 }}>估算费用</div>
                      <div style={{ color: '#fbbf24', fontSize: 22, fontWeight: 700 }}>{fmtCost(callStats.total_cost, 'CNY')}</div>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 2 }}>美元按 1:7.2 折算</div>
                    </div>
                    <div style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, padding: '12px 14px' }}>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginBottom: 4 }}>成功率</div>
                      <div style={{ color: C('color-text'), fontSize: 22, fontWeight: 700 }}>
                        {((callStats.success_calls / callStats.total_calls) * 100).toFixed(1)}%
                      </div>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 2 }}>{callStats.per_model.length} 个模型</div>
                    </div>
                    <div style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, padding: '12px 14px' }}>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginBottom: 4 }}>平均耗时</div>
                      <div style={{ color: C('color-text'), fontSize: 22, fontWeight: 700 }}>{(callStats.avg_duration_ms / 1000).toFixed(1)}s</div>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 2 }}>累计 {(callStats.total_duration_ms / 1000).toFixed(1)}s</div>
                    </div>
                  </div>

                  {/* 用量趋势（CCSwitch 风格：请求折线 + Token 堆叠 + 费用线） */}
                  <div style={{ marginBottom: 16 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10, flexWrap: 'wrap' }}>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>用量趋势</Typography.Text>
                      <Segmented
                        size="small"
                        value={trendRange}
                        onChange={(v) => setTrendRange(v as TrendRange)}
                        options={[
                          { value: 'today', label: '今日' },
                          { value: '7d', label: '最近 7 天' },
                          { value: '30d', label: '最近 30 天' },
                        ]}
                        style={{ fontSize: 11 }}
                      />
                    </div>
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: 12 }}>
                      <Card size="small" style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
                          <Typography.Text strong style={{ color: C('color-text'), fontSize: 12 }}>请求趋势</Typography.Text>
                          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10 }}>折线 = 调用 · 红点 = 失败</Typography.Text>
                        </div>
                        {trendData.length > 0 ? (
                          <RequestsTrendChart data={trendData} color="#60a5fa" />
                        ) : (
                          <div style={{ height: 200, display: 'flex', alignItems: 'center', justifyContent: 'center', color: C('color-text-secondary'), fontSize: 12 }}>暂无趋势数据</div>
                        )}
                      </Card>
                      <Card size="small" style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
                          <Typography.Text strong style={{ color: C('color-text'), fontSize: 12 }}>Token 趋势</Typography.Text>
                          <span style={{ display: 'flex', alignItems: 'center', gap: 10, fontSize: 10, color: C('color-text-secondary') }}>
                            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><i style={{ width: 8, height: 8, borderRadius: 2, background: '#60a5fa', display: 'inline-block' }} />输入</span>
                            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><i style={{ width: 8, height: 8, borderRadius: 2, background: '#34d399', display: 'inline-block' }} />输出</span>
                            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}><i style={{ width: 14, height: 0, borderTop: '2px dashed #f87171', display: 'inline-block' }} />费用</span>
                          </span>
                        </div>
                        {trendData.length > 0 ? (
                          <TokenTrendChart data={trendData} />
                        ) : (
                          <div style={{ height: 200, display: 'flex', alignItems: 'center', justifyContent: 'center', color: C('color-text-secondary'), fontSize: 12 }}>暂无趋势数据</div>
                        )}
                      </Card>
                    </div>
                  </div>

                  {/* 按引擎分组的模型用量（CCSwitch 风格：提供商卡片 + 模型行） */}
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                    {(() => {
                      const groups = new Map<string, typeof callStats.per_model>()
                      callStats.per_model.forEach(s => {
                        const list = groups.get(s.engine_id) || []
                        list.push(s)
                        groups.set(s.engine_id, list)
                      })
                      return Array.from(groups.entries()).map(([engineId, rows]) => {
                        const color = engineColors[engineId] || '#888'
                        const calls = rows.reduce((a, b) => a + b.call_count, 0)
                        const succ = rows.reduce((a, b) => a + b.success_count, 0)
                        const tokens = rows.reduce((a, b) => a + b.total_tokens, 0)
                        const engCost = rows.reduce((a, b) => a + costToCNY(b.estimated_cost ?? 0, b.currency), 0)
                        const rate = calls > 0 ? ((succ / calls) * 100).toFixed(0) : '0'
                        const sorted = [...rows].sort((a, b) => {
                          if (statsSort === 'tokens') return b.total_tokens - a.total_tokens
                          if (statsSort === 'cost') return (b.estimated_cost ?? 0) - (a.estimated_cost ?? 0)
                          return b.call_count - a.call_count
                        })
                        const maxCalls = Math.max(...sorted.map(s => s.call_count), 1)
                        return (
                          <Card key={engineId} size="small" style={{ background: 'var(--bg-glass)', border: `1px solid ${color}28`, borderRadius: 12, padding: 0 }}>
                            {/* 引擎汇总 */}
                            <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 14px', borderBottom: '1px solid var(--border-subtle)', background: `color-mix(in srgb, ${color} 8%, transparent)` }}>
                              <span style={{ fontSize: 18, color }}>{engineIcons[engineId]}</span>
                              <div style={{ minWidth: 0 }}>
                                <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block' }}>{engineLabels[engineId] || engineId}</Typography.Text>
                                <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10.5 }}>{rows.length} 个模型</Typography.Text>
                              </div>
                              <div style={{ marginLeft: 'auto', display: 'flex', gap: 18, alignItems: 'center' }}>
                                <div style={{ textAlign: 'right' }}>
                                  <div style={{ color: C('color-text'), fontSize: 15, fontWeight: 600 }}>{calls}</div>
                                  <div style={{ color: C('color-text-secondary'), fontSize: 10 }}>调用</div>
                                </div>
                                <div style={{ textAlign: 'right' }}>
                                  <div style={{ color: C('color-text'), fontSize: 15, fontWeight: 600 }}>{tokens.toLocaleString()}</div>
                                  <div style={{ color: C('color-text-secondary'), fontSize: 10 }}>Token</div>
                                </div>
                                <div style={{ textAlign: 'right' }}>
                                  <div style={{ color: '#fbbf24', fontSize: 15, fontWeight: 600 }}>{fmtCost(engCost, 'CNY')}</div>
                                  <div style={{ color: C('color-text-secondary'), fontSize: 10 }}>估算费用</div>
                                </div>
                                <div style={{ textAlign: 'right' }}>
                                  <div style={{ color: succ === calls ? '#34d399' : '#fb7185', fontSize: 15, fontWeight: 600 }}>{rate}%</div>
                                  <div style={{ color: C('color-text-secondary'), fontSize: 10 }}>成功率</div>
                                </div>
                              </div>
                            </div>
                            {/* 模型明细（对齐 CCSwitch 模型统计字段：模型 / 请求数 / 输入输出 Token / 平均延迟 / 估算费用 / 成功率） */}
                            <div style={{ display: 'flex', flexDirection: 'column' }}>
                              <div style={{ display: 'grid', gridTemplateColumns: 'minmax(140px, 1.6fr) 56px 128px 64px 92px 52px', gap: 8, padding: '6px 14px', borderBottom: '1px solid var(--border-subtle)', color: C('color-text-secondary'), fontSize: 10 }}>
                                <div>模型</div>
                                <div style={{ textAlign: 'right' }}>请求数</div>
                                <div style={{ textAlign: 'right' }}>输入 / 输出 Token</div>
                                <div style={{ textAlign: 'right' }}>平均延迟</div>
                                <div style={{ textAlign: 'right' }}>估算费用</div>
                                <div style={{ textAlign: 'right' }}>成功率</div>
                              </div>
                              {sorted.map(s => {
                                const r2 = s.call_count > 0 ? ((s.success_count / s.call_count) * 100).toFixed(0) : '0'
                                const share = Math.round((s.call_count / maxCalls) * 100)
                                const avgSec = s.call_count > 0 ? (s.total_duration_ms / s.call_count / 1000).toFixed(1) : '0.0'
                                const costText = fmtCost(s.estimated_cost, s.currency) || (isLocalEngine(s.engine_id) ? '免费' : '—')
                                return (
                                  <div key={s.engine_id + '|' + s.model} style={{ position: 'relative', display: 'grid', gridTemplateColumns: 'minmax(140px, 1.6fr) 56px 128px 64px 92px 52px', gap: 8, alignItems: 'center', padding: '8px 14px', borderBottom: '1px solid rgba(255,255,255,0.04)', overflow: 'hidden' }}>
                                    {/* 调用占比背景条 */}
                                    <div style={{ position: 'absolute', left: 0, top: 0, bottom: 0, width: `${share}%`, background: `color-mix(in srgb, ${color} 7%, transparent)`, pointerEvents: 'none' }} />
                                    <div style={{ position: 'relative', minWidth: 0 }}>
                                      <Typography.Text style={{ color: C('color-text'), fontSize: 12, display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                        {s.model}
                                        {s.last_error && <span style={{ color: '#fb7185', marginLeft: 6 }} title={s.last_error}>⚠</span>}
                                      </Typography.Text>
                                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10 }}>
                                        {s.last_called_at ? `最近 ${s.last_called_at.slice(5, 16)}` : '—'}
                                      </Typography.Text>
                                    </div>
                                    <div style={{ position: 'relative', textAlign: 'right' }}>
                                      <Typography.Text style={{ color: C('color-text'), fontSize: 12 }}>{s.call_count}</Typography.Text>
                                      {s.fail_count > 0 && (
                                        <Typography.Text style={{ color: '#fb7185', fontSize: 10, display: 'block' }}>失败 {s.fail_count}</Typography.Text>
                                      )}
                                    </div>
                                    <div style={{ position: 'relative', textAlign: 'right' }}>
                                      <Typography.Text style={{ color: C('color-text'), fontSize: 11, display: 'block' }}>入 {s.input_tokens.toLocaleString()}</Typography.Text>
                                      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10 }}>出 {s.output_tokens.toLocaleString()}</Typography.Text>
                                    </div>
                                    <div style={{ position: 'relative', textAlign: 'right' }}>
                                      <Typography.Text style={{ color: C('color-text'), fontSize: 12 }}>{avgSec}s</Typography.Text>
                                    </div>
                                    <div style={{ position: 'relative', textAlign: 'right' }}>
                                      <Typography.Text style={{ color: costText === '—' ? C('color-text-secondary') : '#fbbf24', fontSize: 12, fontWeight: 600 }}>{costText}</Typography.Text>
                                    </div>
                                    <div style={{ position: 'relative', textAlign: 'right' }}>
                                      <Typography.Text style={{ color: s.fail_count > 0 ? '#fb7185' : '#34d399', fontSize: 12, fontWeight: 600 }}>{r2}%</Typography.Text>
                                    </div>
                                  </div>
                                )
                              })}
                            </div>
                          </Card>
                        )
                      })
                    })()}
                  </div>
                </>
              )}
            </div>
          )}

          {category === 'image' && (
            <>
              {/* 图片后端 + ComfyUI + 模型 */}
              <Card style={{ marginBottom: 20, background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
                  {/* 后端选择 */}
                  <div>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 14, display: 'block', marginBottom: 8 }}>图片生成后端</Typography.Text>
                    <Segmented
                      value={imageBackend}
                      onChange={(v) => setImageBackend(v as string)}
                      options={[
                        { value: 'xai', label: '☁️ xAI 云端' },
                        { value: 'comfyui', label: '🏠 ComfyUI 本地' },
                        { value: 'herdsman', label: '🚀 Herdsman' },
                        { value: 'ollama', label: '🖥 Ollama' },
                      ]}
                    />
                  </div>

                  {/* ComfyUI 配置（本机默认已写死，仅展示 + 启停） */}
                  {imageBackend === 'comfyui' && (
                    <div style={{ padding: '12px 14px', borderRadius: 10, background: 'rgba(255,255,255,0.03)', border: '1px solid var(--border-subtle)' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                        <span style={{
                          width: 8, height: 8, borderRadius: '50%',
                          background: comfyStatus.running ? '#22c55e' : '#64748b',
                          boxShadow: comfyStatus.running ? '0 0 8px #22c55e' : 'none',
                        }} />
                        <Typography.Text style={{ color: C('color-text'), fontSize: 13, fontWeight: 600 }}>
                          ComfyUI {comfyStatus.running ? `运行中 (端口 ${comfyStatus.port || 8188})` : '未启动'}
                        </Typography.Text>
                        <Button size="small" type={comfyStatus.running ? 'default' : 'primary'} loading={comfyBusy}
                          onClick={handleToggleComfy} style={{ fontSize: 11 }}>
                          {comfyStatus.running ? '⏹ 停止' : '▶ 启动'}
                        </Button>
                      </div>
                      <div style={{ fontSize: 11, color: C('color-text-secondary'), marginTop: 8, lineHeight: 1.8 }}>
                        <div>URL：<span style={{ color: C('color-text') }}>{comfyUIURL}</span></div>
                        <div>启动位置：<span style={{ color: C('color-text') }}>{comfyUIPath || 'C:\\AI\\ComfyUI\\ComfyUI（默认）'}</span></div>
                        <div>Python：<span style={{ color: C('color-text') }}>{comfyUIPythonPath || 'C:\\AI\\ComfyUI\\standalone-env\\python.exe（默认）'}</span></div>
                      </div>
                    </div>
                  )}

                  {/* 图片模型选择 */}
                  <div>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 14, display: 'block', marginBottom: 8 }}>图片模型</Typography.Text>
                    <Select
                      size="middle" style={{ width: 320 }} value={imageModel}
                      onChange={setImageModel}
                      options={[
                        ...(imageBackend === 'comfyui' ? COMFY_IMAGES : []).map(m => ({ value: m.modelId, label: `${m.modelName}（ComfyUI 本地）` })),
                        ...imageModels.map(m => ({ value: m.modelId, label: m.modelName })),
                        ...(imageBackend !== 'comfyui' ? COMFY_IMAGES.map(m => ({ value: m.modelId, label: `${m.modelName}（ComfyUI 本地）` })) : []),
                        { value: 'grok-imagine-image-quality', label: 'Grok Imagine（xAI）' },
                      ]}
                    />
                  </div>

                  {/* 保存 */}
                  <div>
                    <Button type="primary" onClick={handleSaveImageBackend} loading={imageBackendSaving} style={{ borderRadius: 8 }}>
                      💾 保存图片后端设置
                    </Button>
                  </div>
                </div>
              </Card>

              {/* 发现的图片模型（含 ComfyUI 本地模型 krea2 / z-image-turbo） */}
              {(imageModels.length > 0 || imageBackend === 'comfyui') && (
                <div style={{ marginBottom: 24 }}>
                  <Typography.Text strong style={{ color: C('color-text'), fontSize: 15, display: 'block', marginBottom: 10 }}>发现的图片模型</Typography.Text>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 10 }}>
                    {[...COMFY_IMAGES, ...imageModels.filter(m => m.engineId !== 'comfyui')].map(m => (
                      <Card key={m.modelId} size="small" style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 10 }}>
                        <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 6, wordBreak: 'break-all' }}>{m.modelName}</Typography.Text>
                        <Space>
                          <Tag color={engineColors[m.engineId]} style={{ fontSize: 10, margin: 0 }}>{engineLabels[m.engineId]}</Tag>
                          <Tag color="orange" style={{ fontSize: 10, margin: 0 }}>图片</Tag>
                          <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10, margin: 0 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                        </Space>
                      </Card>
                    ))}
                  </div>
                </div>
              )}
              <Collapse ghost size="small" defaultActiveKey={['img-cfg']} items={[{
                key: 'img-cfg', label: <span style={{ color: C('color-text-secondary'), fontSize: 13 }}><SettingOutlined style={{ marginRight: 6 }} />图片存储</span>,
                children: (
                  <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <SettingField label="图片保存目录" value={imageSaveDir} onChange={v => setImageSaveDir(v)} placeholder="默认: Pictures/gaea" />
                      <Button type="primary" onClick={handleSaveImageBackend} loading={imageBackendSaving} style={{ borderRadius: 8 }}>💾 保存</Button>
                    </Space>
                  </Card>
                ),
              }]} />
            </>
          )}

          {/* TTS / Voice */}
          {category === 'tts' && (
            <>
              {/* 三段激活模型汇总（模型中心 → 语音管道） */}
              <Card style={{ marginBottom: 16, background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center', fontSize: 12 }}>
                  <span style={{ color: C('color-text-secondary'), fontWeight: 600, marginRight: 4 }}>语音管道：</span>
                  <Tag color={voiceCfg.stt.model ? 'blue' : 'default'} style={{ fontSize: 11 }}>
                    🎙️ 识别 {voiceCfg.stt.model || '自动'}
                  </Tag>
                  <Tag color={voiceCfg.llm.model ? 'green' : 'default'} style={{ fontSize: 11 }}>
                    💬 对话 {voiceCfg.llm.model || '默认'}
                  </Tag>
                  <Tag color={voiceCfg.tts.model ? 'purple' : 'default'} style={{ fontSize: 11 }}>
                    🔊 合成 {voiceCfg.tts.model || '自动'}
                  </Tag>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, marginLeft: 'auto' }}>
                    点击下方卡片可切换识别/合成模型（自动持久化，重启保留）
                  </Typography.Text>
                </div>
              </Card>

              {ttsModels.length === 0 && sttModels.length === 0 ? (
                <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, textAlign: 'center', padding: 40, marginBottom: 16 }}>
                  <SoundOutlined style={{ fontSize: 32, color: C('color-text-secondary'), marginBottom: 12 }} />
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14, display: 'block' }}>未发现语音模型</Typography.Text>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginTop: 6 }}>
                    请先在「引擎管理」中刷新模型列表（Herdsman 本地引擎可提供 whisper / qwen3-tts 等）
                  </Typography.Text>
                </Card>
              ) : (
                <>
                  {ttsModels.length > 0 && (
                    <div style={{ marginBottom: 24 }}>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 15, display: 'block', marginBottom: 10 }}>🔊 TTS 语音合成</Typography.Text>
                      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(230px, 1fr))', gap: 10 }}>
                        {ttsModels.map(m => {
                          const active = voiceCfg.tts.engine === m.engineId && voiceCfg.tts.model === m.modelId
                          return (
                            <Card key={m.modelId} size="small" style={{
                              background: 'var(--bg-glass)',
                              border: active ? '1px solid var(--md-sys-color-primary)' : '1px solid var(--border-subtle)',
                              borderRadius: 10,
                              boxShadow: active ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 30%, transparent)' : 'none',
                              transition: 'box-shadow 0.2s, border-color 0.2s',
                            }}>
                              <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 6 }}>{m.modelName}</Typography.Text>
                              <Space>
                                <Tag color={engineColors[m.engineId]} style={{ fontSize: 10 }}>{engineLabels[m.engineId]}</Tag>
                                <Tag color="purple" style={{ fontSize: 10 }}>TTS</Tag>
                                <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                              </Space>
                              {active && <Tag color="purple" style={{ marginTop: 6, fontSize: 10 }}>● 语音合成中</Tag>}
                              <div style={{ marginTop: 8 }}>
                                <Button size="small" type={active ? 'primary' : 'default'} icon={<SoundOutlined />}
                                  onClick={() => handleSetVoiceModel('tts', m.engineId, m.modelId)}
                                  style={{ fontSize: 11 }}>{active ? '已设为语音合成' : '设为语音合成'}</Button>
                              </div>
                            </Card>
                          )
                        })}
                      </div>
                    </div>
                  )}
                  {sttModels.length > 0 && (
                    <div style={{ marginBottom: 24 }}>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 15, display: 'block', marginBottom: 10 }}>🎙️ STT 语音识别</Typography.Text>
                      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(230px, 1fr))', gap: 10 }}>
                        {sttModels.map(m => {
                          const active = voiceCfg.stt.engine === m.engineId && voiceCfg.stt.model === m.modelId
                          return (
                            <Card key={m.modelId} size="small" style={{
                              background: 'var(--bg-glass)',
                              border: active ? '1px solid var(--md-sys-color-primary)' : '1px solid var(--border-subtle)',
                              borderRadius: 10,
                              boxShadow: active ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 30%, transparent)' : 'none',
                              transition: 'box-shadow 0.2s, border-color 0.2s',
                            }}>
                              <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 6 }}>{m.modelName}</Typography.Text>
                              <Space>
                                <Tag color={engineColors[m.engineId]} style={{ fontSize: 10 }}>{engineLabels[m.engineId]}</Tag>
                                <Tag color="blue" style={{ fontSize: 10 }}>STT</Tag>
                                <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                              </Space>
                              {active && <Tag color="blue" style={{ marginTop: 6, fontSize: 10 }}>● 语音识别中</Tag>}
                              <div style={{ marginTop: 8 }}>
                                <Button size="small" type={active ? 'primary' : 'default'} icon={<AudioOutlined />}
                                  onClick={() => handleSetVoiceModel('asr', m.engineId, m.modelId)}
                                  style={{ fontSize: 11 }}>{active ? '已设为语音识别' : '设为语音识别'}</Button>
                              </div>
                            </Card>
                          )
                        })}
                      </div>
                    </div>
                  )}
                </>
              )}
            </>
          )}

          {/* Engine Management */}
          {category === 'engine' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              {engines.map(engine => {
                const color = engineColors[engine.id] || '#888'
                const em = makeModels(engine)
                const mc = { llm: em.filter(m => classifyModel(m.modelId) === 'llm').length, tts: em.filter(m => classifyModel(m.modelId) === 'tts').length, stt: em.filter(m => classifyModel(m.modelId) === 'stt').length, image: em.filter(m => classifyModel(m.modelId) === 'image').length }
                return (
                  <Card key={engine.id} size="small" style={{ background: 'var(--bg-glass)', border: engine.enabled ? `1px solid ${color}30` : '1px solid var(--border-subtle)', borderRadius: 12 }}>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 }}>
                      <Space size={8}>
                        <span style={{ fontSize: 20, color }}>{engineIcons[engine.id]}</span>
                        <div>
                          <Typography.Text strong style={{ color: C('color-text'), fontSize: 14 }}>{engine.name}</Typography.Text>
                          <div style={{ marginTop: 2 }}>
                            <Tag color={color} style={{ fontSize: 10 }}>{engineLabels[engine.id]}</Tag>
                            <Switch size="small" checked={engine.enabled} onChange={(v) => handleToggleEngine(engine, v)} />
                          </div>
                        </div>
                      </Space>
                      <Space size={4}>
                        <Button size="small" onClick={() => handleTestConnection(engine.id)} loading={testingEngine === engine.id} disabled={!engine.enabled} style={{ fontSize: 11 }}>测试连接</Button>
                        <Button size="small" onClick={() => handleRefreshModels(engine.id)} loading={testingEngine === engine.id} disabled={!engine.enabled} style={{ fontSize: 11 }}>刷新</Button>
                      </Space>
                    </div>
                    <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
                      {mc.llm > 0 && <Tag style={{ fontSize: 10 }}>💬 {mc.llm} 语言</Tag>}
                      {mc.tts > 0 && <Tag color="purple" style={{ fontSize: 10 }}>🔊 {mc.tts} TTS</Tag>}
                      {mc.stt > 0 && <Tag color="blue" style={{ fontSize: 10 }}>🎙️ {mc.stt} STT</Tag>}
                      {mc.image > 0 && <Tag color="orange" style={{ fontSize: 10 }}>🖼️ {mc.image} 图片</Tag>}
                      {em.length === 0 && <Tag style={{ fontSize: 10 }}>暂无模型</Tag>}
                    </div>
                    {engine.type !== 'xai' && engine.type !== 'deepseek' && engine.type !== 'opencode-go' && engine.type !== 'opencode-zen' && (
                      <Space.Compact style={{ width: '100%' }}>
                        <Input size="small" value={editingURLs[engine.id] || ''} onChange={e => setEditingURLs(prev => ({ ...prev, [engine.id]: e.target.value }))} disabled={!engine.enabled} style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', color: C('color-text'), fontSize: 12 }} />
                        <Button size="small" onClick={() => handleSaveURL(engine)} loading={savingEngine === engine.id} disabled={!engine.enabled}>保存</Button>
                      </Space.Compact>
                    )}
                    {engine.type === 'deepseek' && (
                      <Space.Compact style={{ width: '100%' }}>
                        <Input size="small" value={deepseekKey} onChange={e => setDeepseekKeyState(e.target.value)} placeholder={deepseekKeyMasked || 'sk-...'} disabled={!engine.enabled} style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', color: C('color-text'), fontSize: 12 }} />
                        <Button size="small" onClick={handleSaveDeepseekKey} loading={savingEngine === engine.id} disabled={!engine.enabled}>保存 Key</Button>
                      </Space.Compact>
                    )}
                    {engine.type === 'opencode-go' && (
                      <Space.Compact style={{ width: '100%' }}>
                        <Input size="small" value={opencodeGoKey} onChange={e => setOpencodeGoKeyState(e.target.value)} placeholder={opencodeGoKeyMasked || 'oc-...（opencode.ai 订阅获取）'} disabled={!engine.enabled} style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', color: C('color-text'), fontSize: 12 }} />
                        <Button size="small" onClick={handleSaveOpencodeGoKey} loading={savingEngine === engine.id} disabled={!engine.enabled}>保存 Key</Button>
                      </Space.Compact>
                    )}
                    {engine.type === 'opencode-zen' && (
                      <Space.Compact style={{ width: '100%' }}>
                        <Input size="small" value={opencodeZenKey} onChange={e => setOpencodeZenKeyState(e.target.value)} placeholder={opencodeZenKeyMasked || 'zen-...（opencode.ai/auth 获取）'} disabled={!engine.enabled} style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', color: C('color-text'), fontSize: 12 }} />
                        <Button size="small" onClick={handleSaveOpencodeZenKey} loading={savingEngine === engine.id} disabled={!engine.enabled}>保存 Key</Button>
                      </Space.Compact>
                    )}
                    {engineStatuses[engine.id] && (
                      <div style={{ marginTop: 6, fontSize: 11 }}>
                        {engineStatuses[engine.id].connected
                          ? <span style={{ color: '#34d399' }}>✓ 已连接（{engineStatuses[engine.id].model_count} 个模型）</span>
                          : <span style={{ color: '#fb7185' }}>✗ {engineStatuses[engine.id].error}</span>}
                        {engineStatuses[engine.id].last_checked && (
                          <span style={{ color: 'var(--md-sys-color-text-secondary)', marginLeft: 8 }}>上次检查 {engineStatuses[engine.id].last_checked}</span>
                        )}
                      </div>
                    )}
                  </Card>
                )
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default ModelCenterPage
