import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { THEME_PRESETS, getThemeTokens, type ThemePreset } from './appStore'
import { NEUTRAL_PALETTE, PALETTE_SPECS } from '../theme/bloomPalette'
import { mixOklchHex, oklchToHex } from '../theme/color'

const PRESETS: ThemePreset[] = THEME_PRESETS.map((preset) => preset.key)
const BLACK = oklchToHex(NEUTRAL_PALETTE.black)
const BODY_TOKENS = [
  'colorText',
  'colorTextSecondary',
  'colorTextTertiary',
] as const
const STATE_TOKENS = [
  'colorSuccess',
  'colorWarning',
  'colorDestructive',
  'colorInfo',
] as const

function srgbChannel(value: number): number {
  const channel = value / 255
  return channel <= 0.03928 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4
}

function luminance(hex: string): number {
  const value = hex.trim().replace('#', '')
  const [r, g, b] = [0, 2, 4].map((offset) => Number.parseInt(value.slice(offset, offset + 2), 16))
  return 0.2126 * srgbChannel(r) + 0.7152 * srgbChannel(g) + 0.0722 * srgbChannel(b)
}

function contrast(a: string, b: string): number {
  const first = luminance(a)
  const second = luminance(b)
  return (Math.max(first, second) + 0.05) / (Math.min(first, second) + 0.05)
}

describe('Bloom 15 套主题对比度契约', () => {
  for (const preset of PRESETS) {
    for (const dark of [true, false]) {
      it(`${preset}${dark ? 'D' : 'L'} 正文语义令牌达到 WCAG AA`, () => {
        const tokens = getThemeTokens(preset, dark)
        const backgrounds = [
          ['surface', tokens.surface],
          ['colorBgContainer', tokens.colorBgContainer],
          ['surfaceTint1', tokens.surfaceTint1],
          ['surfaceTint2', tokens.surfaceTint2],
          ['surfaceTint3', tokens.surfaceTint3],
        ] as const
        for (const token of BODY_TOKENS) {
          for (const [backgroundName, background] of backgrounds) {
            expect(background, `${preset} ${backgroundName} 必须是 hex`).toMatch(/^#[0-9a-fA-F]{6}$/)
            expect(
              contrast(tokens[token], background),
              `${preset}${dark ? 'D' : 'L'} ${token} on ${backgroundName}`,
            ).toBeGreaterThanOrEqual(4.5)
          }
        }
        // Upstream sharedDswTokens keeps semantic state hues fixed and guarantees
        // their primary/secondary track against the palette base, not every
        // elevated surface. Raised status surfaces consume the tertiary track.
        for (const token of STATE_TOKENS) {
          expect(
            contrast(tokens[token], tokens.bodyBackground),
            `${preset}${dark ? 'D' : 'L'} ${token} on base`,
          ).toBeGreaterThanOrEqual(4.5)
        }
      })
    }
  }
})

describe('Bloom 壳层与按钮前景契约', () => {
  for (const preset of PRESETS) {
    for (const dark of [true, false]) {
      it(`${preset}${dark ? 'D' : 'L'} 侧栏正文与主按钮前景可读`, () => {
        const tokens = getThemeTokens(preset, dark)
        for (const token of ['colorText', 'colorTextSecondary', 'colorTextTertiary'] as const) {
          expect(
            contrast(tokens[token], tokens.shellSidebar),
            `${preset}${dark ? 'D' : 'L'} ${token} on shellSidebar`,
          ).toBeGreaterThanOrEqual(4.5)
        }
        expect(contrast(tokens.onPrimary, tokens.colorPrimary)).toBeGreaterThanOrEqual(4.5)
      })
    }
  }
})

describe('Bloom 0.29 可读强调轨契约', () => {
  for (const preset of PRESETS) {
    for (const dark of [true, false]) {
      it(`${preset}${dark ? 'D' : 'L'} 使用上游 readableAccent 与背景前景`, () => {
        const tokens = getThemeTokens(preset, dark)
        const palette = PALETTE_SPECS[preset]
        const rawAccent = oklchToHex(dark ? palette.accentD : palette.accentL)
        const expected = dark ? rawAccent : mixOklchHex(rawAccent, BLACK, 0.12)
        expect(tokens.colorPrimary).toBe(expected)
        expect(tokens.accentReadable).toBe(expected)
        expect(tokens.onPrimary).toBe(tokens.bodyBackground)
      })
    }
  }
})

describe('旧 7 套主题彻底退役', () => {
  it('主题注册表源码不再保留旧键', () => {
    const root = resolve(__dirname, '..')
    const sources = [
      readFileSync(resolve(root, 'stores/appStore.ts'), 'utf8'),
      readFileSync(resolve(root, 'theme/bloomPalette.ts'), 'utf8'),
    ].join('\n')
    for (const legacy of ['nightJade', 'nightViolet', 'nightRose', 'nightAmber', 'nightMoss', 'nightSlate', 'tianXuan']) {
      expect(sources).not.toContain(legacy)
    }
  })
})
