// ChapterPage.test.tsx — 章节页组件冒烟测试（ChapterPage 拆分补测第一步）。
// 手法沿用 SettingsPage.test / NovelSettingPage.test：vi.mock 屏蔽 wails 绑定、
// 重型子组件（TTS/编辑器/导出/配图）替换为桩，zustand store 用真实实例 + setState 注入。
// 只锁：组件可渲染、章节 Tab 出现、阅读模式正文渲染，全程不抛错——不追求深覆盖。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go，章节读写全部给确定性返回。
vi.mock('../../src/wailsjsCompat', () => ({
  GetChapter: vi.fn().mockResolvedValue({ content: '夜色沉沉，雨落在窗台上。\n\n他推门而入，灯还亮着。' }),
  GetChapterBranch: vi.fn().mockResolvedValue({ content: '' }),
  SaveChapterContent: vi.fn().mockResolvedValue(undefined),
  SaveChapterBranchContent: vi.fn().mockResolvedValue(undefined),
}))

// 重型子组件桩：冒烟只关心 ChapterPage 自身的装配与状态流转
vi.mock('../components/TTSPlayer', () => ({ default: () => <div data-testid="tts-player-stub" /> }))
vi.mock('../components/novel/ChapterEditor', () => ({ default: () => <div data-testid="chapter-editor-stub" /> }))
vi.mock('../components/novel/ExportPanel', () => ({ default: () => <div /> }))
vi.mock('./chapter/ChapterIllustration', () => ({ default: () => <div /> }))

import ChapterPage from './ChapterPage'
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
})
