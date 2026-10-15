// whatif.ts — 关键线路 what-if 推演（阶段十一 11.4）
//
// 「任务 X 延长/压缩 N 天，总工期与关键线路怎么变」——纯前端本地重算
//（computeCpm 纯函数，零服务零 LLM，出口判据「推演纯本地」）。
// 不写盘不改原计划：推演结果是只读对比快照。
import type { SchedProject } from './types'
import { computeCpm } from './cpm'
import type { CpmResult, TaskCpm } from './types'

export interface WhatIfResult {
  /** 推演可行（任务存在且重算成功）；false 时 reason 说明原因 */
  ok: boolean
  reason?: string
  /** 基线总工期 / 推演总工期 / 变化天数（正=延长） */
  baseDuration: number
  whatIfDuration: number
  delta: number
  /** 基线关键线路（任务名序列） / 推演关键线路 */
  baseCritical: string[]
  whatIfCritical: string[]
  /** 因推演新进入关键线路的任务名 */
  newlyCritical: string[]
  /** 推演后脱离关键线路的任务名 */
  noLongerCritical: string[]
}

const criticalNames = (p: SchedProject, r: CpmResult): string[] => {
  const byId = new Map(p.tasks.map((t) => [t.id, t]))
  const entries = Object.entries(r.rows) as [string, TaskCpm][]
  return entries
    .filter(([, row]) => row.critical)
    .sort((a, b) => a[1].es - b[1].es || a[1].ef - b[1].ef)
    .map(([id]) => byId.get(id)?.name ?? id)
}

/**
 * 任务工期 what-if 推演：把 taskId 的工期加 delta（可负，下限 0；里程碑恒 0）
 * 后重算 CPM，与基线对比总工期与关键线路。原计划不被修改。
 */
export function whatIfDuration(p: SchedProject, taskId: string, delta: number): WhatIfResult {
  const base = computeCpm(p.tasks, p.links)
  const task = p.tasks.find((t) => t.id === taskId)
  if (!task) {
    return {
      ok: false, reason: `任务 ${taskId} 不存在`, baseDuration: base.duration, whatIfDuration: base.duration,
      delta: 0, baseCritical: criticalNames(p, base), whatIfCritical: criticalNames(p, base),
      newlyCritical: [], noLongerCritical: [],
    }
  }
  if (task.isMilestone) {
    return {
      ok: false, reason: '里程碑工期恒为 0，无法推演工期变化', baseDuration: base.duration, whatIfDuration: base.duration,
      delta: 0, baseCritical: criticalNames(p, base), whatIfCritical: criticalNames(p, base),
      newlyCritical: [], noLongerCritical: [],
    }
  }
  if (base.duration <= 0) {
    return {
      ok: false, reason: '基线计划无可推演的工期（全部任务工期为 0 或无叶任务）', baseDuration: base.duration, whatIfDuration: base.duration,
      delta: 0, baseCritical: criticalNames(p, base), whatIfCritical: criticalNames(p, base),
      newlyCritical: [], noLongerCritical: [],
    }
  }

  const clamped = task.duration + delta < 0 ? -task.duration : delta
  const modified = p.tasks.map((t) => (t.id === taskId ? { ...t, duration: t.duration + clamped } : t))
  const sim = computeCpm(modified, p.links)

  const baseCrit = criticalNames(p, base)
  const simCrit = criticalNames(p, sim)
  const baseSet = new Set(baseCrit)
  const simSet = new Set(simCrit)

  return {
    ok: true,
    baseDuration: base.duration,
    whatIfDuration: sim.duration,
    delta: sim.duration - base.duration,
    baseCritical: baseCrit,
    whatIfCritical: simCrit,
    newlyCritical: simCrit.filter((n) => !baseSet.has(n)),
    noLongerCritical: baseCrit.filter((n) => !simSet.has(n)),
  }
}
