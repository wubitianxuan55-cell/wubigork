/**
 * drag.test.ts — 横道拖拽落点换算用例（v4.118.0 刀9）
 *
 * 周一开工（2026-09-07）默认工作制的自然日偏移序列：
 * wd0..wd4 = 偏移 0..4（一~五），wd5 = 偏移 7（下周一）……跨周末跳 3。
 */
import { describe, expect, it } from 'vitest'
import { dropToWd, nearestWd, resizeToDuration, workdayOffsets } from './drag'

const MON = '2026-09-07'
/** 12 个工作日的反查表（覆盖两周+），手算基准 */
const OFFSETS = workdayOffsets(MON, 12)
// 周一~五=0..4，周六日=5,6；下周一=7 … 第二个周五=11，周六日=12,13
const EXPECTED = [0, 1, 2, 3, 4, 7, 8, 9, 10, 11, 14, 15]

describe('workdayOffsets', () => {
  it('工作日→自然日偏移单调，跨周末跳档', () => {
    expect(OFFSETS).toEqual(EXPECTED)
  })
})

describe('nearestWd', () => {
  it('正中命中、落在周末吸附最近工作日', () => {
    expect(nearestWd(OFFSETS, 0)).toBe(0)
    expect(nearestWd(OFFSETS, 7)).toBe(5) // 下周一
    expect(nearestWd(OFFSETS, 5)).toBe(4) // 周六→周五（距 1）
    expect(nearestWd(OFFSETS, 6)).toBe(5) // 周日→下周一（距 1）
  })
  it('钳位两端；空表返 0', () => {
    expect(nearestWd(OFFSETS, -9)).toBe(0)
    expect(nearestWd(OFFSETS, 99)).toBe(11)
    expect(nearestWd([], 3)).toBe(0)
  })
})

describe('dropToWd', () => {
  it('像素位移按 dayW 换算后吸附（20px/自然日）', () => {
    // 条形从 wd0（偏移 0）向右拖 60px=3 自然日 → 偏移 3 → wd3
    expect(dropToWd(OFFSETS, 0, 60, 20)).toBe(3)
    // 向右拖 100px=5 自然日 → 偏移 5（周六）→ 吸附 wd4 周五
    expect(dropToWd(OFFSETS, 0, 100, 20)).toBe(4)
    // 向左越界钳 0
    expect(dropToWd(OFFSETS, 1, -999, 20)).toBe(0)
  })
})

describe('resizeToDuration', () => {
  it('右缘落点 - es = 新工期，负值钳 0', () => {
    // es=wd2（偏移 2），右缘在 wd4（偏移 4）：拖 +60px → 偏移 7 → ef=wd5 → 工期 3
    expect(resizeToDuration(OFFSETS, 2, 4, 60, 20)).toBe(3)
    // 向左缩过头 → 0
    expect(resizeToDuration(OFFSETS, 2, 4, -999, 20)).toBe(0)
    // 不动 → 原工期
    expect(resizeToDuration(OFFSETS, 2, 4, 0, 20)).toBe(2)
  })
})
