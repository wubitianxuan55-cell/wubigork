/**
 * schedule/warn.ts — 预警状态机（斑马复刻 S5，逆向报告 §5.3 口径）
 *
 * 斑马的预警回答一个问题：**这个工作有没有突破上级下达的计划时间（要求完成 askFinish），
 * 突破了多少天、什么档位**。三条口径都来自斑马行为，不是我们的发明：
 *
 * 1. **红绿灯**：红灯=突破上级下达的计划时间；绿灯=未突破（压线 overdue=0 也算绿）。
 *    未设置要求时间的工作没有预警（斑马「无级别」），不参与红灯计数。
 * 2. **两套超期算法并存、用户可切**：
 *    - `plan-ask`（计划-要求）：计划完成对要求完成；
 *    - `predict-ask`（预测-要求）：按当前实际推算的预测完成对要求完成。
 *    「完成日」统一取**最后工作日口径**：CPM 的 ef 是独占边界（es+d，cpm.ts 口径），
 *    有工期的工作完成日=ef−1；里程碑/零工期完成日=ef（=es）。
 *    预测口径=**均匀速率外推**（已完成的用回填 finish；进行中的按
 *    「已完成量/已跨工作日」的速率摊剩余量；未开始/无数据日期按计划完成）——
 *    与 reviews.ts 当期量的均匀摊近似同族（斑马 pert_json 同样无逐日曲线）。
 * 3. **滞后五档**：(0,15] / (15,30] / (30,60] / (60,100] / 100+ 天；只有红灯有档位
 *    （绿灯=未滞后，档位为空）。
 *
 * 边界（本刀不做）：斑马「预警天数升高/降低」的期际升降档子档（0-5/6-10/…）挂下刀；
 * 里程碑预警/工期风险预警的呈现分类由消费方按 isMilestone 自行取用。
 *
 * 纯函数，无 store 依赖；分组行不参与预警（返回 null）。
 */

import { dateToWd, isoOf, wdToDate } from './calendar'
import { leavesOf, reviewPercent } from './reviews'
import type { CpmResult, SchedProject, SchedReview, SchedTask, TaskCpm } from './types'

/** 超期算法（斑马两算法并存，用户可切） */
export type WarnAlgorithm = 'plan-ask' | 'predict-ask'

/** 预警级别：红灯（突破要求）/ 绿灯（未突破）/ 无级别（未设置要求时间或不可比） */
export type WarnLevel = 'red' | 'green' | 'none'

/** 单任务预警结论 */
export interface TaskWarn {
  taskId: string
  name: string
  isMilestone: boolean
  level: WarnLevel
  /** 超期天数（工作日）：正=突破要求完成（滞后）；0=压线；负=提前；null=不可比 */
  overdue: number | null
  /** 滞后档位（斑马五档文本；红灯才有，绿灯/无级别为 null） */
  band: string | null
  /** 参与比较的完成口径（计划 ef 或预测完成，工作日序号） */
  compareFinish: number | null
  /** 要求完成（工作日序号；未设置为 null） */
  askFinish: number | null
  algorithm: WarnAlgorithm
  /** 不可比对的原因（CPM 缺失/循环等），none 时尽量给 */
  note?: string
}

/** 滞后五档：只有 >0 的超期天数有档（斑马档位是「滞后」的档） */
export function warnBand(overdue: number): string | null {
  if (overdue <= 0) return null
  if (overdue <= 15) return '0-15天'
  if (overdue <= 30) return '15-30天'
  if (overdue <= 60) return '30-60天'
  if (overdue <= 100) return '60-100天'
  return '100天以上'
}

/**
 * 计划完成日（最后工作日口径）：ef 是独占边界（es+d），有工期 → ef−1；
 * 里程碑/零工期 → ef（=es，完成发生在当天）。
 */
function planFinishDay(task: SchedTask, row: TaskCpm): number {
  const d = (task.isMilestone || task.level === 0) ? 0 : Math.max(0, Math.round(task.duration))
  return d > 0 ? row.ef - 1 : row.ef
}

/**
 * 预测完成（工作日序号，最后工作日口径）——均匀速率外推，不是斑马内核的反编译口径
 * （未做该点反编译），与 reviews.ts 当期量同族的近似：已做多少/跨了多少天 = 速率，
 * 剩余量按速率摊。
 *
 * - 已完成（pct≥100）：回填 finish 优先，缺省回落检视日；
 * - 未开始（pct≤0）或无检视：无证据偏移，按计划完成日；
 * - 进行中：elapsed 从「回填 start ?? 计划 es」起算（至少 1 天），预测=检视日+剩余量/速率。
 */
export function predictFinishWd(
  task: SchedTask,
  row: TaskCpm | undefined,
  review: SchedReview | null | undefined,
): number | null {
  if (!row) return null
  const pct = reviewPercent(review, task)
  const d = (task.isMilestone || task.level === 0) ? 0 : Math.max(0, Math.round(task.duration))
  if (pct >= 100) {
    return review?.actuals?.[task.id]?.finish ?? review?.at ?? planFinishDay(task, row)
  }
  if (pct <= 0 || !review) return planFinishDay(task, row)
  if (d <= 0) return planFinishDay(task, row) // 里程碑/零工期：完成量分母为 0，按计划
  const startWd = review.actuals?.[task.id]?.start ?? row.es
  const elapsed = Math.max(1, review.at - startWd + 1)
  const doneWd = (d * pct) / 100
  const rate = doneWd / elapsed
  if (rate <= 0) return planFinishDay(task, row)
  const remaining = d - doneWd
  return Math.max(review.at, review.at + Math.ceil(remaining / rate))
}

/** 单任务预警结论（分组行返回 null；纯函数不改入参） */
export function taskWarn(
  task: SchedTask,
  cpm: CpmResult | null | undefined,
  review: SchedReview | null | undefined,
  algorithm: WarnAlgorithm = 'plan-ask',
): TaskWarn | null {
  if (task.level === 0) return null
  const ask = typeof task.askFinish === 'number' && Number.isInteger(task.askFinish) && task.askFinish >= 0
    ? task.askFinish
    : null
  const row = cpm?.ok ? cpm.rows?.[task.id] : undefined
  const base: TaskWarn = {
    taskId: task.id,
    name: task.name,
    isMilestone: !!task.isMilestone,
    level: 'none',
    overdue: null,
    band: null,
    compareFinish: null,
    askFinish: ask,
    algorithm,
  }
  if (ask === null) return base
  if (!row) return { ...base, note: 'CPM 未通过或任务未参与排程，无法比对要求完成' }
  const compareFinish = algorithm === 'predict-ask'
    ? predictFinishWd(task, row, review)
    : planFinishDay(task, row)
  if (compareFinish === null) return { ...base, note: '无排程数据，无法比对要求完成' }
  const overdue = compareFinish - ask
  if (overdue > 0) {
    return { ...base, level: 'red', overdue, band: warnBand(overdue), compareFinish }
  }
  return { ...base, level: 'green', overdue, band: null, compareFinish }
}

/** 整份计划预警计数 + 明细（斑马「网络图显示预警个数」口径；分组行不参与） */
export interface WarnSummary {
  red: number
  green: number
  /** 未设置要求时间的叶任务数（斑马「无级别」，不计入红绿灯） */
  unassigned: number
  /** 明细：红灯在前（overdue 降序），绿灯次之，无级别殿后 */
  items: TaskWarn[]
}

export function warnSummary(
  project: SchedProject,
  cpm: CpmResult | null | undefined,
  review: SchedReview | null | undefined,
  algorithm: WarnAlgorithm = 'plan-ask',
): WarnSummary {
  let red = 0
  let green = 0
  let unassigned = 0
  const items: TaskWarn[] = []
  for (const t of leavesOf(project)) {
    const w = taskWarn(t, cpm, review, algorithm)
    if (!w) continue
    items.push(w)
    if (w.level === 'red') red++
    else if (w.level === 'green') green++
    else unassigned++
  }
  return { red, green, unassigned, items: sortWarns(items, 'desc') }
}

/**
 * 排序（斑马「从提前到滞后/从滞后到提前排序」两向）：overdue 降序=从滞后到提前（缺省）。
 * null（不可比）恒殿后；同天按任务表序稳定。返回新数组不改入参。
 */
export function sortWarns(items: TaskWarn[], dir: 'desc' | 'asc' = 'desc'): TaskWarn[] {
  const idx = new Map(items.map((w, i) => [w.taskId, i]))
  const cmp = (a: TaskWarn, b: TaskWarn): number => {
    const av = a.overdue === null ? Number.NEGATIVE_INFINITY : a.overdue
    const bv = b.overdue === null ? Number.NEGATIVE_INFINITY : b.overdue
    return dir === 'desc' ? bv - av : av - bv
  }
  return [...items].sort((a, b) => cmp(a, b) || (idx.get(a.taskId)! - idx.get(b.taskId)!))
}

/** ISO 日期 → 要求完成工作日序号（早于开工日/不可解析回落 0=开工日；TaskInspector 编辑用） */
export function askFinishOfDate(project: SchedProject, dateISO: string): number {
  const wd = dateToWd(project.startDate, dateISO, project.calendar)
  return wd === null ? 0 : Math.max(0, wd)
}

/** 要求完成工作日序号 → ISO 日期（显示用） */
export function askFinishDateOf(project: SchedProject, wd: number): string {
  return isoOf(wdToDate(project.startDate, wd, project.calendar))
}
