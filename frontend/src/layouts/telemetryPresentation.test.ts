import { describe, expect, it } from 'vitest'
import {
  telemetryMeterWidth,
  telemetrySystemTone,
  telemetryTone,
  telemetryWorstTone,
} from './telemetryPresentation'

describe('telemetry presentation thresholds', () => {
  it('splits sampled values into normal, watch, and hot tones', () => {
    expect(telemetryTone(42)).toBe('normal')
    expect(telemetryTone(70)).toBe('watch')
    expect(telemetryTone(85)).toBe('hot')
  })

  it('keeps missing and invalid samples idle', () => {
    expect(telemetryTone(null)).toBe('idle')
    expect(telemetryTone(undefined)).toBe('idle')
    expect(telemetryTone(Number.NaN)).toBe('idle')
    expect(telemetrySystemTone([null, undefined])).toBe('idle')
  })

  it('uses the highest sampled metric for the system tone', () => {
    expect(telemetrySystemTone([12, 76, null])).toBe('watch')
    expect(telemetrySystemTone([12, 91, 55])).toBe('hot')
    expect(telemetrySystemTone([12, 55, 61])).toBe('normal')
  })

  it('reduces mixed metric tones to the most severe tone', () => {
    expect(telemetryWorstTone(['idle', 'normal', 'idle'])).toBe('normal')
    expect(telemetryWorstTone(['normal', 'watch', 'hot'])).toBe('hot')
  })

  it('clamps quick-meter widths to the visible range', () => {
    expect(telemetryMeterWidth(null)).toBe(0)
    expect(telemetryMeterWidth(-4)).toBe(0)
    expect(telemetryMeterWidth(48)).toBe(48)
    expect(telemetryMeterWidth(140)).toBe(100)
  })
})
