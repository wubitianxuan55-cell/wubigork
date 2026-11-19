import { useEffect, useMemo } from 'react'
import { message } from 'antd'
import { useEngineState } from './hooks/useEngineState'
import { useStatsState } from './hooks/useStatsState'
import { useImageState } from './hooks/useImageState'
import { useVoiceState } from './hooks/useVoiceState'
import { useBindState } from './hooks/useBindState'
import { engineLabel, type Category } from './utils'
import {
  type ModelCenterContextValue, type ModelCenterStateValue, type ModelCenterActionsValue,
} from './context'

/** 订阅 runtime 事件并返回退订函数（桌面与 HTTP polyfill 的 EventsOn 均返回退订；旧运行时返回 void 时跳过退订，宁可残留不全清） */
export function runtimeOn(event: string, handler: (data: unknown) => void): (() => void) | undefined {
  try {
    const on = window.runtime?.EventsOn as ((e: string, h: (d: unknown) => void) => (() => void) | void) | undefined
    const res = on?.(event, handler)
    return typeof res === 'function' ? res : undefined
  } catch {
    return undefined
  }
}

/**
 * useModelCenterBundle — 模型中心五 hook 组装 + 三 Context 值单源
 * （设置中枢批 3 抽出：ModelCenterPage 与设置页 ModelSettingsProvider 共用，
 * 组装/同步 effects 不再两份漂移）。category 仅驱动 hooks 内刷新节奏（定时器/
 * 在途守卫），无 rail 导航语义——设置侧传固定值即可。
 */
export function useModelCenterBundle(
  category: Category,
  statsOpen: boolean,
  deps: { setCategory: (c: Category) => void; openStatsDrawer: () => void },
) {
  const { setCategory, openStatsDrawer } = deps
  const engine = useEngineState(category)
  const stats = useStatsState(statsOpen)
  const image = useImageState()
  const voice = useVoiceState()
  const bind = useBindState(engine.engines)

  // 同步：其他页面（FeatureModelBar 等）修改绑定后，面板即时刷新
  const { loadFeatureCfg: loadFeatureCfgSync, refreshRoutes: refreshRoutesSync } = bind
  useEffect(() => {
    const reload = () => { loadFeatureCfgSync(); refreshRoutesSync() }
    const unsub = runtimeOn('feature-model-changed', reload)
    return () => {
      try { unsub?.() } catch (_) {}
    }
  }, [loadFeatureCfgSync, refreshRoutesSync])

  // 同步：其他页面切换活跃模型/语音模型后，面板即时刷新。
  // enginesLive 一并解出：failover 提示要把 to_engine id 映射为显示名，
  // 必须读到最新列表（沿用「解构 + 入依赖」的闭包新鲜度模式）。
  const { loadAll: loadAllSync, engines: enginesLive } = engine
  const { loadVoiceCfg: loadVoiceCfgSync } = voice
  useEffect(() => {
    const reload = () => { loadAllSync(); refreshRoutesSync() }
    const reloadVoice = () => { loadVoiceCfgSync() }
    const unsub1 = runtimeOn('model-changed', reload)
    const unsub2 = runtimeOn('voice-model-changed', reloadVoice)
    // 本地 TTS 服务就绪/失败后同步刷新（breeze 冷启动含模型加载，约 6–10 秒）
    const unsub3 = runtimeOn('tts-service-status', reload)
    return () => {
      try { unsub1?.() } catch (_) {}
      try { unsub2?.() } catch (_) {}
      try { unsub3?.() } catch (_) {}
    }
  }, [loadAllSync, loadVoiceCfgSync, refreshRoutesSync])

  // 同步（C 刀：健康巡检 + 故障转移 v0）：后端巡检发现引擎连通性变化 → 重拉
  // 引擎列表；调用失败转移 → info 级提示。payload 契约：
  // engine-health-changed {id, connected}；model-failover {from_engine, to_engine, model}。
  useEffect(() => {
    const onHealth = () => { loadAllSync() }
    const onFailover = (data: unknown) => {
      const p = (data ?? {}) as { to_engine?: unknown }
      const to = typeof p.to_engine === 'string' ? p.to_engine : ''
      if (!to) {
        message.info('调用失败，已自动切换引擎重试')
        return
      }
      // to_engine 是引擎 id，对用户不友好；查最新引擎列表换显示名
      // （engineLabel：后端 label → 内置映射 → id 兜底）。列表缺失/流式竞态
      // 查不到时按 id 原样提示，三级回退语义不变。
      const hit = enginesLive.find(e => e.id === to)
      message.info(`调用失败，已切换到 ${hit ? engineLabel(hit) : to} 重试`)
    }
    const unsub1 = runtimeOn('engine-health-changed', onHealth)
    const unsub2 = runtimeOn('model-failover', onFailover)
    return () => {
      try { unsub1?.() } catch (_) {}
      try { unsub2?.() } catch (_) {}
    }
  }, [loadAllSync, enginesLive])

  // v4.352：ctx 拆 state/actions 两通道——五 hook 返回对象字面量每渲染新引用，
  // deps 无法建立，拆分后 state 通道由各 hook 的状态字段组成（内容变化才重渲染），
  // 动作通道单独承载函数引用波动。
  const stateCtx: ModelCenterStateValue = useMemo(() => ({
    category,
    engines: engine.engines, engineStatuses: engine.engineStatuses,
    editingURLs: engine.editingURLs,
    savingEngine: engine.savingEngine, testingEngine: engine.testingEngine,
    activeEngine: engine.activeEngine, activeModel: engine.activeModel,
    deepseekKey: engine.deepseekKey, deepseekKeyMasked: engine.deepseekKeyMasked,
    glmKey: engine.glmKey, glmKeyMasked: engine.glmKeyMasked,
    opencodeGoKey: engine.opencodeGoKey, opencodeGoKeyMasked: engine.opencodeGoKeyMasked,
    settingGlmEndpoint: engine.settingGlmEndpoint,
    callStats: stats.callStats, statsSort: stats.statsSort,
    loadError: stats.loadError,
    trendRange: stats.trendRange, trendData: stats.trendData,
    imageBackend: image.imageBackend,
    comfyUIURL: image.comfyUIURL, comfyUIPath: image.comfyUIPath, comfyUIPythonPath: image.comfyUIPythonPath,
    imageModel: image.imageModel,
    imageSaveDir: image.imageSaveDir,
    imageBackendSaving: image.imageBackendSaving, comfyStatus: image.comfyStatus, comfyBusy: image.comfyBusy,
    voiceCfg: voice.voiceCfg,
    ocrCfg: voice.ocrCfg,
    chatVoiceCfg: voice.chatVoiceCfg, chatVoiceDraft: voice.chatVoiceDraft,
    chatVoiceSaving: voice.chatVoiceSaving, chatVoiceSpeakers: voice.chatVoiceSpeakers,
    chatVoiceOptions: voice.chatVoiceOptions, chatVoiceValue: voice.chatVoiceValue,
    featureCfg: bind.featureCfg, featureDraft: bind.featureDraft,
    featureEnabled: bind.featureEnabled, modelRoutes: bind.modelRoutes, visionRoute: bind.visionRoute,
    portraitCfg: bind.portraitCfg, portraitDraft: bind.portraitDraft,
    portraitModelOptions: bind.portraitModelOptions, portraitSaving: bind.portraitSaving,
    sinImageCfg: bind.sinImageCfg, sinImageDraft: bind.sinImageDraft,
    sinImageModelOptions: bind.sinImageModelOptions, sinImageSaving: bind.sinImageSaving,
    llmModels: engine.llmModels, ttsModels: engine.ttsModels, sttModels: engine.sttModels,
    imageModels: engine.imageModels, specialtyModels: engine.specialtyModels,
  }), [category, engine, stats, image, voice, bind])

  const actionsCtx: ModelCenterActionsValue = useMemo(() => ({
    setCategory,
    setEditingURLs: engine.setEditingURLs,
    setDeepseekKeyState: engine.setDeepseekKeyState,
    setGlmKeyState: engine.setGlmKeyState,
    setOpencodeGoKeyState: engine.setOpencodeGoKeyState,
    setStatsSort: stats.setStatsSort,
    setTrendRange: stats.setTrendRange,
    setImageBackend: image.setImageBackend,
    setImageModel: image.setImageModel,
    setImageSaveDir: image.setImageSaveDir,
    setVoiceCfg: voice.setVoiceCfg,
    setOcrCfg: voice.setOcrCfg,
    setChatVoiceDraft: voice.setChatVoiceDraft,
    setFeatureDraft: bind.setFeatureDraft,
    setPortraitDraft: bind.setPortraitDraft,
    setSinImageDraft: bind.setSinImageDraft,
    makeModels: engine.makeModels, isModelActive: engine.isModelActive,
    handleTestConnection: engine.handleTestConnection,
    handleRefreshModels: engine.handleRefreshModels,
    handleStartModel: engine.handleStartModel,
    handleSaveURL: engine.handleSaveURL,
    handleToggleEngine: engine.handleToggleEngine,
    handleBulkToggleEngines: engine.handleBulkToggleEngines,
    handleSaveDeepseekKey: engine.handleSaveDeepseekKey,
    handleSaveGlmKey: engine.handleSaveGlmKey,
    handleSaveOpencodeGoKey: engine.handleSaveOpencodeGoKey,
    handleAddCustomEngine: engine.handleAddCustomEngine,
    handleUpdateCustomEngine: engine.handleUpdateCustomEngine,
    handleRemoveCustomEngine: engine.handleRemoveCustomEngine,
    handleSetGlmEndpoint: engine.handleSetGlmEndpoint,
    handleResetCallStats: stats.handleResetCallStats, loadCallStats: stats.loadCallStats,
    openStatsDrawer,
    handleToggleComfy: image.handleToggleComfy, handleSaveImageBackend: image.handleSaveImageBackend,
    handleSetVoiceModel: voice.handleSetVoiceModel, handleSetOCRModel: voice.handleSetOCRModel,
    handleSaveFeature: bind.handleSaveFeature, handleToggleFeatureEnabled: bind.handleToggleFeatureEnabled,
    handleSavePortrait: bind.handleSavePortrait,
    handleSaveSinImage: bind.handleSaveSinImage,
    handleSaveChatVoice: voice.handleSaveChatVoice, handleClearChatVoice: voice.handleClearChatVoice,
  }), [setCategory, openStatsDrawer, engine, stats, image, voice, bind])

  const ctx: ModelCenterContextValue = useMemo(
    () => ({ ...stateCtx, ...actionsCtx }),
    [stateCtx, actionsCtx],
  )

  return { ctx, stateCtx, actionsCtx, engine, stats, image, voice, bind }
}
