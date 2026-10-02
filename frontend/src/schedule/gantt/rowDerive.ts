/**
 * schedule/gantt/rowDerive.ts — 横道行派生纯函数（FE7-05 单源）
 *
 * GanttTable（表格窗格）与 GanttBars（画布窗格）此前各自内联同一套逐行派生：
 * 合成组头判定（synSeq 闭包）/idx 提取/t·row·group·span·manual·dur 派生/
 * colorCls·dimCls 类名拼接——逐行镜像（审计 FE7-05）。本模块把它收敛为唯一
 * 实现：deriveRows 一次遍历产出两窗格共用的派生序列，纯函数零 DOM、可单测。
 *
 * 口径（与拆分前逐字段一致，行为零变化）：
 *  - 合成组头行 = kind==='group' 且 key 不以 `wbs:` 开头（分组模式的合成组头；
 *    不分组模式的 WBS 组头行 key=`wbs:原索引`，走任务行分支渲染）；
 *  - synSeq = 本行之前已出现的合成组头数（两窗格遍历同一 rows，序号天然一致，
 *    由 deriveRows 预派生取代各自维护的 `let synSeq = -1` 闭包）；
 *  - 任务行 idx = kind==='task' ? r.idx : Number(r.key.slice(4))（`wbs:` 前缀 4 字符）；
 *  - dur = 里程碑 0，否则 max(0, round(duration))；colorCls/dimCls 含前导空格，
 *    与行 className 直接拼接。
 */
import type { CpmResult, SchedProject, SchedTask, TaskCpm } from '../types'
import type { GanttRow } from '../ganttGroup'

/** 合成组头行（GanttRow 的 group 变体；syn 行渲染 label/count/minEs/maxEf） */
export type SynGroupRow = Extract<GanttRow, { kind: 'group' }>

/** 合成组头行判定（唯一判定源）：分组模式组头；`wbs:` 前缀=不分组 WBS 组头行，不是合成行。
 *  返回收窄变体或 null（布尔判定由 isSynGroupRow 派生）。 */
function synGroupOf(r: GanttRow): SynGroupRow | null {
  if (r.kind === 'group' && !r.key.startsWith('wbs:')) return r
  return null
}

/** 合成组头行布尔判定（测试/文档口径用；deriveRows 走 synGroupOf 取收窄变体） */
export function isSynGroupRow(r: GanttRow): boolean {
  return synGroupOf(r) !== null
}

/** 派生后的单行（FE7-05 单源）：syn=合成组头行（两窗格只共用判定与 synSeq），
 *  任务/WBS 行携带完整派生字段（两窗格按各自专有内容消费）。 */
export type DerivedGanttRow =
  | { syn: true; r: SynGroupRow; synSeq: number }
  | {
      syn: false
      r: GanttRow
      synSeq: number
      /** 任务原索引（`wbs:` 行=反解 key；task 行=idx） */
      i: number
      t: SchedTask
      /** CPM 行（与拆分前 `cpm.rows[t.id]` 同型，渲染按缺数据分支处理） */
      row: TaskCpm
      /** WBS 分组行（合成组头以外的组行：有汇总跨度/组配色，无叶行编辑件） */
      group: boolean
      /** 汇总跨度（仅 WBS 分组行，取组头统计 minEs/maxEf） */
      span: { es: number; ef: number } | null
      manual: boolean
      dur: number
      /** 行配色类（含前导空格；非组行=空串） */
      colorCls: string
      /** 路径分析淡化类（含前导空格；组行不淡化） */
      dimCls: string
    }

/**
 * 逐行派生（FE7-05 单源）：一次遍历 rows 产出表格/画布两窗格共用的派生序列。
 * groupColorSeq=组配色序号表（GanttView 按 level 0 循环预派生）；rowDim=路径
 * 分析淡化判定（闭包注入）。两窗格对同一 rows 各渲染一次，派生只算一遍。
 */
export function deriveRows(
  rows: GanttRow[],
  project: SchedProject,
  cpm: CpmResult,
  groupColorSeq: number[],
  rowDim: (id: string) => boolean,
): DerivedGanttRow[] {
  let synSeq = -1
  return rows.map((r) => {
    const syn = synGroupOf(r)
    if (syn) {
      synSeq++
      return { syn: true as const, r: syn, synSeq }
    }
    const i = r.kind === 'task' ? r.idx : Number(r.key.slice(4))
    const t = project.tasks[i]
    const row = cpm.rows[t.id]
    const group = r.kind === 'group'
    const span = group ? { es: r.minEs, ef: r.maxEf } : null
    const manual = t.mode === 'manual'
    const dur = t.isMilestone ? 0 : Math.max(0, Math.round(t.duration))
    const colorCls = group ? ` sched-group-c${groupColorSeq[i]}` : ''
    const dimCls = !group && rowDim(t.id) ? ' sched-row-dim' : ''
    return { syn: false as const, r, synSeq, i, t, row, group, span, manual, dur, colorCls, dimCls }
  })
}
