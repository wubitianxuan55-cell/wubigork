/**
 * schedule/gschedSummary.ts — .gsched.json 摘要解析（v4.121.0 刀12 办公联动）
 *
 * 办公板块文件预览识别进度计划文件：把计划 JSON 文本解析为「工程 + CPM 摘要」，
 * 供办公侧计划摘要卡展示。纯函数零 DOM；任何解析/形状失败返回 null，
 * 预览端回落原始文本视图（宁回落勿误报）。
 */
import { computeCpm } from './cpm'
import { normalizeProject } from './store'
import { isoOf, wdToDate } from './calendar'
import { computeBaselineDrift } from './baseline'
import { checkDeadline, planFinishOf } from './deadline'
import { SCHEDULE_DEFAULT_REL } from './paths'
import type { SchedProject } from './types'

/**
 * 当前计划文件路径（缺省 rel；字面量单源在 schedule/paths.ts，FE7-09）。
 * v4.139 #15 多工程：本常量降级为「缺省 rel」——真实当前工程以 Go 侧索引指针
 * （store.currentPath，GaeaScheduleProjects 返回的 current）为准；索引不可用/
 * 旧 store 形态时各消费点（页头状态栏、办公卡 isCurrentPlan）回落本常量值。
 */
export const SCHEDULE_FILE_PATH = SCHEDULE_DEFAULT_REL

/** 路径是否为进度计划文件：.gsched.json 后缀（大小写不敏感，两种分隔符兼容） */
export function isScheduleFilePath(relPath: string): boolean {
  return relPath.replaceAll('\\', '/').toLowerCase().endsWith('.gsched.json')
}

export interface GschedSummary {
  project: SchedProject
  /** CPM 通过（false=循环依赖等，工期/关键/竣工不可算） */
  ok: boolean
  duration: number
  /** 叶任务数（level>0，与板块状态栏「共 N 项工作」同口径） */
  taskCount: number
  groupCount: number
  criticalCount: number
  linkCount: number
  milestoneCount: number
  /** 竣工日期（开工 + 总工期工作日；无叶任务/循环依赖时 null） */
  finishDate: string | null
  /** 基线漂移摘要（未保存基线时 null） */
  baseline: { name: string; savedAt: string; duration: number; drift: number } | null
  /** 保存的基线槽位数（v4.137 #11 多基线 max3；未保存=0） */
  baselineCount: number
  /** 倒排校核（未设目标竣工或 CPM 未通过时 null） */
  deadline: { date: string; targetWorkdays: number; feasible: boolean; overrun: number } | null
}

/** 解析计划 JSON → 摘要；坏 JSON / 非 Project 形状返回 null */
export function parseSchedSummary(raw: string): GschedSummary | null {
  return parseSchedSummaryDetail(raw).summary
}

/** FE7-14：把「为什么没摘要」如实交给调用方（此前 parseSchedSummary 失败只
 *  静默返 null，调用点无法区分「本来就不是计划文件」与「计划文件坏了」）。
 *  reason 为 null = 解析成功；非 null = 中文原因，调用方可据此渲染诚实提示。
 *  注意 reason 串不参与渲染去重（同一原因每次都新建，禁止拿它当 key 比较）。 */
export function parseSchedSummaryDetail(raw: string): { summary: GschedSummary | null; reason: string | null } {
  let data: unknown
  try {
    data = JSON.parse(raw)
  } catch (e: unknown) {
    const why = e instanceof Error ? e.message : String(e)
    return { summary: null, reason: `进度计划解析失败：JSON 不合法（${why}）` }
  }
  if (!data || typeof data !== 'object' || !Array.isArray((data as SchedProject).tasks)) {
    return { summary: null, reason: '进度计划解析失败：JSON 合法但不是计划工程形状（缺 tasks 数组）' }
  }
  return { summary: buildSchedSummary(data as SchedProject), reason: null }
}

function buildSchedSummary(data: SchedProject): GschedSummary {
  const project = normalizeProject(data)
  const cpm = computeCpm(project.tasks, project.links, { planFinish: planFinishOf(project), calendar: project.calendar, startDate: project.startDate })
  const leaf = project.tasks.filter((t) => t.level > 0)
  const hasSchedule = cpm.ok && leaf.length > 0 && /^\d{4}-\d{2}-\d{2}$/.test(project.startDate)
  const drift = cpm.ok ? computeBaselineDrift(project, cpm) : null
  const dl = cpm.ok ? checkDeadline(project, cpm) : null
  return {
    project,
    ok: cpm.ok,
    duration: cpm.duration,
    taskCount: leaf.length,
    groupCount: project.tasks.length - leaf.length,
    criticalCount: leaf.filter((t) => cpm.rows[t.id]?.critical).length,
    linkCount: project.links.length,
    milestoneCount: leaf.filter((t) => t.isMilestone).length,
    finishDate: hasSchedule ? isoOf(wdToDate(project.startDate, cpm.duration, project.calendar)) : null,
    baseline: drift
      ? { name: drift.baselineName, savedAt: drift.baselineSavedAt, duration: drift.baselineDuration, drift: drift.durationDrift }
      : null,
    baselineCount: project.baselines?.length ?? 0,
    deadline: dl ? { date: dl.deadline, targetWorkdays: dl.targetWorkdays, feasible: dl.feasible, overrun: dl.overrun } : null,
  }
}
