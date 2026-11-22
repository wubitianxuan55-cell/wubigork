import type { DictKey } from '../gaea/locales/en'
import type { ThemePreset } from './bloomPalette'

/** Shared locale keys for every surface that renders the Bloom palette catalogue. */
export const THEME_COPY_KEYS: Record<ThemePreset, { label: DictKey; desc: DictKey }> = {
  cinnabar: { label: 'settings.appear.themeCinnabarLabel', desc: 'settings.appear.themeCinnabarDesc' },
  dawn: { label: 'settings.appear.themeDawnLabel', desc: 'settings.appear.themeDawnDesc' },
  tangerine: { label: 'settings.appear.themeTangerineLabel', desc: 'settings.appear.themeTangerineDesc' },
  aurora: { label: 'settings.appear.themeAuroraLabel', desc: 'settings.appear.themeAuroraDesc' },
  stone: { label: 'settings.appear.themeStoneLabel', desc: 'settings.appear.themeStoneDesc' },
  amber: { label: 'settings.appear.themeAmberLabel', desc: 'settings.appear.themeAmberDesc' },
  osmanthus: { label: 'settings.appear.themeOsmanthusLabel', desc: 'settings.appear.themeOsmanthusDesc' },
  mint: { label: 'settings.appear.themeMintLabel', desc: 'settings.appear.themeMintDesc' },
  sage: { label: 'settings.appear.themeSageLabel', desc: 'settings.appear.themeSageDesc' },
  ripple: { label: 'settings.appear.themeRippleLabel', desc: 'settings.appear.themeRippleDesc' },
  mist: { label: 'settings.appear.themeMistLabel', desc: 'settings.appear.themeMistDesc' },
  lapis: { label: 'settings.appear.themeLapisLabel', desc: 'settings.appear.themeLapisDesc' },
  lavender: { label: 'settings.appear.themeLavenderLabel', desc: 'settings.appear.themeLavenderDesc' },
  silverrose: { label: 'settings.appear.themeSilverRoseLabel', desc: 'settings.appear.themeSilverRoseDesc' },
  petal: { label: 'settings.appear.themePetalLabel', desc: 'settings.appear.themePetalDesc' },
}

/** One-character palette marks used by the compact swatch grid. */
export const THEME_GLYPHS: Record<ThemePreset, string> = {
  cinnabar: '朱',
  dawn: '晓',
  tangerine: '橙',
  aurora: '霞',
  stone: '赭',
  amber: '琥',
  osmanthus: '桂',
  mint: '草',
  sage: '竹',
  ripple: '天',
  mist: '黛',
  lapis: '金',
  lavender: '莲',
  silverrose: '银',
  petal: '桃',
}

/** Keep the compact menu poetic first line; the full descriptor remains in `title`. */
export function themePoem(desc: string): string {
  return desc.split(/\s+·\s+/)[0] || desc
}
