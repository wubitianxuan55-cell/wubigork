import { describe, expect, it } from 'vitest'
import { diffProjects } from './applyDiff'
import type { SchedProject } from './types'

// applyDiff.test.ts — diffProjects 纯比较全分支（diff 确认闭环刀B）。
// 两侧都经 normalizeProject 归一后再比，避免缺省字段差异冒充改动。

function proj(patch: Partial<SchedProject>): SchedProject {
  return {
    name: '测试工程', startDate: '2026-09-07',
    tasks: [
      { id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 },
      { id: 'B', name: '垫层', duration: 3, level: 1, progress: 0 },
    ],
    links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }],
    ...patch,
  }
}

describe('diffProjects 任务行（id 键控 + 字段级 from→to）', () => {
  it('新增/删除/修改（字段级展开）', () => {
    const d = diffProjects(
      proj({}),
      proj({
        tasks: [
          { id: 'A', name: '挖土', duration: 4, level: 1, progress: 50 },
          { id: 'C', name: '基础', duration: 2, level: 1, progress: 0 },
        ],
      }),
    )
    expect(d.tasks.added.map((t) => t.name)).toEqual(['基础'])
    expect(d.tasks.removed.map((t) => t.name)).toEqual(['垫层'])
    expect(d.tasks.modified).toHaveLength(1)
    expect(d.tasks.modified[0].id).toBe('A')
    const f = Object.fromEntries(d.tasks.modified[0].fields.map((x) => [x.label, `${x.from}→${x.to}`]))
    expect(f['工期（工作日）']).toBe('2→4')
    expect(f['进度（%）']).toBe('0→50')
    expect(f['任务名']).toBeUndefined() // 名字没改不出现
  })

  it('里程碑/模式/手动开始/固定成本字段变化均展开', () => {
    const d = diffProjects(
      proj({ tasks: [{ id: 'A', name: 'x', duration: 1, level: 1, progress: 0 }] }),
      proj({ tasks: [{ id: 'A', name: 'x', duration: 1, level: 1, progress: 0, isMilestone: true, mode: 'manual', manualStart: 3, fixedCost: 100 }] }),
    )
    const labels = d.tasks.modified[0].fields.map((f) => f.label)
    expect(labels).toEqual(expect.arrayContaining(['里程碑', '排程模式', '手动开始（工作日序号）', '固定成本（元）']))
  })
})

describe('diffProjects 搭接（按 to 分组入边集合比对）', () => {
  it('类型/时距变化=changed；新增与删除各归其类', () => {
    const d = diffProjects(
      proj({ links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }, { from: 'A', to: 'A2', type: 'FS', lag: 0 } as never] }),
      proj({ links: [{ from: 'A', to: 'B', type: 'SS', lag: 1 }, { from: 'B', to: 'A2', type: 'FS', lag: 0 } as never] }),
    )
    // A→B 键相同但 type/lag 变了 → changed
    expect(d.links.changed).toHaveLength(1)
    expect(d.links.changed[0].taskName).toBe('垫层')
    expect(d.links.changed[0].from).toContain('FS')
    expect(d.links.changed[0].to).toContain('SS+1')
  })

  it('纯新增/纯删除入边', () => {
    const d = diffProjects(proj({}), proj({ links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }, { from: 'B', to: 'A', type: 'FF', lag: 0 }] }))
    expect(d.links.added).toHaveLength(1)
    expect(d.links.added[0].taskName).toBe('挖土')
    expect(d.links.removed).toHaveLength(0)
    const d2 = diffProjects(proj({}), proj({ links: [] }))
    expect(d2.links.removed).toHaveLength(1)
  })
})

describe('diffProjects 资源与分配', () => {
  it('资源删除标注级联分配数；新增/修改各归其类', () => {
    const before = proj({
      resources: [
        { id: 'r1', name: '挖机', type: 'work', standardRate: 800 },
        { id: 'r2', name: '工人', type: 'work', standardRate: 300 },
      ],
      assignments: [{ taskId: 'A', resourceId: 'r1' }, { taskId: 'B', resourceId: 'r1' }],
    })
    const after = proj({
      resources: [{ id: 'r2', name: '工人', type: 'work', standardRate: 350 }, { id: 'r3', name: '吊车', type: 'cost' }],
    })
    const d = diffProjects(before, after)
    expect(d.resources.removed).toEqual([{ name: '挖机', cascade: 2 }])
    expect(d.resources.added.map((r) => r.name)).toEqual(['吊车'])
    expect(d.resources.modified[0].fields[0]).toEqual({ label: '标准费率（元）', from: '300', to: '350' })
  })

  it('分配增删改（units/quantity/amount 字段级）', () => {
    const before = proj({ resources: [{ id: 'r1', name: '挖机', type: 'work', standardRate: 800 }], assignments: [{ taskId: 'A', resourceId: 'r1', units: 1 }] })
    const after = proj({ resources: [{ id: 'r1', name: '挖机', type: 'work', standardRate: 800 }], assignments: [{ taskId: 'A', resourceId: 'r1', units: 2 }, { taskId: 'B', resourceId: 'r1', amount: 500 }] })
    const d = diffProjects(before, after)
    expect(d.assignments.added).toHaveLength(2)
    const mod = d.assignments.added.find((a) => a.taskName === '挖土')!
    expect(mod.fields).toEqual([{ label: '投入强度', from: '1', to: '2' }])
    expect(d.assignments.added.find((a) => a.taskName === '垫层')!.fields).toBeUndefined()
  })
})

describe('diffProjects meta 与汇总', () => {
  it('meta 只列变化项（名/开工/日历/竣工）', () => {
    const d = diffProjects(proj({}), proj({ name: '新名', startDate: '2026-09-08', deadline: '2026-09-30' }))
    const labels = d.meta.map((m) => m.label)
    expect(labels).toEqual(['工程名', '开工日期', '目标竣工'])
  })

  it('汇总行：总工期/关键数/总成本预览（双方 CPM）', () => {
    const d = diffProjects(
      proj({ tasks: [{ id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 }, { id: 'B', name: '垫层', duration: 3, level: 1, progress: 0 }] }),
      proj({ tasks: [{ id: 'A', name: '挖土', duration: 4, level: 1, progress: 0, fixedCost: 100 }, { id: 'B', name: '垫层', duration: 3, level: 1, progress: 0 }] }),
    )
    expect(d.summary.durationFrom).toBe(5)
    expect(d.summary.durationTo).toBe(7)
    expect(d.summary.criticalFrom).toBe(2)
    expect(d.summary.criticalTo).toBe(2)
    expect(d.summary.costFrom).toBe(0)
    expect(d.summary.costTo).toBe(100)
  })

  it('after CPM 循环 → 预览层 fail-closed 标注引擎将拒绝', () => {
    const d = diffProjects(
      proj({}),
      proj({ links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }, { from: 'B', to: 'A', type: 'FS', lag: 0 }] }),
    )
    expect(d.summary.cpmError).toBeTruthy()
    expect(d.summary.durationTo).toBeUndefined()
  })

  it('两侧一致 → empty=true', () => {
    const d = diffProjects(proj({}), proj({}))
    expect(d.empty).toBe(true)
    expect(d.summary.durationFrom).toBe(d.summary.durationTo)
  })

  it('倒排校核预览：after 有 deadline 且超期时标注', () => {
    const d = diffProjects(proj({}), proj({ deadline: '2026-09-08' }))
    // 总工期 5 天 > 目标 2 工作日 → 不可行
    expect(d.summary.deadlineOk).toBe(false)
    expect(d.summary.deadlineOver).toBeGreaterThan(0)
  })
})
