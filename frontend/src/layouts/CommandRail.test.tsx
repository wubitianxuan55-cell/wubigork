import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import { CommandRail } from './MainLayout'
import { LocaleProvider } from '../gaea/lib/i18n'

// CommandRail 双空间分域导航（v4.169）：切换器（SHELL_SPACES 驱动）+
// 主体按当前空间过滤（shared 恒在 / independent 剔除 / inMenu 过滤）+
// 编程（independent）仅 foot 单列 ⇒ rail 全量 code 入口 = 1。
// LocaleProvider 钉 zh（localStorage gaea-lang）让切换器 label 走字典中文文案。
const wrap = (ui: ReactElement) => <LocaleProvider>{ui}</LocaleProvider>

function renderRail(props: Partial<Parameters<typeof CommandRail>[0]> = {}) {
  const onNavigate = vi.fn()
  const onSwitchSpace = vi.fn()
  const utils = render(wrap(
    <CommandRail
      page="home"
      space="work"
      darkMode={false}
      toggleDarkMode={vi.fn()}
      onNavigate={onNavigate}
      onSwitchSpace={onSwitchSpace}
      {...props}
    />,
  ))
  return { ...utils, onNavigate, onSwitchSpace }
}

/** rail 主体（menubar）内按钮的 aria-label 序 */
function navLabels(): (string | null)[] {
  const nav = screen.getByRole('menubar')
  return Array.from(nav.querySelectorAll('button')).map((b) => b.getAttribute('aria-label'))
}

beforeEach(() => {
  localStorage.setItem('gaea-lang', 'zh')
})

describe('CommandRail 书斋/闲庭双空间分域导航（v4.169）', () => {
  it('rail 顶部渲染当前空间指示徽标（v4.182 切换器迁首页顶栏：非交互、label 走字典）', () => {
    renderRail()
    const ind = screen.getByTestId('v3-rail-space-indicator')
    expect(ind.textContent).toBe('书斋')
    expect(ind.getAttribute('aria-pressed')).toBeNull()
    // 切换器移除：rail 不再渲染空间切换按钮
    expect(screen.queryByTestId('v3-rail-space-switch')).toBeNull()
    expect(screen.queryByTestId('v3-rail-space-play')).toBeNull()
  })

  it('work 空间：主体 = 共享 + 书斋板块（首页/办公/造价数据库/记忆中枢/模型中心/青鸟）', () => {
    renderRail()
    expect(navLabels()).toEqual(['首页', '办公', '造价数据库', '记忆中枢', '模型中心', '青鸟'])
  })

  it('编程（independent）仅 foot 单列：全 rail 编程入口 = 1（基线 §3.1 双入口缺陷闭合）', () => {
    renderRail()
    expect(screen.getAllByLabelText(/编程/)).toHaveLength(1)
    expect(screen.getByLabelText('编程').closest('.v3-rail-foot')).not.toBeNull()
  })

  it('play 空间：主体 = 共享 + 闲庭板块（聊天/小说/绘梦/模型中心/角色库），空间标识激活态跟随', () => {
    const { rerender } = renderRail()
    rerender(wrap(
      <CommandRail
        page="home"
        space="play"
        darkMode={false}
        toggleDarkMode={vi.fn()}
        onNavigate={vi.fn()}
        onSwitchSpace={vi.fn()}
      />,
    ))
    expect(navLabels()).toEqual(['首页', '聊天', '小说', '绘梦', '模型中心', '角色库'])
    expect(screen.getByTestId('v3-rail-space-indicator').textContent).toBe('闲庭')
    expect(screen.getAllByLabelText(/编程/)).toHaveLength(1)
  })

  it('点击办公 → onNavigate("gaea")；点击编程(foot) → onNavigate("code")；空间切换点击语义已迁首页顶栏（ModuleLauncher SpaceSwitch）', () => {
    const { onNavigate } = renderRail()
    fireEvent.click(screen.getByLabelText('办公'))
    expect(onNavigate).toHaveBeenCalledWith('gaea')
    fireEvent.click(screen.getByLabelText('编程'))
    expect(onNavigate).toHaveBeenCalledWith('code')
  })
})