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

export interface ModelCenterStateValue {
  category: Category
  engines: EngineConfig[]
  engineStatuses: Record<string, EngineStatus>
  editingURLs: Record<string, string>
  savingEngine: string | null
  testingEngine: string | null
  activeEngine: string
  activeModel: string
  deepseekKey: string
  deepseekKeyMasked: string
  glmKey: string
  glmKeyMasked: string
  opencodeGoKey: string
  opencodeGoKeyMasked: string
  opencodeZenKey: string
  opencodeZenKeyMasked: string
  modelHubKey: string
  modelHubKeyMasked: string
  settingGlmEndpoint: boolean
  /** MH2：正在 Studio 侧加载的 modelhub 模型 id（模型卡显示「加载中」并禁用按钮） */
  hubLoadingIds: string[]
  callStats: ModelStatsSummary | null
  loadError: string | null
  statsSort: StatsSort
  trendRange: TrendRange
  trendData: TrendDatum[]
  imageBackend: string
  comfyUIURL: string
  comfyUIPath: string
  comfyUIPythonPath: string
  imageModel: string
  imageSaveDir: string
  imageBackendSaving: boolean
  comfyStatus: { running: boolean; port: number }
  comfyBusy: boolean
  voiceCfg: VoiceCfg
  ocrCfg: { engine: string; model: string }
  chatVoiceCfg: { engine: string; model: string }
  chatVoiceDraft: { engine: string; model: string }
  chatVoiceSaving: boolean
  chatVoiceSpeakers: string[]
  chatVoiceOptions: { value: string; label: string }[]
  chatVoiceValue?: string
  featureCfg: Record<string, { engine: string; model: string }>
  featureDraft: Record<string, { engine: string; model: string }>
  featureEnabled: Record<string, boolean>
  modelRoutes: Record<string, { engine: string; model: string; source: string }>
  portraitCfg: { backend: string; model: string }
  portraitDraft: { backend: string; model: string }
  portraitModelOptions: { label: string; value: string }[]
  portraitSaving: boolean
  llmModels: ModelCardData[]
  ttsModels: ModelCardData[]
  sttModels: ModelCardData[]
  imageModels: ModelCardData[]
  specialtyModels: ModelCardData[]
}

/** 动作通道（v4.352 拆分）：五个 hook 返回的函数引用随各 hook 重渲染换新——
 *  单独一个通道承载，state 消费者不因动作引用变化重渲染。 */
export interface ModelCenterActionsValue {
  setCategory: (c: Category) => void
  setEditingURLs: Dispatch<SetStateAction<Record<string, string>>>
  setDeepseekKeyState: (v: string) => void
  setGlmKeyState: (v: string) => void
  setOpencodeGoKeyState: (v: string) => void
  setOpencodeZenKeyState: (v: string) => void
  setModelHubKeyState: (v: string) => void
  setStatsSort: (v: StatsSort) => void
  setTrendRange: (v: TrendRange) => void
  setImageBackend: (v: string) => void
  setImageModel: (v: string) => void
  setImageSaveDir: (v: string) => void
  setVoiceCfg: Dispatch<SetStateAction<VoiceCfg>>
  setOcrCfg: Dispatch<SetStateAction<{ engine: string; model: string }>>
  setChatVoiceDraft: Dispatch<SetStateAction<{ engine: string; model: string }>>
  setFeatureDraft: Dispatch<SetStateAction<Record<string, { engine: string; model: string }>>>
  setPortraitDraft: Dispatch<SetStateAction<{ backend: string; model: string }>>
  makeModels: (engine: EngineConfig) => ModelCardData[]
  isModelActive: (card: ModelCardData) => boolean
  handleTestConnection: (id: string) => Promise<void>
  handleRefreshModels: (id: string) => Promise<void>
  handleStartModel: (card: ModelCardData) => Promise<void>
  handleSaveURL: (engine: EngineConfig) => Promise<void>
  handleToggleEngine: (engine: EngineConfig, enabled: boolean) => Promise<void>
  handleBulkToggleEngines: (enabled: boolean) => Promise<void>
  handleSaveDeepseekKey: () => Promise<void>
  handleSaveGlmKey: () => Promise<void>
  handleSaveOpencodeGoKey: () => Promise<void>
  handleSaveOpencodeZenKey: () => Promise<void>
  handleSaveModelHubKey: () => Promise<void>
  // A 刀「自定义引擎」：add/update 成功返回 true（表单据此关闭）
  handleAddCustomEngine: (name: string, baseURL: string, apiKey: string) => Promise<boolean>
  handleUpdateCustomEngine: (engineID: string, name: string, baseURL: string, apiKey: string) => Promise<boolean>
  handleRemoveCustomEngine: (engineID: string) => Promise<void>
  handleSetGlmEndpoint: (family: 'std' | 'coding') => Promise<void>
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

export interface ModelCenterContextValue extends ModelCenterStateValue, ModelCenterActionsValue {
  settingGlmEndpoint: boolean
}

export const ModelCenterStateContext = createContext<ModelCenterStateValue | null>(null)
export const ModelCenterActionsContext = createContext<ModelCenterActionsValue | null>(null)
export const ModelCenterContext = createContext<ModelCenterContextValue | null>(null)

/** state-only 消费者走此口：动作引用变化不触发重渲染（v4.352 拆分通道） */
export function useModelCenterState(): ModelCenterStateValue {
  const v = useContext(ModelCenterStateContext)
  if (!v) throw new Error('useModelCenterState 必须在 ModelCenterPage 内使用')
  return v
}

export function useModelCenterActions(): ModelCenterActionsValue {
  const v = useContext(ModelCenterActionsContext)
  if (!v) throw new Error('useModelCenterActions 必须在 ModelCenterPage 内使用')
  return v
}
