/**
 * ImageGenPanel.test.tsx — 绘梦后端设置
 *
 * · 加载后端/模型回填（含 comfyui_url / image_save_dir，防已存配置被清空），保存走 setImageBackend(backend, url, model, saveDir)
 * · comfyui 后端显示地址输入框；其余后端不显示
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../../api/settings', () => ({
  getImageBackendInfo: vi.fn(),
  setImageBackend: vi.fn(),
}))

vi.mock('../../api/engines', () => ({
  getEngines: vi.fn(),
}))

import ImageGenPanel from './ImageGenPanel'
import { getImageBackendInfo, setImageBackend } from '../../api/settings'
import { getEngines } from '../../api/engines'
import { LocaleProvider } from '../../gaea/lib/i18n'

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getImageBackendInfo).mockResolvedValue({ backend: 'xai', model: 'grok-imagine-image' })
  vi.mocked(getEngines).mockResolvedValue([])
  vi.mocked(setImageBackend).mockResolvedValue(undefined)
  // 面板文案经 useT 读字典（zh 为默认语言），断言为中文文案：固定 zh 语言
  Object.defineProperty(navigator, 'language', { value: 'zh-CN', configurable: true })
})

// S2.2b i18n：面板组件经 useT 读字典，测试需包 LocaleProvider
const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>

/** 当前生效后端展示在下拉选中项（antd 显示 option 的 label 而非 value），等它出现 = 加载回填完成 */
const findSelectedItem = async (text: string) => {
  await waitFor(() => {
    const items = Array.from(document.querySelectorAll('.ant-select-selection-item'))
    expect(items.some((el) => el.textContent === text)).toBe(true)
  })
}

describe('ImageGenPanel 绘梦后端设置', () => {
  it('加载后端/模型回填，保存走 4 参载荷', async () => {
    render(wrap(<ImageGenPanel />))
    await findSelectedItem('xAI（云端）')

    fireEvent.click(screen.getByText('保存绘梦配置'))
    await waitFor(() => {
      expect(setImageBackend).toHaveBeenCalledWith('xai', '', 'grok-imagine-image', '')
    })
  })

  it('comfyui 后端回填已存地址/目录，保存不丢配置', async () => {
    vi.mocked(getImageBackendInfo).mockResolvedValue({
      backend: 'comfyui', model: 'krea2',
      comfyui_url: 'http://127.0.0.1:8188', image_save_dir: 'C:\\imgs',
    })
    render(wrap(<ImageGenPanel />))
    const urlInput = (await screen.findByPlaceholderText('ComfyUI 地址（例如 http://127.0.0.1:8188）')) as HTMLInputElement
    expect(urlInput.value).toBe('http://127.0.0.1:8188')

    fireEvent.click(screen.getByText('保存绘梦配置'))
    await waitFor(() => {
      expect(setImageBackend).toHaveBeenCalledWith('comfyui', 'http://127.0.0.1:8188', 'krea2', 'C:\\imgs')
    })
  })

  it('非 comfyui 后端不显示地址输入框', async () => {
    render(wrap(<ImageGenPanel />))
    await findSelectedItem('xAI（云端）')
    expect(screen.queryByPlaceholderText('ComfyUI 地址（例如 http://127.0.0.1:8188）')).toBeNull()
  })
})
