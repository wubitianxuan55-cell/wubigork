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
 * ⑤ 内置云端引擎地址框仍不可见、Key 输入不受影响（回归锁）；
 * ⑥ 价目 v1（自定义引擎用户价目）：编辑表单预填现价，保存经 SaveEngine 通道
 *    往返 user_price_in/out（留空=0=清除=不计价）；负数由 InputNumber min=0
 *    失焦钳制防线挡下（负价无法入库）。
 */
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ModelCenterContext, type ModelCenterContextValue } from './context'
import { EngineSection } from './EngineSection'
import { saveEngine, type EngineConfig } from '../../api/engines'

// 价目 v1：价目保存走既有 SaveEngine 通道（不经 context handler），mock 掉
// api 层以便断言保存载荷（数字=设置、0=清除）。
vi.mock('../../api/engines', () => ({ saveEngine: vi.fn() }))
const mSaveEngine = vi.mocked(saveEngine)

const ENGINES: EngineConfig[] = [
  {
    id: 'deepseek', name: 'DeepSeek', type: 'deepseek',
    base_url: 'https://api.deepseek.com', enabled: true, default_model: 'deepseek-chat', models: [],
  },
  {
    id: 'custom-123', name: '硅基流动', type: 'custom',
    base_url: 'https://api.siliconflow.cn/v1', enabled: true, default_model: '', models: [],
    user_price_in: 2, user_price_out: 8, // 价目 v1：预填现价用（¥/百万 tokens）
  },
]

function renderSection() {
  mSaveEngine.mockClear()
  mSaveEngine.mockResolvedValue(undefined)
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
  return { handleAddCustomEngine, handleUpdateCustomEngine, handleRemoveCustomEngine, saveEngine: mSaveEngine }
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

  it('⑥ 价目 v1 往返：预填现价 → 改输入价/清空输出价保存 → saveEngine 收到 user_price_in/out（0=清除）', async () => {
    const { saveEngine } = renderSection()
    const customCard = screen.getByText('硅基流动').closest('.mc-engine-card') as HTMLElement

    fireEvent.click(within(customCard).getByRole('button', { name: /^编\s*辑$/ }))
    const inInput = screen.getByLabelText('输入价（每百万 tokens，CNY）') as HTMLInputElement
    const outInput = screen.getByLabelText('输出价（每百万 tokens，CNY）') as HTMLInputElement
    // 预填引擎现价（ENGINES.user_price_in/out）；InputNumber 按 step=0.1 精度
    // 显示（2 → '2.0'），断言按数值比较不锁显示格式
    expect(Number(inInput.value)).toBe(2)
    expect(Number(outInput.value)).toBe(8)

    fireEvent.change(inInput, { target: { value: '3' } })
    fireEvent.change(outInput, { target: { value: '' } }) // 清空 = 清除 = 不计价
    // 表单作用域以纯文本 Input（名称框）为锚——InputNumber 外层有 antd 包裹 div，
    // closest('div') 会落在其内部容器上拿不到保存按钮
    const form = (screen.getByLabelText('编辑引擎名称') as HTMLInputElement).closest('div') as HTMLElement
    fireEvent.click(within(form).getByRole('button', { name: /^保\s*存$/ }))

    await waitFor(() =>
      expect(saveEngine).toHaveBeenCalledWith(
        expect.objectContaining({ id: 'custom-123', user_price_in: 3, user_price_out: 0 }),
      ),
    )
  })

  it('⑥b 价目 v1 负数防线：InputNumber min=0 失焦钳到 0（=清除），负价无法入库', async () => {
    const { saveEngine } = renderSection()
    const customCard = screen.getByText('硅基流动').closest('.mc-engine-card') as HTMLElement

    fireEvent.click(within(customCard).getByRole('button', { name: /^编\s*辑$/ }))
    const inInput = screen.getByLabelText('输入价（每百万 tokens，CNY）') as HTMLInputElement
    fireEvent.change(inInput, { target: { value: '-1' } }) // antd 范围外输入态：onChange 不触发
    fireEvent.blur(inInput) // 失焦钳到 min=0 → onChange(0)（0=清除=不计价）
    await waitFor(() => expect(Number(inInput.value)).toBe(0))
    const form = (screen.getByLabelText('编辑引擎名称') as HTMLInputElement).closest('div') as HTMLElement
    fireEvent.click(within(form).getByRole('button', { name: /^保\s*存$/ }))

    await waitFor(() =>
      expect(saveEngine).toHaveBeenCalledWith(
        expect.objectContaining({ id: 'custom-123', user_price_in: 0, user_price_out: 8 }),
      ),
    )
  })
})
