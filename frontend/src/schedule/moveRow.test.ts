import { describe, expect, it } from 'vitest'
import { planRowMove } from './moveRow'
import type { SchedTask } from './types'

const t = (id: string, level: number, name = id): SchedTask => ({ id, name, duration: 1, level, progress: 0 })

const tasks: SchedTask[] = [
  t('G1', 0, '分区一'),
  t('A', 1),
  t('B', 1),
  t('G2', 0, '分区二'),
  t('C', 1),
]

const ids = (r: { tasks?: SchedTask[] }) => (r.tasks ?? []).map((x) => `${x.id}:${x.level}`)

describe('planRowMove 树形拖动计划器', () => {
  it('工作拖到另一工作之前：同级插入', () => {
    const r = planRowMove(tasks, 'B', 'A', 'before')
    expect(r.ok).toBe(true)
    expect(ids(r)).toEqual(['G1:0', 'B:1', 'A:1', 'G2:0', 'C:1'])
  })

  it('工作拖到另一工作之后：同级插入', () => {
    const r = planRowMove(tasks, 'A', 'B', 'after')
    expect(ids(r)).toEqual(['G1:0', 'B:1', 'A:1', 'G2:0', 'C:1'])
  })

  it('工作拖进分组中段：成为该分区的子工作（插到现有子工作末尾）', () => {
    const r = planRowMove(tasks, 'C', 'G1', 'inside')
    expect(r.ok).toBe(true)
    expect(ids(r)).toEqual(['G1:0', 'A:1', 'B:1', 'C:1', 'G2:0'])
  })

  it('分区拖动：整棵子树一起搬，落到目标分区的子工作之后（不分家）', () => {
    const r = planRowMove(tasks, 'G1', 'G2', 'after')
    expect(r.ok).toBe(true)
    expect(ids(r)).toEqual(['G2:0', 'C:1', 'G1:0', 'A:1', 'B:1'])
  })

  it('拒绝：拖进自己的子树', () => {
    expect(planRowMove(tasks, 'G1', 'A', 'after').reason).toContain('自己的子工作')
  })

  it('拒绝：分区塞进分区；工作塞进工作', () => {
    expect(planRowMove(tasks, 'G1', 'G2', 'inside').reason).toContain('分区不能塞进分区')
    expect(planRowMove(tasks, 'A', 'B', 'inside').reason).toContain('只有分区行可以接收')
  })

  it('拒绝：跨级同级约束（分区对工作、工作对分区）', () => {
    expect(planRowMove(tasks, 'G1', 'C', 'before').reason).toContain('分区只能与分区')
    expect(planRowMove(tasks, 'A', 'G2', 'after').reason).toContain('工作只能与工作')
  })

  it('拒绝：原地未移动 / 不存在的行', () => {
    expect(planRowMove(tasks, 'A', 'A', 'after').reason).toContain('原地')
    expect(planRowMove(tasks, 'NOPE', 'A', 'after').reason).toContain('不存在')
  })

  it('纯函数：入参数组不被修改', () => {
    const snapshot = tasks.map((x) => ({ ...x }))
    planRowMove(tasks, 'B', 'A', 'before')
    expect(tasks).toEqual(snapshot)
  })
})
