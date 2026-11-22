import { oklchToHex } from './color'

export type ThemePreset =
  | 'cinnabar'
  | 'dawn'
  | 'tangerine'
  | 'aurora'
  | 'stone'
  | 'amber'
  | 'osmanthus'
  | 'mint'
  | 'sage'
  | 'ripple'
  | 'mist'
  | 'lapis'
  | 'lavender'
  | 'silverrose'
  | 'petal'

export type PaletteFamily = 'red' | 'orange' | 'yellow' | 'green' | 'cyan' | 'blue' | 'purple' | 'pink'

interface PaletteSpec {
  accentL: string
  accentD: string
  morandi: string
  bgL: string
  bgD: string
  txL: string
  txD: string
  sfL: string
  sfD: string
  sf2L: string
  sf2D: string
  motionL: readonly [string, string, string]
  motionD: readonly [string, string, string]
}

/**
 * Bloom's dual-track palette imported from dsh-bloom-theme@0.28.0:
 * accent is readable UI ink; morandi is decoration only.
 */
export const PALETTE_SPECS = {
  cinnabar: {
    accentL: 'oklch(55% 0.18 25)', accentD: 'oklch(72% 0.12 25)', morandi: '215, 75, 75',
    bgL: 'oklch(97% 0.005 25)', bgD: 'oklch(28% 0.02 25)',
    txL: 'oklch(25% 0.02 25)', txD: 'oklch(96% 0.01 25)',
    sfL: 'oklch(95% 0.005 25)', sfD: 'oklch(34% 0.02 25)',
    sf2L: 'oklch(92% 0.005 25)', sf2D: 'oklch(40% 0.02 25)',
    motionL: ['oklch(55% 0.18 25)', 'oklch(55% 0.16 65)', 'oklch(55% 0.15 350)'],
    motionD: ['oklch(72% 0.12 25)', 'oklch(74% 0.14 65)', 'oklch(73% 0.13 350)'],
  },
  dawn: {
    accentL: 'oklch(51% 0.15 8)', accentD: 'oklch(78% 0.13 8)', morandi: '219, 147, 151',
    bgL: 'oklch(98% 0.006 8)', bgD: 'oklch(24% 0.012 8)',
    txL: 'oklch(24% 0.013 8)', txD: 'oklch(96% 0.007 8)',
    sfL: 'oklch(95.8% 0.009 8)', sfD: 'oklch(30% 0.017 8)',
    sf2L: 'oklch(93% 0.011 8)', sf2D: 'oklch(36% 0.021 8)',
    motionL: ['oklch(51% 0.15 8)', 'oklch(52% 0.12 8)', 'oklch(52% 0.11 50)'],
    motionD: ['oklch(78% 0.13 8)', 'oklch(79% 0.12 8)', 'oklch(79% 0.11 50)'],
  },
  tangerine: {
    accentL: 'oklch(55% 0.16 47)', accentD: 'oklch(73% 0.17 50)', morandi: '226, 128, 64',
    bgL: 'oklch(97.8% 0.005 60)', bgD: 'oklch(24% 0.01 50)',
    txL: 'oklch(24% 0.014 50)', txD: 'oklch(96% 0.006 60)',
    sfL: 'oklch(95.8% 0.006 60)', sfD: 'oklch(30% 0.012 50)',
    sf2L: 'oklch(93% 0.008 58)', sf2D: 'oklch(36% 0.015 50)',
    motionL: ['oklch(55% 0.16 47)', 'oklch(56% 0.15 30)', 'oklch(57% 0.13 65)'],
    motionD: ['oklch(73% 0.17 50)', 'oklch(74% 0.16 32)', 'oklch(78% 0.14 66)'],
  },
  aurora: {
    accentL: 'oklch(55% 0.145 72)', accentD: 'oklch(80% 0.13 72)', morandi: '218, 158, 72',
    bgL: 'oklch(98% 0.004 72)', bgD: 'oklch(24.5% 0.008 70)',
    txL: 'oklch(24% 0.012 72)', txD: 'oklch(95% 0.005 72)',
    sfL: 'oklch(96.2% 0.005 72)', sfD: 'oklch(30.5% 0.01 70)',
    sf2L: 'oklch(93.4% 0.007 72)', sf2D: 'oklch(36.5% 0.012 70)',
    motionL: ['oklch(55% 0.145 72)', 'oklch(56% 0.15 50)', 'oklch(56% 0.13 32)'],
    motionD: ['oklch(80% 0.13 72)', 'oklch(80% 0.14 50)', 'oklch(80% 0.12 32)'],
  },
  stone: {
    accentL: 'oklch(49.9% 0.06 29)', accentD: 'oklch(75% 0.12 30)', morandi: '180, 160, 155',
    bgL: 'oklch(95.9% 0.01 25)', bgD: 'oklch(20.1% 0.019 30)',
    txL: 'oklch(25% 0.02 29)', txD: 'oklch(95.9% 0.01 25)',
    sfL: 'oklch(94.1% 0.01 33)', sfD: 'oklch(27.9% 0.02 28)',
    sf2L: 'oklch(91% 0.01 33)', sf2D: 'oklch(33.9% 0.03 28)',
    motionL: ['oklch(49.9% 0.06 29)', 'oklch(49.9% 0.075 0)', 'oklch(49.9% 0.07 60)'],
    motionD: ['oklch(75% 0.12 30)', 'oklch(75% 0.13 0)', 'oklch(75% 0.11 62)'],
  },
  amber: {
    accentL: 'oklch(55.5% 0.12 70)', accentD: 'oklch(78% 0.11 70)', morandi: '159, 100, 1',
    bgL: 'oklch(97.5% 0.008 74)', bgD: 'oklch(26% 0.018 60)',
    txL: 'oklch(24% 0.02 74)', txD: 'oklch(95% 0.008 70)',
    sfL: 'oklch(95.6% 0.01 82)', sfD: 'oklch(32% 0.02 60)',
    sf2L: 'oklch(93% 0.014 78)', sf2D: 'oklch(38% 0.022 60)',
    motionL: ['oklch(55.5% 0.12 70)', 'oklch(55.5% 0.11 38)', 'oklch(54.1% 0.12 100)'],
    motionD: ['oklch(78% 0.11 70)', 'oklch(78% 0.12 38)', 'oklch(78% 0.11 100)'],
  },
  osmanthus: {
    accentL: 'oklch(50% 0.12 98)', accentD: 'oklch(86% 0.14 98)', morandi: '214, 186, 92',
    bgL: 'oklch(97.8% 0.006 95)', bgD: 'oklch(24% 0.008 95)',
    txL: 'oklch(24% 0.014 95)', txD: 'oklch(96% 0.008 95)',
    sfL: 'oklch(95.8% 0.008 95)', sfD: 'oklch(30% 0.012 95)',
    sf2L: 'oklch(93% 0.01 95)', sf2D: 'oklch(36% 0.015 95)',
    motionL: ['oklch(50% 0.12 98)', 'oklch(52% 0.11 80)', 'oklch(51% 0.10 120)'],
    motionD: ['oklch(86% 0.14 98)', 'oklch(84% 0.13 80)', 'oklch(84% 0.12 120)'],
  },
  mint: {
    accentL: 'oklch(51% 0.10 160)', accentD: 'oklch(82% 0.12 160)', morandi: '128, 196, 166',
    bgL: 'oklch(97.6% 0.006 160)', bgD: 'oklch(24% 0.012 165)',
    txL: 'oklch(24% 0.015 165)', txD: 'oklch(96% 0.008 160)',
    sfL: 'oklch(95.6% 0.008 160)', sfD: 'oklch(30% 0.015 165)',
    sf2L: 'oklch(92.8% 0.01 160)', sf2D: 'oklch(36% 0.018 165)',
    motionL: ['oklch(51% 0.10 160)', 'oklch(52% 0.10 135)', 'oklch(51% 0.09 185)'],
    motionD: ['oklch(82% 0.12 160)', 'oklch(84% 0.12 135)', 'oklch(80% 0.10 185)'],
  },
  sage: {
    accentL: 'oklch(54.1% 0.111 115)', accentD: 'oklch(71.9% 0.120 115)', morandi: '138, 154, 91',
    bgL: 'oklch(97% 0.011 112)', bgD: 'oklch(20% 0.019 113)',
    txL: 'oklch(25% 0.02 116)', txD: 'oklch(96% 0.011 118)',
    sfL: 'oklch(94.9% 0.009 113)', sfD: 'oklch(27.9% 0.02 116)',
    sf2L: 'oklch(91.9% 0.009 113)', sf2D: 'oklch(34% 0.03 116)',
    motionL: ['oklch(54.1% 0.111 115)', 'oklch(54.1% 0.12 83)', 'oklch(53.1% 0.10 152)'],
    motionD: ['oklch(71.9% 0.120 115)', 'oklch(71.9% 0.13 83)', 'oklch(71.9% 0.11 152)'],
  },
  ripple: {
    accentL: 'oklch(51% 0.12 195)', accentD: 'oklch(75% 0.12 195)', morandi: '95, 168, 178',
    bgL: 'oklch(96% 0.01 195)', bgD: 'oklch(20% 0.02 195)',
    txL: 'oklch(25% 0.02 195)', txD: 'oklch(96% 0.01 195)',
    sfL: 'oklch(94% 0.01 195)', sfD: 'oklch(28% 0.02 195)',
    sf2L: 'oklch(92% 0.01 195)', sf2D: 'oklch(38% 0.02 195)',
    motionL: ['oklch(51% 0.12 195)', 'oklch(51% 0.13 225)', 'oklch(51% 0.11 165)'],
    motionD: ['oklch(75% 0.12 195)', 'oklch(76% 0.14 225)', 'oklch(77% 0.12 165)'],
  },
  mist: {
    accentL: 'oklch(50% 0.08 240)', accentD: 'oklch(72% 0.12 240)', morandi: '146, 168, 179',
    bgL: 'oklch(96% 0.01 240)', bgD: 'oklch(28% 0.02 240)',
    txL: 'oklch(25% 0.02 240)', txD: 'oklch(96% 0.01 240)',
    sfL: 'oklch(94% 0.01 240)', sfD: 'oklch(34% 0.02 240)',
    sf2L: 'oklch(91% 0.01 240)', sf2D: 'oklch(40% 0.02 240)',
    motionL: ['oklch(50% 0.08 240)', 'oklch(50% 0.10 210)', 'oklch(50% 0.09 275)'],
    motionD: ['oklch(72% 0.12 240)', 'oklch(74% 0.13 210)', 'oklch(74% 0.11 275)'],
  },
  lapis: {
    accentL: 'oklch(50% 0.13 258)', accentD: 'oklch(74% 0.10 255)', morandi: '47, 98, 172',
    bgL: 'oklch(97.4% 0.006 240)', bgD: 'oklch(23.1% 0.019 249)',
    txL: 'oklch(23% 0.02 249)', txD: 'oklch(96.1% 0.008 237)',
    sfL: 'oklch(95.6% 0.008 242)', sfD: 'oklch(30.1% 0.022 251)',
    sf2L: 'oklch(92.4% 0.013 244)', sf2D: 'oklch(35.9% 0.025 251)',
    motionL: ['oklch(50% 0.13 258)', 'oklch(50% 0.12 226)', 'oklch(50% 0.11 295)'],
    motionD: ['oklch(74% 0.10 255)', 'oklch(74% 0.11 225)', 'oklch(74% 0.10 292)'],
  },
  lavender: {
    accentL: 'oklch(52% 0.11 295)', accentD: 'oklch(75% 0.11 295)', morandi: '164, 148, 190',
    bgL: 'oklch(97% 0.012 295)', bgD: 'oklch(26% 0.02 295)',
    txL: 'oklch(25% 0.02 295)', txD: 'oklch(96% 0.012 295)',
    sfL: 'oklch(95% 0.014 295)', sfD: 'oklch(32% 0.022 295)',
    sf2L: 'oklch(92.5% 0.016 295)', sf2D: 'oklch(38% 0.024 295)',
    motionL: ['oklch(52% 0.11 295)', 'oklch(52% 0.10 265)', 'oklch(53% 0.12 325)'],
    motionD: ['oklch(75% 0.11 295)', 'oklch(75% 0.11 265)', 'oklch(76% 0.12 325)'],
  },
  silverrose: {
    accentL: 'oklch(50% 0.10 320)', accentD: 'oklch(78% 0.10 320)', morandi: '168, 145, 170',
    bgL: 'oklch(97.5% 0.004 320)', bgD: 'oklch(23% 0.010 320)',
    txL: 'oklch(24% 0.012 320)', txD: 'oklch(96% 0.006 320)',
    sfL: 'oklch(95.3% 0.006 320)', sfD: 'oklch(30% 0.015 320)',
    sf2L: 'oklch(92% 0.008 320)', sf2D: 'oklch(36% 0.018 320)',
    motionL: ['oklch(50% 0.10 320)', 'oklch(51% 0.07 340)', 'oklch(50% 0.07 24)'],
    motionD: ['oklch(78% 0.10 320)', 'oklch(79% 0.08 340)', 'oklch(78% 0.08 24)'],
  },
  petal: {
    accentL: 'oklch(58% 0.22 350)', accentD: 'oklch(75% 0.18 350)', morandi: '232, 133, 155',
    bgL: 'oklch(98% 0.01 350)', bgD: 'oklch(28% 0.02 350)',
    txL: 'oklch(25% 0.02 354)', txD: 'oklch(98% 0.01 350)',
    sfL: 'oklch(96% 0.015 350)', sfD: 'oklch(34% 0.02 350)',
    sf2L: 'oklch(94% 0.015 350)', sf2D: 'oklch(40% 0.02 350)',
    motionL: ['oklch(56.7% 0.22 350)', 'oklch(56.7% 0.17 310)', 'oklch(56.7% 0.17 20)'],
    motionD: ['oklch(75% 0.18 350)', 'oklch(75% 0.14 310)', 'oklch(76% 0.14 20)'],
  },
} as const satisfies Record<ThemePreset, PaletteSpec>

export interface ThemePresetMeta {
  key: ThemePreset
  label: string
  desc: string
  lightColor: string
  color: string
  morandi: string
  family: PaletteFamily
  featured: boolean
}

/** Shared contrast endpoints and semantic states also live in the source layer. */
export const NEUTRAL_PALETTE = {
  black: 'oklch(0% 0 0)',
  white: 'oklch(100% 0 0)',
  darkInk: 'oklch(22% 0.006 250)',
} as const

export const STATE_PALETTE = {
  success: {
    light: 'oklch(45% 0.14 150)',
    dark: 'oklch(72% 0.12 150)',
  },
  warning: {
    light: 'oklch(45% 0.11 75)',
    dark: 'oklch(80% 0.12 75)',
  },
  warningLabel: {
    light: 'oklch(40% 0.09 75)',
    dark: 'oklch(88% 0.1 75)',
  },
  danger: {
    light: 'oklch(45% 0.16 25)',
    dark: 'oklch(72% 0.16 25)',
  },
} as const

const META_SEEDS: Omit<ThemePresetMeta, 'lightColor' | 'color' | 'morandi'>[] = [
  { key: 'cinnabar', label: '朱砂', desc: '日出江花红胜火 · 温润果决', family: 'red', featured: true },
  { key: 'dawn', label: '粉霞', desc: '晓看红湿处 · 柔和温暖', family: 'red', featured: true },
  { key: 'tangerine', label: '橙黄', desc: '最是橙黄橘绿时 · 明快专注', family: 'orange', featured: true },
  { key: 'aurora', label: '落霞', desc: '落霞与孤鹜齐飞 · 暖光流动', family: 'orange', featured: true },
  { key: 'stone', label: '赭石', desc: '秋山敛余照 · 沉稳克制', family: 'orange', featured: true },
  { key: 'amber', label: '琥珀', desc: '玉碗盛来琥珀光 · 温暖沉浸', family: 'orange', featured: true },
  { key: 'osmanthus', label: '轻黄', desc: '暗淡轻黄体性柔 · 安静明亮', family: 'yellow', featured: true },
  { key: 'mint', label: '草色', desc: '草色遥看近却无 · 清新舒展', family: 'green', featured: true },
  { key: 'sage', label: '竹青', desc: '绿竹猗猗 · 自然舒展', family: 'green', featured: true },
  { key: 'ripple', label: '天青', desc: '雨过天青云破处 · 清透专注', family: 'cyan', featured: true },
  { key: 'mist', label: '黛蓝', desc: '山色有无中 · 清冷静定', family: 'blue', featured: true },
  { key: 'lapis', label: '青金', desc: '碧海青天夜夜心 · 深邃沉静', family: 'blue', featured: true },
  { key: 'lavender', label: '青莲', desc: '清水出芙蓉 · 灵感轻盈', family: 'purple', featured: true },
  { key: 'silverrose', label: '银红', desc: '银烛秋光冷画屏 · 精致克制', family: 'purple', featured: true },
  { key: 'petal', label: '桃夭', desc: '灼灼其华 · 灵动柔和', family: 'pink', featured: true },
]

export const THEME_PRESETS: ThemePresetMeta[] = META_SEEDS.map((preset) => ({
  ...preset,
  lightColor: oklchToHex(PALETTE_SPECS[preset.key].accentL),
  color: oklchToHex(PALETTE_SPECS[preset.key].accentD),
  morandi: `rgb(${PALETTE_SPECS[preset.key].morandi})`,
}))

export const FEATURED_THEME_PRESETS = THEME_PRESETS.filter((preset) => preset.featured)
export const FEATURED_THEME_KEYS = FEATURED_THEME_PRESETS.map((preset) => preset.key)
export const THEME_PRESET_KEYS = THEME_PRESETS.map((preset) => preset.key)

export const THEME_PRESET_COLORS = Object.fromEntries(
  THEME_PRESETS.map((preset) => [preset.key, preset.color]),
) as Record<ThemePreset, string>

export const THEME_PRESET_LABELS = Object.fromEntries(
  THEME_PRESETS.map((preset) => [preset.key, preset.label]),
) as Record<ThemePreset, string>

export function isThemePreset(value: unknown): value is ThemePreset {
  return typeof value === 'string' && THEME_PRESET_KEYS.includes(value as ThemePreset)
}
