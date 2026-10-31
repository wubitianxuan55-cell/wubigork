import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import { LocaleProvider } from '../gaea/lib/i18n'
import { FRONTEND_EVENTS } from '../events'
import { TaskInboxHost } from './TaskInboxHost'

const bridgeMocks = vi.hoisted(() => ({
  app: {
    GaeaTaskInboxList: vi.fn(async (): Promise<import('../gaea/lib/types').TaskInboxView[]> => []),
    GaeaTaskInboxSave: vi.fn(async (): Promise<import('../gaea/lib/types').TaskInboxView> =>
      ({ id: 'ti-mock', title: 't', space: 'work', status: 'pending', source: 'inbox', createdAt: 0, updatedAt: 0 })),
    GaeaTaskInboxSetStatus: vi.fn(async (): Promise<import('../gaea/lib/types').TaskInboxView> =>
      ({ id: 'ti-mock', title: 't', space: 'work', status: 'pending', source: 'inbox', createdAt: 0, updatedAt: 0 })),
    GaeaTaskInboxDelete: vi.fn(async (): Promise<void> => {}),
  },
}))

vi.mock('../gaea/lib/bridge', () => ({ app: bridgeMocks.app }))

const pendingSessionMocks = vi.hoisted(() => ({ requestSessionResume: vi.fn() }))
vi.mock('../gaea/lib/pendingSessionResume', () => pendingSessionMocks)

const wrap = (ui: ReactElement) => {
  localStorage.setItem('gaea-lang', 'zh')
  return <LocaleProvider>{ui}</LocaleProvider>
}

describe('TaskInboxHost 壳层级收件箱', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('OPEN_TASK_INBOX 事件打开单一收件箱面板，并按跨空间口径拉取', async () => {
    bridgeMocks.app.GaeaTaskInboxList.mockResolvedValue([
      { id: 'ti-a', title: '书斋侧任务', space: 'work', status: 'pending', source: 'ctrlk', createdAt: 1, updatedAt: 1 },
      { id: 'ti-b', title: '闲庭侧任务', space: 'play', status: 'pending', source: 'voice', createdAt: 2, updatedAt: 2 },
    ])
    render(wrap(<TaskInboxHost onNavigate={vi.fn()} />))

    expect(bridgeMocks.app.GaeaTaskInboxList).not.toHaveBeenCalled()
    window.dispatchEvent(new CustomEvent(FRONTEND_EVENTS.OPEN_TASK_INBOX))

    await vi.waitFor(() => expect(document.querySelector('[data-testid="task-inbox-panel"]')).toBeTruthy())
    await vi.waitFor(() => expect(bridgeMocks.app.GaeaTaskInboxList).toHaveBeenCalledWith(''))
    await vi.waitFor(() => expect(screen.getByText('乐园')).toBeTruthy())
    expect(screen.getByText('工位')).toBeTruthy()
  })

  it('新建任务仍按当前壳层空间落标签', async () => {
    render(wrap(<TaskInboxHost onNavigate={vi.fn()} />))
    window.dispatchEvent(new CustomEvent(FRONTEND_EVENTS.OPEN_TASK_INBOX))
    await vi.waitFor(() => expect(document.querySelector('[data-testid="task-inbox-panel"]')).toBeTruthy())

    fireEvent.change(screen.getByTestId('task-inbox-input'), { target: { value: '新任务' } })
    fireEvent.click(screen.getByTestId('task-inbox-add'))

    await vi.waitFor(() => expect(bridgeMocks.app.GaeaTaskInboxSave).toHaveBeenCalled())
    const saveCalls = bridgeMocks.app.GaeaTaskInboxSave.mock.calls as unknown as string[][]
    const payload = JSON.parse(saveCalls[0][0])
    expect(payload).toMatchObject({ title: '新任务', space: 'work', source: 'inbox' })
  })
})
