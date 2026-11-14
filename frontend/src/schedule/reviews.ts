/**
 * schedule/reviews.ts — 检视（多期进度快照）模型：累计完成、执行状态、期际对比、拉直
 *
 * 对标斑马「检视时间」（逆向报告 §5.1–5.3）：计划是连续推演的，执行是离散打点的。
 * 每期检视 = 一个数据日期（`at`，工作日序号）+ 该期逐任务实际回填（`actuals`）。
 *
 * 三条不可妥协的口径（都来自斑马的行为，不是我们的发明）：
 * 1. **历史不可覆盖**：多期并列，新增同一 `at` 时是"改那一期"而不是"并存两期"
 *    （`normalizeReviews` 已按 at 去重，后写覆盖）。
 * 2. **拉直要把实际写回计划**：斑马的"拉直前锋线"会把回填的完成百分比落到计划状态，
 *    然后**冻结**该期（`frozen`），禁止再回改已完成/进行中——否则前锋线可以反复揉。
 * 3. **完成量按工期加权**：逐任务百分比的算术平均没有意义（一个 1 天的任务和
 *    100 天的任务等价），累计完成必须按有效工期加权（里程碑不计分母）。
 *
 * 纯函数，无 store 依赖：面板只做展示与触发，算都在这里。
 */

import { dateToWd, isoOf, wdToDate } from './calendar'
import { MAX_REVIEWS } from './blocks'
import type { CpmResult, SchedProject, SchedReview, SchedTask } from './types'

/** 执行状态三态（斑马口径）：未开始 / 进行中 / 已完成 */
export type ExecStatus = '未开始' | '进行中' | '已完成'

/** 有效工期（里程碑=0；分组行不参与） */
function effDur(t: SchedTask): number {
  if (t.level === 0 || t.isMilestone) return 0
  return Math.max(0, Math.round(t.duration))
}

/** 叶任务列表（分组行不算工作） */
export function leavesOf(project: SchedProject): SchedTask[] {
  return (project.tasks ?? []).filter((t) => t.level > 0)
}

/** 某期的任务完成百分比：先看该期回填，缺省回落任务当前 progress */
export function reviewPercent(review: SchedReview | null | undefined, task: SchedTask): number {
  const a = review?.actuals?.[task.id]
  const v = a?.percent
  if (typeof v === 'number' && Number.isFinite(v)) return Math.min(100, Math.max(0, Math.round(v)))
  return Math.min(100, Math.max(0, Math.round(task.progress ?? 0)))
}

/**
 * 某期的执行状态：
 * - 回填了 finish 或百分比 100 → 已完成；
 * - 回填了 start 或百分比 >0 → 进行中；
 * - 否则未开始（**未回填不等于没开工**，但按口径只能当未开始——面板要提示回填）。
 */
export function taskStatus(task: SchedTask, review: SchedReview | null | undefined): ExecStatus {
  const a = review?.actuals?.[task.id]
  if (a?.finish !== undefined) return '已完成'
  const pct = reviewPercent(review, task)
  if (pct >= 100) return '已完成'
  if (a?.start !== undefined || pct > 0) return '进行中'
  return '未开始'
}

/**
 * 整份计划的累计完成（0-100，按有效工期加权）。
 * 里程碑不进分母（工期 0）；分母为 0（全是里程碑）时退化为里程碑达成率。
 */
export function projectPercent(project: SchedProject, review: SchedReview | null | undefined): number {
  const leaves = leavesOf(project)
  let weight = 0
  let done = 0
  for (const t of leaves) {
    const d = effDur(t)
    if (d === 0) continue
    weight += d
    done += d * reviewPercent(review, t)
  }
  if (weight === 0) {
    const ms = leaves.filter((t) => t.isMilestone)
    if (ms.length === 0) return 0
    return Math.round((ms.filter((t) => reviewPercent(review, t) >= 100).length / ms.length) * 100)
  }
  return Math.round(done / weight)
}

/** 按期升序（不可变） */
export function sortedReviews(project: SchedProject): SchedReview[] {
  const list = Array.isArray(project.reviews) ? project.reviews : []
  return [...list].sort((a, b) => a.at - b.at)
}

/** 最新一期（无则 null） */
export function latestReview(project: SchedProject): SchedReview | null {
  const list = sortedReviews(project)
  return list.length > 0 ? list[list.length - 1] : null
}

/** 取 at ≤ 指定期的最近一期（用于"往前看某一期"） */
export function reviewAtOrBefore(project: SchedProject, at: number): SchedReview | null {
  const list = sortedReviews(project).filter((r) => r.at <= at)
  return list.length > 0 ? list[list.length - 1] : null
}

/** 由 ISO 日期算检视工作日序号（早于开工日 → 0；非工作日落到其后第一个工作日口径由 dateToWd 决定） */
export function reviewWorkday(project: SchedProject, dateISO: string): number {
  const wd = dateToWd(project.startDate, dateISO, project.calendar)
  return wd === null ? 0 : Math.max(0, wd)
}

/**
 * 用当前进度造一期检视（"把今天的样子打一个点"）。
 *
 * `actuals` 只填 `percent`（来自各任务当前 progress）；实际开始/完成留给用户在
 * 检视里回填——凭空推断开工日期是编数据，不干。已完成的（progress=100）顺带把
 * `finish` 填成 `at`（这是唯一可由"进度 100 + 数据日期"确定的信息）。
 */
export function makeReview(project: SchedProject, at: number, dateISO: string): SchedReview {
  const actuals: SchedReview['actuals'] = {}
  for (const t of leavesOf(project)) {
    const pct = Math.min(100, Math.max(0, Math.round(t.progress ?? 0)))
    if (pct <= 0) continue
    actuals[t.id] = pct >= 100 ? { percent: 100, finish: at } : { percent: pct }
  }
  return { at, date: dateISO, actuals }
}

/** 建议的检视日期：今天（YYYY-MM-DD），早于开工日则给开工日 */
export function defaultReviewDate(project: SchedProject, now = new Date()): string {
  const today = isoOf(now)
  return today < project.startDate ? project.startDate : today
}

/**
 * 期际对比（斑马「对比当前与历史检视」）：逐任务完成量差 + 整体完成量差。
 * `from` 为空表示"计划基线口径"（用任务当前 progress 作为起点）。
 */
export interface ReviewDiffRow {
  id: string
  name: string
  fromPct: number
  toPct: number
  delta: number
  status: ExecStatus
}

export interface ReviewDiff {
  fromAt: number | null
  toAt: number
  fromPercent: number
  toPercent: number
  deltaPercent: number
  /** 只含有变化的行（delta≠0）；按任务表序 */
  rows: ReviewDiffRow[]
}

export function compareReviews(
  project: SchedProject,
  from: SchedReview | null | undefined,
  to: SchedReview,
): ReviewDiff {
  const rows: ReviewDiffRow[] = []
  for (const t of leavesOf(project)) {
    const a = reviewPercent(from, t)
    const b = reviewPercent(to, t)
    if (a === b) continue
    rows.push({ id: t.id, name: t.name, fromPct: a, toPct: b, delta: b - a, status: taskStatus(t, to) })
  }
  const fromPercent = projectPercent(project, from)
  const toPercent = projectPercent(project, to)
  return {
    fromAt: from ? from.at : null,
    toAt: to.at,
    fromPercent,
    toPercent,
    deltaPercent: toPercent - fromPercent,
    rows,
  }
}

/**
 * 拉直（斑马「拉直前锋线」）：把某期的回填写回计划，并把该期冻结。
 *
 * 两条纪律：
 * - **已冻结的期不可再拉直**（返回 changed=0 且 frozen=true，让调用方给提示）；
 * - 回写是"只升不降"：实际完成百分比不会因为拉直而变小（避免把已完成又改回进行中）。
 *
 * 返回新 project（不改入参），便于 store 走历史栈。
 */
export function straighten(
  project: SchedProject,
  at: number,
): { project: SchedProject; changed: number; frozen: boolean } {
  const reviews = Array.isArray(project.reviews) ? project.reviews : []
  const target = reviews.find((r) => r.at === at)
  if (!target) return { project, changed: 0, frozen: false }
  if (target.frozen) return { project, changed: 0, frozen: true }

  let changed = 0
  const tasks = (project.tasks ?? []).map((t) => {
    if (t.level === 0) return t
    const pct = reviewPercent(target, t)
    const next = Math.max(Math.round(t.progress ?? 0), pct)
    if (next === (t.progress ?? 0)) return t
    changed++
    return { ...t, progress: next }
  })
  const nextReviews = reviews.map((r) => (r.at === at ? { ...r, frozen: true } : r))
  return { project: { ...project, tasks, reviews: nextReviews }, changed, frozen: false }
}

/**
 * 前锋线所在的工作日：最新一期检视的 `at`（没有检视则 null）。
 * 横道图/网络图按此画竖线——线在哪，就说明"我们现在站在哪一天看计划"。
 */
export function frontlineWorkday(project: SchedProject): number | null {
  const r = latestReview(project)
  return r ? r.at : null
}

/** 该期的计划应完成量 vs 实际完成量（斑马「当期/累计」两列的口径） */
export interface ReviewProgress {
  at: number
  /** 按 CPM 计划，到该工作日应完成的有效工期（工作日） */
  plannedWorkdays: number
  /** 实际已完成的有效工期（按回填百分比折算） */
  actualWorkdays: number
  /** 总有效工期 */
  totalWorkdays: number
  /** 计划应完成百分比 */
  plannedPercent: number
  /** 实际完成百分比 */
  actualPercent: number
  /** 偏差（正=超前） */
  gapPercent: number
}

export function reviewProgress(
  project: SchedProject,
  review: SchedReview,
  cpm?: CpmResult | null,
): ReviewProgress {
  let total = 0
  let planned = 0
  let actual = 0
  for (const t of leavesOf(project)) {
    const d = effDur(t)
    if (d === 0) continue
    total += d
    const row = cpm?.ok ? cpm.rows?.[t.id] : undefined
    if (row) {
      // 到 review.at 为止已跨过的工作日数（clamp 到 [0, d]）
      const elapsed = Math.min(d, Math.max(0, review.at - row.es + 1))
      planned += elapsed
    }
    actual += (d * reviewPercent(review, t)) / 100
  }
  const plannedPercent = total === 0 ? 0 : Math.round((planned / total) * 100)
  const actualPercent = total === 0 ? 0 : Math.round((actual / total) * 100)
  return {
    at: review.at,
    plannedWorkdays: Math.round(planned),
    actualWorkdays: Math.round(actual),
    totalWorkdays: total,
    plannedPercent,
    actualPercent,
    gapPercent: actualPercent - plannedPercent,
  }
}

/** 该期的日期文本（YYYY-MM-DD），由 at 反算（面板显示用） */
export function reviewDate(project: SchedProject, at: number): string {
  return isoOf(wdToDate(project.startDate, at, project.calendar))
}

/** 校验：能不能再加一期（上限 / 重复 / 超冻结） */
export function canAddReview(project: SchedProject, at: number): { ok: boolean; reason?: string } {
  const reviews = Array.isArray(project.reviews) ? project.reviews : []
  if (reviews.some((r) => r.at === at)) return { ok: false, reason: '该日期已有一期检视（同期只能有一条：改那一期，不并存）' }
  if (reviews.length >= MAX_REVIEWS) return { ok: false, reason: `检视期数已达上限 ${MAX_REVIEWS}，先删除早期检视` }
  if (reviews.some((r) => r.frozen && r.at > at)) {
    return { ok: false, reason: '不能插入早于已冻结检视的新期（会让前锋线倒流）' }
  }
  return { ok: true }
}
