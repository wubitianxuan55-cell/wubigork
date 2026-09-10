// evm.test.ts — 6.6 跨域 EVM 纯函数：三数两指数 + 口径注全 + 诚实 n/a。
import { describe, expect, it } from 'vitest'
import { computeCpm } from './cpm'
import { computeEvm } from './evm'
import type { SchedProject, SchedTask } from './types'

function task(id: string, duration: number, progress = 0): SchedTask {
  return { id, name: id, duration, level: 1, progress }
}

/** A(5)→B(10)→C(5) 线性链，基线=当前计划，A/B 各挂成本资源。 */
function project(): SchedProject {
  return {
    name: 'p',
    startDate: '2026-09-01',
    tasks: [task('A', 5), task('B', 10), task('C', 5)],
    links: [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'B', to: 'C', type: 'FS', lag: 0 },
    ],
    baseline: {
      name: '基线',
      savedAt: '2026-09-01 08:00',
      duration: 20,
      rows: {
        A: { name: 'A', es: 0, ef: 5, dur: 5, critical: true },
        B: { name: 'B', es: 5, ef: 15, dur: 10, critical: true },
        C: { name: 'C', es: 15, ef: 20, dur: 5, critical: true },
      },
    },
    resources: [{ id: 'r1', name: '班组', type: 'work', standardRate: 100 }],
    assignments: [
      { taskId: 'A', resourceId: 'r1', units: 1 },
      { taskId: 'C', resourceId: 'r1', units: 2 },
    ],
  }
}

describe('跨域 EVM（6.6）', () => {
  it('三数出：PV 基线线性分摊、EV=完成率×预算、BAC=任务预算合计', () => {
    const p = project()
    const cpm = computeCpm(p.tasks, p.links)
    // 数据日期=10：A 基线 [0,5] 已过→全额；B 基线 [5,15] 过半→50%。
    // A 预算=5天×100=500；C 预算=5×2×100=1000；B 无分配无固定=0。
    const r = computeEvm(p, cpm, { dataDate: 10 })
    expect(r.ok).toBe(true)
    expect(r.pv.value).toBe(500) // A 500 全额 + C 0（[15,20] 未开始）
    expect(r.ev.value).toBe(0)
    expect(r.bac).toBe(1500) // 500+0+1000
    // 口径注全（6.6 判据）：每个数都有来源说明。
    for (const k of ['pv', 'ev', 'ac', 'spi', 'cpi'] as const) {
      expect(r[k].note.length).toBeGreaterThan(10)
    }
  })

  it('EV 完成率 + SPI；SV=EV−PV', () => {
    const p = project()
    p.tasks[1].progress = 50 // B 完成 50%：EV=0×0.5 + 0 = 0（B 无预算）
    p.tasks[0].progress = 100 // A 完成：EV=500
    const cpm = computeCpm(p.tasks, p.links)
    const r = computeEvm(p, cpm, { dataDate: 10 })
    expect(r.ev.value).toBe(500)
    expect(r.spi.value).toBe(1) // 500/500
    expect(r.sv).toBe(0)
  })

  it('进度落后：EV<PV → SPI<1 且 SV<0（口径注在）', () => {
    const p = project()
    const cpm = computeCpm(p.tasks, p.links)
    // 数据日期=10：PV=500；无任何完成 → EV=0 → SPI=0。
    const r = computeEvm(p, cpm, { dataDate: 10 })
    expect(r.pv.value).toBe(500)
    expect(r.ev.value).toBe(0)
    expect(r.spi.value).toBe(0)
    expect(r.sv).toBe(-500)
  })

  it('CPI：未录 AC → null + 口径注明来源；录入后出指数与 CV', () => {
    const p = project()
    p.tasks[0].progress = 100
    const cpm = computeCpm(p.tasks, p.links)
    const r0 = computeEvm(p, cpm, { dataDate: 10 })
    expect(r0.ac.value).toBeNull()
    expect(r0.cpi.value).toBeNull()
    expect(r0.cv).toBeNull()
    // EV=500，AC 录 400 → CPI=1.25，CV=+100（成本节余）。
    const r1 = computeEvm(p, cpm, { dataDate: 10, actualCost: 400 })
    expect(r1.cpi.value).toBe(1.25)
    expect(r1.cv).toBe(100)
  })

  it('里程碑基线行（dur=0）：在完成点整额兑现 PV', () => {
    const p = project()
    p.tasks.push({ id: 'M', name: '开工里程碑', duration: 0, level: 1, progress: 0, isMilestone: true })
    p.baseline!.rows['M'] = { name: 'M', es: 0, ef: 0, dur: 0, critical: true }
    p.resources!.push({ id: 'r2', name: '增项', type: 'cost' })
    p.assignments!.push({ taskId: 'M', resourceId: 'r2', amount: 300 })
    const cpm = computeCpm(p.tasks, p.links)
    // 数据日期=0：里程碑在 ef=0 兑现 → 计入 PV；日期=-1 不可能，改用 dd=0 含 ef。
    const r = computeEvm(p, cpm, { dataDate: 0 })
    // M 300 全额 + A [0,5] dd=0 → 0% → PV=300。
    expect(r.pv.value).toBe(300)
  })

  it('无基线 / CPM 断裂：ok=false + 说明，不伪造三数', () => {
    const p = project()
    delete p.baseline
    const cpm = computeCpm(p.tasks, p.links)
    const r0 = computeEvm(p, cpm, { dataDate: 10 })
    expect(r0.ok).toBe(false)
    expect(r0.error).toContain('基线')
    p.baseline = project().baseline
    p.links.push({ from: 'C', to: 'A', type: 'FS', lag: 0 })
    const bad = computeCpm(p.tasks, p.links)
    const r1 = computeEvm(p, bad, { dataDate: 10 })
    expect(r1.ok).toBe(false)
  })
})
