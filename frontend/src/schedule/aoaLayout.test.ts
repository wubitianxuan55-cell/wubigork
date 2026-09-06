import { describe, expect, it } from 'vitest'
import { buildAoa, type AoaGraph } from './aoa'
import { AOA_COL_W, AOA_MARGIN, AOA_ROW_H } from './aoa'
import { applyPins, normalizeAoaLayout, prunePins, snapPt } from './aoaLayout'
import type { SchedLink, SchedTask } from './types'

function t(id: string, duration: number, extra?: Partial<SchedTask>): SchedTask {
  return { id, name: id, duration, level: 1, progress: 0, ...extra }
}
function l(from: string, to: string, type: SchedLink['type'] = 'FS', lag = 0): SchedLink {
  return { from, to, type, lag }
}
function chain(): SchedTask[] {
  return [t('A', 3), t('B', 2), t('C', 4)]
}
function anchorsOf(g: AoaGraph): Record<string, string> {
  return Object.fromEntries(g.nodes.map((n) => [n.id, n.anchor]))
}

describe('锚点键推导与稳定性（镜像设计 §3.2 稳定性分析）', () => {
  it('链式合并：B/C 共用前置完成事件，代表键=end:前任务', () => {
    const g = buildAoa(chain(), [l('A', 'B'), l('B', 'C')])
    const a = anchorsOf(g)
    expect(Object.values(a)).toContain('S')
    expect(Object.values(a)).toContain('T')
    // B 的开始事件与 A 的完成事件合并：成员键 {end:A, start:B}，end: 字典序在前
    expect(Object.values(a)).toContain('end:A')
  })

  it('菱形：独立分支事件各自代表 end:/start: 最小者', () => {
    const g = buildAoa([t('A', 3), t('B', 2), t('C', 4), t('D', 1)], [l('A', 'B'), l('A', 'C'), l('B', 'D'), l('C', 'D')])
    const values = Object.values(anchorsOf(g))
    // A 的完成事件被 B/C 共用开始？否——A 完成事件 finishClean（无 FF/SF 指入）且 FS(lag0) 唯一
    // → B 与 C 的开始事件都与 A 的完成事件合并：{end:A, start:B} / {end:A, start:C}，
    // 但两个任务共用同一事件 → 成员键合集 {end:A, start:B, start:C}，最小=end:A
    expect(values.filter((v) => v === 'end:A').length).toBe(1)
    expect(values.filter((v) => v === 'S').length).toBe(1)
    expect(values.filter((v) => v === 'T').length).toBe(1)
  })

  it('改工期/名称/开工日期：锚点集不变（手动坐标全保的前提）', () => {
    const before = anchorsOf(buildAoa(chain(), [l('A', 'B'), l('B', 'C')]))
    const after = anchorsOf(buildAoa(
      chain().map((x) => (x.id === 'B' ? { ...x, duration: 9, name: '改名' } : x)),
      [l('A', 'B'), l('B', 'C')],
    ))
    expect(after).toEqual(before)
  })

  it('新增任务：既有锚点全保，新事件带新锚点', () => {
    const before = anchorsOf(buildAoa(chain(), [l('A', 'B'), l('B', 'C')]))
    const after = anchorsOf(buildAoa([...chain(), t('D', 2)], [l('A', 'B'), l('B', 'C'), l('C', 'D')]))
    for (const [id, a] of Object.entries(before)) {
      if (id in after) expect(after[id]).toBe(a)
    }
    expect(Object.values(after).length).toBeGreaterThan(Object.values(before).length)
  })

  it('删除任务：失配锚点消失、其余锚点稳定（合并回收不改写幸存者代表）', () => {
    const before = anchorsOf(buildAoa([...chain(), t('D', 2)], [l('A', 'B'), l('B', 'C'), l('C', 'D')]))
    const after = anchorsOf(buildAoa(chain(), [l('A', 'B'), l('B', 'C')]))
    // 幸存锚点（S/T/end:*）在删 D 后仍存在
    for (const a of Object.values(after)) expect(Object.values(before)).toContain(a)
  })
})

describe('snapPt / normalizeAoaLayout / applyPins / prunePins', () => {
  it('snapPt 半格吸附（55×37）', () => {
    expect(snapPt(0, 0)).toEqual({ x: 0, y: 0 })
    expect(snapPt(52, 30)).toEqual({ x: 55, y: 37 })
    expect(snapPt(AOA_MARGIN, AOA_MARGIN)).toEqual({ x: 55, y: 37 })
    expect(snapPt(AOA_MARGIN + AOA_COL_W, AOA_MARGIN + AOA_ROW_H)).toEqual({ x: 165, y: 111 })
  })

  it('normalizeAoaLayout：坏形丢弃、坐标取整、键数上限', () => {
    const raw = {
      pins: {
        S: { x: 55.4, y: 37.6 },
        bad: { x: Number.NaN, y: 1 },
        bad2: { x: 'a', y: 2 },
        bad3: null,
      },
    }
    const n = normalizeAoaLayout(raw)
    expect(n.pins.S).toEqual({ x: 55, y: 38 })
    expect(Object.keys(n.pins)).toEqual(['S'])
    expect(normalizeAoaLayout(null).pins).toEqual({})
    expect(normalizeAoaLayout({ pins: 'x' }).pins).toEqual({})
    expect(normalizeAoaLayout({ pins: { a: { x: 1, y: 2 } }, extra: 1 }).pins.a).toEqual({ x: 1, y: 2 })
    const many = normalizeAoaLayout({ pins: Object.fromEntries(Array.from({ length: 5100 }, (_, i) => [`k${i}`, { x: i, y: i }])) })
    expect(Object.keys(many.pins).length).toBe(5000)
  })

  it('applyPins：命中锚点手动位、未命中保留自动位', () => {
    const g = buildAoa(chain(), [l('A', 'B'), l('B', 'C')])
    const start = g.nodes.find((n) => n.anchor === 'S')!
    const auto = g.nodes.find((n) => n.anchor !== 'S')!
    const pinned = applyPins(g, { S: { x: 500, y: 400 } })
    expect(pinned.nodes.find((n) => n.anchor === 'S')!.x).toBe(500)
    expect(pinned.nodes.find((n) => n.id === auto.id)!.y).toBe(auto.y)
    expect(start).toBeDefined()
    // 原 graph 不被改写（纯函数）
    expect(g.nodes.find((n) => n.anchor === 'S')!.x).toBe(start.x)
  })

  it('prunePins：失配键清理、命中键保留', () => {
    const g = buildAoa(chain(), [l('A', 'B'), l('B', 'C')])
    const pruned = prunePins({ S: { x: 1, y: 2 }, 'end:ghost': { x: 3, y: 4 } }, g)
    expect(pruned).toEqual({ S: { x: 1, y: 2 } })
  })
})
