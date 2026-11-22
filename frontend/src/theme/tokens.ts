import {
  NEUTRAL_PALETTE,
  PALETTE_SPECS,
  STATE_PALETTE,
  type ThemePreset,
} from './bloomPalette'
import {
  contrastRatio,
  hexToRgbTuple,
  mixHex,
  oklchToHex,
  rgba,
} from './color'

export interface ThemeTokens {
  colorPrimary: string
  onPrimary: string
  primaryContainer: string
  onPrimaryContainer: string
  surface: string
  onSurface: string
  surfaceVariant: string
  onSurfaceVariant: string
  surfaceContainer: string
  surfaceContainerHigh: string
  surfaceContainerHighest: string
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
const ERROR_DARK = oklchToHex(STATE_PALETTE.danger.dark)
const ERROR_LIGHT = oklchToHex(STATE_PALETTE.danger.light)

function readableForeground(background: string): string {
  return contrastRatio(background, WHITE) >= contrastRatio(background, DARK_INK)
    ? WHITE
    : DARK_INK
}

function rgbTuple(hex: string): string {
  return hexToRgbTuple(hex).join(',')
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
  const accent = oklchToHex(darkMode ? palette.accentD : palette.accentL)
  const surface = oklchToHex(darkMode ? palette.bgD : palette.bgL)
  const container = oklchToHex(darkMode ? palette.sfD : palette.sfL)
  const containerHigh = oklchToHex(darkMode ? palette.sf2D : palette.sf2L)
  const text = oklchToHex(darkMode ? palette.txD : palette.txL)
  const motion = (darkMode ? palette.motionD : palette.motionL).map(oklchToHex)
  const black = BLACK
  const white = WHITE
  const secondary = mixHex(text, surface, darkMode ? 0.24 : 0.22)
  const tertiary = mixHex(text, surface, darkMode ? 0.34 : 0.32)
  const containerHighest = mixHex(containerHigh, darkMode ? white : black, 0.05)
  const shellSidebar = darkMode ? mixHex(surface, black, 0.10) : mixHex(surface, white, 0.34)
  const accentContainer = mixHex(accent, surface, darkMode ? 0.68 : 0.86)
  const tertiaryContainer = mixHex(motion[1], surface, darkMode ? 0.80 : 0.86)
  const auroraBg = darkMode
    ? [
        `radial-gradient(1200px circle at 8% -6%, color-mix(in oklab, ${motion[0]} 22%, transparent), transparent 52%)`,
        `radial-gradient(1000px circle at 95% 0%, color-mix(in oklab, ${motion[1]} 18%, transparent), transparent 56%)`,
        `radial-gradient(900px circle at 96% 100%, color-mix(in oklab, ${motion[2]} 15%, transparent), transparent 60%)`,
        `linear-gradient(155deg, ${surface} 0%, ${mixHex(surface, container, 0.48)} 54%, ${mixHex(surface, black, 0.10)} 100%)`,
      ].join(', ')
    : [
        `linear-gradient(180deg, ${mixHex(motion[0], white, 0.82)} 0%, ${surface} 44%, ${mixHex(surface, white, 0.62)} 100%)`,
        `radial-gradient(1100px circle at 94% -8%, color-mix(in oklab, ${motion[1]} 16%, transparent), transparent 60%)`,
        `radial-gradient(900px circle at 8% 10%, color-mix(in oklab, ${motion[2]} 11%, transparent), transparent 62%)`,
      ].join(', ')

  return {
    colorPrimary: accent,
    onPrimary: readableForeground(accent),
    primaryContainer: accentContainer,
    onPrimaryContainer: mixHex(text, accent, 0.08),
    surface,
    onSurface: text,
    surfaceVariant: mixHex(surface, container, 0.70),
    onSurfaceVariant: secondary,
    surfaceContainer: container,
    surfaceContainerHigh: containerHigh,
    surfaceContainerHighest: containerHighest,
    surfaceDim: mixHex(surface, black, darkMode ? 0.14 : 0.04),
    surfaceBright: mixHex(surface, white, darkMode ? 0.14 : 0.28),
    outline: rgba(motion[0], darkMode ? 0.28 : 0.22),
    outlineVariant: rgba(motion[2], darkMode ? 0.13 : 0.10),
    colorBgContainer: surface,
    colorBgLayout: surface,
    colorText: text,
    colorTextSecondary: secondary,
    colorTextTertiary: tertiary,
    colorBorder: rgba(text, darkMode ? 0.12 : 0.09),
    colorSuccess: darkMode ? SUCCESS_DARK : SUCCESS_LIGHT,
    colorWarning: darkMode ? WARNING_DARK : WARNING_LIGHT,
    colorInfo: mixHex(motion[2], text, darkMode ? 0.08 : 0.18),
    colorDestructive: darkMode ? ERROR_DARK : ERROR_LIGHT,
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
    glow: accent,
    glassBg: rgba(container, darkMode ? 0.74 : 0.68),
    auroraBg,
    tertiaryContainer,
    onTertiaryContainer: mixHex(text, tertiaryContainer, 0.10),
    shellSidebar,
  }
}
