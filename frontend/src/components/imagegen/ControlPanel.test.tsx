/**
 * ControlPanel.test.tsx — 引擎枚举与模式门禁（百炼改图 dashscope）
 *
 * · dashscope 在 img2img 模式可选（白名单放行，点击触发 onSwitchBackend）
 * · txt2img 模式下 dashscope 选项禁用（百炼仅支持改图），点击不生效
 * · dashscope 为云端引擎：不渲染本地引擎启停块，img2img 不再提示切换引擎
 * · dashscope + txt2img 残留态：显示「百炼仅支持改图」警告
 * · dashscope 模型帮助文案（默认 qwen-image-edit-plus，后端兜底）
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ControlPanel } from './ControlPanel'
import type { ImageMode } from './types'

const baseProps = {
  mode: 'txt2img' as ImageMode,
  prompt: '',
  negative: '',
  onPromptChange: vi.fn(),
  onNegativeChange: vi.fn(),
  onOpenTemplatePicker: vi.fn(),
  model: '',
  modelOptions: [] as { label: string; value: string }[],
  onModelChange: vi.fn(),
  size: '1024x1024',
  onSizeChange: vi.fn(),
  customWidth: 1024,
  customHeight: 1024,
  onCustomWidthChange: vi.fn(),
  onCustomHeightChange: vi.fn(),
  seed: 0,
  onSeedChange: vi.fn(),
  count: 1,
  onCountChange: vi.fn(),
  initImage: '',
  onInitImageChange: vi.fn(),
  denoise: 0.6,
  onDenoiseChange: vi.fn(),
  frames: 0,
  onFramesChange: vi.fn(),
  fps: 0,
  onFpsChange: vi.fn(),
  selectedLoras: [] as string[],
  loraOptions: [] as { label: string; value: string }[],
  onLorasChange: vi.fn(),
  backend: 'xai',
  backendSwitching: false,
  engineRunning: false,
  engineStarting: false,
  engineModelCount: 0,
  onSwitchBackend: vi.fn(),
  onStartEngine: vi.fn(),
  onStopEngine: vi.fn(),
  sysStats: null,
}

/** 打开引擎下拉（引擎 Select 是面板里第一个 .ant-select-selector），弹层挂在 body */
const openEngineSelect = () => {
  const selector = document.querySelectorAll<HTMLElement>('.ant-select-selector')[0]
  fireEvent.mouseDown(selector)
}

/** 在打开的下拉里按文案找选项节点 */
const findOption = (text: string): HTMLElement | null => {
  const opts = Array.from(document.querySelectorAll<HTMLElement>('.ant-select-item-option'))
  return opts.find((o) => o.textContent?.includes(text)) || null
}

/** antd 此版本禁用选项不打 aria-disabled，只加 disabled 类 */
const isOptionDisabled = (opt: HTMLElement): boolean => opt.className.includes('ant-select-item-option-disabled')

describe('ControlPanel 引擎枚举与模式门禁（百炼改图）', () => {
  beforeEach(() => {
    // baseProps 的 mock 跨用例共享，清掉上一用例的调用记录
    vi.clearAllMocks()
  })

  it('img2img 模式：百炼改图可选，点击触发 onSwitchBackend("dashscope")', async () => {
    render(<ControlPanel {...baseProps} mode="img2img" backend="xai" />)
    openEngineSelect()
    await waitFor(() => expect(findOption('百炼改图')).toBeTruthy())
    const opt = findOption('百炼改图') as HTMLElement
    expect(isOptionDisabled(opt)).toBe(false)
    fireEvent.click(opt)
    // Select onChange 以 (value, option) 两参触发，这里只断言首参
    expect(baseProps.onSwitchBackend).toHaveBeenCalled()
    expect(baseProps.onSwitchBackend.mock.calls[0][0]).toBe('dashscope')
  })

  it('txt2img 模式：百炼改图禁用，点击不切换（百炼仅支持改图）', async () => {
    render(<ControlPanel {...baseProps} mode="txt2img" backend="xai" />)
    openEngineSelect()
    await waitFor(() => expect(findOption('百炼改图')).toBeTruthy())
    const opt = findOption('百炼改图') as HTMLElement
    expect(isOptionDisabled(opt)).toBe(true)
    fireEvent.click(opt)
    expect(baseProps.onSwitchBackend).not.toHaveBeenCalled()
  })

  it('dashscope 为云端引擎：不渲染本地引擎启停块，img2img 不提示切换引擎', () => {
    render(<ControlPanel {...baseProps} mode="img2img" backend="dashscope" />)
    expect(screen.queryByText('启动')).toBeNull()
    expect(screen.queryByText('停止')).toBeNull()
    // img2img 门禁白名单含 dashscope → 不出现「请切换引擎」警告
    expect(screen.queryByText(/请切换引擎/)).toBeNull()
    // 参考图重绘提示走百炼分支而非 Herdsman 文案
    expect(screen.getByText('百炼改图按参考图重绘，不支持重绘幅度调节')).toBeTruthy()
  })

  it('dashscope + txt2img 残留态：显示「百炼仅支持改图」警告', () => {
    render(<ControlPanel {...baseProps} mode="txt2img" backend="dashscope" />)
    expect(screen.getByText('百炼仅支持改图，请切换到图生图模式或更换引擎')).toBeTruthy()
  })

  it('dashscope 下模型区显示帮助文案（默认 qwen-image-edit-plus，后端兜底）', () => {
    render(<ControlPanel {...baseProps} mode="img2img" backend="dashscope" />)
    expect(screen.getByText(
      '百炼改图默认模型 qwen-image-edit-plus（留空由后端兜底），可选 qwen-image-edit / qwen-image-edit-plus / qwen-image-edit-max',
    )).toBeTruthy()
    expect(screen.getByPlaceholderText('模型名，如 qwen-image-edit-plus（留空后端兜底）')).toBeTruthy()
  })

  it('img2img + xai（不在白名单）：提示可用后端含百炼改图', () => {
    render(<ControlPanel {...baseProps} mode="img2img" backend="xai" />)
    // 警告 div 的直接文本 = 主文案 + 「，请切换引擎」两段相邻文本节点
    expect(screen.getByText('图生图需使用 ComfyUI / Herdsman / 百炼改图 后端，请切换引擎')).toBeTruthy()
  })
})
