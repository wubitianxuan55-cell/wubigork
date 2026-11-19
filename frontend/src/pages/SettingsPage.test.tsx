import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { LocaleProvider } from '../gaea/lib/i18n'
import { useAppStore } from '../stores/appStore'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go，面板里的数据请求经 gaea/lib/bridge
// 的 dev mock 兜底（SettingsPage 全链路已走 bridge app；无需再 mock 具体绑定，仅
// 确保 bridge 模块可解析——importOriginal 保留原模块，app 复用真实 dev mock）。
vi.mock('../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../gaea/lib/bridge')>()
  return { ...actual }
})

// 批 2 语音收编：chat 组挂 VoiceSettingsPanel，其订阅 tts-service-status 需
// wails EventsOn（jsdom 无 runtime，mock 成空订阅——同 VoiceSettingsPanel.test）
vi.mock('../../wailsjs/runtime/runtime', () => ({ EventsOn: vi.fn(() => () => {}) }))

import SettingsPage from './SettingsPage'

// S2.2b i18n：壳层组件经 useT 读字典，测试需包 LocaleProvider（zh 为默认语言）。
const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>

beforeEach(() => {
  // jsdom 默认 en-US；断言为中文文案，固定 zh 语言（可配置属性，探测每次读取）
  Object.defineProperty(navigator, 'language', { value: 'zh-CN', configurable: true })
})

describe('SettingsPage 按功能板块组织', () => {
  it('渲染八个功能分组导航（小说分区随板块退役拆除，2026-11-15）', () => {
    render(wrap(<SettingsPage />))
    for (const label of ['通用', '聊天', '绘梦', '办公', '模型', '安全', '数据', '关于']) {
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

describe('SettingsPage 设置中枢骨架（批 1：两级导航+锚点直达）', () => {
  beforeEach(() => {
    // 锚点 pending 为一次性消费态，用例间清零防串扰
    act(() => useAppStore.getState().clearPendingSettingsCategory())
  })

  it('单 zone 可见时不渲染分区标题（现状零 UI 变化，board 区批 5 挂入后才出现两级）', () => {
    const { container } = render(wrap(<SettingsPage />))
    expect(container.querySelector('.settings-nav__zone')).toBeNull()
    // 现有 10 组全部仍可达（含用户拍板保留的代码图谱）
    for (const label of ['代码图谱', '安全', '数据']) {
      expect(screen.getByRole('button', { name: new RegExp(label) })).toBeTruthy()
    }
  })

  it('openSettings(category) 锚点直达：挂载即定位目标分组并清空 pending', async () => {
    act(() => useAppStore.getState().openSettings('model'))
    render(wrap(<SettingsPage />))
    expect(await screen.findByText('当前模型')).toBeTruthy()
    expect(useAppStore.getState().pendingSettingsCategory).toBeNull()
  })

  it('openSettings 过滤态直达：先有搜索词也清空搜索再定位', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.change(screen.getByPlaceholderText(/搜索设置项/), { target: { value: '外观' } })
    act(() => useAppStore.getState().openSettings('data'))
    expect(await screen.findByText('数据备份', { selector: 'strong' })).toBeTruthy()
    expect(screen.queryByText('外观实时预览')).toBeNull()
  })

  it('语音收编（批 2）：聊天组含全局音色段与 VoiceSettingsPanel 朗读引擎深配', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.click(screen.getByRole('button', { name: /聊天/ }))
    // ChatVoiceSection（从 ChatPanel 拆出的 tts_voice 全局默认音色）；与面板内
    // 「合成音色」字样并存，用 findAllByText 容重复
    expect((await screen.findAllByText('合成音色')).length).toBeGreaterThan(0)
    // VoiceSettingsPanel（原聊天页 Modal 迁入）：朗读引擎路由卡常驻
    expect(await screen.findByText('朗读引擎')).toBeTruthy()
  })
})
