/**
 * pathDriver.test.ts — 任务路径分析引擎纯函数（驱动依赖/驱动链）
 *
 * 用例口径：逐依赖 ff===0 且后继 auto 即驱动（ProjectLibre E8 蒸馏）；
 * SS/SF 锚点=前置 ES；手动后继/手动上游为链端点不扩展；分组行不进链；
 * 循环依赖 fail-closed 返回空；菱形汇合去重保序。
 */
import { describe, expect, it } from 'vitest'
import { computeCpm } from './cpm'
import { depFreeSlacks, drivingChain, drivingLinkKeys, linkKey } from './pathDriver'
import type { SchedLink, SchedTask } from './types'

function t(id: string, duration: number, extra?: Partial<SchedTask>): SchedTask {
  return { id, name: id, duration, level: 1, progress: 0, ...extra }
}
function l(from: string, to: string, type: SchedLink['type'] = 'FS', lag = 0): SchedLink {
  return { from, to, type, lag }
}
function group(id: string, name: string): SchedTask {
  return { id, name, duration: 0, level: 0, progress: 0 }
}

describe('linkKey', () => {
  it('from->to 键', () => {
    expect(linkKey({ from: 'A', to: 'B', type: 'SS', lag: 3 })).toBe('A->B')
  })
})

describe('depFreeSlacks（复用 freeFloatPart 四型口径）', () => {
  // Q(10)→P(10)FS；P SS+6→B；C(20) FS→B：B.es=max(P.es+6, C.ef)=20
  // ff(P→B)=B.es-lag-ES(P)=20-6-10=4（锚点=前置 ES；旧版漏减得 14 口径尽失）
  const tasks = [t('Q', 10), t('P', 10), t('C', 20), t('B', 5)]
  const links = [l('Q', 'P'), l('P', 'B', 'SS', 6), l('C', 'B')]
  const cpm = computeCpm(tasks, links)

  it('SS+6 时距：ff=to.es-lag-ES_from=4（非 0，不驱动）', () => {
    expect(cpm.rows.B).toMatchObject({ es: 20, ef: 25 })
    const slacks = depFreeSlacks(tasks, links, cpm)
    expect(slacks['P->B']).toBe(4)
    expect(slacks['C->B']).toBe(0)
  })

  it('悬空搭接（端点不在任务表/rows）跳过，不产幽灵键', () => {
    const slacks = depFreeSlacks(tasks, [...links, l('X', 'B'), l('P', 'P')], cpm)
    expect(slacks['X->B']).toBeUndefined()
    expect(slacks['P->P']).toBeUndefined()
  })
})

describe('drivingLinkKeys', () => {
  it('三连 FS 链 A→B→C：两条边 ff=0 全在驱动集', () => {
    const tasks = [t('A', 3), t('B', 2), t('C', 4)]
    const links = [l('A', 'B'), l('B', 'C')]
    const keys = drivingLinkKeys(tasks, links, computeCpm(tasks, links))
    expect(keys.size).toBe(2)
    expect(keys.has('A->B')).toBe(true)
    expect(keys.has('B->C')).toBe(true)
  })

  it('SS+6 单独成链：ff=0 即驱动（B.es=6 恰由该边定）', () => {
    const tasks = [t('A', 10), t('B', 5)]
    const links = [l('A', 'B', 'SS', 6)]
    const cpm = computeCpm(tasks, links)
    expect(cpm.rows.B).toMatchObject({ es: 6 })
    expect(depFreeSlacks(tasks, links, cpm)['A->B']).toBe(0)
    expect(drivingLinkKeys(tasks, links, cpm).has('A->B')).toBe(true)
  })

  it('平行双前继：只选 ff=0 那条（绑定边）', () => {
    // A(6)、C(2) 并行入 D(3)：D.es=6；ff(A→D)=0 驱动，ff(C→D)=4 不驱动
    const tasks = [t('A', 6), t('C', 2), t('D', 3)]
    const links = [l('A', 'D'), l('C', 'D')]
    const keys = drivingLinkKeys(tasks, links, computeCpm(tasks, links))
    expect(keys.has('A->D')).toBe(true)
    expect(keys.has('C->D')).toBe(false)
  })

  it('手动后继无驱动边（日期锁死）；循环依赖 fail-closed 空集', () => {
    const tasks = [t('A', 10), t('B', 3, { mode: 'manual', manualStart: 2 })]
    const keys = drivingLinkKeys(tasks, [l('A', 'B')], computeCpm(tasks, [l('A', 'B')]))
    expect(keys.size).toBe(0)

    const cyc = [t('A', 1), t('B', 1), t('C', 1)]
    const cycLinks = [l('A', 'B'), l('B', 'C'), l('C', 'A')]
    const bad = computeCpm(cyc, cycLinks)
    expect(bad.ok).toBe(false)
    expect(drivingLinkKeys(cyc, cycLinks, bad).size).toBe(0)
  })
})

describe('drivingChain 基础方向', () => {
  const tasks = [t('A', 3), t('B', 2), t('C', 4)]
  const links = [l('A', 'B'), l('B', 'C')]
  const cpm = computeCpm(tasks, links)

  it('对 B 取 pred=[A]；succ=[C]；both=[A,C]（去重保序+链上边）', () => {
    expect(drivingChain(tasks, links, cpm, 'B', 'pred')).toEqual({
      taskIds: ['A'],
      links: [links[0]],
    })
    expect(drivingChain(tasks, links, cpm, 'B', 'succ')).toEqual({
      taskIds: ['C'],
      links: [links[1]],
    })
    expect(drivingChain(tasks, links, cpm, 'B', 'both')).toEqual({
      taskIds: ['A', 'C'],
      links: links,
    })
  })

  it('菱形 A→B,A→C,B→D,C→D：对 A 取 succ=[B,C,D] 不重复访问', () => {
    const dt = [t('A', 1), t('B', 2), t('C', 2), t('D', 2)]
    const dl = [l('A', 'B'), l('A', 'C'), l('B', 'D'), l('C', 'D')]
    const dc = computeCpm(dt, dl)
    const ch = drivingChain(dt, dl, dc, 'A', 'succ')
    expect(ch.taskIds).toEqual(['B', 'C', 'D'])
    expect(ch.links).toEqual(dl)
    // 汇合点上溯：both 双向也不重复（BFS 保序 B、C 先于 A）
    const up = drivingChain(dt, dl, dc, 'D', 'both')
    expect(up.taskIds).toEqual(['B', 'C', 'A'])
    expect(up.links).toEqual(dl)
  })

  it('里程碑正常参与（A→M 驱动）', () => {
    const mt = [t('A', 4), t('M', 0, { isMilestone: true })]
    const ml = [l('A', 'M')]
    const mc = computeCpm(mt, ml)
    expect(drivingChain(mt, ml, mc, 'M', 'pred').taskIds).toEqual(['A'])
    expect(drivingChain(mt, ml, mc, 'A', 'succ').taskIds).toEqual(['M'])
  })
})

describe('drivingChain 平行双前继只走驱动边', () => {
  it('A(6)/C(2) 入 D：pred(D)=[A]，非驱动前继 C 不进链', () => {
    const tasks = [t('A', 6), t('C', 2), t('D', 3)]
    const links = [l('A', 'D'), l('C', 'D')]
    const ch = drivingChain(tasks, links, computeCpm(tasks, links), 'D', 'pred')
    expect(ch.taskIds).toEqual(['A'])
    expect(ch.links).toEqual([links[0]])
  })
})

describe('drivingChain 手动任务打断（端点不扩展）', () => {
  // A(auto,10)→B(manual,start=2,dur=3)→C(auto,1)：B.es=2,ef=5；C.es=5
  const tasks = [
    t('A', 10),
    t('B', 3, { mode: 'manual', manualStart: 2 }),
    t('C', 1),
  ]
  const links = [l('A', 'B'), l('B', 'C')]
  const cpm = computeCpm(tasks, links)

  it('对 C 取 pred 含 B 不含 A（手动任务上游不再上溯）', () => {
    expect(cpm.rows.B).toMatchObject({ es: 2, ef: 5 })
    const ch = drivingChain(tasks, links, cpm, 'C', 'pred')
    expect(ch.taskIds).toEqual(['B'])
    expect(ch.links).toEqual([links[1]])
  })

  it('对 A 取 succ 含 B 不含 C（手动后继进链即端点）', () => {
    const ch = drivingChain(tasks, links, cpm, 'A', 'succ')
    expect(ch.taskIds).toEqual(['B'])
    expect(ch.links).toEqual([links[0]])
    expect(drivingChain(tasks, links, cpm, 'A', 'both').taskIds).toEqual(['B'])
  })

  it('驱动集同样跳过指向手动任务的边（B→C 仍可驱动 C）', () => {
    const keys = drivingLinkKeys(tasks, links, cpm)
    expect(keys.has('A->B')).toBe(false)
    expect(keys.has('B->C')).toBe(true)
  })
})

describe('drivingChain fail-closed 与分组行', () => {
  it('循环依赖（ok=false）：链返回空', () => {
    const tasks = [t('A', 1), t('B', 1), t('C', 1)]
    const links = [l('A', 'B'), l('B', 'C'), l('C', 'A')]
    const bad = computeCpm(tasks, links)
    expect(drivingChain(tasks, links, bad, 'A', 'both')).toEqual({ taskIds: [], links: [] })
  })

  it('任务不存在：返回空', () => {
    const tasks = [t('A', 3), t('B', 2)]
    const links = [l('A', 'B')]
    expect(drivingChain(tasks, links, computeCpm(tasks, links), 'Z', 'both')).toEqual({
      taskIds: [],
      links: [],
    })
  })

  it('起点是分组行（level=0）：直接返回空', () => {
    const tasks = [group('G1', '前期'), t('A', 3)]
    const links = [l('G1', 'A')]
    const cpm = computeCpm(tasks, links)
    expect(drivingChain(tasks, links, cpm, 'G1', 'both')).toEqual({ taskIds: [], links: [] })
  })

  it('分组行不进链：即便其边 ff=0 驱动里程碑也不高亮', () => {
    const tasks = [group('G1', '前期'), t('M', 0, { isMilestone: true })]
    const links = [l('G1', 'M')]
    const cpm = computeCpm(tasks, links)
    const ch = drivingChain(tasks, links, cpm, 'M', 'pred')
    expect(ch.taskIds).toEqual([])
    expect(ch.links).toEqual([])
  })
})
