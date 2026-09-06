/**
 * deadline.test.ts — 倒排校核纯函数用例（v4.117.0 刀8）
 *
 * 与 internal/schedule/deadline_test.go 互为镜像：同一批场景同一批期望值。
 */
import { describe, expect, it } from 'vitest'
import { checkDeadline } from './deadline'
import { computeCpm } from './cpm'
import type { SchedProject } from './types'

/** 三任务串联样板：A(3) → B(2) → C(4)，总工期 9，全关键（与基线测试同款） */
function chainProject(): SchedProject {
  return {
    name: '链式样板',
    startDate: '2026-09-07',
    tasks: [
      { id: 'A', name: '挖土', duration: 3, level: 1, progress: 0 },
      { id: 'B', name: '垫层', duration: 2, level: 1, progress: 0 },
      { id: 'C', name: '浇筑', duration: 4, level: 1, progress: 0 },
    ],
    links: [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'B', to: 'C', type: 'FS', lag: 0 },
    ],
  }
}

describe('checkDeadline', () => {
  it('未设目标竣工返回 null', () => {
    const p = chainProject()
    expect(checkDeadline(p, computeCpm(p.tasks, p.links))).toBeNull()
  })

  it('可行且富余：目标竣工换算工作日、overrun 为负', () => {
    const p = chainProject()
    p.deadline = '2026-09-21' // 周一开工起第 11 个工作日（09-07~09-21 含两个周末）
    const d = checkDeadline(p, computeCpm(p.tasks, p.links))!
    expect(d.targetWorkdays).toBe(11)
    expect(d.currentDuration).toBe(9)
    expect(d.feasible).toBe(true)
    expect(d.overrun).toBe(-2)
    expect(d.criticalTasks).toEqual(['挖土', '垫层', '浇筑'])
  })

  it('压线达成：duration == target → feasible、overrun 0', () => {
    const p = chainProject()
    p.deadline = '2026-09-17' // 周四=第 9 个工作日
    const d = checkDeadline(p, computeCpm(p.tasks, p.links))!
    expect(d.targetWorkdays).toBe(9)
    expect(d.feasible).toBe(true)
    expect(d.overrun).toBe(0)
  })

  it('不可达：overrun 为正、criticalTasks 列出压缩对象', () => {
    const p = chainProject()
    p.deadline = '2026-09-15' // 第 7 个工作日
    const d = checkDeadline(p, computeCpm(p.tasks, p.links))!
    expect(d.targetWorkdays).toBe(7)
    expect(d.feasible).toBe(false)
    expect(d.overrun).toBe(2)
    expect(d.criticalTasks).toEqual(['挖土', '垫层', '浇筑'])
  })

  it('手动任务不进 criticalTasks 且不约束前置', () => {
    const p = chainProject()
    p.tasks = p.tasks.map((t) => (t.id === 'B' ? { ...t, mode: 'manual' as const, manualStart: 0 } : t))
    p.deadline = '2026-09-15'
    const d = checkDeadline(p, computeCpm(p.tasks, p.links))!
    // B 手动锁定 0 起：忽略入边不回传约束 → A 也有时差，关键仅剩 C
    expect(d.criticalTasks).toEqual(['浇筑'])
  })

  it('目标竣工早于开工：target=0 必然不可达', () => {
    const p = chainProject()
    p.deadline = '2026-09-06'
    const d = checkDeadline(p, computeCpm(p.tasks, p.links))!
    expect(d.targetWorkdays).toBe(0)
    expect(d.feasible).toBe(false)
    expect(d.overrun).toBe(9)
  })
})
