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
import { HerdsmanCatalogSection } from './modelcenter/HerdsmanCatalogSection'
import { BenchmarkSection } from './modelcenter/BenchmarkSection'
import { RetrievalEvalSection } from './modelcenter/RetrievalEvalSection'
import { SchedulingSection } from './modelcenter/SchedulingSection'
import { InspectorPanel } from './modelcenter/InspectorPanel'
import { useEngineState } from './modelcenter/hooks/useEngineState'
import { useStatsState } from './modelcenter/hooks/useStatsState'
import { useImageState } from './modelcenter/hooks/useImageState'
import { useVoiceState } from './modelcenter/hooks/useVoiceState'
import { useBindState } from './modelcenter/hooks/useBindState'
import { FEATURES, type Category } from './modelcenter/utils'
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

// gaea 3.0「引擎控制台」：3 分区工作台（§2.7）——
// 左 = 分类导航栏（v3-panel，激活项 = 主色容器 + 光条 + 光晕 orb）；
// 中 = 主区（KPI 卡行 + 引擎卡网格，各 Section 渲染）；
// 右 = 统计/资源检查器（v3-panel，可折叠，InspectorPanel）。
// 头部收敛为细条：板块名已在左侧指挥轨道/面包屑，仅保留账号连接状态与必要操作。
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

  // 模型中心左侧分类切换后，把主区滚动容器回到顶部，避免上一分类的滚动位置
  // 残留，导致功能绑定等页面顶部的控件落在可视区域外、看起来像下拉框点不开。
  useEffect(() => {
    const scroller = document.querySelector<HTMLElement>('.mc-main') || document.querySelector<HTMLElement>('.ant-layout-content')
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
    glmKey: engine.glmKey, setGlmKeyState: engine.setGlmKeyState, glmKeyMasked: engine.glmKeyMasked,
    opencodeGoKey: engine.opencodeGoKey, setOpencodeGoKeyState: engine.setOpencodeGoKeyState, opencodeGoKeyMasked: engine.opencodeGoKeyMasked,
    opencodeZenKey: engine.opencodeZenKey, setOpencodeZenKeyState: engine.setOpencodeZenKeyState, opencodeZenKeyMasked: engine.opencodeZenKeyMasked,
    callStats: stats.callStats, statsSort: stats.statsSort, setStatsSort: stats.setStatsSort,
    loadError: stats.loadError,
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
    handleSaveGlmKey: engine.handleSaveGlmKey,
    handleSaveOpencodeGoKey: engine.handleSaveOpencodeGoKey,
    handleSaveOpencodeZenKey: engine.handleSaveOpencodeZenKey,
    settingGlmEndpoint: engine.settingGlmEndpoint,
    handleSetGlmEndpoint: engine.handleSetGlmEndpoint,
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

  // 左栏「绑定状态」摘要：已绑定（引擎+模型齐全）的功能数
  const boundFeatures = FEATURES.filter(f => {
    const c = bind.featureCfg[f.key]
    return !!c?.engine && !!c?.model
  }).length

  if (engine.loading) {
    return (
      <div className="mc-page">
        <div className="mc-skeleton" style={{ height: 34 }} />
        <div className="mc-workbench">
          <div className="v3-panel mc-rail">
            <div className="mc-skeleton" style={{ height: 300 }} />
          </div>
          <div className="mc-main">
            <div className="mc-skeleton" style={{ height: 52 }} />
            <div className="mc-skeleton" style={{ height: 120 }} />
            <div className="mc-skeleton" style={{ height: 220 }} />
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="mc-page">
      {/* 细条头部：板块名已在左侧指挥轨道/面包屑，仅保留账号连接状态与必要操作 */}
      <header className="mc-header">
        <span className={`mc-account${loggedIn ? ' is-online' : ''}`}>
          {loggedIn ? <CheckCircleOutlined /> : <LoginOutlined />}
          {loggedIn ? 'xAI 已连接' : '未登录 xAI'}
        </span>
        <div className="mc-header-actions">
          {loggedIn ? (
            <Button size="small" icon={<LogoutOutlined />} onClick={() => logout()}>退出</Button>
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
          <Button size="small" icon={<BarChartOutlined />} onClick={() => setStatsOpen(true)}>详细统计</Button>
          <Button size="small" icon={<ReloadOutlined />} onClick={engine.loadAll}>刷新状态</Button>
        </div>
      </header>

      <ModelCenterContext.Provider value={ctx}>
        <div className="mc-workbench">
          {/* 左：分类导航栏（v3-panel；激活项 = 主色容器 + 左缘光条 + 光晕 orb） */}
          <nav className="v3-panel mc-rail" aria-label="引擎控制台分类导航">
            <div className="v3-panel-head">
              <span className="v3-panel-title">引擎控制台</span>
            </div>
            <div className="mc-rail-nav">
              {TABS.map(tab => (
                <button
                  type="button"
                  key={tab.key}
                  className={`mc-rail-item${category === tab.key ? ' is-active' : ''}`}
                  aria-selected={category === tab.key}
                  onClick={() => setCategory(tab.key)}
                >
                  <span className="mc-rail-icon" aria-hidden="true">{tab.icon}</span>
                  <span className="mc-rail-label">{tab.label}</span>
                  <span className="mc-rail-orb" aria-hidden="true" />
                </button>
              ))}
            </div>
            <div className="mc-rail-foot">
              <span className="mc-rail-foot-title">功能绑定</span>
              <button
                type="button"
                className="mc-rail-foot-item"
                onClick={() => setCategory('bind')}
                aria-label={`查看功能绑定，${boundFeatures} / ${FEATURES.length} 已绑定`}
              >
                <span className="mc-rail-foot-dot" aria-hidden="true" />
                {boundFeatures} / {FEATURES.length} 已绑定
              </button>
            </div>
          </nav>

          {/* 中：主区（各分类 Section：KPI 卡行 + 引擎卡网格等） */}
          <main className="mc-main" aria-label="引擎控制台内容">
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
          </main>

          {/* 右：统计/资源检查器（v3-panel，可折叠） */}
          <InspectorPanel />
        </div>
      </ModelCenterContext.Provider>
    </div>
  )
}

export default ModelCenterPage
