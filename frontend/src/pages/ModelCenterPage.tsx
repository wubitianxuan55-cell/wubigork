import React, { useState, useEffect, useCallback, useMemo } from 'react'
import { Typography, Card, Button, Space, Tag, message, Spin } from 'antd'
import {
  ThunderboltOutlined, PictureOutlined, SoundOutlined, SettingOutlined, LinkOutlined,
  CloudOutlined, CheckCircleOutlined, LoginOutlined, LogoutOutlined,
} from '@ant-design/icons'
import { useAppStore } from '../stores/appStore'
import { C } from '../utils/theme'
import * as App from '../../wailsjs/go/app/App'
import {
  getEngines, saveEngine, testEngineConnection,
  refreshEngineModels, setEngineDefaultModel,
  setActiveEngine, getActiveEngine, setDeepseekKey, getDeepseekKeyStatus,
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
import {
  FEATURES, XAI_VOICES, classifyModel, kindOf, localTTSDefaultVoice, localTTSFallbackVoices,
  type Category, type ModelCardData, type ModelKind,
} from './modelcenter/utils'
import { type StatsSort, type TrendDatum, type TrendRange } from './modelcenter/charts'

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
  const [voiceCfg, setVoiceCfg] = useState<VoiceCfg>({ stt: { engine: '', model: '' }, llm: { engine: '', model: '' }, tts: { engine: '', model: '', voice: '' } })
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
    return () => {
      try { if (typeof unsub1 === 'function') unsub1() } catch (_) {}
      try { if (typeof unsub2 === 'function') unsub2() } catch (_) {}
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
    const kind = kindOf(card)
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
      message.success(`已启动${card.modelName}`)
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
    (engine.models || []).map(m => ({ modelId: m.id, modelName: m.id, engineId: engine.id, engineName: engine.name, engineType: engine.type, engineEnabled: engine.enabled, status: m.status || 'running', kind: m.kind || '' }))

  const allModels = engines.filter(e => e.enabled).flatMap(e => makeModels(e))
  const llmModels = allModels.filter(m => kindOf(m) === 'llm')
  const ttsModels = allModels.filter(m => kindOf(m) === 'tts')
  const sttModels = allModels.filter(m => kindOf(m) === 'stt')
  const imageModels = allModels.filter(m => kindOf(m) === 'image')

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
    const imgs = (eng?.models || []).filter(m => ((m.kind as ModelKind) || classifyModel(m.id)) === 'image')
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

  const sidebarBtn = (key: Category, icon: React.ReactNode, label: string) => (
    <Button type={category === key ? 'primary' : 'text'} icon={icon as any}
      onClick={() => setCategory(key)}
      style={{ justifyContent: 'flex-start', textAlign: 'left', borderRadius: 8, color: category === key ? '#fff' : C('color-text-secondary'), background: category === key ? C('color-primary') : 'transparent', fontWeight: category === key ? 500 : 400, padding: '8px 14px', height: 38 }}>
      {label}
    </Button>
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
    voiceCfg, setVoiceCfg, chatVoiceCfg, chatVoiceDraft, setChatVoiceDraft, chatVoiceSaving, chatVoiceSpeakers, chatVoiceOptions, chatVoiceValue,
    featureCfg, featureDraft, setFeatureDraft, featureEnabled, modelRoutes,
    portraitCfg, portraitDraft, setPortraitDraft, portraitModelOptions, portraitSaving,
    llmModels, ttsModels, sttModels, imageModels, makeModels, isModelActive,
    handleTestConnection, handleRefreshModels, handleStartModel, handleSaveURL, handleToggleEngine,
    handleSaveDeepseekKey, handleSaveOpencodeGoKey, handleSaveOpencodeZenKey,
    handleResetCallStats, loadCallStats, handleToggleComfy, handleSaveImageBackend, handleSetVoiceModel,
    handleSaveFeature, handleToggleFeatureEnabled, handleSavePortrait, handleSaveChatVoice, handleClearChatVoice,
  }

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
          <ModelCenterContext.Provider value={ctx}>
            {/* XAI 账号卡片 */}
            {category !== 'engine' && (
              <Card style={{ marginBottom: 20, background: loggedIn ? 'linear-gradient(135deg, rgba(52,211,153,0.06), rgba(16,185,129,0.03))' : 'linear-gradient(135deg, rgba(99,102,241,0.08), rgba(37,99,235,0.04))', border: loggedIn ? '1px solid rgba(52,211,153,0.25)' : '1px solid rgba(99,102,241,0.2)', borderRadius: 12 }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Space size={12}>
                    <div style={{ width: 36, height: 36, borderRadius: 10, background: loggedIn ? 'linear-gradient(135deg, #34d399, #10b981)' : 'linear-gradient(135deg, #6366f1, #2563eb)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                      {loggedIn ? <CheckCircleOutlined style={{ fontSize: 18, color: '#fff' }} /> : <CloudOutlined style={{ fontSize: 18, color: '#fff' }} />}
                    </div>
                    <div>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>{loggedIn ? 'xAI 已连接' : 'xAI 账号'}</Typography.Text><br />
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

            {category === 'llm' && <LLMSection />}
            {category === 'bind' && <BindSection />}
            {category === 'stats' && <StatsSection />}
            {category === 'image' && <ImageSection />}
            {category === 'tts' && <VoiceSection />}
            {category === 'engine' && <EngineSection />}
          </ModelCenterContext.Provider>
        </div>
      </div>
    </div>
  )
}

export default ModelCenterPage
