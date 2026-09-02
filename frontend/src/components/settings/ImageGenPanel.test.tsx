/**
 * ImageGenPanel.test.tsx — 绘梦后端设置
 *
 * · 加载当前后端/模型回填，保存走 setImageBackend(backend, url, model, saveDir)
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

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getImageBackendInfo).mockResolvedValue({ backend: 'xai', model: 'grok-imagine-image' })
  vi.mocked(getEngines).mockResolvedValue([])
  vi.mocked(setImageBackend).mockResolvedValue(undefined)
})

describe('ImageGenPanel 绘梦后端设置', () => {
  it('加载后端/模型回填，保存走 4 参载荷', async () => {
    render(<ImageGenPanel />)
    // 等加载回填完成（当前后端区显示模型名）
    await screen.findByText('grok-imagine-image')

    fireEvent.click(screen.getByText('保存绘梦配置'))
    await waitFor(() => {
      expect(setImageBackend).toHaveBeenCalledWith('xai', '', 'grok-imagine-image', '')
    })
  })

  it('comfyui 后端显示地址输入框', async () => {
    vi.mocked(getImageBackendInfo).mockResolvedValue({ backend: 'comfyui', model: 'krea2' })
    render(<ImageGenPanel />)
    await screen.findByText('krea2')
    expect(screen.getByPlaceholderText('ComfyUI 地址（例如 http://127.0.0.1:8188）')).toBeTruthy()
  })

  it('非 comfyui 后端不显示地址输入框', async () => {
    render(<ImageGenPanel />)
    await screen.findByText('grok-imagine-image')
    expect(screen.queryByPlaceholderText('ComfyUI 地址（例如 http://127.0.0.1:8188）')).toBeNull()
  })
})
