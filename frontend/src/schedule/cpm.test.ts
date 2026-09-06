import { describe, expect, it } from 'vitest'
import { computeCpm } from './cpm'
import type { SchedLink, SchedTask } from './types'

function t(id: string, duration: number, extra?: Partial<SchedTask>): SchedTask {
  return { id, name: id, duration, level: 1, progress: 0, ...extra }
}
function l(from: string, to: string, type: SchedLink['type'] = 'FS', lag = 0): SchedLink {
  return { from, to, type, lag }
}

describe('computeCpm 基础链路', () => {
  it('顺序链 A(3)→B(2)→C(4)：全关键，总工期 9', () => {
    const r = computeCpm([t('A', 3), t('B', 2), t('C', 4)], [l('A', 'B'), l('B', 'C')])
    expect(r.ok).toBe(true)
    expect(r.duration).toBe(9)
    expect(r.rows.A).toMatchObject({ es: 0, ef: 3, ls: 0, lf: 3, tf: 0, ff: 0, critical: true })
    expect(r.rows.B).toMatchObject({ es: 3, ef: 5, tf: 0, critical: true })
    expect(r.rows.C).toMatchObject({ es: 5, ef: 9, tf: 0, critical: true })
  })

  it('并行支路：短支路有时差，长支路关键', () => {
    const r = computeCpm([t('A', 2), t('B', 6), t('C', 1)], [l('A', 'B'), l('A', 'C')])
    expect(r.duration).toBe(8)
    expect(r.rows.B.critical).toBe(true)
    expect(r.rows.C).toMatchObject({ es: 2, ef: 3, tf: 5 })
    // C 无后续 → 自由时差 = 总工期 - EF = 5
    expect(r.rows.C.ff).toBe(5)
    // A 的自由时差受两支约束取小：B.es-lag-EF_A = 2-0-2 = 0
    expect(r.rows.A.ff).toBe(0)
  })

  it('空项目：ok 且总工期 0', () => {
    const r = computeCpm([], [])
    expect(r.ok).toBe(true)
    expect(r.duration).toBe(0)
  })

  it('无搭接多任务：并行起算，总工期 = 最长工期', () => {
    const r = computeCpm([t('A', 5), t('B', 9), t('C', 2)], [])
    expect(r.duration).toBe(9)
    expect(r.rows.A).toMatchObject({ es: 0, ef: 5, tf: 4 })
    expect(r.rows.C.critical).toBe(false)
  })
})

describe('computeCpm 搭接类型', () => {
  it('SS+时距：B 在 A 开工 4 天后开工', () => {
    const r = computeCpm([t('A', 10), t('B', 5)], [l('A', 'B', 'SS', 4)])
    expect(r.rows.B).toMatchObject({ es: 4, ef: 9 })
    expect(r.duration).toBe(10)
  })

  it('FF：B 完成不得早于 A 完成（EF_B ≥ EF_A）', () => {
    const r = computeCpm([t('A', 10), t('B', 2)], [l('A', 'B', 'FF', 0)])
    expect(r.rows.B).toMatchObject({ es: 8, ef: 10 })
    expect(r.duration).toBe(10)
  })

  it('SF：里程碑 M 在 A 开工后 5 天完成', () => {
    const r = computeCpm([t('A', 5), t('M', 0, { isMilestone: true })], [l('A', 'M', 'SF', 5)])
    expect(r.rows.M).toMatchObject({ es: 5, ef: 5 })
    expect(r.duration).toBe(5)
  })

  it('FS+正时距：B 待 A 完成后隔 2 天开工', () => {
    const r = computeCpm([t('A', 3), t('B', 1)], [l('A', 'B', 'FS', 2)])
    expect(r.rows.B).toMatchObject({ es: 5, ef: 6 })
    expect(r.duration).toBe(6)
  })

  it('负时距搭接：允许搭接过半（FS -2）', () => {
    const r = computeCpm([t('A', 6), t('B', 4)], [l('A', 'B', 'FS', -2)])
    expect(r.rows.B).toMatchObject({ es: 4, ef: 8 })
    expect(r.duration).toBe(8)
  })

  it('混合搭接：SS 推开工、FF 收尾，总工期取大', () => {
    // A(10)；B(6) SS A lag4 → ES_B=4,EF_B=10；且 FF A lag2 → EF_B ≥ EF_A+2=12 → ES_B=6
    const r = computeCpm([t('A', 10), t('B', 6)], [l('A', 'B', 'SS', 4), l('A', 'B', 'FF', 2)])
    expect(r.rows.B).toMatchObject({ es: 6, ef: 12 })
    expect(r.duration).toBe(12)
  })
})

describe('computeCpm 逆推与时差', () => {
  it('逆推：非关键任务 LF 由后继约束，TF = LS - ES', () => {
    // A(2)→C(1)；A(2)→B(6)：C 的 LS=7（LF=8），ES=2 → TF=5
    const r = computeCpm([t('A', 2), t('B', 6), t('C', 1)], [l('A', 'B'), l('A', 'C')])
    expect(r.rows.C).toMatchObject({ ls: 7, lf: 8, tf: 5 })
    expect(r.rows.A.lf).toBe(2)
  })

  it('中间任务的自由时差受后继最早时间约束', () => {
    // A(1)→C(1)→D(5)；A(1)→B(6)→D(5)：D.es=max(EF_C,EF_B)=7；C.ef=2 → C.ff=5
    const r = computeCpm(
      [t('A', 1), t('B', 6), t('C', 1), t('D', 5)],
      [l('A', 'C'), l('C', 'D'), l('A', 'B'), l('B', 'D')],
    )
    expect(r.rows.C).toMatchObject({ es: 1, ef: 2, ff: 5 })
    expect(r.rows.D.critical).toBe(true)
  })
})

describe('computeCpm 手动/自动双模式（Project 口径）', () => {
  it('manual 锁定开始：入边不推它，后继以它的 EF 为约束', () => {
    // A(10) FS M；M manual start=2 dur=3；B FS M
    const r = computeCpm(
      [t('A', 10), { id: 'M', name: 'M', duration: 3, level: 1, progress: 0, mode: 'manual', manualStart: 2 }, t('B', 1)],
      [l('A', 'M'), l('M', 'B')],
    )
    expect(r.rows.M).toMatchObject({ es: 2, ef: 5, tf: 0, critical: false })
    expect(r.rows.B).toMatchObject({ es: 5, ef: 6 })
    expect(r.rows.A.critical).toBe(true)
    expect(r.duration).toBe(10)
  })

  it('manual 不回传约束：前置 LF 不被手动任务收小，且不被标关键误伤', () => {
    // P(2) FS M(manual start=5,dur=1)：P.lf 应=总工期6（不被 M 的 ls=5 收小）
    const r = computeCpm(
      [t('P', 2), { id: 'M', name: 'M', duration: 1, level: 1, progress: 0, mode: 'manual', manualStart: 5 }],
      [l('P', 'M')],
    )
    expect(r.rows.M).toMatchObject({ es: 5, ef: 6 })
    expect(r.rows.P).toMatchObject({ es: 0, ef: 2, lf: 6, tf: 4, critical: false })
    // P 的自由时差不被 manual 收：= 总工期 - EF = 4
    expect(r.rows.P.ff).toBe(4)
  })

  it('manual 任务缺省 manualStart=0：与开工同日起算', () => {
    const r = computeCpm(
      [{ id: 'M', name: 'M', duration: 4, level: 1, progress: 0, mode: 'manual' }],
      [],
    )
    expect(r.rows.M).toMatchObject({ es: 0, ef: 4, critical: false })
  })
})

describe('computeCpm 健壮性', () => {
  it('循环依赖：fail-closed 并给出环上任务名', () => {
    const r = computeCpm([t('A', 1), t('B', 1), t('C', 1)], [l('A', 'B'), l('B', 'C'), l('C', 'A')])
    expect(r.ok).toBe(false)
    expect(r.error).toContain('循环依赖')
    expect(r.cycle).toEqual(expect.arrayContaining(['A', 'B', 'C']))
  })

  it('自环与悬空搭接被忽略，不影响计算', () => {
    const r = computeCpm([t('A', 3), t('B', 2)], [l('A', 'A'), l('X', 'B'), l('A', 'B')])
    expect(r.ok).toBe(true)
    expect(r.rows.B).toMatchObject({ es: 3 })
    expect(r.duration).toBe(5)
  })

  it('里程碑（工期 0）不占工期但参与搭接', () => {
    const r = computeCpm([t('A', 4), t('M', 0, { isMilestone: true }), t('B', 2)], [l('A', 'M'), l('M', 'B')])
    expect(r.rows.M).toMatchObject({ es: 4, ef: 4, critical: true })
    expect(r.rows.B).toMatchObject({ es: 4, ef: 6 })
    expect(r.duration).toBe(6)
  })
})
