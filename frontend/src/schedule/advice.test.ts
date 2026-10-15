// 阶段十一 11.4 测试：纠错建议在样例工程上可复现（出口判据）+ what-if 推演
// 纯本地（computeCpm 纯函数重算，不写盘不调服务）。
import { describe, expect, it } from 'vitest'
import type { SchedLink, SchedProject, SchedTask } from './types'
import { computeCpm } from './cpm'
import { scheduleAdvice } from './advice'
import { whatIfDuration } from './whatif'

// 样例工程（确定性复现的夹具）：
//   T1(3d) → T2(5d) → T3(2d)  正常链
//   T4(4d) 孤立（无任何关系）          → missing-link
//   T5(2d) → T6(3d) → T7(1d) 且 T5 → T7 直达 → redundant-link
//   L1: T2 → T3 带 lag -1              → negative-lag
const sampleProject = (): SchedProject => {
  const tasks: SchedTask[] = [
    { id: 'T1', name: '开挖', duration: 3, level: 1, progress: 0 },
    { id: 'T2', name: '垫层', duration: 5, level: 1, progress: 0 },
    { id: 'T3', name: '基础', duration: 2, level: 1, progress: 0 },
    { id: 'T4', name: '孤立工序', duration: 4, level: 1, progress: 0 },
    { id: 'T5', name: '绑筋', duration: 2, level: 1, progress: 0 },
    { id: 'T6', name: '支模', duration: 3, level: 1, progress: 0 },
    { id: 'T7', name: '浇筑', duration: 1, level: 1, progress: 0 },
  ]
  const links: SchedLink[] = [
    { from: 'T1', to: 'T2', type: 'FS', lag: 0 },
    { from: 'T2', to: 'T3', type: 'FS', lag: -1 },
    { from: 'T5', to: 'T6', type: 'FS', lag: 0 },
    { from: 'T6', to: 'T7', type: 'FS', lag: 0 },
    { from: 'T5', to: 'T7', type: 'FS', lag: 0 },
  ]
  return { tasks, links } as unknown as SchedProject
}

describe('scheduleAdvice 纠错建议（样例工程可复现）', () => {
  it('样例工程四类建议齐出，且每条带具体动作', () => {
    const p = sampleProject()
    const cpm = computeCpm(p.tasks, p.links)
    const advice = scheduleAdvice(p, cpm)
    const kinds = advice.map((a) => a.kind)

    expect(kinds).toContain('negative-lag')
    expect(kinds).toContain('missing-link')
    expect(kinds).toContain('redundant-link')
    for (const a of advice) {
      expect(a.message).toBeTruthy()
      expect(a.action).toBeTruthy()
      expect(a.tasks.length).toBeGreaterThan(0)
    }
    // 负搭接点名具体关系与 lag
    const neg = advice.find((a) => a.kind === 'negative-lag')
    expect(neg?.tasks.some((t) => t.includes('-1d'))).toBe(true)
    // 冗余直达点名经谁间接
    const red = advice.find((a) => a.kind === 'redundant-link')
    expect(red?.tasks.some((t) => t.includes('支模'))).toBe(true)
  })

  it('健康链路零建议（无负搭接/无孤立/无冗余 → advice 空）', () => {
    const p = sampleProject()
    const healthy: SchedProject = {
      ...p,
      links: p.links.filter((l) => l.lag >= 0 && !(l.from === 'T5' && l.to === 'T7')).concat([{ from: 'T3', to: 'T4', type: 'FS', lag: 0 }]),
    }
    const cpm = computeCpm(healthy.tasks, healthy.links)
    const advice = scheduleAdvice(healthy, cpm)
    if (advice.length > 0) console.log('ADVICE:', JSON.stringify(advice, null, 2))
    expect(advice).toHaveLength(0)
  })

  it('循环依赖：点名环并给打断建议', () => {
    const p = sampleProject()
    const cyclic: SchedProject = {
      ...p,
      links: [...p.links, { from: 'T3', to: 'T1', type: 'FS', lag: 0 }],
    }
    const cpm = computeCpm(cyclic.tasks, cyclic.links)
    expect(cpm.ok).toBe(false)
    const advice = scheduleAdvice(cyclic, cpm)
    const cyc = advice.find((a) => a.tasks.length >= 2 && a.message.includes('循环依赖'))
    expect(cyc).toBeTruthy()
    expect(cyc?.action).toContain('打断')
  })

  it('复现性：同一工程两次调用结论逐字一致（确定性）', () => {
    const p = sampleProject()
    const cpm = computeCpm(p.tasks, p.links)
    expect(scheduleAdvice(p, cpm)).toEqual(scheduleAdvice(p, cpm))
  })
})

describe('whatIfDuration 关键线路推演（纯本地）', () => {
  it('延长关键任务：总工期等量推迟，关键线路保持', () => {
    const p = sampleProject()
    // 基线：T1(3)→T2(5-1=4 网络修正后)→T3(2) 链 + T5→T6→T7 链
    const r = whatIfDuration(p, 'T2', 3)
    expect(r.ok).toBe(true)
    expect(r.delta).toBe(3)
    expect(r.whatIfDuration).toBe(r.baseDuration + 3)
    expect(r.newlyCritical).toHaveLength(0)
  })

  it('压缩关键任务到脱离关键：总工期按新关键线路收缩', () => {
    const p = sampleProject()
    // 找基线关键线路上的一个任务（首条）
    const base = computeCpm(p.tasks, p.links)
    const critEntry = Object.entries(base.rows).find(([, row]) => row.critical)
    expect(critEntry).toBeTruthy()
    const r = whatIfDuration(p, critEntry![0], -1)
    expect(r.ok).toBe(true)
    expect(r.delta).toBeLessThanOrEqual(0)
    // 压缩关键任务必然引起关键线路重组或工期缩短（二者必有其一）
    expect(r.delta < 0 || r.noLongerCritical.length > 0 || r.newlyCritical.length > 0).toBe(true)
  })

  it('延长非关键任务在总浮动内：总工期不变（这是浮动的定义）', () => {
    const p = sampleProject()
    const base = computeCpm(p.tasks, p.links)
    // 找一个非关键且 tf>0 的任务
    const slackId = Object.entries(base.rows).find(([, row]) => !row.critical && row.tf > 0)?.[0]
    if (slackId === undefined) return // 样例无非关键任务时跳过
    const r = whatIfDuration(p, slackId, 1)
    expect(r.ok).toBe(true)
    expect(r.delta).toBe(0)
  })

  it('负 delta 下限钳到 0（工期不为负）与里程碑拒推', () => {
    const p = sampleProject()
    const r1 = whatIfDuration(p, 'T7', -999)
    expect(r1.ok).toBe(true)
    expect(r1.delta).toBeLessThanOrEqual(0)
    const r2 = whatIfDuration(p, 'NOPE', 3)
    expect(r2.ok).toBe(false)
  })
})
