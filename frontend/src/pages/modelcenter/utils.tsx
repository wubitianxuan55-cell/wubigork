import type { ReactNode } from 'react'
import { CloudOutlined, DesktopOutlined, GlobalOutlined, KeyOutlined, RocketOutlined } from '@ant-design/icons'
import type { EngineConfig, ModelInfo } from '../../api/engines'

export type Category = 'overview' | 'llm' | 'image' | 'tts' | 'specialty' | 'catalog' | 'engine' | 'bind' | 'stats' | 'benchmark' | 'retrieval'

/** 模型元数据（B 刀）：后端 ModelInfo 目录字段的展示子集，全可选 */
export interface ModelMeta {
  context_length?: number // 上下文窗口（tokens 绝对值）
  max_output?: number
  price_in?: number
  price_out?: number
  currency?: 'CNY' | 'USD'
  unit?: '' | 'call' | 'minute'
  free?: boolean
  caps?: string[]
  price_note?: string
  points_in?: number
  points_cached?: number
  points_out?: number
  points_peak?: number
}

export interface ModelCardData {
  modelId: string; modelName: string
  engineId: string; engineName: string
  engineType: string; engineEnabled: boolean
  status: string
  kind?: string
  /** B 刀：模型元数据（目录未下发/无任一字段时不构造，保持零破坏） */
  meta?: ModelMeta
}

export type ModelKind = 'llm' | 'tts' | 'stt' | 'image' | 'embedding' | 'rerank' | 'ocr'

export function classifyModel(id: string): ModelKind {
  const lid = id.toLowerCase()
  if (lid.includes('tts') || lid.includes('voice') || lid.includes('edge')) return 'tts'
  if (lid.includes('sherpa') || lid.includes('whisper') || lid.includes('zipformer') || lid.includes('asr') || lid.includes('funasr')) return 'stt'
  if (lid.includes('paddleocr') || lid.includes('ocr') || lid.includes('mineru')) return 'ocr'
  if (lid.includes('rerank')) return 'rerank'
  if (lid.includes('embedding') || lid.includes('bge-m3') || lid.includes('bge')) return 'embedding'
  if (lid.includes('image') || lid.includes('zimage') || lid.includes('flux') || lid.includes('cogview') || lid.includes('turbo') || lid.includes('sd') || lid.includes('dalle') || lid.includes('krea')) return 'image'
  return 'llm'
}

// GLM 官方双端点家族判定（与后端 GLMBaseURLStd/GLMBaseURLCoding 同源锚定
// docs.bigmodel.cn coding-plan/quick-start：std=按量付费，coding=套餐额度）。
export const glmEndpointFamily = (baseURL?: string): 'std' | 'coding' =>
  (baseURL || '').includes('/api/coding/') ? 'coding' : 'std'

// GLM 编码套餐别名注记：后端只在 coding 家族的 ModelInfo 上填 alias_of
// （套餐旧名由服务端自动切换到实际模型，std 家族为空）。无别名返回空串。
export const glmAliasNote = (model?: { id?: string; alias_of?: string }): string =>
  model?.alias_of
    ? `服务端自动切换：调用 ${model.id || '?'} 实际按 ${model.alias_of} 服务（编码套餐）`
    : ''

// 计费口径标签（stats 条目 billing_mode，后端取值 coding_points=编码套餐积分内
// 调用、不计价）；空/其他口径返回空串，调用方据此不渲染标签。
export const billingModeLabel = (mode?: string): string =>
  mode === 'coding_points' ? '编码套餐 · 积分口径（不计价）' : ''

// ── 模型元数据徽标（B 刀：目录字段 → 卡片徽标 / 积分口径展示） ──

// ModelInfo → ModelMeta 聚合：仅当任一展示字段存在才构造对象，
// 避免目录未下发时产生空 meta（卡片据此不渲染徽标、不占位）。
export function modelMeta(m?: ModelInfo): ModelMeta | undefined {
  if (!m) return undefined
  const meta: ModelMeta = {
    context_length: m.context_length, max_output: m.max_output,
    price_in: m.price_in, price_out: m.price_out,
    currency: m.currency, unit: m.unit, free: m.free,
    caps: m.caps, price_note: m.price_note,
    points_in: m.points_in, points_cached: m.points_cached,
    points_out: m.points_out, points_peak: m.points_peak,
  }
  const has = meta.context_length != null || meta.max_output != null
    || meta.price_in != null || meta.price_out != null
    || meta.currency != null || !!meta.unit || !!meta.free
    || (meta.caps?.length ?? 0) > 0 || !!meta.price_note
    || meta.points_in != null || meta.points_cached != null
    || meta.points_out != null || meta.points_peak != null
  return has ? meta : undefined
}

// 引擎列表定位模型的 meta（stats 逐行积分估算用：engine_id + model → 系数）
export function findModelMeta(engines: EngineConfig[], engineId: string, modelId: string): ModelMeta | undefined {
  const eng = engines.find(e => e.id === engineId)
  return modelMeta(eng?.models?.find(m => m.id === modelId))
}

// 上下文窗口格式化：≥1M 用 M（一位小数去尾 0），≥1K 用 K（整数）；空/非法返回空串。
export function formatCtx(tokens?: number): string {
  if (tokens == null || !Number.isFinite(tokens) || tokens <= 0) return ''
  if (tokens >= 1e6) return `${(tokens / 1e6).toFixed(1).replace(/\.0$/, '')}M`
  if (tokens >= 1e3) return `${Math.round(tokens / 1e3)}K`
  return String(Math.round(tokens))
}

// 价格数字：最多 2 位小数去尾 0（1.40→"1.4"、4.00→"4"、0.10→"0.1"）
const fmtPriceNum = (v: number): string => {
  if (!Number.isFinite(v)) return '0'
  return v.toFixed(2).replace(/\.?0+$/, '')
}

// 价格徽标文案：free→「免费」；unit=call/minute→「¥0.1/次」「¥0.18/分」；
// 默认每百万 tokens→「¥1.4/M」「$1.4·$4.4/M」（双侧）/「$4.4/M」（单侧）。
// 未计价（无 free 且无价格）返回空串，调用方不渲染。
export function formatPrice(meta?: ModelMeta): string {
  if (!meta) return ''
  if (meta.free) return '免费'
  const sym = meta.currency === 'USD' ? '$' : '¥'
  if (meta.unit === 'call') return meta.price_in == null ? '' : `${sym}${fmtPriceNum(meta.price_in)}/次`
  if (meta.unit === 'minute') return meta.price_in == null ? '' : `${sym}${fmtPriceNum(meta.price_in)}/分`
  const hasIn = meta.price_in != null
  const hasOut = meta.price_out != null
  if (!hasIn && !hasOut) return ''
  if (hasIn && hasOut) return `${sym}${fmtPriceNum(meta.price_in!)}·${sym}${fmtPriceNum(meta.price_out!)}/M`
  const single = (hasIn ? meta.price_in : meta.price_out)!
  return `${sym}${fmtPriceNum(single)}/M`
}

// 能力标签（caps）中文映射；未收录的键由调用方回退原文透传
export const capLabels: Record<string, string> = {
  vision: '视觉', tools: '工具', reasoning: '推理', search: '搜索', json: '结构化',
}

// 积分估算（GLM coding 套餐口径，与后端同式）：
// 积分 = (输入tokens×points_in + 缓存命中tokens×points_cached + 输出tokens×points_out) / 10000
export function estimatePoints(
  inputTokens: number,
  cachedTokens: number,
  outputTokens: number,
  coef: Pick<ModelMeta, 'points_in' | 'points_cached' | 'points_out'>,
): number {
  return (
    inputTokens * (coef.points_in ?? 0)
    + cachedTokens * (coef.points_cached ?? 0)
    + outputTokens * (coef.points_out ?? 0)
  ) / 10000
}

// 是否具备可估算的积分系数（仅 points_peak 不算——缺 in/cached/out 任一主系数无法估）
export const hasPointsCoef = (meta?: ModelMeta): boolean =>
  !!meta && (meta.points_in != null || meta.points_cached != null || meta.points_out != null)

export const engineIcons: Record<string, ReactNode> = {
  xai: <CloudOutlined />, ollama: <DesktopOutlined />, herdsman: <RocketOutlined />, deepseek: <KeyOutlined />, glm: <KeyOutlined />, cosyvoice: <RocketOutlined />, 'opencode-go': <GlobalOutlined />, 'opencode-zen': <GlobalOutlined />, custom: <GlobalOutlined />,
}
export const engineColors: Record<string, string> = {
  xai: '#60a5fa', ollama: '#f59e0b', herdsman: '#84cc16', deepseek: '#8b5cf6', glm: '#38bdf8', cosyvoice: '#f472b6', 'opencode-go': '#22d3ee', 'opencode-zen': '#a78bfa', custom: '#94a3b8', // hex-exempt 引擎品牌识别色（模型中心身份色板）
}
export const engineLabels: Record<string, string> = {
  xai: 'xAI 云端', ollama: 'Ollama 本地', herdsman: 'Herdsman 本地', deepseek: 'DeepSeek 云端', glm: 'GLM 云端', cosyvoice: 'CosyVoice2 本地', 'opencode-go': 'OpenCode Go 云端', 'opencode-zen': 'OpenCode Zen 云端', custom: '自定义 OpenAI 兼容',
}

// ── 自定义引擎（A 刀：OpenAI 兼容自定义服务商，type=custom / id=custom-*） ──

// custom 引擎判定：优先 type，兜底 id 前缀（后端契约 id 以 custom- 开头）。
export const isCustomEngine = (e: { id?: string; type?: string }): boolean =>
  e.type === 'custom' || (e.id || '').startsWith('custom-')

// 与后端 validBaseURL 同口径（http:// 或 https:// 前缀）：表单校验双保险，
// 防 API Key 被粘进地址框（v4.9.1 防线在自定义引擎上的延伸）。
export const isValidBaseURL = (url: string): boolean =>
  url.startsWith('http://') || url.startsWith('https://')

// 引擎展示元数据：优先使用后端下发（label/color），未下发时回退本地映射。
// custom 引擎 id 动态（custom-*），本地映射按 type 兜底到 custom 专用条目。
export const engineLabel = (e: { id?: string; engineId?: string; label?: string; type?: string }) =>
  e.label || engineLabels[e.id || e.engineId || ''] || (e.type === 'custom' ? engineLabels.custom : '') || e.id || e.engineId || ''
export const engineColor = (e: { id?: string; engineId?: string; color?: string; type?: string }) =>
  e.color || engineColors[e.id || e.engineId || ''] || (e.type === 'custom' ? engineColors.custom : '') || 'var(--color-text-secondary)'
// 引擎图标：按 id 命中内置映射；custom-* 动态 id 未命中时按 type 兜底。
export const engineIcon = (e: { id?: string; type?: string }): ReactNode | undefined =>
  engineIcons[e.id || ''] ?? (isCustomEngine(e) ? engineIcons.custom : undefined)
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

// 指定引擎的模型下拉选项；当前模型不在候选中时兜底补一条，避免表单值悬空。
export function modelOptionsForEngine(
  engineId: string,
  models: ModelCardData[],
  currentModel = '',
): { value: string; label: string }[] {
  const base = models.filter(m => m.engineId === engineId).map(m => ({ value: m.modelId, label: m.modelName }))
  if (currentModel && !base.some(o => o.value === currentModel)) {
    base.push({ value: currentModel, label: currentModel })
  }
  return base
}

// 引擎列表过滤：隐藏不常用的已停用引擎，减少管理页视觉噪音。
export function filterEnginesByEnabled(engines: EngineConfig[], onlyEnabled: boolean): EngineConfig[] {
  return onlyEnabled ? engines.filter(e => e.enabled) : engines
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
