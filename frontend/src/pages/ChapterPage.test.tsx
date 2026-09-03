// ChapterPage.test.tsx — 章节页组件冒烟测试（ChapterPage 拆分补测第一步）。
// 手法沿用 SettingsPage.test / NovelSettingPage.test：vi.mock 屏蔽 wails 绑定、
// 重型子组件（TTS/编辑器/导出/配图）替换为桩，zustand store 用真实实例 + setState 注入。
// 只锁：组件可渲染、章节 Tab 出现、阅读模式正文渲染，全程不抛错——不追求深覆盖。
// 另锁搜索定位接线（第三批）：阅读模式内点命中可定位、同章再点仍能重新定位（回归修复）。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go，章节读写全部给确定性返回。
vi.mock('../../src/wailsjsCompat', () => ({
  GetChapter: vi.fn().mockResolvedValue({ content: '夜色沉沉，雨落在窗台上。\n\n他推门而入，灯还亮着。' }),
  GetChapterBranch: vi.fn().mockResolvedValue({ content: '' }),
  SaveChapterContent: vi.fn().mockResolvedValue(undefined),
  SaveChapterBranchContent: vi.fn().mockResolvedValue(undefined),
  NovelSearch: vi.fn(),
}))

// 重型子组件桩：冒烟只关心 ChapterPage 自身的装配与状态流转
vi.mock('../components/TTSPlayer', () => ({ default: () => <div data-testid="tts-player-stub" /> }))
vi.mock('../components/novel/ChapterEditor', () => ({ default: () => <div data-testid="chapter-editor-stub" /> }))
vi.mock('../components/novel/ExportPanel', () => ({ default: () => <div /> }))
vi.mock('./chapter/ChapterIllustration', () => ({ default: () => <div /> }))

import ChapterPage from './ChapterPage'
import { NovelSearch } from '../../src/wailsjsCompat'
import { useAppStore } from '../stores/appStore'
import { useOutlineStore } from '../stores/outlineStore'
import type { OutlineNode } from '../types'

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
