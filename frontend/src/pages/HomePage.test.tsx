import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach } from 'vitest'
import { Modal } from 'antd'
import { wailsApp } from '../lib/wailsApp'
import type { ProjectCard } from '../stores/appStore'

vi.mock('../lib/wailsApp', () => ({
  wailsApp: vi.fn(),
}))

vi.mock('../components/WelcomePage', () => ({
  default: () => <div>welcome-stub</div>,
}))

vi.mock('../components/novel/CreateNovelModal', () => ({
  default: ({ open }: { open: boolean }) => (open ? <div>createnovel-stub</div> : null),
}))

vi.mock('../components/novel/BookSearchModal', () => ({
  default: ({ open }: { open: boolean }) => (open ? <div>booksearch-stub</div> : null),
}))

import HomePage from './HomePage'
import { useAppStore } from '../stores/appStore'
import { writeReadingProgress } from '../utils/readingProgress'
import { registerNovelDirtyProvider, resetNovelDirtyProviders } from '../components/novel/novelSwitchGuard'

const mockedWailsApp = vi.mocked(wailsApp)

// imperative Modal 的 DOM 不随 destroy()/cleanup 卸载（仓内实测坑）→ 断言一律
// 「取最新 .ant-modal-confirm + within」，afterEach 只 destroyAll + 清脏状态登记。
const confirms = () => Array.from(document.querySelectorAll<HTMLElement>('.ant-modal-confirm'))
afterEach(async () => {
  Modal.destroyAll()
  resetNovelDirtyProviders()
  await new Promise((r) => setTimeout(r, 0))
})

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
      projectsError: null,
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

  // ── 线2 防复发守卫：书架页常驻隐藏时不得抢窗口级快捷键（v4.421.0）──
  // 书架在 NovelPage 里常驻挂载（CSS display:none 隐藏，隐藏 ≠ 卸载），若不按
  // active 门控，用户在设定/阅读页按 Ctrl+N 会莫名弹出「新建小说」。
  it('Ctrl+N：不传 active（默认 true）打开新建小说弹窗并 preventDefault', async () => {
    render(<HomePage />)
    await waitFor(() => expect(screen.getByText('风雪夜归')).toBeTruthy())
    expect(screen.queryByText('createnovel-stub')).toBeNull()

    const ev = new KeyboardEvent('keydown', { key: 'n', ctrlKey: true, cancelable: true })
    await act(async () => { window.dispatchEvent(ev) })
    expect(ev.defaultPrevented).toBe(true)
    expect(screen.getByText('createnovel-stub')).toBeTruthy()
  })

  it('Ctrl+N：active=false 时不开弹窗、也不 preventDefault', async () => {
    render(<HomePage active={false} />)
    await waitFor(() => expect(screen.getByText('风雪夜归')).toBeTruthy())

    const ev = new KeyboardEvent('keydown', { key: 'n', ctrlKey: true, cancelable: true })
    await act(async () => { window.dispatchEvent(ev) })
    expect(ev.defaultPrevented).toBe(false)
    expect(screen.queryByText('createnovel-stub')).toBeNull()
  })

  // B13（v4.425 审计）：删除「正在编辑」的那本书后，后端已 closePM，但 store 的
  // projectOpen/projectPath 原样留着 → 顶栏继续挂着已删的书，其余子页随后的保存
  // 全报「请先打开项目」。守卫：删到当前项目即关闭项目上下文。
  it('删除正在编辑的书：一并关闭项目上下文（不留已删除的书在「正在编辑」）', async () => {
    useAppStore.setState({
      projectOpen: true, projectPath: sample.path, projectTitle: sample.title,
    })
    render(<HomePage />)
    await waitFor(() => expect(screen.getByText('风雪夜归')).toBeTruthy())

    fireEvent.click(screen.getByRole('button', { name: '删除「风雪夜归」' }))
    // Popconfirm 的确认按钮（antd 会给两字按钮插空格 → 用正则）
    const del = await screen.findByRole('button', { name: /^删\s*除$/ })
    fireEvent.click(del)

    await waitFor(() => expect(useAppStore.getState().projectOpen).toBe(false))
    expect(useAppStore.getState().projectPath).toBe('')
    expect(useAppStore.getState().projectTitle).toBe('')
  })

  it('删除非当前项目：不动项目上下文', async () => {
    useAppStore.setState({
      projectOpen: true, projectPath: 'C:/novels/other', projectTitle: '另一本',
    })
    render(<HomePage />)
    await waitFor(() => expect(screen.getByText('风雪夜归')).toBeTruthy())

    fireEvent.click(screen.getByRole('button', { name: '删除「风雪夜归」' }))
    const del = await screen.findByRole('button', { name: /^删\s*除$/ })
    fireEvent.click(del)

    await waitFor(() => expect(mockedWailsApp().DeleteProject).toHaveBeenCalled())
    expect(useAppStore.getState().projectOpen).toBe(true)
    expect(useAppStore.getState().projectPath).toBe('C:/novels/other')
  })

  // M2（v4.425）跨页切书闸门在**调用点**的接线守卫：单元测试已覆盖闸门逻辑，
  // 但「书架真的把 openProject 包进 guardNovelSwitch 了吗」会静默回归。
  it('有未保存内容时打开另一本书：先弹确认，不直接切（放弃修改后才切）', async () => {
    useAppStore.setState({
      projectOpen: true, projectPath: 'C:/novels/old', projectTitle: '旧书',
    })
    registerNovelDirtyProvider({
      id: 'create-body', label: () => '创作页正文',
      dirty: () => true, save: async () => true,
    })
    render(<HomePage />)
    await waitFor(() => expect(screen.getByText('风雪夜归')).toBeTruthy())
    const open = mockedWailsApp().OpenProject as ReturnType<typeof vi.fn>

    fireEvent.click(screen.getByRole('button', { name: '打开小说「风雪夜归」' }))
    await waitFor(() => expect(confirms().length).toBeGreaterThan(0))
    expect(open).not.toHaveBeenCalled()

    const scoped = within(confirms()[confirms().length - 1])
    fireEvent.click(scoped.getByRole('button', { name: /放\s*弃\s*修\s*改\s*并\s*切\s*换/ }))
    await waitFor(() => expect(open).toHaveBeenCalledWith(sample.path))
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
    // 必须排除 imperative confirm 弹窗：`Modal.confirm` 的 DOM **不随 destroy 卸载**
    // （仓内实测坑），本文件前面的跨页切书闸门用例会留下 `.ant-modal-confirm` 残影，
    // 裸 `document.querySelector('.ant-modal')` 会取到它而不是导入弹窗。
    const modalEl = document.querySelector('.ant-modal:not(.ant-modal-confirm)')
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

  // v4.349 稳健性刀：ListProjects 失败此前被 loadProjects 吞掉（只 console.error），
  // 首页把「读失败」渲染成「书架空空如也」——假空态比无提示更误导。
  it('ListProjects 失败：显示可重试的错误态，不冒充「书架空空如也」', async () => {
    mockedWailsApp.mockReturnValue({
      ListProjects: vi.fn(async () => { throw new Error('内核未就绪') }),
      GetNovelsDir: vi.fn(async () => 'C:/novels'),
      OpenProject: vi.fn(async () => {}),
      DeleteProject: vi.fn(async () => {}),
      GaeaPickFiles: vi.fn(async () => []),
    } as unknown as ReturnType<typeof wailsApp>)

    render(<HomePage />)
    await waitFor(() => expect(screen.getByTestId('shelf-load-error')).toBeTruthy())
    expect(screen.getByText('书架读取失败')).toBeTruthy()
    expect(screen.getByText(/内核未就绪/)).toBeTruthy()
    expect(screen.queryByText('书架空空如也')).toBeNull()

    // 重试成功 → 回到正常书架
    mockedWailsApp.mockReturnValue({
      ListProjects: vi.fn(async () => [sample]),
      GetNovelsDir: vi.fn(async () => 'C:/novels'),
      OpenProject: vi.fn(async () => {}),
      DeleteProject: vi.fn(async () => {}),
      GaeaPickFiles: vi.fn(async () => []),
    } as unknown as ReturnType<typeof wailsApp>)
    fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }))
    await waitFor(() => expect(screen.getByRole('button', { name: '打开小说「风雪夜归」' })).toBeTruthy())
    expect(screen.queryByTestId('shelf-load-error')).toBeNull()
  })
})
