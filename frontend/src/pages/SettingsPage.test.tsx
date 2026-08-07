import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go，面板里的数据请求全部 mock 为空。
vi.mock('../../wailsjs/go/app/App', () => ({
  GetAppInfo: vi.fn().mockResolvedValue({ name: 'gaea', version: '2.4.0', tagline: '测试', releases: [] }),
  GetConfig: vi.fn().mockResolvedValue({}),
  WhisperGetPersonalities: vi.fn().mockResolvedValue([]),
  GaeaSettings: vi.fn().mockResolvedValue({}),
  GetActiveModel: vi.fn().mockResolvedValue(''),
  GetImageBackendInfo: vi.fn().mockResolvedValue({}),
}))

import SettingsPage from './SettingsPage'

describe('SettingsPage 按功能板块组织', () => {
  it('渲染七个功能分组导航', () => {
    render(<SettingsPage />)
    for (const label of ['通用', '聊天', '小说', '绘梦', '办公', '模型', '关于']) {
      expect(screen.getByRole('button', { name: new RegExp(label) })).toBeTruthy()
    }
  })

  it('默认展示通用（外观）分组', () => {
    render(<SettingsPage />)
    expect(screen.getByText('外观实时预览')).toBeTruthy()
  })

  it('搜索可过滤分组并自动切换到匹配项', async () => {
    render(<SettingsPage />)
    fireEvent.change(screen.getByPlaceholderText(/搜索设置项/), { target: { value: '推理强度' } })
    expect(screen.getByText('模型')).toBeTruthy()
    expect(screen.queryByText('通用')).toBeNull()
    expect(await screen.findByText('当前模型')).toBeTruthy()
  })

  it('点击关于分组展示系统信息与更新日志', async () => {
    render(<SettingsPage />)
    fireEvent.click(screen.getByRole('button', { name: /关于/ }))
    expect(await screen.findByText('系统信息')).toBeTruthy()
    expect(screen.getByText('更新日志')).toBeTruthy()
  })
})
