import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import ModuleLauncher from './ModuleLauncher'
import { LocaleProvider } from '../gaea/lib/i18n'
import { FRONTEND_EVENTS, OPEN_TOOL_FLAG } from '../events'
import { clearRecentFilesForTest, recordRecentFile } from '../gaea/lib/recentFiles'

// ModuleLauncher 首页（方向一「输入即首页」，2026-10-26 拍板；调研
// docs/gaea-home-redesign-survey-2026-10.md）：classic 形态从六区收敛为中轴
// 三段——开始（问候+命令条+建议词）/ 继续（最近文档+最近会话）/ 待办一行。
// 12 板块卡与独立工具节退役（导航归 rail+命令面板；工具收进命令条「工具」
// 下拉，OPEN_TOOL 双通道机制原样）；状态卡排收为待办一行。
// 数据 hooks（会话）后端未就绪时静默兜底，无需 mock；语音 hook 整体 mock
// （真实实现依赖 Wails EventsOn）；paneFileOpen mock（走桥的打开动作不属本测）。
// 断言焦点：三段结构、退役面零残留（12 卡/工具节/状态排/进度环）、建议词
// 填入命令条、工具下拉双通道、最近行真打开、收件箱挂点跨空间口径、形态分支。

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
vi.mock('../gaea/lib/paneFileOpen', () => ({ openPaneFileOrPreview: vi.fn() }))
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

// 最近文档种子跨用例隔离（localStorage 单源 + 解析缓存都是模块级状态，
// 直写 localStorage 不失效缓存——走模块自身的 record/clear 辅助）
beforeEach(() => {
  clearRecentFilesForTest()
})

describe('ModuleLauncher 输入即首页（方向一 · 2026-10-26）', () => {
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

  it('开始段：时辰问候大标（衬线）+ 印章「文」+ 题辞 + 大命令条；就绪 pill 退役', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    expect(document.querySelector('.w-greet')).toBeTruthy()
    const h1 = document.querySelector('.w-greet h1.w-hero-title')
    expect(h1).toBeTruthy()
    expect(['早上好', '下午好', '晚上好', '夜深了']).toContain(h1?.textContent ?? '')
    // 印章：纯装饰（aria-hidden），内容为台名首字「文」（身份记号保留）
    const seal = document.querySelector('.w-greet .w-seal')
    expect(seal).toBeTruthy()
    expect(seal?.getAttribute('aria-hidden')).toBe('true')
    expect(seal?.textContent).toBe('文')
    // 题辞：home.sub 长文案（锁存在 + 非空）
    const lede = document.querySelector('.w-greet .w-hero-lede')
    expect(lede).toBeTruthy()
    expect((lede?.textContent ?? '').length).toBeGreaterThan(0)
    // 命令条本体保留；空间 chip / v9 masthead / 就绪 pill 全退役
    expect(document.querySelector('.w-cmd')).toBeTruthy()
    expect(screen.queryByTestId('ml-space-chip')).toBeNull()
    expect(document.querySelector('.w-masthead')).toBeNull()
    expect(screen.queryByText('GAEA 已就绪 · 本地 AI 创作中枢')).toBeNull()
  })

  it('六区退役面零残留：板块卡网格/工具节/状态卡排/侧列进度环/画廊版式全 null', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    expect(document.querySelector('[data-testid="home-boards"]')).toBeNull()
    expect(document.querySelector('[data-testid="home-tools"]')).toBeNull()
    expect(document.querySelector('.w-modules')).toBeNull()
    expect(document.querySelector('.w-mod')).toBeNull()
    expect(document.querySelector('.w-stat-row')).toBeNull()
    expect(document.querySelector('.w-stat')).toBeNull()
    expect(document.querySelector('.w-aside')).toBeNull()
    expect(document.querySelector('.w-progress')).toBeNull()
    expect(document.querySelector('.w-hero-pill')).toBeNull()
    expect(document.querySelector('.garden-banner')).toBeNull()
    expect(document.querySelector('.p-board')).toBeNull()
    // 板块名称不再上首页（导航归常驻 rail + 命令面板）
    expect(screen.queryByText('造价数据库')).toBeNull()
    expect(screen.queryByText('板块')).toBeNull()
  })

  it('建议词（NN/g：具体场景）点击=填入命令条可改可发，不直接发送', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    const chips = document.querySelectorAll('.w-chip')
    expect(chips.length).toBe(4)
    const chip1 = screen.getByTestId('home-sug-1')
    expect(chip1.textContent).toBe('帮我把这份文档的要点整理成一页纪要')
    fireEvent.click(chip1)
    const input = document.querySelector('input.w-cmd-input') as HTMLInputElement
    expect(input.value).toBe('帮我把这份文档的要点整理成一页纪要')
    // 只填入不发送：VoiceChatText 零调用（发送留给用户回车/点发送）
    expect(bridgeMocks.app.VoiceChatText).not.toHaveBeenCalled()
  })

  it('命令条「工具」下拉（§7.5 刀6 机制随迁）：CAD/进度两档，点击=OPEN_TOOL 事件+冷启标记+导航办公', async () => {
    sessionStorage.clear()
    const onNavigate = vi.fn()
    render(wrap(<ModuleLauncher onNavigate={onNavigate} />))
    fireEvent.click(screen.getByTestId('home-tools-btn'))
    const cad = await screen.findByTestId('home-tool-cad')
    const sched = screen.getByTestId('home-tool-schedule')
    expect(cad.textContent).toContain('CAD 看图')
    expect(sched.textContent).toContain('进度计划')
    // 点击 cad：双通道全发——事件（已挂载直达 setActiveTool）+ sessionStorage 标记
    //（办公 App 未懒加载挂载时由其挂载效应补开）+ 导航办公（不导航去板块房间）
    const seen: Array<{ name: string; detail: unknown }> = []
    const rec = (ev: Event) => seen.push({ name: ev.type, detail: (ev as CustomEvent).detail })
    window.addEventListener(FRONTEND_EVENTS.OPEN_TOOL, rec)
    fireEvent.click(cad)
    window.removeEventListener(FRONTEND_EVENTS.OPEN_TOOL, rec)
    expect(seen).toEqual([{ name: FRONTEND_EVENTS.OPEN_TOOL, detail: { tool: 'cad' } }])
    expect(sessionStorage.getItem(OPEN_TOOL_FLAG)).toBe('cad')
    expect(onNavigate).toHaveBeenCalledWith('gaea')
    // 点击 schedule 同链路（菜单已关，重开再点；标记覆写为当前工具）
    fireEvent.click(screen.getByTestId('home-tools-btn'))
    fireEvent.click(await screen.findByTestId('home-tool-schedule'))
    expect(sessionStorage.getItem(OPEN_TOOL_FLAG)).toBe('schedule')
    expect(onNavigate).toHaveBeenLastCalledWith('gaea')
    sessionStorage.clear()
  })

  it('继续段：最近文档行（点行真打开→办公）+ 最近会话行（点开进聊天）', async () => {
    ;['第五份.docx', '第四份.docx', '第三份.docx', '预算.xlsx', '合同审查.docx'].forEach((n) => {
      recordRecentFile(`C:/docs/${n}`, n)
    })
    bridgeMocks.app.ListSessions.mockResolvedValue([
      { title: '造价对标', turns: 5, modTime: Date.now() - 3600_000, current: false },
      { title: '周报整理', turns: 2, modTime: Date.now() - 7200_000, current: false },
    ] as never)
    const onNavigate = vi.fn()
    render(wrap(<ModuleLauncher onNavigate={onNavigate} />))
    expect(screen.getByTestId('home-recent')).toBeTruthy()
    const docs = screen.getByTestId('desk-recent-docs')
    // 最近行收 4 条（方向一：首页只答「现在开始做什么」，行数从 6 收 4）
    await vi.waitFor(() => expect(docs.querySelectorAll('.w-docs-row').length).toBe(4))
    // 点行=真打开该文档 + 导航办公（v4.439 语义不变）
    fireEvent.click(docs.querySelector('.w-docs-btn') as HTMLElement)
    expect(onNavigate).toHaveBeenCalledWith('gaea')
    // 最近会话行出现（有会话才渲染）
    await vi.waitFor(() => expect(document.querySelector('.w-recent-sess .ml-sess-item')).toBeTruthy())
    expect(screen.getByText('造价对标')).toBeTruthy()
  })

  it('最近文档空态：横排空态 + 「进入办公」CTA 导航', () => {
    const onNavigate = vi.fn()
    render(wrap(<ModuleLauncher onNavigate={onNavigate} />))
    const docs = screen.getByTestId('desk-recent-docs')
    expect(docs.querySelector('.w-empty')).toBeTruthy()
    fireEvent.click(screen.getByText('进入办公'))
    expect(onNavigate).toHaveBeenCalledWith('gaea')
  })

  it('待办段：收件箱挂点收为单行（aria-label=任务），不再渲染四节状态带', async () => {
    bridgeMocks.app.GaeaTaskInboxList.mockResolvedValue([
      { id: 'ti-a', title: 'a', space: 'work', status: 'pending', source: 'ctrlk', createdAt: 1, updatedAt: 1 },
    ])
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    const todo = screen.getByTestId('desk-task-inbox')
    expect(todo.getAttribute('aria-label')).toBe('任务')
    expect(todo.querySelector('.ml-krow-strong')).toBeTruthy()
    expect(todo.querySelector('[data-testid="task-inbox-open-btn"]')).toBeTruthy()
    // 状态卡排退役：容器内不再有内核/会话/记忆节
    expect(document.querySelectorAll('.w-stat').length).toBe(0)
    await vi.waitFor(() => expect(todo.textContent).toContain('待处理 1 项'))
  })
})

// 7.3-1 任务收件箱挂点与面板：行为零变化（方向一只改呈现行）。
// 挂点计数跨空间全量（GaeaTaskInboxList('')）；面板单例同口径，新建任务仍按
// 当前壳层空间落标签（Go ValidSpace 仅收 work|play）。
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

// ── 7.3-2 板块降级为任务视图：homeLayout 分支（tasks 形态结构不动）──
describe('ModuleLauncher 首页形态分支（7.3-2 层跃升）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.removeItem('gaea.home.layout')
  })

  it('缺省 classic：渲染输入即首页，快捷钮为「任务优先」入口', async () => {
    const { useAppStore } = await import('../stores/appStore')
    useAppStore.setState({ homeLayout: 'classic' })
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    expect(screen.getByTestId('desk-task-inbox')).toBeTruthy()
    expect(screen.getByTestId('home-recent')).toBeTruthy()
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
