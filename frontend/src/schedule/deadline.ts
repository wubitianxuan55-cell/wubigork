/**
 * deadline.ts — 倒排校核（纯函数，v4.117.0 刀8）
 *
 * 对齐 Project「必须完成期限」：目标是**引擎裁决**合同/定额竣工约束的可行性，
 * 不自动改排程（自动锁死任务违背 manual 语义；压缩方案由 AI 建议、用户拍板）。
 * 目标总工期=deadlineWorkdays 换算（工作日口径）；可行性=当前总工期 ≤ 目标。
 * 与 Go 侧 internal/schedule/deadline.go 互为镜像（测试对齐）。
 */
import type { CpmResult, SchedProject } from './types'
import { deadlineWorkdays } from './calendar'

/** 倒排校核结果 */
export interface DeadlineCheck {
  /** 目标竣工日期（YYYY-MM-DD） */
  deadline: string
  /** 目标总工期（工作日，1-based：第 N 个工作日竣工） */
  targetWorkdays: number
  currentDuration: number
  /** 可行：当前总工期 ≤ 目标 */
  feasible: boolean
  /** 超期量（正=超期工作日，负=富余，0=压线达成） */
  overrun: number
  /** 关键工作名清单（超期时的压缩对象；手动不计） */
  criticalTasks: string[]
}

/** 倒排校核。未设目标竣工或计划未通过 CPM 返回 null。 */
export function checkDeadline(project: SchedProject, cpm: CpmResult): DeadlineCheck | null {
  if (!project.deadline || !cpm.ok) return null
  const target = deadlineWorkdays(project.startDate, project.deadline, project.calendar)
  const criticalTasks = project.tasks
    .filter((t) => t.level > 0 && t.mode !== 'manual' && cpm.rows[t.id]?.critical)
    .map((t) => t.name)
  return {
    deadline: project.deadline,
    targetWorkdays: target,
    currentDuration: cpm.duration,
    feasible: cpm.duration <= target,
    overrun: cpm.duration - target,
    criticalTasks,
  }
}

/**
 * 计划工期锚点（v4.129.0 刀G，G3）：目标竣工 → 工作日边界索引（第 N 工作日竣工
 * = N）。规程口径：计划工期 Tp 小于计算工期 Tc 时逆推从 Tp 起（负时差诚实呈现）。
 * 未设目标竣工返回 null（调用方按计算工期逆推，与旧口径零差异）。
 */
export function planFinishOf(project: Pick<SchedProject, 'startDate' | 'deadline' | 'calendar'>): number | null {
  if (!project.deadline) return null
  return deadlineWorkdays(project.startDate, project.deadline, project.calendar)
}
