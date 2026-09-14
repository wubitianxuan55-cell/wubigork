// SinBookSourcePanel.test.tsx — 原罪右栏「书源」页签（sin 书源线 t3）。
// bridge app 代理 mock（BookSearchModal.test 同款）+ window.runtime 桩捕事件：
// 覆盖候选表 HasRule 打标与免规则禁用、目录预览与范围默认、下载起跑参数、
// progress/done/error 三路事件、取消、成书清单渲染与删除确认。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import React from 'react'

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      SinBookSourceSearch: vi.fn(),
      SinBookSourceToc: vi.fn(),
      SinBookSourceDownload: vi.fn(),
      SinBookSourceDownloadCancel: vi.fn(),
      SinBookSourceBooksList: vi.fn(),
      SinBookSourceBookDelete: vi.fn(),
      SinBookSourceBookExportEpub: vi.fn(),
      ImportNovelBookEx: vi.fn(),
    },
  }
})

import { SinBookSourcePanel } from './SinBookSourcePanel'
import { app } from '../../gaea/lib/bridge'
import type { NovelBookSourceCandidate, NovelBookSourceTocPreview } from '../../gaea/lib/bridge/novel'
import type { SinBookSourceBook } from '../../gaea/lib/bridge/sin'

const search = vi.mocked(app.SinBookSourceSearch)
const toc = vi.mocked(app.SinBookSourceToc)
const download = vi.mocked(app.SinBookSourceDownload)
const cancelJob = vi.mocked(app.SinBookSourceDownloadCancel)
const booksList = vi.mocked(app.SinBookSourceBooksList)
const bookDelete = vi.mocked(app.SinBookSourceBookDelete)
const exportEpub = vi.mocked(app.SinBookSourceBookExportEpub)
const importEx = vi.mocked(app.ImportNovelBookEx)

// window.runtime 桩：subscribe() 走 EventsOn，捕获 handler 手工投递事件。
let deliver: ((data: unknown) => void) | null = null
function stubRuntime() {
  deliver = null
  ;(window as unknown as { runtime: unknown }).runtime = {
    EventsOn: vi.fn((_ch: string, h: (data: unknown) => void) => { deliver = h }),
    EventsOff: vi.fn(),
  }
}

const candidates: NovelBookSourceCandidate[] = [
  { kind: 'rule', source: '测试源', title: '风雪夜归', author: '佚名', url: 'https://a.test/book/1', host: 'a.test', hasRule: true },
  { kind: 'web', source: 'bing', title: '风雪夜归 最新章节', url: 'https://b.test/fsyg/', host: 'b.test', hasRule: false },
]

const tocPreview: NovelBookSourceTocPreview = {
  total: 3,
  sample: [
    { title: '第1章 起风', url: 'u1', order: 1 },
    { title: '第2章 夜行', url: 'u2', order: 2 },
    { title: '第3章 归途', url: 'u3', order: 3 },
  ],
  truncated: false,
}

const books: SinBookSourceBook[] = [
  { title: '风雪夜归', path: '/books/风雪夜归.txt', sizeBytes: 2048, modifiedAt: '2026-09-14T08:00:00Z' },
]

async function renderPanel() {
  return render(<SinBookSourcePanel />)
}

async function searchAndPick() {
  search.mockResolvedValue({ candidates, warnings: [] })
  toc.mockResolvedValue(tocPreview)
  await renderPanel()
  const input = screen.getByLabelText('书源搜索关键字')
  fireEvent.change(input, { target: { value: '风雪夜归' } })
  fireEvent.click(screen.getByRole('button', { name: /搜\s*书/ }))
  await screen.findByRole('list', { name: '搜书候选' })
  // 免规则候选禁用（后端 fail-closed 的 UI 同款诚实）
  // 注：listitem 的可访问名来自 title 属性而非内容（dom-accessibility-api），
  // 定位用文本 + closest('button')。
  const refOnly = screen.getByText('仅参考').closest('button') as HTMLButtonElement
  expect(refOnly.disabled).toBe(true)
  fireEvent.click(screen.getByText('风雪夜归').closest('button') as HTMLButtonElement)
  await screen.findByLabelText('目录预览与下载')
}

beforeEach(() => {
  vi.clearAllMocks()
  stubRuntime()
  booksList.mockResolvedValue([])
})

describe('SinBookSourcePanel（原罪·书源页签）', () => {
  it('搜书渲染候选：可下载/仅参考打标，免规则候选禁用；告警直显', async () => {
    search.mockResolvedValue({ candidates, warnings: ['书源「X」搜索失败: HTTP 403'] })
    renderPanel()
    fireEvent.change(screen.getByLabelText('书源搜索关键字'), { target: { value: '风雪夜归' } })
    fireEvent.click(screen.getByRole('button', { name: /搜\s*书/ }))
    await screen.findByRole('list', { name: '搜书候选' })
    expect(screen.getByText('书源「X」搜索失败: HTTP 403')).toBeTruthy()
    expect(screen.getByText('可下载')).toBeTruthy()
    const refOnly = screen.getByText('仅参考').closest('button') as HTMLButtonElement
    expect(refOnly.disabled).toBe(true)
  })

  it('选中候选→目录预览，范围默认全本；下载起跑参数逐参对齐', async () => {
    await searchAndPick()
    expect(screen.getByText('共 3 章')).toBeTruthy()
    expect(screen.getByText('第1章 起风')).toBeTruthy()
    // antd InputNumber 把 aria-label 落在内层 input 上（role=spinbutton），直接取值
    const startInput = screen.getByLabelText('起始章') as HTMLInputElement
    const endInput = screen.getByLabelText('结束章') as HTMLInputElement
    expect(startInput.value).toBe('1')
    expect(endInput.value).toBe('3')
    download.mockResolvedValue({ jobId: 'bss_t1' })
    fireEvent.click(screen.getByRole('button', { name: '下载成书' }))
    await waitFor(() => expect(download).toHaveBeenCalledTimes(1))
    expect(download).toHaveBeenCalledWith('测试源', 'https://a.test/book/1', 1, 3, '风雪夜归')
  })

  it('progress 更新进度条文案；done 收尾刷成书清单并清选中', async () => {
    await searchAndPick()
    download.mockResolvedValue({ jobId: 'bss_t2' })
    fireEvent.click(screen.getByRole('button', { name: '下载成书' }))
    await screen.findByLabelText('下载进度')
    expect(booksList).toHaveBeenCalledTimes(1) // 挂载时
    await waitFor(() => expect(deliver).toBeTruthy())
    deliver!({ type: 'progress', done: 2, total: 3 })
    await screen.findByText('2/3 章（进度按每 20 章上报）')
    booksList.mockResolvedValue(books)
    deliver!({ type: 'done', result: { chapters: 3 } })
    await screen.findByRole('list', { name: '成书清单' })
    // 候选表仍在（仅清选中），限定成书条目的标题
    expect(screen.getByText('风雪夜归', { selector: '.sin-bs-book-title' })).toBeTruthy()
    expect(screen.queryByLabelText('下载进度')).toBeNull()
    expect(screen.queryByLabelText('目录预览与下载')).toBeNull() // 选中已清
    expect(booksList).toHaveBeenCalledTimes(2)
  })

  it('error 事件直显失败原因并提示可重试', async () => {
    await searchAndPick()
    download.mockResolvedValue({ jobId: 'bss_t3' })
    fireEvent.click(screen.getByRole('button', { name: '下载成书' }))
    await screen.findByLabelText('下载进度')
    await waitFor(() => expect(deliver).toBeTruthy())
    deliver!({ type: 'error', error: '整本下载失败：3 章全部未取到' })
    await screen.findByText(/整本下载失败：3 章全部未取到/)
    expect(screen.queryByLabelText('下载进度')).toBeNull()
  })

  it('取消调用 Cancel 并立即收尾下载态', async () => {
    await searchAndPick()
    download.mockResolvedValue({ jobId: 'bss_t4' })
    fireEvent.click(screen.getByRole('button', { name: '下载成书' }))
    await screen.findByLabelText('下载进度')
    cancelJob.mockResolvedValue(true)
    // antd 两字按钮自动插空格（「取 消」），name 用正则容错
    fireEvent.click(screen.getByRole('button', { name: /取\s*消/ }))
    await waitFor(() => expect(cancelJob).toHaveBeenCalledWith('bss_t4'))
    expect(screen.queryByLabelText('下载进度')).toBeNull()
  })

  it('成书清单删除走确认弹层，删除后刷新清单', async () => {
    booksList.mockResolvedValue(books)
    renderPanel()
    await screen.findByRole('list', { name: '成书清单' })
    fireEvent.click(screen.getByLabelText('删除 风雪夜归'))
    bookDelete.mockResolvedValue(undefined)
    // Popconfirm 确认键=「删 除」（antd 两字插空格）；精确匹配避免命中左侧图标钮
    fireEvent.click(await screen.findByRole('button', { name: /^删\s除$/ }))
    await waitFor(() => expect(bookDelete).toHaveBeenCalledWith('/books/风雪夜归.txt'))
    await waitFor(() => expect(booksList).toHaveBeenCalledTimes(2))
  })

  it('送小说：确认后逐参对齐导入（full 全本）并派发 NAVIGATE 跳书架', async () => {
  booksList.mockResolvedValue(books)
  importEx.mockResolvedValue({ path: '/p', title: '风雪夜归', chapter_count: 3, total_words: 100 })
  const spy = vi.spyOn(window, 'dispatchEvent')
  renderPanel()
  await screen.findByRole('list', { name: '成书清单' })
  fireEvent.click(screen.getByLabelText('送 风雪夜归 入小说书架'))
  fireEvent.click(await screen.findByRole('button', { name: /^导\s*入$/ }))
  await waitFor(() => expect(importEx).toHaveBeenCalledWith('/books/风雪夜归.txt', '风雪夜归', '未分类', '默认', 'full', 0))
  await waitFor(() => {
    const ev = spy.mock.calls.map(([e]) => e as Event).find((e) => e.type === 'navigate') as CustomEvent | undefined
    expect(ev?.detail).toEqual({ page: 'novel' })
  })
})

it('EPUB：点击导出调用绑定并提示成功', async () => {
  booksList.mockResolvedValue(books)
  exportEpub.mockResolvedValue('/books/风雪夜归.epub')
  renderPanel()
  await screen.findByRole('list', { name: '成书清单' })
  fireEvent.click(screen.getByLabelText('导出 风雪夜归 的 EPUB'))
  await waitFor(() => expect(exportEpub).toHaveBeenCalledWith('/books/风雪夜归.txt'))
  await screen.findByText('已导出 EPUB（与 TXT 同目录）')
})
})
