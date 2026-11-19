import React, { useState, useEffect } from 'react'
import { Button, Drawer, message } from 'antd'
import {
  ThunderboltOutlined, PictureOutlined, SoundOutlined, SettingOutlined, LinkOutlined,
  CheckCircleOutlined, LoginOutlined, LogoutOutlined, DashboardOutlined, ReloadOutlined, BarChartOutlined,
  AppstoreOutlined, ExperimentOutlined, PieChartOutlined, MonitorOutlined,
} from '@ant-design/icons'
import { useAppStore } from '../stores/appStore'
import { emitFrontendEvent, FRONTEND_EVENTS, OPEN_STATS_FLAG } from '../events'
import { ModelCenterContext, ModelCenterStateContext, ModelCenterActionsContext } from './modelcenter/context'
import { useModelCenterBundle } from './modelcenter/useModelCenterBundle'
import { LLMSection } from './modelcenter/LLMSection'
import { BindSection } from './modelcenter/BindSection'
import { StrategySection } from './modelcenter/StrategySection'
import { StatsSection } from './modelcenter/StatsSection'
import { ImageSection } from './modelcenter/ImageSection'
import { VoiceSection } from './modelcenter/VoiceSection'
import { EngineSection } from './modelcenter/EngineSection'
import { SpecialtySection } from './modelcenter/SpecialtySection'
import { OverviewSection } from './modelcenter/OverviewSection'
import { HerdsmanCatalogSection } from './modelcenter/HerdsmanCatalogSection'
import { BenchmarkSection } from './modelcenter/BenchmarkSection'
import { RecommendSection } from './modelcenter/RecommendSection'
import { RetrievalEvalSection } from './modelcenter/RetrievalEvalSection'
import { SchedulingSection } from './modelcenter/SchedulingSection'
import { AttributionSection } from './modelcenter/AttributionSection'
import { FEATURES, type Category } from './modelcenter/utils'
import './modelcenter/modelcenter.css'

// gaea 3.0「引擎控制台」：2 分区工作台（§2.7，2026-10 检查器合并刀）——
// 左 = 分类导航栏（v3-panel，激活项 = 主色容器 + 光条 + 光晕 orb；2026-10-27
// 瘦身重组 13 分类温和合并为 9，按概览/模型/质量与成本/接入四节分组）；
// 中 = 主区（KPI 卡行 + 引擎卡网格，各 Section 渲染；合并 tab 为纵向多段）。
// 右侧「统计与资源」检查器已并入运行监测板块（资源指标卡/引擎卡/ComfyUI
// 标记/调用趋势同源呈现），本页头部保留「详细统计」抽屉与「运行监测」跳转。
// 头部收敛为细条：板块名已在左侧指挥轨道/面包屑，仅保留账号连接状态与必要操作。
const ModelCenterPage: React.FC = () => {
  // v4.365：粗订阅改选择器
  const loggedIn = useAppStore((s) => s.loggedIn)
  const login = useAppStore((s) => s.login)
  const logout = useAppStore((s) => s.logout)
  // T6-6.4：顶层仅保留全局状态（分类/统计抽屉/登录中），
  // 42 个 useState 按 5 分类（引擎管理/调用统计/图片生成/语音模型/功能绑定）
  // 下沉到独立 Hook；各 Section 经共享 Context 消费同一份状态。
  const [category, setCategory] = useState<Category>('overview')
  const [statsOpen, setStatsOpen] = useState(false)
  const [loggingIn, setLoggingIn] = useState(false)

  // 「按模型明细」跨板跳转联动（运行监测）：已挂载（keepAlive 常驻）听事件
  // 即开抽屉；首次未挂载场景 navigate 才懒加载本页，事件已错过 → 挂载效应
  // 消费 sessionStorage 标记补开，读后即清（一次性）。事件直达=已挂载场景，
  // 发射方写入的标记已无用——监听里顺带清除，杜绝残留到 webview reload 后
  // 重挂载误开抽屉（与 OPEN_TOOL_FLAG 监听双清同款，刀6 一致性收口）。
  useEffect(() => {
    try {
      if (sessionStorage.getItem(OPEN_STATS_FLAG) === '1') {
        sessionStorage.removeItem(OPEN_STATS_FLAG)
        setStatsOpen(true)
      }
    } catch { /* sessionStorage 不可用（隐私模式等）静默跳过 */ }
    const open = () => {
      try { sessionStorage.removeItem(OPEN_STATS_FLAG) } catch { /* 同上 */ }
      setStatsOpen(true)
    }
    window.addEventListener(FRONTEND_EVENTS.OPEN_STATS_DRAWER, open)
    return () => window.removeEventListener(FRONTEND_EVENTS.OPEN_STATS_DRAWER, open)
  }, [])

  // 五 hook 组装+同步 effects+三 Context 值单源（设置中枢批 3 抽出
  // useModelCenterBundle，与设置页 ModelSettingsProvider 共享同一份组装）
  const { ctx, stateCtx, actionsCtx, engine, bind } = useModelCenterBundle(category, statsOpen, {
    setCategory,
    openStatsDrawer: () => setStatsOpen(true),
  })

  // 模型中心左侧分类切换后，把主区滚动容器回到顶部，避免上一分类的滚动位置
  // 残留，导致功能绑定等页面顶部的控件落在可视区域外、看起来像下拉框点不开。
  useEffect(() => {
    const scroller = document.querySelector<HTMLElement>('.mc-main') || document.querySelector<HTMLElement>('.ant-layout-content')
    if (scroller && typeof scroller.scrollTo === 'function') {
      scroller.scrollTo({ top: 0, behavior: 'auto' })
    }
  }, [category])

  // 2026-10-27 瘦身重组：13 分类温和合并为 9（llm+specialty+recommend→models、
  // benchmark+retrieval→eval、strategy+attribution→governance），rail 按语义分四节
  // （概览/模型/质量与成本/接入）。Section 组件全部保留，合并 tab 内纵向多段渲染
  // （与 engine tab 的 Scheduling+Engine 两段先例同构）。
  const TAB_GROUPS: { label: string | null; tabs: { key: Category; icon: React.ReactNode; label: string }[] }[] = [
    { label: '概览', tabs: [
      { key: 'overview', icon: <DashboardOutlined />, label: '总览' },
    ] },
    { label: '模型', tabs: [
      { key: 'models', icon: <ThunderboltOutlined />, label: '模型一览' },
      { key: 'catalog', icon: <AppstoreOutlined />, label: '模型库' },
      { key: 'image', icon: <PictureOutlined />, label: '图片生成' },
      { key: 'tts', icon: <SoundOutlined />, label: '语音模型' },
    ] },
    { label: '质量与成本', tabs: [
      { key: 'eval', icon: <ExperimentOutlined />, label: '测评中心' },
      { key: 'governance', icon: <PieChartOutlined />, label: '策略与成本' },
    ] },
    { label: '接入', tabs: [
      { key: 'bind', icon: <LinkOutlined />, label: '功能绑定' },
      { key: 'engine', icon: <SettingOutlined />, label: '引擎管理' },
    ] },
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
          <Button
            size="small"
            icon={<MonitorOutlined />}
            onClick={() => emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: 'runtime', crossSpace: true })}
          >
            运行监测
          </Button>
          <Button size="small" icon={<ReloadOutlined />} onClick={engine.loadAll}>刷新状态</Button>
        </div>
      </header>

      <ModelCenterStateContext.Provider value={stateCtx}>
      <ModelCenterActionsContext.Provider value={actionsCtx}>
      <ModelCenterContext.Provider value={ctx}>
        <div className="mc-workbench">
          {/* 左：分类导航栏（v3-panel；激活项 = 主色容器 + 左缘光条 + 光晕 orb） */}
          <nav className="v3-panel mc-rail" aria-label="引擎控制台分类导航">
            <div className="v3-panel-head">
              <span className="v3-panel-title">引擎控制台</span>
            </div>
            <div className="mc-rail-nav">
              {TAB_GROUPS.map(group => (
                <React.Fragment key={group.label ?? group.tabs[0].key}>
                  <div className="mc-rail-group" aria-hidden="true">{group.label}</div>
                  {group.tabs.map(tab => (
                    <button
                      type="button"
                      key={tab.key}
                      className={`mc-rail-item${category === tab.key ? ' is-active' : ''}`}
                      // aria-selected 只允许出现在 tab/radio/option/gridcell 等角色上；此处
                      // 分区按钮无 tablist 结构，用 aria-pressed（合法且语义同为「已选中」）
                      aria-pressed={category === tab.key}
                      onClick={() => setCategory(tab.key)}
                    >
                      <span className="mc-rail-icon" aria-hidden="true">{tab.icon}</span>
                      <span className="mc-rail-label">{tab.label}</span>
                      <span className="mc-rail-orb" aria-hidden="true" />
                    </button>
                  ))}
                </React.Fragment>
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
            {category === 'models' && (
              <>
                <LLMSection />
                <SpecialtySection />
                <RecommendSection />
              </>
            )}
            {category === 'image' && <ImageSection />}
            {category === 'tts' && <VoiceSection />}
            {category === 'catalog' && <HerdsmanCatalogSection />}
            {category === 'eval' && (
              <>
                <BenchmarkSection />
                <RetrievalEvalSection />
              </>
            )}
            {category === 'engine' && (
              <>
                <SchedulingSection />
                <EngineSection />
              </>
            )}
            {category === 'bind' && <BindSection />}
            {category === 'governance' && (
              <>
                <StrategySection />
                <AttributionSection />
              </>
            )}
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
        </div>
      </ModelCenterContext.Provider>
      </ModelCenterActionsContext.Provider>
      </ModelCenterStateContext.Provider>
    </div>
  )
}

export default ModelCenterPage
