// BookSearchModal.test.tsx — 书架「在线搜书」流程 Modal（书源取书→拆书导入 t2）。
// bridge app 代理 mock（ConsistencyPanel.test 同款）+ window.runtime 桩捕事件：
// 覆盖候选表 HasRule 打标与免规则禁用、目录预览与范围默认/收窄、导入起跑参数、
// 进度/完成/失败三路事件与取消。
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'

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
      NovelBookSourceEnginesGet: vi.fn(),
      NovelBookSourceEnginesSave: vi.fn(),
      // v4.421.0 失落兜底：唯一可用于对账的既有绑定（书架成书清单）
      ListProjects: vi.fn(),
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
const listProjects = vi.mocked(app.ListProjects)

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

  it('搜索历史：seed 渲染 chips，点击即用该关键字搜索（t3 余项）', async () => {
    localStorage.setItem('gaea.booksearch.history', JSON.stringify(['大道朝天']))
    search.mockResolvedValue({ candidates: [{ kind: 'rule', source: '测试源', title: '大道朝天', url: 'https://a.test/b/1', host: 'a.test', hasRule: true }], warnings: [] })
    renderModal()

    const chip = await screen.findByText('大道朝天')
    fireEvent.click(chip)
    await waitFor(() => expect(search).toHaveBeenCalledWith('大道朝天'))
    expect(screen.getByText(/来源：测试源/)).toBeTruthy()

    // 搜索成功后历史记录该关键字
    expect(localStorage.getItem('gaea.booksearch.history')).toContain('大道朝天')
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

// v4.421.0 失落兜底（P1）：后端 NovelBookSourceImport 先起 goroutine 再返回 jobId
// （novel_booksource_handler.go:332/:359），快速失败/瞬时完成的终态事件会在前端
// 订阅建立前就 emit 完 → UI 曾永远停在「后台下载中…」。前端无 job 状态绑定，唯一可用于
// 对账的既有绑定是书架成书清单（CoreB.ListProjects）；兜底只按清单如实对账，不假装成功。
describe('导入进度事件失落兜底（v4.421.0）', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  /** fake timers 下不用 waitFor（只推进定时器 + 冲 microtask）。 */
  const flush = async () => { await act(async () => { await vi.advanceTimersByTimeAsync(0) }) }

  function renderControlled() {
    const props: ModalProps = { open: true, onClose: vi.fn(), onImported: vi.fn(), onAppended: vi.fn() }
    const view = render(<BookSearchModal {...props} />)
    return { props, view }
  }

  /** 走到「已选书 + 已填书名」的导入表单（全程 fake timers）。 */
  async function toImportForm() {
    search.mockResolvedValue({ candidates, warnings: [] })
    toc.mockResolvedValue(tocPreview)
    const out = renderControlled()
    fireEvent.change(screen.getByLabelText('在线搜书关键字'), { target: { value: '风雪夜归' } })
    fireEvent.click(screen.getByRole('button', { name: /搜\s*书/ }))
    await flush()
    fireEvent.click(screen.getAllByRole('button', { name: /选\s*书/ })[0])
    await flush()
    fireEvent.change(screen.getByPlaceholderText('书名（必填）'), { target: { value: '风雪夜归' } })
    return out
  }

  it('3s 内无任何进度事件 → 轮询成书清单对账：命中即如实报已入库 + 给「完成」', async () => {
    vi.useFakeTimers()
    listProjects.mockResolvedValue([{ title: '风雪夜归', path: 'C:/novels/风雪夜归', chapter_count: 12 }])
    const { props } = await toImportForm()
    importStart.mockResolvedValue({ jobId: 'job-lost-1' })

    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))
    await flush()
    expect(deliver).not.toBeNull()
    expect(listProjects).not.toHaveBeenCalled()

    // 3s 到点：进入兜底 → 按清单对账命中
    await act(async () => { await vi.advanceTimersByTimeAsync(3000) })
    const lost = screen.getByTestId('import-progress-lost')
    expect(listProjects).toHaveBeenCalledTimes(1)
    expect(within(lost).getByText(/已按书架成书清单确认《风雪夜归》入库（12 章）/)).toBeTruthy()
    expect(screen.getByTestId('import-shelf-done')).toBeTruthy()

    fireEvent.click(screen.getByTestId('import-shelf-done'))
    expect(props.onClose).toHaveBeenCalled()
  })

  // v4.425 B9 反向守卫：兜底对账命中书架时必须通知父层刷新书架（否则这本新书在
  // 书架上仍不可见——对账只发生在本 Modal 内部，父层无从得知）。
  it('兜底对账命中：回调 onShelfReconciled 通知父层刷新书架', async () => {
    vi.useFakeTimers()
    listProjects.mockResolvedValue([{ title: '风雪夜归', path: 'C:/novels/风雪夜归', chapter_count: 12 }])
    const onShelfReconciled = vi.fn()
    const props: ModalProps = {
      open: true, onClose: vi.fn(), onImported: vi.fn(), onAppended: vi.fn(), onShelfReconciled,
    }
    search.mockResolvedValue({ candidates, warnings: [] })
    toc.mockResolvedValue(tocPreview)
    render(<BookSearchModal {...props} />)
    fireEvent.change(screen.getByLabelText('在线搜书关键字'), { target: { value: '风雪夜归' } })
    fireEvent.click(screen.getByRole('button', { name: /搜\s*书/ }))
    await flush()
    fireEvent.click(screen.getAllByRole('button', { name: /选\s*书/ })[0])
    await flush()
    fireEvent.change(screen.getByPlaceholderText('书名（必填）'), { target: { value: '风雪夜归' } })
    importStart.mockResolvedValue({ jobId: 'job-lost-9' })

    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))
    await flush()
    await act(async () => { await vi.advanceTimersByTimeAsync(3000) })

    expect(onShelfReconciled).toHaveBeenCalledTimes(1)
  })

  it('清单查不到：如实提示未确认 + 刷新清单按钮；刷新后命中才报已入库', async () => {
    vi.useFakeTimers()
    listProjects.mockResolvedValue([])
    await toImportForm()
    importStart.mockResolvedValue({ jobId: 'job-lost-2' })

    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))
    await flush()
    await act(async () => { await vi.advanceTimersByTimeAsync(3000) })

    // 未命中：如实说「无法确认」，不假装成功，也不静默
    const lost = screen.getByTestId('import-progress-lost')
    expect(within(lost).getByText(/未收到导入进度回调，无法确认后台状态/)).toBeTruthy()
    expect(screen.getByTestId('import-shelf-close')).toBeTruthy()
    expect(screen.queryByTestId('import-shelf-done')).toBeNull()

    // 手动刷新清单：此时下载已落库 → 报实章数
    listProjects.mockResolvedValue([{ title: '风雪夜归', path: 'C:/novels/风雪夜归', chapter_count: 12 }])
    fireEvent.click(screen.getByTestId('import-shelf-refresh'))
    await flush()
    expect(within(screen.getByTestId('import-progress-lost')).getByText(/已按书架成书清单确认《风雪夜归》入库（12 章）/)).toBeTruthy()
    expect(screen.getByTestId('import-shelf-done')).toBeTruthy()
  })

  it('兜底期间真实事件到达：撤销兜底、回到正常进度（真实事件优先于清单对账）', async () => {
    vi.useFakeTimers()
    listProjects.mockResolvedValue([])
    await toImportForm()
    importStart.mockResolvedValue({ jobId: 'job-lost-3' })

    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))
    await flush()
    await act(async () => { await vi.advanceTimersByTimeAsync(3000) })
    expect(screen.getByTestId('import-progress-lost')).toBeTruthy()

    await act(async () => { deliver?.({ type: 'progress', done: 6, total: 12 }) })
    expect(screen.queryByTestId('import-progress-lost')).toBeNull()
    expect(screen.getByText('6/12 章')).toBeTruthy()
  })

  it('兜底态关闭弹窗：清掉 importing（重开不残留「后台下载中…」）', async () => {
    vi.useFakeTimers()
    listProjects.mockResolvedValue([])
    const { props, view } = await toImportForm()
    importStart.mockResolvedValue({ jobId: 'job-lost-4' })

    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))
    await flush()
    await act(async () => { await vi.advanceTimersByTimeAsync(3000) })

    fireEvent.click(screen.getByTestId('import-shelf-close'))
    expect(props.onClose).toHaveBeenCalled()

    // 父层关窗 → 打开：导入态已被清掉，回到搜索起点
    view.rerender(<BookSearchModal {...props} open={false} />)
    await flush()
    view.rerender(<BookSearchModal {...props} open />)
    await flush()
    expect(screen.queryByText('后台下载中…')).toBeNull()
    expect(screen.getByLabelText('在线搜书关键字')).toBeTruthy()
  })

  it('补下（append-done）同款竞态：无回调时按清单章数如实对账，不谎称补下成功', async () => {    vi.useFakeTimers()
    listProjects.mockResolvedValue([{ title: '风雪夜归', path: 'C:/novels/风雪夜归', chapter_count: 25 }])
    await toImportForm()
    importStart.mockResolvedValue({ jobId: 'job-lost-5' })
    retryChapters.mockResolvedValue({ jobId: 'job-lost-6' })

    fireEvent.click(screen.getByRole('button', { name: /开始导入/ }))
    await flush()
    // 整本 done 带失败章 → 留失败面板
    await act(async () => {
      deliver?.({ type: 'done', result: { path: 'C:/novels/风雪夜归', title: '风雪夜归', chapter_count: 12, total_words: 34000 }, failed: [{ title: '第7章', url: 'u7', error: '超时' }] })
    })
    fireEvent.click(screen.getByRole('button', { name: /重试补下 1 章/ }))
    await flush()

    await act(async () => { await vi.advanceTimersByTimeAsync(3000) })
    expect(within(screen.getByTestId('import-progress-lost'))
      .getByText(/未收到补下进度回调；书架清单显示《风雪夜归》现有 25 章/)).toBeTruthy()
  })
})
