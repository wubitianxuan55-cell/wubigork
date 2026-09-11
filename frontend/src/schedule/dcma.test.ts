// dcma.test.ts — 6.4 DCMA 14 点纯函数全测：逐点可评估判定 + 阈值判定 +
// n/a 诚实口径（缺基线/缺数据日期/未启用资源）。
import { describe, expect, it } from 'vitest'
import { computeCpm } from './cpm'
import { dcmaAudit, DCMA_HIGH_FLOAT_DAYS, DCMA_HIGH_DURATION_DAYS, DCMA_RATIO_LIMIT, DCMA_FS_RATIO_MIN, DCMA_CPLI_MIN, DCMA_BEI_MIN } from './dcma'
import type { SchedProject, SchedTask } from './types'

function task(p: Partial<SchedTask> & { id: string; name?: string }): SchedTask {
  return { name: p.id, duration: 5, level: 1, progress: 0, ...p } as SchedTask
}

/** 健康基线计划：A→B→C 线性 FS，工期 5/10/5。 */
function healthyProject(): SchedProject {
  return {
    name: 'p',
    startDate: '2026-09-01',
    tasks: [task({ id: 'A' }), task({ id: 'B', duration: 10 }), task({ id: 'C' })],
    links: [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'B', to: 'C', type: 'FS', lag: 0 },
    ],
  }
}

describe('DCMA 14 点体检（6.4）', () => {
  it('健康线性计划：14 点全可评估且全过（无约束字段按口径恒过）', () => {
    const p = healthyProject()
    const cpm = computeCpm(p.tasks, p.links)
    const r = dcmaAudit(p, cpm, null)
    expect(r.checks).toHaveLength(14)
    // 缺基线（11/13/14）与未启用资源（10）共 4 点诚实 n/a。
    expect(r.checks.filter((c) => c.na)).toHaveLength(4)
    expect(r.evaluable).toBe(10)
    expect(r.failed).toBe(0)
    expect(r.score).toBe(100)
    // 线性链：FS 占比 100、关键连续 1 段。
    expect(r.checks[3].value).toBe(100)
    expect(r.checks[11].value).toBe(1)
  })

  it('缺逻辑：孤立任务被点名，首末任务豁免', () => {
    const p = healthyProject()
    p.tasks.push(task({ id: 'ISO', duration: 3 })) // 孤立叶任务（非首末）
    const cpm = computeCpm(p.tasks, p.links)
    const c1 = dcmaAudit(p, cpm, null).checks[0]
    expect(c1.na).toBeUndefined()
    expect(c1.value).toBeGreaterThan(0)
    expect(c1.pass).toBe(false)
    expect(c1.offenders).toContain('ISO')
  })

  it('Leads/Lags/FS 占比：负 lag、正 lag、非 FS 各自点名', () => {
    const p = healthyProject()
    p.links.push({ from: 'A', to: 'C', type: 'SS', lag: -2 })
    p.links.push({ from: 'B', to: 'C', type: 'FS', lag: 3 })
    const cpm = computeCpm(p.tasks, p.links)
    const r = dcmaAudit(p, cpm, null)
    const leads = r.checks[1]
    const lags = r.checks[2]
    const fs = r.checks[3]
    expect(leads.value).toBeGreaterThan(0)
    expect(lags.value).toBeGreaterThan(0)
    expect(fs.value).toBe(75) // 4 条关系 3 条 FS
    expect(fs.pass).toBe(false) // 75% < 90% 下限
    expect(fs.offenders.some((o) => o.includes('SS'))).toBe(true)
  })

  it('高浮时+长工期：超阈值任务计数并点名', () => {
    const p = healthyProject()
    p.tasks[1].duration = DCMA_HIGH_DURATION_DAYS + 10 // B 长工期 → 也拉长 C 浮时
    const cpm = computeCpm(p.tasks, p.links)
    const r = dcmaAudit(p, cpm, null)
    const hd = r.checks[7]
    expect(hd.value).toBeGreaterThan(0)
    expect(hd.offenders.join('、')).toContain('B')
    // 高浮：A 前移后 C 的浮时并不必然>44；构造纯高浮任务
    const p2 = healthyProject()
    p2.tasks.push(task({ id: 'FLOAT', duration: 1 }))
    p2.tasks.push(task({ id: 'TAIL', duration: 60 }))
    p2.links.push({ from: 'A', to: 'FLOAT', type: 'FS', lag: 0 })
    p2.links.push({ from: 'A', to: 'TAIL', type: 'FS', lag: 0 })
    const cpm2 = computeCpm(p2.tasks, p2.links)
    const hf = dcmaAudit(p2, cpm2, null).checks[5]
    expect(hf.value).toBeGreaterThan(0)
    expect(hf.offenders.some((o) => o.startsWith('FLOAT'))).toBe(true)
  })

  it('负浮时：健康计划无负浮时（0 个，通过）', () => {
    const p = healthyProject()
    const cpm = computeCpm(p.tasks, p.links)
    const nf = dcmaAudit(p, cpm, null).checks[6]
    expect(nf.value).toBe(0)
    expect(nf.pass).toBe(true)
  })

  it('资源缺失：启用资源表后点名未分配叶任务；未启用则 n/a', () => {
    const p = healthyProject()
    const cpm = computeCpm(p.tasks, p.links)
    const r0 = dcmaAudit(p, cpm, null)
    expect(r0.checks[9].na).toBeTruthy()
    p.resources = [{ id: 'r1', name: '力工', type: 'work' }]
    p.assignments = [{ taskId: 'A', resourceId: 'r1', units: 1 }]
    p.assignments.push({ taskId: 'B', resourceId: 'r1', units: 1 })
    const r1 = dcmaAudit(p, cpm, null)
    const c10 = r1.checks[9]
    expect(c10.na).toBeUndefined()
    expect(c10.value).toBeGreaterThan(0)
    expect(c10.offenders).toContain('C')
  })

  it('错过基线：当前 EF 晚于基线 EF 且未完成的任务被点名', () => {
    const p = healthyProject()
    p.baseline = {
      name: '基线',
      savedAt: '2026-09-01 08:00',
      duration: 20,
      rows: {
        A: { name: 'A', es: 0, ef: 5, dur: 5, critical: true },
        B: { name: 'B', es: 5, ef: 15, dur: 10, critical: true },
        C: { name: 'C', es: 15, ef: 20, dur: 5, critical: true },
      },
    }
    p.tasks[1].duration = 20 // B 拖长 → C EF 晚于基线
    const cpm = computeCpm(p.tasks, p.links)
    const r = dcmaAudit(p, cpm, null)
    const c11 = r.checks[10]
    expect(c11.na).toBeUndefined()
    expect(c11.value).toBeGreaterThan(0)
    expect(c11.offenders.length).toBeGreaterThan(0)
  })

  it('关键路径连续性：单链=1 段通过；双独立关键段=2 段不通过', () => {
    const p = healthyProject()
    const cpm = computeCpm(p.tasks, p.links)
    expect(dcmaAudit(p, cpm, null).checks[11].value).toBe(1)
    // 双链并列（均关键）：A→B 与 X→Y 两条独立关键线路。
    const p2 = healthyProject()
    p2.tasks.push(task({ id: 'X' }), task({ id: 'Y', duration: 15 }))
    p2.links.push({ from: 'X', to: 'Y', type: 'FS', lag: 0 })
    const cpm2 = computeCpm(p2.tasks, p2.links)
    const r2 = dcmaAudit(p2, cpm2, null)
    const c12 = r2.checks[11]
    expect(c12.value).toBe(2)
    expect(c12.pass).toBe(false)
  })

  it('CPLI/BEI：缺基线或缺数据日期 → n/a；提供后可评估', () => {
    const p = healthyProject()
    const cpm = computeCpm(p.tasks, p.links)
    const r0 = dcmaAudit(p, cpm, null)
    expect(r0.checks[12].na).toBeTruthy()
    expect(r0.checks[13].na).toBeTruthy()

    p.baseline = {
      name: '基线', savedAt: '2026-09-01 08:00', duration: 20,
      rows: {
        A: { name: 'A', es: 0, ef: 5, dur: 5, critical: true },
        B: { name: 'B', es: 5, ef: 15, dur: 10, critical: true },
        C: { name: 'C', es: 15, ef: 20, dur: 5, critical: true },
      },
    }
    const r1 = dcmaAudit(p, cpm, 10)
    const cpli = r1.checks[12]
    const bei = r1.checks[13]
    expect(cpli.na).toBeUndefined()
    // CPLI=(20-10)/(20-10)=1 ≥0.95 通过
    expect(cpli.value).toBe(1)
    expect(cpli.pass).toBe(true)
    // BEI：数据日期 10 前应完成 A（基线 ef=5）；progress=0 → 0/1=0 不通过
    expect(bei.na).toBeUndefined()
    expect(bei.value).toBe(0)
    expect(bei.pass).toBe(false)
    // 完成后 BEI=1
    p.tasks[0].progress = 100
    const r2 = dcmaAudit(p, cpm, 10)
    expect(r2.checks[13].value).toBe(1)
    expect(r2.checks[13].pass).toBe(true)
  })

  it('CPM 循环依赖：依赖排程的点全部 n/a，不伪造结论', () => {
    const p = healthyProject()
    p.links.push({ from: 'C', to: 'A', type: 'FS', lag: 0 })
    const cpm = computeCpm(p.tasks, p.links)
    expect(cpm.ok).toBe(false)
    const r = dcmaAudit(p, cpm, null)
    const nas = r.checks.filter((c) => c.na).map((c) => c.id)
    for (const id of [1, 6, 7, 9, 11, 12]) {
      expect(nas).toContain(id)
    }
    // CPM 断裂时整体分不可信：score=null（逐点结论仍在）。
    expect(r.score).toBeNull()
  })

  it('阈值常量与 DCMA 标准一致（44 天 / 5% / 90% / 0.95）', () => {
    expect(DCMA_HIGH_FLOAT_DAYS).toBe(44)
    expect(DCMA_HIGH_DURATION_DAYS).toBe(44)
  })
})

// ── v4.226 阈值数据资产（规范知识出内核）────────────────────────────────
describe('DCMA 阈值数据资产（v4.226）', () => {
  it('内置默认表漂移守卫：与行业标准值逐字段一致', () => {
    expect(DCMA_RATIO_LIMIT).toBe(5)
    expect(DCMA_FS_RATIO_MIN).toBe(90)
    expect(DCMA_CPLI_MIN).toBe(0.95)
    expect(DCMA_BEI_MIN).toBe(0.95)
  })

  it('阈值覆盖：部分字段覆盖生效，展示串与判定同步（缺省字段用内置默认）', () => {
    // B 工期抬到 50 天：默认长工期线（>44）应判未过；把线抬到 60 后转通过。
    const p = healthyProject()
    p.tasks[1].duration = 50
    const cpm = computeCpm(p.tasks, p.links)
    const base = dcmaAudit(p, cpm, null)
    expect(base.checks[7].pass).toBe(false)
    expect(base.checks[7].threshold).toContain('>44')

    const tuned = dcmaAudit(p, cpm, null, { highDurationDays: 60, ratioLimitPct: 10 })
    expect(tuned.checks[7].pass).toBe(true)
    expect(tuned.checks[7].threshold).toContain('>60')
    // 未覆盖字段保持默认：FS 占比下限仍显示 ≥90%（不在覆盖字段里）。
    expect(tuned.checks[3].threshold).toBe('≥90%')
    // 阈值抬到 60 后 B 不再超标：长工期实测占比归零。
    expect(tuned.checks[7].value).toBe(0)
  })
})
