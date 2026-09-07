// applyDiff.ts — schedule_apply 事前 diff 的纯函数层（diff 确认闭环刀B）。
//
// Why: ask 级别下 schedule_apply 本来就弹审批卡，但卡上只有 path 一行字——
// 「有确认之形、无审阅之实」（设计 docs/gaea-schedule-diff-confirm-design-
// 2026-09.md §1.2）。本层把 before（文件实况）与 after（args.project）做
// **纯比较**，输出直接面向卡体渲染的结构化 diff；零绑定依赖，与 gschedSummary
// 同层同范式。ops 通道投影 simulateOps 是刀D（Go ApplyOps 的 TS 镜像），
// 本刀不涉及。
//
// 诚实纪律：预览层复刻引擎 fail-closed 口径——after 侧 CPM 不过就如实标注
// 「引擎将拒绝」，绝不把算不出的数字粉饰成 0；两侧算不出对比值就缺省，
// 绝不伪造 before 值。

import { computeCpm } from './cpm'
import { computeCosts } from './cost'
import { computeBaselineDrift } from './baseline'
import { checkDeadline, planFinishOf } from './deadline'
import { normalizeProject } from './store'
import type { CpmResult, SchedAssignment, SchedLink, SchedProject, SchedResource, SchedTask } from './types'

/** 字段级 from→to（label=字段中文名，面向渲染）。 */
export interface DiffFieldChange {
  label: string
  from: string
  to: string
}

/** 任务 modified 行：id 键控 + 字段级展开。 */
export interface TaskModify {
  id: string
  name: string
  fields: DiffFieldChange[]
}

export interface TaskChange {
  id: string
  name: string
}

/** 搭接入边变化（与 set_links「整体替换入边」语义对齐，按 to 任务分组呈现）。 */
export interface LinkChange {
  /** 入边归属任务名（to 端）。 */
  taskName: string
  from: string
  to: string
}

export interface ResourceChange {
  name: string
  /** 删除资源时随行级联删除的分配数。 */
  cascade?: number
  fields?: DiffFieldChange[]
}

export interface AssignmentChange {
  taskName: string
  resourceName: string
  fields?: DiffFieldChange[]
}

/** 卡头汇总行：预览层口径（数字是投影，落盘后以回执为准——卡上标注「预览」）。 */
export interface ApplyDiffSummary {
  durationFrom?: number
  durationTo?: number
  criticalFrom?: number
  criticalTo?: number
  costFrom?: number
  costTo?: number
  /** after 侧 CPM 未过：预览层复刻引擎 fail-closed（「引擎将拒绝」）。 */
  cpmError?: string
  /** 倒排校核预览（after 有 deadline 且 CPM 过时）：可行/超期 N 天。 */
  deadlineOk?: boolean
  deadlineOver?: number
  /** 基线漂移预览（after 有活跃基线且 CPM 过时）：挪移/新增/移除行数。 */
  baselineShifted?: number
  baselineAdded?: number
  baselineRemoved?: number
}

export interface ScheduleApplyDiff {
  tasks: { added: TaskChange[]; removed: TaskChange[]; modified: TaskModify[] }
  links: { added: LinkChange[]; removed: LinkChange[]; changed: LinkChange[] }
  resources: { added: ResourceChange[]; removed: ResourceChange[]; modified: TaskModify[] }
  assignments: { added: AssignmentChange[]; removed: AssignmentChange[] }
  meta: DiffFieldChange[]
  summary: ApplyDiffSummary
  /** 两侧完全一致（逐项空+汇总同值）：卡上可提示「无实质改动」。 */
  empty: boolean
}

const TASK_FIELD_LABELS: Array<{ key: keyof SchedTask; label: string; fmt?: (v: unknown) => string }> = [
  { key: 'name', label: '任务名' },
  { key: 'duration', label: '工期（工作日）' },
  { key: 'progress', label: '进度（%）' },
  { key: 'level', label: '大纲层级' },
  { key: 'isMilestone', label: '里程碑' },
  { key: 'mode', label: '排程模式' },
  { key: 'manualStart', label: '手动开始（工作日序号）' },
  { key: 'fixedCost', label: '固定成本（元）' },
]

const RESOURCE_FIELD_LABELS: Array<{ key: keyof SchedResource; label: string }> = [
  { key: 'name', label: '资源名' },
  { key: 'type', label: '类型' },
  { key: 'standardRate', label: '标准费率（元）' },
  { key: 'costPerUse', label: '每次使用成本（元）' },
  { key: 'maxUnits', label: '可用上限' },
]

function fmtVal(v: unknown): string {
  if (v === undefined || v === null || v === '') return '（无）'
  if (typeof v === 'boolean') return v ? '是' : '否'
  return String(v)
}

/** 数值/布尔等标量比较（undefined 与 undefined 视为同值）。 */
function sameScalar(a: unknown, b: unknown): boolean {
  const na = a === undefined || a === null ? undefined : a
  const nb = b === undefined || b === null ? undefined : b
  return na === nb
}

const linkKeyOf = (l: SchedLink) => `${l.from}\u0000${l.to}`
const linkLabel = (l: SchedLink, nameById: Map<string, string>) =>
  `${nameById.get(l.from) ?? l.from} → ${nameById.get(l.to) ?? l.to}（${l.type}${l.lag ? (l.lag > 0 ? `+${l.lag}` : `${l.lag}`) : ''}）`

/** project 通道 diff：两份 project 的纯比较（无模拟、无绑定）。before/after
 *  都先 normalizeProject 归一（缺省字段口径一致，避免归一差异冒充改动）。 */
export function diffProjects(beforeRaw: SchedProject, afterRaw: SchedProject): ScheduleApplyDiff {
  const before = normalizeProject(beforeRaw)
  const after = normalizeProject(afterRaw)

  // ── 任务（id 键控，modified 展开字段级 from→to）────────────────
  const beforeTasks = new Map(before.tasks.map((t) => [t.id, t]))
  const afterTasks = new Map(after.tasks.map((t) => [t.id, t]))
  const added: TaskChange[] = []
  const removed: TaskChange[] = []
  const modified: TaskModify[] = []
  for (const [id, t] of afterTasks) {
    const b = beforeTasks.get(id)
    if (!b) {
      added.push({ id, name: t.name })
      continue
    }
    const fields: DiffFieldChange[] = []
    for (const { key, label } of TASK_FIELD_LABELS) {
      const bv = (b as unknown as Record<string, unknown>)[key as string]
      const av = (t as unknown as Record<string, unknown>)[key as string]
      if (!sameScalar(bv, av)) fields.push({ label, from: fmtVal(bv), to: fmtVal(av) })
    }
    if (fields.length > 0) modified.push({ id, name: t.name, fields })
  }
  for (const [id, t] of beforeTasks) {
    if (!afterTasks.has(id)) removed.push({ id, name: t.name })
  }

  // ── 搭接（按 to 任务分组比对该任务入边集合）────────────────────
  const nameById = new Map(afterTasks.size > 0 ? after.tasks.map((t) => [t.id, t.name]) : before.tasks.map((t) => [t.id, t.name]))
  const beforeLinks = new Map(before.links.map((l) => [linkKeyOf(l), l]))
  const afterLinks = new Map(after.links.map((l) => [linkKeyOf(l), l]))
  const linkAdded: LinkChange[] = []
  const linkRemoved: LinkChange[] = []
  const linkChanged: LinkChange[] = []
  for (const [key, l] of afterLinks) {
    const b = beforeLinks.get(key)
    if (!b) {
      linkAdded.push({ taskName: nameById.get(l.to) ?? l.to, from: '', to: linkLabel(l, nameById) })
    } else if (b.type !== l.type || b.lag !== l.lag) {
      linkChanged.push({
        taskName: nameById.get(l.to) ?? l.to,
        from: linkLabel({ ...b, to: l.to }, nameById),
        to: linkLabel(l, nameById),
      })
    }
  }
  for (const [key, l] of beforeLinks) {
    if (!afterLinks.has(key)) {
      linkRemoved.push({ taskName: nameById.get(l.to) ?? l.to, from: linkLabel(l, nameById), to: '' })
    }
  }

  // ── 资源（id 键控 upsert/删除，删除标注级联分配数）──────────────
  const beforeRes = new Map(before.resources?.map((r) => [r.id, r]) ?? [])
  const afterRes = new Map(after.resources?.map((r) => [r.id, r]) ?? [])
  const resAdded: ResourceChange[] = []
  const resRemoved: ResourceChange[] = []
  const resModified: TaskModify[] = []
  for (const [id, r] of afterRes) {
    const b = beforeRes.get(id)
    if (!b) {
      resAdded.push({ name: r.name })
      continue
    }
    const fields: DiffFieldChange[] = []
    for (const { key, label } of RESOURCE_FIELD_LABELS) {
      const bv = (b as unknown as Record<string, unknown>)[key as string]
      const av = (r as unknown as Record<string, unknown>)[key as string]
      if (!sameScalar(bv, av)) fields.push({ label, from: fmtVal(bv), to: fmtVal(av) })
    }
    if (fields.length > 0) resModified.push({ id, name: r.name, fields })
  }
  for (const [id, r] of beforeRes) {
    if (!afterRes.has(id)) {
      const cascade = (before.assignments ?? []).filter((a) => a.resourceId === id).length
      resRemoved.push({ name: r.name, cascade: cascade > 0 ? cascade : undefined })
    }
  }

  // ── 分配（(taskId, resourceId) 键控，改 units/quantity/amount 展开）──
  const taskName = (id: string) => (afterTasks.get(id) ?? beforeTasks.get(id))?.name ?? id
  const resName = (id: string) => (afterRes.get(id) ?? beforeRes.get(id))?.name ?? id
  const asgKey = (a: SchedAssignment) => `${a.taskId}\u0000${a.resourceId}`
  const asgFields = (b: SchedAssignment, a: SchedAssignment): DiffFieldChange[] => {
    const out: DiffFieldChange[] = []
    const specs: Array<{ label: string; bv: unknown; av: unknown }> = [
      { label: '投入强度', bv: b.units, av: a.units },
      { label: '材料总量', bv: b.quantity, av: a.quantity },
      { label: '金额（元）', bv: b.amount, av: a.amount },
    ]
    for (const s of specs) if (!sameScalar(s.bv, s.av)) out.push({ label: s.label, from: fmtVal(s.bv), to: fmtVal(s.av) })
    return out
  }
  const beforeAsg = new Map((before.assignments ?? []).map((a) => [asgKey(a), a]))
  const afterAsg = new Map((after.assignments ?? []).map((a) => [asgKey(a), a]))
  const asgAdded: AssignmentChange[] = []
  const asgRemoved: AssignmentChange[] = []
  for (const [key, a] of afterAsg) {
    const b = beforeAsg.get(key)
    if (!b) {
      asgAdded.push({ taskName: taskName(a.taskId), resourceName: resName(a.resourceId), fields: b ? asgFields(b, a) : undefined })
    } else if (asgFields(b, a).length > 0) {
      asgAdded.push({ taskName: taskName(a.taskId), resourceName: resName(a.resourceId), fields: asgFields(b, a) })
    }
  }
  for (const [key, a] of beforeAsg) {
    if (!afterAsg.has(key)) {
      asgRemoved.push({ taskName: taskName(a.taskId), resourceName: resName(a.resourceId) })
    }
  }

  // ── meta（工程名/开工/日历/目标竣工）──────────────────────────
  const meta: DiffFieldChange[] = []
  if (before.name !== after.name) meta.push({ label: '工程名', from: before.name, to: after.name })
  if (before.startDate !== after.startDate) meta.push({ label: '开工日期', from: before.startDate, to: after.startDate })
  const calStr = (c: SchedProject['calendar']) =>
    c ? `工作制 ${[...c.workweek].join(',')}${c.holidays.length ? `，停工 ${c.holidays.length} 天` : ''}` : '（缺省周一~五）'
  if (JSON.stringify(before.calendar ?? null) !== JSON.stringify(after.calendar ?? null)) {
    meta.push({ label: '工作日历', from: calStr(before.calendar), to: calStr(after.calendar) })
  }
  if ((before.deadline ?? null) !== (after.deadline ?? null)) {
    meta.push({ label: '目标竣工', from: fmtVal(before.deadline), to: fmtVal(after.deadline) })
  }

  // ── 汇总（预览层复刻引擎 fail-closed：after CPM 不过如实标注）────
  const cpmOf = (p: SchedProject): CpmResult =>
    computeCpm(p.tasks, p.links, { planFinish: planFinishOf(p) })
  const cpmB = cpmOf(before)
  const cpmA = cpmOf(after)
  const costB = computeCosts(before, cpmB)
  const costA = computeCosts(after, cpmA)
  const summary: ApplyDiffSummary = {}
  if (cpmB.ok) summary.durationFrom = cpmB.duration
  if (cpmA.ok) summary.durationTo = cpmA.duration
  if (cpmB.ok) summary.criticalFrom = Object.values(cpmB.rows).filter((r) => r.critical).length
  if (cpmA.ok) summary.criticalTo = Object.values(cpmA.rows).filter((r) => r.critical).length
  if (costB.ok) summary.costFrom = costB.total
  if (costA.ok) summary.costTo = costA.total
  if (!cpmA.ok) summary.cpmError = cpmA.error || 'CPM 校验未过'
  if (cpmA.ok && after.deadline) {
    const chk = checkDeadline(after, cpmA)
    if (chk) {
      summary.deadlineOk = chk.feasible
      summary.deadlineOver = chk.feasible ? undefined : chk.overrun
    }
  }
  // 有基线附漂移预览（after 侧）：挪移/新增/移除行数进卡头（细节在板块基线面板）
  if (cpmA.ok) {
    const drift = computeBaselineDrift(after, cpmA)
    if (drift) {
      summary.baselineShifted = drift.rows.filter((r) => r.kind === 'shifted').length
      summary.baselineAdded = drift.rows.filter((r) => r.kind === 'added').length
      summary.baselineRemoved = drift.rows.filter((r) => r.kind === 'removed').length
    }
  }

  const empty =
    added.length === 0 && removed.length === 0 && modified.length === 0 &&
    linkAdded.length === 0 && linkRemoved.length === 0 && linkChanged.length === 0 &&
    resAdded.length === 0 && resRemoved.length === 0 && resModified.length === 0 &&
    asgAdded.length === 0 && asgRemoved.length === 0 && meta.length === 0 &&
    summary.durationFrom === summary.durationTo && summary.criticalFrom === summary.criticalTo &&
    summary.costFrom === summary.costTo

  return {
    tasks: { added, removed, modified },
    links: { added: linkAdded, removed: linkRemoved, changed: linkChanged },
    resources: { added: resAdded, removed: resRemoved, modified: resModified },
    assignments: { added: asgAdded, removed: asgRemoved },
    meta,
    summary,
    empty,
  }
}

/** 宿主取 args：Transcript 里该次 schedule_apply 调用（弹卡期间 status=running）
 *  的完整参数（设计 §1.3 ToolDispatch 先于闸门弹卡）。取不到返回 null=卡体降级。
 *  放纯函数层（非组件文件）以守 react-refresh 单一组件导出约束。 */
export function scheduleApplyArgsOf(items: ReadonlyArray<{ kind: string; name?: string; status?: string; args?: string }>): string | null {
  for (let i = items.length - 1; i >= 0; i--) {
    const it = items[i]
    if (it.kind === 'tool' && it.name === 'schedule_apply' && it.status === 'running') return it.args ?? null
  }
  return null
}

/** project 整量通道判定（前端镜像 Go control.scheduleApplyIsProjectChannel）：
 *  args.project 非空对象=整计划替换/生成 → 审批卡按 hardAsk 三钮形态渲染
 *  （禁会话记忆，拍板项 2）；ops 增量通道维持五钮（拍板项 3）。展示层判定，
 *  强制权威在 Go 闸门（alwaysPrompt 不读不写会话放行）。 */
export function scheduleApplyIsProjectChannel(args: string | null): boolean {
  if (!args) return false
  try {
    const parsed = JSON.parse(args) as { project?: unknown }
    return !!parsed.project && typeof parsed.project === 'object'
  } catch {
    return false
  }
}
