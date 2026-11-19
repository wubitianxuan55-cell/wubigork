import React from 'react'
import { useModelCenterBundle } from './useModelCenterBundle'
import { ModelCenterStateContext, ModelCenterActionsContext, ModelCenterContext } from './context'

/**
 * ModelSettingsProvider — 设置页专用模型中心 Context 宿主（设置中枢批 3）。
 * 组装走 useModelCenterBundle 单源（与 ModelCenterPage 同一份），设置页各分组
 * （模型/绘梦/聊天语音）用它包裹迁入的 Section；同键同源，跨页变更经
 * model-changed/feature-model-changed 等既有事件双宿主互同步。
 * category 固定 'engine'（仅驱动 hooks 内刷新节奏，无 rail 语义）；统计抽屉
 * 归模型中心页，这里 openStatsDrawer 为 noop。
 */
const ModelSettingsProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { stateCtx, actionsCtx, ctx } = useModelCenterBundle('engine', false, {
    setCategory: () => {},
    openStatsDrawer: () => {},
  })
  return (
    <ModelCenterStateContext.Provider value={stateCtx}>
      <ModelCenterActionsContext.Provider value={actionsCtx}>
        <ModelCenterContext.Provider value={ctx}>
          {children}
        </ModelCenterContext.Provider>
      </ModelCenterActionsContext.Provider>
    </ModelCenterStateContext.Provider>
  )
}

export default ModelSettingsProvider
