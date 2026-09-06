import { describe, expect, it } from 'vitest'
import { fmtLinkRefs, ganttLinkPath } from './ganttLinks'

/** 夹具：前置条 100~200/行 30，后续条 260~380/行 120（FS 有水平余量） */
const BASE = { x1s: 100, x1f: 200, y1: 30, x2s: 260, x2f: 380, y2: 120 }

describe('ganttLinkPath 四种搭接锚点', () => {
  it('FS：前完成列 → 后开始列', () => {
    const d = ganttLinkPath('FS', BASE)
    expect(d).toBe('M 200 30 H 230 V 120 H 257')
  })
  it('SS：起→起（锚点换到开始列）', () => {
    const d = ganttLinkPath('SS', BASE)
    expect(d.startsWith('M 100 30')).toBe(true)
    expect(d.endsWith('H 257')).toBe(true)
  })
  it('FF：完→完（终点锚在后续完成列）', () => {
    const d = ganttLinkPath('FF', BASE)
    expect(d.endsWith('H 377')).toBe(true)
  })
  it('SF：前开始 → 后完成（反向进入不崩）', () => {
    const d = ganttLinkPath('SF', BASE)
    expect(d.startsWith('M 100 30')).toBe(true)
    expect(d.endsWith('H 377')).toBe(true)
  })
  it('后续在前置左侧（负时距/倒排）：反向进入，箭头指向右缘+3', () => {
    const d = ganttLinkPath('FS', { ...BASE, x2s: 60, x2f: 90, x1s: 100, x1f: 200 })
    // x1f=200 → x2s=60：反向绕行，末端 H 63（60+3）
    expect(d.endsWith('V 120 H 63')).toBe(true)
  })
  it('水平空间不足：短桩绕行不压目标条', () => {
    const d = ganttLinkPath('FS', { ...BASE, x2s: 205 })
    expect(d).toContain('V 112') // jog = y2-8
    expect(d.endsWith('V 120')).toBe(true)
  })
})

describe('fmtLinkRefs Project 口径引用文本', () => {
  const noOf = (id: string) => ({ t1: 1, t2: 2, t3: 3 }[id] ?? 0)
  it('lag=0 省略时距，lag≠0 带 ±', () => {
    expect(
      fmtLinkRefs(
        [
          { id: 't2', type: 'FS', lag: 0 },
          { id: 't3', type: 'SS', lag: 2 },
          { id: 't1', type: 'FF', lag: -1 },
        ],
        noOf,
      ),
    ).toBe('2FS,3SS+2,1FF-1')
  })
  it('空引用返回空串（列内留白不占位假数据）', () => {
    expect(fmtLinkRefs([], noOf)).toBe('')
  })
})
