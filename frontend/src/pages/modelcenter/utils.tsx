import type { ReactNode } from 'react'
import { CloudOutlined, DesktopOutlined, GlobalOutlined, KeyOutlined, RocketOutlined } from '@ant-design/icons'
import type { EngineConfig } from '../../api/engines'

export type Category = 'overview' | 'llm' | 'image' | 'tts' | 'specialty' | 'engine' | 'bind' | 'stats'

export interface ModelCardData {
  modelId: string; modelName: string
  engineId: string; engineName: string
  engineType: string; engineEnabled: boolean
  status: string
  kind?: string
}

export type ModelKind = 'llm' | 'tts' | 'stt' | 'image' | 'embedding' | 'rerank' | 'ocr'

export function classifyModel(id: string): ModelKind {
  const lid = id.toLowerCase()
  if (lid.includes('tts') || lid.includes('voice') || lid.includes('edge')) return 'tts'
  if (lid.includes('sherpa') || lid.includes('whisper') || lid.includes('zipformer') || lid.includes('asr') || lid.includes('funasr')) return 'stt'
  if (lid.includes('paddleocr') || lid.includes('ocr') || lid.includes('mineru')) return 'ocr'
  if (lid.includes('rerank')) return 'rerank'
  if (lid.includes('embedding') || lid.includes('bge-m3') || lid.includes('bge')) return 'embedding'
  if (lid.includes('image') || lid.includes('zimage') || lid.includes('flux') || lid.includes('turbo') || lid.includes('sd') || lid.includes('dalle') || lid.includes('krea')) return 'image'
  return 'llm'
}

export const engineIcons: Record<string, ReactNode> = {
  xai: <CloudOutlined />, ollama: <DesktopOutlined />, herdsman: <RocketOutlined />, deepseek: <KeyOutlined />, cosyvoice: <RocketOutlined />, 'opencode-go': <GlobalOutlined />, 'opencode-zen': <GlobalOutlined />,
}
export const engineColors: Record<string, string> = {
  xai: '#60a5fa', ollama: '#f59e0b', herdsman: '#84cc16', deepseek: '#8b5cf6', cosyvoice: '#f472b6', 'opencode-go': '#22d3ee', 'opencode-zen': '#a78bfa',
}
export const engineLabels: Record<string, string> = {
  xai: 'xAI 云端', ollama: 'Ollama 本地', herdsman: 'Herdsman 本地', deepseek: 'DeepSeek 云端', cosyvoice: 'CosyVoice2 本地', 'opencode-go': 'OpenCode Go 云端', 'opencode-zen': 'OpenCode Zen 云端',
}

// 引擎展示元数据：优先使用后端下发（label/color），未下发时回退本地映射。
export const engineLabel = (e: { id?: string; engineId?: string; label?: string }) => e.label || engineLabels[e.id || e.engineId || ''] || e.id || e.engineId || ''
export const engineColor = (e: { id?: string; engineId?: string; color?: string }) => e.color || engineColors[e.id || e.engineId || ''] || '#888'
// 模型分类：优先使用后端 kind，缺失时回退旧名称启发式。
export const kindOf = (m: ModelCardData): ModelKind => (m.kind as ModelKind) || classifyModel(m.modelId)

// ── 模型可用性（引擎状态 → 模型可见性联动） ──────────────────

export type ModelAvailability = 'ready' | 'stopped' | 'disconnected' | 'disabled'

// 引擎禁用/未连接/模型已停止时，前端据此置灰卡片或禁用动作，避免误选不可用模型。
// connected === undefined 视为「尚未检测」，保持可用（乐观），不误伤首次进入的用户。
export function modelAvailability(card: ModelCardData, enabled: boolean, connected?: boolean): ModelAvailability {
  if (!enabled) return 'disabled'
  if (connected === false) return 'disconnected'
  if (card.status === 'stopped') return 'stopped'
  return 'ready'
}

// 模型网格搜索（按名称或模型 ID 匹配，大小写不敏感）
export function filterModelsBySearch<T extends ModelCardData>(models: T[], query: string): T[] {
  const q = query.trim().toLowerCase()
  if (!q) return models
  return models.filter(m => m.modelName.toLowerCase().includes(q) || m.modelId.toLowerCase().includes(q))
}

// 收藏/置顶模型排到最前（其余保持原顺序）
export function sortModelsPinnedFirst<T extends ModelCardData>(models: T[], pinnedIds: string[]): T[] {
  const pinned = new Set(pinnedIds)
  return [...models].sort((a, b) => (pinned.has(b.modelId) ? 1 : 0) - (pinned.has(a.modelId) ? 1 : 0))
}

// 引擎模型是否图片类（优先后端 kind，缺失时回退名称启发式）
export const isImageModel = (m: { id: string; kind?: string }): boolean =>
  ((m.kind as ModelKind) || classifyModel(m.id)) === 'image'

// ComfyUI 本地出图模型（无需依赖引擎模型列表，恒可用）
export const COMFY_IMAGE_MODELS: ModelCardData[] = [
  { modelId: 'krea2', modelName: 'Krea2 Turbo', engineId: 'comfyui', engineName: 'ComfyUI', engineType: 'comfyui', engineEnabled: true, status: 'running', kind: 'image' },
  { modelId: 'z-image-turbo', modelName: 'Z-Image-Turbo', engineId: 'comfyui', engineName: 'ComfyUI', engineType: 'comfyui', engineEnabled: true, status: 'running', kind: 'image' },
]

// 图片模型下拉选项：严格按后端过滤，避免「选了 xAI 却列出 krea2」的错配。
// currentModel 不在候选中时兜底补一条，防止切换后端后表单值悬空。
export function imageModelOptionsFor(backend: string, engines: EngineConfig[], currentModel = ''): { value: string; label: string }[] {
  if (backend === 'comfyui') {
    const base = COMFY_IMAGE_MODELS.map(m => ({ value: m.modelId, label: `${m.modelName}（ComfyUI 本地）` }))
    if (currentModel && !base.some(o => o.value === currentModel)) {
      base.push({ value: currentModel, label: `${currentModel}（ComfyUI 本地）` })
    }
    return base
  }
  if (backend === 'xai') {
    const base = [
      { value: 'grok-imagine-image-quality', label: 'Grok Imagine 高质量（xAI）' },
      { value: 'grok-imagine-image', label: 'Grok Imagine（xAI）' },
    ]
    return base
  }
  const eng = engines.find(e => e.id === backend)
  const imgs = (eng?.models || []).filter(isImageModel)
  const base = imgs.map(m => ({ value: m.id, label: m.id }))
  if (base.length === 0 && currentModel) {
    base.push({ value: currentModel, label: currentModel })
  }
  return base
}

// 切换后端时建议的默认图片模型：xAI/ComfyUI 恒有内置；引擎取第一个图片模型。
export function imageModelDefaultFor(backend: string, engines: EngineConfig[]): string {
  if (backend === 'xai') return 'grok-imagine-image-quality'
  if (backend === 'comfyui') return 'krea2'
  const eng = engines.find(e => e.id === backend)
  const first = (eng?.models || []).find(isImageModel)
  return first?.id || ''
}

// 功能模型绑定（聊天/小说/办公/角色库，各自独立 LLM，持久化重启不丢）
export const FEATURES: { key: string; label: string; icon: string; mergeKeys?: string[] }[] = [
  { key: 'chat', label: '聊天', icon: '💬' },
  { key: 'novel', label: '小说', icon: '📖' },
  { key: 'office', label: '办公', icon: '🛠️', mergeKeys: ['gaea'] },
  { key: 'characterlib', label: '角色库', icon: '🎭' },
  { key: 'routine', label: '常规办公', icon: '⚙️' },
]

// ── 功能绑定状态（绑定 + 启停 → 明确回退态） ─────────────────

export type FeatureState = 'bound-active' | 'bound-disabled' | 'fallback'

export function featureState(bound: boolean, enabled: boolean): FeatureState {
  if (bound && enabled) return 'bound-active'
  if (bound && !enabled) return 'bound-disabled'
  return 'fallback'
}

export const featureStateMeta: Record<FeatureState, { label: string; color: string }> = {
  'bound-active': { label: '已绑定生效', color: 'green' },
  'bound-disabled': { label: '已绑定·停用（回退全局）', color: 'orange' },
  'fallback': { label: '跟随全局默认', color: 'default' },
}

// model.route 的 source → 友好文案
export const routeSourceLabel = (source?: string): string =>
  source === 'feature' ? '功能绑定' : source === 'global' ? '全局默认' : source === 'fallback' ? '兜底' : (source || '-')

// xAI Grok TTS 音色（与设置面板一致，模型中心绑定卡内可直接选择）
export const XAI_VOICES = [
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
export const COSYVOICE_FALLBACK_VOICES = ['中文女', '中文男', '英文女', '英文男']

// ── 费用展示工具（汇率单一来源：优先取后端下发的 usd_to_cny） ───
export const USD_TO_CNY = 7.2
export const isLocalEngine = (id: string) => id === 'ollama' || id === 'herdsman' || id === 'cosyvoice'
export const costToCNY = (cost: number, currency?: string, rate: number = USD_TO_CNY) => (currency === 'USD' ? cost * rate : cost)

// 本地 TTS 引擎的服务端音色兜底：CosyVoice2 4 个内置音色（中文女/男、英文女/男）
export const localTTSFallbackVoices = (model: string): string[] => {
  const l = model.toLowerCase()
  if (l.includes('cosyvoice')) return COSYVOICE_FALLBACK_VOICES
  return []
}
export const localTTSDefaultVoice = (model: string): string | undefined => {
  const l = model.toLowerCase()
  return l.includes('cosyvoice') ? '中文女' : undefined
}
export const fmtCost = (cost?: number, currency?: string): string => {
  const c = cost ?? 0
  if (currency === 'USD') return `$${c.toFixed(2)}`
  if (currency === 'CNY') return `¥${c.toFixed(2)}`
  return ''
}
