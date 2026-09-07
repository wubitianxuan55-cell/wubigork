/**
 * usage.ts — 资源使用/超载计算纯函数（v4.137 #12）
 *
 * 与 UsageView 并行开发，只共享 usageTypes.ts 钉死的契约；口径逐条对齐其头注释：
 *  - cpm fail-closed（ok=false，循环依赖等）→ 整体 fail-closed（days=0、rows=[]），
 *    不静默出半结果；
 *  - 时间轴=项目工作日序号 wd（0..cpm.duration-1），与 es/ef 同一口径；
 *  - rows 只收 work（工时）型资源——材料按总量计价、成本按金额计价，无「按天
 *    投入」语义；按 name localeCompare('zh') 排序，无分配的工时资源保留=全零行；
 *  - workload(资源,wd)=Σ 在工任务 units（缺省 1；在工=es ≤ wd < ef 的叶任务，
 *    auto/manual 均按 CPM 排程结果；里程碑 ef=es 永不在工——含头不含尾的区间
 *    判定天然排除，无需特判）；
 *  - availability(资源,wd)=maxUnits（缺省 1）；资源设了个人日历（v4.137 #13）
 *    时在项目工作日上进一步收紧：wd 对应自然日非其工作日 → 0（个人休假/停工）；
 *  - 超载 overloaded = workload > availability + 1e-9（浮点容差）。
 */
import { isWorkingDate, normalizeCalendar, wdToDate } from './calendar'
import type { CpmResult, SchedProject } from './types'
import type { UsageActiveTask, UsageDayCell, UsageResult, UsageResourceRow } from './usageTypes'

/** 浮点容差：workload 超出 availability 至少该余量才算超载 */
const OVERLOAD_EPS = 1e-9

export function computeUsage(project: SchedProject, cpm: CpmResult): UsageResult {
  // cpm fail-closed：整体 fail-closed（与 pathDriver 等口径一致）
  if (!cpm.ok) return { ok: false, error: cpm.error, days: 0, rows: [], totalOverloadDays: 0 }

  const days = cpm.duration

  // rows：只收 work 型资源，按名称 zh 序；无分配的工时资源也保留（全零行）
  const workRes = (project.resources ?? []).filter((r) => r.type === 'work')
  workRes.sort((a, b) => a.name.localeCompare(b.name, 'zh'))

  const byId = new Map(project.tasks.map((t) => [t.id, t]))
  const orderOf = new Map(project.tasks.map((t, i) => [t.id, i]))

  // 每资源的在工任务：assignment（units 缺省 1）→ 叶任务 → cpm 行（分组行/缺行跳过）
  const actsByRes = new Map<string, UsageActiveTask[]>()
  for (const a of project.assignments ?? []) {
    const task = byId.get(a.taskId)
    if (!task || task.level === 0) continue // 分配只对叶任务生效（分组行禁止分配）
    const row = cpm.rows[a.taskId]
    if (!row) continue
    const list = actsByRes.get(a.resourceId) ?? []
    list.push({ taskId: a.taskId, name: task.name, es: row.es, ef: row.ef, units: a.units ?? 1 })
    actsByRes.set(a.resourceId, list)
  }
  // 在工任务排序：es 升序，同 es 按任务表序
  const orderOfTask = (a: UsageActiveTask) => orderOf.get(a.taskId) ?? 0
  const byStart = (x: UsageActiveTask, y: UsageActiveTask) => x.es - y.es || orderOfTask(x) - orderOfTask(y)

  // 个人日历可用性要按 wd 反查自然日：整条时间轴换算一次，各行复用
  const needDates = workRes.some((r) => r.calendar)
  const wdDates = needDates ? Array.from({ length: days }, (_, wd) => wdToDate(project.startDate, wd, project.calendar)) : []

  let totalOverloadDays = 0
  const rows: UsageResourceRow[] = workRes.map((res) => {
    const acts = (actsByRes.get(res.id) ?? []).sort(byStart)
    const maxUnits = res.maxUnits ?? 1
    const resCal = res.calendar ? normalizeCalendar(res.calendar) : null
    let peakWorkload = 0
    let overloadDays = 0
    let totalWorkload = 0
    const cells = Array.from({ length: days }, (_, wd): UsageDayCell => {
      let workload = 0
      for (const a of acts) {
        if (a.es <= wd && wd < a.ef) workload += a.units // 含头不含尾；里程碑 ef=es 永不在工
      }
      // 个人日历在项目工作日上进一步收紧：wd 的自然日非资源工作日 → 0（休假/停工）
      const availability = resCal && !isWorkingDate(wdDates[wd], resCal) ? 0 : maxUnits
      const overloaded = workload > availability + OVERLOAD_EPS
      if (overloaded) overloadDays++
      if (workload > peakWorkload) peakWorkload = workload
      totalWorkload += workload
      return { wd, workload, availability, overloaded }
    })
    totalOverloadDays += overloadDays
    return {
      resourceId: res.id,
      name: res.name,
      type: res.type,
      tasks: acts,
      cells,
      peakWorkload,
      overloadDays,
      totalWorkload,
    }
  })

  return { ok: true, days, rows, totalOverloadDays }
}
