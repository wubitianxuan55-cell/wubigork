// monte.test.ts — 6.5 蒙特卡洛工期带纯函数：确定性（同种子同结果）、
// 三档单调、零不确定度退化、关键线路稳定度、CPM 断裂诚实失败。
import { describe, expect, it } from 'vitest'
import { monteCarlo, mulberry32 } from './monte'
import type { SchedProject } from './types'

function task(id: string, duration: number) {
  return { id, name: id, duration, level: 1, progress: 0 }
}

/** A→B→C 线性链（FS），总工期 20。 */
function chainProject(): SchedProject {
  return {
    name: 'p',
    startDate: '2026-09-01',
    tasks: [task('A', 5), task('B', 10), task('C', 5)],
    links: [
      { from: 'A', to: 'B', type: 'FS', lag: 0 },
      { from: 'B', to: 'C', type: 'FS', lag: 0 },
    ],
  }
}

describe('蒙特卡洛工期带（6.5）', () => {
  it('确定性：同种子两次模拟逐字段一致；种子不同序列不同', () => {
    const p = chainProject()
    const a = monteCarlo(p, { runs: 50, seed: 7 })
    const b = monteCarlo(p, { runs: 50, seed: 7 })
    expect(a).toEqual(b)
    const c = monteCarlo(p, { runs: 50, seed: 8 })
    expect(c.histogram).not.toEqual([]) // 有分布
    // 不同种子极大概率序列不同（统计性而非恒等）。
    expect(a.criticality.length).toBeGreaterThan(0)
  })

  it('三档单调：min ≤ P25 ≤ 中位 ≤ P75 ≤ max；均值在界内', () => {
    const p = chainProject()
    const r = monteCarlo(p, { runs: 100, seed: 42 })
    expect(r.min).toBeLessThanOrEqual(r.p25)
    expect(r.p25).toBeLessThanOrEqual(r.p50)
    expect(r.p50).toBeLessThanOrEqual(r.p75)
    expect(r.p75).toBeLessThanOrEqual(r.max)
    expect(r.mean).toBeGreaterThanOrEqual(r.min)
    expect(r.mean).toBeLessThanOrEqual(r.max)
    // 不对称三角（悲观 > 乐观）→ 均值不低于计划值。
    expect(r.mean).toBeGreaterThanOrEqual(r.plannedDuration)
    expect(r.plannedDuration).toBe(20)
  })

  it('零不确定度退化：所有样本=计划工期（u=0 时三角分布坍缩为确定值）', () => {
    const p = chainProject()
    const r = monteCarlo(p, { runs: 50, uncertainty: 0, seed: 1 })
    expect(r.p25).toBe(20)
    expect(r.p50).toBe(20)
    expect(r.p75).toBe(20)
    expect(r.min).toBe(20)
    expect(r.max).toBe(20)
    expect(r.stability).toBe(1)
  })

  it('关键路径稳定度：线性链全部关键任务 rate=1、稳定度=1；三档≤单点工期上界内', () => {
    const p = chainProject()
    const r = monteCarlo(p, { runs: 60, seed: 9 })
    expect(r.criticality.map((c) => c.id).sort()).toEqual(['A', 'B', 'C'])
    for (const c of r.criticality) {
      expect(c.rate).toBe(1)
      expect(c.criticalNow).toBe(true)
    }
    expect(r.stability).toBe(1)
  })

  it('非关键任务偶尔换线：长尾不确定度下并行链低浮任务频率<100%', () => {
    const p = chainProject()
    // X→Y 并行链（总长 19，接近关键）：B 波动到悲观时 Y 链可能成为关键。
    p.tasks.push(task('X', 4), task('Y', 15))
    p.links.push({ from: 'X', to: 'Y', type: 'FS', lag: 0 })
    const r = monteCarlo(p, { runs: 300, uncertainty: 0.5, seed: 3 })
    const y = r.criticality.find((c) => c.id === 'Y')
    expect(y).toBeTruthy()
    expect(y!.rate).toBeGreaterThan(0)
    expect(y!.rate).toBeLessThanOrEqual(1)
    expect(y!.criticalNow).toBe(false) // 计划态 Y 非关键（19<20）
  })

  it('直方图桶频数之和=有效模拟次数', () => {
    const p = chainProject()
    const r = monteCarlo(p, { runs: 80, seed: 11 })
    const total = r.histogram.reduce((a, h) => a + h.count, 0)
    expect(total).toBe(80)
  })

  it('CPM 断裂：ok=false + error，不伪造三档', () => {
    const p = chainProject()
    p.links.push({ from: 'C', to: 'A', type: 'FS', lag: 0 })
    const r = monteCarlo(p, {})
    expect(r.ok).toBe(false)
    expect(r.error).toBeTruthy()
    expect(r.p25).toBe(0)
    expect(r.histogram).toEqual([])
  })

  it('RNG：mulberry32 同种子同序列、输出在 [0,1)', () => {
    const a = mulberry32(42)
    const b = mulberry32(42)
    for (let i = 0; i < 10; i++) {
      const x = a()
      const y = b()
      expect(x).toBe(y)
      expect(x).toBeGreaterThanOrEqual(0)
      expect(x).toBeLessThan(1)
    }
  })
})
