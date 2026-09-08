/**
 * schedule/gantt/ganttUtil.ts — 横道图共享常量与纯函数（自 GanttView.tsx 拆分，行为零变化）
 *
 * 与 aoaRuler.ts 同 style：视图无关的取数/换算/格式化一律放这里，组件只做渲染。
 * 常量：行高/列宽表/缩放档位/周刻度阈值/星期标签/自定义字段索引；
 * 函数：前置/后续/任务名/行号/竣工偏移/分组成本/前锋点位/工作日格式化。
 * 依赖仅指向 schedule/ 根部纯模块（types/calendar/cpm/cost/ganttCols/customFields/store），
 * 不反向引用组件，无循环导入。
 */
import type { SchedProject, SchedTask } from '../types'
import type { CostResult } from '../cost'
import { isCd } from '../cpm'
import { wdToDate } from '../calendar'
import { descendantIds } from '../store'
import { GANTT_COLS } from '../ganttCols'
import { CUSTOM_FIELDS, type CustomFieldDef } from '../customFields'

export const ROW_H = 30

/** 列宽表（键=列 key，取 CUSTOM_FIELDS 追加列；与 ganttCols.ts 同源） */
export const W: Record<string, number> = Object.fromEntries(GANTT_COLS.map((c) => [c.key, c.w]))
/** 日宽连续缩放（v4.141）：下限 2px 防长计划爆画布，<12px 自动切周刻度层 */
export const DAY_W_MIN = 2
export const DAY_W_MAX = 40
export const DAY_W_DEFAULT = 20
export const ZOOM_FACTOR = 1.35
/** 周刻度切换阈值：dayW 低于此值时底层刻度由「日」切「周」（MS Project 口径） */
export const WEEK_TIER_BELOW = 12
export const WEEKDAY_LABELS = ['日', '一', '二', '三', '四', '五', '六']
/** 自定义字段注册表按 key 索引（行渲染 O(1) 取槽类型） */
export const CUSTOM_FIELD_BY_KEY: Record<string, CustomFieldDef> = Object.fromEntries(CUSTOM_FIELDS.map((f) => [f.key, f]))

/** 拖拽态（刀9）：move=拖移（auto 转手动锁定）、resize=右缘改工期 */
export interface GanttDragState {
  id: string
  kind: 'move' | 'resize'
  origEs: number
  origDur: number
  preview: number
  moved: boolean
}

/** 入边（前置）/出边（后续）引用（刀B 前置/后续列数据源） */
export function predsOf(id: string, project: SchedProject) {
  return project.links.filter((l) => l.to === id)
}
export function succsOf(id: string, project: SchedProject) {
  return project.links.filter((l) => l.from === id)
}
export function taskNameOf(project: SchedProject, id: string): string {
  return project.tasks.find((t) => t.id === id)?.name ?? id
}
/** 任务 id → 行号（Project 引用文本口径，1-based） */
export function noOf(project: SchedProject): (id: string) => number {
  return (id) => project.tasks.findIndex((t) => t.id === id) + 1
}

/** 目标竣工日期 → 开工日起自然日偏移（未设返回 -1） */
export function deadlineOff(deadline: string | null | undefined, startMs: number): number {
  if (!deadline) return -1
  const t = new Date(`${deadline}T00:00:00Z`).getTime()
  if (!Number.isFinite(t)) return -1
  return Math.round((t - startMs) / 86400000)
}

/** 分组行成本汇总（子孙叶任务 total 求和；复用扁平 WBS 滚动口径，同甘特汇总条） */
export function groupCost(project: SchedProject, costs: CostResult, idx: number): number {
  const ids = descendantIds(project.tasks, project.tasks[idx].id)
  let sum = 0
  for (const id of ids) sum += costs.rows[id]?.total ?? 0
  return sum
}

/** 前锋线任务点：按完成进度取前锋位置（工作日序号，可为小数）。
 *  cd 任务 dur=等效工作日跨度 ef−es（v4.152 刀3：进度×等效工期，非自然日数）。 */
export function frontierWd(t: SchedTask, row: { es: number; ef: number } | undefined, checkIdx: number): number | null {
  if (!row || t.level === 0) return null
  const dur = t.isMilestone ? 0 : isCd(t) ? row.ef - row.es : Math.max(0, Math.round(t.duration))
  if (row.es >= checkIdx) return row.es // 未开始
  const pct = Math.min(100, Math.max(0, t.progress)) / 100
  return Math.min(row.ef, row.es + pct * dur)
}

/** 工作日序号 → M/D 文本（按日历换算日期） */
export function fmtDate(startISO: string, workdayIdx: number, cal: SchedProject['calendar']): string {
  const d = wdToDate(startISO, workdayIdx, cal)
  return `${d.getUTCMonth() + 1}/${d.getUTCDate()}`
}

