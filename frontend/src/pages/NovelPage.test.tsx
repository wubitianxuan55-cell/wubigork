// NovelPage.test.tsx — 常驻五 pane 的 active 下发（规格线2「常驻页快捷键门控」防复发守卫）。
//
// 背景（读码确证）：NovelPage 把五个子页**一次全挂**，只靠
// `.novel-tab-pane{display:none}` 隐藏——隐藏 ≠ 卸载。于是子页的窗口级副作用
// （F11 / Ctrl+S / Ctrl+N、novel:open-chapter）只能靠 NovelPage 下发的 active 门控，
// 本文件锁两条不变量：
//   ①当前页 active=true、其余四个常驻隐藏页 active=false（且恰好一个 true）；
//   ②点目录树时 novel:open-chapter 必须在阅读 pane **已成为当前页之后**才派发
//     （同 tick 同步派发时隐藏页拿到的 active 还是 false，会被门控丢弃 → 点章失效）。
//
// 手法：五个子页替换为只暴露 `data-active` 的桩（线2 允许的两种手法之一），
// 大纲树 OutlinePanel 替换为「点一下即 onSelectNode」的按钮。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'

vi.mock('./HomePage', () => ({
  default: ({ active }: { active?: boolean }) => <div data-testid="pane-home" data-active={String(active)} />,
}))
vi.mock('./NovelSettingPage', () => ({
  default: ({ active }: { active?: boolean }) => <div data-testid="pane-novelsetting" data-active={String(active)} />,
}))
vi.mock('./CharacterPage', () => ({
  default: ({ active }: { active?: boolean }) => <div data-testid="pane-character" data-active={String(active)} />,
}))
vi.mock('./CreatePage', () => ({
  default: ({ active }: { active?: boolean }) => <div data-testid="pane-create" data-active={String(active)} />,
}))
vi.mock('./ChapterPage', () => ({
  default: ({ active }: { active?: boolean }) => <div data-testid="pane-chapter" data-active={String(active)} />,
}))

// 壳层左 zone 的目录树：真实 OutlinePanel 依赖大纲数据，这里只保留「点击 → onSelectNode」。
// 注意：.novel-side-zone 在书架/设定/角色/创作 tab 上是 display:none
// （novel-workspace.css:272-291），只有阅读 tab 才可见——所以第 3 例用 testid 直接点
// （fireEvent 不做可见性判定），刻意复现「pane 还没 active 就被要求开章」这条时序。
vi.mock('../components/novel/OutlinePanel', () => ({
  default: ({ onSelectNode }: { onSelectNode: (node: unknown) => void }) => (
    <button
      type="button"
      data-testid="outline-node"
      onClick={() => onSelectNode({ id: 'ch-1', title: '第一回 风雪夜归人', order_index: 1 })}
    >
      目录节点
    </button>
  ),
}))

import NovelPage from './NovelPage'
import { useAppStore } from '../stores/appStore'
import { useOutlineStore } from '../stores/outlineStore'
import type { OutlineNode } from '../types'

/** 最小消费面的大纲叶子（目录树点击只用到 id/title/order_index） */
const leaf = { id: 'ch-1', title: '第一回 风雪夜归人', order_index: 1 } as unknown as OutlineNode

/** 当前「可见」的 pane（data-active=true）——不变量：恰好一个 */
const activePanes = () =>
  Array.from(document.querySelectorAll<HTMLElement>('[data-testid^="pane-"][data-active="true"]'))
    .map((el) => el.getAttribute('data-testid'))

const paneActive = (key: string) =>
  screen.getByTestId(`pane-${key}`).getAttribute('data-active')

const PANE_KEYS = ['home', 'novelsetting', 'character', 'create', 'chapter']

describe('NovelPage 常驻 pane 的 active 下发', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // activeTab 从 localStorage 读（gaea.novel.activeTab），清掉保证默认书架页
    localStorage.clear()
    // 不开书：projectPath 空 → 不触发 loadOutlines/GetStats，用例只测 active 下发
    useAppStore.setState({ projectPath: '', projectTitle: '' })
    useOutlineStore.setState({ outlines: [], loading: false, error: null })
  })

  it('默认书架 tab：home=true、其余四个常驻隐藏页=false，且恰好一个可见', async () => {
    render(<NovelPage />)
    await screen.findByTestId('pane-home')

    expect(paneActive('home')).toBe('true')
    for (const key of PANE_KEYS.filter((k) => k !== 'home')) {
      expect(paneActive(key)).toBe('false')
    }
    // 五 pane 全部挂载（常驻），只有 active 不同——这正是「隐藏 ≠ 卸载」的契约
    expect(document.querySelectorAll('[data-testid^="pane-"]').length).toBe(5)
    expect(activePanes()).toEqual(['pane-home'])
  })

  it('切 tab：active 跟着切，隐藏页一律 false', async () => {
    render(<NovelPage />)
    await screen.findByTestId('pane-home')

    fireEvent.click(within(screen.getByRole('navigation', { name: '小说板块' }))
      .getByRole('button', { name: /阅读/ }))

    await waitFor(() => expect(paneActive('chapter')).toBe('true'))
    expect(paneActive('home')).toBe('false')
    expect(activePanes()).toEqual(['pane-chapter'])
  })

  it('目录点章：novel:open-chapter 在阅读 pane 已 active 之后才派发（否则被门控丢弃）', async () => {
    useOutlineStore.setState({ outlines: [leaf] })

    // 用独立的 window 监听记录「事件到达时阅读 pane 的 DOM 状态」——若 NovelPage
    // 仍在切 tab 的同一 tick 同步派发，此刻读到的 data-active 还是 false。
    const seen: Array<string | null> = []
    const listener = () => {
      seen.push(document.querySelector('[data-testid="pane-chapter"]')?.getAttribute('data-active') ?? null)
    }
    window.addEventListener('novel:open-chapter', listener)
    try {
      render(<NovelPage />)
      await screen.findByTestId('pane-home')
      // 起点：书架 tab，阅读 pane 是隐藏常驻页（active=false）
      expect(paneActive('chapter')).toBe('false')

      fireEvent.click(screen.getByTestId('outline-node'))

      await waitFor(() => expect(seen.length).toBe(1))
      expect(seen[0]).toBe('true')
      expect(paneActive('chapter')).toBe('true')
    } finally {
      window.removeEventListener('novel:open-chapter', listener)
    }
  })
})
