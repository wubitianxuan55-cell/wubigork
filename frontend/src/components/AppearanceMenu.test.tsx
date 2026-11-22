import { fireEvent, render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AppearanceMenu, ThemeMenuTrigger } from './AppearanceMenu'
import { LocaleProvider } from '../gaea/lib/i18n'
import { pickNextFeaturedTheme } from '../hooks/useThemeAutoRotation'
import { FEATURED_THEME_KEYS, useAppStore } from '../stores/appStore'

const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>

beforeEach(() => {
  localStorage.clear()
  Object.defineProperty(navigator, 'language', { value: 'zh-CN', configurable: true })
  useAppStore.setState({
    baseTheme: 'ripple',
    mode: 'dark',
    surfaceStyle: 'veil',
    surfaceStyleBeforeGlass: 'veil',
    ambienceStyle: 'grid',
    motion: 'full',
    themeRotationMinutes: 0,
  })
})

describe('顶栏 Bloom 外观菜单', () => {
  it('顶栏胶囊可反复打开完整外观面板', async () => {
    render(wrap(<ThemeMenuTrigger />))
    const trigger = screen.getByRole('button', { name: /打开外观菜单/ })
    fireEvent.click(trigger)
    expect(await screen.findByRole('dialog', { name: '外观' })).toBeTruthy()
    expect(within(screen.getByRole('listbox', { name: '主题' })).getAllByRole('option')).toHaveLength(15)
    fireEvent.click(trigger)
    expect(await screen.findByRole('button', { name: /打开外观菜单/ })).toBeTruthy()
    fireEvent.click(trigger)
    expect(await screen.findByRole('dialog', { name: '外观' })).toBeTruthy()
  })

  it('一次提供 15 套配色，并标记当前项', () => {
    render(wrap(<AppearanceMenu onOpenSettings={() => {}} />))
    const listbox = screen.getByRole('listbox', { name: '主题' })
    const options = within(listbox).getAllByRole('option')
    expect(options).toHaveLength(15)
    expect(options.map((option) => option.getAttribute('aria-selected'))).toContain('true')
    expect(within(listbox).getByRole('option', { name: /天青/ }).getAttribute('aria-selected')).toBe('true')
  })

  it('直接选择配色后持久化，面板保持可继续调节', () => {
    render(wrap(<AppearanceMenu onOpenSettings={() => {}} />))
    fireEvent.click(screen.getByRole('option', { name: /青莲/ }))
    expect(useAppStore.getState().baseTheme).toBe('lavender')
    expect(localStorage.getItem('gaea-theme')).toBe('lavender')
    expect(screen.getByRole('option', { name: /青莲/ }).getAttribute('aria-selected')).toBe('true')
  })

  it('配色网格支持二维方向键、Home 与 End', () => {
    render(wrap(<AppearanceMenu onOpenSettings={() => {}} />))
    const options = within(screen.getByRole('listbox', { name: '主题' })).getAllByRole('option')
    options[0].focus()
    fireEvent.keyDown(options[0], { key: 'ArrowDown' })
    expect(document.activeElement).toBe(options[6])
    fireEvent.keyDown(options[6], { key: 'ArrowRight' })
    expect(document.activeElement).toBe(options[7])
    fireEvent.keyDown(options[7], { key: 'End' })
    expect(document.activeElement).toBe(options[options.length - 1])
    fireEvent.keyDown(options[options.length - 1], { key: 'Home' })
    expect(document.activeElement).toBe(options[0])
  })

  it('悬停色块时更新配色字幕', () => {
    render(wrap(<AppearanceMenu onOpenSettings={() => {}} />))
    fireEvent.mouseEnter(screen.getByRole('option', { name: /青莲/ }))
    expect(screen.getByText('清水出芙蓉')).toBeTruthy()
  })

  it('模式图标即时生效，细节配置提示指向设置页', () => {
    render(wrap(<AppearanceMenu onOpenSettings={() => {}} />))
    fireEvent.click(screen.getByRole('radio', { name: '亮色' }))
    expect(useAppStore.getState().mode).toBe('light')
    expect(localStorage.getItem('gaea-display-mode')).toBe('light')
    expect(screen.getByText(/质感、背景与动效/)).toBeTruthy()
  })

  it('“更多外观设置”进入通用外观页签', () => {
    const onOpenSettings = vi.fn()
    render(wrap(<AppearanceMenu onOpenSettings={onOpenSettings} />))
    fireEvent.click(screen.getByRole('button', { name: '更多外观设置' }))
    expect(onOpenSettings).toHaveBeenCalledTimes(1)
  })
})

describe('自动换色', () => {
  it('只从精选色中选择且永远避开当前色', () => {
    expect(FEATURED_THEME_KEYS).toContain(pickNextFeaturedTheme('ripple', () => 0))
    expect(pickNextFeaturedTheme('ripple', () => 0)).not.toBe('ripple')
    expect(pickNextFeaturedTheme('ripple', () => 0.999999)).not.toBe('ripple')
  })
})
