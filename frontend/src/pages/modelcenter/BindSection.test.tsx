import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ModelCenterActionsContext, ModelCenterContext, ModelCenterStateContext, type ModelCenterContextValue } from './context'
import { BindSection } from './BindSection'

function renderBind(overrides: Partial<ModelCenterContextValue> = {}) {
  const value = {
    engines: [
      { id: 'herdsman', name: 'Herdsman', enabled: true, label: 'Herdsman 本地', type: 'herdsman' },
      { id: 'xai', name: 'xAI', enabled: true, label: 'xAI 云端', type: 'xai' },
    ],
    featureCfg: { chat: { engine: '', model: '' } },
    featureDraft: { chat: { engine: '', model: '' } },
    featureEnabled: { chat: true },
    modelRoutes: {},
    visionRoute: null,
    chatVoiceCfg: { engine: '', model: '' },
    chatVoiceDraft: { engine: '', model: '' },
    chatVoiceSaving: false,
    chatVoiceSpeakers: [],
    chatVoiceOptions: [],
    chatVoiceValue: undefined,
    voiceCfg: { stt: { engine: '', model: '' }, llm: { engine: '', model: '' }, tts: { engine: '', model: '', voice: '' } },
    setVoiceCfg: () => {},
    portraitCfg: { backend: '', model: '' },
    portraitDraft: { backend: '', model: '' },
    portraitModelOptions: [],
    portraitSaving: false,
    sinImageCfg: { backend: '', model: '' },
    sinImageDraft: { backend: '', model: '' },
    sinImageModelOptions: [{ label: '跟随全局', value: '' }],
    sinImageSaving: false,
    llmModels: [
      { engineId: 'herdsman', modelId: 'qwen3-8b', modelName: 'qwen3-8b' },
      { engineId: 'xai', modelId: 'grok-4.20', modelName: 'grok-4.20' },
    ],
    ttsModels: [],
    setFeatureDraft: () => {},
    setChatVoiceDraft: () => {},
    setPortraitDraft: () => {},
    setSinImageDraft: () => {},
    handleSaveFeature: () => {},
    handleToggleFeatureEnabled: () => {},
    handleSaveChatVoice: () => {},
    handleClearChatVoice: () => {},
    handleSavePortrait: () => {},
    handleSaveSinImage: () => {},
    handleRefreshModels: () => Promise.resolve(),
    testingEngine: null,
    ...overrides,
  } as unknown as ModelCenterContextValue
  return render(
    <ModelCenterStateContext.Provider value={value}><ModelCenterActionsContext.Provider value={value}><ModelCenterContext.Provider value={value}>
      <BindSection />
    </ModelCenterContext.Provider></ModelCenterActionsContext.Provider></ModelCenterStateContext.Provider>,
  )
}

describe('BindSection', () => {
  it('renders feature engine select with enabled engine options', () => {
    const { container } = renderBind()
    const selects = container.querySelectorAll('.ant-select')
    expect(selects.length).toBeGreaterThanOrEqual(2)
    expect(screen.getByText('功能模型绑定')).toBeTruthy()
  })

  it('opens engine dropdown via body portal', async () => {
    renderBind()
    const selects = document.querySelectorAll('.ant-select-selector')
    expect(selects.length).toBeGreaterThan(0)
    fireEvent.mouseDown(selects[0])
    await waitFor(() => {
      expect(document.querySelector('.ant-select-dropdown')).toBeTruthy()
    })
    expect(document.body.contains(document.querySelector('.ant-select-dropdown'))).toBe(true)
  })

  it('fires engine change when selecting an option', async () => {
    const setFeatureDraft = vi.fn()
    renderBind({ setFeatureDraft })
    const selects = document.querySelectorAll('.ant-select-selector')
    fireEvent.mouseDown(selects[0])
    await waitFor(() => {
      expect(document.querySelector('.ant-select-dropdown')).toBeTruthy()
    })
    const option = await screen.findByText('Herdsman 本地')
    fireEvent.click(option)
    expect(setFeatureDraft).toHaveBeenCalled()
  })

  // 内置引擎种子不预置模型清单：选中 models 为空的引擎时自动拉取一次，
  // 已有清单的引擎不重复拉取。
  it('auto-refreshes models when selected engine has none', async () => {
    const handleRefreshModels = vi.fn().mockResolvedValue(undefined)
    renderBind({
      handleRefreshModels,
      engines: [
        { id: 'strata', name: 'Strata', enabled: true, label: 'Strata 本地', type: 'strata', base_url: 'http://127.0.0.1:8091/v1', default_model: '', models: [] },
        { id: 'herdsman', name: 'Herdsman', enabled: true, label: 'Herdsman 本地', type: 'herdsman', base_url: 'http://localhost:8080/v1', default_model: '', models: [{ id: 'qwen3-8b', owned_by: 'herdsman', status: 'running' }] },
      ],
    })
    const selects = document.querySelectorAll('.ant-select-selector')
    fireEvent.mouseDown(selects[0])
    const option = await screen.findByText('Strata 本地')
    fireEvent.click(option)
    await waitFor(() => {
      expect(handleRefreshModels).toHaveBeenCalledWith('strata')
    })
    // 已有模型清单的引擎不再触发
    fireEvent.mouseDown(selects[0])
    const opt2 = await screen.findByText('Herdsman 本地')
    fireEvent.click(opt2)
    expect(handleRefreshModels).not.toHaveBeenCalledWith('herdsman')
  })

  // v4.388 原罪插图卡：默认跟随全局；绑定后端选择走 setSinImageDraft、保存走 handleSaveSinImage。
  it('renders sin illustration card and wires draft/save (v4.388)', async () => {
    const setSinImageDraft = vi.fn()
    const handleSaveSinImage = vi.fn()
    renderBind({ setSinImageDraft, handleSaveSinImage })

    expect(screen.getByText('原罪插图')).toBeTruthy()
    expect(screen.getByText('跟随全局')).toBeTruthy()

    // 卡内「绑定插图后端」按钮触发保存动作。
    fireEvent.click(screen.getByText('绑定插图后端'))
    expect(handleSaveSinImage).toHaveBeenCalled()

    // 已绑定态显示生效后端/模型。
    renderBind({ sinImageCfg: { backend: 'herdsman', model: 'qwen-image' } })
    expect(screen.getByText('herdsman / qwen-image')).toBeTruthy()
  })

  // 办公识图功能绑定卡（不再固定于本地默认视觉模型）：卡渲染、未绑定态按
  // 真实链文案（跟随聊天模型优先），卡内「绑定」按钮把 vision 键传给保存动作。
  it('renders vision binding card and saves via handleSaveFeature (识图)', () => {
    const handleSaveFeature = vi.fn()
    renderBind({ handleSaveFeature })

    expect(screen.getByText('识图')).toBeTruthy()

    const visionNote = screen.getByText(/办公识图（看懂图）/)
    const card = visionNote.closest('.mc-bind-card')
    expect(card).toBeTruthy()
    // 未绑定态元信息按识图真实链（跟随聊天模型优先，非「跟随全局默认」）
    expect(within(card as HTMLElement).getByText('未绑定：优先跟随聊天模型，无本地多模态在线时回落静态默认')).toBeTruthy()
    const bindBtn = within(card as HTMLElement).getByText(/^绑\s*定$/)
    fireEvent.click(bindBtn)
    expect(handleSaveFeature).toHaveBeenCalledWith('vision')
  })

  // 识图卡「当前生效」行跟随态对齐（App.VisionRoute）：跟随聊天模型命中时
  // 出真实生效行 + chip「跟随聊天模型 · 绑定待命」（独立绑定此刻不生效）；
  // 通用 modelRoutes['vision'] 行对识图不渲染（routeModel 口径失真）。
  it('shows vision effective route with chat-follow source (识图跟随态)', () => {
    renderBind({
      featureCfg: { vision: { engine: 'herdsman', model: 'qwen3-8b' } },
      featureDraft: { vision: { engine: 'herdsman', model: 'qwen3-8b' } },
      featureEnabled: { vision: true },
      visionRoute: { engine: 'tianxuan', model: 'qwen3.8-flash-next-uncensored', source: 'chat-follow' },
      modelRoutes: { vision: { engine: 'herdsman', model: 'qwen3-8b', source: 'feature' } },
    })

    expect(screen.getByText(/当前生效：.*qwen3\.8-flash-next-uncensored（跟随聊天模型）/)).toBeTruthy()
    expect(screen.getByText(/独立绑定待命（聊天回云端时接管）/)).toBeTruthy()
    expect(screen.getByText('跟随聊天模型 · 绑定待命')).toBeTruthy()
    // 绑定行改名「独立绑定：」且旧的 routeModel 生效行不再出现
    expect(screen.getByText(/独立绑定：.*qwen3-8b/)).toBeTruthy()
    expect(screen.queryByText(/当前生效：.*（功能绑定）/)).toBeNull()
  })

  // 静态兜底态：engine 不可知报空 → 「静态默认 / 模型」；未绑定时 chip 中性。
  it('shows vision effective route with static fallback (识图静态兜底)', () => {
    renderBind({
      visionRoute: { engine: '', model: 'Qwen3.6-35B-A3B-U3B-Q4_K_P', source: 'static' },
    })

    expect(screen.getByText(/当前生效：静态默认 \/ Qwen3\.6-35B/)).toBeTruthy()
    expect(screen.getByText(/（静态默认）/)).toBeTruthy()
    expect(screen.getByText('静态兜底')).toBeTruthy()
  })
})
