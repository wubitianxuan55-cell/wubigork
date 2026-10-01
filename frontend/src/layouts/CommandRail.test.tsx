import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import { CommandRail } from './MainLayout'
import { LocaleProvider } from '../gaea/lib/i18n'

// CommandRail 双空间分域导航（v4.169）：切换器（SHELL_SPACES 驱动）+
// 主体按当前空间过滤（shared 恒在 / independent 剔除 / inMenu 过滤）+
// independent 独立窗口仅 foot 单列（v4.439 编程板块删除后 foot 段为空 ⇒ 自动隐藏）。
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

  it('independent 分面为空（v4.439 编程板块删除）：foot 段不渲染任何独立窗口入口', () => {
    renderRail()
    expect(screen.queryAllByLabelText(/编程/)).toHaveLength(0)
    // foot 容器仍在（深浅色切换常驻），但独立窗口分隔线/入口整块不渲染
    expect(document.querySelector('.v3-rail-foot')).not.toBeNull()
    expect(document.querySelector('.v3-rail-divider')).toBeNull()
    expect(screen.getByLabelText('切换暗色')).toBeTruthy()
  })

  it('play 空间：主体 = 共享 + 闲庭板块（聊天/小说/绘梦/模型中心/角色库/原罪），空间标识激活态跟随', () => {
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
    expect(navLabels()).toEqual(['首页', '聊天', '小说', '绘梦', '模型中心', '角色库', '原罪'])
    expect(screen.getByTestId('v3-rail-space-indicator').textContent).toBe('闲庭')
    expect(screen.queryAllByLabelText(/编程/)).toHaveLength(0)
  })

  it('点击办公 → onNavigate("gaea")（编程入口已随板块删除；空间切换点击语义已迁首页顶栏）', () => {
    const { onNavigate } = renderRail()
    fireEvent.click(screen.getByLabelText('办公'))
    expect(onNavigate).toHaveBeenCalledWith('gaea')
    expect(screen.queryByLabelText('编程')).toBeNull()
  })
})
