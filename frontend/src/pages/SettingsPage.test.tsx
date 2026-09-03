import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { LocaleProvider } from '../gaea/lib/i18n'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go，面板里的数据请求全部 mock 为空。
vi.mock('../../src/wailsjsCompat', () => ({
  GetAppInfo: vi.fn().mockResolvedValue({ name: 'gaea', version: '2.4.0', tagline: '测试', releases: [] }),
  GetConfig: vi.fn().mockResolvedValue({}),
  WhisperGetPersonalities: vi.fn().mockResolvedValue([]),
  GaeaSettings: vi.fn().mockResolvedValue({}),
  GetActiveModel: vi.fn().mockResolvedValue(''),
  GetImageBackendInfo: vi.fn().mockResolvedValue({}),
  GaeaDataBackupInfo: vi.fn().mockResolvedValue({ data_root: 'C:\\data', entries: [], total_bytes: 0, pending: false, app_version: '2.20.0' }),
  GaeaDataBackupRestoreResult: vi.fn().mockResolvedValue({ has_result: false }),
}))

import SettingsPage from './SettingsPage'

// S2.2b i18n：壳层组件经 useT 读字典，测试需包 LocaleProvider（zh 为默认语言）。
const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>

beforeEach(() => {
  // jsdom 默认 en-US；断言为中文文案，固定 zh 语言（可配置属性，探测每次读取）
  Object.defineProperty(navigator, 'language', { value: 'zh-CN', configurable: true })
})

describe('SettingsPage 按功能板块组织', () => {
  it('渲染九个功能分组导航', () => {
    render(wrap(<SettingsPage />))
    for (const label of ['通用', '聊天', '小说', '绘梦', '办公', '模型', '安全', '数据', '关于']) {
      expect(screen.getByRole('button', { name: new RegExp(label) })).toBeTruthy()
    }
  })

  it('默认展示通用（外观）分组', () => {
    render(wrap(<SettingsPage />))
    expect(screen.getByText('外观实时预览')).toBeTruthy()
  })

  it('通用分组含界面语言切换', () => {
    render(wrap(<SettingsPage />))
    expect(screen.getByText('界面语言')).toBeTruthy()
    expect(screen.getAllByText(/跟随系统（当前 简体中文）/).length).toBeGreaterThan(0)
  })

  it('搜索可过滤分组并自动切换到匹配项', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.change(screen.getByPlaceholderText(/搜索设置项/), { target: { value: '推理强度' } })
    expect(screen.getByText('模型')).toBeTruthy()
    expect(screen.queryByText('通用')).toBeNull()
    expect(await screen.findByText('当前模型')).toBeTruthy()
  })

  it('点击关于分组展示存储路径与更新日志', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.click(screen.getByRole('button', { name: /关于/ }))
    expect(await screen.findByText('存储路径')).toBeTruthy()
    expect(screen.getByText('更新日志')).toBeTruthy()
  })
})
