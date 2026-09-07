import { describe, expect, it } from 'vitest'
import { buildAoa, type AoaEdge, type AoaGraph } from './aoa'
import { AOA_COL_W, AOA_MARGIN, AOA_R, AOA_ROW_H } from './aoa'
import { applyPins, edgeSegs, findBridgeArcs, normalizeAoaLayout, prunePins, segsToPath, snapPt } from './aoaLayout'
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

  it('snapPt 网格含节点半径口径不变（AOA_R=16 常量迁移）', () => {
    expect(AOA_R).toBe(16)
  })
})

// ── 刀H 图例语言（G1 过桥 / G2 标注归位 / G4 波形线）─────────────

function edge(from: string, to: string, dur = 2): AoaEdge {
  return { id: `${from}>${to}`, from, to, kind: 'task', dur, critical: false, label: `${dur}d` }
}
function nodeMap(entries: [string, number, number][]): Map<string, { x: number; y: number }> {
  return new Map(entries.map(([id, x, y]) => [id, { x, y }]))
}

describe('edgeSegs：G2 标注归位（名称在箭线上、工期在下）', () => {
  it('同行列边：名称 y 在线上方、工期 y 在线下方，x 居中', () => {
    const nodes = nodeMap([['a', 100, 100], ['b', 300, 100]])
    const g = edgeSegs(edge('a', 'b'), nodes, { idx: 0, cnt: 1 })
    expect(g.segs).toEqual([{ x1: 116, y1: 100, x2: 284, y2: 100 }])
    expect(g.name.y).toBe(95) // 100-5：箭线上方
    expect(g.dur.y).toBe(107) // 100+7：箭线下方
    expect(g.name.x).toBe(200)
    expect(g.dur.x).toBe(200)
    expect(g.name.anchor).toBe('middle')
  })

  it('波形线存在时标注居中于实体段（不落在波形上）', () => {
    const nodes = nodeMap([['a', 100, 100], ['b', 300, 100]])
    const g = edgeSegs(edge('a', 'b'), nodes, { idx: 0, cnt: 1 }, 200)
    expect(g.name.x).toBe((116 + 200) / 2)
    expect(g.dur.x).toBe((116 + 200) / 2)
  })

  it('H-V-H 边：名称/工期堆叠贴第一段水平线上下', () => {
    const nodes = nodeMap([['a', 100, 100], ['b', 300, 180]])
    const g = edgeSegs(edge('a', 'b'), nodes, { idx: 0, cnt: 1 })
    expect(g.segs).toEqual([
      { x1: 116, y1: 100, x2: 200, y2: 100 },
      { x1: 200, y1: 100, x2: 200, y2: 180 },
      { x1: 200, y1: 180, x2: 284, y2: 180 },
    ])
    expect(g.name.y).toBe(95)
    expect(g.dur.y).toBe(107)
  })

  it('同列竖直边：名称在上、工期在下（右移堆叠）', () => {
    const nodes = nodeMap([['a', 100, 100], ['b', 100, 180]])
    const g = edgeSegs(edge('a', 'b'), nodes, { idx: 0, cnt: 1 })
    expect(g.segs).toEqual([{ x1: 100, y1: 116, x2: 100, y2: 164 }])
    expect(g.name).toEqual({ x: 107, y: 140, anchor: 'start' })
    expect(g.dur).toEqual({ x: 107, y: 152, anchor: 'start' })
  })

  it('后退边（手动布局）：名称在通道线上、工期在下', () => {
    const nodes = nodeMap([['a', 300, 100], ['b', 100, 100]])
    const g = edgeSegs(edge('a', 'b'), nodes, { idx: 0, cnt: 1 })
    expect(g.name.y).toBe(126) // laneY=130-4
    expect(g.dur.y).toBe(140) // laneY+10
  })
})

describe('segsToPath：G4 波形线', () => {
  it('水平段超出实体工期终点的尾段画波形、终点精确落段尾', () => {
    const d = segsToPath([{ x1: 100, y1: 50, x2: 300, y2: 50 }], 160)
    expect(d.startsWith('M 100 50')).toBe(true)
    expect(d).toContain('L 160 50')
    expect(d).toMatch(/ q [\d.]+ -4/)
    expect(d.endsWith('L 300 50')).toBe(true)
  })

  it('无自由时差（waveFromX≥段尾）不画波形', () => {
    const d = segsToPath([{ x1: 100, y1: 50, x2: 300, y2: 50 }], 300)
    expect(d).not.toContain(' q ')
    expect(d).toBe('M 100 50 L 100 50 L 300 50')
  })

  it('尾段不足半个波（<6px）直连不画波形；左行段（后退边）永不画波形', () => {
    expect(segsToPath([{ x1: 100, y1: 50, x2: 300, y2: 50 }], 296)).not.toContain(' q ')
    expect(segsToPath([{ x1: 300, y1: 50, x2: 100, y2: 50 }], 200)).not.toContain(' q ')
  })

  it('waveFromX=null（手动布局）不画波形', () => {
    const d = segsToPath([{ x1: 100, y1: 50, x2: 300, y2: 50 }], null)
    expect(d).not.toContain(' q ')
  })
})

describe('findBridgeArcs + segsToPath：G1 过桥法', () => {
  it('竖段垂直穿越他边横段：检出桥位、半圆右凸（下行 sweep=1）', () => {
    const segsByEdge = [
      [{ x1: 0, y1: 100, x2: 200, y2: 100 }],
      [{ x1: 100, y1: 0, x2: 100, y2: 200 }],
    ]
    const bridges = findBridgeArcs(segsByEdge)
    expect(bridges.get(1)?.get(0)).toEqual([100])
    const d = segsToPath(segsByEdge[1], null, bridges.get(1))
    expect(d).toContain('L 100 95')
    expect(d).toContain('A 5 5 0 0 1 100 105')
    expect(d.endsWith('L 100 200')).toBe(true)
  })

  it('上行竖段半圆同样右凸（sweep=0）', () => {
    const segsByEdge = [
      [{ x1: 0, y1: 100, x2: 200, y2: 100 }],
      [{ x1: 100, y1: 200, x2: 100, y2: 0 }],
    ]
    const d = segsToPath(segsByEdge[1], null, findBridgeArcs(segsByEdge).get(1))
    expect(d).toContain('L 100 105')
    expect(d).toContain('A 5 5 0 0 0 100 95')
  })

  it('同边拐角/端点相触不算交叉（严格内交）', () => {
    const segsByEdge = [
      [
        { x1: 0, y1: 100, x2: 200, y2: 100 },
        { x1: 200, y1: 100, x2: 200, y2: 200 },
      ],
    ]
    expect(findBridgeArcs(segsByEdge).size).toBe(0)
    // 竖段端点恰落在横段上（入节点）不算
    const touching = [
      [{ x1: 0, y1: 100, x2: 200, y2: 100 }],
      [{ x1: 100, y1: 40, x2: 100, y2: 100 }],
    ]
    expect(findBridgeArcs(touching).size).toBe(0)
  })

  it('同段多桥按序保留、间距过密(<2R+4)与距端过近(<R+1)丢弃', () => {
    const segsByEdge = [
      [
        { x1: 0, y1: 100, x2: 200, y2: 100 },
        { x1: 0, y1: 106, x2: 200, y2: 106 },
        { x1: 0, y1: 150, x2: 200, y2: 150 },
        { x1: 0, y1: 195, x2: 200, y2: 195 },
      ],
      [{ x1: 100, y1: 0, x2: 100, y2: 200 }],
    ]
    const bridges = findBridgeArcs(segsByEdge)
    // y=100 保留；y=106 距前桥 6<14 丢弃；y=150 保留；y=195 距段端 5<6 丢弃
    expect(bridges.get(1)?.get(0)).toEqual([100, 150])
    const d = segsToPath(segsByEdge[1], null, bridges.get(1))
    expect(d.match(/A 5 5/g)?.length).toBe(2)
  })
})
