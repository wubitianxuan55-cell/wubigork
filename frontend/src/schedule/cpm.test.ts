import { describe, expect, it } from 'vitest'
import { computeCpm, freeFloatPart } from './cpm'
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

describe('freeFloatPart 四型搭接口径（v4.129 刀G 修 E1）', () => {
  // to 行：ES/EF 任意给（贡献只看 to 的锚点减 from 侧锚点与时距）
  const to = { es: 21, ef: 25, ls: 0, lf: 0, tf: 0, ff: 0, critical: false }
  it('FS 锚点=前置 EF', () => {
    expect(freeFloatPart({ from: 'a', to: 'b', type: 'FS', lag: 2 }, to, 15, 10)).toBe(4) // 21-2-15
  })
  it('SS 锚点=前置 ES（旧版漏减致虚高 5）', () => {
    expect(freeFloatPart({ from: 'a', to: 'b', type: 'SS', lag: 6 }, to, 15, 10)).toBe(5) // 21-6-10
  })
  it('FF 锚点=前置 EF', () => {
    expect(freeFloatPart({ from: 'a', to: 'b', type: 'FF', lag: 1 }, to, 15, 10)).toBe(9) // 25-1-15
  })
  it('SF 锚点=前置 ES（旧版漏减致虚高 10）', () => {
    expect(freeFloatPart({ from: 'a', to: 'b', type: 'SF', lag: 3 }, to, 15, 10)).toBe(12) // 25-3-10
  })
})

describe('SS 搭接下 FF 积分口径（E1 回归钉死）', () => {
  // A(5)→P(10)FS；Q(20) 根；S2 前置=P SS+2 与 Q FS：ES(S2)=max(7,20)=20
  // FF(P)=min(32-15=17, ES(S2)-2-ES(P)=13)=13（旧版漏减 ES(P) 得 17）
  const tasks = [t('A', 5), t('P', 10), t('Q', 20), t('S2', 12)]
  const links = [l('A', 'P'), l('P', 'S2', 'SS', 2), l('Q', 'S2')]
  it('FF(P)=13 而非 17', () => {
    const r = computeCpm(tasks, links)
    expect(r.rows.P).toMatchObject({ es: 5, ef: 15, ff: 13 })
  })
  it('「TF=0 ⇒ FF=0」定理：P 非关键路径上 FF ≤ TF', () => {
    const r = computeCpm(tasks, links)
    expect(r.rows.P.ff).toBeLessThanOrEqual(r.rows.P.tf)
  })
})

describe('SF 搭接下 FF 积分口径（E1 回归钉死）', () => {
  // A(5)→P(10)FS；Q(16) 根；S 前置=P SF+3 与 Q FS：ES(S)=max(8,16)=16，EF(S)=18
  // T(20)←S：duration=38；FF(P)=min(38-15=23, EF(S)-3-ES(P)=10)=10（旧版得 15）
  const tasks = [t('A', 5), t('P', 10), t('Q', 16), t('S', 2), t('T', 20)]
  const links = [l('A', 'P'), l('P', 'S', 'SF', 3), l('Q', 'S'), l('S', 'T')]
  it('FF(P)=10 而非 15', () => {
    const r = computeCpm(tasks, links)
    expect(r.rows.P).toMatchObject({ es: 5, ef: 15, ff: 10 })
  })
})

describe('计划工期锚点（G3：deadline 进逆推）', () => {
  const tasks = [t('A', 3), t('B', 4), t('C', 5)]
  const links = [l('A', 'B'), l('B', 'C')]
  it('planFinish=10 < Tc=12：绑定链负时差、关键=TF 最小集', () => {
    const r = computeCpm(tasks, links, { planFinish: 10 })
    expect(r.duration).toBe(12)
    expect(r.rows.A).toMatchObject({ tf: -2, critical: true })
    expect(r.rows.B).toMatchObject({ tf: -2, critical: true })
    expect(r.rows.C).toMatchObject({ tf: -2, critical: true, ff: -2 })
  })
  it('planFinish=Tc：与无锚点一致（TF=0）', () => {
    const r = computeCpm(tasks, links, { planFinish: 12 })
    expect(r.rows.A).toMatchObject({ tf: 0, critical: true })
    expect(r.rows.C).toMatchObject({ ff: 0 })
  })
  it('planFinish>Tc：不拉伸（锚点回落计算工期）', () => {
    const r = computeCpm(tasks, links, { planFinish: 15 })
    expect(r.rows.A).toMatchObject({ tf: 0, critical: true })
  })
  it('无锚点：与旧口径完全一致', () => {
    const r = computeCpm(tasks, links)
    expect(r.rows.B).toMatchObject({ es: 3, ef: 7, tf: 0, critical: true })
  })
})

describe('computeCpm 双工期口径（v4.150 刀1：cd=日历天，镜像 Go TestCpmCd*）', () => {
  const START = { startDate: '2026-09-07' } // 周一开工

  it('养护 28cd：ef=第 20 工作日，等效跨度 20，单链全关键', () => {
    const r = computeCpm([t('养护', 28, { durationUnit: 'cd' })], [], START)
    expect(r.ok).toBe(true)
    expect(r.duration).toBe(20)
    expect(r.rows['养护']).toMatchObject({ es: 0, ef: 20, ls: 0, lf: 20, tf: 0, ff: 0, critical: true })
  })

  it('混合链 挖土5wd → 养护28cd → 回填5wd：总工期 30（=5+20+5），全关键', () => {
    const r = computeCpm(
      [t('挖土', 5), t('养护', 28, { durationUnit: 'cd' }), t('回填', 5)],
      [l('挖土', '养护'), l('养护', '回填')],
      START,
    )
    expect(r.duration).toBe(30)
    expect(r.rows['养护']).toMatchObject({ es: 5, ef: 25, ls: 5, lf: 25, tf: 0, critical: true })
    expect(r.rows['挖土']).toMatchObject({ es: 0, ef: 5, tf: 0, critical: true })
    expect(r.rows['回填']).toMatchObject({ es: 25, ef: 30, tf: 0, critical: true })
  })

  it('吸附折叠：26/27/28cd 并行无搭接，ef 全=20', () => {
    const r = computeCpm([t('a', 26, { durationUnit: 'cd' }), t('b', 27, { durationUnit: 'cd' }), t('c', 28, { durationUnit: 'cd' })], [], START)
    expect(r.duration).toBe(20)
    expect(r.rows.a.ef).toBe(20)
    expect(r.rows.b.ef).toBe(20)
    expect(r.rows.c.ef).toBe(20)
  })

  it('日历感知：节假日命中养护窗口，ef 顺延（ctx 传自定义日历）', () => {
    const cal = { workweek: [1, 2, 3, 4, 5], holidays: ['2026-09-16'] }
    const r = computeCpm([t('养护', 9, { durationUnit: 'cd' })], [], { ...START, calendar: cal })
    expect(r.rows['养护'].ef).toBe(7)
  })

  it('FS+lag：cd 前置完成边界吸附后，后继从 ef+lag 开工', () => {
    const r = computeCpm([t('养护', 28, { durationUnit: 'cd' }), t('验收', 2)], [l('养护', '验收', 'FS', 2)], START)
    expect(r.duration).toBe(24)
    expect(r.rows['验收']).toMatchObject({ es: 22, ef: 24, tf: 0, critical: true })
    expect(r.rows['养护']).toMatchObject({ es: 0, ef: 20, ls: 0, critical: true })
  })

  it('cd 有富余：长支路 30wd 关键，养护支路 tf=10', () => {
    const r = computeCpm([t('主线', 30), t('养护', 28, { durationUnit: 'cd' })], [], START)
    expect(r.duration).toBe(30)
    expect(r.rows['主线'].critical).toBe(true)
    expect(r.rows['养护']).toMatchObject({ es: 0, ef: 20, ls: 10, tf: 10, ff: 10, critical: false })
  })

  it('逆推平段：养护支路 lf=20 时 ls 取平段最大 s（cdLatestStart 镜像）', () => {
    const r = computeCpm(
      [t('主线', 20), t('养护', 26, { durationUnit: 'cd' }), t('收尾', 4)],
      [l('主线', '收尾'), l('养护', '收尾')],
      START,
    )
    // 收尾 es=20；养护 lf=20 → 26cd 平段 fwd(0..2)=20 → ls=2（非 0）
    expect(r.rows['养护']).toMatchObject({ es: 0, ef: 20, ls: 2, tf: 2, critical: false })
    expect(r.rows['主线']).toMatchObject({ tf: 0, critical: true })
  })

  it('manual cd：es 锁 manualStart，ef=日历换算，不标关键不回传约束', () => {
    const r = computeCpm(
      [t('养护', 28, { durationUnit: 'cd', mode: 'manual', manualStart: 2 }), t('验收', 3)],
      [l('养护', '验收')],
      START,
    )
    expect(r.duration).toBe(25)
    expect(r.rows['养护']).toMatchObject({ es: 2, ef: 22, ls: 2, lf: 22, tf: 0, critical: false })
    expect(r.rows['验收']).toMatchObject({ es: 22, ef: 25, tf: 0, critical: true })
  })

  it('快路径铁律：无 cd 任务时 ctx 不影响结果（逐位一致）', () => {
    const tasks = [t('A', 3), t('B', 2), t('C', 4)]
    const links = [l('A', 'B'), l('B', 'C')]
    const plain = computeCpm(tasks, links)
    const withCtx = computeCpm(tasks, links, { ...START, calendar: { workweek: [1, 2, 3, 4, 5], holidays: ['2026-09-08'] } })
    expect(withCtx).toEqual(plain)
    // 显式 wd 单位同样走快路径
    const wdMarked = computeCpm([t('A', 3, { durationUnit: 'wd' })], [], START)
    expect(wdMarked).toEqual(computeCpm([t('A', 3)], []))
  })

  it('fail-closed：有 cd 任务而缺开工日期 → ok=false 不静默', () => {
    const r = computeCpm([t('养护', 28, { durationUnit: 'cd' })], [], {})
    expect(r.ok).toBe(false)
    expect(r.error).toContain('缺开工日期')
  })

  it('fail-closed：cd 涉非 FS 搭接 → ok=false（SS/FF 双例）', () => {
    const ss = computeCpm([t('养护', 28, { durationUnit: 'cd' }), t('后续', 3)], [l('养护', '后续', 'SS')], START)
    expect(ss.ok).toBe(false)
    expect(ss.error).toContain('FS')
    const ff = computeCpm([t('A', 3), t('养护', 28, { durationUnit: 'cd' })], [l('A', '养护', 'FF')], START)
    expect(ff.ok).toBe(false)
    expect(ff.error).toContain('FS')
  })

  it('planFinish 收紧 + cd：逆推锚点按工作日边界，负时差诚实呈现', () => {
    const r = computeCpm(
      [t('挖土', 5), t('养护', 28, { durationUnit: 'cd' }), t('回填', 5)],
      [l('挖土', '养护'), l('养护', '回填')],
      { ...START, planFinish: 25 },
    )
    expect(r.duration).toBe(30)
    expect(r.rows['养护']).toMatchObject({ ls: 0, tf: -5, critical: true })
    expect(r.rows['挖土']).toMatchObject({ tf: -5, critical: true })
  })
})
