import { afterEach, describe, expect, it } from 'vitest'
import {
  clampAutoScrollSpeed, clampBrightness, clampFontSize,
  DEFAULT_READING_SETTINGS, READING_COLUMN_WIDTH,
  readReadingSettings, writeReadingSettings,
} from './readingSettings'

const KEY = 'gaea.novel.readingSettings'

describe('readingSettings', () => {
  afterEach(() => {
    try { localStorage.removeItem(KEY) } catch { /* ignore */ }
  })

  it('默认：17px / 行距2 / 铺满 / 主题跟随 / 亮度100 / 滚屏3档', () => {
    expect(readReadingSettings()).toEqual(DEFAULT_READING_SETTINGS)
    expect(READING_COLUMN_WIDTH.wide).toBe('none')
    expect(DEFAULT_READING_SETTINGS.theme).toBe('auto')
    expect(DEFAULT_READING_SETTINGS.brightness).toBe(100)
    expect(DEFAULT_READING_SETTINGS.autoScrollSpeed).toBe(3)
  })

  it('round-trips 自定义偏好', () => {
    writeReadingSettings({ fontSize: 19, lineHeight: 1.8, column: 'narrow', theme: 'sepia', brightness: 85, autoScrollSpeed: 5 })
    expect(readReadingSettings()).toEqual({
      fontSize: 19, lineHeight: 1.8, column: 'narrow', theme: 'sepia', brightness: 85, autoScrollSpeed: 5,
    })
  })

  it('非法值回退默认并夹紧字号', () => {
    localStorage.setItem(KEY, JSON.stringify({ fontSize: 99, lineHeight: 5, column: 'huge', theme: 'pink', brightness: 200, autoScrollSpeed: 9 }))
    const v = readReadingSettings()
    expect(v.fontSize).toBe(24)
    expect(v.lineHeight).toBe(2)
    expect(v.column).toBe('wide')
    expect(v.theme).toBe('auto')
    expect(v.brightness).toBe(120)
    expect(v.autoScrollSpeed).toBe(5)
    expect(clampFontSize(9)).toBe(14)
    expect(clampFontSize(20.4)).toBe(20)
    expect(clampBrightness(55)).toBe(70)
    expect(clampAutoScrollSpeed(0)).toBe(1)
  })
})
