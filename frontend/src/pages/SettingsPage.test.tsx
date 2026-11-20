import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react'
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
    fireEvent.click(screen.getByRole('tab', { name: '动效与语言' }))
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
    expect(useAppStore.getState().pendingSettingsTab).toBeNull()
  })

  it('openSettings(category, tab) 锚点到内部任务页签：语音设置直达朗读引擎', async () => {
    act(() => useAppStore.getState().openSettings('chat', 'speech-engine'))
    render(wrap(<SettingsPage />))
    const tab = await screen.findByRole('tab', { name: '朗读引擎' })
    expect(tab.getAttribute('aria-selected')).toBe('true')
    expect((await screen.findAllByText('朗读引擎')).length).toBeGreaterThan(1)
    expect(useAppStore.getState().pendingSettingsTab).toBeNull()
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
    expect(await screen.findByRole('tab', { name: '角色与人格' })).toBeTruthy()
    fireEvent.click(screen.getByRole('tab', { name: '默认音色' }))
    // ChatVoiceSection（从 ChatPanel 拆出的 tts_voice 全局默认音色）；与面板内
    // 「合成音色」字样并存，用 findAllByText 容重复
    expect((await screen.findAllByText('合成音色')).length).toBeGreaterThan(0)
    fireEvent.click(screen.getByRole('tab', { name: '朗读引擎' }))
    // VoiceSettingsPanel（原聊天页 Modal 迁入）：朗读引擎路由卡常驻
    expect((await screen.findAllByText('朗读引擎')).length).toBeGreaterThan(1)
  })

  it('模型与引擎收编（批 3）：模型组含功能绑定/空间策略/本地调度/引擎管理', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.click(screen.getByRole('button', { name: /模型/ }))
    expect(await screen.findByText('当前模型')).toBeTruthy()
    fireEvent.click(screen.getByRole('tab', { name: '功能绑定' }))
    // BindSection 无大标题（区块名在模型中心 rail 上），以小节锚「聊天语音」证挂载
    expect(await screen.findByText(/聊天语音/, { selector: '.mc-bind-title' })).toBeTruthy()
    fireEvent.click(screen.getByRole('tab', { name: '空间策略' }))
    expect(await screen.findByText('空间策略', { selector: '.mc-section-title span' })).toBeTruthy()
    fireEvent.click(screen.getByRole('tab', { name: '本地调度' }))
    expect(await screen.findByText('本地调度', { selector: '.mc-section-title span' })).toBeTruthy()
    fireEvent.click(screen.getByRole('tab', { name: '引擎管理' }))
    expect(await screen.findByText('引擎管理', { selector: '.mc-section-title span' })).toBeTruthy()
  })

  it('办公扩容（批 4）：办公组含记忆四态/工位面板启停/MCP 能力区', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.click(screen.getByRole('button', { name: /办公/ }))
    fireEvent.click(await screen.findByRole('tab', { name: '记忆与面板' }))
    // 记忆四态段（原右栏 MemoryPanel 胶囊）
    expect(await screen.findByText('办公记忆', { selector: 'strong' })).toBeTruthy()
    expect(await screen.findByText('晨报预载')).toBeTruthy()
    // 工位面板启停段（原「管理」弹层，CAD 可选卡常驻入口）
    expect(await screen.findByText('工位面板', { selector: 'strong' })).toBeTruthy()
    fireEvent.click(screen.getByRole('tab', { name: '工具与技能' }))
    // MCP 能力区（原 CapabilitiesPanel 抽屉主体）：tab 栏常驻
    expect(await screen.findByRole('tab', { name: /连接器|服务器/ })).toBeTruthy()
  })
})

describe('SettingsPage 密集设置任务分栏（2026-11-20 第二轮 UI 重设计）', () => {
  it('模型组只显示一个任务域，页签切换后再显示对应分区', async () => {
    const { container } = render(wrap(<SettingsPage />))
    fireEvent.click(screen.getByRole('button', { name: /模型/ }))

    expect((await screen.findByRole('tab', { name: '概览' })).getAttribute('aria-selected')).toBe('true')
    for (const label of ['功能绑定', '空间策略', '本地调度', '引擎管理']) {
      expect(screen.getByRole('tab', { name: label })).toBeTruthy()
    }
    expect(container.querySelectorAll('[role="tabpanel"]:not([hidden])')).toHaveLength(1)

    fireEvent.click(screen.getByRole('tab', { name: '功能绑定' }))
    expect(screen.getByRole('tab', { name: '功能绑定' }).getAttribute('aria-selected')).toBe('true')
    expect(await screen.findByText(/聊天语音/, { selector: '.mc-bind-title' })).toBeTruthy()
    expect(container.querySelectorAll('[role="tabpanel"]:not([hidden])')).toHaveLength(1)
  })

  it('全局搜索命中内部任务关键词时自动切到对应任务页签', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.change(screen.getByPlaceholderText(/搜索设置项/), { target: { value: '故障转移' } })

    expect((await screen.findByRole('tab', { name: '本地调度' })).getAttribute('aria-selected')).toBe('true')
    expect(await screen.findByText('本地调度', { selector: '.mc-section-title span' })).toBeTruthy()
  })

  it('内部页签支持方向键/Home/End 标准 tablist 键盘语义', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.click(screen.getByRole('button', { name: /模型/ }))
    const overview = await screen.findByRole('tab', { name: '概览' })
    overview.focus()

    fireEvent.keyDown(overview, { key: 'ArrowRight' })
    expect(screen.getByRole('tab', { name: '功能绑定' }).getAttribute('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(screen.getByRole('tab', { name: '功能绑定' }))

    fireEvent.keyDown(document.activeElement as Element, { key: 'End' })
    expect(screen.getByRole('tab', { name: '引擎管理' }).getAttribute('aria-selected')).toBe('true')
    expect(document.activeElement).toBe(screen.getByRole('tab', { name: '引擎管理' }))

    fireEvent.keyDown(document.activeElement as Element, { key: 'Home' })
    expect(screen.getByRole('tab', { name: '概览' }).getAttribute('aria-selected')).toBe('true')
  })

  it('访问过的任务页签保持挂载：切换页签不丢未保存草稿', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.click(screen.getByRole('button', { name: /聊天/ }))
    const companionName = await screen.findByPlaceholderText(/给 AI 起个名字/)
    fireEvent.change(companionName, { target: { value: '临时称呼' } })

    fireEvent.click(screen.getByRole('tab', { name: '默认音色' }))
    fireEvent.click(screen.getByRole('tab', { name: '角色与人格' }))
    expect((screen.getByPlaceholderText(/给 AI 起个名字/) as HTMLInputElement).value).toBe('临时称呼')
  })

  it('离开分类后记住上次任务页签：模型页返回仍在引擎管理', async () => {
    render(wrap(<SettingsPage />))
    fireEvent.click(screen.getByRole('button', { name: /模型/ }))
    fireEvent.click(await screen.findByRole('tab', { name: '引擎管理' }))
    expect(screen.getByRole('tab', { name: '引擎管理' }).getAttribute('aria-selected')).toBe('true')

    fireEvent.click(screen.getByRole('button', { name: /通用/ }))
    fireEvent.click(screen.getByRole('button', { name: /模型/ }))
    const restored = await screen.findByRole('tab', { name: '引擎管理' })
    expect(restored.getAttribute('aria-selected')).toBe('true')
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

describe('SettingsPage 键盘可达（2026-11-20：↑/↓/Home/End roving focus + `/` 聚焦搜索 + Esc 清词）', () => {
  it('导航方向键在可见导航项间移焦（↓/↑ 环绕，Home/End 跳端）', () => {
    const { container } = render(wrap(<SettingsPage />))
    const list = container.querySelector('.settings-nav__list') as HTMLElement
    const items = () => Array.from(container.querySelectorAll<HTMLButtonElement>('.settings-nav-item'))
    expect(items().length).toBe(15)
    const first = items()[0]
    first.focus()
    expect(document.activeElement).toBe(first)
    fireEvent.keyDown(list, { key: 'ArrowDown' })
    expect(document.activeElement).toBe(items()[1])
    fireEvent.keyDown(list, { key: 'ArrowUp' })
    expect(document.activeElement).toBe(items()[0])
    fireEvent.keyDown(list, { key: 'ArrowUp' })
    expect(document.activeElement).toBe(items()[items().length - 1])
    fireEvent.keyDown(list, { key: 'Home' })
    expect(document.activeElement).toBe(items()[0])
    fireEvent.keyDown(list, { key: 'End' })
    expect(document.activeElement).toBe(items()[items().length - 1])
  })

  it('`/` 聚焦搜索框（页面级监听，输入态内不劫持）', () => {
    const { container } = render(wrap(<SettingsPage />))
    const nav = container.querySelector('.settings-nav__list') as HTMLElement
    fireEvent.keyDown(nav, { key: '/' })
    const input = container.querySelector('.settings-page__search input') as HTMLInputElement
    expect(document.activeElement).toBe(input)
    // 焦点已在输入框内再按 / 不重入（target=INPUT 被忽略，输入不被打断）
    fireEvent.keyDown(input, { key: '/' })
    expect(document.activeElement).toBe(input)
  })

  it('Esc：有词清词恢复全量导航，无词退焦回页面', () => {
    const { container } = render(wrap(<SettingsPage />))
    const input = container.querySelector('.settings-page__search input') as HTMLInputElement
    fireEvent.change(input, { target: { value: '外观' } })
    expect(container.querySelectorAll('.settings-nav-item').length).toBeLessThan(15)
    fireEvent.keyDown(input, { key: 'Escape' })
    expect(container.querySelectorAll('.settings-nav-item').length).toBe(15)
    fireEvent.keyDown(input, { key: 'Escape' })
    expect(document.activeElement).not.toBe(input)
  })

  it('搜索命中段锚定：过滤态切组后组内命中查询词的段卡出 settings-sec-flash', async () => {
    const { container } = render(wrap(<SettingsPage />))
    // 「记忆」命中办公组关键词 → 自动切办公组 → 面板内「办公记忆」strong 命中 → 段卡闪示
    // （段标题随 bridge 数据异步挂载，组件侧短窗轮询，waitFor 容忍挂载延迟）
    fireEvent.change(screen.getByPlaceholderText(/搜索设置项/), { target: { value: '记忆' } })
    await waitFor(
      () => expect(container.querySelector('.settings-sec-flash')).toBeTruthy(),
      { timeout: 3000 },
    )
  })
})
