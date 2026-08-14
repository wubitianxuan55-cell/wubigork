import React, { useState, useEffect, useCallback, useMemo } from 'react'
import { Button, Drawer, message } from 'antd'
import {
  ThunderboltOutlined, PictureOutlined, SoundOutlined, SettingOutlined, LinkOutlined,
  CheckCircleOutlined, LoginOutlined, LogoutOutlined, DatabaseOutlined, DashboardOutlined, ReloadOutlined, BarChartOutlined,
  AppstoreOutlined, ExperimentOutlined,
} from '@ant-design/icons'
import { useAppStore } from '../stores/appStore'
import * as App from '../../src/wailsjsCompat'
import {
  getEngines, saveEngine, testEngineConnection,
  refreshEngineModels, setEngineDefaultModel,
  setActiveEngine, getActiveEngine, setDeepseekKey, getDeepseekKeyStatus,
  getActiveOCRModel, setActiveOCRModel,
  setOpencodeGoKey, getOpencodeGoKeyStatus,
  setOpencodeZenKey, getOpencodeZenKeyStatus,
  getModelCallStats, resetModelCallStats,
  type EngineConfig, type EngineStatus, type ModelStatsSummary,
} from '../api/engines'
import {
  getImageBackendInfo, setImageBackend as setImageBackendAPI,
} from '../api/settings'
import { getPortraitConfig, setPortraitConfig } from '../api/image'
import { startComfyUI, stopComfyUI, getComfyUIStatus } from '../api/image'
import { ModelCenterContext, type ModelCenterContextValue, type VoiceCfg } from './modelcenter/context'
import { LLMSection } from './modelcenter/LLMSection'
import { BindSection } from './modelcenter/BindSection'
import { StatsSection } from './modelcenter/StatsSection'
import { ImageSection } from './modelcenter/ImageSection'
import { VoiceSection } from './modelcenter/VoiceSection'
import { EngineSection } from './modelcenter/EngineSection'
import { SpecialtySection } from './modelcenter/SpecialtySection'
import { OverviewSection } from './modelcenter/OverviewSection'
import { ResourceMonitor } from './modelcenter/ResourceMonitor'
import { HerdsmanCatalogSection } from './modelcenter/HerdsmanCatalogSection'
import { BenchmarkSection } from './modelcenter/BenchmarkSection'
import './modelcenter/modelcenter.css'
import {
  FEATURES, XAI_VOICES, imageModelOptionsFor, kindOf, localTTSDefaultVoice, localTTSFallbackVoices, engineLabel,
  type Category, type ModelCardData,
} from './modelcenter/utils'
import { type StatsSort, type TrendDatum, type TrendRange } from './modelcenter/charts'

const ModelCenterPage: React.FC = () => {
  const { loggedIn, login, logout } = useAppStore()
  const [category, setCategory] = useState<Category>('overview')
  const [statsOpen, setStatsOpen] = useState(false)
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
  const [voiceCfg, setVoiceCfg] = useState<VoiceCfg>({ stt: { engine: '', model: '' }, llm: { engine: '', model: '' }, tts: { engine: '', model: '', voice: '' } })
  const [ocrCfg, setOcrCfg] = useState<{ engine: string; model: string }>({ engine: '', model: '' })
  // 功能绑定：聊天语音合成（优先于全局 TTS，便于后续扩展更多语音绑定）
  const [chatVoiceCfg, setChatVoiceCfg] = useState<{ engine: string; model: string }>({ engine: '', model: '' })
  const [chatVoiceDraft, setChatVoiceDraft] = useState<{ engine: string; model: string }>({ engine: '', model: '' })
  const [chatVoiceSaving, setChatVoiceSaving] = useState(false)
  const [chatVoiceSpeakers, setChatVoiceSpeakers] = useState<string[]>([])
  const [featureCfg, setFeatureCfg] = useState<Record<string, { engine: string; model: string }>>({})
  const [featureDraft, setFeatureDraft] = useState<Record<string, { engine: string; model: string }>>({})
  const [featureEnabled, setFeatureEnabled] = useState<Record<string, boolean>>({})
  const [modelRoutes, setModelRoutes] = useState<Record<string, { engine: string; model: string; source: string }>>({})
  const [portraitCfg, setPortraitCfg] = useState<{ backend: string; model: string }>({ backend: '', model: '' })
  const [portraitDraft, setPortraitDraft] = useState<{ backend: string; model: string }>({ backend: '', model: '' })
  const [portraitSaving, setPortraitSaving] = useState(false)

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
    } catch (err: any) {
      message.error(err?.message || '加载引擎列表失败，请重试')
    }
    finally { setLoading(false) }
  }, [])

  // 模型中心首次进入时先只读取已保存的引擎状态，避免启动阶段刷新请求和用户首次点击抢线程。
  // 本地引擎模型列表延后 5 秒刷新，且仅在用户尚未切换分类时执行。
  const refreshLocalModels = useCallback(async () => {
    await Promise.allSettled(['herdsman', 'ollama', 'cosyvoice'].map(id => refreshEngineModels(id)))
    await loadAll()
  }, [loadAll])

  // 模型中心左侧分类切换后，把外层滚动容器回到顶部，避免上一分类的滚动位置
  // 残留，导致功能绑定等页面顶部的控件落在可视区域外、看起来像下拉框点不开。
  useEffect(() => {
    const scroller = document.querySelector('.ant-layout-content')
    if (scroller && typeof scroller.scrollTo === 'function') {
      scroller.scrollTo({ top: 0, behavior: 'auto' })
    }
  }, [category])

  const loadCallStats = useCallback(async () => {
    try {
      const s = await getModelCallStats()
      if (s) setCallStats(s)
    } catch (_) {}
  }, [])

  // 调用统计抽屉打开时定时刷新
  useEffect(() => {
    if (!statsOpen) return
    loadCallStats()
    const timer = window.setInterval(loadCallStats, 15000)
    return () => window.clearInterval(timer)
  }, [statsOpen, loadCallStats])

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
      if (cfg?.image_model || cfg?.model) setImageModel(cfg.image_model || cfg.model)
      if (cfg?.comfyui_url) setComfyUIURL(cfg.comfyui_url)
      if (cfg?.image_save_dir) setImageSaveDir(cfg.image_save_dir)
      if (cfg?.comfyui_path) setComfyUIPath(cfg.comfyui_path)
      if (cfg?.comfyui_python_path) setComfyUIPythonPath(cfg.comfyui_python_path)
    } catch (_) {}
    try {
      const st: any = await getComfyUIStatus()
      if (st) {
        const port = typeof st.port === 'number'
          ? st.port
          : st.url ? Number(String(st.url).split(':').pop()) || 8188 : 0
        setComfyStatus({ running: !!st.running, port })
        if (st.url) setComfyUIURL(st.url)
      }
    } catch (_) {}
  }, [])

  const handleToggleComfy = async () => {
    setComfyBusy(true)
    try {
      if (comfyStatus.running) { await stopComfyUI(); setComfyStatus({ running: false, port: 0 }) }
      else { await startComfyUI(); setComfyStatus({ running: true, port: 8188 }) }
      const st: any = await getComfyUIStatus()
      if (st) {
        const port = typeof st.port === 'number' ? st.port : 8188
        setComfyStatus({ running: !!st.running, port })
        message.success(st.running ? 'ComfyUI 已启动' : 'ComfyUI 已停止')
      } else {
        message.success(comfyStatus.running ? 'ComfyUI 已停止' : 'ComfyUI 已启动')
      }
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

  const loadOCRCfg = useCallback(async () => {
    try {
      const cfg = await getActiveOCRModel()
      if (cfg) setOcrCfg({ engine: cfg.engine || '', model: cfg.model || '' })
    } catch (_) {}
  }, [])

  // 当前生效路由（后端 routeModel 降级链结果：feature / global / fallback）
  const refreshRoutes = useCallback(async () => {
    const bind = (window as any).go?.app?.App
    if (!bind?.GetModelRoute) return
    const next: Record<string, { engine: string; model: string; source: string }> = {}
    for (const key of ['chat', 'novel', 'office', 'gaea', 'characterlib', 'routine']) {
      try {
        next[key] = JSON.parse(await bind.GetModelRoute(key))
      } catch { /* 单功能失败忽略 */ }
    }
    setModelRoutes(next)
  }, [])
  useEffect(() => { refreshRoutes() }, [refreshRoutes])

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

  // 同步：其他页面切换活跃模型/语音模型后，本面板即时刷新
  useEffect(() => {
    const reload = () => { loadAll(); refreshRoutes() }
    const reloadVoice = () => { loadVoiceCfg() }
    let unsub1: any, unsub2: any
    try { unsub1 = (window as any).runtime?.EventsOn?.('model-changed', reload) } catch (_) {}
    try { unsub2 = (window as any).runtime?.EventsOn?.('voice-model-changed', reloadVoice) } catch (_) {}
    // 本地 TTS 服务就绪/失败后同步刷新（CosyVoice2 异步启动约 1–2 分钟）
    let unsub3: any
    try { unsub3 = (window as any).runtime?.EventsOn?.('tts-service-status', reload) } catch (_) {}
    return () => {
      try { if (typeof unsub1 === 'function') unsub1() } catch (_) {}
      try { if (typeof unsub2 === 'function') unsub2() } catch (_) {}
      try { if (typeof unsub3 === 'function') unsub3() } catch (_) {}
    }
  }, [loadAll, loadVoiceCfg, refreshRoutes])

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

  useEffect(() => {
    const timer = window.setTimeout(() => { refreshLocalModels() }, 5000)
    return () => window.clearTimeout(timer)
  }, [refreshLocalModels])

  useEffect(() => { loadAll(); loadImageBackend(); loadVoiceCfg(); loadOCRCfg(); loadFeatureCfg() }, [loadAll, loadVoiceCfg, loadOCRCfg, loadFeatureCfg])
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

  const handleSetOCRModel = async (engineId: string, modelId: string) => {
    try {
      await setActiveOCRModel(engineId, modelId)
      message.success(engineId && modelId ? `已设为 OCR：${modelId}` : 'OCR 已恢复自动选择')
      loadOCRCfg()
    } catch (err: any) {
      message.error(err?.message || '设置 OCR 失败')
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
    const kind = kindOf(card)
    if (kind === 'tts') {
      // 本地 TTS 模型：启动对应本地服务（CosyVoice2 8010，幂等）
      try {
        const res = await App.StartLocalTTSService(card.engineId)
        if (res?.ready) message.success(`本地 TTS 服务已就绪：${card.engineName}`)
        else message.success(`正在启动本地 TTS 服务：${card.engineName}（首次约 1–2 分钟）`)
        loadAll()
      } catch (err: any) { message.error(`启动失败：${err.message || err}`) }
      return
    }
    if (kind !== 'llm') return
    try {
      if (activeEngine !== card.engineId) { await setActiveEngine(card.engineId); setActiveEngineState(card.engineId) }
      await setEngineDefaultModel(card.engineId, card.modelId)
      setEngines(prev => prev.map(e => e.id === card.engineId ? { ...e, default_model: card.modelId } : e))
      setActiveModel(card.modelId)
      message.success(`已启动${card.modelName}`)
    } catch (err: any) { message.error(`启动失败：${err.message || err}`) }
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

  const handleBulkToggleEngines = async (enabled: boolean) => {
    try {
      await Promise.all(engines.map(e => saveEngine({ id: e.id, base_url: e.base_url || '', enabled } as any)))
      message.success(enabled ? '已全部启用' : '已全部禁用')
      await loadAll()
    } catch (err: any) { message.error(err.message) }
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
    (engine.models || []).map(m => ({ modelId: m.id, modelName: m.id, engineId: engine.id, engineName: engine.name, engineType: engine.type, engineEnabled: engine.enabled, status: m.status || 'running', kind: m.kind || '' }))

  const allModels = engines.filter(e => e.enabled).flatMap(e => makeModels(e))
  const llmModels = allModels.filter(m => kindOf(m) === 'llm')
  const ttsModels = allModels.filter(m => kindOf(m) === 'tts')
  const sttModels = allModels.filter(m => kindOf(m) === 'stt')
  const imageModels = allModels.filter(m => kindOf(m) === 'image')
  const specialtyModels = allModels.filter(m => ['embedding', 'rerank', 'ocr'].includes(kindOf(m)))

  // 角色库剧照独立后端/模型选项（空 = 跟随绘梦）
  const portraitModelOptions = useMemo(() => {
    const b = portraitDraft.backend
    if (!b) return [{ label: '跟随绘梦', value: '' }]
    return [
      { label: '跟随绘梦', value: '' },
      ...imageModelOptionsFor(b, engines, portraitDraft.model),
    ]
  }, [portraitDraft.backend, portraitDraft.model, engines])

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

  const TABS: { key: Category; icon: React.ReactNode; label: string }[] = [
    { key: 'overview', icon: <DashboardOutlined />, label: '总览' },
    { key: 'llm', icon: <ThunderboltOutlined />, label: '语言模型' },
    { key: 'image', icon: <PictureOutlined />, label: '图片生成' },
    { key: 'tts', icon: <SoundOutlined />, label: '语音模型' },
    { key: 'specialty', icon: <DatabaseOutlined />, label: '专业模型' },
    { key: 'catalog', icon: <AppstoreOutlined />, label: '模型库' },
    { key: 'benchmark', icon: <ExperimentOutlined />, label: '受控测评' },
    { key: 'bind', icon: <LinkOutlined />, label: '功能绑定' },
    { key: 'engine', icon: <SettingOutlined />, label: '引擎管理' },
  ]

  const navTab = (tab: { key: Category; icon: React.ReactNode; label: string }) => (
    <button
      type="button"
      key={tab.key}
      className={`mc-tab${category === tab.key ? ' is-active' : ''}`}
      aria-selected={category === tab.key}
      onClick={() => setCategory(tab.key)}
    >
      {tab.icon}
      <span>{tab.label}</span>
    </button>
  )

  const ctx: ModelCenterContextValue = {
    category, setCategory,
    engines, engineStatuses, editingURLs, setEditingURLs, savingEngine, testingEngine, activeEngine, activeModel,
    deepseekKey, setDeepseekKeyState, deepseekKeyMasked,
    opencodeGoKey, setOpencodeGoKeyState, opencodeGoKeyMasked,
    opencodeZenKey, setOpencodeZenKeyState, opencodeZenKeyMasked,
    callStats, statsSort, setStatsSort, trendRange, setTrendRange, trendData,
    imageBackend, setImageBackend, comfyUIURL, comfyUIPath, comfyUIPythonPath, imageModel, setImageModel,
    imageSaveDir, setImageSaveDir, imageBackendSaving, comfyStatus, comfyBusy,
    voiceCfg, setVoiceCfg, ocrCfg, setOcrCfg, chatVoiceCfg, chatVoiceDraft, setChatVoiceDraft, chatVoiceSaving, chatVoiceSpeakers, chatVoiceOptions, chatVoiceValue,
    featureCfg, featureDraft, setFeatureDraft, featureEnabled, modelRoutes,
    portraitCfg, portraitDraft, setPortraitDraft, portraitModelOptions, portraitSaving,
    llmModels, ttsModels, sttModels, imageModels, specialtyModels, makeModels, isModelActive,
    handleTestConnection, handleRefreshModels, handleStartModel, handleSaveURL, handleToggleEngine, handleBulkToggleEngines,
    handleSaveDeepseekKey, handleSaveOpencodeGoKey, handleSaveOpencodeZenKey,
    handleResetCallStats, loadCallStats, handleToggleComfy, handleSaveImageBackend, handleSetVoiceModel, handleSetOCRModel,
    handleSaveFeature, handleToggleFeatureEnabled, handleSavePortrait, handleSaveChatVoice, handleClearChatVoice,
  }

  if (loading) {
    return (
      <div className="mc-page">
        <div className="mc-header">
          <div className="mc-title-row">
            <div className="mc-eyebrow"><ThunderboltOutlined /> Model Center</div>
            <h1 className="mc-title">模型引擎中心</h1>
            <p className="mc-subtitle">正在读取引擎、模型和调用统计</p>
          </div>
        </div>
        <div className="mc-skeleton" style={{ height: 56 }} />
        <div className="mc-skeleton" style={{ height: 46 }} />
        <div className="mc-skeleton" style={{ height: 260 }} />
      </div>
    )
  }

  return (
    <div className="mc-page">
      <header className="mc-header">
        <div className="mc-title-row">
          <div className="mc-eyebrow"><ThunderboltOutlined /> Model Center</div>
          <h1 className="mc-title">模型引擎中心</h1>
          <p className="mc-subtitle">统一管理云端与本地引擎、模型路由、语音/图片/专业模型与调用统计。</p>
        </div>
        <div className="mc-header-actions">
          {loggedIn ? (
            <>
              <span className="mc-account is-online"><CheckCircleOutlined /> xAI 已连接</span>
              <Button size="small" icon={<LogoutOutlined />} onClick={() => logout()}>退出</Button>
            </>
          ) : (
            <Button
              size="small"
              type="primary"
              icon={<LoginOutlined />}
              loading={loggingIn}
              onClick={async () => {
                setLoggingIn(true)
                try {
                  await login()
                  message.success('xAI 登录成功')
                  await loadAll()
                } catch (err: any) {
                  message.error('登录失败：' + (err?.message || err || '未知错误，请检查浏览器是否完成了 xAI 授权'))
                } finally {
                  setLoggingIn(false)
                }
              }}
            >
              登录 xAI
            </Button>
          )}
          <Button icon={<BarChartOutlined />} onClick={() => setStatsOpen(true)}>调用统计</Button>
          <Button icon={<ReloadOutlined />} onClick={loadAll}>刷新状态</Button>
        </div>
      </header>

      <ResourceMonitor />

      <nav className="mc-tabs" aria-label="模型中心导航">
        {TABS.map(navTab)}
      </nav>

      <ModelCenterContext.Provider value={ctx}>
        {category === 'overview' && <OverviewSection />}
        {category === 'llm' && <LLMSection />}
        {category === 'image' && <ImageSection />}
        {category === 'tts' && <VoiceSection />}
        {category === 'specialty' && <SpecialtySection />}
        {category === 'catalog' && <HerdsmanCatalogSection />}
        {category === 'benchmark' && <BenchmarkSection />}
        {category === 'engine' && <EngineSection />}
        {category === 'bind' && <BindSection />}
        <Drawer
          title="模型调用统计"
          open={statsOpen}
          onClose={() => setStatsOpen(false)}
          width={860}
          styles={{ body: { padding: 0 } }}
        >
          <StatsSection />
        </Drawer>
      </ModelCenterContext.Provider>
    </div>
  )
}

export default ModelCenterPage
