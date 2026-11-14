import { describe, expect, it } from 'vitest'
import { EXPORT_PALETTE_MAP, applyExportPalette } from './exportPalette'
import { EXP_COLORS } from './ganttExport'
import { toSvgPalette } from './displayPreset'

const SAMPLE = `<svg width="100" height="40"><rect fill="${EXP_COLORS.critical}"/><rect fill="${EXP_COLORS.bar}"/><path stroke="${EXP_COLORS.link}"/><rect fill="${EXP_COLORS.weekend}"/><rect stroke="${EXP_COLORS.grid}"/><text fill="${EXP_COLORS.ink}">x</text></svg>`

describe('导出配色映射（W6 预设 → 纸面档）', () => {
  it('normal 原样返回（零改动）', () => {
    expect(applyExportPalette(SAMPLE, 'normal')).toBe(SAMPLE)
    expect(applyExportPalette(SAMPLE, undefined)).toBe(SAMPLE)
    expect(applyExportPalette('', 'mono')).toBe('')
  })

  it('黑白：五个映射色全变灰阶，版式色（墨色）不动', () => {
    const out = applyExportPalette(SAMPLE, 'mono')
    const mono = toSvgPalette('mono')
    for (const { variable } of EXPORT_PALETTE_MAP) {
      expect(out).toContain(mono[variable])
    }
    for (const { key } of EXPORT_PALETTE_MAP) {
      expect(out).not.toContain(EXP_COLORS[key])
    }
    expect(out).toContain(EXP_COLORS.ink)
  })

  it('护眼：替换为低饱和纸面档', () => {
    const out = applyExportPalette(SAMPLE, 'eyeCare')
    const eye = toSvgPalette('eyeCare')
    expect(out).toContain(eye['--sched-critical'])
    expect(out).toContain(eye['--sched-normal'])
    expect(out).not.toContain(EXP_COLORS.critical)
  })

  it('大小写不敏感的 hex 也替换', () => {
    const upper = SAMPLE.toUpperCase()
    const out = applyExportPalette(upper, 'mono')
    expect(out).not.toContain(EXP_COLORS.critical.toUpperCase())
    expect(out).toContain(toSvgPalette('mono')['--sched-critical'])
  })
})
