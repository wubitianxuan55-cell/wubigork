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
