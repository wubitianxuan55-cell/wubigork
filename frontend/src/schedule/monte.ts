// monte.ts — 6.5 蒙特卡洛工期带：本地模拟 → 工期 P25/中位/P75 三档 +
// 关键路径稳定度。与造价价格带同一哲学：「工期也给你三档」。
// Why：单点 CPM 只给一个确定工期；工期风险（哪档更可能、关键线路会不会
// 换线）没有量化。本地 N 次模拟（每次按不确定性采样任务工期后重算 CPM）
// 直接给出工期分布与关键线路稳定度——全本地零云依赖（6.5 出口判据）。
// How：纯函数无副作用——种子化 RNG（同种子同结果，确定性可测）；采样
// 分布=三角分布 [乐观 d·(1−u/2), 最可能 d, 悲观 d·(1+u)]（默认 u=0.3，
// 工期偏乐观的不对称形态）；每次模拟对采样后的计划重算 CPM（复用
// computeCpm），记录总工期与关键任务集合。

import type { SchedProject, SchedTask } from './types'
import { computeCpm } from './cpm'
import { planFinishOf } from './deadline'

/** 默认模拟次数（本地毫秒级；小计划可上调） */
export const MONTE_DEFAULT_RUNS = 200
/** 默认不确定度：悲观 = 计划工期 × 1.3（工期普遍偏乐观的不对称先验） */
export const MONTE_DEFAULT_UNCERTAINTY = 0.3

/** 种子化 RNG（mulberry32）：同种子产生同序列，模拟结果可复现可测试。 */
export function mulberry32(seed: number): () => number {
  let a = seed >>> 0
  return function () {
    a |= 0
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

/** 三角分布采样：u=0 时退化为确定值 d。 */
function triangular(d: number, uncertainty: number, rand: () => number): number {
  const a = d * (1 - uncertainty / 2)
  const c = d
  const b = d * (1 + uncertainty)
  if (b <= a) return d
  const fc = (c - a) / (b - a)
  const u = rand()
  if (u < fc) {
    return a + Math.sqrt(u * (b - a) * (c - a))
  }
  return b - Math.sqrt((1 - u) * (b - a) * (b - c))
}

export interface MonteTaskRate {
  id: string
  name: string
  /** 该任务在模拟中处于关键线路的频率（0-1；当前计划关键任务若长期<100% = 可能换线） */
  rate: number
  /** 当前计划是否关键 */
  criticalNow: boolean
}

export interface MonteResult {
  ok: boolean
  error?: string
  runs: number
  /** 工期三档（工作日，取整） */
  p25: number
  p50: number
  p75: number
  min: number
  max: number
  mean: number
  /** 单点 CPM 的计划工期（对照基准） */
  plannedDuration: number
  /** 关键路径稳定度：按出现频率降序（含频率<100% 的换线候选） */
  criticality: MonteTaskRate[]
  /** 稳定度指标：当前关键任务的平均关键频率（0-1，越接近 1 越稳） */
  stability: number
  /** 分布直方图桶（供纯展示）：[上界(不含), 频数]，覆盖 min..max */
  histogram: { upTo: number; count: number }[]
}

function percentile(sorted: number[], q: number): number {
  if (sorted.length === 0) return 0
  const idx = Math.min(sorted.length - 1, Math.ceil(q * sorted.length) - 1)
  return sorted[Math.max(0, idx)]
}

export interface MonteOptions {
  /** 模拟次数（缺省 200，下限 20） */
  runs?: number
  /** 不确定度（悲观 = 工期×(1+u)，缺省 0.3） */
  uncertainty?: number
  /** 随机种子（缺省固定 42——同输入同结果，跨会话可比） */
  seed?: number
}

export function monteCarlo(project: SchedProject, opts?: MonteOptions): MonteResult {
  const runs = Math.max(20, opts?.runs ?? MONTE_DEFAULT_RUNS)
  const u = opts?.uncertainty ?? MONTE_DEFAULT_UNCERTAINTY
  const rand = mulberry32(opts?.seed ?? 42)

  const planned = computeCpm(project.tasks, project.links, {
    planFinish: undefined,
    calendar: project.calendar,
    startDate: project.startDate,
  })
  const fail: MonteResult = {
    ok: false,
    error: planned.error ?? 'CPM 未通过（循环依赖）',
    runs,
    p25: 0, p50: 0, p75: 0, min: 0, max: 0, mean: 0,
    plannedDuration: planned.ok ? planned.duration : 0,
    criticality: [],
    stability: 0,
    histogram: [],
  }
  if (!planned.ok) return fail

  // 可采样任务：叶任务（level>0）非里程碑且工期>0。
  const sampleable: { t: SchedTask; d: number }[] = []
  for (const t of project.tasks) {
    if (t.level <= 0 || t.isMilestone) continue
    if (t.duration > 0) sampleable.push({ t, d: t.duration })
  }

  const durations: number[] = []
  const critCount = new Map<string, number>()
  const planFinish = planFinishOf(project)
  for (let i = 0; i < runs; i++) {
    const sampledDur = new Map<string, number>()
    for (const { t, d } of sampleable) {
      sampledDur.set(t.id, Math.max(0, Math.round(triangular(d, u, rand))))
    }
    const sampled: SchedTask[] = project.tasks.map((t) => {
      const d = sampledDur.get(t.id)
      return d === undefined ? t : { ...t, duration: d }
    })
    const sim = computeCpm(sampled, project.links, {
      planFinish,
      calendar: project.calendar,
      startDate: project.startDate,
    })
    if (!sim.ok) continue
    durations.push(sim.duration)
    for (const id in sim.rows) {
      if (sim.rows[id].critical) critCount.set(id, (critCount.get(id) ?? 0) + 1)
    }
  }

  durations.sort((a, b) => a - b)
  const p25 = percentile(durations, 0.25)
  const p50 = percentile(durations, 0.5)
  const p75 = percentile(durations, 0.75)
  const min = durations[0] ?? 0
  const max = durations[durations.length - 1] ?? 0
  const mean = durations.length
    ? Math.round((durations.reduce((a, b) => a + b, 0) / durations.length) * 10) / 10
    : 0

  // 关键路径稳定度：所有曾关键的任务按频率降序；当前关键任务平均频率=稳定度。
  const criticality: MonteTaskRate[] = []
  for (const [id, n] of critCount) {
    const t = project.tasks.find((x) => x.id === id)
    if (!t) continue
    criticality.push({
      id,
      name: t.name,
      rate: Math.round((n / runs) * 1000) / 1000,
      criticalNow: planned.rows[id]?.critical ?? false,
    })
  }
  criticality.sort((a, b) => b.rate - a.rate || a.name.localeCompare(b.name))
  const nowCrit = criticality.filter((c) => c.criticalNow)
  const stability =
    nowCrit.length === 0
      ? 0
      : Math.round((nowCrit.reduce((a, c) => a + c.rate, 0) / nowCrit.length) * 1000) / 1000

  // 直方图：10 桶覆盖 [min, max]。
  const histogram: { upTo: number; count: number }[] = []
  if (durations.length > 0 && max > min) {
    const step = (max - min) / 10
    for (let i = 0; i < 10; i++) histogram.push({ upTo: Math.ceil(min + step * (i + 1)), count: 0 })
    for (const d of durations) {
      const idx = Math.min(9, Math.floor((d - min) / step))
      histogram[idx].count++
    }
  } else if (durations.length > 0) {
    histogram.push({ upTo: max, count: durations.length })
  }

  return {
    ok: true,
    runs,
    p25, p50, p75, min, max, mean,
    plannedDuration: planned.duration,
    criticality,
    stability,
    histogram,
  }
}
