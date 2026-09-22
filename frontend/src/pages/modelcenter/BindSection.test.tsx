import { fireEvent, render, screen, waitFor } from '@testing-library/react'
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
})
