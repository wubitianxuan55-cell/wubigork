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

vi.mock('../components/novel/BookSearchModal', () => ({
  default: ({ open }: { open: boolean }) => (open ? <div>booksearch-stub</div> : null),
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

  it('工具条「在线搜书」开关搜索 Modal（t2）', async () => {
    render(<HomePage />)
    await waitFor(() => expect(screen.getByText('风雪夜归')).toBeTruthy())
    expect(screen.queryByText('booksearch-stub')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: /在线搜书/ }))
    expect(screen.getByText('booksearch-stub')).toBeTruthy()
  })
})

// tail×反推串联（v4.292）：tail 导入完成后弹「立即反推」确认，
// 确认后派发 goto-tab + auto-reconstruct 两个事件。
describe('HomePage tail 导入串联反推', () => {
  beforeEach(() => {
    useAppStore.setState({
      loggedIn: true, projectOpen: false, projectPath: '', projectTitle: '', projects: [],
    })
  })

  it('tail 模式导入完成弹反推确认，确认后派发事件', async () => {
    mockedWailsApp.mockReturnValue({
      ListProjects: vi.fn(async () => []),
      GetNovelsDir: vi.fn(async () => 'C:/novels'),
      GaeaPickFiles: vi.fn(async () => [{ path: 'C:/b/长书.txt', name: '长书.txt', size: 10 }]),
      ImportNovelBookEx: vi.fn(async () => ({
        path: 'C:/novels/长书', title: '长书', chapter_count: 10, total_words: 100, split_strategy: 'booksource',
      })),
    } as unknown as ReturnType<typeof wailsApp>)

    const events: string[] = []
    const spy = vi.fn((e: Event) => events.push(e.type))
    window.addEventListener('novel:goto-tab', spy)
    window.addEventListener('novel:auto-reconstruct', spy)

    render(<HomePage />)
    await waitFor(() => expect(screen.getByRole('button', { name: /导入小说/ })).toBeTruthy())
    fireEvent.click(screen.getByRole('button', { name: /导入小说/ }))

    // 提取范围选末 10 章（antd 下拉挂在 body portal，需 mouseDown selector 展开）
    await screen.findByPlaceholderText('小说标题（必填）') // 等 modal 渲染
    const selector = document.querySelector('.ant-modal .ant-select-selector')
    expect(selector).toBeTruthy()
    fireEvent.mouseDown(selector as Element)
    const opt = await screen.findByText('末 10 章', {}, { timeout: 5000 })
    fireEvent.click(opt.closest('.ant-select-item-option') ?? opt)

    fireEvent.change(screen.getByPlaceholderText('小说标题（必填）'), { target: { value: '长书' } })
    const modalEl = document.querySelector('.ant-modal')
    expect(modalEl).toBeTruthy()
    const importBtn = Array.from(modalEl!.querySelectorAll('button')).find(
      (b) => b.textContent !== null && /导\s*入/.test(b.textContent),
    )
    expect(importBtn).toBeTruthy()
    fireEvent.click(importBtn as Element)

    // 导入完成 → tail 确认弹窗
    await waitFor(() => expect(document.body.textContent).toContain('立即反推这部分大纲？'))
    const confirmBtn = document.querySelector('.ant-modal-confirm .ant-modal-confirm-btns button:last-child')
    expect(confirmBtn).toBeTruthy() // 确认弹窗存在
    fireEvent.click(confirmBtn as Element)

    await waitFor(() => expect(events).toContain('novel:auto-reconstruct'))
    expect(events).toContain('novel:goto-tab')
    window.removeEventListener('novel:goto-tab', spy)
    window.removeEventListener('novel:auto-reconstruct', spy)
  })
})
