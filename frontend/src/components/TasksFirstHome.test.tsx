// 任务优先首页（7.3-2 板块降级为任务视图·层跃升刀）：收件箱大区内联渲染/
// 能力 chips 全量可达/切回经典回调/最近会话区。v4.475 首页合一：不再按壳层
// 空间过滤——chips 全板块，收件箱列表跨空间全量（scope=''）。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  list: vi.fn().mockResolvedValue([]),
}))

vi.mock('../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      GaeaTaskInboxList: mocks.list,
    },
  }
})

import TasksFirstHome from './TasksFirstHome'
import { LocaleProvider } from '../gaea/lib/i18n'
import { useAppStore } from '../stores/appStore'

const DATA = {
  stats: null,
  projectOpen: false,
  monitor: null,
  recentFiles: [],
  sessions: [
    { title: '昨天关于报价的会话', preview: '', turns: 12, modTime: Date.now() - 3600_000 },
  ],
  memoryHub: null,
}

const open = () => render(<LocaleProvider>
  <TasksFirstHome data={DATA} onNavigate={vi.fn()} />
</LocaleProvider>)

describe('TasksFirstHome 任务优先首页（7.3-2 层跃升 + v4.475 合一口径）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.list.mockResolvedValue([])
    useAppStore.setState({ homeLayout: 'tasks' })
  })

  it('首屏=收件箱内联板（TaskInboxBoard 直嵌，列表跨空间全量 scope=""）', async () => {
    mocks.list.mockResolvedValue([
      { id: 'ti-1', title: '核对第三章伏笔', space: 'play', status: 'pending', source: 'ctrlk', createdAt: 1, updatedAt: 2 },
    ])
    open()
    // 内联板数据面（TaskInboxBoard 的 testid）
    await waitFor(() => expect(screen.getByTestId('task-inbox-panel')).toBeTruthy())
    expect(mocks.list).toHaveBeenCalledWith('')
    await waitFor(() => expect(screen.getByText('核对第三章伏笔')).toBeTruthy())
    // 跨空间混合清单：行内带归属空间标
    await waitFor(() => expect(screen.getByText('闲庭')).toBeTruthy())
    // 首页容器与收件箱大区
    expect(screen.getByTestId('tasks-first-home')).toBeTruthy()
    expect(screen.getByTestId('tasks-first-inbox')).toBeTruthy()
  })

  it('能力视图：全板块 chips 渲染且点击直达（书斋+闲庭同屏，藏≠删）', async () => {
    const onNavigate = vi.fn()
    render(
      <LocaleProvider>
        <TasksFirstHome data={DATA} onNavigate={onNavigate} />,
      </LocaleProvider>,
    )
    await waitFor(() => expect(screen.getByTestId('tasks-first-caps')).toBeTruthy())
    const chips = screen.getAllByTestId(/^tasks-first-cap-/)
    expect(chips.length).toBeGreaterThanOrEqual(10) // 全部业务板块 + settings
    // 闲庭板块 chips 可见（合一判据）
    expect(screen.getByTestId('tasks-first-cap-novel')).toBeTruthy()
    expect(screen.getByTestId('tasks-first-cap-sin')).toBeTruthy()
    fireEvent.click(screen.getByTestId('tasks-first-cap-settings'))
    expect(onNavigate).toHaveBeenCalledWith('settings')
  })

  it('顶栏无空间切换钮（合一后切空间由板块导航隐式驱动）', () => {
    open()
    expect(screen.queryByTestId('ml-space-switch')).toBeNull()
    expect(screen.queryByTestId('ml-space-work')).toBeNull()
    expect(screen.queryByTestId('ml-space-play')).toBeNull()
    // beforeEach 已把 homeLayout 设为 tasks → 快捷钮呈「切回经典」形态
    expect(screen.getByTestId('home-layout-back-classic')).toBeTruthy()
  })

  it('切回经典：按钮写回 store（回退开关快捷位）', async () => {
    open()
    await waitFor(() => expect(screen.getByTestId('tasks-first-back')).toBeTruthy())
    fireEvent.click(screen.getByTestId('tasks-first-back'))
    expect(useAppStore.getState().homeLayout).toBe('classic')
    useAppStore.setState({ homeLayout: 'tasks' })
  })

  it('最近会话区渲染会话标题', async () => {
    open()
    await waitFor(() => expect(screen.getByText('昨天关于报价的会话')).toBeTruthy())
  })
})
