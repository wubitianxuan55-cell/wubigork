import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import { CommandRail } from './MainLayout'
import { LocaleProvider } from '../gaea/lib/i18n'

// CommandRail 一级导航合一（2026-10-26 用户拍板）：rail 不再按 书斋/闲庭
// 分域——全部 inMenu 板块一栏尽收（getActiveMenuBoards）；空间指示徽标随
// 分域退役；space/onSwitchSpace props 退役（切空间由 navigateBoard 按目标
// 板块归属隐式驱动）。LocaleProvider 钉 zh（localStorage gaea-lang）。
const wrap = (ui: ReactElement) => <LocaleProvider>{ui}</LocaleProvider>

function renderRail(props: Partial<Parameters<typeof CommandRail>[0]> = {}) {
  const onNavigate = vi.fn()
  const utils = render(wrap(
    <CommandRail
      page="gaea"
      darkMode={false}
      toggleDarkMode={vi.fn()}
      onNavigate={onNavigate}
      {...props}
    />,
  ))
  return { ...utils, onNavigate }
}

/** rail 主体（menubar）内按钮的 aria-label 序 */
function navLabels(): (string | null)[] {
  const nav = screen.getByRole('menubar')
  return Array.from(nav.querySelectorAll('button')).map((b) => b.getAttribute('aria-label'))
}

beforeEach(() => {
  localStorage.setItem('gaea-lang', 'zh')
})

describe('CommandRail 一级导航合一（2026-10-26）', () => {
  it('全量一级导航：办公默认 + 全部 inMenu 板块同栏尽收（首页已删除；进度计划 2026-11-14 拍板恢复一级；文学 2026-11-15 接替退役小说板块一级位）', () => {
    renderRail()
    expect(navLabels()).toEqual([
      '聊天', '文学', '绘梦', '办公', '造价数据库', '众神殿', '记忆中枢', '模型中心', '角色库', '微信助手', '进度计划', '运行监测',
    ])
  })

  it('空间分域残面零残留：指示徽标/切换器/竖排微章全 null', () => {
    renderRail()
    expect(screen.queryByTestId('v3-rail-space-indicator')).toBeNull()
    expect(screen.queryByTestId('v3-rail-space-switch')).toBeNull()
    expect(document.querySelector('.v3-rail-space')).toBeNull()
  })

  it('independent 分面恢复编程：foot 段单列，主 rail 不重复', () => {
    renderRail()
    expect(screen.queryAllByLabelText(/编程/)).toHaveLength(1)
    expect(navLabels()).not.toContain('编程')
    expect(document.querySelector('.v3-rail-foot')).not.toBeNull()
    expect(document.querySelector('.v3-rail-divider')).not.toBeNull()
    expect(screen.getByLabelText('切换暗色')).toBeTruthy()
  })

  it('点击办公/文学均直达（同栏跨域可点；空间切换由 navigateBoard 隐式驱动）', () => {
    const { onNavigate } = renderRail()
    fireEvent.click(screen.getByLabelText('办公'))
    expect(onNavigate).toHaveBeenCalledWith('gaea')
    fireEvent.click(screen.getByLabelText('文学'))
    expect(onNavigate).toHaveBeenCalledWith('wenxue')
  })

  it('点击独立窗口“编程”直达 code 页面', () => {
    const { onNavigate } = renderRail()
    fireEvent.click(screen.getByLabelText('编程'))
    expect(onNavigate).toHaveBeenCalledWith('code')
  })

  it('激活态跟随 page（aria-current=page + is-active 容器）', () => {
    renderRail({ page: 'gaea' })
    const active = document.querySelector('.v3-rail-item.is-active')
    expect(active?.getAttribute('aria-label')).toBe('办公')
    expect(active?.getAttribute('aria-current')).toBe('page')
  })

  it('一级导航均渲染矢量图标，微信助手使用品牌图标', () => {
    renderRail()
    const labels = navLabels().filter((label): label is string => Boolean(label))
    expect(labels).toHaveLength(12)
    for (const label of labels) {
      const button = screen.getByLabelText(label)
      expect(button.querySelector('.v3-rail-icon svg')).not.toBeNull()
    }
    expect(screen.getByLabelText('编程').querySelector('.v3-rail-icon svg')).not.toBeNull()
    expect(screen.getByLabelText('微信助手').querySelector('.anticon-wechat')).not.toBeNull()
  })
})
