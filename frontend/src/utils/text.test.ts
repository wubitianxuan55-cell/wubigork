import { describe, expect, it } from 'vitest'
import { countTextChars } from './text'

describe('countTextChars', () => {
  it('counts BMP and CJK characters correctly', () => {
    expect(countTextChars('abc')).toBe(3)
    expect(countTextChars('林晚')).toBe(2)
    expect(countTextChars('第一章正文')).toBe(5)
  })

  it('counts surrogate pairs as a single character', () => {
    expect(countTextChars('😀a')).toBe(2)
  })
})
