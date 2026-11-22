import {
  NEUTRAL_PALETTE,
  PALETTE_SPECS,
  STATE_PALETTE,
  type ThemePreset,
} from './bloomPalette'
import {
  contrastRatio,
  hexToRgbTuple,
  mixOklchHex,
  oklchToHex,
  rgba,
} from './color'

export interface ThemeTokens {
  colorPrimary: string
  onPrimary: string
  primaryContainer: string
  onPrimaryContainer: string
  surface: string
  bodyBackground: string
  onSurface: string
  surfaceVariant: string
  onSurfaceVariant: string
  surfaceContainer: string
  surfaceContainerLow: string
  surfaceContainerHigh: string
  surfaceContainerHighest: string
  surfaceTint1: string
  surfaceTint2: string
  surfaceTint3: string
  surfaceDim: string
  surfaceBright: string
  outline: string
  outlineVariant: string
  colorBgContainer: string
  colorBgLayout: string
  colorText: string
  colorTextSecondary: string
  colorTextTertiary: string
  colorBorder: string
  colorSuccess: string
  colorWarning: string
  colorInfo: string
  colorDestructive: string
  accentReadable: string
  labelSecondary: string
  labelTertiary: string
  labelCaption: string
  labelDimmed: string
  border1: string
  border2: string
  border3: string
  border4: string
  bloomHairline: string
  bloomHairlineStrong: string
  bloomGlow: string
  bloomCodeBg: string
  stateSuccessTertiary: string
  stateWarningTertiary: string
  stateWarningLabel: string
  stateDangerSecondary: string
  stateDangerHoverBg: string
  businessTertiary: string
  sidebarNavActive: string
  sidebarNavActiveAccent: string
  sidebarNavHover: string
  elevation1: string
  elevation2: string
  elevation3: string
  elevation4: string
  elevation5: string
  radiusSm: string
  radiusMd: string
  radiusLg: string
  radiusXl: string
  transitionFast: string
  transitionNormal: string
  transitionSlow: string
  accentRgb: string
  ambienceRgb: string
  motion1: string
  motion2: string
  motion3: string
  glow: string
  glassBg: string
  auroraBg: string
  tertiaryContainer: string
  onTertiaryContainer: string
  shellSidebar: string
}

const BLACK = oklchToHex(NEUTRAL_PALETTE.black)
const WHITE = oklchToHex(NEUTRAL_PALETTE.white)
const DARK_INK = oklchToHex(NEUTRAL_PALETTE.darkInk)
const SUCCESS_DARK = oklchToHex(STATE_PALETTE.success.dark)
const SUCCESS_LIGHT = oklchToHex(STATE_PALETTE.success.light)
const WARNING_DARK = oklchToHex(STATE_PALETTE.warning.dark)
const WARNING_LIGHT = oklchToHex(STATE_PALETTE.warning.light)
const WARNING_LABEL_DARK = oklchToHex(STATE_PALETTE.warningLabel.dark)
const WARNING_LABEL_LIGHT = oklchToHex(STATE_PALETTE.warningLabel.light)
const ERROR_DARK = oklchToHex(STATE_PALETTE.danger.dark)
const ERROR_LIGHT = oklchToHex(STATE_PALETTE.danger.light)

function rgbTuple(hex: string): string {
  return hexToRgbTuple(hex).join(',')
}

function rgbTupleToHex(tuple: string): string {
  const channels = tuple.split(',').map((part) => Math.max(0, Math.min(255, Number(part.trim()) || 0)))
  return `#${channels.map((value) => Math.round(value).toString(16).padStart(2, '0')).join('')}`
}

function mixTransparent(color: string, transparentPercent: number): string {
  return `color-mix(in oklch, ${color}, transparent ${transparentPercent}%)`
}

function mixBackground(color: string, background: string, backgroundPercent: number): string {
  return `color-mix(in oklch, ${color}, ${background} ${backgroundPercent}%)`
}

function parseOklch(value: string): { l: number; c: number; h: number } | null {
  const match = /^oklch\(\s*([\d.]+)%\s+([\d.]+)\s+([\d.]+)\s*\)$/i.exec(value.trim())
  if (!match) return null
  return { l: Number(match[1]), c: Number(match[2]), h: Number(match[3]) }
}

/**
 * Follow Bloom's chroma-normalized tint rule: low-chroma palettes receive a
 * larger accent mix, high-chroma palettes a smaller one, so every theme has a
 * comparable perceived amount of colour in large surfaces.
 */
function mixPercentForTargetChroma(
  accent: string,
  base: string,
  targetChroma: number,
  fallback: number,
  min: number,
  max: number,
): number {
  const accentColor = parseOklch(accent)
  const baseColor = parseOklch(base)
  if (!accentColor || !baseColor) return fallback
  const denominator = accentColor.c - baseColor.c
  if (denominator <= 0.005) return fallback
  const percent = ((targetChroma - baseColor.c) / denominator) * 100
  return Math.min(max, Math.max(min, percent))
}

function ensureTextContrast(
  text: string,
  surfaces: string[],
  dark: boolean,
  white: string,
  darkInk: string,
): string {
  let candidate = text
  for (let step = 0; step < 12; step++) {
    if (surfaces.every((surface) => contrastRatio(candidate, surface) >= 4.5)) return candidate
    candidate = mixOklchHex(candidate, dark ? white : darkInk, 0.04)
  }
  return dark ? white : darkInk
}

function elevationTokens(dark: boolean, morandi: string): Pick<ThemeTokens,
  'elevation1' | 'elevation2' | 'elevation3' | 'elevation4' | 'elevation5'> {
  const tint = `rgba(${morandi},${dark ? 0.18 : 0.08})`
  const shade = dark ? 'rgba(0,0,0,0.24)' : 'rgba(35,42,44,0.055)'
  return {
    elevation1: `0 1px 2px ${shade}`,
    elevation2: `0 1px 2px ${shade}, 0 8px 22px ${tint}`,
    elevation3: `0 2px 4px ${shade}, 0 14px 34px ${tint}`,
    elevation4: `0 4px 10px ${shade}, 0 22px 54px ${tint}`,
    elevation5: `0 8px 18px ${shade}, 0 34px 84px ${tint}`,
  }
}

export function getBloomThemeTokens(base: ThemePreset, darkMode: boolean): ThemeTokens {
  const palette = PALETTE_SPECS[base]
  const accentSpec = darkMode ? palette.accentD : palette.accentL
  const surfaceSpec = darkMode ? palette.bgD : palette.bgL
  const containerSpec = darkMode ? palette.sfD : palette.sfL
  const containerHighSpec = darkMode ? palette.sf2D : palette.sf2L
  const accentRaw = oklchToHex(accentSpec)
  const rawSurface = oklchToHex(surfaceSpec)
  const rawContainer = oklchToHex(darkMode ? palette.sfD : palette.sfL)
  const rawContainerHigh = oklchToHex(darkMode ? palette.sf2D : palette.sf2L)
  const text = oklchToHex(darkMode ? palette.txD : palette.txL)
  const motion = (darkMode ? palette.motionD : palette.motionL).map(oklchToHex)
  const morandi = rgbTupleToHex(palette.morandi)
  const black = BLACK
  const white = WHITE
  // Match the latest Bloom token layer: tint every neutral surface with the
  // palette accent, but normalize the mix by target chroma rather than using
  // one fixed percentage. This keeps low-chroma themes visible and prevents
  // saturated themes from turning whole panels into flat colour blocks.
  const baseMix = mixPercentForTargetChroma(
    accentSpec,
    surfaceSpec,
    darkMode ? 0.018 : 0.012,
    darkMode ? 8 : 6,
    5,
    darkMode ? 22 : 18,
  )
  const layerMix = mixPercentForTargetChroma(
    accentSpec,
    containerSpec,
    darkMode ? 0.018 : 0.012,
    darkMode ? 8 : 6,
    5,
    darkMode ? 22 : 18,
  )
  const layerHighMix = mixPercentForTargetChroma(
    accentSpec,
    containerHighSpec,
    darkMode ? 0.018 : 0.012,
    darkMode ? 8 : 6,
    5,
    darkMode ? 22 : 18,
  )
  const sidebarMix = mixPercentForTargetChroma(
    accentSpec,
    surfaceSpec,
    darkMode ? 0.030 : 0.022,
    12,
    5,
    darkMode ? 34 : 30,
  )
  const surface = mixOklchHex(rawSurface, accentRaw, baseMix / 100)
  // Bloom paints the page canvas with the raw palette bg, then reintroduces
  // accent through the larger panel token layers and ambience gradients.
  const bodyBackground = rawSurface
  const container = mixOklchHex(rawContainer, accentRaw, layerMix / 100)
  const containerHigh = mixOklchHex(rawContainerHigh, accentRaw, layerHighMix / 100)
  const containerHighest = mixOklchHex(
    mixOklchHex(containerHigh, darkMode ? white : black, darkMode ? 0.075 : 0.045),
    morandi,
    darkMode ? 0.045 : 0.022,
  )
  const surfaceContainerLow = mixOklchHex(surface, container, darkMode ? 0.52 : 0.46)
  const surfaceTint1 = container
  const surfaceTint2 = containerHigh
  const surfaceTint3 = containerHighest
  const shellSidebar = mixOklchHex(rawSurface, accentRaw, sidebarMix / 100)
  // Upstream keeps a separate readable accent track: dark uses accentD as-is,
  // light mixes accentL with 12% black before it is used as UI ink.
  const accent = darkMode ? accentRaw : mixOklchHex(accentRaw, black, 0.12)
  const textSurfaces = [surface, container, containerHigh, containerHighest, shellSidebar]
  const secondary = ensureTextContrast(
    mixOklchHex(text, rawSurface, darkMode ? 0.30 : 0.28),
    textSurfaces,
    darkMode,
    white,
    DARK_INK,
  )
  const tertiary = ensureTextContrast(
    mixOklchHex(text, rawSurface, darkMode ? 0.35 : 0.32),
    textSurfaces,
    darkMode,
    white,
    DARK_INK,
  )
  const labelSecondary = mixTransparent(text, darkMode ? 30 : 28)
  const labelTertiary = mixTransparent(text, darkMode ? 35 : 32)
  const labelCaption = mixTransparent(text, darkMode ? 40 : 35)
  const labelDimmed = mixTransparent(text, darkMode ? 43 : 37)
  const border1 = darkMode ? rgba(morandi, 0.10) : mixTransparent(text, 90)
  const border2 = darkMode ? rgba(morandi, 0.18) : mixTransparent(text, 80)
  const border3 = darkMode ? rgba(morandi, 0.28) : mixTransparent(text, 70)
  const border4 = darkMode ? rgba(morandi, 0.40) : mixTransparent(text, 60)
  const success = darkMode ? SUCCESS_DARK : SUCCESS_LIGHT
  const warning = darkMode ? WARNING_DARK : WARNING_LIGHT
  const destructive = darkMode ? ERROR_DARK : ERROR_LIGHT
  const stateSuccessTertiary = mixBackground(success, rawSurface, darkMode ? 78 : 88)
  const stateWarningTertiary = mixBackground(warning, rawSurface, darkMode ? 78 : 88)
  const stateWarningLabel = darkMode ? WARNING_LABEL_DARK : WARNING_LABEL_LIGHT
  const stateDangerHoverBg = mixTransparent(destructive, darkMode ? 86 : 90)
  const businessTertiary = mixBackground(accent, rawSurface, darkMode ? 78 : 88)
  const accentContainer = mixOklchHex(accent, rawSurface, darkMode ? 0.78 : 0.88)
  const tertiaryContainer = mixOklchHex(motion[1], surface, darkMode ? 0.80 : 0.86)
  const auroraBg = darkMode
    ? [
        `radial-gradient(1200px circle at 8% -6%, ${rgba(morandi, 0.12)}, transparent 58%)`,
        `radial-gradient(1000px circle at 95% 0%, ${rgba(morandi, 0.085)}, transparent 60%)`,
        `radial-gradient(900px circle at 96% 100%, ${rgba(morandi, 0.065)}, transparent 64%)`,
        `linear-gradient(155deg, ${surface} 0%, ${mixOklchHex(surface, container, 0.40)} 54%, ${mixOklchHex(surface, black, 0.075)} 100%)`,
      ].join(', ')
    : [
        `linear-gradient(180deg, ${mixOklchHex(surface, white, 0.34)} 0%, ${surface} 44%, ${mixOklchHex(surface, white, 0.48)} 100%)`,
        `radial-gradient(1100px circle at 94% -8%, ${rgba(morandi, 0.075)}, transparent 64%)`,
        `radial-gradient(900px circle at 8% 10%, ${rgba(morandi, 0.045)}, transparent 66%)`,
      ].join(', ')

  return {
    colorPrimary: accent,
    onPrimary: bodyBackground,
    primaryContainer: accentContainer,
    onPrimaryContainer: text,
    surface,
    bodyBackground,
    onSurface: text,
    surfaceVariant: mixOklchHex(surface, container, 0.70),
    onSurfaceVariant: secondary,
    surfaceContainer: container,
    surfaceContainerLow,
    surfaceContainerHigh: containerHigh,
    surfaceContainerHighest: containerHighest,
    surfaceTint1,
    surfaceTint2,
    surfaceTint3,
    surfaceDim: mixOklchHex(surface, black, darkMode ? 0.14 : 0.04),
    surfaceBright: mixOklchHex(surface, white, darkMode ? 0.14 : 0.28),
    outline: border2,
    outlineVariant: border1,
    colorBgContainer: surface,
    colorBgLayout: surface,
    colorText: text,
    colorTextSecondary: secondary,
    colorTextTertiary: tertiary,
    colorBorder: border1,
    colorSuccess: success,
    colorWarning: warning,
    colorInfo: accent,
    colorDestructive: destructive,
    accentReadable: accent,
    labelSecondary,
    labelTertiary,
    labelCaption,
    labelDimmed,
    border1,
    border2,
    border3,
    border4,
    bloomHairline: rgba(morandi, 0.30),
    bloomHairlineStrong: rgba(morandi, darkMode ? 0.55 : 0.50),
    bloomGlow: rgba(morandi, darkMode ? 0.34 : 0.20),
    bloomCodeBg: rgba(morandi, darkMode ? 0.16 : 0.13),
    stateSuccessTertiary,
    stateWarningTertiary,
    stateWarningLabel,
    stateDangerSecondary: destructive,
    stateDangerHoverBg,
    businessTertiary,
    sidebarNavActive: mixTransparent(accent, darkMode ? 88 : 93),
    sidebarNavActiveAccent: mixTransparent(accent, darkMode ? 80 : 86),
    sidebarNavHover: mixTransparent(text, 95),
    ...elevationTokens(darkMode, palette.morandi),
    radiusSm: '8px',
    radiusMd: '12px',
    radiusLg: '16px',
    radiusXl: '24px',
    transitionFast: '160ms cubic-bezier(0.2, 0.8, 0.2, 1)',
    transitionNormal: '240ms cubic-bezier(0.2, 0.8, 0.2, 1)',
    transitionSlow: '320ms cubic-bezier(0.2, 0.8, 0.2, 1)',
    accentRgb: rgbTuple(accent),
    ambienceRgb: palette.morandi.replace(/\s+/g, ''),
    motion1: motion[0],
    motion2: motion[1],
    motion3: motion[2],
    glow: accent,
    glassBg: rgba(container, darkMode ? 0.74 : 0.68),
    auroraBg,
    tertiaryContainer,
    onTertiaryContainer: text,
    shellSidebar,
  }
}
