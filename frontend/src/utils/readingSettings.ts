/**
 * 阅读模式排版偏好（全局持久化，跨小说项目生效）
 *  - fontSize: 14–24，默认 17
 *  - lineHeight: 1.8 / 2 / 2.3，默认 2
 *  - column: narrow(38rem) / standard(52rem) / wide(铺满)，默认 wide
 *  - theme: auto(跟随应用) / sepia(米黄护眼) / green(护眼绿) / dark(夜间)
 *  - brightness: 70–120%，默认 100
 *  - autoScrollSpeed: 1–5（自动滚屏速度档），默认 3
 */
export type ReadingColumn = 'narrow' | 'standard' | 'wide'
export type ReadingTheme = 'auto' | 'sepia' | 'green' | 'dark'

export interface ReadingSettings {
  fontSize: number
  lineHeight: number
  column: ReadingColumn
  theme: ReadingTheme
  brightness: number
  autoScrollSpeed: number
}

export const READING_SETTINGS_KEY = 'gaea.novel.readingSettings'

export const READING_COLUMN_WIDTH: Record<ReadingColumn, string> = {
  narrow: '38rem',
  standard: '52rem',
  wide: 'none',
}

export const DEFAULT_READING_SETTINGS: ReadingSettings = {
  fontSize: 17,
  lineHeight: 2,
  column: 'wide',
  theme: 'auto',
  brightness: 100,
  autoScrollSpeed: 3,
}

export function clampFontSize(n: number): number {
  return Math.min(24, Math.max(14, Math.round(n)))
}

export function clampBrightness(n: number): number {
  return Math.min(120, Math.max(70, Math.round(n)))
}

export function clampAutoScrollSpeed(n: number): number {
  return Math.min(5, Math.max(1, Math.round(n)))
}

export function readReadingSettings(): ReadingSettings {
  try {
    const raw = localStorage.getItem(READING_SETTINGS_KEY)
    if (!raw) return { ...DEFAULT_READING_SETTINGS }
    const v = JSON.parse(raw) as Partial<ReadingSettings>
    return {
      fontSize: clampFontSize(typeof v.fontSize === 'number' ? v.fontSize : DEFAULT_READING_SETTINGS.fontSize),
      lineHeight: v.lineHeight === 1.8 || v.lineHeight === 2 || v.lineHeight === 2.3
        ? v.lineHeight
        : DEFAULT_READING_SETTINGS.lineHeight,
      column: v.column === 'narrow' || v.column === 'standard' || v.column === 'wide'
        ? v.column
        : DEFAULT_READING_SETTINGS.column,
      theme: v.theme === 'sepia' || v.theme === 'green' || v.theme === 'dark'
        ? v.theme
        : DEFAULT_READING_SETTINGS.theme,
      brightness: clampBrightness(typeof v.brightness === 'number'
        ? v.brightness
        : DEFAULT_READING_SETTINGS.brightness),
      autoScrollSpeed: clampAutoScrollSpeed(typeof v.autoScrollSpeed === 'number'
        ? v.autoScrollSpeed
        : DEFAULT_READING_SETTINGS.autoScrollSpeed),
    }
  } catch {
    return { ...DEFAULT_READING_SETTINGS }
  }
}

export function writeReadingSettings(s: ReadingSettings) {
  try {
    localStorage.setItem(READING_SETTINGS_KEY, JSON.stringify(s))
  } catch { /* ignore */ }
}
