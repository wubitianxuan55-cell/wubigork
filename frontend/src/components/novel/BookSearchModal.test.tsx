// BookSearchModal.test.tsx — 书架「在线搜书」流程 Modal（书源取书→拆书导入 t2）。
// bridge app 代理 mock（ConsistencyPanel.test 同款）+ window.runtime 桩捕事件：
// 覆盖候选表 HasRule 打标与免规则禁用、目录预览与范围默认/收窄、导入起跑参数、
// 进度/完成/失败三路事件与取消。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import React from 'react'

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      NovelBookSourceSearch: vi.fn(),
      NovelBookSourceToc: vi.fn(),
      NovelBookSourceImport: vi.fn(),
      NovelBookSourceImportCancel: vi.fn(),
      NovelBookSourceImportChapters: vi.fn(),
    },
  }
})

import BookSearchModal from './BookSearchModal'
import { app } from '../../gaea/lib/bridge'
import type { NovelBookSourceCandidate, NovelBookSourceTocPreview } from '../../gaea/lib/bridge/novel'

const search = vi.mocked(app.NovelBookSourceSearch)
const toc = vi.mocked(app.NovelBookSourceToc)
const importStart = vi.mocked(app.NovelBookSourceImport)
const cancelJob = vi.mocked(app.NovelBookSourceImportCancel)
const retryChapters = vi.mocked(app.NovelBookSourceImportChapters)

// window.runtime 桩：subscribe() 走 EventsOn，这里捕获 handler 手工投递事件。
let deliver: ((data: unknown) => void) | null = null
function stubRuntime() {
  deliver = null
  ;(window as unknown as { runtime: unknown }).runtime = {
    EventsOn: vi.fn((_ch: string, h: (data: unknown) => void) => { deliver = h }),
    EventsOff: vi.fn(),
  }
}

const candidates: NovelBookSourceCandidate[] = [
  {
    kind: 'rule', source: '笔趣阁', title: '风雪夜归', author: '佚名',
    url: 'https://a.example.com/book/1', host: 'a.example.com', hasRule: true,
    latestChapter: '第12章 大结局',
  },
  {
    kind: 'web', source: 'bing', title: '风雪夜归 最新章节',
    url: 'https://b.example.com/fsyg/', host: 'b.example.com', hasRule: false,
  },
]

const tocPreview: NovelBookSourceTocPreview = {
  total: 12,
  sample: Array.from({ length: 12 }, (_, i) => ({ title: `第${i + 1}章 样例`, url: `u${i + 1}`, order: i + 1 })),
  truncated: false,
}

type ModalProps = Parameters<typeof BookSearchModal>[0]

function renderModal(overrides: Partial<ModalProps> = {}) {
  const props: ModalProps = { open: true, onClose: vi.fn(), onImported: vi.fn(), onAppended: vi.fn(), ...overrides }
  render(<BookSearchModal {...props} />)
  return props
}

async function searchAndPick(props: Partial<ModalProps> = {}) {
  search.mockResolvedValue({ candidates, warnings: [] })
  toc.mockResolvedValue(tocPreview)
  const out = renderModal(props)
  fireEvent.change(screen.getByLabelText('在线搜书关键字'), { target: { value: '风雪夜归' } })
  fireEvent.click(screen.getByRole('button', { name: /搜\s*书/ }))
  await waitFor(() => expect(screen.getAllByText('风雪夜归').length).toBeGreaterThan(0))
  fireEvent.click(screen.getAllByRole('button', { name: /选\s*书/ })[0])
  await waitFor(() => expect(screen.getByText('共 12 章')).toBeTruthy())
  return out
}

function numberInput(label: string): HTMLInputElement {
  const el = screen.getByLabelText(label)
  return (el.tagName === 'INPUT' ? el : el.querySelector('input')) as HTMLInputElement
}

beforeEach(() => {
  vi.clearAllMocks()
  stubRuntime()
})

describe('候选表（搜索结果）', () => {
  it('HasRule 打标：规则命中可导入，免规则禁用按钮且标注仅参考', async () => {
    search.mockResolvedValue({ candidates, warnings: [] })
    renderModal()
    fireEvent.change(screen.getByLabelText('在线搜书关键字'), { target: { value: '风雪夜归' } })
    fireEvent.click(screen.getByRole('button', { name: /搜\s*书/ }))
    await waitFor(() => expect(screen.getAllByText('风雪夜归').length).toBeGreaterThan(0))

    expect(screen.getByText('可导入')).toBeTruthy()
    expect(screen.getByText('免规则 · 正文不可解析')).toBeTruthy()
    expect(screen.getByText(/来源：笔趣阁（书源）/)).toBeTruthy()
    expect(screen.getByText(/最新：第12章 大结局/)).toBeTruthy()
    const pickButtons = screen.getAllByRole('button', { name: /选\s*书|仅参考/ })
    expect((pickButtons[0] as HTMLButtonElement).disabled).toBe(false)
    expect((pickButtons[1] as HTMLButtonElement).disabled).toBe(true)
  })

  it('各源失败告警如实透出不静默', async () => {
    search.mockResolvedValue({ candidates: [], warnings: ['搜索引擎「bing」失败: 超时'] })
    renderModal()
    fireEvent.change(screen.getByLabelText('在线搜书关键字'), { target: { value: 'x' } })
    fireEvent.click(screen.getByRole('button', { name: /搜\s*书/ }))
    await waitFor(() =>
      expect(screen.getByText(/部分来源未返回：搜索引擎「bing」失败: 超时/)).toBeTruthy())
  })
})

describe('目录预览与范围选择', () => {
  it('总数 + 样例 + 范围默认全本，起跑参数逐参对齐', async () => {
    await searchAndPick()
    importStart.mockResolvedValue({ jobId: 'job-1' })

    fireEvent.change(screen.getByPlaceholderText('书名（必填）'), { target: { value: '风雪夜归' } })
    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))

    await waitFor(() => expect(importStart).toHaveBeenCalledTimes(1))
    expect(importStart).toHaveBeenCalledWith(
      '笔趣阁', 'https://a.example.com/book/1', 1, 12, '风雪夜归', '未分类', '默认')
  })

  it('范围收窄后按所选范围起跑', async () => {
    await searchAndPick()
    importStart.mockResolvedValue({ jobId: 'job-2' })

    fireEvent.change(numberInput('起始章'), { target: { value: '3' } })
    fireEvent.change(screen.getByPlaceholderText('书名（必填）'), { target: { value: '风雪夜归' } })
    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))

    await waitFor(() => expect(importStart).toHaveBeenCalledTimes(1))
    expect(importStart).toHaveBeenCalledWith(
      '笔趣阁', 'https://a.example.com/book/1', 3, 12, '风雪夜归', '未分类', '默认')
  })
})

describe('导入进度与终态', () => {
  const result = {
    path: 'C:/novels/风雪夜归', title: '风雪夜归',
    chapter_count: 12, total_words: 34000, split_strategy: 'booksource',
  }

  it('progress 更新计数，done 带结果回调 onImported 并退订', async () => {
    const { onImported } = await searchAndPick()
    importStart.mockResolvedValue({ jobId: 'job-9' })

    fireEvent.change(screen.getByPlaceholderText('书名（必填）'), { target: { value: '风雪夜归' } })
    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))
    await waitFor(() => expect(deliver).not.toBeNull())

    deliver?.({ type: 'progress', done: 6, total: 12 })
    expect(await screen.findByText('6/12 章')).toBeTruthy()

    deliver?.({ type: 'done', result, failed: [] })
    await waitFor(() => expect(onImported).toHaveBeenCalledWith(result))
  })

  it('done 带失败清单：Modal 留失败面板可一键重试补下（t3）', async () => {
    const { onImported, onAppended } = await searchAndPick()
    importStart.mockResolvedValue({ jobId: 'job-12' })
    retryChapters.mockResolvedValue({ jobId: 'job-13' })

    fireEvent.change(screen.getByPlaceholderText('书名（必填）'), { target: { value: '风雪夜归' } })
    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))
    await waitFor(() => expect(deliver).not.toBeNull())

    const failed = [{ title: '第7章 断章', url: 'https://a.example.com/ch/7', error: '超时' }]
    deliver?.({ type: 'done', result, failed })
    expect(await screen.findByText(/1 章下载失败，未入库/)).toBeTruthy()
    expect(onImported).toHaveBeenCalledWith(result)

    fireEvent.click(screen.getByRole('button', { name: /重试补下 1 章/ }))
    await waitFor(() => expect(retryChapters).toHaveBeenCalledTimes(1))
    expect(retryChapters).toHaveBeenCalledWith(
      '笔趣阁', result.path, JSON.stringify([{ title: '第7章 断章', url: 'https://a.example.com/ch/7' }]))

    deliver?.({ type: 'progress', done: 1, total: 1 })
    expect(await screen.findByText('1/1 章')).toBeTruthy()

    deliver?.({ type: 'append-done', result: { path: result.path, title: result.title, appended: 1, totalChapters: 25, addedWords: 120 } })
    await waitFor(() => expect(onAppended).toHaveBeenCalledWith({ path: result.path, title: result.title, appended: 1, totalChapters: 25, addedWords: 120 }))
    expect(await screen.findByText(/已补下 1 章，全部章节已齐/)).toBeTruthy()
  })

  it('done 无失败：Modal 自动关闭（关闭权在组件）', async () => {
    const { onClose } = await searchAndPick()
    importStart.mockResolvedValue({ jobId: 'job-14' })

    fireEvent.change(screen.getByPlaceholderText('书名（必填）'), { target: { value: '风雪夜归' } })
    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))
    await waitFor(() => expect(deliver).not.toBeNull())

    deliver?.({ type: 'done', result, failed: [] })
    await waitFor(() => expect(onClose).toHaveBeenCalled())
  })

  it('error 事件直显失败原因（含未取到计数），可重试', async () => {
    await searchAndPick()
    importStart.mockResolvedValue({ jobId: 'job-10' })

    fireEvent.change(screen.getByPlaceholderText('书名（必填）'), { target: { value: '风雪夜归' } })
    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))
    await waitFor(() => expect(deliver).not.toBeNull())

    deliver?.({ type: 'error', error: '整本下载失败：12 章全部未取到', failed: 12 })
    expect(await screen.findByText(/整本下载失败：12 章全部未取到（12 章未取到）/)).toBeTruthy()
    expect((screen.getByRole('button', { name: /开始导入/ }) as HTMLButtonElement).disabled).toBe(false)
  })

  it('取消按钮调用 NovelBookSourceImportCancel', async () => {
    await searchAndPick()
    importStart.mockResolvedValue({ jobId: 'job-11' })
    cancelJob.mockResolvedValue(true)

    fireEvent.change(screen.getByPlaceholderText('书名（必填）'), { target: { value: '风雪夜归' } })
    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))
    await waitFor(() => expect(deliver).not.toBeNull())

    fireEvent.click(screen.getByRole('button', { name: /取\s*消/ }))
    await waitFor(() => expect(cancelJob).toHaveBeenCalledWith('job-11'))
  })
})
