import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('深色主题硬编码深灰字（对称于 text-white）', () => {
  it('关系图 overlay / 图例不再写死 #555 #ddd', () => {
    const src = readFileSync(resolve(__dirname, 'components/RelationGraph.tsx'), 'utf8')
    expect(src).not.toContain("color: '#555'")
    expect(src).not.toContain("color: '#ddd'")
    expect(src).not.toContain("color: '#9ca3af'")
    expect(src).toContain('var(--color-text-secondary)')
  })

  it('进度计划里程碑标签/菱形跟 on-surface，不写死 #374151 / #1f2937', () => {
    const src = readFileSync(resolve(__dirname, 'schedule/schedule.css'), 'utf8')
    const tag = src.slice(src.indexOf('.sched-milestone-tag'), src.indexOf('.sched-baseline-bar'))
    expect(tag).not.toContain('#374151')
    expect(tag).toContain('--md-sys-color-on-surface-variant')
    const diamond = src.slice(src.indexOf('.sched-milestone {'), src.indexOf('.sched-milestone-tag'))
    expect(diamond).not.toContain('#1f2937')
    expect(diamond).toContain('--md-sys-color-on-surface')
  })
})

describe('亮态弱化文字对比度（v4.361 观察池清账）', () => {
  it('schedule 文字不再把 outline 当颜色（0.25 alpha 压浅底≈1.3~1.8:1）——文字用 on-surface-variant', () => {
    const css = readFileSync(resolve(__dirname, 'schedule/schedule.css'), 'utf8')
    expect(css).not.toMatch(/^\s+color:\s*var\(--md-sys-color-outline[,)]/m)
    expect(css).toContain('color: var(--md-sys-color-on-surface-variant, #6b7280)')
    const usage = readFileSync(resolve(__dirname, 'schedule/UsageView.tsx'), 'utf8')
    expect(usage).toContain("DIM_COLOR = 'var(--md-sys-color-on-surface-variant, #6b7280)'")
    const aoa = readFileSync(resolve(__dirname, 'schedule/AoaView.tsx'), 'utf8')
    expect(aoa).not.toContain('fill="var(--md-sys-color-outline')
    const pred = readFileSync(resolve(__dirname, 'schedule/gantt/PredEditor.tsx'), 'utf8')
    expect(pred).not.toContain('--md-sys-color-outline,')
  })
})
