/**
 * ControlPanel.test.tsx — 引擎枚举与模式门禁
 *
 * · GLM 在 txt2img 模式可选（点击触发 onSwitchBackend("glm")）
 * · img2img 模式 GLM 禁用（官方图像端点仅文生图）
 * · glm + img2img 残留态显示「GLM 仅支持文生图」警告
 * · img2img + xai（不在白名单）提示切换引擎
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

describe('ControlPanel 引擎枚举与模式门禁', () => {
  beforeEach(() => {
    // baseProps 的 mock 跨用例共享，清掉上一用例的调用记录
    vi.clearAllMocks()
  })

  it('img2img + xai（不在白名单）：提示可用后端（无百炼）', () => {
    render(<ControlPanel {...baseProps} mode="img2img" backend="xai" />)
    // 警告 div 的直接文本 = 主文案 + 「，请切换引擎」两段相邻文本节点
    expect(screen.getByText('图生图需使用 ComfyUI / Herdsman 后端，请切换引擎')).toBeTruthy()
  })

  it('txt2img 模式：GLM 可选，点击触发 onSwitchBackend("glm")', async () => {
    render(<ControlPanel {...baseProps} mode="txt2img" backend="xai" />)
    openEngineSelect()
    await waitFor(() => expect(findOption('GLM')).toBeTruthy())
    const opt = findOption('GLM') as HTMLElement
    expect(isOptionDisabled(opt)).toBe(false)
    fireEvent.click(opt)
    expect(baseProps.onSwitchBackend).toHaveBeenCalled()
    expect(baseProps.onSwitchBackend.mock.calls[0][0]).toBe('glm')
  })

  it('img2img 模式：GLM 禁用（官方图像端点仅文生图），点击不切换', async () => {
    render(<ControlPanel {...baseProps} mode="img2img" backend="xai" />)
    openEngineSelect()
    await waitFor(() => expect(findOption('GLM')).toBeTruthy())
    const opt = findOption('GLM') as HTMLElement
    expect(isOptionDisabled(opt)).toBe(true)
    fireEvent.click(opt)
    expect(baseProps.onSwitchBackend).not.toHaveBeenCalled()
  })

  it('glm + img2img 残留态：显示「GLM 仅支持文生图」警告', () => {
    render(<ControlPanel {...baseProps} mode="img2img" backend="glm" />)
    expect(screen.getByText('GLM 仅支持文生图，请切换到文生图模式或更换引擎')).toBeTruthy()
  })

  it('glm + txt2img：无警告且不出现缺引擎提示', () => {
    render(<ControlPanel {...baseProps} mode="txt2img" backend="glm" />)
    expect(screen.queryByText(/GLM 仅支持文生图/)).toBeNull()
    expect(screen.queryByText(/请切换引擎/)).toBeNull()
  })
})
