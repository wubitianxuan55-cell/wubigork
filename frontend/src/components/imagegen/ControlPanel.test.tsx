/**
 * ControlPanel.test.tsx — 引擎枚举与模式门禁
 *
 * · GLM 在 txt2img 模式可选（点击触发 onSwitchBackend("glm")）
 * · img2img 模式 GLM 禁用（官方图像端点仅文生图）
 * · glm + img2img 残留态显示「GLM 仅支持文生图」警告
 * · img2img + xai（不在白名单）提示切换引擎
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
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

// ── 审计刀C-1：Wails 壳内「上传参考图」收口到 GaeaPickFiles 系统对话框 ────
// mock 口径照 schedule/store.sync.test.ts：window.go.app.<门面> 装真壳同款
// Gaea 前缀绑定（bridge realApp 按方法名路由），afterEach delete 还原。

describe('ControlPanel 壳内上传参考图（刀C-1）', () => {
  beforeEach(() => {
    // baseProps 的 mock 跨用例共享（含上方 describe），清掉上一用例调用记录
    vi.clearAllMocks()
  })

  afterEach(() => {
    delete (window as unknown as { go?: unknown }).go
  })

  it('壳内选图：GaeaPickFiles+GaeaReadFileB64 还原 File 喂同一条 readFile 管线（不点隐藏 input）', async () => {
    const pickFiles = vi.fn(async () => ([
      { path: 'C:/imgs/ref.png', name: 'ref.png', type: 'image', size: 5 },
    ]))
    const readFileB64 = vi.fn(async () => 'aGVsbG8=') // "hello"
    ;(window as unknown as { go?: unknown }).go = {
      app: { ImgPickTestC: { GaeaPickFiles: pickFiles, GaeaReadFileB64: readFileB64 } },
    }
    // 壳内分支不得回落浏览器 input.click()（jsdom 点了也不弹框，正是 P0 本体）
    const inputClick = vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(() => {})
    try {
      render(<ControlPanel {...baseProps} mode="img2img" backend="comfyui" />)
      fireEvent.click(screen.getByText('点击上传或拖入参考图'))
      await waitFor(() => {
        expect(pickFiles).toHaveBeenCalledTimes(1)
        expect(readFileB64).toHaveBeenCalledWith('C:/imgs/ref.png')
      })
      // 同一数据通路：还原 File → readFile(FileReader) → onInitImageChange(dataURL)
      await waitFor(() => {
        const arg = String(baseProps.onInitImageChange.mock.calls[0]?.[0] ?? '')
        expect(arg).toContain('base64,aGVsbG8=')
      })
      expect(inputClick).not.toHaveBeenCalled()
    } finally {
      inputClick.mockRestore()
    }
  })

  it('壳内选到非图片扩展名：fail-closed message 提示，不进 readFile 管线', async () => {
    const pickFiles = vi.fn(async () => ([
      { path: 'C:/docs/ref.txt', name: 'ref.txt', type: 'file', size: 5 },
    ]))
    const readFileB64 = vi.fn(async () => 'eHg=')
    ;(window as unknown as { go?: unknown }).go = {
      app: { ImgPickTestC: { GaeaPickFiles: pickFiles, GaeaReadFileB64: readFileB64 } },
    }
    render(<ControlPanel {...baseProps} mode="img2img" backend="comfyui" />)
    fireEvent.click(screen.getByText('点击上传或拖入参考图'))
    await waitFor(() => expect(pickFiles).toHaveBeenCalledTimes(1))
    expect(await screen.findByText(/仅支持 \.png\/\.jpg/)).toBeTruthy()
    expect(readFileB64).not.toHaveBeenCalled()
    expect(baseProps.onInitImageChange).not.toHaveBeenCalled()
  })

  it('浏览器环境回退原生 input 弹框（不调 GaeaPickFiles）', () => {
    const inputClick = vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(() => {})
    try {
      render(<ControlPanel {...baseProps} mode="img2img" backend="comfyui" />)
      fireEvent.click(screen.getByText('点击上传或拖入参考图'))
      expect(inputClick).toHaveBeenCalledTimes(1)
    } finally {
      inputClick.mockRestore()
    }
  })
})

describe('ControlPanel 一致性方法选择（阶段三刀 A）', () => {
  it('带参考角色时渲染方法 Radio；qedit 态文案指向 Qwen 参考编辑', () => {
    const onRefMethodChange = vi.fn()
    const { rerender } = render(<ControlPanel
      {...baseProps}
      refChars={[{ id: 'c1', name: '苏离', refCount: 2 }]}
      refMethod="qedit"
      onRefMethodChange={onRefMethodChange}
    />)
    expect(screen.getByTestId('ref-method-row')).toBeTruthy()
    expect(screen.getByTestId('ref-method-group')).toBeTruthy()
    expect(screen.getByText(/Qwen 参考编辑（推荐）/)).toBeTruthy()
    // 切回 img2img：文案切换 + 回调
    fireEvent.click(screen.getAllByText('图生图近似')[0])
    expect(onRefMethodChange).toHaveBeenCalledWith('img2img')
    rerender(<ControlPanel
      {...baseProps}
      refChars={[{ id: 'c1', name: '苏离', refCount: 2 }]}
      refMethod="img2img"
      onRefMethodChange={onRefMethodChange}
    />)
    expect(screen.getByText(/自动切到图生图/)).toBeTruthy()
  })

  it('无参考角色时不渲染方法选择（缺省不渲染惯例）', () => {
    render(<ControlPanel {...baseProps} refMethod="qedit" />)
    expect(screen.queryByTestId('ref-method-row')).toBeNull()
  })
})
