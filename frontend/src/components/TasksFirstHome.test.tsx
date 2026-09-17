// 任务优先首页（7.3-2 板块降级为任务视图·层跃升刀）：收件箱大区内联渲染/
// 能力 chips 全量可达/切回经典回调/最近会话区。
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
  <TasksFirstHome data={DATA} onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} activeModel="grok" />
</LocaleProvider>)

describe('TasksFirstHome 任务优先首页（7.3-2 层跃升）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.list.mockResolvedValue([])
    useAppStore.setState({ homeLayout: 'tasks' })
  })

  it('首屏=收件箱内联板（TaskInboxBoard 直嵌，拉当前空间任务）', async () => {
    mocks.list.mockResolvedValue([
      { id: 'ti-1', title: '核对第三章伏笔', status: 'pending', source: 'ctrlk', createdAt: 1, updatedAt: 2 },
    ] as never)
    open()
    // 内联板数据面（TaskInboxBoard 的 testid）
    await waitFor(() => expect(screen.getByTestId('task-inbox-panel')).toBeTruthy())
    expect(mocks.list).toHaveBeenCalledWith('work')
    await waitFor(() => expect(screen.getByText('核对第三章伏笔')).toBeTruthy())
    // 首页容器与收件箱大区
    expect(screen.getByTestId('tasks-first-home')).toBeTruthy()
    expect(screen.getByTestId('tasks-first-inbox')).toBeTruthy()
  })

  it('能力视图：板块 chips 全量渲染且点击直达（藏≠删）', async () => {
    const onNavigate = vi.fn()
    render(
      <LocaleProvider>
        <TasksFirstHome data={DATA} onNavigate={onNavigate} space="work" onSwitchSpace={vi.fn()} />,
      </LocaleProvider>,
    )
    await waitFor(() => expect(screen.getByTestId('tasks-first-caps')).toBeTruthy())
    const chips = screen.getAllByTestId(/^tasks-first-cap-/)
    expect(chips.length).toBeGreaterThanOrEqual(5) // 当前空间板块全量
    fireEvent.click(screen.getByTestId('tasks-first-cap-settings'))
    expect(onNavigate).toHaveBeenCalledWith('settings')
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
