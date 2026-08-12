import type { ReactNode } from 'react'
import { CloudOutlined, DesktopOutlined, GlobalOutlined, KeyOutlined, RocketOutlined } from '@ant-design/icons'

export type Category = 'llm' | 'image' | 'tts' | 'engine' | 'bind' | 'stats'

export interface ModelCardData {
  modelId: string; modelName: string
  engineId: string; engineName: string
  engineType: string; engineEnabled: boolean
  status: string
  kind?: string
}

export type ModelKind = 'llm' | 'tts' | 'stt' | 'image'

export function classifyModel(id: string): ModelKind {
  const lid = id.toLowerCase()
  if (lid.includes('tts') || lid.includes('voice') || lid.includes('edge')) return 'tts'
  if (lid.includes('sherpa') || lid.includes('whisper') || lid.includes('zipformer') || lid.includes('asr')) return 'stt'
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

// 功能模型绑定（聊天/小说/办公/角色库，各自独立 LLM，持久化重启不丢）
export const FEATURES: { key: string; label: string; icon: string; mergeKeys?: string[] }[] = [
  { key: 'chat', label: '聊天', icon: '💬' },
  { key: 'novel', label: '小说', icon: '📖' },
  { key: 'office', label: '办公', icon: '🛠️', mergeKeys: ['gaea'] },
  { key: 'characterlib', label: '角色库', icon: '🎭' },
  { key: 'routine', label: '常规办公', icon: '⚙️' },
]

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
