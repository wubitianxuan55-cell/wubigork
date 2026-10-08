// ChapterPage.test.tsx — 章节页组件冒烟测试（ChapterPage 拆分补测第一步）。
// 手法沿用 SettingsPage.test / NovelSettingPage.test：vi.mock 屏蔽 wails 绑定、
// 重型子组件（TTS/编辑器/导出/配图）替换为桩，zustand store 用真实实例 + setState 注入。
// 只锁：组件可渲染、章节 Tab 出现、阅读模式正文渲染，全程不抛错——不追求深覆盖。
// 另锁搜索定位接线（第三批）：阅读模式内点命中可定位、同章再点仍能重新定位（回归修复）。
// 第四批补锁：搜索命中「落为划线」→ 划线 state/持久化/正文回渲染，标题命中按钮禁用。
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { message, Modal } from 'antd'
import { writeReadingProgress } from '../utils/readingProgress'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go，章节读写全部给确定性返回。
// P3 版3 双轨退役（终局）：wailsjsCompat shim 已退役——NovelSearch/章节族全部
// 并入 bindingsBridge 共享 vi.fn，bridge Proxy 统一从 bindingsBridge 取函数，
// 断言两端皆命中。
const bindingsBridge = vi.hoisted(() => ({
  NovelSearch: vi.fn(),
  // AI 伴读（摘要 / 问书）：迟到响应守卫（A9）用例需要挂起在途请求
  NovelReadingAsk: vi.fn(),
  GetChapter: vi.fn().mockResolvedValue({ content: '夜色沉沉，雨落在窗台上。\n\n他推门而入，灯还亮着。' }),
  GetChapterBranch: vi.fn().mockResolvedValue({ content: '' }),
  SaveChapterContent: vi.fn().mockResolvedValue(undefined),
  SaveChapterBranchContent: vi.fn().mockResolvedValue(undefined),
}))
// NovelB 场景族直调（阅读页场景化）：默认 IsProjectV4=false → 既有用例全走
// blob 模式；场景制用例在各自 it 内覆写为 true 并给 GetChapterScenes/SaveScene 桩。
const novelB = vi.hoisted(() => ({
  IsProjectV4: vi.fn().mockResolvedValue(false),
  GetChapterScenes: vi.fn().mockResolvedValue([]),
  SaveScene: vi.fn().mockResolvedValue(undefined),
  CreateScene: vi.fn().mockResolvedValue({ id: '009-scene-n' }),
}))
vi.mock('../../wailsjs/go/app/NovelB', () => ({
  IsProjectV4: novelB.IsProjectV4,
  GetChapterScenes: novelB.GetChapterScenes,
  SaveScene: novelB.SaveScene,
  CreateScene: novelB.CreateScene,
}))
vi.mock('../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../gaea/lib/bridge')>()
  return {
    ...actual,
    app: new Proxy({} as typeof actual.app, {
      get(_t, prop: string) {
        const bridgeMock = (bindingsBridge as unknown as Record<string, unknown>)[prop]
        if (typeof bridgeMock === 'function') return bridgeMock
        const fallback = (actual.app as unknown as Record<string, unknown>)[prop]
        if (typeof fallback === 'function') return fallback.bind(actual.app)
        return undefined
      },
    }),
  }
})

// 重型子组件桩：冒烟只关心 ChapterPage 自身的装配与状态流转。
// 编辑器桩额外暴露「当前缓冲正文」与一次「模拟输入」——A2b/A10 的护栏要断言
// 「本地未保存正文没有被服务端正文覆盖」，而真实编辑器不在本文件的渲染面内。
vi.mock('../components/TTSPlayer', () => ({ default: () => <div data-testid="tts-player-stub" /> }))
vi.mock('../components/novel/ChapterEditor', () => ({
  default: ({ tab, onUpdate }: { tab: { scenes: string[] }; onUpdate: (field: string, value: unknown) => void }) => (
    <div data-testid="chapter-editor-stub">
      <span data-testid="chapter-editor-scenes">{tab.scenes.join('|')}</span>
      <button
        type="button"
        data-testid="chapter-editor-type"
        onClick={() => {
          onUpdate('scenes', [...tab.scenes, '作者手写的新段落。'])
          onUpdate('saved', false)
        }}
      >
        模拟输入
      </button>
    </div>
  ),
}))
vi.mock('../components/novel/ExportPanel', () => ({ default: () => <div /> }))
vi.mock('./chapter/ChapterIllustration', () => ({ default: () => <div /> }))
// 重写历史面板桩：只保留「打开与否」与「服务端已应用 → onApplied」两个接线面，
// 面板自身的列表/详情交互由 RewriteHistoryPanel.test 覆盖（其文件归属另一条线）。
vi.mock('../components/novel/RewriteHistoryPanel', () => ({
  default: ({ open, chapterNum, onApplied }: { open: boolean; chapterNum: number | null; onApplied?: () => void }) => (
    open ? (
      <div data-testid="rw-hist-panel" data-chapter={String(chapterNum)}>
        <button type="button" data-testid="rw-hist-apply" onClick={() => onApplied?.()}>应用此版本</button>
      </div>
    ) : null
  ),
}))

import ChapterPage from './ChapterPage'
import { useAppStore } from '../stores/appStore'
import { useOutlineStore } from '../stores/outlineStore'
import { dirtyNovelProviders } from '../components/novel/novelSwitchGuard'
import type { OutlineNode } from '../types'

// NovelSearch 断言引用 = bindingsBridge 共享 vi.fn（shim 已退役）
const NovelSearch = bindingsBridge.NovelSearch

// 最小消费面的大纲叶子节点（章节打开走 order_index → GetChapter）
const leaf = {
  id: 'ch-1',
  title: '第一回 风雪夜归人',
  order_index: 1,
} as unknown as OutlineNode

beforeEach(() => {
  vi.clearAllMocks()
  // 真实 zustand store：与 NovelSettingPage.test 同款 setState 注入，测试间复位
  useAppStore.setState({ projectPath: '' })
  useOutlineStore.setState({ outlines: [], loading: false, error: null })
  // vi.clearAllMocks 会清掉工厂默认返回值：恢复（默认 blob 模式）
  novelB.IsProjectV4.mockResolvedValue(false)
  novelB.GetChapterScenes.mockResolvedValue([])
  novelB.SaveScene.mockResolvedValue(undefined)
  novelB.CreateScene.mockResolvedValue({ id: '009-scene-n' })
})

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ChapterPage 冒烟', () => {
  it('无章节时渲染空态提示，不抛错', async () => {
    render(<ChapterPage />)
    expect(await screen.findByText('从左侧大纲选择章节开始阅读')).toBeTruthy()
  })

  it('经 novel:open-chapter 事件打开章节：Tab 出现、编辑器装配、拉取正文', async () => {
    useOutlineStore.setState({ outlines: [leaf] })
    render(<ChapterPage />)
    // ChapterPage 在壳层外监听该事件（大纲树位于壳层左 zone）
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leaf } }))
    expect(await screen.findByRole('tab', { name: /第一回 风雪夜归人/ })).toBeTruthy()
    expect(await screen.findByTestId('chapter-editor-stub')).toBeTruthy()
    // 章节出现后底部快捷键栏可见（空态时不渲染）。
    // 注意「F11」在 <kbd> 内，RTL 默认只匹配元素直属文本节点，故按后半段断言。
    expect(screen.getByText(/专注模式/)).toBeTruthy()
    // 未保存 → 加载成功后转「已保存」
    expect(await screen.findByText('已保存')).toBeTruthy()
  })

  it('进入阅读模式：渲染 .novel-reading-p 正文段落', async () => {
    useOutlineStore.setState({ outlines: [leaf] })
    render(<ChapterPage />)
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leaf } }))
    await screen.findByTestId('chapter-editor-stub')
    fireEvent.click(screen.getByRole('button', { name: '进入阅读模式' }))
    expect(await screen.findByText('夜色沉沉，雨落在窗台上。')).toBeTruthy()
    // 场景按空行切成两段阅读段落
    expect(document.querySelectorAll('.novel-reading-p').length).toBe(2)
  })

  it('阅读模式添加书签：列表出现摘录（textAtScrollTop → 书签预览接线）', async () => {
    useOutlineStore.setState({ outlines: [leaf] })
    render(<ChapterPage />)
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leaf } }))
    await screen.findByTestId('chapter-editor-stub')
    fireEvent.click(screen.getByRole('button', { name: '进入阅读模式' }))
    await screen.findByText('夜色沉沉，雨落在窗台上。')
    // 打开书签 Popover → 在当前位置添加
    fireEvent.click(screen.getByRole('button', { name: '书签' }))
    fireEvent.click(await screen.findByRole('button', { name: '在当前位置添加书签' }))
    expect(await screen.findByText('本章书签（1）')).toBeTruthy()
    // jsdom 中 offsetTop 恒 0：所有段落都在 48px 容差内 → 摘录取最后一段
    const list = await screen.findByText('本章书签（1）')
    expect(list.closest('.novel-read-bookmarks')!.textContent).toContain('他推门而入，灯还亮着。')
  })

  it('搜索定位：同章内点击命中可定位，再点另一处命中可重新定位（回归修复）', async () => {
    // 默认章节正文两段各含一个「，」→ 构造同章两处命中（第1段 offset 4 / 第2段 offset 5）
    const hits = [
      { node_id: 'ch-1', title: '第一回 风雪夜归人', chapter_num: 1, snippet: '夜色沉沉，', title_hit: false, match_index: 1, paragraph_index: 0, char_offset: 4, match_len: 1, total_hits: 2, chapter_count: 1 },
      { node_id: 'ch-1', title: '第一回 风雪夜归人', chapter_num: 1, snippet: '他推门而入，', title_hit: false, match_index: 2, paragraph_index: 1, char_offset: 5, match_len: 1, total_hits: 2, chapter_count: 1 },
    ] as unknown as Awaited<ReturnType<typeof NovelSearch>>
    vi.mocked(NovelSearch).mockResolvedValue(hits)
    useOutlineStore.setState({ outlines: [leaf] })
    render(<ChapterPage />)
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leaf } }))
    await screen.findByTestId('chapter-editor-stub')
    fireEvent.click(screen.getByRole('button', { name: '进入阅读模式' }))
    await screen.findByText('夜色沉沉，雨落在窗台上。')

    // 打开搜索 → 输入（300ms 防抖为真实计时）→ 命中列表出现
    fireEvent.click(screen.getByRole('button', { name: '全文搜索' }))
    fireEvent.change(screen.getByPlaceholderText('搜索全书（标题 + 正文）'), { target: { value: '，' } })
    const row1 = await screen.findByText('第一回 风雪夜归人 · 第1段')
    fireEvent.click(row1.closest('.novel-read-search-hit-row')!)

    // 修复前缺陷：同章内点命中（readMode/readNodeId 均不变）定位 effect 不重跑，无任何高亮
    await waitFor(() => expect(document.querySelector('span.novel-reading-search-hit')).not.toBeNull())
    const paras = () => Array.from(document.querySelectorAll('.novel-reading-p'))
    expect(paras()[0].querySelector('span.novel-reading-search-hit')).not.toBeNull()

    // 重开搜索浮层 → 点第2段的命中：应清掉旧高亮、重新定位到第2段
    fireEvent.click(screen.getByRole('button', { name: '全文搜索' }))
    const row2 = await screen.findByText('第一回 风雪夜归人 · 第2段')
    fireEvent.click(row2.closest('.novel-read-search-hit-row')!)
    await waitFor(() => {
      const spans = document.querySelectorAll('span.novel-reading-search-hit')
      expect(spans.length).toBe(1)
      expect(paras()[1].contains(spans[0])).toBe(true)
    })
  })
})

describe('搜索命中「落为划线」', () => {
  const project = 'C:/demo-proj'
  const annKey = `gaea.novel.readingAnnotations.${project}`
  const bodyHit = {
    node_id: 'ch-1',
    title: '第一回 风雪夜归人',
    chapter_num: 1,
    snippet: '夜色沉沉，',
    title_hit: false,
    match_index: 1,
    paragraph_index: 0,
    char_offset: 4,
    match_len: 1,
    total_hits: 1,
    chapter_count: 1,
  }

  // 挂载到「阅读模式 + 搜索浮层已出命中列表」状态（非 projectPath 下 writeAnnotations 不落盘，
  // 故本组用例注入真实项目路径以断言持久化）
  const mountReadingWithSearch = async (hits: unknown) => {
    vi.mocked(NovelSearch).mockResolvedValue(hits as Awaited<ReturnType<typeof NovelSearch>>)
    useAppStore.setState({ projectPath: project })
    useOutlineStore.setState({ outlines: [leaf] })
    render(<ChapterPage />)
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leaf } }))
    await screen.findByTestId('chapter-editor-stub')
    fireEvent.click(screen.getByRole('button', { name: '进入阅读模式' }))
    await screen.findByText('夜色沉沉，雨落在窗台上。')
    fireEvent.click(screen.getByRole('button', { name: '全文搜索' }))
    fireEvent.change(screen.getByPlaceholderText('搜索全书（标题 + 正文）'), { target: { value: '，' } })
    // 汇总提示出现即命中列表就绪（标题命中行无「· 第N段」后缀，不能按行文本等）
    await screen.findByText('共 1 处 · 1 章')
  }

  beforeEach(() => { localStorage.removeItem(annKey) })
  afterEach(() => { localStorage.removeItem(annKey) })

  it('点「落为划线」：message 反馈、持久化写命中章节 + 命中原文、正文即时回渲染 mark、不触发定位跳转', async () => {
    const msgSpy = vi.spyOn(message, 'success')
    await mountReadingWithSearch([bodyHit])

    fireEvent.click(screen.getByRole('button', { name: '落为划线' }))
    expect(msgSpy).toHaveBeenCalledTimes(1)

    // 持久化走 writeAnnotations 既有管线：归属命中章节，摘录为命中原文（落库后可回渲染）
    const stored = JSON.parse(localStorage.getItem(annKey) || '[]') as Array<Record<string, unknown>>
    expect(stored).toHaveLength(1)
    expect(stored[0]).toMatchObject({ nodeId: 'ch-1', text: '，', color: 'yellow' })

    // 划线 state：既有回渲染 effect 立即把当前章命中处包成 mark；列表计数 +1
    await waitFor(() => expect(document.querySelector('mark.novel-reading-mark')?.textContent).toBe('，'))
    fireEvent.click(screen.getByRole('button', { name: '划线 / 想法' }))
    expect(await screen.findByText('本章划线 / 想法（1）')).toBeTruthy()

    // 行内按钮不冒泡触发行点击（openSearchHit）：搜索浮层仍开、无 2.6s 临时 search-hit 高亮
    expect(screen.getByPlaceholderText('搜索全书（标题 + 正文）')).toBeTruthy()
    expect(document.querySelector('span.novel-reading-search-hit')).toBeNull()
    msgSpy.mockRestore()
  })

  it('标题命中（paragraph_index = -1）：「落为划线」禁用，点击不产生划线', async () => {
    const msgSpy = vi.spyOn(message, 'success')
    await mountReadingWithSearch([{
      ...bodyHit,
      snippet: '风雪夜归人',
      title_hit: true,
      paragraph_index: -1,
      char_offset: -1,
    }])
    const btn = screen.getByRole('button', { name: '落为划线' })
    expect((btn as HTMLButtonElement).disabled).toBe(true)
    fireEvent.click(btn)
    expect(msgSpy).not.toHaveBeenCalled()
    expect(localStorage.getItem(annKey)).toBeNull()
    msgSpy.mockRestore()
  })
})

describe('ChapterPage 阅读页场景化（V4 主线章读场景、逐场景保存）', () => {
  async function mountSceneChapter() {
    novelB.IsProjectV4.mockResolvedValue(true)
    novelB.GetChapterScenes.mockResolvedValue([
      { id: '001-chapter', content: '第一场正文。' },
      { id: '002-scene-2', content: '第二场正文。' },
    ])
    useOutlineStore.setState({ outlines: [leaf] })
    useAppStore.setState({ projectPath: 'C:/proj/novel' })
    render(<ChapterPage />)
    // 等 V4 探测完成并 flush 微任务（ref 镜像写入），再派发开章事件
    await waitFor(() => expect(novelB.IsProjectV4).toHaveBeenCalled())
    await act(async () => {})
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leaf } }))
    await screen.findByTestId('chapter-editor-stub')
    // 场景载入完成后 saved 翻转
    await screen.findByText('已保存')
  }

  it('V4 主线章：载入走 GetChapterScenes（不走 GetChapter），保存逐场景 SaveScene', async () => {
    await mountSceneChapter()
    // 载入：场景制下不读整章 blob
    expect(novelB.GetChapterScenes).toHaveBeenCalledWith(1)
    expect(bindingsBridge.GetChapter).not.toHaveBeenCalled()
    // Ctrl+S 保存 → 每场景一次 SaveScene，blob 投影由 Go 侧同步
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 's', ctrlKey: true }))
    await waitFor(() => expect(novelB.SaveScene).toHaveBeenCalledTimes(2))
    expect(novelB.SaveScene).toHaveBeenNthCalledWith(1, 1, '001-chapter', '第一场正文。')
    expect(novelB.SaveScene).toHaveBeenNthCalledWith(2, 1, '002-scene-2', '第二场正文。')
    expect(bindingsBridge.SaveChapterContent).not.toHaveBeenCalled()
  })

  it('V3/blob 章（IsProjectV4=false）：仍整章读存，行为不变', async () => {
    useOutlineStore.setState({ outlines: [leaf] })
    useAppStore.setState({ projectPath: 'C:/proj/novel' })
    render(<ChapterPage />)
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leaf } }))
    await screen.findByTestId('chapter-editor-stub')
    await screen.findByText('已保存')
    expect(bindingsBridge.GetChapter).toHaveBeenCalledWith(1)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 's', ctrlKey: true }))
    await waitFor(() => expect(bindingsBridge.SaveChapterContent).toHaveBeenCalledWith(1, '夜色沉沉，雨落在窗台上。\n\n他推门而入，灯还亮着。'))
    expect(novelB.GetChapterScenes).not.toHaveBeenCalled()
  })
})

// ── 线2 防复发守卫：常驻隐藏页不得抢窗口级快捷键（v4.421.0）──
// 背景：小说五个子页在 NovelPage 里常驻挂载（只靠 CSS display:none 隐藏，隐藏 ≠ 卸载），
// 阅读页的 window keydown 在隐藏期间依然活着。active=false 时必须「既不干活、
// 也不 preventDefault」——否则隐藏页吞掉浏览器 F11 全屏、与设定页抢一次 Ctrl+S 的
// 归属（正文反而不保存）、readMode 残留时在书架/设定页翻隐藏阅读页的章。
// 默认（不传 active）必须与改前完全一致：这是既有用例 + 本组 T2/T3 共同锁的零回归。
describe('窗口级快捷键按 active 门控（隐藏常驻页不抢键）', () => {
  const leaf2 = { id: 'ch-2', title: '第二回 灯下白头人', order_index: 2 } as unknown as OutlineNode

  /** 记录 novel:focus-mode 上报（专注模式翻转的唯一外部可观测面） */
  const watchFocusMode = () => {
    const seen: Array<boolean | undefined> = []
    const spy = (e: Event) => seen.push((e as CustomEvent<{ active?: boolean }>).detail?.active)
    window.addEventListener('novel:focus-mode', spy)
    return { seen, off: () => window.removeEventListener('novel:focus-mode', spy) }
  }

  /** 打开第 1 章并进入阅读模式（render 由调用方先行；用于「切走后 readMode 残留」真实场景） */
  async function openChapterAndRead() {
    useOutlineStore.setState({ outlines: [leaf, leaf2] })
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leaf } }))
    await screen.findByTestId('chapter-editor-stub')
    await screen.findByText('已保存')
    fireEvent.click(screen.getByRole('button', { name: '进入阅读模式' }))
    await screen.findByText('夜色沉沉，雨落在窗台上。')
  }

  it('active=false：Ctrl+S 不保存、F11 不翻转专注、阅读模式 ←/→ 不翻章，且三者都不 preventDefault', async () => {
    const focus = watchFocusMode()
    const { rerender } = render(<ChapterPage />)
    await openChapterAndRead()
    // 切走 tab：本页变成隐藏常驻页（readMode 残留，正是缺陷场景）
    rerender(<ChapterPage active={false} />)

    const saveEv = new KeyboardEvent('keydown', { key: 's', ctrlKey: true, cancelable: true })
    await act(async () => { window.dispatchEvent(saveEv) })
    expect(saveEv.defaultPrevented).toBe(false)
    expect(bindingsBridge.SaveChapterContent).not.toHaveBeenCalled()
    expect(novelB.SaveScene).not.toHaveBeenCalled()

    const f11Ev = new KeyboardEvent('keydown', { key: 'F11', cancelable: true })
    await act(async () => { window.dispatchEvent(f11Ev) })
    expect(f11Ev.defaultPrevented).toBe(false)
    // 只应有挂载时上报的初始 false，没有 F11 翻转出的 true
    expect(focus.seen).not.toContain(true)

    const rightEv = new KeyboardEvent('keydown', { key: 'ArrowRight', cancelable: true })
    await act(async () => { window.dispatchEvent(rightEv) })
    expect(rightEv.defaultPrevented).toBe(false)
    // 仍停在第 1 章：隐藏页没有翻到下一章
    expect(document.querySelector('.novel-reading-title')?.textContent).toBe('第一回 风雪夜归人')
    focus.off()
  })

  it('不传 active（默认 true）：Ctrl+S 照常保存、F11 照常翻转并 preventDefault（零回归）', async () => {
    const focus = watchFocusMode()
    render(<ChapterPage />)
    useOutlineStore.setState({ outlines: [leaf, leaf2] })
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leaf } }))
    await screen.findByTestId('chapter-editor-stub')
    await screen.findByText('已保存')

    const saveEv = new KeyboardEvent('keydown', { key: 's', ctrlKey: true, cancelable: true })
    await act(async () => { window.dispatchEvent(saveEv) })
    expect(saveEv.defaultPrevented).toBe(true)
    await waitFor(() => expect(bindingsBridge.SaveChapterContent)
      .toHaveBeenCalledWith(1, '夜色沉沉，雨落在窗台上。\n\n他推门而入，灯还亮着。'))

    const f11Ev = new KeyboardEvent('keydown', { key: 'F11', cancelable: true })
    await act(async () => { window.dispatchEvent(f11Ev) })
    expect(f11Ev.defaultPrevented).toBe(true)
    expect(focus.seen).toContain(true)
    expect(await screen.findByText(/专注模式已开启/)).toBeTruthy()
    focus.off()
  })

  it('不传 active（默认 true）时阅读模式 ←/→ 照常翻章（门控不误伤）', async () => {
    render(<ChapterPage />)
    await openChapterAndRead()

    const rightEv = new KeyboardEvent('keydown', { key: 'ArrowRight', cancelable: true })
    await act(async () => { window.dispatchEvent(rightEv) })
    expect(rightEv.defaultPrevented).toBe(true)
    await waitFor(() => expect(document.querySelector('.novel-reading-title')?.textContent)
      .toBe('第二回 灯下白头人'))
  })
})

// ── v4 场景章整章重写入口（t4-C3 收官；简化批改走「章节工具」菜单）──
describe('ChapterPage 场景章重写入口', () => {
  it('v4 场景章：chrome「工具」菜单含「整章重写」「重写历史」', async () => {
    novelB.IsProjectV4.mockResolvedValue(true)
    novelB.GetChapterScenes.mockResolvedValue([
      { id: '001-s1', content: '场景一正文。' },
      { id: '002-s2', content: '场景二正文。' },
    ])
    useOutlineStore.setState({ outlines: [leaf] })
    useAppStore.setState({ projectPath: 'C:/proj/novel' })
    render(<ChapterPage />)
    // 等 V4 探测完成（ref 镜像写入）再派发开章事件（mountSceneChapter 同款）
    await waitFor(() => expect(novelB.IsProjectV4).toHaveBeenCalled())
    await act(async () => {})
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leaf } }))
    await screen.findByTestId('chapter-editor-stub')
    await screen.findByText('已保存')
    const tools = await openToolsMenu()
    expect(tools.getByRole('menuitem', { name: '整章重写' })).toBeTruthy()
    expect(tools.getByRole('menuitem', { name: '重写历史' })).toBeTruthy()
  })

  it('v3 blob 章「工具」菜单不含重写入口（CreatePage 已有入口，不重复挂）', async () => {
    useOutlineStore.setState({ outlines: [leaf] })
    render(<ChapterPage />)
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leaf } }))
    expect(await screen.findByRole('tab', { name: /第一回 风雪夜归人/ })).toBeTruthy()
    expect(await screen.findByTestId('chapter-editor-stub')).toBeTruthy()
    const tools = await openToolsMenu()
    expect(tools.queryByRole('menuitem', { name: '整章重写' })).toBeNull()
    expect(tools.queryByRole('menuitem', { name: '重写历史' })).toBeNull()
  })
})

// ── 线3 防复发守卫（小说优化批 2）──────────────────────────────────────────
// 坑复训（本轮实测沿用）：imperative Modal 的 DOM 不随 destroy()/cleanup 立即卸载 →
// 断言一律「取最新 .ant-modal-confirm + within()」，并且必须「等弹窗数量变多」而不是
// 等「存在」（旧弹窗残留会让 waitFor 立刻通过、拿到已销毁的那个）；两字按钮被插空格
// （「取 消」）→ 按钮名一律正则。
const confirms = () => Array.from(document.querySelectorAll<HTMLElement>('.ant-modal-confirm'))
const latestConfirm = () => {
  const all = confirms()
  return within(all[all.length - 1])
}

/** 触发后等「确认弹窗数量增加」，返回最新弹窗的限定查询器。 */
async function clickAndAwaitConfirm(trigger: () => void) {
  const before = confirms().length
  trigger()
  await waitFor(() => expect(confirms().length).toBeGreaterThan(before))
  return latestConfirm()
}

/** 简化批：配图/整章重写/重写历史收进「章节工具」菜单——开菜单并取最后挂载的 overlay。 */
async function openToolsMenu() {
  fireEvent.click(await screen.findByRole('button', { name: '章节工具' }))
  const menus = await screen.findAllByRole('menu')
  return within(menus[menus.length - 1])
}

// ── A2b（P1）：重写历史应用/恢复原文不问脏 ──
// 因果：面板里的「应用此版本 / 恢复原文」在服务端写盘后回调 onApplied，旧实现无条件
// loadChapterIntoTab(..., true)——该路径把磁盘正文灌回 scenes 并把 saved 置 true，
// 作者未落盘的手写正文被覆盖**且脏标志被抹掉**（此后再关标签连确认都不弹）。
describe('重写历史（A2b）：打开前过脏闸、应用后脏则不重载', () => {
  const sceneLeaf = { id: 'ch-1', title: '第一回 风雪夜归人', order_index: 1 } as unknown as OutlineNode

  beforeEach(() => {
    localStorage.clear()
    // vi.clearAllMocks 不清实现（含 mockResolvedValueOnce 队列）：「若护栏失效则会用到」
    // 的那条一次性返回值在护栏生效时不会被消费，会泄漏给后续用例 → 本组显式复位。
    novelB.GetChapterScenes.mockReset()
  })
  afterEach(async () => {
    Modal.destroyAll()
    await new Promise((r) => setTimeout(r, 0))
  })

  /** 场景章挂载：等 V4 探测完成（ref 镜像写入）后再派发开章事件（既有用例同款时序） */
  async function mountSceneChapter(first: Array<{ id: string; content: string }>) {
    novelB.IsProjectV4.mockResolvedValue(true)
    novelB.GetChapterScenes.mockResolvedValueOnce(first)
    useOutlineStore.setState({ outlines: [sceneLeaf] })
    useAppStore.setState({ projectPath: 'C:/proj/rewrite-hist' })
    render(<ChapterPage />)
    await waitFor(() => expect(novelB.IsProjectV4).toHaveBeenCalled())
    await act(async () => {})
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: sceneLeaf } }))
    await screen.findByTestId('chapter-editor-stub')
    await screen.findByText('已保存')
  }

  it('打开重写历史前过脏闸：三选弹窗；取消不开面板，「放弃修改」才开', async () => {
    await mountSceneChapter([{ id: '001-s1', content: '磁盘正文。' }])
    fireEvent.click(screen.getByTestId('chapter-editor-type'))
    expect(await screen.findByText('未保存')).toBeTruthy()

    const tools1 = await openToolsMenu()
    const box = await clickAndAwaitConfirm(() => fireEvent.click(tools1.getByRole('menuitem', { name: '重写历史' })))
    // 三选齐备 + 面板没被打开
    expect(box.getByRole('button', { name: /先\s*保\s*存/ })).toBeTruthy()
    expect(box.getByRole('button', { name: /放\s*弃\s*修\s*改/ })).toBeTruthy()
    expect(box.getByRole('button', { name: /取\s*消/ })).toBeTruthy()
    expect(screen.queryByTestId('rw-hist-panel')).toBeNull()

    // ✕/Esc 同义＝取消：点取消不开面板、也不动缓冲（破坏性分支不许挂在 onCancel 上）
    fireEvent.click(box.getByRole('button', { name: /取\s*消/ }))
    await new Promise((r) => setTimeout(r, 0))
    expect(screen.queryByTestId('rw-hist-panel')).toBeNull()
    expect(screen.getByText('未保存')).toBeTruthy()

    // 「放弃修改」＝继续打开；本地缓冲随后仍由 onApplied 的脏判保留
    const tools2 = await openToolsMenu()
    const box2 = await clickAndAwaitConfirm(() => fireEvent.click(tools2.getByRole('menuitem', { name: '重写历史' })))
    fireEvent.click(box2.getByRole('button', { name: /放\s*弃\s*修\s*改/ }))
    expect(await screen.findByTestId('rw-hist-panel')).toBeTruthy()
  })

  it('应用版本后本地有未保存正文：不重载、保留本地版本、脏标志保留', async () => {
    const warn = vi.spyOn(message, 'warning')
    await mountSceneChapter([{ id: '001-s1', content: '磁盘上的旧正文。' }])
    // 若护栏失效而发生重载，服务端返回的是这段新正文——用它把「被覆盖」钉成可诊断的红
    novelB.GetChapterScenes.mockResolvedValueOnce([{ id: '001-s1', content: '重写后落盘的新正文。' }])

    // 打开面板时缓冲是干净的（所以不弹脏闸）；打开期间该章标签转脏（本轮由编辑器桩构造，
    // 真实来源包括面板打开时仍在途的场景写入 / 程序化更新）——onApplied 必须再判一次。
    fireEvent.click((await openToolsMenu()).getByRole('menuitem', { name: '重写历史' }))
    expect(await screen.findByTestId('rw-hist-panel')).toBeTruthy()
    fireEvent.click(screen.getByTestId('chapter-editor-type'))
    expect(await screen.findByText('未保存')).toBeTruthy()

    fireEvent.click(screen.getByTestId('rw-hist-apply'))

    await waitFor(() => expect(warn).toHaveBeenCalledWith(
      '重写已在服务端生效；当前标签有未保存的修改，已保留你的本地版本，未刷新'))
    // 不重载：只有首次载入那一次 GetChapterScenes
    expect(novelB.GetChapterScenes).toHaveBeenCalledTimes(1)
    // 本地版本仍在、脏标志仍在（关标签仍会弹未保存确认）
    expect(screen.getByTestId('chapter-editor-scenes').textContent).toContain('作者手写的新段落。')
    expect(screen.getByText('未保存')).toBeTruthy()
    expect(dirtyNovelProviders().some((p) => p.id === 'chapter-tabs')).toBe(true)
    warn.mockRestore()
  })
})

// ── A10（P3）：载入期间输入被覆盖后还标 saved=true ──
// 因果：先塞空 tab 让编辑器立即可输入，载入完成无条件覆盖 scenes 并置 saved=true。
// 采用「载入在途渲染只读 loading 占位而不渲染编辑区」：从根上不存在可输入窗口，
// 比「完成时检测已输入并保留」更不易漏（后者仍要面对输入与回填的交错时序）。
describe('章节载入在途（A10）：编辑区只读 loading，不静默覆盖输入', () => {
  const leafA = { id: 'ch-1', title: '第一回 风雪夜归人', order_index: 1 } as unknown as OutlineNode

  beforeEach(() => {
    localStorage.clear()
    // 同 A2b：清掉可能从上一组泄漏的一次性实现（本组自身用 mockResolvedValueOnce 排程）
    novelB.GetChapterScenes.mockReset()
  })
  afterEach(async () => {
    Modal.destroyAll()
    await new Promise((r) => setTimeout(r, 0))
  })

  it('载入未完成时只有 loading 占位（无任何输入面），完成后才渲染编辑区', async () => {
    const resolves: Array<(v: unknown) => void> = []
    bindingsBridge.GetChapter.mockImplementationOnce(
      () => new Promise((r) => { resolves.push(r as (v: unknown) => void) }) as never)
    // 不设 projectPath：projectPath effect 的「恢复上次阅读章」是在途异步，若与本次
    // 开章并发会 setTabs 覆盖本次标签（本用例只关心载入在途的编辑区形态）。
    useOutlineStore.setState({ outlines: [leafA] })
    render(<ChapterPage />)
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leafA } }))
    await waitFor(() => expect(bindingsBridge.GetChapter).toHaveBeenCalledWith(1))
    await waitFor(() => expect(resolves.length).toBe(1))

    // 载入在途：编辑区不存在（连文本框都没有）→「敲字后被服务端正文覆盖」不可达
    expect(await screen.findByTestId('chapter-editor-loading')).toBeTruthy()
    expect(screen.queryByTestId('chapter-editor-stub')).toBeNull()
    expect(screen.queryByTestId('chapter-editor-type')).toBeNull()
    expect(screen.getByText('正在载入第 1 章正文…')).toBeTruthy()

    await act(async () => { resolves[0]({ content: '服务端最终正文。' }) })
    expect(await screen.findByTestId('chapter-editor-stub')).toBeTruthy()
    expect(screen.queryByTestId('chapter-editor-loading')).toBeNull()
    expect(screen.getByTestId('chapter-editor-scenes').textContent).toBe('服务端最终正文。')
    expect(await screen.findByText('已保存')).toBeTruthy()
  })

  it('脏闸按「面板当前操作的那一章」判：别的章的未保存标签不拦本章', async () => {
    const leafB = { id: 'ch-2', title: '第二回 灯下白头人', order_index: 2 } as unknown as OutlineNode
    novelB.IsProjectV4.mockResolvedValue(true)
    novelB.GetChapterScenes
      .mockResolvedValueOnce([{ id: '001-s1', content: '第一章场景正文。' }])
      .mockResolvedValueOnce([{ id: '002-s1', content: '第二章场景正文。' }])
    useOutlineStore.setState({ outlines: [leafA, leafB] })
    useAppStore.setState({ projectPath: 'C:/proj/rewrite-hist-multi' })
    render(<ChapterPage />)
    await waitFor(() => expect(novelB.IsProjectV4).toHaveBeenCalled())
    await act(async () => {})

    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leafA } }))
    await waitFor(() => expect(screen.getByTestId('chapter-editor-scenes').textContent).toBe('第一章场景正文。'))
    // 在第 2 章标签上制造未保存修改（多标签缓冲：脏的可能不是当前标签）
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leafB } }))
    await waitFor(() => expect(screen.getByTestId('chapter-editor-scenes').textContent).toBe('第二章场景正文。'))
    fireEvent.click(screen.getByTestId('chapter-editor-type'))
    expect(screen.getByTestId('chapter-editor-scenes').textContent).toContain('作者手写的新段落。')
    // 切回干净的第 1 章：第 2 章的脏仍在缓冲里
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leafA } }))
    await waitFor(() => expect(screen.getByTestId('chapter-editor-scenes').textContent).toBe('第一章场景正文。'))

    // 面板操作的是第 1 章 → 不该拿第 2 章的脏拦本章（判脏按章号，不看「缓冲里有没有脏」）
    fireEvent.click((await openToolsMenu()).getByRole('menuitem', { name: '重写历史' }))
    expect(await screen.findByTestId('rw-hist-panel')).toBeTruthy()
    await new Promise((r) => setTimeout(r, 0))
    expect(confirms().length).toBe(0)
    expect(screen.getByTestId('rw-hist-panel').getAttribute('data-chapter')).toBe('1')
  })
})

// ── A9（P2）：摘要 / 问书迟到响应错章 ──
// 因果：runSummary/runAsk 在 await 后直接回填，没有任何「还属于当前章吗」的守卫；
// 切章 effect 先把 summaryText 置 null，随后被旧响应回填 → toggleSummary 见 summaryText
// 非空就不再发请求，上一章的摘要长期显示在新章；问书答案同型（串进新章会话）。
describe('AI 伴读迟到响应（A9）：切章后旧摘要/旧答案不回填', () => {
  const leafA = { id: 'ch-1', title: '第一回 风雪夜归人', order_index: 1 } as unknown as OutlineNode
  const leafB = { id: 'ch-2', title: '第二回 灯下白头人', order_index: 2 } as unknown as OutlineNode

  // jsdom 30 未实现 Range.getBoundingClientRect，而 readSelectionInRoot 读选区几何时调用
  // （划词工具条/问书共用判定）→ 补零矩形，仅本文件需要真实划词路径。
  beforeAll(() => {
    Range.prototype.getBoundingClientRect = () => ({
      x: 0, y: 0, top: 0, left: 0, right: 0, bottom: 0, width: 0, height: 0,
      toJSON: () => ({}),
    }) as DOMRect
  })

  beforeEach(() => {
    localStorage.clear()
    // vi.clearAllMocks 不清实现：显式复位，避免「在途」用例的一次性实现泄漏到后续用例
    bindingsBridge.NovelReadingAsk.mockReset()
  })
  afterEach(async () => {
    Modal.destroyAll()
    await new Promise((r) => setTimeout(r, 0))
  })

  /** 开第 1 章并进入阅读模式（以正文段落渲染完成为准） */
  async function openReading() {
    useOutlineStore.setState({ outlines: [leafA, leafB] })
    render(<ChapterPage />)
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leafA } }))
    await screen.findByTestId('chapter-editor-stub')
    await screen.findByText('已保存')
    fireEvent.click(screen.getByRole('button', { name: '进入阅读模式' }))
    await screen.findByText('夜色沉沉，雨落在窗台上。')
  }

  it('切章后迟到的摘要被丢弃：不回填新章，也不吞掉新章的摘要请求', async () => {
    const resolves: Array<(v: string) => void> = []
    bindingsBridge.NovelReadingAsk
      .mockImplementationOnce(() => new Promise<string>((r) => { resolves.push(r) }) as never)
      .mockImplementationOnce(() => new Promise<string>((r) => { resolves.push(r) }) as never)
    await openReading()

    fireEvent.click(screen.getByRole('button', { name: /AI 摘要/ }))
    await waitFor(() => expect(resolves.length).toBe(1))
    expect(await screen.findByText('AI 正在阅读本章…')).toBeTruthy()

    // 切到第 2 章：旧摘要请求仍在途
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leafB } }))
    await waitFor(() => expect(document.querySelector('.novel-reading-title')?.textContent)
      .toBe('第二回 灯下白头人'))
    // 切章即收尾 loading：新章不许停在「AI 正在阅读本章…」
    await waitFor(() => expect(screen.queryByText('AI 正在阅读本章…')).toBeNull())

    // 旧响应到达：整体丢弃（回填会让新章显示上一章的摘要）
    await act(async () => { resolves[0]('第一章的摘要') })
    expect(screen.queryByText('第一章的摘要')).toBeNull()

    // 反向守卫核：旧摘要若被回填，summaryText 非空 → toggleSummary 不再发请求
    fireEvent.click(screen.getByRole('button', { name: /AI 摘要/ }))
    await waitFor(() => expect(bindingsBridge.NovelReadingAsk).toHaveBeenCalledTimes(2))
    expect(screen.queryByText('第一章的摘要')).toBeNull()
    await act(async () => { resolves[1]('第二章的摘要') })
    expect(await screen.findByText('第二章的摘要')).toBeTruthy()
  })

  it('切章后迟到的问书答案被丢弃，且不在新章留下「正在思考」', async () => {
    const resolves: Array<(v: string) => void> = []
    bindingsBridge.NovelReadingAsk
      .mockImplementationOnce(() => new Promise<string>((r) => { resolves.push(r) }) as never)
    await openReading()

    // 划词 → 浮动工具条 → 问书弹窗
    const para = document.querySelector('.novel-reading-p') as HTMLElement
    const range = document.createRange()
    range.setStart(para.firstChild as Text, 0)
    range.setEnd(para.firstChild as Text, 4)
    const sel = window.getSelection() as Selection
    sel.removeAllRanges(); sel.addRange(range)
    fireEvent.mouseUp(document.querySelector('.novel-reading-scroll') as HTMLElement)
    fireEvent.click(screen.getByRole('button', { name: /问书/ }))

    fireEvent.change(screen.getByPlaceholderText('针对摘选内容提问，例如：这句话暗示了什么？'),
      { target: { value: '这句话什么意思？' } })
    // antd 两字按钮插空格（「提 问」）→ 按钮名一律正则
    fireEvent.click(screen.getByRole('button', { name: /提\s*问/ }))
    await waitFor(() => expect(resolves.length).toBe(1))
    expect(await screen.findByText('正在思考…')).toBeTruthy()

    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leafB } }))
    await waitFor(() => expect(document.querySelector('.novel-reading-title')?.textContent)
      .toBe('第二回 灯下白头人'))
    // loading 必须收尾：旧响应被守卫拦下后不会再走到 finally
    await waitFor(() => expect(
      document.querySelector('.novel-read-ask-actions .ant-btn-loading')).toBeNull())

    await act(async () => { resolves[0]('第一轮的答案') })
    expect(screen.queryByText('第一轮的答案')).toBeNull()
    expect(document.querySelectorAll('.novel-read-ask-msg').length).toBe(0)
  })
})

// 「恢复上次阅读章」也是**迟到的异步回填**：projectPath effect 里先 await IsProjectV4 /
// loadOutlines，之后才 `setTabs([createTabData(恢复章)])`——那是**整体替换**缓冲。若作者在
// 这段窗口里已经从目录点了章（或敲了字），照写就把刚打开的标签连同内容换掉。本组把
// 「已有标签＝用户意图优先，恢复动作放弃」这条护栏钉死。
// 灵敏度：把 `if (tabsRef.current.length > 0) return` 去掉，GetChapter 会被调用两次（本用例红）。
describe('切书恢复上次阅读章（迟到回填不覆盖已打开标签）', () => {
  it('恢复分支在途时作者已开章：不整体替换缓冲、不重复载入', async () => {
    const PATH = 'C:/novel/race'
    let releaseV4: (v: boolean) => void = () => { /* 由 render 后的 effect 赋值 */ }
    novelB.IsProjectV4.mockImplementation(
      () => new Promise<boolean>((resolve) => { releaseV4 = resolve }) as never,
    )
    useOutlineStore.setState({ outlines: [leaf] })
    writeReadingProgress(PATH, { nodeId: leaf.id, chapterNum: 1, title: '第一回 风雪夜归人' })
    useAppStore.setState({ projectPath: PATH })

    render(<ChapterPage />)
    // 恢复分支此刻卡在 IsProjectV4 上（未 resolve）；作者从目录点章
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node: leaf } }))
    await screen.findByRole('tab', { name: /第一回 风雪夜归人/ })
    await waitFor(() => expect(bindingsBridge.GetChapter).toHaveBeenCalledTimes(1))

    // 放行恢复分支：它现在读到的进度就是刚打开的这一章；无护栏则再整体替换 + 二次载入
    await act(async () => { releaseV4(false) })
    await new Promise((r) => setTimeout(r, 0))
    expect(screen.getByRole('tab', { name: /第一回 风雪夜归人/ })).toBeTruthy()
    expect(bindingsBridge.GetChapter).toHaveBeenCalledTimes(1)
  })
})
