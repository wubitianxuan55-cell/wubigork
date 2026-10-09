import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import ModuleLauncher from './ModuleLauncher'
import { LocaleProvider } from '../gaea/lib/i18n'

// ModuleLauncher 统一首页（v4.475 书斋/闲庭合一；前身 v4.182 双空间首页）：
// 首页不再按壳层空间分裂版式——空间切换钮退役（切空间由板块导航隐式驱动），
// 能力矩阵全板块出卡（书斋+闲庭一屏尽收），收件箱跨空间全量（列表 scope=''，
// 新建任务仍按当前壳层空间落标签）。
// 数据 hooks（遥测/会话/记忆）后端未就绪时静默兜底，无需 mock；语音 hook 整体
// mock（真实实现依赖 Wails EventsOn）。
// 断言焦点：统一台版式（w-hero 指挥卡 / desk-recent-docs / w-modules 全量网格 /
// w-stat-row 状态带）、收件箱挂点与面板的跨空间口径、homeLayout 形态分支。

const bridgeMocks = vi.hoisted(() => ({
  app: {
    ListSessions: vi.fn(async () => []),
    MemoryHubOverview: vi.fn(async () => ({})),
    VoiceApplySettings: vi.fn(async () => ({})),
    VoiceChatText: vi.fn(async () => ({})),
    // 7.3-1 任务收件箱：四方法桩（返回值给完整最小样本——类型即契约防漂移）。
    GaeaTaskInboxList: vi.fn(async (): Promise<import('../gaea/lib/types').TaskInboxView[]> => []),
    GaeaTaskInboxSave: vi.fn(async (): Promise<import('../gaea/lib/types').TaskInboxView> =>
      ({ id: 'ti-mock', title: 't', space: 'work', status: 'pending', source: 'inbox', createdAt: 0, updatedAt: 0 })),
    GaeaTaskInboxSetStatus: vi.fn(async (): Promise<import('../gaea/lib/types').TaskInboxView> =>
      ({ id: 'ti-mock', title: 't', space: 'work', status: 'pending', source: 'inbox', createdAt: 0, updatedAt: 0 })),
    GaeaTaskInboxDelete: vi.fn(async (): Promise<void> => {}),
  },
}))

vi.mock('../gaea/lib/bridge', () => ({ app: bridgeMocks.app }))
// FE6-02：语音 hook 可切换降级位（默认 null = 既有用例口径不变）
const voiceMock = vi.hoisted(() => ({
  state: { active: false, listening: false, aiSpeaking: false, error: null, degraded: null as string | null },
}))
vi.mock('../hooks/useVoiceChat', () => ({
  useVoiceChat: () => ({
    state: voiceMock.state,
    start: vi.fn(),
    stop: vi.fn(),
    interrupt: vi.fn(),
  }),
}))

const wrap = (ui: ReactElement) => {
  localStorage.setItem('gaea-lang', 'zh')
  return <LocaleProvider>{ui}</LocaleProvider>
}

describe('ModuleLauncher 统一台首页（v4.475 书斋/闲庭合一）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('顶栏不再有空间切换钮（切空间由板块导航隐式驱动）；首页形态快捷钮仍在', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    expect(screen.queryByTestId('ml-space-switch')).toBeNull()
    expect(screen.queryByTestId('ml-space-work')).toBeNull()
    expect(screen.queryByTestId('ml-space-play')).toBeNull()
    expect(screen.getByTestId('home-layout-tasks-entry')).toBeTruthy()
  })

  it('指挥卡——台名大标 + 印章（台名首字，去空间语义）+ 就绪徽记；空间 chip 退役', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    // 全宽指挥卡：h1.w-hero-title = home.title「文书台」
    expect(document.querySelector('.w-hero')).toBeTruthy()
    const title = screen.getByText('文书台')
    expect(title.tagName.toLowerCase()).toBe('h1')
    expect(title.classList.contains('w-hero-title')).toBe(true)
    // 题辞：w-hero-lede 承载 home.sub 长文案（锁存在 + 非空，不锁全串避免标点级脆断）
    const lede = document.querySelector('.w-hero-lede')
    expect(lede).toBeTruthy()
    expect((lede?.textContent ?? '').length).toBeGreaterThan(0)
    // 印章：纯装饰（aria-hidden），内容为台名首字「文」（v4.475 起不再取空间名首字）
    const seal = document.querySelector('.w-seal')
    expect(seal).toBeTruthy()
    expect(seal?.getAttribute('aria-hidden')).toBe('true')
    expect(seal?.textContent).toBe('文')
    // 空间 chip 退役（合一后首页无空间身份）；就绪徽记 = home.pill（zh.ts 精确串）
    expect(screen.queryByTestId('ml-space-chip')).toBeNull()
    expect(screen.getByText('GAEA 已就绪 · 本地 AI 创作中枢')).toBeTruthy()
    // v9 的 open-masthead 结构退役；命令条本体保留
    expect(document.querySelector('.w-masthead')).toBeNull()
    expect(document.querySelector('.w-cmd')).toBeTruthy()
  })

  it('能力矩阵全板块出卡且全卡同权：闲庭与书斋板块同屏，无旗舰/徽记/箭头（v4.476 极简化）', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    expect(document.querySelector('.w-modules')).toBeTruthy()
    // 静态 canonicalBoards 全量派生：10 业务板块 + settings，11 张同构卡
    const mods = document.querySelectorAll('.w-mod')
    expect(mods.length).toBe(11)
    mods.forEach((p) => expect(p.tagName.toLowerCase()).toBe('button'))
    // 全卡同权：无旗舰/徽记/水印/悬停箭头，设置也是普通卡
    expect(document.querySelectorAll('.w-mod.is-featured').length).toBe(0)
    expect(document.querySelectorAll('.w-mod.is-settings').length).toBe(0)
    expect(document.querySelectorAll('.w-mod-badge, .w-mod-mark, .w-mod-go, .w-mod-arrow').length).toBe(0)
    // 每卡=图标座+名称+一行描述三件套
    mods.forEach((p) => {
      expect(p.querySelector('.w-mod-icon')).toBeTruthy()
      expect(p.querySelector('.w-mod-name')).toBeTruthy()
      expect(p.querySelector('.w-mod-desc')).toBeTruthy()
    })
    // 闲庭板块卡可见（合一判据：不再按 work/play 过滤）
    expect(screen.getByText('小说')).toBeTruthy()
    expect(screen.getByText('原罪')).toBeTruthy()
    expect(screen.getByText('绘梦')).toBeTruthy()
    expect(screen.getByText('聊天')).toBeTruthy()
    // 书斋板块卡同屏（办公不再有旗舰待遇，同样是一张普通卡）
    expect(screen.getByText('造价数据库')).toBeTruthy()
    expect(screen.getByText('办公')).toBeTruthy()
    // 画廊版式残留不出现（GardenHome 系随合一退役）
    expect(document.querySelector('.garden-banner')).toBeNull()
    expect(document.querySelector('.p-board')).toBeNull()
  })

  it('首屏为卡片网格：指挥卡 + 文档卡 + 侧列 + 能力卡排 + 状态卡排', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    expect(document.querySelector('.w-board')).toBeTruthy()
    // 最近文档：卡片形态（卡头 + 行式文档流），testid 契约不变
    const docs = screen.getByTestId('desk-recent-docs')
    expect(docs.classList.contains('ml-card')).toBe(true)
    expect(docs.querySelector('.ml-card-head')).toBeTruthy()
    // 写作进度卡（侧列内，共享 ml-ring 原语）
    const progress = document.querySelector('.w-progress')
    expect(progress).toBeTruthy()
    expect(progress?.querySelector('.ml-ring')).toBeTruthy()
  })

  it('状态卡排四张卡齐备（内核/会话/记忆/任务，aria-label 对齐 zh 精确文案）', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    expect(document.querySelector('.w-stat-row')).toBeTruthy()
    const stats = document.querySelectorAll('.w-stat')
    expect(stats.length).toBe(4)
    const labels = Array.from(stats).map((el) => el.getAttribute('aria-label'))
    expect(labels).toContain('内核状态')
    expect(labels).toContain('最近会话')
    expect(labels).toContain('记忆脉搏')
    expect(labels).toContain('任务')
    stats.forEach((el) => expect(el.classList.contains('ml-card')).toBe(true))
  })
})

// 7.3-1 任务收件箱挂点：desk-task-inbox 状态带第五节。v4.475 首页合一：
// 挂点计数跨空间全量（GaeaTaskInboxList('')）；面板单例同口径，新建任务仍
// 按当前壳层空间落标签（Go ValidSpace 仅收 work|play）。
describe('ModuleLauncher 任务收件箱挂点（7.3-1 + v4.475 跨空间口径）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('任务节渲染，挂载一次性拉全量清单（scope=""）并展示跨空间待处理计数', async () => {
    bridgeMocks.app.GaeaTaskInboxList.mockResolvedValue([
      { id: 'ti-a', title: 'a', space: 'work', status: 'pending', source: 'ctrlk', createdAt: 1, updatedAt: 1 },
      { id: 'ti-b', title: 'b', space: 'play', status: 'pending', source: 'voice', createdAt: 2, updatedAt: 2 },
      { id: 'ti-c', title: 'c', space: 'work', status: 'done', source: 'inbox', createdAt: 3, updatedAt: 3 },
    ])
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    const sec = screen.getByTestId('desk-task-inbox')
    expect(sec).toBeTruthy()
    expect(sec.textContent).toContain('任务')
    // 跨空间全量：scope 空串（后端 FilterBySpace('')=全部）
    await vi.waitFor(() => expect(bridgeMocks.app.GaeaTaskInboxList).toHaveBeenCalledWith(''))
    await vi.waitFor(() => expect(sec.textContent).toContain('待处理 2 项'))
    // 零轮询：无定时器增量重拉（挂载后调用数稳定）
    const calls = bridgeMocks.app.GaeaTaskInboxList.mock.calls.length
    await new Promise((r) => setTimeout(r, 30))
    expect(bridgeMocks.app.GaeaTaskInboxList.mock.calls.length).toBe(calls)
  })

  it('点「打开收件箱」开面板单例：面板内按全量再拉，跨空间行带归属空间标', async () => {
    bridgeMocks.app.GaeaTaskInboxList.mockResolvedValue([
      { id: 'ti-a', title: '书斋侧任务', space: 'work', status: 'pending', source: 'ctrlk', createdAt: 1, updatedAt: 1 },
      { id: 'ti-b', title: '闲庭侧任务', space: 'play', status: 'pending', source: 'voice', createdAt: 2, updatedAt: 2 },
    ])
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    await vi.waitFor(() => expect(bridgeMocks.app.GaeaTaskInboxList).toHaveBeenCalledWith(''))
    fireEvent.click(screen.getByTestId('task-inbox-open-btn'))
    // antd Modal portal 到 body：document 直查面板
    await vi.waitFor(() => expect(document.querySelector('[data-testid="task-inbox-panel"]')).toBeTruthy())
    // 面板 open → 全量再拉（scope=''）
    await vi.waitFor(() => expect(bridgeMocks.app.GaeaTaskInboxList).toHaveBeenCalledWith(''))
    // 跨空间混合清单：行内带归属空间标（书斋/闲庭）
    await vi.waitFor(() => expect(screen.getByText('闲庭')).toBeTruthy())
    expect(screen.getByText('书斋')).toBeTruthy()
  })

  it('面板内新建任务：保存仍按当前壳层空间落标签（列表全量 ≠ 标签漂移）', async () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    await vi.waitFor(() => expect(bridgeMocks.app.GaeaTaskInboxList).toHaveBeenCalled())
    fireEvent.click(screen.getByTestId('task-inbox-open-btn'))
    await vi.waitFor(() => expect(document.querySelector('[data-testid="task-inbox-panel"]')).toBeTruthy())
    const input = screen.getByTestId('task-inbox-input')
    fireEvent.change(input, { target: { value: '新任务' } })
    fireEvent.click(screen.getByTestId('task-inbox-add'))
    await vi.waitFor(() => expect(bridgeMocks.app.GaeaTaskInboxSave).toHaveBeenCalled())
    const saveCalls = bridgeMocks.app.GaeaTaskInboxSave.mock.calls as unknown as string[][]
    const payload = JSON.parse(saveCalls[0][0])
    expect(payload.title).toBe('新任务')
    expect(payload.space).toBe('work') // 测试环境 localStorage 空 → appStore.space 缺省 work
    expect(payload.source).toBe('inbox')
  })
})

// ── 7.3-2 板块降级为任务视图：homeLayout 分支（合一后台版式不分空间）──
describe('ModuleLauncher 首页形态分支（7.3-2 层跃升）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.removeItem('gaea.home.layout')
  })

  it('缺省 classic：渲染统一台首页，快捷钮为「任务优先」入口', async () => {
    const { useAppStore } = await import('../stores/appStore')
    useAppStore.setState({ homeLayout: 'classic' })
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    expect(screen.getByTestId('desk-task-inbox')).toBeTruthy()
    expect(screen.queryByTestId('tasks-first-home')).toBeNull()
    expect(screen.getByTestId('home-layout-tasks-entry')).toBeTruthy()
  })

  it('tasks 形态：渲染任务优先首页（收件箱首屏）；快捷钮变「切回经典」', async () => {
    const { useAppStore } = await import('../stores/appStore')
    useAppStore.setState({ homeLayout: 'tasks' })
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    expect(screen.getByTestId('tasks-first-home')).toBeTruthy()
    expect(screen.getByTestId('tasks-first-inbox')).toBeTruthy()
    expect(screen.queryByTestId('desk-task-inbox')).toBeNull()
    expect(screen.getByTestId('home-layout-back-classic')).toBeTruthy()
    useAppStore.setState({ homeLayout: 'classic' })
  })
})

// ── FE6-02：主壳命令条不得在麦克风不可用时宣称「聆听中」 ──
describe('ModuleLauncher 语音降级可见化（FE6-02）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    voiceMock.state = { active: true, listening: false, aiSpeaking: false, error: null, degraded: 'mic-unavailable' }
  })
  afterEach(() => {
    voiceMock.state = { active: false, listening: false, aiSpeaking: false, error: null, degraded: null }
  })

  it('degraded=mic-unavailable → 命令条出现中文警示，且不再显示「聆听中」', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    const warn = screen.getByTestId('voice-mic-degraded')
    expect(warn.textContent).toContain('麦克风不可用，未在采集音频，本回合走文本输入')
    expect(screen.getByText('麦克风不可用')).toBeTruthy()
    expect(screen.queryByText('正在聆听')).toBeNull()
  })

  it('未降级 → 不出现警示（反向验证：不是恒亮）', () => {
    voiceMock.state = { active: false, listening: false, aiSpeaking: false, error: null, degraded: null }
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    expect(screen.queryByTestId('voice-mic-degraded')).toBeNull()
  })
})
