/**
 * viewPresets.ts — 斑马 ZPert8 内置横道视图（W1）
 *
 * 逆向证据：ViewDB.gdb 的 9 个内置视图与「视图 = 列集 + 分组 + 筛选 + 排序 +
 * 显示模式」语义（UIUX 报告 §19.3–§19.4）。
 *
 * 这里只定义**内置预设**，不写入 `.gsched.json`；项目里的 `SchedView` 仍是
 * 用户另存视图的权威承载位。横道 UI 选择预设后，把差异落回既有列显隐、
 * 分组、排序与筛选状态，避免出现第二套视图执行器。
 */
import { GANTT_COLS, GANTT_HIDEABLE_KEYS } from './ganttCols'
import type { GanttGroupField, GanttSort } from './ganttGroup'
import type { GanttFilterKind } from './ganttFilter'

export const CUSTOM_GANTT_VIEW_ID = 'custom'
export const DEFAULT_GANTT_VIEW_ID = 'plan'

export interface BuiltinGanttView {
  id: string
  name: string
  description: string
  /** 当前视图可见的列键；固定列仍会在 visibleCols 内兜底保留。 */
  visible: readonly string[]
  groupBy: readonly GanttGroupField[]
  sort: GanttSort
  filterKind: GanttFilterKind
}

const PLAN_VISIBLE = ['no', 'partition', 'marker', 'name', 'dur', 'preds', 'start', 'finish'] as const
const ZPERT_TABLE_W = 730

/**
 * 顺序与 ViewDB 解密结果一致。当前只有既有横道列可被消费；资源/工程量等
 * 不适用的高级字段用现有资源、进度、自定义槽承接，缺失时由选择器诚实降级。
 */
export const BUILTIN_GANTT_VIEWS: readonly BuiltinGanttView[] = [
  {
    id: 'plan',
    name: '计划编制',
    description: '工作名称、分区、工期、前置与计划日期（默认）',
    visible: PLAN_VISIBLE,
    groupBy: [],
    sort: { field: 'none', dir: 'asc' },
    filterKind: 'all',
  },
  {
    id: 'control',
    name: '进度管控',
    description: '按执行状态分组，突出进度、时差与日期偏差',
    visible: ['no', 'partition', 'marker', 'name', 'dur', 'progress', 'start', 'finish', 'tf'],
    groupBy: ['status'],
    sort: { field: 'start', dir: 'asc' },
    filterKind: 'all',
  },
  {
    id: 'period-plan',
    name: '期间计划',
    description: '当期计划窗口的紧凑编制列',
    visible: ['no', 'partition', 'marker', 'name', 'dur', 'preds', 'start', 'finish'],
    groupBy: ['category'],
    sort: { field: 'start', dir: 'asc' },
    filterKind: 'all',
  },
  {
    id: 'period-report',
    name: '期间进度报告',
    description: '按工作分类汇总完成状态与计划日期',
    visible: ['no', 'partition', 'marker', 'name', 'mode', 'progress', 'start', 'finish'],
    groupBy: ['category'],
    sort: { field: 'start', dir: 'asc' },
    filterKind: 'all',
  },
  {
    id: 'resource',
    name: '资源',
    description: '按资源名称聚合工期、进度与成本',
    visible: ['no', 'partition', 'marker', 'name', 'dur', 'progress', 'cost'],
    groupBy: ['resource'],
    sort: { field: 'start', dir: 'asc' },
    filterKind: 'all',
  },
  {
    id: 'quantity',
    name: '工程量计划',
    description: '工作面、工期、进度与数值自定义列',
    visible: ['no', 'partition', 'marker', 'name', 'dur', 'progress', 'num1', 'num2'],
    groupBy: ['site'],
    sort: { field: 'start', dir: 'asc' },
    filterKind: 'all',
  },
  {
    id: 'compare',
    name: '进度对比分析',
    description: '按执行状态对比计划日期、完成量与总时差',
    visible: ['no', 'partition', 'marker', 'name', 'dur', 'progress', 'start', 'finish', 'tf', 'ff'],
    groupBy: ['status'],
    sort: { field: 'start', dir: 'asc' },
    filterKind: 'all',
  },
  {
    id: 'change',
    name: '计划变动分析',
    description: '突出关键工作、排程模式与计划窗口',
    visible: ['no', 'partition', 'marker', 'name', 'dur', 'mode', 'progress', 'start', 'finish'],
    groupBy: ['critical'],
    sort: { field: 'start', dir: 'asc' },
    filterKind: 'all',
  },
  {
    id: 'milestone',
    name: '里程碑列表',
    description: '只看里程碑节点，保留所属分区与计划日期',
    visible: ['no', 'partition', 'marker', 'name', 'mode', 'start', 'finish'],
    groupBy: [],
    sort: { field: 'start', dir: 'asc' },
    filterKind: 'milestone',
  },
] as const

export function builtinViewOf(id: string): BuiltinGanttView | undefined {
  return BUILTIN_GANTT_VIEWS.find((view) => view.id === id)
}

export function isBuiltinGanttViewId(id: unknown): id is string {
  return typeof id === 'string' && builtinViewOf(id) !== undefined
}

/** 预设 → 既有 hide 语义：只隐藏可隐藏且不在 visible 集合内的列。 */
export function hideForBuiltinView(view: BuiltinGanttView): string[] {
  const visible = new Set(view.visible)
  return GANTT_HIDEABLE_KEYS.filter((key) => !visible.has(key))
}

/** 自定义视图表格默认宽（viewbackend-re §6：同一引擎切列集；宽度=可见列注册表
 *  宽度和钳在 ZPERT_TABLE_W 内，与内置预设同式）。 */
export function widthForHide(hide: string[]): number {
  const hidden = new Set(hide)
  const full = GANTT_COLS.filter((col) => !hidden.has(col.key)).reduce((sum, col) => sum + col.w, 0)
  return Math.min(ZPERT_TABLE_W, full)
}

/** 当前视图表格默认宽=可见列注册表宽度和，避免全列宽把画布挤没。 */
export function widthForBuiltinView(view: BuiltinGanttView): number {
  const visible = new Set(view.visible)
  const full = GANTT_COLS.filter((col) => visible.has(col.key)).reduce((sum, col) => sum + col.w, 0)
  // 右侧还有 238px 检查泊靠面；宽视图在 1440 窗口下也必须给画布留出可读空间。
  return Math.min(ZPERT_TABLE_W, full)
}
