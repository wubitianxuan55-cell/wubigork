import { act, fireEvent, render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import AppearancePanel, {
  AmbiencePanel,
  DarkModePanel,
  DensityPanel,
  MeteorPanel,
  MotionPanel,
  SurfacePanel,
} from './AppearancePanel'
import { THEME_PRESETS, useAppStore } from '../../stores/appStore'
import { LocaleProvider } from '../../gaea/lib/i18n'

const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>

const directText = (element: Element): string =>
  Array.from(element.childNodes)
    .filter((node) => node.nodeType === Node.TEXT_NODE)
    .map((node) => node.textContent ?? '')
    .join('')
    .trim()

const byDirectText = (text: string) => (_content: string, element: Element | null): boolean => {
  if (!element) return false
  const value = directText(element)
  return value.length > 0 && (value === text || value.endsWith(text))
}

const openSelect = (container: HTMLElement) => {
  const selector = container.querySelector('.ant-select-selector')
  expect(selector).toBeTruthy()
  fireEvent.mouseDown(selector as HTMLElement)
}

beforeEach(() => {
  localStorage.clear()
  useAppStore.setState({
    baseTheme: 'ripple',
    mode: 'dark',
    density: 'standard',
    motion: 'full',
    meteorColorMode: 'colorful',
    meteorDensity: 'standard',
    meteorSpeed: 'standard',
  })
  Object.defineProperty(navigator, 'language', { value: 'zh-CN', configurable: true })
})

describe('Bloom 主题选择器', () => {
  it('设置页显示全部 15 套，默认天青', async () => {
    const { container } = render(wrap(<AppearancePanel />))
    const selector = container.querySelector('.ant-select-selector')
    expect(within(selector as HTMLElement).getByText(byDirectText('天青'))).toBeTruthy()
    openSelect(container)
    expect(await screen.findByText(byDirectText('赭石'))).toBeTruthy()
    expect(await screen.findByText(byDirectText('琥珀'))).toBeTruthy()
    expect(await screen.findByText(byDirectText('竹青'))).toBeTruthy()
    expect(THEME_PRESETS).toHaveLength(15)
  })

  it('选择非精选色也写入 store 与持久化', async () => {
    const { container } = render(wrap(<AppearancePanel />))
    openSelect(container)
    const amber = await screen.findByText(byDirectText('琥珀'))
    fireEvent.click(amber)
    expect(useAppStore.getState().baseTheme).toBe('amber')
    expect(localStorage.getItem('gaea-theme')).toBe('amber')
  })

  it('悬停主题出现实时预览', async () => {
    const { container } = render(wrap(<AppearancePanel />))
    openSelect(container)
    const lavender = await screen.findByText(byDirectText('青莲'))
    fireEvent.mouseEnter(lavender)
    expect(await screen.findByText(byDirectText('预览中'))).toBeTruthy()
  })

  it('显示模式切换', () => {
    render(wrap(<DarkModePanel />))
    fireEvent.click(screen.getByText(byDirectText('亮色模式')))
    expect(useAppStore.getState().mode).toBe('light')
    expect(localStorage.getItem('gaea-display-mode')).toBe('light')
  })

  it('密度切换', () => {
    render(wrap(<DensityPanel />))
    fireEvent.click(screen.getByText(byDirectText('紧凑')))
    expect(useAppStore.getState().density).toBe('compact')
    expect(localStorage.getItem('gaea-density')).toBe('compact')
  })

  it('动效切换', () => {
    render(wrap(<MotionPanel />))
    fireEvent.click(screen.getByText(byDirectText('减弱动态')))
    expect(useAppStore.getState().motion).toBe('reduced')
    expect(localStorage.getItem('gaea-motion')).toBe('reduced')
  })

  it('表面材质切换', () => {
    useAppStore.setState({ surfaceStyle: 'veil' })
    render(wrap(<SurfacePanel />))
    fireEvent.click(screen.getByText(byDirectText('磨砂玻璃')))
    expect(useAppStore.getState().surfaceStyle).toBe('matte')
    expect(localStorage.getItem('gaea-surface-style')).toBe('matte')
    fireEvent.click(screen.getByText(byDirectText('液态玻璃')))
    expect(useAppStore.getState().surfaceStyle).toBe('soft')
    expect(localStorage.getItem('gaea-surface-style')).toBe('soft')
  })

  it('亮暗背景氛围分别切换并持久化', async () => {
    useAppStore.setState({ darkMode: true, lightAmbienceStyle: 'stream', darkAmbienceStyle: 'corner' })
    render(wrap(<AmbiencePanel />))
    expect(screen.queryByRole('radio', { name: '水彩晕染' })).toBeNull()
    fireEvent.click(screen.getByRole('radio', { name: '对角漫染' }))
    expect(useAppStore.getState().darkAmbienceStyle).toBe('diagonal')
    expect(localStorage.getItem('gaea-ambience-dark')).toBe('diagonal')
    await act(async () => {
      useAppStore.setState({ darkMode: false })
    })
    fireEvent.click(screen.getByRole('radio', { name: '水彩晕染' }))
    expect(useAppStore.getState().lightAmbienceStyle).toBe('wash')
    expect(localStorage.getItem('gaea-ambience-light')).toBe('wash')
  })

  it('流星细节仅在暗色漫天星辰下出现', () => {
    useAppStore.setState({ darkMode: true, darkAmbienceStyle: 'corner' })
    const { rerender } = render(wrap(<MeteorPanel />))
    expect(screen.queryByText(byDirectText('流星'))).toBeNull()
    useAppStore.setState({ darkAmbienceStyle: 'galaxy' })
    rerender(wrap(<MeteorPanel />))
    expect(screen.getByText(byDirectText('流星'))).toBeTruthy()
  })

  it('流星颜色、密度与速度切换并持久化', () => {
    useAppStore.setState({ darkMode: true, darkAmbienceStyle: 'galaxy' })
    render(wrap(<MeteorPanel />))
    fireEvent.click(screen.getByText(byDirectText('纯色')))
    expect(useAppStore.getState().meteorColorMode).toBe('solid')
    expect(localStorage.getItem('gaea-meteor-color')).toBe('solid')
    fireEvent.click(screen.getByText(byDirectText('密集')))
    expect(useAppStore.getState().meteorDensity).toBe('dense')
    expect(localStorage.getItem('gaea-meteor-density')).toBe('dense')
    fireEvent.click(screen.getByText(byDirectText('快')))
    expect(useAppStore.getState().meteorSpeed).toBe('fast')
    expect(localStorage.getItem('gaea-meteor-speed')).toBe('fast')
  })
})
