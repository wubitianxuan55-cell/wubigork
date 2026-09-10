import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { wailsApp } from '../lib/wailsApp'
import type { ProjectCard } from '../stores/appStore'

vi.mock('../lib/wailsApp', () => ({
  wailsApp: vi.fn(),
}))

vi.mock('../components/WelcomePage', () => ({
  default: () => <div>welcome-stub</div>,
}))

vi.mock('../components/novel/CreateNovelModal', () => ({
  default: () => null,
}))

vi.mock('../components/novel/ImportNovelModal', () => ({
  default: () => null,
}))

import HomePage from './HomePage'
import { useAppStore } from '../stores/appStore'
import { writeReadingProgress } from '../utils/readingProgress'

const mockedWailsApp = vi.mocked(wailsApp)

const sample: ProjectCard = {
  title: '风雪夜归',
  genre: '玄幻',
  style: '史诗',
  path: 'C:/novels/fengxue',
  word_count: 12000,
  chapter_count: 8,
  created_at: '2026-09-01T00:00:00Z',
  last_opened_at: '2026-09-10T00:00:00Z',
}

describe('HomePage 书房书架', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useAppStore.setState({
      loggedIn: true,
      projectOpen: false,
      projectPath: '',
      projectTitle: '',
      projects: [],
    })
    mockedWailsApp.mockReturnValue({
      ListProjects: vi.fn(async () => [sample]),
      GetNovelsDir: vi.fn(async () => 'C:/novels'),
      OpenProject: vi.fn(async () => {}),
      DeleteProject: vi.fn(async () => {}),
      GaeaPickFiles: vi.fn(async () => []),
    } as unknown as ReturnType<typeof wailsApp>)
  })

  it('画廊头显示部数，书封入栅格', async () => {
    render(<HomePage />)
    expect(screen.getByRole('heading', { name: '书架' })).toBeTruthy()
    await waitFor(() => {
      expect(screen.getByText('1 部')).toBeTruthy()
      expect(screen.getByRole('button', { name: '打开小说「风雪夜归」' })).toBeTruthy()
    })
  })

  it('已打开小说时出现正在编辑横条，继续阅读发 goto-tab', async () => {
    writeReadingProgress(sample.path, { nodeId: 'ch-3', chapterNum: 3, title: '夜色' })
    useAppStore.setState({
      loggedIn: true,
      projectOpen: true,
      projectPath: sample.path,
      projectTitle: sample.title,
    })
    const goto = vi.fn()
    window.addEventListener('novel:goto-tab', goto as EventListener)
    render(<HomePage />)
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: '风雪夜归' })).toBeTruthy()
    })
    expect(screen.getByText(/读到第3章/)).toBeTruthy()
    fireEvent.click(within(screen.getByRole('region', { name: '正在编辑' })).getByRole('button', { name: /继续阅读/ }))
    expect(goto).toHaveBeenCalled()
    const ev = goto.mock.calls[0][0] as CustomEvent<{ tab?: string }>
    expect(ev.detail?.tab).toBe('chapter')
    window.removeEventListener('novel:goto-tab', goto as EventListener)
  })

  it('空书架给新建/导入动作', async () => {
    mockedWailsApp.mockReturnValue({
      ListProjects: vi.fn(async () => []),
      GetNovelsDir: vi.fn(async () => 'C:/novels'),
    } as unknown as ReturnType<typeof wailsApp>)
    render(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('书架空空如也')).toBeTruthy()
    })
    expect(screen.getAllByRole('button', { name: /新建小说/ }).length).toBeGreaterThan(0)
    expect(screen.getByRole('button', { name: /导入成品小说/ })).toBeTruthy()
  })
})
