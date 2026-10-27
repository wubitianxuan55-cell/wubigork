// NovelPage.test.tsx — 常驻 pane 的 active 下发（规格线2「常驻页快捷键门控」防复发守卫）。
//
// 背景（读码确证）：NovelPage 把五个小说子页**一次全挂**，只靠
// `.novel-tab-pane{display:none}` 隐藏——隐藏 ≠ 卸载。于是子页的窗口级副作用
// （F11 / Ctrl+S / Ctrl+N、novel:open-chapter）只能靠 NovelPage 下发的 active 门控，
// 本文件锁两条不变量：
//   ①当前页 active=true、其余常驻隐藏页 active=false（且恰好一个 true）；
//   ②点目录树时 novel:open-chapter 必须在阅读 pane **已成为当前页之后**才派发
//     （同 tick 同步派发时隐藏页拿到的 active 还是 false，会被门控丢弃 → 点章失效）。
//
// 2026-10-27 起第六 pane = 原罪页（导航级合并收编，lazy import）——**惰性挂载**：
// 首次激活才挂（独立数据域零请求）、挂载过即常驻（切走不卸载）。
//
// 2026-10-27 大道至简重组：角色页并入设定——外壳 6→5 页签，'novelsetting' 的
// component 换成真 SettingHub（内部再挂 mock 的世界观/人物两子 pane）。DOM 里
// pane-* 计数 = 外壳 4（home/create/chapter+惰性 sin）+ 设定 hub 内 2 子 pane。
//
// 手法：子页替换为只暴露 `data-active` 的桩（线2 允许的两种手法之一），
// 大纲树 OutlinePanel 替换为「点一下即 onSelectNode」的按钮。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'

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
vi.mock('./OriginalSinPage', () => ({
  default: ({ active }: { active?: boolean }) => <div data-testid="pane-sin" data-active={String(active)} />,
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
import { notifyBoardActive, resetBoardActive } from '../lib/boardActive'
import type { OutlineNode } from '../types'

/** 最小消费面的大纲叶子（目录树点击只用到 id/title/order_index） */
const leaf = { id: 'ch-1', title: '第一回 风雪夜归人', order_index: 1 } as unknown as OutlineNode

/** 当前「可见」的 pane（data-active=true）——不变量：恰好一个 */
const activePanes = () =>
  Array.from(document.querySelectorAll<HTMLElement>('[data-testid^="pane-"][data-active="true"]'))
    .map((el) => el.getAttribute('data-testid'))

const paneActive = (key: string) =>
  screen.getByTestId(`pane-${key}`).getAttribute('data-active')

const PANE_KEYS = ['home', 'novelsetting', 'character', 'create', 'chapter', 'sin']

describe('NovelPage 常驻 pane 的 active 下发', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // 板块级可见信号是模块级状态：置回「未通知」（= 可见），避免用例间串味
    resetBoardActive()
    // activeTab 从 localStorage 读（gaea.novel.activeTab），清掉保证默认书架页
    localStorage.clear()
    // 不开书：projectPath 空 → 不触发 loadOutlines/GetStats，用例只测 active 下发
    useAppStore.setState({ projectPath: '', projectTitle: '' })
    useOutlineStore.setState({ outlines: [], loading: false, error: null })
  })

  it('默认书架 tab：home=true、其余常驻隐藏页=false，且恰好一个可见；sin 惰性未挂载', async () => {
    render(<NovelPage />)
    await screen.findByTestId('pane-home')

    expect(paneActive('home')).toBe('true')
    for (const key of PANE_KEYS.filter((k) => k !== 'home' && k !== 'sin')) {
      expect(paneActive(key)).toBe('false')
    }
    // 外壳 4 pane 一次全挂（home/create/chapter + 设定 hub；隐藏 ≠ 卸载），设定 hub
    // 内部再挂世界观/人物两子 pane——DOM pane-* 计数 = 3 + 2 = 5；
    // sin pane 惰性：未进过原罪页签就不挂载（独立数据域零请求）
    expect(document.querySelectorAll('[data-testid^="pane-"]').length).toBe(5)
    expect(screen.queryByTestId('pane-sin')).toBeNull()
    expect(activePanes()).toEqual(['pane-home'])
  })

  it('设定页签：hub 转发 active——世界观子 pane true、人物子 pane false（Ctrl+S 门控链）', async () => {
    render(<NovelPage />)
    await screen.findByTestId('pane-home')
    fireEvent.click(within(screen.getByRole('navigation', { name: '小说板块' }))
      .getByRole('button', { name: /设定/ }))
    await waitFor(() => expect(paneActive('novelsetting')).toBe('true'))
    expect(paneActive('character')).toBe('false')
    expect(activePanes()).toEqual(['pane-novelsetting'])
  })

  it('旧值迁移：activeTab 残留已退役的 character → 落设定页签并直达人物视图', async () => {
    localStorage.setItem('gaea.novel.activeTab', 'character')
    render(<NovelPage />)
    await screen.findByTestId('pane-character')
    // 迁移即写 settingView=chars：设定 hub 直达人物视图（人物子 pane active，
    // 世界观子 pane 隐藏）——用户上次在角色页，本次不丢语境
    expect(paneActive('character')).toBe('true')
    expect(paneActive('novelsetting')).toBe('false')
    expect(localStorage.getItem('gaea.novel.settingView')).toBe('chars')
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

  // ── 板块级门控（lib/boardActive，v4.425 补的「板块级 keepAlive」收口）──────────
  // 反例（v4.421 只做到子页之间）：壳层对访问过的板块做 keepAlive（display:none），
  // 小说板块被切走后本组件仍挂载，只按 activeTab 判断的话，隐藏的阅读页照样
  // preventDefault 吞掉 Ctrl+S / F11——本条把「板块不可见 ⇒ 全部 pane 非当前页」钉死。
  // 灵敏度：把 NovelPage 里的 `&& boardVisible` 去掉，本条必红。
  it('板块被切走：全部 pane active=false（连带子页窗口级副作用一起停）', async () => {
    render(<NovelPage />)
    await screen.findByTestId('pane-home')

    // 切到阅读 tab（最危险的一页：Ctrl+S/F11/←→ 都在它身上）
    fireEvent.click(within(screen.getByRole('navigation', { name: '小说板块' }))
      .getByRole('button', { name: /阅读/ }))
    await waitFor(() => expect(paneActive('chapter')).toBe('true'))

    // 壳层切到别的板块（如办公）→ 小说板块整体进入后台保活
    act(() => { notifyBoardActive('gaea') })
    expect(paneActive('chapter')).toBe('false')
    expect(activePanes()).toEqual([])

    // 切回小说板块 → 当前 tab（阅读）恢复 active
    act(() => { notifyBoardActive('novel') })
    expect(paneActive('chapter')).toBe('true')
    expect(activePanes()).toEqual(['pane-chapter'])
  })

  // ── 第六 pane：原罪（2026-10-27 导航级合并收编；惰性挂载=首次激活才挂）──────
  it('原罪页签：首次点击才挂载 pane-sin 并独占 active，身份区切原罪语境、封面按钮退场、切走常驻不卸载', async () => {
    render(<NovelPage />)
    await screen.findByTestId('pane-home')
    // 惰性：未点过原罪页签，pane 不存在
    expect(screen.queryByTestId('pane-sin')).toBeNull()

    fireEvent.click(within(screen.getByRole('navigation', { name: '小说板块' }))
      .getByRole('button', { name: /原罪/ }))

    await waitFor(() => expect(paneActive('sin')).toBe('true'))
    expect(activePanes()).toEqual(['pane-sin'])
    // data-novel-tab='sin' 是 novel-workspace.css 收敛项目级目录树/检查器/AI 控制台
    // FAB 的选择器锚点——丢了它 sin 页签会挤在双 zone 之间
    expect(document.querySelector('.novel-hub')?.getAttribute('data-novel-tab')).toBe('sin')
    // 身份区切原罪语境：kicker 不再是书房，书名/项目统计退场
    expect(document.querySelector('.novel-atelier-kicker')?.textContent).toBe('原罪')
    expect(screen.getByText('图文故事')).toBeTruthy()
    expect(screen.queryByText('书房')).toBeNull()
    expect(screen.queryByTitle('为当前项目生成书封（3:4 竖版）')).toBeNull()
    // activeTab 持久化（回访落在原罪页签）
    expect(localStorage.getItem('gaea.novel.activeTab')).toBe('sin')

    // 挂载过即常驻：切回书架只隐藏不卸载（active=false 但 pane 仍在 DOM）
    fireEvent.click(within(screen.getByRole('navigation', { name: '小说板块' }))
      .getByRole('button', { name: /书架/ }))
    await waitFor(() => expect(paneActive('home')).toBe('true'))
    expect(paneActive('sin')).toBe('false')
    expect(screen.getByTestId('pane-sin')).toBeTruthy()
  })

  it('回访恢复：activeTab 持久为 sin 时，NovelPage 挂载即直挂 sin pane（不闪空白）', async () => {
    localStorage.setItem('gaea.novel.activeTab', 'sin')
    render(<NovelPage />)
    await waitFor(() => expect(paneActive('sin')).toBe('true'))
    expect(activePanes()).toEqual(['pane-sin'])
  })
})
