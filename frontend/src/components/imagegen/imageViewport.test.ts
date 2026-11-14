import { describe, expect, it } from 'vitest'
import {
  ZOOM_MAX,
  ZOOM_MIN,
  centerView,
  clampZoom,
  fitZoom,
  formatZoom,
  imagePointAt,
  stepZoom,
  tickStep,
  ticksFor,
  zoomAtPoint,
} from './imageViewport'

describe('imageViewport', () => {
  it('clamps zoom to the supported range', () => {
    expect(clampZoom(0)).toBe(ZOOM_MIN)
    expect(clampZoom(100)).toBe(ZOOM_MAX)
    expect(clampZoom(Number.NaN)).toBe(1)
  })

  it('fits without upscaling and centers the image', () => {
    expect(fitZoom(2000, 1000, 800, 600)).toBeCloseTo(0.376, 3)
    expect(fitZoom(200, 100, 800, 600)).toBe(1)
    expect(centerView(200, 100, 800, 600, 1)).toEqual({ zoom: 1, x: 300, y: 250 })
  })

  it('keeps the image point under the cursor fixed while zooming', () => {
    const next = zoomAtPoint({ zoom: 1, x: 20, y: 10 }, 2, 220, 110)
    expect(next).toEqual({ zoom: 2, x: -180, y: -90 })
    expect(imagePointAt(next, 220, 110)).toEqual({ x: 200, y: 100 })
  })

  it('chooses readable ticks and returns only visible ruler marks', () => {
    expect(tickStep(1)).toBe(100)
    expect(tickStep(0.25)).toBe(500)
    const view = { zoom: 1, x: -100, y: 0 }
    const ticks = ticksFor(view, 500, 300)
    expect(ticks.step).toBe(100)
    expect(ticks.ticks).toEqual([
      { pos: 0, label: 100 },
      { pos: 100, label: 200 },
      { pos: 200, label: 300 },
      { pos: 300, label: 400 },
    ])
  })

  it('formats and steps zoom predictably', () => {
    expect(formatZoom(0.333)).toBe('33%')
    expect(stepZoom(1, 1)).toBe(1.25)
    expect(stepZoom(1, -1)).toBe(0.8)
  })
})
