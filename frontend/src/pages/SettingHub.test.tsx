/**
 * SettingHub.test.tsx — 「设定」页签薄壳契约（2026-10-27 大道至简重组：
 * 角色页并入设定，页签 6→5）。锁三件事：
 * 1. 双 pane **常驻挂载**（隐藏 ≠ 卸载——CharacterPage 数据不因切视图丢失）
 * 2. `active` 转发：仅当前可见子 pane 收到 true（世界观编辑器的 Ctrl+S 门控语义）
 * 3. 子视图选择持久化（gaea.novel.settingView）
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('./NovelSettingPage', () => ({
  default: ({ active }: { active?: boolean }) => <div data-testid="page-setting" data-active={active ? '1' : '0'} />,
}))
vi.mock('./CharacterPage', () => ({
  default: ({ active }: { active?: boolean }) => <div data-testid="page-character" data-active={active ? '1' : '0'} />,
}))

import SettingHub from './SettingHub'

const VIEW_KEY = 'gaea.novel.settingView'

describe('SettingHub · 设定页签薄壳（角色并入）', () => {
  afterEach(() => {
    try { localStorage.removeItem(VIEW_KEY) } catch { /* ignore */ }
  })

  it('双 pane 常驻挂载：默认世界观视图，人物 pane 隐藏但仍在 DOM', async () => {
    render(<SettingHub />)
    expect(await screen.findByTestId('page-setting')).toBeTruthy()
    expect(screen.getByTestId('page-character')).toBeTruthy() // 隐藏 ≠ 卸载
    // active 转发：仅当前可见子页为 true
    expect(screen.getByTestId('page-setting').getAttribute('data-active')).toBe('1')
    expect(screen.getByTestId('page-character').getAttribute('data-active')).toBe('0')
    // pane 容器 display 语义（结构断言）
    const panes = Array.from(document.querySelectorAll('.novel-setting-hub-pane'))
    expect(panes[0].classList.contains('is-active')).toBe(true)
    expect(panes[1].classList.contains('is-active')).toBe(false)
  })

  it('切「人物」：两个 pane 都在 DOM，active 与 is-active 翻转，选择落 localStorage', async () => {
    render(<SettingHub />)
    await screen.findByTestId('page-setting')
    fireEvent.click(screen.getByText('人物'))
    await waitFor(() => {
      expect(screen.getByTestId('page-character').getAttribute('data-active')).toBe('1')
      expect(screen.getByTestId('page-setting').getAttribute('data-active')).toBe('0')
    })
    const panes = Array.from(document.querySelectorAll('.novel-setting-hub-pane'))
    expect(panes[0].classList.contains('is-active')).toBe(false)
    expect(panes[1].classList.contains('is-active')).toBe(true)
    expect(localStorage.getItem(VIEW_KEY)).toBe('chars')
  })

  it('记忆视图：localStorage=chars 时直达人物视图', async () => {
    localStorage.setItem(VIEW_KEY, 'chars')
    render(<SettingHub />)
    await waitFor(() => {
      expect(screen.getByTestId('page-character').getAttribute('data-active')).toBe('1')
    })
  })
})
