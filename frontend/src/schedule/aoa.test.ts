import { describe, expect, it } from 'vitest'
import { buildAoa } from './aoa'
import type { SchedLink, SchedTask } from './types'

function t(id: string, duration: number, extra?: Partial<SchedTask>): SchedTask {
  return { id, name: id, duration, level: 1, progress: 0, ...extra }
}
function l(from: string, to: string, type: SchedLink['type'] = 'FS', lag = 0): SchedLink {
  return { from, to, type, lag }
}

describe('buildAoa 教材画法', () => {
  it('顺序链 A(3)→B(2)→C(4)：FS 合并成一条链，仅终点汇出一条虚工作', () => {
    const g = buildAoa([t('A', 3), t('B', 2), t('C', 4)], [l('A', 'B'), l('B', 'C')])
    expect(g.ok).toBe(true)
    // 5 个事件：起点、A 完成(=B 开始)、B 完成(=C 开始)、C 完成、终点
    expect(g.nodes).toHaveLength(5)
    // 4 条箭线：3 条实工作 + 1 条汇出虚工作
    expect(g.edges).toHaveLength(4)
    const tasks = g.edges.filter((e) => e.kind === 'task')
    expect(tasks.map((e) => e.dur)).toEqual([3, 2, 4])
    expect(g.edges.filter((e) => e.kind === 'dummy')).toHaveLength(1)
    // 全部关键
    expect(g.edges.every((e) => e.critical)).toBe(true)
    // 事件最早时间
    const byNum = new Map(g.nodes.map((n) => [n.num, n]))
    expect(byNum.get(1)!.es).toBe(0)
    expect(byNum.get(5)!.es).toBe(9)
  })

  it('经典虚工作案例 A→B/C→D：B、C 汇入 D 需两条虚工作', () => {
    const g = buildAoa(
      [t('A', 2), t('B', 6), t('C', 1), t('D', 5)],
      [l('A', 'B'), l('A', 'C'), l('B', 'D'), l('C', 'D')],
    )
    expect(g.ok).toBe(true)
    expect(g.nodes).toHaveLength(7)
    expect(g.edges.filter((e) => e.kind === 'task')).toHaveLength(4)
    const dummies = g.edges.filter((e) => e.kind === 'dummy')
    expect(dummies).toHaveLength(3) // E_B→S_D、E_C→S_D、E_D→END
    // 关键线路 = A→B→D（经虚工作 E_B→S_D），C 支路有时差
    const critIds = g.edges.filter((e) => e.critical).map((e) => e.taskId ?? `dummy:${e.from}>${e.to}`)
    expect(critIds).toContain('A')
    expect(critIds).toContain('B')
    expect(critIds).toContain('D')
    expect(critIds).not.toContain('C')
  })

  it('并行起算：两任务共用起点事件（一始一终），无虚工作拆分', () => {
    const g = buildAoa([t('A', 3), t('B', 2)], [])
    expect(g.ok).toBe(true)
    const starts = g.edges.filter((e) => e.kind === 'task').map((e) => e.from)
    expect(new Set(starts).size).toBe(1) // 同一出发事件
    // 汇入同一终点事件（各自一条虚工作）
    const ends = g.edges.filter((e) => e.kind === 'dummy').map((e) => e.to)
    expect(new Set(ends).size).toBe(1)
  })

  it('SS(lag=0) 合并：B 与 A 共起点（开始事件复用）', () => {
    const g = buildAoa([t('A', 5), t('B', 2)], [l('A', 'B', 'SS', 0)])
    expect(g.ok).toBe(true)
    const edgeA = g.edges.find((e) => e.taskId === 'A')!
    const edgeB = g.edges.find((e) => e.taskId === 'B')!
    expect(edgeB.from).toBe(edgeA.from)
    expect(edgeB.from).not.toBe(edgeA.to) // 不是 FS 合并
  })

  it('SS+时距：虚工作从 A 开始事件指向 B 开始事件并携带时距', () => {
    const g = buildAoa([t('A', 5), t('B', 2)], [l('A', 'B', 'SS', 3)])
    expect(g.ok).toBe(true)
    const edgeB = g.edges.find((e) => e.taskId === 'B')!
    const dummy = g.edges.find((e) => e.kind === 'dummy' && e.to === edgeB.from)!
    expect(dummy).toBeDefined()
    expect(dummy.dur).toBe(3)
    expect(dummy.label).toContain('+3')
  })
})

describe('buildAoa 结构不变量', () => {
  const cases: { name: string; tasks: SchedTask[]; links: SchedLink[] }[] = [
    { name: '链式', tasks: [t('A', 3), t('B', 2), t('C', 4)], links: [l('A', 'B'), l('B', 'C')] },
    {
      name: '菱形+混合搭接',
      tasks: [t('A', 10), t('B', 6), t('C', 4), t('D', 3), t('E', 5)],
      links: [l('A', 'B'), l('A', 'C', 'SS', 2), l('B', 'D'), l('C', 'D', 'FF', 1), l('D', 'E'), l('B', 'E')],
    },
    { name: '多起多终', tasks: [t('A', 2), t('B', 5), t('C', 3)], links: [l('A', 'C')] },
  ]
  for (const c of cases) {
    it(`${c.name}：节点编号 i<j、每任务恰一条实箭线、无重复虚箭线`, () => {
      const g = buildAoa(c.tasks, c.links)
      expect(g.ok, g.error).toBe(true)
      const num = new Map(g.nodes.map((n) => [n.id, n.num]))
      for (const e of g.edges) {
        expect(num.get(e.from)!, `${e.from}>${e.to}`).toBeLessThan(num.get(e.to)!)
      }
      expect(g.edges.filter((e) => e.kind === 'task')).toHaveLength(c.tasks.length)
      expect(Object.keys(g.taskEdge)).toHaveLength(c.tasks.length)
      const dummyKeys = g.edges.filter((e) => e.kind === 'dummy').map((e) => `${e.from}>${e.to}`)
      expect(new Set(dummyKeys).size).toBe(dummyKeys.length)
    })
  }
})

describe('buildAoa 健壮性', () => {
  it('循环依赖拒绝', () => {
    const g = buildAoa([t('A', 1), t('B', 1)], [l('A', 'B'), l('B', 'A')])
    expect(g.ok).toBe(false)
    expect(g.error).toContain('循环依赖')
  })

  it('空项目返回空图', () => {
    const g = buildAoa([], [])
    expect(g.ok).toBe(true)
    expect(g.nodes).toHaveLength(0)
    expect(g.edges).toHaveLength(0)
  })

  it('里程碑（0 工期）参与转换不炸', () => {
    const g = buildAoa([t('A', 4), t('M', 0, { isMilestone: true })], [l('A', 'M')])
    expect(g.ok).toBe(true)
    const m = g.edges.find((e) => e.taskId === 'M')!
    expect(m.dur).toBe(0)
  })
})
