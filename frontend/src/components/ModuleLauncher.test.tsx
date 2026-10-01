import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import ModuleLauncher from './ModuleLauncher'
import { LocaleProvider } from '../gaea/lib/i18n'

// ModuleLauncher 双首页（v4.182）：空间切换器迁首页顶栏 + 书斋/闲庭两变体版式。
// 断言焦点：切换器（SHELL_SPACES 驱动、aria-pressed、点击回调）与两空间
// 版式分化（书斋=文档流水面板；闲庭=旗舰横幅画廊+无遥测右舷）。
// 数据 hooks（遥测/会话/记忆）后端未就绪时静默兜底，无需 mock；语音 hook 整体
// mock（真实实现依赖 Wails EventsOn）。
// v10 卡片工作台追加断言：书斋改卡片网格（w-board / ml-card）——指挥卡 w-hero
// （h1.w-hero-title 文书台 / w-hero-lede / w-seal 印章 / w-hero-side 内空间 chip）、
// 文档卡 desk-recent-docs（ml-card-head + w-docs 行）、能力卡网格 w-modules
// （旗舰 w-mod.is-featured + 普通 w-mod，v9 案牌 w-plaque 退役）、状态卡排
// w-stat-row（4×w-stat，aria-label 用 zh 精确文案）；命令条 w-cmd 保留而
// v9 的 w-deck/w-masthead/w-plaques/w-vitals 在书斋全部退役；闲庭版式不受影响。

const bridgeMocks = vi.hoisted(() => ({
  app: {
    ListSessions: vi.fn(async () => []),
    MemoryHubOverview: vi.fn(async () => ({})),
    VoiceApplySettings: vi.fn(async () => ({})),
    VoiceChatText: vi.fn(async () => ({})),
    // 7.3-1 任务收件箱：四方法桩（bridge 签名由主代理收口时补）。
    // 返回值给完整最小样本（TaskInboxView 必填字段齐——类型即契约防漂移）。
    GaeaTaskInboxList: vi.fn(async (): Promise<import('../gaea/lib/types').TaskInboxView[]> => []),
    GaeaTaskInboxSave: vi.fn(async (): Promise<import('../gaea/lib/types').TaskInboxView> =>
      ({ id: 'ti-mock', title: 't', space: 'work', status: 'pending', source: 'inbox', createdAt: 0, updatedAt: 0 })),
    GaeaTaskInboxSetStatus: vi.fn(async (): Promise<import('../gaea/lib/types').TaskInboxView> =>
      ({ id: 'ti-mock', title: 't', space: 'work', status: 'pending', source: 'inbox', createdAt: 0, updatedAt: 0 })),
    GaeaTaskInboxDelete: vi.fn(async (): Promise<void> => {}),
  },
}))

vi.mock('../gaea/lib/bridge', () => ({ app: bridgeMocks.app }))
vi.mock('../hooks/useVoiceChat', () => ({
  useVoiceChat: () => ({
    state: { active: false, listening: false, aiSpeaking: false, error: null },
    start: vi.fn(),
    stop: vi.fn(),
    interrupt: vi.fn(),
  }),
}))

const wrap = (ui: ReactElement) => {
  localStorage.setItem('gaea-lang', 'zh')
  return <LocaleProvider>{ui}</LocaleProvider>
}

describe('ModuleLauncher 双空间首页（v4.182 书斋/闲庭）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('顶栏渲染空间切换器（SHELL_SPACES 驱动：当前空间 aria-pressed，点击另一空间回调 onSwitchSpace）', () => {
    const onSwitch = vi.fn()
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={onSwitch} />))
    const work = screen.getByTestId('ml-space-work')
    const play = screen.getByTestId('ml-space-play')
    expect(work.textContent).toBe('书斋')
    expect(play.textContent).toBe('闲庭')
    expect(work.getAttribute('aria-pressed')).toBe('true')
    expect(play.getAttribute('aria-pressed')).toBe('false')
    fireEvent.click(play)
    expect(onSwitch).toHaveBeenCalledWith('play')
    // 当前空间按钮点击不重复触发
    fireEvent.click(work)
    expect(onSwitch).toHaveBeenCalledTimes(1)
  })

  it('1B 前置：切换钮与空间 chip 的 title 说明「仅导航，不影响办公引擎空间」', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    const hint = '不影响办公引擎空间'
    expect(screen.getByTestId('ml-space-work').getAttribute('title')).toContain(hint)
    expect(screen.getByTestId('ml-space-play').getAttribute('title')).toContain(hint)
    expect(screen.getByTestId('ml-space-chip').getAttribute('title')).toContain(hint)
  })

  it('书斋（work）：文档流水面板为主角 + 空间 chip；闲庭板块卡不出现', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    expect(screen.getByTestId('desk-recent-docs')).toBeTruthy()
    expect(screen.getByTestId('ml-space-chip').textContent).toBe('书斋')
    // 闲庭版式元素不在书斋出现
    expect(screen.queryByTestId('garden-banner-impl')).toBeNull()
  })

  it('闲庭（play）：全幅画廊——旗舰横幅 + 激活态，书斋文档流水不出现', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="play" onSwitchSpace={vi.fn()} />))
    const play = screen.getByTestId('ml-space-play')
    expect(play.getAttribute('aria-pressed')).toBe('true')
    // 闲庭版式：会客厅旗舰横幅在画廊渲染；书斋专有元素（文档流水/chip）不渲染
    expect(document.querySelector('.garden-banner')).toBeTruthy()
    expect(screen.queryByTestId('desk-recent-docs')).toBeNull()
    expect(screen.queryByTestId('ml-space-chip')).toBeNull()
  })
})

describe('书斋 v10 卡片工作台版式', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('指挥卡——台名大标 + 印章装饰 + 空间 chip + 就绪徽记（v9 开放排印身份区退役）', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    // 全宽指挥卡：h1.w-hero-title = home.title「文书台」
    expect(document.querySelector('.w-hero')).toBeTruthy()
    const title = screen.getByText('文书台')
    expect(title.tagName.toLowerCase()).toBe('h1')
    expect(title.classList.contains('w-hero-title')).toBe(true)
    // 题辞：w-hero-lede 承载 home.sub 长文案（锁存在 + 非空，不锁全串避免实现侧标点级脆断）
    const lede = document.querySelector('.w-hero-lede')
    expect(lede).toBeTruthy()
    expect((lede?.textContent ?? '').length).toBeGreaterThan(0)
    // 印章：纯装饰（aria-hidden），内容为空间名首字「书」
    const seal = document.querySelector('.w-seal')
    expect(seal).toBeTruthy()
    expect(seal?.getAttribute('aria-hidden')).toBe('true')
    expect(seal?.textContent).toBe('书')
    // 侧翼：既有空间 chip 落位 w-hero-side，就绪徽记 = home.pill（zh.ts 精确串）
    const side = document.querySelector('.w-hero-side')
    expect(side).toBeTruthy()
    expect(side?.querySelector('[data-testid="ml-space-chip"]')).toBeTruthy()
    expect(screen.getByText('GAEA 已就绪 · 本地 AI 创作中枢')).toBeTruthy()
    // v9 的 open-masthead 结构在书斋退役；命令条本体保留
    expect(document.querySelector('.w-masthead')).toBeNull()
    expect(document.querySelector('.w-cmd')).toBeTruthy()
  })

  it('书斋首屏为卡片网格：指挥卡 + 文档卡 + 侧列 + 能力卡排 + 状态卡排', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
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

  it('能力矩阵为卡片网格：旗舰跨列大卡 + 模块卡，v9 案牌/行式索引退役', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    expect(document.querySelector('.w-modules')).toBeTruthy()
    // fallback 清单派生：静态 canonicalBoards 在 work 空间可得 6 模块
    // （gaea/cost/memoryhub/weixin/modelcenter/settings）；旗舰是否跨列由实现定
    // → 宽松断言只锁「非空 + 全部为按钮 + 恰有一张旗舰卡」。
    const mods = document.querySelectorAll('.w-mod')
    expect(mods.length).toBeGreaterThan(0)
    mods.forEach((p) => expect(p.tagName.toLowerCase()).toBe('button'))
    expect(document.querySelectorAll('.w-mod.is-featured').length).toBe(1)
    expect(document.querySelector('.w-plaque')).toBeNull()
    expect(document.querySelector('.w-index-item')).toBeNull()
  })

  it('状态卡排四张卡齐备（内核/会话/记忆/任务，aria-label 对齐 zh 精确文案）', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    expect(document.querySelector('.w-stat-row')).toBeTruthy()
    const stats = document.querySelectorAll('.w-stat')
    expect(stats.length).toBe(4)
    // zh.ts 精确键值：home.kernel=内核状态、shell.launcher.sessions=最近会话、
    // shell.launcher.memoryPulse=记忆脉搏、shell.launcher.taskInbox=任务
    const labels = Array.from(stats).map((el) => el.getAttribute('aria-label'))
    expect(labels).toContain('内核状态')
    expect(labels).toContain('最近会话')
    expect(labels).toContain('记忆脉搏')
    expect(labels).toContain('任务')
    // 状态卡同款卡片壳（与指挥卡/文档卡同一 .ml-card 基底）
    stats.forEach((el) => expect(el.classList.contains('ml-card')).toBe(true))
  })

  it('闲庭不受 v10 卡片化影响', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="play" onSwitchSpace={vi.fn()} />))
    expect(document.querySelector('.w-hero')).toBeNull()
    expect(document.querySelector('.w-modules')).toBeNull()
    expect(document.querySelector('.w-stat-row')).toBeNull()
    expect(document.querySelector('.w-masthead')).toBeNull()
    expect(document.querySelector('.w-plaques')).toBeNull()
    expect(document.querySelector('.garden-banner')).toBeTruthy()
  })
})

// 7.3-1 任务收件箱双空间挂点：书斋 w-vitals 第五节（desk-task-inbox）+
// 闲庭 p-foot 第五节（garden-task-inbox，跨全列不破坏既有四节）；挂点计数
// 一次性读 GaeaTaskInboxList(space)；点击开 TaskInboxPanel 单例（space 跟随
// 当前 home 空间，面板内 List 同空间）。
describe('ModuleLauncher 任务收件箱挂点（7.3-1）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('书斋：w-vitals 任务节渲染，挂载一次性拉 work 待处理数并展示计数', async () => {
    bridgeMocks.app.GaeaTaskInboxList.mockResolvedValue([
      { id: 'ti-a', title: 'a', space: 'work', status: 'pending', source: 'ctrlk', createdAt: 1, updatedAt: 1 },
      { id: 'ti-b', title: 'b', space: 'work', status: 'pending', source: 'voice', createdAt: 2, updatedAt: 2 },
      { id: 'ti-c', title: 'c', space: 'work', status: 'done', source: 'inbox', createdAt: 3, updatedAt: 3 },
    ])
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    const sec = screen.getByTestId('desk-task-inbox')
    expect(sec).toBeTruthy()
    expect(sec.textContent).toContain('任务')
    // 一次性读：work 空间、pending 计数 2（done 不计）
    await vi.waitFor(() => expect(bridgeMocks.app.GaeaTaskInboxList).toHaveBeenCalledWith('work'))
    await vi.waitFor(() => expect(sec.textContent).toContain('待处理 2 项'))
    // 零轮询：无定时器增量重拉（挂载后调用数稳定）
    const calls = bridgeMocks.app.GaeaTaskInboxList.mock.calls.length
    await new Promise((r) => setTimeout(r, 30))
    expect(bridgeMocks.app.GaeaTaskInboxList.mock.calls.length).toBe(calls)
  })

  it('书斋：点「打开收件箱」开面板单例（space=work），面板内按 work 再拉', async () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    await vi.waitFor(() => expect(bridgeMocks.app.GaeaTaskInboxList).toHaveBeenCalled())
    fireEvent.click(screen.getByTestId('task-inbox-open-btn'))
    // antd Modal portal 到 body：document 直查面板
    await vi.waitFor(() => expect(document.querySelector('[data-testid="task-inbox-panel"]')).toBeTruthy())
    // 面板 open → 按 work 空间拉清单（空间隔离判据③）
    await vi.waitFor(() => expect(bridgeMocks.app.GaeaTaskInboxList).toHaveBeenCalledWith('work'))
  })

  it('闲庭：p-foot 第五节渲染且既有四节 testid 不受影响（布局铁律）', async () => {
    bridgeMocks.app.GaeaTaskInboxList.mockResolvedValue([])
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="play" onSwitchSpace={vi.fn()} />))
    // 既有四节齐备（零回归）
    expect(screen.getByTestId('garden-progress')).toBeTruthy()
    expect(screen.getByTestId('garden-sessions')).toBeTruthy()
    expect(screen.getByTestId('garden-memory')).toBeTruthy()
    expect(screen.getByTestId('garden-meters')).toBeTruthy()
    // 第五节：任务收件箱（跨全列形态）
    const sec = screen.getByTestId('garden-task-inbox')
    expect(sec).toBeTruthy()
    expect(sec.textContent).toContain('任务')
    // 闲庭侧按 play 拉取
    await vi.waitFor(() => expect(bridgeMocks.app.GaeaTaskInboxList).toHaveBeenCalledWith('play'))
  })

  it('闲庭：点开面板单例 space=play（两空间各查各空间）', async () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="play" onSwitchSpace={vi.fn()} />))
    await vi.waitFor(() => expect(bridgeMocks.app.GaeaTaskInboxList).toHaveBeenCalledWith('play'))
    fireEvent.click(screen.getByTestId('task-inbox-open-btn'))
    await vi.waitFor(() => expect(document.querySelector('[data-testid="task-inbox-panel"]')).toBeTruthy())
  })

// ── 7.3-2 板块降级为任务视图：homeLayout 分支（classic 零变化由上方既有用例
// 钉死；此处只钉 tasks 分支与快捷切换钮）──
describe('ModuleLauncher 首页形态分支（7.3-2 层跃升）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.removeItem('gaea.home.layout')
  })

  it('缺省 classic：渲染既有书斋首页，快捷钮为「任务优先」入口', async () => {
    const { useAppStore } = await import('../stores/appStore')
    useAppStore.setState({ homeLayout: 'classic' })
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    expect(screen.getByTestId('desk-task-inbox')).toBeTruthy()
    expect(screen.queryByTestId('tasks-first-home')).toBeNull()
    expect(screen.getByTestId('home-layout-tasks-entry')).toBeTruthy()
  })

  it('tasks 形态：渲染任务优先首页（收件箱首屏）；快捷钮变「切回经典」', async () => {
    const { useAppStore } = await import('../stores/appStore')
    useAppStore.setState({ homeLayout: 'tasks' })
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    expect(screen.getByTestId('tasks-first-home')).toBeTruthy()
    expect(screen.getByTestId('tasks-first-inbox')).toBeTruthy()
    expect(screen.queryByTestId('desk-task-inbox')).toBeNull()
    expect(screen.getByTestId('home-layout-back-classic')).toBeTruthy()
    useAppStore.setState({ homeLayout: 'classic' })
  })
})
})
