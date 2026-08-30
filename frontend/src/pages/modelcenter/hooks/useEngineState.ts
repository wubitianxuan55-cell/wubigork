/**
 * useEngineState — 模型中心「引擎管理/语言模型」状态 Hook（T6-6.4 UI 拆分）
 *
 * 归集引擎列表、活跃模型、连接测试、引擎启停与各云端 API Key 的全部状态，
 * 原 ModelCenterPage 顶部 42 个 useState 中的引擎/密钥 14 项下沉于此。
 * 顶层页面仅保留全局状态（category/statsOpen/loggingIn），并通过本 Hook
 * 组合出模型中心共享上下文。
 *
 * T6-6.5 竞态修复：refreshLocalModels 带请求序号守卫，切分类（category 变化）
 * 时 effect 清理会作废在途刷新的结果；5s 定时器随 category 重置。
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import type { Dispatch, SetStateAction } from 'react'
import { message } from 'antd'
import * as App from '../../../wailsjsCompat'
import {
  getEngines, saveEngine, testEngineConnection, refreshEngineModels, setEngineDefaultModel,
  setActiveEngine, getActiveEngine, setDeepseekKey, getDeepseekKeyStatus,
  setGlmKey, getGlmKeyStatus, setGlmEndpoint,
  setOpencodeGoKey, getOpencodeGoKeyStatus, setOpencodeZenKey, getOpencodeZenKeyStatus,
  type EngineConfig, type EngineStatus,
} from '../../../api/engines'
import { kindOf, engineLabel, type Category, type ModelCardData } from '../utils'

/** 提取错误消息（unknown 收窄；无 message 用 fallback） */
function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}

export interface EngineState {
  loading: boolean
  engines: EngineConfig[]
  engineStatuses: Record<string, EngineStatus>
  activeEngine: string
  activeModel: string
  testingEngine: string | null
  editingURLs: Record<string, string>
  setEditingURLs: Dispatch<SetStateAction<Record<string, string>>>
  savingEngine: string | null
  deepseekKey: string
  setDeepseekKeyState: (v: string) => void
  deepseekKeyMasked: string
  glmKey: string
  setGlmKeyState: (v: string) => void
  glmKeyMasked: string
  opencodeGoKey: string
  setOpencodeGoKeyState: (v: string) => void
  opencodeGoKeyMasked: string
  opencodeZenKey: string
  setOpencodeZenKeyState: (v: string) => void
  opencodeZenKeyMasked: string
  settingGlmEndpoint: boolean
  handleSetGlmEndpoint: (family: 'std' | 'coding') => Promise<void>
  loadAll: () => Promise<void>
  refreshLocalModels: () => Promise<void>
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
  handleBulkToggleEngines: (enabled: boolean) => Promise<void>
  handleSaveDeepseekKey: () => Promise<void>
  handleSaveGlmKey: () => Promise<void>
  handleSaveOpencodeGoKey: () => Promise<void>
  handleSaveOpencodeZenKey: () => Promise<void>
}

export function useEngineState(category: Category): EngineState {
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
  const [glmKey, setGlmKeyState] = useState('')
  const [settingGlmEndpoint, setSettingGlmEndpoint] = useState(false)
  const [glmKeyMasked, setGlmKeyMasked] = useState('')
  const [opencodeGoKey, setOpencodeGoKeyState] = useState('')
  const [opencodeGoKeyMasked, setOpencodeGoKeyMasked] = useState('')
  const [opencodeZenKey, setOpencodeZenKeyState] = useState('')
  const [opencodeZenKeyMasked, setOpencodeZenKeyMasked] = useState('')

  // T6-6.5 竞态守卫：loadAll 的最新请求序号，晚到的旧响应直接丢弃。
  const loadAllSeq = useRef(0)
  // refreshLocalModels 的请求序号：切分类（category effect 清理）会自增使在途刷新作废。
  const refreshSeq = useRef(0)

  const loadAll = useCallback(async () => {
    const seq = ++loadAllSeq.current
    try {
      const list = await getEngines()
      if (seq !== loadAllSeq.current) return // 已有更新的加载请求，丢弃过期结果
      setEngines(list)
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
        const ks = await getGlmKeyStatus()
        if (ks) { setGlmKeyMasked(ks.masked || '') }
      } catch (_) {}
      try {
        const ks = await getOpencodeGoKeyStatus()
        if (ks) { setOpencodeGoKeyMasked(ks.masked || '') }
      } catch (_) {}
      try {
        const ks = await getOpencodeZenKeyStatus()
        if (ks) { setOpencodeZenKeyMasked(ks.masked || '') }
      } catch (_) {}
    } catch (err: unknown) {
      if (seq === loadAllSeq.current) {
        message.error(errText(err, '加载引擎列表失败，请重试'))
      }
    } finally {
      if (seq === loadAllSeq.current) setLoading(false)
    }
  }, [])

  // 本地引擎模型列表延后 5 秒刷新，且仅在用户尚未切换分类时执行。
  // T6-6.5：请求序号守卫——切分类时 effect 体自增 refreshSeq，在途刷新
  // 返回时序号不匹配即丢弃（不触发 loadAll，避免旧结果覆盖新分类视图）。
  const refreshLocalModels = useCallback(async () => {
    const seq = ++refreshSeq.current
    await Promise.allSettled(['herdsman', 'ollama', 'cosyvoice'].map(id => refreshEngineModels(id)))
    if (seq !== refreshSeq.current) return // 过期刷新结果：切分类后已作废，丢弃
    await loadAll()
  }, [loadAll])

  // T6-6.5：5s 定时器随 category 重置（effect 依赖 category）。
  // effect 体开头自增 refreshSeq 作废上一分类在途的刷新：切分类后旧分类
  // 触发的刷新返回时序号不匹配即被丢弃（避免旧结果覆盖新分类视图）。
  useEffect(() => {
    refreshSeq.current++ // 作废上一分类（或挂载前）在途刷新
    const timer = window.setTimeout(() => { void refreshLocalModels() }, 5000)
    return () => window.clearTimeout(timer)
  }, [refreshLocalModels, category])

  // 首次进入模型中心读取已保存的引擎状态（启动阶段不抢线程，仅读取）。
  useEffect(() => { void loadAll() }, [loadAll])

  const handleStartModel = async (card: ModelCardData) => {
    const kind = kindOf(card)
    if (kind === 'tts') {
      // 本地 TTS 模型：启动对应本地服务（CosyVoice2 8010，幂等）
      try {
        const res = await App.StartLocalTTSService(card.engineId)
        if (res?.ready) message.success(`本地 TTS 服务已就绪：${card.engineName}`)
        else message.success(`正在启动本地 TTS 服务：${card.engineName}（首次约 1–2 分钟）`)
        loadAll()
      } catch (err: unknown) { message.error(`启动失败：${err instanceof Error ? err.message : String(err)}`) }
      return
    }
    if (kind !== 'llm') return
    try {
      if (activeEngine !== card.engineId) { await setActiveEngine(card.engineId); setActiveEngineState(card.engineId) }
      await setEngineDefaultModel(card.engineId, card.modelId)
      setEngines(prev => prev.map(e => e.id === card.engineId ? { ...e, default_model: card.modelId } : e))
      setActiveModel(card.modelId)
      message.success(`已启动${card.modelName}`)
    } catch (err: unknown) { message.error(`启动失败：${err instanceof Error ? err.message : String(err)}`) }
  }

  const isModelActive = (card: ModelCardData) => activeEngine === card.engineId && activeModel === card.modelId

  const handleTestConnection = async (id: string) => {
    setTestingEngine(id)
    try {
      const st = await testEngineConnection(id)
      if (st) {
        setEngineStatuses(prev => ({ ...prev, [id]: st }))
        const ms = typeof st.latency_ms === 'number' ? st.latency_ms : 0
        if (st.connected) {
          message.success(`${engineLabel({ id })} 连接成功（${st.model_count} 个模型${ms ? `，${ms}ms` : ''}）`)
        } else {
          message.error(`${engineLabel({ id })} 连接失败：${st.error || '未知错误'}`)
        }
      }
      await loadAll()
    } catch (err: unknown) { message.error(errText(err, '操作失败')) }
    finally { setTestingEngine(null) }
  }

  const handleRefreshModels = async (id: string) => {
    setTestingEngine(id)
    try { await refreshEngineModels(id); await loadAll() } catch (err: unknown) { message.error(errText(err, '操作失败')) }
    finally { setTestingEngine(null) }
  }

  const handleSaveURL = async (engine: EngineConfig) => {
    setSavingEngine(engine.id)
    try { await saveEngine({ id: engine.id, base_url: editingURLs[engine.id] || '', enabled: engine.enabled } as EngineConfig); message.success('已保存') }
    catch (err: unknown) { message.error(errText(err, '操作失败')) }
    finally { setSavingEngine(null) }
  }

  const handleToggleEngine = async (engine: EngineConfig, enabled: boolean) => {
    try { await saveEngine({ id: engine.id, base_url: engine.base_url || '', enabled } as EngineConfig); await loadAll() }
    catch (err: unknown) { message.error(errText(err, '操作失败')) }
  }

  const handleBulkToggleEngines = async (enabled: boolean) => {
    try {
      await Promise.all(engines.map(e => saveEngine({ id: e.id, base_url: e.base_url || '', enabled } as EngineConfig)))
      message.success(enabled ? '已全部启用' : '已全部禁用')
      await loadAll()
    } catch (err: unknown) { message.error(errText(err, '操作失败')) }
  }

  const handleSaveDeepseekKey = async () => {
    if (!deepseekKey.trim()) { message.warning('请输入 API Key'); return }
    try {
      await setDeepseekKey(deepseekKey.trim())
      message.success('DeepSeek Key 已保存')
      const ks = await getDeepseekKeyStatus()
      if (ks) setDeepseekKeyMasked(ks.masked || '')
      setDeepseekKeyState('')
    } catch (err: unknown) { message.error(errText(err, '操作失败')) }
  }

  const handleSaveGlmKey = async () => {
    if (!glmKey.trim()) { message.warning('请输入 API Key'); return }
    try {
      await setGlmKey(glmKey.trim())
      message.success('GLM Key 已保存')
      const ks = await getGlmKeyStatus()
      if (ks) setGlmKeyMasked(ks.masked || '')
      setGlmKeyState('')
    } catch (err: unknown) { message.error(errText(err, '操作失败')) }
  }

  const handleSaveOpencodeGoKey = async () => {
    if (!opencodeGoKey.trim()) { message.warning('请输入 API Key'); return }
    try {
      await setOpencodeGoKey(opencodeGoKey.trim())
      message.success('OpenCode Go Key 已保存')
      const ks = await getOpencodeGoKeyStatus()
      if (ks) setOpencodeGoKeyMasked(ks.masked || '')
      setOpencodeGoKeyState('')
    } catch (err: unknown) { message.error(errText(err, '操作失败')) }
  }

  const handleSaveOpencodeZenKey = async () => {
    if (!opencodeZenKey.trim()) { message.warning('请输入 API Key'); return }
    try {
      await setOpencodeZenKey(opencodeZenKey.trim())
      message.success('OpenCode Zen Key 已保存')
      const ks = await getOpencodeZenKeyStatus()
      if (ks) setOpencodeZenKeyMasked(ks.masked || '')
      setOpencodeZenKeyState('')
    } catch (err: unknown) { message.error(errText(err, '操作失败')) }
  }

  // GLM 端点家族切换（std=标准按量付费 / coding=编码套餐额度；后端只收官方双端点）
  const handleSetGlmEndpoint = async (family: 'std' | 'coding') => {
    setSettingGlmEndpoint(true)
    try {
      await setGlmEndpoint(family)
      message.success(family === 'coding' ? '已切换到 GLM 编码套餐端点' : '已切换到 GLM 标准端点')
      await loadAll()
    } catch (err: unknown) { message.error(errText(err, '端点切换失败')) }
    finally { setSettingGlmEndpoint(false) }
  }

  const makeModels = (engine: EngineConfig): ModelCardData[] =>
    (engine.models || []).map(m => ({ modelId: m.id, modelName: m.id, engineId: engine.id, engineName: engine.name, engineType: engine.type, engineEnabled: engine.enabled, status: m.status || 'running', kind: m.kind || '' }))

  const allModels = engines.filter(e => e.enabled).flatMap(e => makeModels(e))
  const llmModels = allModels.filter(m => kindOf(m) === 'llm')
  const ttsModels = allModels.filter(m => kindOf(m) === 'tts')
  const sttModels = allModels.filter(m => kindOf(m) === 'stt')
  const imageModels = allModels.filter(m => kindOf(m) === 'image')
  const specialtyModels = allModels.filter(m => ['embedding', 'rerank', 'ocr'].includes(kindOf(m)))

  return {
    loading,
    engines,
    engineStatuses,
    activeEngine,
    activeModel,
    testingEngine,
    editingURLs,
    setEditingURLs,
    savingEngine,
    deepseekKey,
    setDeepseekKeyState,
    deepseekKeyMasked,
    glmKey,
    setGlmKeyState,
    glmKeyMasked,
    opencodeGoKey,
    setOpencodeGoKeyState,
    opencodeGoKeyMasked,
    opencodeZenKey,
    setOpencodeZenKeyState,
    opencodeZenKeyMasked,
    settingGlmEndpoint,
    handleSetGlmEndpoint,
    loadAll,
    refreshLocalModels,
    llmModels, ttsModels, sttModels, imageModels, specialtyModels,
    makeModels,
    isModelActive,
    handleTestConnection,
    handleRefreshModels,
    handleStartModel,
    handleSaveURL,
    handleToggleEngine,
    handleBulkToggleEngines,
    handleSaveDeepseekKey,
    handleSaveGlmKey,
    handleSaveOpencodeGoKey,
    handleSaveOpencodeZenKey,
  }
}

