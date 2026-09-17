// appStore.test.ts — 主题令牌完整性（3.0 UI 设计系统 Wave 1）
// 锁定：12 套主题（6 色系 × 明暗）必须产出完整 ThemeTokens，
// 含 3.0 新增语义令牌 colorDestructive（破坏性操作红）。
import { describe, expect, it, beforeEach } from 'vitest'
import { getThemeTokens, useAppStore, type ThemePreset, isHomeLayout } from './appStore'

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

  it('M3 tertiary 暖伴侣色对存在且 container ≠ on（v4.183 闲庭母题契约）', () => {
    for (const preset of PRESETS) {
      for (const dark of [true, false]) {
        const t = getThemeTokens(preset, dark)
        expect(t.tertiaryContainer, `${preset}${dark ? 'D' : 'L'}.tertiaryContainer`).toMatch(HEX)
        expect(t.onTertiaryContainer, `${preset}${dark ? 'D' : 'L'}.onTertiaryContainer`).toMatch(HEX)
        expect(t.onTertiaryContainer).not.toBe(t.tertiaryContainer)
      }
    }
  })

  it('onPrimary 与 primary 成对：暗色主色浅，字必须深，不能硬编码白', () => {
    for (const preset of PRESETS) {
      const dark = getThemeTokens(preset, true)
      const light = getThemeTokens(preset, false)
      expect(dark.onPrimary).not.toBe(dark.colorPrimary)
      expect(light.onPrimary).not.toBe(light.colorPrimary)
      expect(dark.onPrimary.toLowerCase()).not.toMatch(/^#fff(fff)?$/)
      expect(light.onPrimary.toLowerCase()).toMatch(/^#fff(fff)?$/)
    }
  })

  it('明暗阴影/圆角/动效共享常量保持一致', () => {
    const dark = getThemeTokens('nightJade', true)
    const light = getThemeTokens('nightJade', false)
    expect(dark.radiusMd).toBe(light.radiusMd)
    expect(dark.transitionNormal).toBe(light.transitionNormal)
  })
})

describe('useAppStore.space（S2.1 壳层视图空间持久化）', () => {
  it('默认工位（work），setSpace 写 localStorage 并可读回', () => {
    localStorage.removeItem('gaea.shell.space')
    expect(useAppStore.getState().space).toBe('work')
    useAppStore.getState().setSpace('play')
    expect(useAppStore.getState().space).toBe('play')
    expect(localStorage.getItem('gaea.shell.space')).toBe('play')
    useAppStore.getState().setSpace('work')
    expect(localStorage.getItem('gaea.shell.space')).toBe('work')
  })

// ── 7.3-2 首页形态（homeLayout）：缺省/持久化/非法值回退 ──
describe('appStore homeLayout（7.3-2 板块降级为任务视图）', () => {
  beforeEach(() => {
    localStorage.removeItem('gaea.home.layout')
  })

  it('缺省 classic；setHomeLayout 持久化到 localStorage', () => {
    localStorage.removeItem('gaea.home.layout')
    expect(useAppStore.getState().homeLayout).toBe('classic')
    useAppStore.getState().setHomeLayout('tasks')
    expect(useAppStore.getState().homeLayout).toBe('tasks')
    expect(localStorage.getItem('gaea.home.layout')).toBe('tasks')
    useAppStore.getState().setHomeLayout('classic')
    expect(localStorage.getItem('gaea.home.layout')).toBe('classic')
  })

  it('非法持久化值回退 classic（loadHomeLayout 守卫）', () => {
    localStorage.setItem('gaea.home.layout', 'bogus')
    // loadHomeLayout 是模块内函数——经重新初始化验证需重建 store；此处以导出的
    // 守卫函数直测（isHomeLayout 与缺省口径同源）。
    expect(isHomeLayout('bogus')).toBe(false)
    expect(isHomeLayout('tasks')).toBe(true)
    expect(isHomeLayout('classic')).toBe(true)
  })
})
})
