/* eslint-disable local/no-raw-hex -- conversion fixtures */
import { describe, expect, it } from 'vitest'
import { contrastRatio, oklchToHex } from './color'

describe('Bloom OKLCH conversion', () => {
  it('converts representative accent values to sRGB hex', () => {
    expect(oklchToHex('oklch(75% 0.12 195)')).toBe('#2cc5c5')
    expect(oklchToHex('oklch(50% 0.13 258)')).toBe('#2f62ac')
  })

  it('computes WCAG contrast for generated colours', () => {
    expect(contrastRatio('#ffffff', '#000000')).toBeCloseTo(21, 5)
  })
})
