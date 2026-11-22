import { describe, expect, it } from 'vitest'
import {
  FEATURED_THEME_PRESETS,
  THEME_PRESETS,
  getThemeTokens,
  useAppStore,
  type ThemePreset,
} from './appStore'

const PRESETS: ThemePreset[] = THEME_PRESETS.map((preset) => preset.key)
const HEX = /^#[0-9a-fA-F]{6}$/

describe('Bloom 15 套主题注册表与令牌完整性', () => {
  it('全量 15 套，精选 12 套，三个非精选色仍保留', () => {
    expect(THEME_PRESETS).toHaveLength(15)
    expect(FEATURED_THEME_PRESETS).toHaveLength(12)
    expect(new Set(PRESETS).size).toBe(15)
    expect(PRESETS).toEqual(expect.arrayContaining(['stone', 'amber', 'sage']))
    expect(FEATURED_THEME_PRESETS.map((preset) => preset.key)).not.toEqual(
      expect.arrayContaining(['stone', 'amber', 'sage']),
    )
  })

  it('15 × 明暗全部产出完整令牌集', () => {
    for (const preset of PRESETS) {
      for (const dark of [true, false]) {
        const tokens = getThemeTokens(preset, dark)
        expect(tokens.colorPrimary, `${preset}${dark ? 'D' : 'L'}.colorPrimary`).toMatch(HEX)
        expect(tokens.surface, `${preset}.surface`).toMatch(HEX)
        expect(tokens.colorText, `${preset}.colorText`).toMatch(HEX)
        expect(tokens.accentRgb, `${preset}.accentRgb`).toMatch(/^\d+,\d+,\d+$/)
        expect(tokens.ambienceRgb, `${preset}.ambienceRgb`).toMatch(/^\d+,\d+,\d+$/)
      }
    }
  })

  it('亮色不再被统一暖纸覆盖，15 套画布保持独立色相', () => {
    const lightLayouts = new Set(PRESETS.map((preset) => getThemeTokens(preset, false).colorBgLayout))
    const darkLayouts = new Set(PRESETS.map((preset) => getThemeTokens(preset, true).colorBgLayout))
    expect(lightLayouts.size).toBe(PRESETS.length)
    expect(darkLayouts.size).toBe(PRESETS.length)
  })

  it('danger 与 tertiary 语义完整，主色前景自动取高对比侧', () => {
    for (const preset of PRESETS) {
      for (const dark of [true, false]) {
        const tokens = getThemeTokens(preset, dark)
        expect(tokens.colorDestructive).toMatch(HEX)
        expect(tokens.tertiaryContainer).toMatch(HEX)
        expect(tokens.onTertiaryContainer).toMatch(HEX)
        expect(tokens.onPrimary).not.toBe(tokens.colorPrimary)
      }
    }
  })

  it('明暗共享圆角与动效常量的节奏一致', () => {
    const dark = getThemeTokens('ripple', true)
    const light = getThemeTokens('ripple', false)
    expect(dark.radiusMd).toBe(light.radiusMd)
    expect(dark.transitionNormal).toBe(light.transitionNormal)
  })
})

describe('主题选择持久化', () => {
  it('setTheme 写 store 和 localStorage', () => {
    localStorage.clear()
    useAppStore.getState().setTheme('lapis')
    expect(useAppStore.getState().baseTheme).toBe('lapis')
    expect(localStorage.getItem('gaea-theme')).toBe('lapis')
  })

  it('一键玻璃态保存并恢复此前的非玻璃表面', () => {
    localStorage.clear()
    useAppStore.setState({ surfaceStyle: 'solid', surfaceStyleBeforeGlass: 'veil' })
    useAppStore.getState().toggleGlassMode()
    expect(useAppStore.getState().surfaceStyle).toBe('soft')
    expect(useAppStore.getState().surfaceStyleBeforeGlass).toBe('solid')
    expect(localStorage.getItem('gaea-surface-style')).toBe('soft')
    expect(localStorage.getItem('gaea-surface-style-restore')).toBe('solid')

    useAppStore.getState().toggleGlassMode()
    expect(useAppStore.getState().surfaceStyle).toBe('solid')
    expect(localStorage.getItem('gaea-surface-style')).toBe('solid')
  })
})

describe('useAppStore.space', () => {
  it('默认工位 work，setSpace 可持久化', () => {
    localStorage.removeItem('gaea.shell.space')
    expect(useAppStore.getState().space).toBe('work')
    useAppStore.getState().setSpace('play')
    expect(useAppStore.getState().space).toBe('play')
    expect(localStorage.getItem('gaea.shell.space')).toBe('play')
    useAppStore.getState().setSpace('work')
    expect(localStorage.getItem('gaea.shell.space')).toBe('work')
  })
})
