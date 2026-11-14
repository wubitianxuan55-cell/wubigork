/**
 * schedule/views.ts — 自定义视图模型（斑马复刻 S6.2「保存视图/视图管理」）
 *
 * 对标斑马 ViewDB 视图对象（viewbackend-re 2026-11-14 §1–§6）：**视图 = 列可见性
 * 预设 + 少量开关的命名快照**，九个内置视图共用同一套表格/图引擎——复刻同样不为
 * 每个视图建引擎，只切列集。gaea 口径：
 * - 列集 = `hide`（隐藏键集，语义与既有列显隐一致——斑马列对象的 `is_visible`
 *   是白名单、gaea 是黑名单，形态不同但表达力等价）；
 * - 分组 = `groupBy[]`（ganttGroup 维度，≤3 级）；
 * - 排序/筛选档 = `sort` / `filter`（filter 存 GanttFilterKind 档名，斑马
 *   column_filters 的表达式语法 gaea 暂未建模——诚实收窄，不造假字段）；
 * - `is_show_pert_editor`（显图区）/`lock_col`（冻结列）gaea v1 不建模：
 *   图区显隐是页面布局态，冻结列未实现——快照里不带死字段。
 *
 * 命名校验沿用斑马原文：「存在同名视图，请重新保存」。
 * 纯函数，无 store 依赖；快照源=GanttView 持久化的 chatPrefs（实时镜像）。
 */

import { MAX_VIEWS } from './blocks'
import { sanitizeHide } from './ganttCols'
import { GANTT_GROUP_OPTIONS, MAX_GANTT_GROUP_DEPTH, type GanttGroupField, type GanttSort, type GanttSortField } from './ganttGroup'
import { FILTER_KINDS, type GanttFilterKind } from './ganttFilter'
import type { SchedView } from './types'

/** 视图快照采集源（GanttView 的可持久化视图态；filter.text 为会话态不采集） */
export interface ViewSnapshot {
  hide: string[]
  groupBy: GanttGroupField[]
  sort: GanttSort
  filterKind: GanttFilterKind
}

const SORT_FIELDS: GanttSortField[] = ['none', 'name', 'duration', 'start', 'progress', 'tf']
const GROUP_FIELDS = GANTT_GROUP_OPTIONS.map((o) => o.value) as GanttGroupField[]

/** 校验能否保存（斑马同名校验原文；上限=blocks MAX_VIEWS=20） */
export function canSaveView(name: string, existing: SchedView[]): { ok: boolean; reason?: string } {
  const trimmed = name.trim()
  if (!trimmed) return { ok: false, reason: '视图名不能为空' }
  if (existing.some((v) => v.name === trimmed)) {
    return { ok: false, reason: '存在同名视图，请重新保存' }
  }
  if (existing.length >= MAX_VIEWS) {
    return { ok: false, reason: `视图数量已达上限 ${MAX_VIEWS}，请先删除不用的视图` }
  }
  return { ok: true }
}

/** 生成自定义视图（id 由调用方生成后传入；形状按 gaea 口径收窄，坏值丢弃不抛） */
export function makeCustomView(name: string, id: string, snap: ViewSnapshot, existing: SchedView[]): { view?: SchedView; reason?: string } {
  const gate = canSaveView(name, existing)
  if (!gate.ok) return { reason: gate.reason }
  const view: SchedView = { id, name: name.trim(), kind: 'gantt' }
  const hide = sanitizeHide(snap.hide)
  if (hide.length > 0) view.hide = hide
  const groupBy = snap.groupBy.filter((g) => GROUP_FIELDS.includes(g)).slice(0, MAX_GANTT_GROUP_DEPTH)
  if (groupBy.length > 0) view.groupBy = groupBy
  if (SORT_FIELDS.includes(snap.sort?.field)) {
    view.sort = { field: snap.sort.field, dir: snap.sort.dir === 'desc' ? 'desc' : 'asc' }
  }
  if (FILTER_KINDS.includes(snap.filterKind)) view.filter = snap.filterKind
  return { view }
}

/** 改名校验（排除自身后走同名原文） */
export function canRenameView(id: string, name: string, existing: SchedView[]): { ok: boolean; reason?: string } {
  const trimmed = name.trim()
  if (!trimmed) return { ok: false, reason: '视图名不能为空' }
  if (existing.some((v) => v.id !== id && v.name === trimmed)) {
    return { ok: false, reason: '存在同名视图，请重新保存' }
  }
  return { ok: true }
}

/** 应用快照 → GanttView 视图态（形状逐项容错，坏值回落缺省；不改入参） */
export function applyCustomViewSnapshot(
  view: SchedView,
): ViewSnapshot & { ok: boolean } {
  const hide = sanitizeHide(view.hide)
  const groupBy = (Array.isArray(view.groupBy) ? view.groupBy : []).filter(
    (g): g is GanttGroupField => GROUP_FIELDS.includes(g as GanttGroupField),
  ).slice(0, MAX_GANTT_GROUP_DEPTH)
  const sort: GanttSort = {
    field: SORT_FIELDS.includes(view.sort?.field as GanttSortField) ? (view.sort!.field as GanttSortField) : 'none',
    dir: view.sort?.dir === 'desc' ? 'desc' : 'asc',
  }
  const filterKind: GanttFilterKind = FILTER_KINDS.includes(view.filter as GanttFilterKind)
    ? (view.filter as GanttFilterKind)
    : 'all'
  return { hide, groupBy, sort, filterKind, ok: true }
}
