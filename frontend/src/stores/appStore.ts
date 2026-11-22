import { wailsApp } from '../lib/wailsApp'
import { create } from 'zustand'
import { isShellSpace, type ShellSpace } from '../boards/space'
import { emitFrontendEvent, FRONTEND_EVENTS } from '../events'
import {
  FEATURED_THEME_KEYS,
  FEATURED_THEME_PRESETS,
  THEME_PRESET_COLORS,
  THEME_PRESET_KEYS,
  THEME_PRESET_LABELS,
  THEME_PRESETS,
  isThemePreset,
  type ThemePreset,
  type ThemePresetMeta,
} from '../theme/bloomPalette'
import { getBloomThemeTokens, type ThemeTokens } from '../theme/tokens'

export {
  FEATURED_THEME_KEYS,
  FEATURED_THEME_PRESETS,
  THEME_PRESET_COLORS,
  THEME_PRESET_KEYS,
  THEME_PRESET_LABELS,
  THEME_PRESETS,
  type ThemePreset,
  type ThemePresetMeta,
  type ThemeTokens,
}

export function getThemeTokens(base: ThemePreset, darkMode: boolean): ThemeTokens {
  return getBloomThemeTokens(base, darkMode)
}

const THEME_KEY = 'gaea-theme'
const DARK_KEY = 'gaea-dark'
const MODE_KEY = 'gaea-display-mode'
const DENSITY_KEY = 'gaea-density'
const MOTION_KEY = 'gaea-motion'
const ACCENT_KEY = 'gaea-accent'
const FONT_KEY = 'gaea-font-family'
const FONT_SIZE_KEY = 'gaea-font-size'
const SURFACE_KEY = 'gaea-surface-style'
const SURFACE_RESTORE_KEY = 'gaea-surface-style-restore'
const AMBIENCE_KEY = 'gaea-ambience-style'
const SHELL_SPACE_KEY = 'gaea.shell.space'

/** Preset UI font families. */
export const FONT_OPTIONS: { key: string; label: string; value: string }[] = [
  { key: 'system', label: '系统默认', value: "system-ui, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif" },
  { key: 'yahei', label: '微软雅黑', value: "'Microsoft YaHei', 'PingFang SC', 'Segoe UI', sans-serif" },
  { key: 'noto', label: '思源黑体', value: "'Noto Sans SC', 'Source Han Sans SC', 'Microsoft YaHei', sans-serif" },
  { key: 'songti', label: '宋体衬线', value: "'SimSun', 'Songti SC', 'Noto Serif SC', serif" },
  { key: 'mono', label: '等宽字体', value: "'Cascadia Code', 'Consolas', 'JetBrains Mono', monospace" },
]

function loadFontFamily(): string {
  try {
    const value = localStorage.getItem(FONT_KEY)
    if (value && FONT_OPTIONS.some((option) => option.key === value)) return value
  } catch {}
  return 'system'
}

function loadFontSize(): number {
  try {
    const value = Number.parseInt(localStorage.getItem(FONT_SIZE_KEY) || '', 10)
    if (value >= 12 && value <= 20) return value
  } catch {}
  return 14
}

export type Density = 'standard' | 'compact'
export type MotionPref = 'full' | 'reduced'
export type SurfaceStyle = 'veil' | 'soft' | 'solid'
export type SurfaceBaseStyle = Exclude<SurfaceStyle, 'soft'>
export type AmbienceStyle = 'bloom' | 'wash' | 'grid' | 'off'

function loadDensity(): Density {
  try {
    const value = localStorage.getItem(DENSITY_KEY)
    if (value === 'compact' || value === 'standard') return value
  } catch {}
  return 'standard'
}

function loadMotion(): MotionPref {
  try {
    const value = localStorage.getItem(MOTION_KEY)
    if (value === 'reduced' || value === 'full') return value
  } catch {}
  return 'full'
}

function loadSurfaceStyle(): SurfaceStyle {
  try {
    const value = localStorage.getItem(SURFACE_KEY)
    if (value === 'veil' || value === 'soft' || value === 'solid') return value
  } catch {}
  return 'veil'
}

function loadSurfaceStyleBeforeGlass(): SurfaceBaseStyle {
  try {
    const value = localStorage.getItem(SURFACE_RESTORE_KEY)
    if (value === 'veil' || value === 'solid') return value
  } catch {}
  return 'veil'
}

function loadAmbienceStyle(): AmbienceStyle {
  try {
    const value = localStorage.getItem(AMBIENCE_KEY)
    if (value === 'bloom' || value === 'wash' || value === 'grid' || value === 'off') return value
  } catch {}
  return 'grid'
}

function loadAccent(): string {
  try {
    return localStorage.getItem(ACCENT_KEY) || ''
  } catch {
    return ''
  }
}

function loadShellSpace(): ShellSpace {
  try {
    const value = localStorage.getItem(SHELL_SPACE_KEY)
    if (isShellSpace(value)) return value
  } catch {}
  return 'work'
}

export type DisplayMode = 'light' | 'dark' | 'system'

function resolveDark(mode: DisplayMode, systemDark: boolean): boolean {
  return mode === 'system' ? systemDark : mode === 'dark'
}

function systemPrefersDark(): boolean {
  try {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  } catch {
    return true
  }
}

function loadMode(): DisplayMode {
  try {
    const value = localStorage.getItem(MODE_KEY)
    if (value === 'light' || value === 'dark' || value === 'system') return value
    const legacy = localStorage.getItem(DARK_KEY)
    if (legacy === '0') return 'light'
  } catch {}
  return 'dark'
}

function loadBaseTheme(): ThemePreset {
  try {
    const value = localStorage.getItem(THEME_KEY)
    if (isThemePreset(value)) return value
    if (value) localStorage.removeItem(THEME_KEY)
  } catch {}
  return 'ripple'
}

interface AppState {
  loggedIn: boolean
  projectPath: string
  baseTheme: ThemePreset
  mode: DisplayMode
  systemDark: boolean
  darkMode: boolean
  density: Density
  motion: MotionPref
  surfaceStyle: SurfaceStyle
  surfaceStyleBeforeGlass: SurfaceBaseStyle
  ambienceStyle: AmbienceStyle
  accentColor: string
  fontFamily: string
  fontSize: number
  space: ShellSpace
  login: () => Promise<void>
  logout: () => Promise<void>
  checkLogin: () => Promise<void>
  setLoggedIn: (value: boolean) => void
  setTheme: (base: ThemePreset) => void
  setMode: (mode: DisplayMode) => void
  toggleDarkMode: () => void
  setDensity: (density: Density) => void
  setMotion: (motion: MotionPref) => void
  setSurfaceStyle: (style: SurfaceStyle) => void
  toggleGlassMode: () => void
  setAmbienceStyle: (style: AmbienceStyle) => void
  setAccentColor: (color: string) => void
  setFontFamily: (font: string) => void
  setFontSize: (size: number) => void
  setSpace: (space: ShellSpace) => void
  pendingSettingsCategory: string | null
  pendingSettingsTab: string | null
  openSettings: (category?: string, tab?: string) => void
  clearPendingSettingsCategory: () => void
}

export const useAppStore = create<AppState>((set, get) => ({
  loggedIn: false,
  projectPath: '',
  baseTheme: loadBaseTheme(),
  mode: loadMode(),
  systemDark: systemPrefersDark(),
  darkMode: resolveDark(loadMode(), systemPrefersDark()),
  density: loadDensity(),
  motion: loadMotion(),
  surfaceStyle: loadSurfaceStyle(),
  surfaceStyleBeforeGlass: loadSurfaceStyleBeforeGlass(),
  ambienceStyle: loadAmbienceStyle(),
  accentColor: loadAccent(),
  fontFamily: loadFontFamily(),
  fontSize: loadFontSize(),
  space: loadShellSpace(),

  setTheme: (base) => {
    if (!isThemePreset(base)) return
    set({ baseTheme: base })
    try {
      localStorage.setItem(THEME_KEY, base)
    } catch {}
  },

  toggleDarkMode: () => {
    const next = !get().darkMode
    set({ mode: next ? 'dark' : 'light', darkMode: next })
    try {
      localStorage.setItem(MODE_KEY, next ? 'dark' : 'light')
      localStorage.setItem(DARK_KEY, next ? '1' : '0')
    } catch {}
  },

  setMode: (mode) => {
    set({ mode, darkMode: resolveDark(mode, get().systemDark) })
    try {
      localStorage.setItem(MODE_KEY, mode)
    } catch {}
  },

  setDensity: (density) => {
    set({ density })
    try {
      localStorage.setItem(DENSITY_KEY, density)
    } catch {}
  },

  setMotion: (motion) => {
    set({ motion })
    try {
      localStorage.setItem(MOTION_KEY, motion)
    } catch {}
  },

  setSurfaceStyle: (surfaceStyle) => {
    const beforeGlass = surfaceStyle === 'soft' ? get().surfaceStyleBeforeGlass : surfaceStyle
    set({ surfaceStyle, surfaceStyleBeforeGlass: beforeGlass })
    try {
      localStorage.setItem(SURFACE_KEY, surfaceStyle)
      localStorage.setItem(SURFACE_RESTORE_KEY, beforeGlass)
    } catch {}
  },

  toggleGlassMode: () => {
    const current = get()
    if (current.surfaceStyle === 'soft') {
      const restore = current.surfaceStyleBeforeGlass
      set({ surfaceStyle: restore })
      try {
        localStorage.setItem(SURFACE_KEY, restore)
      } catch {}
      return
    }
    set({
      surfaceStyle: 'soft',
      surfaceStyleBeforeGlass: current.surfaceStyle,
    })
    try {
      localStorage.setItem(SURFACE_KEY, 'soft')
      localStorage.setItem(SURFACE_RESTORE_KEY, current.surfaceStyle)
    } catch {}
  },

  setAmbienceStyle: (ambienceStyle) => {
    set({ ambienceStyle })
    try {
      localStorage.setItem(AMBIENCE_KEY, ambienceStyle)
    } catch {}
  },

  setAccentColor: (color) => {
    set({ accentColor: color })
    try {
      if (color) localStorage.setItem(ACCENT_KEY, color)
      else localStorage.removeItem(ACCENT_KEY)
    } catch {}
  },

  setFontFamily: (font) => {
    set({ fontFamily: font })
    try {
      localStorage.setItem(FONT_KEY, font)
    } catch {}
  },

  setFontSize: (size) => {
    set({ fontSize: size })
    try {
      localStorage.setItem(FONT_SIZE_KEY, String(size))
    } catch {}
  },

  setSpace: (space) => {
    set({ space })
    try {
      localStorage.setItem(SHELL_SPACE_KEY, space)
    } catch {}
  },

  login: async () => {
    try {
      await wailsApp().Login()
      for (let attempt = 0; attempt < 75; attempt++) {
        await new Promise((resolve) => setTimeout(resolve, 4000))
        try {
          if (await wailsApp().GetLoginStatus()) {
            set({ loggedIn: true })
            return
          }
        } catch {}
      }
      throw new Error('登录超时：请检查浏览器是否完成了 xAI 授权')
    } catch (error) {
      console.error('login 失败:', error)
      throw error
    }
  },

  checkLogin: async () => {
    try {
      set({ loggedIn: await wailsApp().GetLoginStatus() })
    } catch {}
  },

  logout: async () => {
    try {
      await wailsApp().Logout()
    } catch {}
    set({ loggedIn: false })
  },

  setLoggedIn: (loggedIn) => set({ loggedIn }),

  pendingSettingsCategory: null,
  pendingSettingsTab: null,

  openSettings: (category, tab) => {
    emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: 'settings' })
    set({ pendingSettingsCategory: category ?? null, pendingSettingsTab: tab ?? null })
  },

  clearPendingSettingsCategory: () => set({ pendingSettingsCategory: null, pendingSettingsTab: null }),
}))

if (typeof window !== 'undefined' && window.matchMedia) {
  const media = window.matchMedia('(prefers-color-scheme: dark)')
  const handler = (event: MediaQueryListEvent) => {
    useAppStore.setState((state) => ({
      systemDark: event.matches,
      darkMode: state.mode === 'system' ? event.matches : state.darkMode,
    }))
  }
  if (typeof media.addEventListener === 'function') media.addEventListener('change', handler)
}
