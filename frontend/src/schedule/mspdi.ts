/**
 * schedule/mspdi.ts — MS Project XML（mspdi）导入/导出（纯函数，v4.111.0 刀2）
 *
 * 口径：务实子集——任务（大纲/工期/里程碑/进度/手动模式）、搭接（FS/SS/FF/SF+时距）、
 * 基准日历（周工作制+节假日例外）。时长换算按 MinutesPerDay=480（8h/日，ISO8601
 * PT#H#M#S），时距按 Project 惯例 LinkLag=十分之一分钟（4800/工作日）。
 * DayType 映射：Project 1=周日..7=周六 → JS getDay 0=周日..6=周六。
 * 真机 MS Project/WPS 兼容性走查入真机池（本项目纪律）。
 */
import type { LinkType, SchedCalendar, SchedLink, SchedProject, SchedTask, TaskMode } from './types'
import { dateToWd, normalizeCalendar, wdToDate } from './calendar'

/** Project 搭接类型枚举：0=FF 1=FS 2=SF 3=SS */
const P_TYPE: Record<LinkType, number> = { FF: 0, FS: 1, SF: 2, SS: 3 }
const P_TYPE_REV: Record<number, LinkType> = { 0: 'FF', 1: 'FS', 2: 'SF', 3: 'SS' }
const MINUTES_PER_DAY = 480
const TENTHS_PER_DAY = MINUTES_PER_DAY * 10

function xmlEsc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

function iso8601Days(days: number): string {
  return `PT${days * 8}H0M0S`
}

function dt(iso: string): string {
  return `${iso}T08:00:00`
}

/** WBS 编码：分组 n，叶 n.k（k 为组内序号，1 起） */
export function wbsCodes(tasks: SchedTask[]): string[] {
  const out: string[] = []
  let g = 0
  let k = 0
  for (const t of tasks) {
    if (t.level === 0) { g++; k = 0; out.push(`${g}`) } else { k++; out.push(`${g}.${k}`) }
  }
  return out
}

/** 导出 mspdi XML（须先经 computeCpm 得到 rows；分组行 Start/Finish 按子孙滚动） */
export function buildProjectXml(project: SchedProject, rows: Record<string, { es: number; ef: number }>, calInput?: SchedCalendar): string {
  const cal = normalizeCalendar(calInput ?? project.calendar)
  const wbs = wbsCodes(project.tasks)
  const lines: string[] = []
  lines.push('<?xml version="1.0" encoding="UTF-8" standalone="yes"?>')
  lines.push('<Project xmlns="http://schemas.microsoft.com/project">')
  lines.push(`  <Name>${xmlEsc(project.name)}</Name><Title>${xmlEsc(project.name)}</Title>`)
  lines.push(`  <StartDate>${dt(project.startDate)}</StartDate>`)
  lines.push('  <ScheduleFromStart>1</ScheduleFromStart>')
  lines.push(`  <MinutesPerDay>${MINUTES_PER_DAY}</MinutesPerDay><DaysPerMonth>20</DaysPerMonth>`)
  // 基准日历：DayType 1=周日..7=周六
  lines.push('  <Calendars><Calendar><UID>1</UID><Name>标准</Name><IsBaseCalendar>1</IsBaseCalendar>')
  lines.push('    <WeekDays>')
  for (let dayType = 1; dayType <= 7; dayType++) {
    const jsDay = dayType - 1
    const working = cal.workweek.includes(jsDay)
    lines.push(`      <WeekDay><DayType>${dayType}</DayType><DayWorking>${working ? 1 : 0}</DayWorking>`)
    if (working) {
      lines.push('        <WorkingTimes><WorkingTime><FromTime>08:00:00</FromTime><ToTime>17:00:00</ToTime></WorkingTime></WorkingTimes>')
    }
    lines.push('      </WeekDay>')
  }
  lines.push('    </WeekDays>')
  if (cal.holidays.length > 0) {
    lines.push('    <Exceptions>')
    for (const h of [...cal.holidays].sort()) {
      lines.push(`      <Exception><TimePeriod><FromDate>${dt(h)}</FromDate><ToDate>${dt(h)}</ToDate></TimePeriod><DayWorking>0</DayWorking></Exception>`)
    }
    lines.push('    </Exceptions>')
  }
  lines.push('  </Calendar></Calendars>')

  const uid = new Map(project.tasks.map((t, i) => [t.id, i + 1]))
  lines.push('  <Tasks>')
  project.tasks.forEach((t, i) => {
    const row = rows[t.id] ?? { es: 0, ef: 0 }
    const dur = t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
    const summary = t.level === 0
    const startD = wdToDate(project.startDate, summary ? summaryStart(project, rows, i) : row.es, cal)
    const finishD = wdToDate(project.startDate, summary ? summaryFinish(project, rows, i) : row.ef, cal)
    lines.push('    <Task>')
    lines.push(`      <UID>${uid.get(t.id)}</UID><ID>${i + 1}</ID><WBS>${xmlEsc(wbs[i])}</WBS>`)
    lines.push(`      <Name>${xmlEsc(t.name)}</Name>`)
    lines.push(`      <Type>0</Type><IsSummary>${summary ? 1 : 0}</IsSummary>`)
    lines.push(`      <Milestone>${t.isMilestone || (dur === 0 && !summary) ? 1 : 0}</Milestone>`)
    lines.push(`      <Duration>${iso8601Days(summary ? summaryFinish(project, rows, i) - summaryStart(project, rows, i) : dur)}</Duration>`)
    lines.push('      <DurationFormat>7</DurationFormat>')
    lines.push(`      <OutlineLevel>${Math.max(1, t.level + 1)}</OutlineLevel>`)
    lines.push(`      <Start>${dt(startD.toISOString().slice(0, 10))}</Start><Finish>${dt(finishD.toISOString().slice(0, 10))}</Finish>`)
    lines.push(`      <PercentComplete>${Math.min(100, Math.max(0, Math.round(t.progress)))}</PercentComplete>`)
    lines.push(`      <Manual>${t.mode === 'manual' ? 1 : 0}</Manual>`)
    for (const l of project.links.filter((x) => x.to === t.id)) {
      lines.push(`      <PredecessorLink><PredecessorUID>${uid.get(l.from)}</PredecessorUID><Type>${P_TYPE[l.type]}</Type><LinkLag>${Math.round(l.lag * TENTHS_PER_DAY)}</LinkLag><LagFormat>7</LagFormat></PredecessorLink>`)
    }
    lines.push('    </Task>')
  })
  lines.push('  </Tasks>')
  lines.push('</Project>')
  return lines.join('\n')

  function summaryStart(p: SchedProject, r: Record<string, { es: number; ef: number }>, idx: number): number {
    let es = Number.POSITIVE_INFINITY
    for (let j = idx + 1; j < p.tasks.length && p.tasks[j].level > p.tasks[idx].level; j++) {
      es = Math.min(es, r[p.tasks[j].id]?.es ?? 0)
    }
    return Number.isFinite(es) ? es : 0
  }
  function summaryFinish(p: SchedProject, r: Record<string, { es: number; ef: number }>, idx: number): number {
    let ef = 0
    for (let j = idx + 1; j < p.tasks.length && p.tasks[j].level > p.tasks[idx].level; j++) {
      ef = Math.max(ef, r[p.tasks[j].id]?.ef ?? 0)
    }
    return ef
  }
}

/** 解析 ISO8601 时长（PT#H#M#S）→ 工作日数 */
function parseDurationDays(s: string | undefined): number {
  if (!s) return 0
  const m = /^-?PT(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:(\d+(?:\.\d+)?)S)?$/.exec(s.trim())
  if (!m) return 0
  const minutes = (Number(m[1]) || 0) * 60 + (Number(m[2]) || 0) + (Number(m[3]) || 0) / 60
  return Math.round(minutes / MINUTES_PER_DAY)
}

function textOf(el: Element | null | undefined, tag: string): string | undefined {
  const n = el?.getElementsByTagName(tag)[0]
  return n?.textContent ?? undefined
}

/** 解析日期为 YYYY-MM-DD（容忍带时间的 ISO） */
function parseISODate(s: string | undefined): string | undefined {
  if (!s) return undefined
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(s.trim())
  return m ? `${m[1]}-${m[2]}-${m[3]}` : undefined
}

export interface MspdiParseResult {
  ok: boolean
  error?: string
  project?: SchedProject
}

/** 导入 mspdi XML → SchedProject（宽松容错：缺失字段走缺省） */
export function parseProjectXml(text: string): MspdiParseResult {
  let doc: Document
  try {
    doc = new DOMParser().parseFromString(text, 'text/xml')
  } catch (e) {
    return { ok: false, error: `XML 解析失败：${e}` }
  }
  if (doc.getElementsByTagName('parsererror').length > 0) {
    return { ok: false, error: 'XML 解析失败：格式不合法' }
  }
  const projectEl = doc.getElementsByTagName('Project')[0]
  if (!projectEl) return { ok: false, error: '缺少 Project 根节点（非 MS Project XML？）' }

  const startDate = parseISODate(textOf(projectEl, 'StartDate')) ?? new Date().toISOString().slice(0, 10)
  const name = textOf(projectEl, 'Name') ?? textOf(projectEl, 'Title') ?? '导入的工程'

  // 基准日历（第一个 IsBaseCalendar=1 或第一个 Calendar）
  let calendar: SchedCalendar | undefined
  const calEls = Array.from(doc.getElementsByTagName('Calendar'))
  const cal = calEls.find((c) => textOf(c, 'IsBaseCalendar') === '1') ?? calEls[0]
  if (cal) {
    const workweek: number[] = []
    for (const wd of Array.from(cal.getElementsByTagName('WeekDay'))) {
      const dayType = Number(textOf(wd, 'DayType'))
      if (!Number.isInteger(dayType) || dayType < 1 || dayType > 7) continue
      if (textOf(wd, 'DayWorking') === '0') continue
      workweek.push(dayType - 1) // 1=周日..7=周六 → 0=周日..6=周六
    }
    const holidays: string[] = []
    for (const ex of Array.from(cal.getElementsByTagName('Exception'))) {
      if (textOf(ex, 'DayWorking') !== '0') continue
      const from = parseISODate(textOf(ex, 'FromDate'))
      const to = parseISODate(textOf(ex, 'ToDate'))
      if (!from) continue
      holidays.push(from)
      // 连续区间展开（上限 500 天防呆）
      if (to && to !== from) {
        let cur = new Date(`${from}T00:00:00Z`)
        const end = new Date(`${to}T00:00:00Z`)
        for (let i = 0; i < 500 && cur < end; i++) {
          cur = new Date(cur.getTime() + 86400000)
          holidays.push(cur.toISOString().slice(0, 10))
        }
      }
    }
    if (workweek.length > 0) calendar = { workweek, holidays }
  }

  const tasks: SchedTask[] = []
  const links: SchedLink[] = []
  const uidToId = new Map<string, string>()
  const taskEls = Array.from(doc.getElementsByTagName('Task')).filter((t) => t.parentElement === projectEl || t.parentElement?.tagName === 'Tasks')
  let seq = 0
  for (const el of taskEls) {
    const taskUid = textOf(el, 'UID') ?? `u${seq + 1}`
    const id = `t${++seq}`
    uidToId.set(taskUid, id)
    const name2 = textOf(el, 'Name') ?? `任务${seq}`
    const isSummary = textOf(el, 'IsSummary') === '1'
    const milestone = textOf(el, 'Milestone') === '1'
    const durRaw = textOf(el, 'Duration') ? parseDurationDays(textOf(el, 'Duration')) : textOf(el, 'ManualDuration') ? parseDurationDays(textOf(el, 'ManualDuration')) : 0
    const mode: TaskMode = textOf(el, 'Manual') === '1' ? 'manual' : 'auto'
    const progress = Math.min(100, Math.max(0, Math.round(Number(textOf(el, 'PercentComplete') ?? 0) || 0)))
    const task: SchedTask = {
      id,
      name: name2,
      duration: isSummary || milestone ? 0 : Math.max(0, durRaw),
      // 两级模型：汇总行=分组(0)，其余（含 OutlineLevel=1 顶层级叶子）一律叶(1)
      level: isSummary ? 0 : 1,
      progress,
    }
    if (milestone) task.isMilestone = true
    if (mode === 'manual') {
      task.mode = 'manual'
      const mStart = parseISODate(textOf(el, 'ManualStart') ?? textOf(el, 'Start'))
      task.manualStart = mStart ? (dateToWd(startDate, mStart, calendar) ?? 0) : 0
    }
    tasks.push(task)
  }
  // 搭接第二遍（依赖 uidToId 全量建好）
  for (const el of taskEls) {
    const toId = uidToId.get(textOf(el, 'UID') ?? '')
    if (!toId) continue
    for (const pl of Array.from(el.getElementsByTagName('PredecessorLink'))) {
      const fromId = uidToId.get(textOf(pl, 'PredecessorUID') ?? '')
      const type = P_TYPE_REV[Number(textOf(pl, 'Type') ?? 1)] ?? 'FS'
      const lagRaw = Number(textOf(pl, 'LinkLag') ?? 0) || 0
      if (!fromId || fromId === toId) continue
      links.push({ from: fromId, to: toId, type, lag: Math.round(lagRaw / TENTHS_PER_DAY) })
    }
  }
  if (tasks.length === 0) return { ok: false, error: 'XML 中没有可导入的任务' }
  return { ok: true, project: { name, startDate, tasks, links, ...(calendar ? { calendar } : {}) } }
}
