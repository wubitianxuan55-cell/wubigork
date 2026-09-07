/**
 * ganttFilter.ts — 横道任务行筛选（纯函数，v4.135.0 刀C 余项）
 *
 * 三种筛选模（全部/只看关键/只看手动）+ 名称文本包含（不区分大小写）。
 * 返回可见行的**原索引**序列（表格/条形/依赖线共用同一可见集，行号列仍按
 * 全表序号显示）。分组行保留规则：至少一个可见子孙才保留——分组行自身
 * 不是关键/手动，不能按叶条件误杀。网络图视图不参与筛选（逻辑图语义）。
 */
import type { CpmResult, SchedTask } from './types'
import { descendantIds } from './store'

export type GanttFilterKind = 'all' | 'critical' | 'manual'

export interface GanttFilter {
  kind: GanttFilterKind
  /** 名称包含文本（trim 后空串=不过滤；不区分大小写） */
  text: string
}

export const GANTT_FILTER_DEFAULT: GanttFilter = { kind: 'all', text: '' }

function leafVisible(t: SchedTask, f: GanttFilter, cpm: CpmResult): boolean {
  if (f.kind === 'critical') return cpm.rows[t.id]?.critical === true
  if (f.kind === 'manual') return t.mode === 'manual'
  return true
}

/**
 * 行筛选两遍法：
 * 1. 强制集=分组名命中文本的分组及其全部子孙（「搜支名看整支」的树搜索直觉）；
 * 2. 主遍历：叶=（模条件+文本）命中或被强制；分组=任一直接子孙可见或被强制。
 * 返回可见行原索引（行号列仍按全表序号显示）。
 */
export function filterGanttRows(tasks: SchedTask[], f: GanttFilter, cpm: CpmResult): number[] {
  const needle = f.text.trim().toLowerCase()
  const leafOk = (t: SchedTask): boolean =>
    leafVisible(t, f, cpm) && (!needle || t.name.toLowerCase().includes(needle))
  const forced = new Set<number>()
  for (let i = 0; i < tasks.length; i++) {
    const t = tasks[i]
    if (t.level === 0 && needle && t.name.toLowerCase().includes(needle)) {
      for (const id of descendantIds(tasks, t.id)) {
        const j = tasks.findIndex((x) => x.id === id)
        if (j > i) forced.add(j)
      }
    }
  }
  const out: number[] = []
  for (let i = 0; i < tasks.length; i++) {
    const t = tasks[i]
    if (t.level === 0) {
      const kids = descendantIds(tasks, t.id)
      let anyKid = false
      for (const id of kids) {
        const j = tasks.findIndex((x) => x.id === id)
        if (j > i && tasks[j].level > 0 && (forced.has(j) || leafOk(tasks[j]))) {
          anyKid = true
          break
        }
      }
      if (anyKid) out.push(i)
      continue
    }
    if (forced.has(i) || leafOk(t)) out.push(i)
  }
  return out
}
