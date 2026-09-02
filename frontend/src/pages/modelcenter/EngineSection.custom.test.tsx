/**
 * EngineSection.custom.test.tsx — A 刀「自定义引擎」（OpenAI 兼容自定义服务商）分区测试
 *
 * 覆盖：
 * ① 添加表单校验：空名 / 非 http(s) 地址被拦（与后端 validBaseURL 同口径双保险，
 *    防 API Key 粘进地址框——v4.9.1 防线在自定义引擎上的延伸）；
 * ② 添加成功：handleAddCustomEngine 收到名称/地址/Key 三元组，表单关闭（刷新由
 *    useEngineState 层契约测试覆盖，见 hooks/useEngineState.test.tsx）；
 * ③ 删除需 Popconfirm 确认（文案提示会同时移除功能绑定能力），确认后才调用
 *    handleRemoveCustomEngine；
 * ④ custom 引擎卡地址框可见（复用 handleSaveURL 保存流程）+「自定义」徽标；
 * ⑤ 内置云端引擎地址框仍不可见、Key 输入不受影响（回归锁）。
 */
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ModelCenterContext, type ModelCenterContextValue } from './context'
import { EngineSection } from './EngineSection'
import type { EngineConfig } from '../../api/engines'

const ENGINES: EngineConfig[] = [
  {
    id: 'deepseek', name: 'DeepSeek', type: 'deepseek',
    base_url: 'https://api.deepseek.com', enabled: true, default_model: 'deepseek-chat', models: [],
  },
  {
    id: 'custom-123', name: '硅基流动', type: 'custom',
    base_url: 'https://api.siliconflow.cn/v1', enabled: true, default_model: '', models: [],
  },
]

function renderSection() {
  const handleAddCustomEngine = vi.fn().mockResolvedValue(true)
  const handleUpdateCustomEngine = vi.fn().mockResolvedValue(true)
  const handleRemoveCustomEngine = vi.fn().mockResolvedValue(undefined)
  const value = {
    engines: ENGINES,
    engineStatuses: {},
    editingURLs: {},
    setEditingURLs: () => {},
    savingEngine: null,
    testingEngine: null,
    activeEngine: '',
    deepseekKey: '', setDeepseekKeyState: () => {}, deepseekKeyMasked: '',
    glmKey: '', setGlmKeyState: () => {}, glmKeyMasked: '',
    opencodeGoKey: '', setOpencodeGoKeyState: () => {}, opencodeGoKeyMasked: '',
    opencodeZenKey: '', setOpencodeZenKeyState: () => {}, opencodeZenKeyMasked: '',
    settingGlmEndpoint: false,
    handleSetGlmEndpoint: async () => {},
    handleTestConnection: async () => {},
    handleRefreshModels: async () => {},
    handleSaveURL: async () => {},
    handleToggleEngine: async () => {},
    handleBulkToggleEngines: async () => {},
    handleSaveDeepseekKey: async () => {},
    handleSaveGlmKey: async () => {},
    handleSaveOpencodeGoKey: async () => {},
    handleSaveOpencodeZenKey: async () => {},
    handleAddCustomEngine,
    handleUpdateCustomEngine,
    handleRemoveCustomEngine,
    makeModels: (engine: EngineConfig) => engine.models || [],
  } as unknown as ModelCenterContextValue
  render(
    <ModelCenterContext.Provider value={value}>
      <EngineSection />
    </ModelCenterContext.Provider>,
  )
  return { handleAddCustomEngine, handleUpdateCustomEngine, handleRemoveCustomEngine }
}

/** 打开「添加自定义引擎」行内表单，返回表单控件与作用域化的保存按钮 */
function openAddForm() {
  // 按钮内 PlusOutlined 图标的 aria-label 会并入可访问名，用正则匹配尾部文案
  fireEvent.click(screen.getByRole('button', { name: /添加自定义引擎/ }))
  const nameInput = screen.getByLabelText('自定义引擎名称')
  const urlInput = screen.getByLabelText('自定义引擎 API 地址')
  const keyInput = screen.getByLabelText('自定义引擎 API Key')
  const form = nameInput.closest('div') as HTMLElement
  // antd 会在两个汉字按钮文案中插空格（保存 → 保 存），用正则兼容
  return { nameInput, urlInput, keyInput, form, saveBtn: within(form).getByRole('button', { name: /^保\s*存$/ }) }
}

describe('EngineSection · 自定义引擎', () => {
  it('⑤ 防线回归锁：内置云端引擎不露地址框（Key 输入不受影响），custom 引擎地址框可见', () => {
    renderSection()

    const dsCard = screen.getByText('DeepSeek').closest('.mc-engine-card') as HTMLElement
    expect(within(dsCard).queryByLabelText('引擎服务地址')).toBeNull() // 云端不露地址框
    expect(within(dsCard).getByPlaceholderText('sk-...')).toBeTruthy() // Key 输入仍在
    expect(within(dsCard).queryByText('自定义')).toBeNull() // 内置引擎无自定义徽标

    const customCard = screen.getByText('硅基流动').closest('.mc-engine-card') as HTMLElement
    expect(within(customCard).getByLabelText('引擎服务地址')).toBeTruthy() // custom 显示地址框
    expect(within(customCard).getByText('自定义')).toBeTruthy() // 「自定义」徽标
  })

  it('① 添加表单校验：空名 / 非 http(s) 地址被拦，不发起调用', () => {
    const { handleAddCustomEngine } = renderSection()
    const { nameInput, urlInput, saveBtn } = openAddForm()

    fireEvent.click(saveBtn) // 空名直接保存
    expect(handleAddCustomEngine).not.toHaveBeenCalled()

    fireEvent.change(nameInput, { target: { value: '硅基流动' } })
    fireEvent.change(urlInput, { target: { value: 'sk-abc123' } }) // Key 粘进地址框形态
    fireEvent.click(saveBtn)
    expect(handleAddCustomEngine).not.toHaveBeenCalled()

    fireEvent.change(urlInput, { target: { value: 'ftp://api.example.com' } }) // 非 http(s) 前缀
    fireEvent.click(saveBtn)
    expect(handleAddCustomEngine).not.toHaveBeenCalled()
  })

  it('② 添加成功：handleAddCustomEngine 收到名称/地址/Key 三元组，成功后表单关闭', async () => {
    const { handleAddCustomEngine } = renderSection()
    const { nameInput, urlInput, keyInput, saveBtn } = openAddForm()

    fireEvent.change(nameInput, { target: { value: '硅基流动' } })
    fireEvent.change(urlInput, { target: { value: 'https://api.siliconflow.cn/v1' } })
    fireEvent.change(keyInput, { target: { value: 'sk-xxx' } })
    fireEvent.click(saveBtn)

    await waitFor(() => expect(handleAddCustomEngine).toHaveBeenCalledWith('硅基流动', 'https://api.siliconflow.cn/v1', 'sk-xxx'))
    await waitFor(() => expect(screen.queryByLabelText('自定义引擎名称')).toBeNull()) // 表单关闭
  })

  it('③ 删除需确认：确认后才调用 handleRemoveCustomEngine，且文案提示移除功能绑定能力', async () => {
    const { handleRemoveCustomEngine } = renderSection()
    const customCard = screen.getByText('硅基流动').closest('.mc-engine-card') as HTMLElement

    fireEvent.click(within(customCard).getByRole('button', { name: /^删\s*除$/ }))
    expect(handleRemoveCustomEngine).not.toHaveBeenCalled() // 未确认不删

    expect(await screen.findByText(/移除其功能绑定能力/)).toBeTruthy() // Popconfirm 文案
    const okBtn = await waitFor(() => {
      const btn = document.querySelector('.ant-popconfirm .ant-btn-primary')
      expect(btn).toBeTruthy()
      return btn as HTMLElement
    })
    fireEvent.click(okBtn)
    await waitFor(() => expect(handleRemoveCustomEngine).toHaveBeenCalledWith('custom-123'))
  })

  it('④ 编辑入口：表单预填名称/地址，Key 留空保存 = 不修改（空串透传）', async () => {
    const { handleUpdateCustomEngine } = renderSection()
    const customCard = screen.getByText('硅基流动').closest('.mc-engine-card') as HTMLElement

    fireEvent.click(within(customCard).getByRole('button', { name: /^编\s*辑$/ }))
    const nameInput = screen.getByLabelText('编辑引擎名称') as HTMLInputElement
    const urlInput = screen.getByLabelText('编辑引擎 API 地址') as HTMLInputElement
    expect(nameInput.value).toBe('硅基流动') // 预填现名
    expect(urlInput.value).toBe('https://api.siliconflow.cn/v1') // 预填现址

    fireEvent.change(nameInput, { target: { value: '硅基流动 Pro' } })
    const form = nameInput.closest('div') as HTMLElement
    fireEvent.click(within(form).getByRole('button', { name: /^保\s*存$/ })) // Key 留空

    await waitFor(() => expect(handleUpdateCustomEngine).toHaveBeenCalledWith('custom-123', '硅基流动 Pro', 'https://api.siliconflow.cn/v1', ''))
  })
})
