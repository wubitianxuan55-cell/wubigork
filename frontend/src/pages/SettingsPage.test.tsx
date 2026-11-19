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
    // 批 3 后模型组挂 BindSection 等段，组内文本含「模型」字样——导航过滤断言
    // 钉到导航 label（.settings-nav-item__label），与组面板内容解耦
    expect(screen.getByText('模型', { selector: '.settings-nav-item__label' })).toBeTruthy()
    expect(screen.queryByText('通用', { selector: '.settings-nav-item__label' })).toBeNull()
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

  it('两级导航点亮（批 5 zone 归位）：全局/板块分区标题常驻，板块区含办公/绘梦/文学/微信/众神殿/进度计划/记忆中枢', () => {
    render(wrap(<SettingsPage />))
    // 重设计后内容头也有 zone 徽标（settings-head__zone），此处只钉导航分区标题
    const zones = screen.getAllByText(/^(全局|板块)$/, { selector: '.settings-nav__zone' })
    expect(zones.length).toBe(2)
    // 板块区七组（办公/绘梦归位 + 文学/微信/众神殿 + 进度计划/记忆中枢索引卡）+ 全局区现有组全部可达
    for (const label of ['办公', '绘梦', '文学', '微信', '众神殿', '进度计划', '记忆中枢', '通用', '模型', '代码图谱']) {
      expect(screen.getByRole('button', { name: new RegExp(label) })).toBeTruthy()
    }
  })

  it('板块薄组可达（批 5）：微信组含回推服务开关段', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.click(screen.getByRole('button', { name: /微信/ }))
    expect(await screen.findByText('回推服务', { selector: 'strong' })).toBeTruthy()
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

  it('模型与引擎收编（批 3）：模型组含功能绑定/空间策略/本地调度/引擎管理', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.click(screen.getByRole('button', { name: /模型/ }))
    // BindSection 无大标题（区块名在模型中心 rail 上），以小节锚「聊天语音」证挂载；
    // Strategy/Scheduling/Engine 用 SectionHead（mc-section-title span，非 strong）
    expect(await screen.findByText(/聊天语音/, { selector: '.mc-bind-title' })).toBeTruthy()
    for (const title of ['空间策略', '本地调度', '引擎管理']) {
      expect(await screen.findByText(title, { selector: '.mc-section-title span' })).toBeTruthy()
    }
  })

  it('办公扩容（批 4）：办公组含记忆四态/工位面板启停/MCP 能力区', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.click(screen.getByRole('button', { name: /办公/ }))
    // 记忆四态段（原右栏 MemoryPanel 胶囊）
    expect(await screen.findByText('办公记忆', { selector: 'strong' })).toBeTruthy()
    expect(await screen.findByText('晨报预载')).toBeTruthy()
    // 工位面板启停段（原「管理」弹层，CAD 可选卡常驻入口）
    expect(await screen.findByText('工位面板', { selector: 'strong' })).toBeTruthy()
    // MCP 能力区（原 CapabilitiesPanel 抽屉主体）：tab 栏常驻
    expect(await screen.findByRole('tab', { name: /连接器|服务器/ })).toBeTruthy()
  })
})

describe('SettingsPage UI 重设计（2026-11-20：内容头方位感/搜索零死端/相关分组快跳）', () => {
  it('内容头：激活组标题随导航切换，zone 徽标标注所属分区（全局/板块）', () => {
    render(wrap(<SettingsPage />))
    expect(screen.getByText('通用', { selector: '.settings-head__title' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: /绘梦/ }))
    expect(screen.getByText('绘梦', { selector: '.settings-head__title' })).toBeTruthy()
    expect(screen.getByText('板块', { selector: '.settings-head__zone' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: /模型/ }))
    expect(screen.getByText('全局', { selector: '.settings-head__zone' })).toBeTruthy()
  })

  it('搜索无匹配零死端：出建议分组胶囊，点击直达并清空搜索恢复全量导航', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.change(screen.getByPlaceholderText(/搜索设置项/), { target: { value: 'zzzzq' } })
    expect(screen.queryByText('通用', { selector: '.settings-nav-item__label' })).toBeNull()
    fireEvent.click(screen.getByText('模型', { selector: '.settings-empty__chip' }))
    expect(await screen.findByText('当前模型')).toBeTruthy()
    // 直达同语义=清搜索（锚点直达同款），导航恢复全量
    expect(screen.getByText('通用', { selector: '.settings-nav-item__label' })).toBeTruthy()
    expect(screen.queryByText('模型', { selector: '.settings-empty__chip' })).toBeNull()
  })

  it('inspector 相关分组快跳：帮助面板内按关键词交集出胶囊，点击切组', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.click(screen.getByLabelText('显示帮助面板'))
    fireEvent.click(screen.getByRole('button', { name: /模型/ }))
    expect(await screen.findByText('相关分组')).toBeTruthy()
    fireEvent.click(screen.getByText('绘梦', { selector: '.settings-related-item' }))
    expect(screen.getByText('绘梦', { selector: '.settings-head__title' })).toBeTruthy()
  })

  it('索引卡直达（批 7 挂账收口）：进度计划组出索引卡，跳转按钮发 NAVIGATE 事件到板块页', () => {
    const handler = vi.fn()
    window.addEventListener('navigate', handler)
    try {
      render(wrap(<SettingsPage />))
      // 先点导航（此刻索引卡未挂载，按钮名不撞「打开进度计划」）
      fireEvent.click(screen.getByRole('button', { name: /进度计划/ }))
      expect(screen.getByText('进度计划偏好（板块内即时调）', { selector: 'strong' })).toBeTruthy()
      fireEvent.click(screen.getByRole('button', { name: /打开进度计划/ }))
      expect(handler).toHaveBeenCalledTimes(1)
      expect((handler.mock.calls[0][0] as CustomEvent).detail).toEqual({ page: 'schedule' })
    } finally {
      window.removeEventListener('navigate', handler)
    }
  })

  it('索引卡直达：记忆中枢组同款（星图显示设置索引 + 跳转按钮）', () => {
    const handler = vi.fn()
    window.addEventListener('navigate', handler)
    try {
      render(wrap(<SettingsPage />))
      fireEvent.click(screen.getByRole('button', { name: /记忆中枢/ }))
      expect(screen.getByText('记忆中枢偏好（板块内即时调）', { selector: 'strong' })).toBeTruthy()
      expect(screen.getByText('星图显示设置：轨道环 / 星尘 / 自动旋转', { selector: '.board-index-card__item' })).toBeTruthy()
      fireEvent.click(screen.getByRole('button', { name: /打开记忆中枢/ }))
      expect((handler.mock.calls[0][0] as CustomEvent).detail).toEqual({ page: 'memoryhub' })
    } finally {
      window.removeEventListener('navigate', handler)
    }
  })
})
