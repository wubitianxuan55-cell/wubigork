import { createContext, useContext } from 'react'
import type { Dispatch, SetStateAction } from 'react'
import type { EngineConfig, EngineStatus, ModelStatsSummary } from '../../api/engines'
import type { StatsSort, TrendDatum, TrendRange } from './charts'
import type { Category, ModelCardData } from './utils'

export interface VoiceCfg {
  stt: { engine: string; model: string }
  llm: { engine: string; model: string }
  tts: { engine: string; model: string; voice: string }
}

export interface ModelCenterContextValue {
  category: Category
  setCategory: (c: Category) => void
  engines: EngineConfig[]
  engineStatuses: Record<string, EngineStatus>
  editingURLs: Record<string, string>
  setEditingURLs: Dispatch<SetStateAction<Record<string, string>>>
  savingEngine: string | null
  testingEngine: string | null
  activeEngine: string
  activeModel: string
  deepseekKey: string
  setDeepseekKeyState: (v: string) => void
  deepseekKeyMasked: string
  opencodeGoKey: string
  setOpencodeGoKeyState: (v: string) => void
  opencodeGoKeyMasked: string
  opencodeZenKey: string
  setOpencodeZenKeyState: (v: string) => void
  opencodeZenKeyMasked: string
  callStats: ModelStatsSummary | null
  statsSort: StatsSort
  setStatsSort: (v: StatsSort) => void
  trendRange: TrendRange
  setTrendRange: (v: TrendRange) => void
  trendData: TrendDatum[]
  imageBackend: string
  setImageBackend: (v: string) => void
  comfyUIURL: string
  comfyUIPath: string
  comfyUIPythonPath: string
  imageModel: string
  setImageModel: (v: string) => void
  imageSaveDir: string
  setImageSaveDir: (v: string) => void
  imageBackendSaving: boolean
  comfyStatus: { running: boolean; port: number }
  comfyBusy: boolean
  voiceCfg: VoiceCfg
  setVoiceCfg: Dispatch<SetStateAction<VoiceCfg>>
  ocrCfg: { engine: string; model: string }
  setOcrCfg: Dispatch<SetStateAction<{ engine: string; model: string }>>
  chatVoiceCfg: { engine: string; model: string }
  chatVoiceDraft: { engine: string; model: string }
  setChatVoiceDraft: Dispatch<SetStateAction<{ engine: string; model: string }>>
  chatVoiceSaving: boolean
  chatVoiceSpeakers: string[]
  chatVoiceOptions: { value: string; label: string }[]
  chatVoiceValue?: string
  featureCfg: Record<string, { engine: string; model: string }>
  featureDraft: Record<string, { engine: string; model: string }>
  setFeatureDraft: Dispatch<SetStateAction<Record<string, { engine: string; model: string }>>>
  featureEnabled: Record<string, boolean>
  modelRoutes: Record<string, { engine: string; model: string; source: string }>
  portraitCfg: { backend: string; model: string }
  portraitDraft: { backend: string; model: string }
  setPortraitDraft: Dispatch<SetStateAction<{ backend: string; model: string }>>
  portraitModelOptions: { label: string; value: string }[]
  portraitSaving: boolean
  llmModels: ModelCardData[]
  ttsModels: ModelCardData[]
  sttModels: ModelCardData[]
  imageModels: ModelCardData[]
  specialtyModels: ModelCardData[]
  makeModels: (engine: EngineConfig) => ModelCardData[]
  isModelActive: (card: ModelCardData) => boolean
  handleTestConnection: (id: string) => Promise<void>
  handleRefreshModels: (id: string) => Promise<void>
  handleStartModel: (card: ModelCardData) => Promise<void>
  handleSaveURL: (engine: EngineConfig) => Promise<void>
  handleToggleEngine: (engine: EngineConfig, enabled: boolean) => Promise<void>
  handleSaveDeepseekKey: () => Promise<void>
  handleSaveOpencodeGoKey: () => Promise<void>
  handleSaveOpencodeZenKey: () => Promise<void>
  handleResetCallStats: () => Promise<void>
  loadCallStats: () => Promise<void>
  handleToggleComfy: () => Promise<void>
  handleSaveImageBackend: () => Promise<void>
  handleSetVoiceModel: (kind: 'asr' | 'tts', engineId: string, modelId: string) => Promise<void>
  handleSetOCRModel: (engineId: string, modelId: string) => Promise<void>
  handleSaveFeature: (key: string) => Promise<void>
  handleToggleFeatureEnabled: (key: string, enabled: boolean) => Promise<void>
  handleSavePortrait: () => Promise<void>
  handleSaveChatVoice: () => Promise<void>
  handleClearChatVoice: () => Promise<void>
}

export const ModelCenterContext = createContext<ModelCenterContextValue | null>(null)

export function useModelCenter(): ModelCenterContextValue {
  const v = useContext(ModelCenterContext)
  if (!v) throw new Error('useModelCenter 必须在 ModelCenterPage 内使用')
  return v
}
