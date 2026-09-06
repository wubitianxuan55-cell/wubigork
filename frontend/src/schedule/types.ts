/**
 * schedule/types.ts — 工程进度计划数据模型（v4.110.0 刀1 / v4.111.0 刀2）
 *
 * 一套数据模型驱动三种视图（横道图 / 单代号网络图 / 双代号网络图），
 * 对齐斑马进度「一表双图」口径：表格做计划，多视图同步生成、自由切换。
 * 日历口径：工期/时距均为**工作日**整数；日期显示由 startDate + 工作日历推导
 * （刀2 起对齐 Project/斑马工作制：默认周一~五，节假日例外）。
 */

/** 搭接关系类型：完成-开始 / 开始-开始 / 完成-完成 / 开始-完成 */
export type LinkType = 'FS' | 'SS' | 'FF' | 'SF'

/** 搭接关系（from 为前置任务，to 为后续任务） */
export interface SchedLink {
  from: string
  to: string
  type: LinkType
  /** 时距（工作日，整数，可为负） */
  lag: number
}

/** 任务排程模式（对齐 Project）：自动=CPM 排程；手动=锁定开始不动 */
export type TaskMode = 'auto' | 'manual'

/** 任务/分组行（level 缩进表达 WBS 层级，扁平数组按顺序渲染） */
export interface SchedTask {
  id: string
  name: string
  /** 工期（工作日；分组行固定 0，汇总条由子项滚动推导） */
  duration: number
  /** 大纲层级：0=分组，1=子任务（两级，足够工程口径） */
  level: number
  /** 完成进度 0-100 */
  progress: number
  /** 里程碑（工期视为 0，菱形显示） */
  isMilestone?: boolean
  /** 排程模式（缺省 auto=自动）；手动任务锁定开始、不参与关键线路 */
  mode?: TaskMode
  /** 手动任务锁定开始（工作日序号，0=开工日） */
  manualStart?: number
}

/** 工作日历（对齐 Project 基准日历）：workweek 为 JS getDay 口径 0=周日..6=周六 */
export interface SchedCalendar {
  /** 视为工作日的星期集合 */
  workweek: number[]
  /** 节假日/停工例外（YYYY-MM-DD，命中即非工作日） */
  holidays: string[]
}

export interface SchedProject {
  name: string
  /** 开工日期（YYYY-MM-DD，工作日历推算锚点） */
  startDate: string
  tasks: SchedTask[]
  links: SchedLink[]
  /** 工作日历（缺省=周一~周五） */
  calendar?: SchedCalendar
  /** 基线（v4.116 刀7：保存时的排程快照，缺省=尚未保存） */
  baseline?: SchedBaseline | null
  /** 目标竣工日期（v4.117 刀8：YYYY-MM-DD，倒排校核用，缺省=未设） */
  deadline?: string | null
}

/** 基线行快照：单任务保存基线时的排程结果（叶任务专属，分组行不入基线） */
export interface SchedBaselineRow {
  /** 任务名（任务被移除后仍可读） */
  name: string
  /** 基线最早开始（工作日序号） */
  es: number
  /** 基线最早完成（工作日序号） */
  ef: number
  /** 有效工期（里程碑 0） */
  dur: number
  /** 保存时是否关键 */
  critical: boolean
}

/** 基线（对齐 Project「设置基线」：快照排程结果，供漂移对比） */
export interface SchedBaseline {
  /** 基线名（缺省「基线」） */
  name: string
  /** 保存时间（YYYY-MM-DD HH:mm） */
  savedAt: string
  /** 保存时总工期（工作日） */
  duration: number
  /** 叶任务基线行 */
  rows: Record<string, SchedBaselineRow>
}

/** 基线漂移行：只含有偏差的任务（shifted=推移 / added=新增 / removed=已移除） */
export interface BaselineDriftRow {
  id: string
  name: string
  kind: 'shifted' | 'added' | 'removed'
  /** 基线行（新增任务为 null） */
  base: SchedBaselineRow | null
  /** 当前排程（已移除任务为 null） */
  now: { es: number; ef: number; dur: number } | null
  esDrift: number
  efDrift: number
  durDrift: number
  criticalNow: boolean
  criticalBase: boolean
}

/** 基线漂移对比汇总 */
export interface BaselineDrift {
  baselineName: string
  baselineSavedAt: string
  baselineDuration: number
  currentDuration: number
  /** 总工期漂移（正=拖后，负=提前） */
  durationDrift: number
  sameCount: number
  shiftedCount: number
  addedCount: number
  removedCount: number
  /** 新进入关键线路的任务名 */
  criticalGained: string[]
  /** 退出关键线路的任务名 */
  criticalLost: string[]
  /** 有偏差的任务行（一致行不进列表） */
  rows: BaselineDriftRow[]
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
