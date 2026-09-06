/**
 * schedule/types.ts — 工程进度计划数据模型（v4.110.0 刀1）
 *
 * 一套数据模型驱动三种视图（横道图 / 单代号网络图 / 双代号网络图），
 * 对齐斑马进度「一表双图」口径：表格做计划，多视图同步生成、自由切换。
 * 日历口径：工期/时距均为自然日整数（刀1）；日期显示由 startDate 锚点推导。
 */

/** 搭接关系类型：完成-开始 / 开始-开始 / 完成-完成 / 开始-完成 */
export type LinkType = 'FS' | 'SS' | 'FF' | 'SF'

/** 搭接关系（from 为前置任务，to 为后续任务） */
export interface SchedLink {
  from: string
  to: string
  type: LinkType
  /** 时距（天，整数，可为负） */
  lag: number
}

/** 任务/分组行（level 缩进表达 WBS 层级，扁平数组按顺序渲染） */
export interface SchedTask {
  id: string
  name: string
  /** 工期（天；分组行固定 0，汇总条由子项滚动推导） */
  duration: number
  /** 大纲层级：0=分组，1=子任务（刀1 两级，足够工程口径） */
  level: number
  /** 完成进度 0-100 */
  progress: number
  /** 里程碑（工期视为 0，菱形显示） */
  isMilestone?: boolean
}

export interface SchedProject {
  name: string
  /** 开工日期（YYYY-MM-DD，横道图日期轴锚点） */
  startDate: string
  tasks: SchedTask[]
  links: SchedLink[]
}

/** 单任务 CPM 计算结果（天数，ES/EF 为相对开工日的天偏移） */
export interface TaskCpm {
  es: number
  ef: number
  ls: number
  lf: number
  /** 总时差 */
  tf: number
  /** 自由时差 */
  ff: number
  /** 关键工作（tf===0） */
  critical: boolean
}

export interface CpmResult {
  ok: boolean
  /** 循环依赖等错误信息（ok=false 时有值） */
  error?: string
  /** 构成循环的任务名（UI 提示用） */
  cycle?: string[]
  rows: Record<string, TaskCpm>
  /** 总工期（天）= max(EF) */
  duration: number
}
