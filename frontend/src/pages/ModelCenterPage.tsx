import React, { useState, useEffect } from 'react'
import { Button, Drawer, message } from 'antd'
import {
  ThunderboltOutlined, PictureOutlined, SoundOutlined, SettingOutlined, LinkOutlined,
  CheckCircleOutlined, LoginOutlined, LogoutOutlined, DatabaseOutlined, DashboardOutlined, ReloadOutlined, BarChartOutlined,
  AppstoreOutlined, ExperimentOutlined, SearchOutlined,
} from '@ant-design/icons'
import { useAppStore } from '../stores/appStore'
import { ModelCenterContext, type ModelCenterContextValue } from './modelcenter/context'
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
import { RetrievalEvalSection } from './modelcenter/RetrievalEvalSection'
import { SchedulingSection } from './modelcenter/SchedulingSection'
import { useEngineState } from './modelcenter/hooks/useEngineState'
import { useStatsState } from './modelcenter/hooks/useStatsState'
import { useImageState } from './modelcenter/hooks/useImageState'
import { useVoiceState } from './modelcenter/hooks/useVoiceState'
import { useBindState } from './modelcenter/hooks/useBindState'
import { type Category } from './modelcenter/utils'
import './modelcenter/modelcenter.css'

/** 订阅 runtime 事件并返回退订函数（原生 runtime 返回退订；HTTP polyfill 返回 void，安全兜底） */
function runtimeOn(event: string, handler: (data: unknown) => void): (() => void) | undefined {
  try {
    const on = window.runtime?.EventsOn as ((e: string, h: (d: unknown) => void) => (() => void) | void) | undefined
    const res = on?.(event, handler)
    return typeof res === 'function' ? res : undefined
  } catch {
    return undefined
  }
}

const ModelCenterPage: React.FC = () => {
  const { loggedIn, login, logout } = useAppStore()
  // T6-6.4：顶层仅保留全局状态（分类/统计抽屉/登录中），
  // 42 个 useState 按 5 分类（引擎管理/调用统计/图片生成/语音模型/功能绑定）
  // 下沉到独立 Hook；各 Section 经共享 Context 消费同一份状态。
  const [category, setCategory] = useState<Category>('overview')
  const [statsOpen, setStatsOpen] = useState(false)
  const [loggingIn, setLoggingIn] = useState(false)

  const engine = useEngineState(category)
  const stats = useStatsState(statsOpen)
  const image = useImageState()
  const voice = useVoiceState()
  const bind = useBindState(engine.engines)

  // 模型中心左侧分类切换后，把外层滚动容器回到顶部，避免上一分类的滚动位置
  // 残留，导致功能绑定等页面顶部的控件落在可视区域外、看起来像下拉框点不开。
  useEffect(() => {
    const scroller = document.querySelector('.ant-layout-content')
    if (scroller && typeof scroller.scrollTo === 'function') {
      scroller.scrollTo({ top: 0, behavior: 'auto' })
    }
  }, [category])

  // 同步：其他页面（FeatureModelBar 等）修改绑定后，本面板即时刷新
  const { loadFeatureCfg: loadFeatureCfgSync, refreshRoutes: refreshRoutesSync } = bind
  useEffect(() => {
    const reload = () => { loadFeatureCfgSync(); refreshRoutesSync() }
    const unsub = runtimeOn('feature-model-changed', reload)
    return () => {
      try { unsub?.() } catch (_) {}
    }
  }, [loadFeatureCfgSync, refreshRoutesSync])

  // 同步：其他页面切换活跃模型/语音模型后，本面板即时刷新
  const { loadAll: loadAllSync } = engine
  const { loadVoiceCfg: loadVoiceCfgSync } = voice
  useEffect(() => {
    const reload = () => { loadAllSync(); refreshRoutesSync() }
    const reloadVoice = () => { loadVoiceCfgSync() }
    const unsub1 = runtimeOn('model-changed', reload)
    const unsub2 = runtimeOn('voice-model-changed', reloadVoice)
    // 本地 TTS 服务就绪/失败后同步刷新（CosyVoice2 异步启动约 1–2 分钟）
    const unsub3 = runtimeOn('tts-service-status', reload)
    return () => {
      try { unsub1?.() } catch (_) {}
      try { unsub2?.() } catch (_) {}
      try { unsub3?.() } catch (_) {}
    }
  }, [loadAllSync, loadVoiceCfgSync, refreshRoutesSync])

  const ctx: ModelCenterContextValue = {
    category, setCategory,
    engines: engine.engines, engineStatuses: engine.engineStatuses,
    editingURLs: engine.editingURLs, setEditingURLs: engine.setEditingURLs,
    savingEngine: engine.savingEngine, testingEngine: engine.testingEngine,
    activeEngine: engine.activeEngine, activeModel: engine.activeModel,
    deepseekKey: engine.deepseekKey, setDeepseekKeyState: engine.setDeepseekKeyState, deepseekKeyMasked: engine.deepseekKeyMasked,
    opencodeGoKey: engine.opencodeGoKey, setOpencodeGoKeyState: engine.setOpencodeGoKeyState, opencodeGoKeyMasked: engine.opencodeGoKeyMasked,
    opencodeZenKey: engine.opencodeZenKey, setOpencodeZenKeyState: engine.setOpencodeZenKeyState, opencodeZenKeyMasked: engine.opencodeZenKeyMasked,
    callStats: stats.callStats, statsSort: stats.statsSort, setStatsSort: stats.setStatsSort,
    trendRange: stats.trendRange, setTrendRange: stats.setTrendRange, trendData: stats.trendData,
    imageBackend: image.imageBackend, setImageBackend: image.setImageBackend,
    comfyUIURL: image.comfyUIURL, comfyUIPath: image.comfyUIPath, comfyUIPythonPath: image.comfyUIPythonPath,
    imageModel: image.imageModel, setImageModel: image.setImageModel,
    imageSaveDir: image.imageSaveDir, setImageSaveDir: image.setImageSaveDir,
    imageBackendSaving: image.imageBackendSaving, comfyStatus: image.comfyStatus, comfyBusy: image.comfyBusy,
    voiceCfg: voice.voiceCfg, setVoiceCfg: voice.setVoiceCfg,
    ocrCfg: voice.ocrCfg, setOcrCfg: voice.setOcrCfg,
    chatVoiceCfg: voice.chatVoiceCfg, chatVoiceDraft: voice.chatVoiceDraft, setChatVoiceDraft: voice.setChatVoiceDraft,
    chatVoiceSaving: voice.chatVoiceSaving, chatVoiceSpeakers: voice.chatVoiceSpeakers,
    chatVoiceOptions: voice.chatVoiceOptions, chatVoiceValue: voice.chatVoiceValue,
    featureCfg: bind.featureCfg, featureDraft: bind.featureDraft, setFeatureDraft: bind.setFeatureDraft,
    featureEnabled: bind.featureEnabled, modelRoutes: bind.modelRoutes,
    portraitCfg: bind.portraitCfg, portraitDraft: bind.portraitDraft, setPortraitDraft: bind.setPortraitDraft,
    portraitModelOptions: bind.portraitModelOptions, portraitSaving: bind.portraitSaving,
    llmModels: engine.llmModels, ttsModels: engine.ttsModels, sttModels: engine.sttModels,
    imageModels: engine.imageModels, specialtyModels: engine.specialtyModels,
    makeModels: engine.makeModels, isModelActive: engine.isModelActive,
    handleTestConnection: engine.handleTestConnection,
    handleRefreshModels: engine.handleRefreshModels,
    handleStartModel: engine.handleStartModel,
    handleSaveURL: engine.handleSaveURL,
    handleToggleEngine: engine.handleToggleEngine,
    handleBulkToggleEngines: engine.handleBulkToggleEngines,
    handleSaveDeepseekKey: engine.handleSaveDeepseekKey,
    handleSaveOpencodeGoKey: engine.handleSaveOpencodeGoKey,
    handleSaveOpencodeZenKey: engine.handleSaveOpencodeZenKey,
    handleResetCallStats: stats.handleResetCallStats, loadCallStats: stats.loadCallStats,
    handleToggleComfy: image.handleToggleComfy, handleSaveImageBackend: image.handleSaveImageBackend,
    handleSetVoiceModel: voice.handleSetVoiceModel, handleSetOCRModel: voice.handleSetOCRModel,
    handleSaveFeature: bind.handleSaveFeature, handleToggleFeatureEnabled: bind.handleToggleFeatureEnabled,
    handleSavePortrait: bind.handleSavePortrait,
    handleSaveChatVoice: voice.handleSaveChatVoice, handleClearChatVoice: voice.handleClearChatVoice,
  }

  const TABS: { key: Category; icon: React.ReactNode; label: string }[] = [
    { key: 'overview', icon: <DashboardOutlined />, label: '总览' },
    { key: 'llm', icon: <ThunderboltOutlined />, label: '语言模型' },
    { key: 'image', icon: <PictureOutlined />, label: '图片生成' },
    { key: 'tts', icon: <SoundOutlined />, label: '语音模型' },
    { key: 'specialty', icon: <DatabaseOutlined />, label: '专业模型' },
    { key: 'catalog', icon: <AppstoreOutlined />, label: '模型库' },
    { key: 'benchmark', icon: <ExperimentOutlined />, label: '受控测评' },
    { key: 'retrieval', icon: <SearchOutlined />, label: '检索质量' },
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

  if (engine.loading) {
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
                  await engine.loadAll()
                } catch (err: unknown) {
                  message.error('登录失败：' + (err instanceof Error ? err.message : (typeof err === 'string' ? err : '未知错误，请检查浏览器是否完成了 xAI 授权')))
                } finally {
                  setLoggingIn(false)
                }
              }}
            >
              登录 xAI
            </Button>
          )}
          <Button icon={<BarChartOutlined />} onClick={() => setStatsOpen(true)}>调用统计</Button>
          <Button icon={<ReloadOutlined />} onClick={engine.loadAll}>刷新状态</Button>
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
        {category === 'retrieval' && <RetrievalEvalSection />}
        {category === 'engine' && (
          <>
            <SchedulingSection />
            <EngineSection />
          </>
        )}
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
