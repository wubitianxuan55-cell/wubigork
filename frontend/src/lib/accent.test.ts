/* eslint-disable local/no-raw-hex -- 被测数据色值，非 UI 声明处 */
import { describe, expect, it } from 'vitest'
import { contrastOnWhite, ensureLightContrast, hexToRgb256 } from './accent'

describe('hexToRgb256', () => {
  it('解析 6 位 hex', () => {
    expect(hexToRgb256('#1dd7bf')).toEqual([29, 215, 191])
    expect(hexToRgb256('FF8800')).toEqual([255, 136, 0])
  })
  it('非法输入返回 null', () => {
    expect(hexToRgb256('')).toBeNull()
    expect(hexToRgb256('#12345')).toBeNull()
    expect(hexToRgb256('not-a-color')).toBeNull()
  })
})

describe('contrastOnWhite', () => {
  it('已知锚点：白=1、黑=21、teal-700≈4.9', () => {
    expect(contrastOnWhite('#ffffff')).toBeCloseTo(1, 2)
    expect(contrastOnWhite('#000000')).toBeCloseTo(21, 1)
    expect(contrastOnWhite('#0f766e')).toBeGreaterThan(4.5)
  })
})

describe('ensureLightContrast', () => {
  it('已达标色原样返回（零视觉变化）', () => {
    expect(ensureLightContrast('#0f766e')).toBe('#0f766e')
    expect(ensureLightContrast('#134e4a')).toBe('#134e4a')
  })
  it('真实案例 #1dd7bf（用户自定义亮青）：深化后对白底 ≥4.5，且是同色系（青）', () => {
    const out = ensureLightContrast('#1dd7bf')
    expect(contrastOnWhite(out)).toBeGreaterThanOrEqual(4.5)
    // 仍是青绿系：g 通道为最大分量
    const rgb = hexToRgb256(out)!
    expect(rgb[1]).toBeGreaterThan(rgb[0])
    expect(rgb[1]).toBeGreaterThan(rgb[2])
  })
  it('暗态亮色 #5eead4 深化后达标', () => {
    const out = ensureLightContrast('#5eead4')
    expect(contrastOnWhite(out)).toBeGreaterThanOrEqual(4.5)
  })
  it('非法输入原样返回', () => {
    expect(ensureLightContrast('nope')).toBe('nope')
  })
})
