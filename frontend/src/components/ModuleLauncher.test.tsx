import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import ModuleLauncher from './ModuleLauncher'
import { LocaleProvider } from '../gaea/lib/i18n'
import { FRONTEND_EVENTS, OPEN_TOOL_FLAG } from '../events'
import { clearRecentObjectsForTest, recordRecentObject } from '../gaea/lib/recentObjects'

// ModuleLauncher 首页（§8.3 DeepSeek 化，2026-10-27 用户拍板；调研
// docs/gaea-home-redesign-survey-2026-10.md §8.3）：中轴只留「开始 + 命令条下
// 板块胶囊行」——问候/印章/大命令条垂直居中唯一主角，命令条正下方一排功能
// 板块胶囊（deriveLauncherModules 全量 12 模块=方向一退役的 12 板块卡回归
// DeepSeek 位）。退役面：最近文档行、待办一行、建议词 chips、顶栏仪表条、
// 任务优先形态（TasksFirstHome+homeLayout 分发全撤）。「继续」段保留工作
// 对象组（书/工程）+最近会话行，皆空时整段不渲染（冷启=纯落地页）。
// 收件箱面板入口归位：命令面板空态「打开任务收件箱」→ OPEN_TASK_INBOX 事件
// （本组件监听开单例面板）。
// 数据 hooks（会话）后端未就绪时静默兜底，无需 mock；语音 hook 整体 mock
// （真实实现依赖 Wails EventsOn）；打开链路桩三件（对象组恢复链路）。
// 断言焦点：两段结构、板块胶囊行全量+直达导航、退役面零残留、对象组双恢复
// 链路、收件箱事件开面板、语音降级。

const bridgeMocks = vi.hoisted(() => ({
  app: {
    ListSessions: vi.fn(async () => []),
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
// 「继续」对象组（二期）三桩：完整打开链路/切书闸门/进度 store（动态 import 同走 mock）
const novelMocks = vi.hoisted(() => ({ openNovelProject: vi.fn(async () => {}) }))
vi.mock('./novel/openNovelProject', () => novelMocks)
const guardMocks = vi.hoisted(() => ({ guardNovelSwitch: vi.fn(() => false) }))
vi.mock('./novel/novelSwitchGuard', () => guardMocks)
const scheduleStoreMocks = vi.hoisted(() => ({ openProject: vi.fn(async () => {}) }))
vi.mock('../schedule/store', () => ({ useScheduleStore: { getState: () => scheduleStoreMocks } }))
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

// 最近对象种子跨用例隔离（localStorage 单源 + 解析缓存都是模块级状态，
// 直写 localStorage 不失效缓存——走模块自身的 record/clear 辅助）
beforeEach(() => {
  clearRecentObjectsForTest()
})

describe('ModuleLauncher DeepSeek 式首页（§8.3）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('开始段：gaea 字标大标（静态品牌位）+ 题辞 + 大命令条；印章/时辰问候/顶栏仪表条退役', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    expect(document.querySelector('.w-greet')).toBeTruthy()
    // §8.4：印章与时辰问候退役——软件名字标 gaea 承接身份位（静态，无时间态）
    const h1 = document.querySelector('.w-greet h1.w-hero-title')
    expect(h1?.textContent).toBe('gaea')
    expect(document.querySelector('.w-seal')).toBeNull()
    for (const greet of ['早上好', '下午好', '晚上好', '夜深了']) {
      expect(screen.queryByText(greet)).toBeNull()
    }
    // 题辞：home.sub 长文案（锁存在 + 非空）
    const lede = document.querySelector('.w-greet .w-hero-lede')
    expect(lede).toBeTruthy()
    expect((lede?.textContent ?? '').length).toBeGreaterThan(0)
    // 命令条本体保留
    expect(document.querySelector('.w-cmd')).toBeTruthy()
    // 顶栏仪表条（形态切换+日期）随任务优先形态退役：整条不渲染
    expect(document.querySelector('.ml-strip')).toBeNull()
    expect(document.querySelector('.ml-space-btn')).toBeNull()
  })

  it('命令条多行（§8.5）：textarea 主体 + 框内动作行；Enter=发送（语音文本通道），Shift+Enter=换行不发送', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    const ta = document.querySelector('textarea.w-cmd-textarea') as HTMLTextAreaElement
    expect(ta).toBeTruthy()
    // 框内底部动作行：工具/语音/发送都在命令条内
    expect(document.querySelector('.w-cmd .w-cmd-row .w-cmd-btn.is-send')).toBeTruthy()
    expect(document.querySelector('.w-cmd .w-cmd-row [data-testid="home-tools-btn"]')).toBeTruthy()
    fireEvent.change(ta, { target: { value: '整理这份纪要' } })
    // Shift+Enter=换行：不发送（交给 textarea 默认行为）
    fireEvent.keyDown(ta, { key: 'Enter', shiftKey: true })
    expect(bridgeMocks.app.VoiceChatText).not.toHaveBeenCalled()
    expect(ta.value).toBe('整理这份纪要')
    // Enter=发送：走语音文本通道，发送后清空
    fireEvent.keyDown(ta, { key: 'Enter' })
    expect(bridgeMocks.app.VoiceChatText).toHaveBeenCalledWith('整理这份纪要')
    expect(ta.value).toBe('')
  })

  it('板块胶囊行（DeepSeek 位）：命令条正下方全量 12 模块（含设置），点击直达导航', () => {
    const onNavigate = vi.fn()
    render(wrap(<ModuleLauncher onNavigate={onNavigate} />))
    const row = screen.getByTestId('home-boards-row')
    expect(row.getAttribute('aria-label')).toBe('功能板块')
    // 全量 12 模块 = deriveLauncherModules（11 业务板块 + settings 隐式入口；
    // schedule 板 v4.168 起 inMenu:false 由办公「进度计划」承接，不在列——与
    // 方向一退役的 12 板块卡同一份清单）
    const pills = row.querySelectorAll('.w-board-pill')
    expect(pills.length).toBe(12)
    for (const key of ['chat', 'novel', 'imagegen', 'gaea', 'cost', 'memoryhub', 'modelcenter', 'characterlib', 'weixin', 'sin', 'runtime', 'settings']) {
      expect(row.querySelector(`[data-testid="home-board-${key}"]`)).toBeTruthy()
    }
    expect(screen.getByTestId('home-board-novel').textContent).toContain('小说')
    expect(screen.getByTestId('home-board-cost').textContent).toContain('造价数据库')
    // 点击胶囊 = 直达板块
    fireEvent.click(screen.getByTestId('home-board-novel'))
    expect(onNavigate).toHaveBeenCalledWith('novel')
    fireEvent.click(screen.getByTestId('home-board-runtime'))
    expect(onNavigate).toHaveBeenCalledWith('runtime')
  })

  it('§8.3 退役面零残留：最近文档行/待办行/建议词/任务优先形态/旧六区全 null', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    // 最近文档行（用户拍板删除）
    expect(screen.queryByTestId('desk-recent-docs')).toBeNull()
    // 待办行（用户拍板删除）
    expect(screen.queryByTestId('desk-task-inbox')).toBeNull()
    expect(screen.queryByTestId('task-inbox-open-btn')).toBeNull()
    // 建议词 chips（DeepSeek 形态输入框下只有功能胶囊）
    expect(document.querySelector('.w-chips')).toBeNull()
    expect(document.querySelector('.w-chip')).toBeNull()
    expect(screen.queryByTestId('home-sug-1')).toBeNull()
    // 任务优先形态与形态切换（整形态删除）
    expect(screen.queryByTestId('tasks-first-home')).toBeNull()
    expect(screen.queryByTestId('home-layout-tasks-entry')).toBeNull()
    expect(screen.queryByTestId('home-layout-back-classic')).toBeNull()
    // 方向一旧退役面维持零残留
    expect(document.querySelector('.w-modules')).toBeNull()
    expect(document.querySelector('.w-mod')).toBeNull()
    expect(document.querySelector('.w-stat-row')).toBeNull()
    expect(document.querySelector('.w-aside')).toBeNull()
    expect(document.querySelector('.w-progress')).toBeNull()
    expect(document.querySelector('.w-board')).toBeNull()
    expect(document.querySelector('.w-todo')).toBeNull()
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
    // §8.6 回归锁：弹层挂 .w-cmd-row 容器内（getPopupContainer）——body 直挂在本页
    // 布局下 rc-trigger 对齐损坏（菜单落 −1000vw 哨兵镜像位，用户侧「没反应」）；
    // jsdom 无布局量不了在屏内，锁容器归属即锁住修复口径
    expect(document.querySelector('.w-cmd-row .ant-dropdown')).toBeTruthy()
    expect(document.body.querySelector(':scope > .ant-dropdown')).toBeNull()
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

  it('继续段：对象组（书/工程）置顶续做行，恢复链路各走各的正确通道', async () => {
    recordRecentObject('novel', 'C:/Novels/长夜', '长夜')
    recordRecentObject('schedule', 'plans/百锦路.gsp', '百锦路')
    const onNavigate = vi.fn()
    render(wrap(<ModuleLauncher onNavigate={onNavigate} />))
    const group = screen.getByTestId('home-recent-objects')
    expect(group.querySelectorAll('.w-docs-row').length).toBe(2)
    // 书行：小说标签 + 书名
    const novelRow = screen.getByTestId('home-obj-novel')
    expect(novelRow.textContent).toContain('长夜')
    expect(novelRow.textContent).toContain('小说')
    // 工程行：进度标签 + 工程名
    const schedRow = screen.getByTestId('home-obj-schedule')
    expect(schedRow.textContent).toContain('百锦路')
    expect(schedRow.textContent).toContain('进度')
    // 点书行：完整打开链路（闸门默认放行）+ 导航小说板块
    fireEvent.click(novelRow)
    await vi.waitFor(() => expect(novelMocks.openNovelProject).toHaveBeenCalledWith('C:/Novels/长夜', '长夜'))
    expect(guardMocks.guardNovelSwitch).toHaveBeenCalled()
    expect(onNavigate).toHaveBeenCalledWith('novel')
    // 点工程行：先切指针（store 动作）再 OPEN_TOOL 双通道开进度工具面板
    const seen: Array<{ name: string; detail: unknown }> = []
    const rec = (ev: Event) => seen.push({ name: ev.type, detail: (ev as CustomEvent).detail })
    window.addEventListener(FRONTEND_EVENTS.OPEN_TOOL, rec)
    fireEvent.click(schedRow)
    await vi.waitFor(() => expect(scheduleStoreMocks.openProject).toHaveBeenCalledWith('plans/百锦路.gsp'))
    await vi.waitFor(() => {
      expect(seen).toEqual([{ name: FRONTEND_EVENTS.OPEN_TOOL, detail: { tool: 'schedule' } }])
      expect(sessionStorage.getItem(OPEN_TOOL_FLAG)).toBe('schedule')
      expect(onNavigate).toHaveBeenLastCalledWith('gaea')
    })
    window.removeEventListener(FRONTEND_EVENTS.OPEN_TOOL, rec)
    sessionStorage.clear()
  })

  it('继续段空态：无对象且无会话时整段不渲染——冷启首页=纯 DeepSeek 落地页', () => {
    bridgeMocks.app.ListSessions.mockResolvedValue([])
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    expect(screen.queryByTestId('home-recent')).toBeNull()
    expect(screen.queryByTestId('home-recent-objects')).toBeNull()
    expect(screen.queryByTestId('home-obj-novel')).toBeNull()
  })

  it('继续段：最近会话行渲染（有会话才出），点开进聊天板块', async () => {
    bridgeMocks.app.ListSessions.mockResolvedValue([
      { title: '造价对标', turns: 5, modTime: Date.now() - 3600_000, current: false },
      { title: '周报整理', turns: 2, modTime: Date.now() - 7200_000, current: false },
    ] as never)
    const onNavigate = vi.fn()
    render(wrap(<ModuleLauncher onNavigate={onNavigate} />))
    await vi.waitFor(() => expect(document.querySelector('.w-recent-sess .ml-sess-item')).toBeTruthy())
    expect(screen.getByText('造价对标')).toBeTruthy()
    fireEvent.click(document.querySelector('.w-recent-sess .ml-sess-item') as HTMLElement)
    expect(onNavigate).toHaveBeenCalledWith('chat')
  })
})

// ── 收件箱入口归位（§8.3）：首页待办行退役后，面板唯一入口=命令面板空态
// 「打开任务收件箱」→ OPEN_TASK_INBOX 事件 → 本组件单例面板。面板行为零变化
// （列表口径跨空间全量 scope=''，新建任务按当前壳层空间落标签）。──
describe('ModuleLauncher 任务收件箱面板（§8.3 入口归位）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('OPEN_TASK_INBOX 事件开面板单例：全量再拉，跨空间行带归属空间标', async () => {
    bridgeMocks.app.GaeaTaskInboxList.mockResolvedValue([
      { id: 'ti-a', title: '书斋侧任务', space: 'work', status: 'pending', source: 'ctrlk', createdAt: 1, updatedAt: 1 },
      { id: 'ti-b', title: '闲庭侧任务', space: 'play', status: 'pending', source: 'voice', createdAt: 2, updatedAt: 2 },
    ])
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} />))
    // 待办行退役：挂载时不拉清单（面板 open 才拉）
    expect(bridgeMocks.app.GaeaTaskInboxList).not.toHaveBeenCalled()
    window.dispatchEvent(new CustomEvent(FRONTEND_EVENTS.OPEN_TASK_INBOX))
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
    window.dispatchEvent(new CustomEvent(FRONTEND_EVENTS.OPEN_TASK_INBOX))
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
