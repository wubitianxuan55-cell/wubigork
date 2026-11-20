import React, { useState, useEffect } from 'react'
import { Button } from 'antd'
import {
  SettingOutlined, LinkOutlined, ThunderboltOutlined,
  ReloadOutlined, MonitorOutlined,
} from '@ant-design/icons'
import { emitFrontendEvent, FRONTEND_EVENTS } from '../events'
import { ModelCenterContext, ModelCenterStateContext, ModelCenterActionsContext } from './modelcenter/context'
import { useModelCenterBundle } from './modelcenter/useModelCenterBundle'
import { ModelGroupNav } from './modelcenter/ModelGroupNav'
import { LLMSection } from './modelcenter/LLMSection'
import { ImageSection } from './modelcenter/ImageSection'
import { VoiceSection } from './modelcenter/VoiceSection'
import { SpecialtySection } from './modelcenter/SpecialtySection'
import { BindSection } from './modelcenter/BindSection'
import { EngineSection } from './modelcenter/EngineSection'
import type { Category } from './modelcenter/utils'
import './modelcenter/modelcenter.css'

// 模型中心首页：只保留三条高频操作入口——模型一览、功能绑定、引擎管理。
// 运行概览、模型库、测评、成本归因与运行策略统一迁入“设置 → 模型”。
const ModelCenterPage: React.FC = () => {
  // T6-6.4：顶层仅保留全局状态（分类/统计抽屉），
  // 42 个 useState 按 5 分类（引擎管理/调用统计/图片生成/语音模型/功能绑定）
  // 下沉到独立 Hook；各 Section 经共享 Context 消费同一份状态。
  const [category, setCategory] = useState<Category>('models')

  // 五 hook 组装+同步 effects+三 Context 值单源（设置中枢批 3 抽出
  // useModelCenterBundle，与设置页 ModelSettingsProvider 共享同一份组装）
  const { ctx, stateCtx, actionsCtx, engine } = useModelCenterBundle(category, false, {
    setCategory,
    openStatsDrawer: () => {},
  })

  // 模型中心左侧分类切换后，把主区滚动容器回到顶部，避免上一分类的滚动位置
  // 残留，导致功能绑定等页面顶部的控件落在可视区域外、看起来像下拉框点不开。
  useEffect(() => {
    const scroller = document.querySelector<HTMLElement>('.mc-main') || document.querySelector<HTMLElement>('.ant-layout-content')
    if (scroller && typeof scroller.scrollTo === 'function') {
      scroller.scrollTo({ top: 0, behavior: 'auto' })
    }
  }, [category])

  const TABS: { key: Category; icon: React.ReactNode; label: string }[] = [
    { key: 'models', icon: <ThunderboltOutlined />, label: '模型一览' },
    { key: 'bind', icon: <LinkOutlined />, label: '功能绑定' },
    { key: 'engine', icon: <SettingOutlined />, label: '引擎管理' },
  ]

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
      {/* 细条头部：低频账号与模型维护入口不再占用主路径，仅保留全局动作 */}
      <header className="mc-header">
        <div className="mc-header-actions">
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
          {/* 左：三条高频一级入口 */}
          <nav className="v3-panel mc-rail" aria-label="模型中心一级导航">
            <div className="v3-panel-head">
              <span className="v3-panel-title">模型中心</span>
            </div>
            <div className="mc-rail-nav">
              {TABS.map(tab => (
                <button
                  type="button"
                  key={tab.key}
                  className={`mc-rail-item${category === tab.key ? ' is-active' : ''}`}
                  // 一级分区按钮无 tablist 结构，用 aria-pressed 表达当前态。
                  aria-pressed={category === tab.key}
                  onClick={() => setCategory(tab.key)}
                >
                  <span className="mc-rail-icon" aria-hidden="true">{tab.icon}</span>
                  <span className="mc-rail-label">{tab.label}</span>
                  <span className="mc-rail-orb" aria-hidden="true" />
                </button>
              ))}
            </div>
          </nav>

          {/* 中：主区 */}
          <main className="mc-main" aria-label="模型中心内容">
            {category === 'models' && (
              <>
                <ModelGroupNav />
                <div id="mc-model-group-llm" className="mc-model-group-anchor">
                  <LLMSection />
                </div>
                <div id="mc-model-group-image" className="mc-model-group-anchor">
                  <ImageSection catalogMode />
                </div>
                <div id="mc-model-group-voice" className="mc-model-group-anchor">
                  <VoiceSection catalogMode />
                </div>
                <div id="mc-model-group-specialty" className="mc-model-group-anchor">
                  <SpecialtySection />
                </div>
              </>
            )}
            {category === 'bind' && <BindSection />}
            {category === 'engine' && <EngineSection />}
          </main>
        </div>
      </ModelCenterContext.Provider>
      </ModelCenterActionsContext.Provider>
      </ModelCenterStateContext.Provider>
    </div>
  )
}

export default ModelCenterPage
