import { describe, expect, it } from 'vitest'
import { computeCpm } from './cpm'
import { computeCosts } from './cost'
import { normalizeProject } from './store'
import type { SchedAssignment, SchedProject, SchedResource, SchedTask } from './types'

function t(id: string, duration: number, extra?: Partial<SchedTask>): SchedTask {
  return { id, name: id, duration, level: 1, progress: 0, ...extra }
}
function g(id: string, extra?: Partial<SchedTask>): SchedTask {
  return { id, name: id, duration: 0, level: 0, progress: 0, ...extra }
}
function res(id: string, type: SchedResource['type'], extra?: Partial<SchedResource>): SchedResource {
  return { id, name: id, type, ...extra }
}
function asg(taskId: string, resourceId: string, extra?: Partial<SchedAssignment>): SchedAssignment {
  return { taskId, resourceId, ...extra }
}

function project(over: Partial<SchedProject> = {}): SchedProject {
  return {
    name: '成本样板',
    startDate: '2026-01-05',
    tasks: [t('A', 3), t('B', 2)],
    links: [],
    ...over,
  }
}

/** 镜像场景样板：A(3,work 人力 300 元/工日)→B(2,material 钢材) + 成本资源差旅（镜像 Go costProject） */
function costProject(): SchedProject {
  return project({
    tasks: [t('A', 3), t('B', 2, { fixedCost: 50 }), g('G1'), t('C', 4), t('M', 0, { isMilestone: true })],
    links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }],
    resources: [
      res('r1', 'work', { standardRate: 300, costPerUse: 200 }),
      res('r2', 'material', { standardRate: 55, unit: 't' }),
      res('r3', 'cost'),
    ],
    assignments: [
      asg('A', 'r1', { units: 2 }),
      asg('B', 'r2', { quantity: 10 }),
      asg('B', 'r3', { amount: 300 }),
      asg('M', 'r1', { units: 1 }),
      asg('G1', 'r1'), // 分组行分配：引擎跳过（闸在 Validate）
    ],
  })
}

describe('computeCosts 成本 rollup（镜像 Go TestComputeCosts*）', () => {
  it('work：工期×units×费率+每次使用；units 缺省=1', () => {
    const p = project({ resources: [res('r1', 'work', { standardRate: 100, costPerUse: 200 })], assignments: [asg('A', 'r1', { units: 2 })] })
    const r = computeCosts(p, computeCpm(p.tasks, p.links))
    expect(r.ok).toBe(true)
    expect(r.rows.A).toEqual({ fixed: 0, assigned: 3 * 2 * 100 + 200, total: 800 })
    expect(r.rows.B).toEqual({ fixed: 0, assigned: 0, total: 0 })
    expect(r.byResource.r1).toBe(800)
    expect(r.total).toBe(800)
  })

  it('material：quantity×单价+每次使用；cost：amount', () => {
    const p = project({
      resources: [res('m', 'material', { standardRate: 55, unit: 't' }), res('c', 'cost')],
      assignments: [asg('A', 'm', { quantity: 10 }), asg('B', 'c', { amount: 300 })],
    })
    const r = computeCosts(p, computeCpm(p.tasks, p.links))
    expect(r.rows.A.assigned).toBe(550)
    expect(r.rows.B.assigned).toBe(300)
    expect(r.byResource).toEqual({ m: 550, c: 300 })
    expect(r.total).toBe(850)
  })

  it('fixedCost 计入任务明细；里程碑工时分配只剩每次使用', () => {
    const p = project({
      tasks: [t('A', 3, { fixedCost: 120.5 }), t('M', 5, { isMilestone: true })],
      resources: [res('r1', 'work', { standardRate: 300, costPerUse: 200 })],
      assignments: [asg('M', 'r1')],
    })
    const r = computeCosts(p, computeCpm(p.tasks, p.links))
    expect(r.rows.A).toEqual({ fixed: 120.5, assigned: 0, total: 120.5 })
    expect(r.rows.M).toEqual({ fixed: 0, assigned: 200, total: 200 }) // 里程碑 effDur=0，费率项归零
    expect(r.total).toBe(320.5)
  })

  it('镜像样板：分组行分配跳过、rows 只含叶任务、byResource 跨任务聚合', () => {
    const r = computeCosts(costProject(), computeCpm(costProject().tasks, costProject().links))
    expect(Object.keys(r.rows).sort()).toEqual(['A', 'B', 'C', 'M'])
    expect(r.rows.A.assigned).toBe(3 * 2 * 300 + 200) // 2000
    expect(r.rows.B).toEqual({ fixed: 50, assigned: 10 * 55 + 300, total: 900 }) // 550+300+50
    expect(r.byResource.r1).toBe(2000 + 200) // A + M 的每次使用
    expect(r.byResource.r2).toBe(550)
    expect(r.byResource.r3).toBe(300)
    expect(r.total).toBe(2000 + 900 + 0 + 200) // C 无分配
  })

  it('无资源维度的旧计划：ok 且 total=0', () => {
    const r = computeCosts(project(), computeCpm(project().tasks, project().links))
    expect(r.ok).toBe(true)
    expect(r.total).toBe(0)
    expect(r.byResource).toEqual({})
  })

  it('CPM 循环依赖：ok=false 成本不出（fail-closed）', () => {
    const p = project({ links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }, { from: 'B', to: 'A', type: 'FS', lag: 0 }] })
    const r = computeCosts(p, computeCpm(p.tasks, p.links))
    expect(r.ok).toBe(false)
    expect(r.rows).toEqual({})
    expect(r.total).toBe(0)
  })

  it('悬空引用的分配跳过（闸在 Validate，引擎只算在册数据）', () => {
    const p = project({
      resources: [res('r1', 'work', { standardRate: 100 })],
      assignments: [asg('A', 'r1'), asg('A', 'ghost'), asg('ghostTask', 'r1')],
    })
    const r = computeCosts(p, computeCpm(p.tasks, p.links))
    expect(r.rows.A.assigned).toBe(300)
    expect(r.byResource).toEqual({ r1: 300 })
  })

  it('金额四舍五入到分', () => {
    const p = project({
      tasks: [t('A', 3)],
      resources: [res('r1', 'work', { standardRate: 33.335 })],
      assignments: [asg('A', 'r1')],
    })
    const r = computeCosts(p, computeCpm(p.tasks, p.links))
    expect(r.rows.A.assigned).toBe(100.01) // 3×33.335=100.005 → 100.01
    expect(r.total).toBe(100.01)
  })

  it('units=0 合法（该分配成本归零）', () => {
    const p = project({
      resources: [res('r1', 'work', { standardRate: 100 })],
      assignments: [asg('A', 'r1', { units: 0 })],
    })
    const r = computeCosts(p, computeCpm(p.tasks, p.links))
    expect(r.rows.A.assigned).toBe(0)
  })

  it('手动任务工期照常计成本（manual 只影响排程，不影响工期口径）', () => {
    const p = project({
      tasks: [t('A', 4, { mode: 'manual', manualStart: 2 })],
      resources: [res('r1', 'work', { standardRate: 100 })],
      assignments: [asg('A', 'r1')],
    })
    const r = computeCosts(p, computeCpm(p.tasks, p.links))
    expect(r.rows.A.assigned).toBe(400)
  })
})

describe('normalizeProject 资源成本 schema 演进（刀1）', () => {
  it('旧文件无 resources/assignments：补空数组', () => {
    const n = normalizeProject(project())
    expect(n.resources).toEqual([])
    expect(n.assignments).toEqual([])
  })

  it('坏形条目逐条丢弃：坏类型/负费率/空 id/悬空字段保留合法条目', () => {
    const n = normalizeProject({
      ...project(),
      resources: [
        res('r1', 'work', { standardRate: 100 }),
        res('bad', 'weird' as SchedResource['type']),
        { id: '', name: '无名', type: 'work' },
        res('neg', 'work', { standardRate: -5 }),
      ] as SchedResource[],
      assignments: [
        asg('A', 'r1'),
        { taskId: '', resourceId: 'r1' },
        asg('A', 'r1', { units: -1 }),
        asg('A', 'r1', { amount: Number.POSITIVE_INFINITY }),
      ],
      tasks: [t('A', 3, { fixedCost: -2 }), t('B', 2)],
    })
    expect(n.resources!.map((x) => x.id)).toEqual(['r1'])
    expect(n.assignments!).toEqual([asg('A', 'r1')])
    expect(n.tasks[0].fixedCost).toBeUndefined()
  })

  it('fixedCost/资源数值字段合法值透传', () => {
    const n = normalizeProject({
      ...project(),
      tasks: [t('A', 3, { fixedCost: 88.8 })],
      resources: [res('r1', 'material', { standardRate: 0, maxUnits: 3, unit: 't' })],
      assignments: [asg('A', 'r1', { quantity: 0, units: 2 })],
    })
    expect(n.tasks[0].fixedCost).toBe(88.8)
    expect(n.resources![0]).toMatchObject({ standardRate: 0, maxUnits: 3, unit: 't' })
    expect(n.assignments![0]).toMatchObject({ quantity: 0, units: 2 })
  })
})
