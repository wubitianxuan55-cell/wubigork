// appStore.test.ts — 主题令牌完整性（3.0 UI 设计系统 Wave 1）
// 锁定：12 套主题（6 色系 × 明暗）必须产出完整 ThemeTokens，
// 含 3.0 新增语义令牌 colorDestructive（破坏性操作红）。
import { describe, expect, it } from 'vitest'
import { getThemeTokens, type ThemePreset } from './appStore'

const PRESETS: ThemePreset[] = ['nightJade', 'nightViolet', 'nightRose', 'nightAmber', 'nightMoss', 'nightSlate']
const HEX = /^#[0-9a-fA-F]{6}$/

describe('getThemeTokens（3.0 设计系统 Wave 1 令牌契约）', () => {
  it('12 套主题（6 色系 × 明暗）全部产出完整令牌集', () => {
    for (const preset of PRESETS) {
      for (const dark of [true, false]) {
        const t = getThemeTokens(preset, dark)
        expect(t.colorPrimary, `${preset}${dark ? 'D' : 'L'}.colorPrimary`).toMatch(HEX)
        expect(t.surface, `${preset}.surface`).toMatch(HEX)
        expect(t.colorText, `${preset}.colorText`).toMatch(HEX)
        expect(t.accentRgb, `${preset}.accentRgb`).toMatch(/^\d+,\d+,\d+$/)
      }
    }
  })

  it('colorDestructive 语义令牌存在（暗 #ef4444 / 亮 #dc2626）', () => {
    for (const preset of PRESETS) {
      const dark = getThemeTokens(preset, true)
      expect(dark.colorDestructive, `${preset} 暗色 destructive`).toBe('#ef4444')
      const light = getThemeTokens(preset, false)
      expect(light.colorDestructive, `${preset} 亮色 destructive`).toBe('#dc2626')
    }
  })

  it('明暗阴影/圆角/动效共享常量保持一致', () => {
    const dark = getThemeTokens('nightJade', true)
    const light = getThemeTokens('nightJade', false)
    expect(dark.radiusMd).toBe(light.radiusMd)
    expect(dark.transitionNormal).toBe(light.transitionNormal)
  })
})
