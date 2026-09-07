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
import type { SchedProject } from './types'

/**
 * 当前计划文件路径（与 Go internal/schedule/project.go DefaultRelPath 镜像）。
 * v4.139 #15 多工程：本常量降级为「缺省 rel」——真实当前工程以 Go 侧索引指针
 * （store.currentPath，GaeaScheduleProjects 返回的 current）为准；索引不可用/
 * 旧 store 形态时各消费点（页头状态栏、办公卡 isCurrentPlan）回落本常量值。
 */
export const SCHEDULE_FILE_PATH = '进度计划/当前计划.gsched.json'

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
  /** 倒排校核（未设目标竣工或 CPM 未通过时 null） */
  deadline: { date: string; targetWorkdays: number; feasible: boolean; overrun: number } | null
}

/** 解析计划 JSON → 摘要；坏 JSON / 非 Project 形状返回 null */
export function parseSchedSummary(raw: string): GschedSummary | null {
  let data: unknown
  try {
    data = JSON.parse(raw)
  } catch {
    return null
  }
  if (!data || typeof data !== 'object' || !Array.isArray((data as SchedProject).tasks)) return null
  const project = normalizeProject(data as SchedProject)
  const cpm = computeCpm(project.tasks, project.links, { planFinish: planFinishOf(project) })
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
    deadline: dl ? { date: dl.deadline, targetWorkdays: dl.targetWorkdays, feasible: dl.feasible, overrun: dl.overrun } : null,
  }
}
