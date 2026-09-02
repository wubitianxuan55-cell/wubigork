/**
 * ImageGenPanel.test.tsx — 百炼改图（dashscope）API Key 设置
 *
 * · 仅 backend==='dashscope' 时渲染「百炼 API Key」密码框（type=password）
 * · Key 只写不读：加载后不回显任何已保存值（value 为空）
 * · 保存载荷：第 5 参 dashscopeKey 仅百炼后端带上，其他后端传空串
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

describe('ImageGenPanel 百炼 API Key（dashscope）', () => {
  it('dashscope 后端：显示 Key 密码框且不回显，保存载荷第 5 参带上 Key', async () => {
    vi.mocked(getImageBackendInfo).mockResolvedValue({ backend: 'dashscope', model: 'qwen-image-edit-plus' })
    render(<ImageGenPanel />)

    const keyInput = await screen.findByPlaceholderText('阿里云百炼 API Key，sk- 开头（留空保持不变）')
    expect(keyInput.getAttribute('type')).toBe('password')
    // 只写不读：后端不回传 Key，前端加载后输入框保持为空
    expect((keyInput as HTMLInputElement).value).toBe('')

    fireEvent.change(keyInput, { target: { value: 'sk-abc123' } })
    fireEvent.click(screen.getByText('保存绘梦配置'))
    await waitFor(() => {
      expect(setImageBackend).toHaveBeenCalledWith('dashscope', '', 'qwen-image-edit-plus', '', 'sk-abc123')
    })
  })

  it('非 dashscope 后端：不渲染 Key 框，保存第 5 参传空串', async () => {
    render(<ImageGenPanel />)
    // 等加载回填完成（当前后端区显示模型名）
    await screen.findByText('grok-imagine-image')
    expect(screen.queryByPlaceholderText('阿里云百炼 API Key，sk- 开头（留空保持不变）')).toBeNull()

    fireEvent.click(screen.getByText('保存绘梦配置'))
    await waitFor(() => {
      expect(setImageBackend).toHaveBeenCalledWith('xai', '', 'grok-imagine-image', '', '')
    })
  })

  it('dashscope 后端 Key 留空保存：第 5 参为空串（保持已保存 Key 不覆盖）', async () => {
    vi.mocked(getImageBackendInfo).mockResolvedValue({ backend: 'dashscope', model: 'qwen-image-edit-plus' })
    render(<ImageGenPanel />)
    await screen.findByPlaceholderText('阿里云百炼 API Key，sk- 开头（留空保持不变）')

    fireEvent.click(screen.getByText('保存绘梦配置'))
    await waitFor(() => {
      expect(setImageBackend).toHaveBeenCalledWith('dashscope', '', 'qwen-image-edit-plus', '', '')
    })
  })
})
