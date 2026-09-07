/**
 * usage.test.ts — 资源使用/超载计算用例（v4.137 #12）
 *
 * 用例口径：同资源重叠任务顶起 workload 超 availability（重叠日 2>1 超载、
 * 非重叠日不超载）；两人各 1 任务负载按资源分摊互不顶格；资源个人日历在项目
 * 工作日上收紧（假期/周末对应的项目工作日 availability=0 → 超载，未设日历
 * 同日不超载）；units/maxUnits 决定超载边界；material/cost 不进 rows、无分配
 * 工时资源=全零行、分组行分配跳过、里程碑进 tasks 但 ef=es 不产生负载；
 * cpm 循环依赖 fail-closed；rows 按名称 zh 序。
 */
import { describe, expect, it } from 'vitest'
import { computeCpm } from './cpm'
import { computeUsage } from './usage'
import type { SchedAssignment, SchedLink, SchedProject, SchedResource, SchedTask } from './types'

const START = '2026-01-05' // 周一：wd0=01-05 … wd4=01-09、wd5=01-10（周六）

function t(id: string, duration: number, extra?: Partial<SchedTask>): SchedTask {
  return { id, name: id, duration, level: 1, progress: 0, ...extra }
}
function group(id: string, name: string): SchedTask {
  return { id, name, duration: 0, level: 0, progress: 0 }
}
function res(id: string, name: string, extra?: Partial<SchedResource>): SchedResource {
  return { id, name, type: 'work', ...extra }
}
function assign(taskId: string, resourceId: string, units?: number): SchedAssignment {
  return units === undefined ? { taskId, resourceId } : { taskId, resourceId, units }
}
function l(from: string, to: string, type: SchedLink['type'] = 'FS', lag = 0): SchedLink {
  return { from, to, type, lag }
}
function proj(tasks: SchedTask[], links: SchedLink[], resources?: SchedResource[], assignments?: SchedAssignment[]): SchedProject {
  return { name: 'P', startDate: START, tasks, links, resources, assignments }
}

describe('computeUsage 基础负载/超载', () => {
  it('同资源两个部分重叠任务（units=1）：重叠日 workload=2>1 超载，非重叠日不超载', () => {
    // A(3) es0-3；B(2) SS+1 → es1-3：wd0 只有 A（1 不超载），wd1/wd2 A+B（2>1 超载）
    const tasks = [t('A', 3), t('B', 2)]
    const links = [l('A', 'B', 'SS', 1)]
    const project = proj(tasks, links, [res('R1', '张工')], [assign('A', 'R1'), assign('B', 'R1')])
    const cpm = computeCpm(tasks, links)
    expect(cpm.rows.B).toMatchObject({ es: 1, ef: 3 })

    const u = computeUsage(project, cpm)
    expect(u.ok).toBe(true)
    expect(u.days).toBe(3)
    expect(u.rows).toHaveLength(1)
    const row = u.rows[0]
    expect(row.tasks).toEqual([
      { taskId: 'A', name: 'A', es: 0, ef: 3, units: 1 },
      { taskId: 'B', name: 'B', es: 1, ef: 3, units: 1 },
    ])
    expect(row.cells.map((c) => c.workload)).toEqual([1, 2, 2])
    expect(row.cells.map((c) => c.overloaded)).toEqual([false, true, true])
    expect(row.peakWorkload).toBe(2)
    expect(row.overloadDays).toBe(2)
    expect(row.totalWorkload).toBe(5)
    expect(u.totalOverloadDays).toBe(2)
  })

  it('两人各 1 任务重叠：负载按资源分摊，各自峰值 1 互不顶格', () => {
    const tasks = [t('A', 3), t('B', 2)]
    const u = computeUsage(
      proj(tasks, [], [res('R1', '甲'), res('R2', '乙')], [assign('A', 'R1'), assign('B', 'R2')]),
      computeCpm(tasks, []),
    )
    expect(u.rows.map((r) => r.resourceId)).toEqual(['R1', 'R2'])
    expect(u.rows[0].cells.map((c) => c.workload)).toEqual([1, 1, 1])
    expect(u.rows[1].cells.map((c) => c.workload)).toEqual([1, 1, 0])
    expect(u.rows[0].overloadDays).toBe(0)
    expect(u.rows[1].overloadDays).toBe(0)
    expect(u.rows[0].totalWorkload).toBe(3)
    expect(u.rows[1].totalWorkload).toBe(2)
    expect(u.totalOverloadDays).toBe(0)
  })
})

describe('资源个人日历收紧可用性', () => {
  it('项目工作日命中个人假期 → availability=0 → 超载；未设日历同日不超载', () => {
    // wd1=2026-01-06（周二）：R1 个人日历放假 availability=0，workload=1 → 超载
    const tasks = [t('A', 5)]
    const u = computeUsage(
      proj(
        tasks,
        [],
        [res('R1', 'R1', { calendar: { workweek: [1, 2, 3, 4, 5], holidays: ['2026-01-06'] } }), res('R2', 'R2')],
        [assign('A', 'R1'), assign('A', 'R2')],
      ),
      computeCpm(tasks, []),
    )
    expect(u.days).toBe(5)
    const r1 = u.rows.find((r) => r.resourceId === 'R1')!
    expect(r1.cells[1]).toMatchObject({ workload: 1, availability: 0, overloaded: true })
    expect(r1.cells[0].availability).toBe(1)
    expect(r1.cells[2].availability).toBe(1)
    expect(r1.overloadDays).toBe(1)
    const r2 = u.rows.find((r) => r.resourceId === 'R2')!
    expect(r2.cells[1]).toMatchObject({ workload: 1, availability: 1, overloaded: false })
    expect(r2.overloadDays).toBe(0)
    expect(u.totalOverloadDays).toBe(1)
  })

  it('项目 7 天工作制 + 资源只做周一~五：项目周六/周日 availability=0 → 超载', () => {
    // startDate 周一：wd5=01-10（周六）、wd6=01-11（周日）项目照常推进，资源休假
    const tasks = [t('A', 7)]
    const project = { ...proj(tasks, [], [res('R1', 'R1', { calendar: { workweek: [1, 2, 3, 4, 5], holidays: [] } })], [assign('A', 'R1')]), calendar: { workweek: [0, 1, 2, 3, 4, 5, 6], holidays: [] } }
    const u = computeUsage(project, computeCpm(tasks, []))
    expect(u.days).toBe(7)
    const row = u.rows[0]
    expect(row.cells.map((c) => c.availability)).toEqual([1, 1, 1, 1, 1, 0, 0])
    expect(row.cells.map((c) => c.workload)).toEqual([1, 1, 1, 1, 1, 1, 1])
    expect(row.overloadDays).toBe(2)
    expect(u.totalOverloadDays).toBe(2)
  })
})

describe('units 与 maxUnits 决定超载边界', () => {
  it('units=2 无 maxUnits：availability=1 全程超载；maxUnits=2 后全程不超载', () => {
    const tasks = [t('A', 3)]
    const over = computeUsage(proj(tasks, [], [res('R1', 'R1')], [assign('A', 'R1', 2)]), computeCpm(tasks, []))
    expect(over.rows[0].cells.map((c) => c.workload)).toEqual([2, 2, 2])
    expect(over.rows[0].cells.every((c) => c.overloaded)).toBe(true)
    expect(over.rows[0].peakWorkload).toBe(2)
    expect(over.rows[0].overloadDays).toBe(3)
    expect(over.rows[0].totalWorkload).toBe(6)
    expect(over.totalOverloadDays).toBe(3)

    const balanced = computeUsage(
      proj(tasks, [], [res('R1', 'R1', { maxUnits: 2 })], [assign('A', 'R1', 2)]),
      computeCpm(tasks, []),
    )
    expect(balanced.rows[0].cells.every((c) => !c.overloaded)).toBe(true)
    expect(balanced.rows[0].cells.every((c) => c.availability === 2)).toBe(true)
    expect(balanced.totalOverloadDays).toBe(0)
  })
})

describe('rows 进出场与零负载', () => {
  it('material/cost 不进 rows；无分配工时资源保留=全零行', () => {
    const tasks = [t('A', 2)]
    const resources = [res('M1', '水泥', { type: 'material' }), res('C1', '管理费', { type: 'cost' }), res('R1', '备用工')]
    const u = computeUsage(proj(tasks, [], resources, []), computeCpm(tasks, []))
    expect(u.ok).toBe(true)
    expect(u.rows.map((r) => r.resourceId)).toEqual(['R1'])
    expect(u.rows[0].tasks).toEqual([])
    expect(u.rows[0].cells).toEqual([
      { wd: 0, workload: 0, availability: 1, overloaded: false },
      { wd: 1, workload: 0, availability: 1, overloaded: false },
    ])
    expect(u.rows[0].peakWorkload).toBe(0)
    expect(u.rows[0].totalWorkload).toBe(0)
    expect(u.totalOverloadDays).toBe(0)
  })

  it('分组行分配跳过；里程碑进 tasks 但 ef=es 不产生负载', () => {
    // A(2)→M（里程碑 FS）：M es=ef=2（在 days=2 时间轴外也无所谓，永不占用）
    const tasks = [group('G1', '一期'), t('A', 2), t('M', 0, { isMilestone: true })]
    const links = [l('A', 'M')]
    const assignments = [assign('G1', 'R1'), assign('A', 'R1'), assign('M', 'R1')]
    const u = computeUsage(proj(tasks, links, [res('R1', 'R1')], assignments), computeCpm(tasks, links))
    expect(u.ok).toBe(true)
    expect(u.days).toBe(2)
    expect(u.rows[0].tasks.map((x) => x.taskId)).toEqual(['A', 'M'])
    expect(u.rows[0].tasks[1]).toMatchObject({ es: 2, ef: 2 })
    expect(u.rows[0].cells.map((c) => c.workload)).toEqual([1, 1])
    expect(u.rows[0].overloadDays).toBe(0)
  })

  it('0 任务：days=0，rows 仍含工时资源（cells 空）且 ok=true', () => {
    const u = computeUsage(proj([], [], [res('R1', 'R1')], []), computeCpm([], []))
    expect(u.ok).toBe(true)
    expect(u.days).toBe(0)
    expect(u.rows).toHaveLength(1)
    expect(u.rows[0].cells).toEqual([])
    expect(u.rows[0].peakWorkload).toBe(0)
    expect(u.totalOverloadDays).toBe(0)
  })
})

describe('fail-closed 与排序', () => {
  it('cpm 循环依赖 fail-closed：ok=false、days=0、rows=[]、totalOverloadDays=0', () => {
    const tasks = [t('A', 1), t('B', 1)]
    const links = [l('A', 'B'), l('B', 'A')]
    const cpm = computeCpm(tasks, links)
    expect(cpm.ok).toBe(false)
    const u = computeUsage(proj(tasks, links, [res('R1', 'R1')], [assign('A', 'R1')]), cpm)
    expect(u).toEqual({ ok: false, error: cpm.error, days: 0, rows: [], totalOverloadDays: 0 })
  })

  it('rows 按名称 localeCompare zh 序（阿<李<张，非码点序）', () => {
    const resources = [res('R3', '张三'), res('R1', '阿明'), res('R2', '李四')]
    const u = computeUsage(proj([t('A', 1)], [], resources, []), computeCpm([t('A', 1)], []))
    expect(u.rows.map((r) => r.name)).toEqual(['阿明', '李四', '张三'])
  })
})
