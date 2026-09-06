/**
 * cost.ts — 成本 rollup 引擎（纯函数，v4.122.0 资源成本刀1）
 *
 * 设计口径（docs/gaea-schedule-resource-cost-design-2026-09.md §3.2，TS↔Go
 * 与 internal/schedule/cost.go 互为镜像，测试同批场景同批期望值）：
 *  - work：effDur(工作日) × units(默认1) × standardRate + costPerUse
 *  - material：quantity × standardRate + costPerUse
 *  - cost：amount
 *  - 任务成本 = fixedCost(默认0) + Σ 分配成本；rows 只含叶任务
 *    （分组行汇总=子孙叶求和，视图层滚动口径，不经独立存储）；
 *  - CPM 未过（循环依赖）→ ok=false，成本不出（fail-closed，成本服从同一裁决）；
 *  - 悬空引用的分配跳过（容错读由 normalize 承担、落盘拒绝由 Go Validate 承担，
 *    引擎只对在册数据计算）；
 *  - 金额 rollup 后四舍五入到分（0.01）。输入经校验非负（元无负值口径）。
 */
import type { CpmResult, SchedAssignment, SchedProject, SchedResource } from './types'
import { effDur } from './cpm'

/** 四舍五入到分；输入为校验后的非负金额（正值下 Math.round 与 math.Round 同律） */
function round2(n: number): number {
  return Math.round(n * 100) / 100
}

/** 单条分配的原始成本（元，未舍入）。资源缺失/类型未知返回 null（跳过） */
function assignmentCost(a: SchedAssignment, res: SchedResource | undefined, dur: number): number | null {
  if (!res) return null
  const units = a.units ?? 1
  const rate = res.standardRate ?? 0
  const perUse = res.costPerUse ?? 0
  switch (res.type) {
    case 'work': return dur * units * rate + perUse
    case 'material': return (a.quantity ?? 0) * rate + perUse
    case 'cost': return a.amount ?? 0
    default: return null
  }
}

/** 单任务成本行 */
export interface TaskCost {
  /** 固定成本（元） */
  fixed: number
  /** 分配成本合计（元） */
  assigned: number
  /** 明细合计 = fixed + assigned（元） */
  total: number
}

export interface CostResult {
  /** CPM 未过（循环依赖）时 ok=false，成本不出（fail-closed） */
  ok: boolean
  error?: string
  /** 叶任务明细（分组行不入 rows，汇总由视图层按子孙求和） */
  rows: Record<string, TaskCost>
  /** 资源维度汇总（元） */
  byResource: Record<string, number>
  /** 项目总成本 = Σ 叶任务 total（元） */
  total: number
}

/**
 * 成本 rollup 主计算。cpm 为 computeCpm 结果（成本服从同一裁决）；
 * 悬空引用的分配跳过；全无资源/成本数据的计划 → ok 且 total=0。
 */
export function computeCosts(project: SchedProject, cpm: CpmResult): CostResult {
  if (!cpm.ok) {
    return { ok: false, error: cpm.error, rows: {}, byResource: {}, total: 0 }
  }
  const resources = new Map((project.resources ?? []).map((r) => [r.id, r]))
  const tasksById = new Map(project.tasks.map((t) => [t.id, t]))
  const assignedRaw = new Map<string, number>()
  const byResourceRaw = new Map<string, number>()
  for (const a of project.assignments ?? []) {
    const res = resources.get(a.resourceId)
    const task = tasksById.get(a.taskId)
    if (!res || !task || task.level === 0) continue // 悬空引用/分组行分配：跳过（闸在 Validate）
    const c = assignmentCost(a, res, effDur(task))
    if (c === null) continue
    assignedRaw.set(a.taskId, (assignedRaw.get(a.taskId) ?? 0) + c)
    byResourceRaw.set(a.resourceId, (byResourceRaw.get(a.resourceId) ?? 0) + c)
  }

  const rows: Record<string, TaskCost> = {}
  let totalRaw = 0
  for (const t of project.tasks) {
    if (t.level === 0) continue
    const fixed = t.fixedCost ?? 0
    const assigned = assignedRaw.get(t.id) ?? 0
    rows[t.id] = { fixed: round2(fixed), assigned: round2(assigned), total: round2(fixed + assigned) }
    totalRaw += fixed + assigned
  }
  const byResource: Record<string, number> = {}
  for (const [rid, raw] of byResourceRaw) byResource[rid] = round2(raw)
  return { ok: true, rows, byResource, total: round2(totalRaw) }
}
