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

/** 工期单位（v4.150 双工期口径刀1）：缺省 wd=工作日（现状口径，旧文件零迁移）；cd=日历天（自然日定时，对齐 mspdi DurationFormat 8）。duration 恒为「当前单位下的数值」 */
export type DurationUnit = 'wd' | 'cd'

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
  /** 任务固定成本（元，叶任务专属；v4.122 资源成本刀1。分组行禁止=汇总唯一口径为子孙求和） */
  fixedCost?: number
  /** 工期单位（v4.150 双工期刀1：缺省/空=wd 工作日；cd=日历天，仅限叶任务非里程碑，数值上限 3650；v4.155 起搭接四型全放开） */
  durationUnit?: DurationUnit
  /** 要求完成时间（斑马复刻 S5 预警：ask_stop_time 口径=上级下达的要求完成；工作日序号，0=开工日。缺省=未下达，预警「无级别」） */
  askFinish?: number
  /** 工作分类（斑马复刻批5：group 字段独立化；分组维度 category 优先取它，缺省回落 WBS 父名） */
  category?: string
  /** 单元格/横道外观（斑马复刻批5：文字颜色/填充/字体/字号；hex fail-closed，见 Go Validate） */
  style?: SchedTaskStyle
  /** 自定义字段值（v4.138 #14：槽位见 customFields.ts 注册表，text|number；只存有值槽，旧文件零迁移） */
  custom?: Partial<Record<string, string | number>>
}

/** 单元格/横道外观（斑马 draw 参数子集：文字颜色/横道填充/字体族/字号；颜色=6 位 hex） */
export interface SchedTaskStyle {
  /** 文字颜色（#RRGGBB） */
  color?: string
  /** 横道条填充色（#RRGGBB） */
  fill?: string
  /** 字体族名（≤40 字符） */
  font?: string
  /** 字号 px（8-72） */
  size?: number
}

/** 工作日历（对齐 Project 基准日历）：workweek 为 JS getDay 口径 0=周日..6=周六 */
export interface SchedCalendar {
  /** 视为工作日的星期集合 */
  workweek: number[]
  /** 节假日/停工例外（YYYY-MM-DD，命中即非工作日） */
  holidays: string[]
}

/** 资源类型（对齐 Project：工时/材料/成本三类） */
export type ResourceType = 'work' | 'material' | 'cost'

/** 资源（Project 资源工作表的工期制子集，v4.122.0 资源成本刀1） */
export interface SchedResource {
  id: string
  name: string
  type: ResourceType
  /** 材料计量单位（type=material 有意义：t/m³/…，展示与导出用） */
  unit?: string
  /** 标准费率（元）：work=元/工日；material=元/单位；cost 不用 */
  standardRate?: number
  /** 每次使用成本（元，每条分配计一次；work/material 可用） */
  costPerUse?: number
  /** 工时资源可用上限（默认 1；v1 仅承载与 mspdi 往返，不做平衡） */
  maxUnits?: number
  /** 资源级日历（v4.137 #13：个人周工作制+休假例外；缺省=跟随项目日历。影响使用视图可用性/超载口径，不改 CPM 排程） */
  calendar?: SchedCalendar
}

/** 分配（任务↔资源；(taskId,resourceId) 唯一，无独立 id） */
export interface SchedAssignment {
  /** 叶任务 id（分组行禁止分配，fail-closed） */
  taskId: string
  resourceId: string
  /** 工时资源投入强度（默认 1；成本=工期×units×费率） */
  units?: number
  /** 材料固定总量（type=material：总量×单价，不随时长变） */
  quantity?: number
  /** 成本资源金额（type=cost：该分配的固定金额，元） */
  amount?: number
}

/** AOA 事件手动布点（整数像素，与 AOA_COL_W/ROW_H 同一坐标系；v4.123 AOA 刀1） */
export interface AoaPin {
  x: number
  y: number
}

/** 双代号布局覆盖：pin 按事件锚点键匹配（混合锚定——命中手动位，未命中自动落位） */
export interface AoaLayout {
  pins: Record<string, AoaPin>
}

/**
 * 检视（多期进度快照）——复刻斑马「检视时间」（references: 斑马进度逆向报告 §5.1）。
 *
 * 语义：计划是连续推演的，执行是离散打点的。每次检视 = 一个数据日期 + 该期的
 * 实际回填。**多期并列**是本模型的关键：历史检视不可被覆盖，否则"前锋线对比"
 * 与"计划变动分析"都无从谈起。
 *
 * 本字段是**承载 + 透传**：CPM 不消费、视图暂不消费（v1 只保证能存能读）。
 * 时间一律用「工作日序号（相对开工日）」，与 CpmResult 同口径；`date` 只作展示
 * 与人工校核，两者必须一致（由 blocks.ts 的 normalizeReviews 兜容错，Go 侧
 * Validate fail-closed 兜落盘）。
 */
export interface SchedReviewActual {
  /** 实际开始（工作日序号；缺省=未回填） */
  start?: number
  /** 实际完成（工作日序号；缺省=未回填） */
  finish?: number
  /** 该期完成百分比 0-100（缺省=沿用任务 progress） */
  percent?: number
}

export interface SchedReview {
  /** 检视时间（工作日序号，相对开工日） */
  at: number
  /** 对应 ISO 日期（YYYY-MM-DD，展示与人工校核） */
  date: string
  /** 冻结：拉直前锋线后为 true，禁止回改已完成/进行中（斑马 frozen_status 口径） */
  frozen?: boolean
  /** 逐任务实际数据（键=叶任务 id；任务被删后仍保留，便于历史对比） */
  actuals: Record<string, SchedReviewActual>
  /** 备注（缺省空） */
  note?: string
}

/**
 * 双代号「点-箭」持久模型（复刻斑马 pert_prop{dots,begin_dot_id} + pert_links，见逆向报告 §3.1/§15.5）。
 *
 * 与 `aoa.ts` 的派生 `AoaGraph` 的区别：那是**每次渲染重算的视图结果**，这里的
 * `dots` 是**落盘的图元身份**。斑马的节点带递增 id 与序号、建关系时会自动补点/
 * 合并重复点；gaea 目前只靠 `anchor`（成员键）临时稳定，拓扑一改编号与 pins 漂移。
 *
 * 缺省（undefined）= 沿用现有派生口径，旧文件零迁移；一旦存在则视为权威图元表。
 */
export interface SchedAoaDot {
  /** 点身份（持久稳定，建议 "d1"/"d2" 或 "end:t1"|"start:t1" 业务键） */
  id: string
  /** 节点编号（1 起，拓扑序，保证箭线 i<j） */
  num: number
  /** 层/行号（布图参数化用，缺省=自动） */
  layer?: number
  /** 起点事件（唯一始） */
  isBegin?: boolean
  /** 终点事件（唯一终） */
  isEnd?: boolean
}

export interface SchedAoaArrow {
  id: string
  /** 尾事件 id（须存在于 dots） */
  from: string
  /** 头事件 id（须存在于 dots） */
  to: string
  /** task=实工作箭线；dummy=虚工作（虚线）；summary=分组汇总箭线 */
  kind: 'task' | 'dummy' | 'summary'
  /** kind=task/summary 时对应的任务 id */
  taskId?: string
  /** 该箭线承载的时距（工作日，缺省 0） */
  lag?: number
}

export interface SchedAoaGraph {
  dots: SchedAoaDot[]
  arrows: SchedAoaArrow[]
  /** 起点事件 id（缺省=按 isBegin 推断） */
  beginDotId?: string
}

/**
 * 图面参数（复刻斑马 `network` 状态块的 gaea 子集，见逆向报告 §4.1–§4.2）。
 *
 * 斑马把 60+ 个图面键与计划一起落盘（`pert_mode / is_stagger / dot_style /
 * arrow_style / name_pos / head_height / layer_height1-2 / layout_density …`）。
 * 这里只取支撑"5 种显示模式 + 布图压实"的最小集；缺省=全部走现有渲染口径。
 */
export type SchedRenderMode =
  | 'timescaled' // 时标网络图（x=ES，现行 AOA 口径）
  | 'timescaledLogic' // 时标逻辑图（有时间轴、不画进度装饰）
  | 'logic' // 逻辑网络图（按拓扑紧凑排布，不按时间）
  | 'pdm' // 单代号网络图
  | 'ganttNetwork' // 横道网络图（横道叠关系）

export interface SchedNetworkStyle {
  /** 显示模式（缺省 timescaled） */
  mode?: SchedRenderMode
  /** 节点形状（斑马 dot_style） */
  dotStyle?: 'circle' | 'rect'
  /** 箭头样式（斑马 arrow_style） */
  arrowStyle?: 'solid' | 'hollow'
  /** 名称排布（斑马 name_pos：自动/横向/纵向） */
  namePos?: 'auto' | 'horizontal' | 'vertical'
  /** 节点半径（px，缺省沿用 AOA_R） */
  nodeRadius?: number
  /** 箭头头部高度（px） */
  headHeight?: number
  /** 两层高度（斑马 layer_height1/2，px） */
  layerHeight1?: number
  layerHeight2?: number
  /** 错层排布（斑马 is_stagger：相邻事件纵向错开减少交叉） */
  stagger?: boolean
  /** 隐藏非重要关系线（斑马 is_hide_unimportant_link） */
  hideUnimportantLink?: boolean
  /** 时标单位（斑马 年-月 / 月-天 / 日-时） */
  tickUnit?: 'yearMonth' | 'monthDay' | 'dayHour'
  /** 刻度线疏密（斑马 不画/画小/画大刻度线） */
  tickLines?: 'none' | 'small' | 'large'
  /** 布图紧凑度（斑马 layout_density） */
  density?: 'compact' | 'normal' | 'loose'
  /** 纵横比上限（斑马 aspect_ratio，布图压实用） */
  aspectMax?: number
  /** 页数上限（斑马按纸张布图的页数窗，默认 200） */
  maxPages?: number
}

/**
 * 视图快照（复刻斑马 views_list，见逆向报告 §6.1）。
 *
 * 斑马的「视图」不是页面，而是**列集 + 分组 + 筛选 + 排序 + 显示模式的命名快照**
 * （内置 10 个 + 自定义 + 云视图）。gaea 现在只有 `ScheduleView` 七个页面的切换，
 * 没有"视图"这一等对象；本结构是它的承载位（v1 只存不用）。
 */
export interface SchedViewSort {
  field: string
  dir: 'asc' | 'desc'
}

export interface SchedView {
  id: string
  name: string
  /** 页面级类型（沿用 store 的 ScheduleView 枚举） */
  kind: 'gantt' | 'pdm' | 'aoa' | 'usage' | 'dcma' | 'monte' | 'evm'
  /** 可见列键集（缺省=全部；键=GanttCol.key） */
  cols?: string[]
  /** 隐藏列键集（兼容现有 hide 语义） */
  hide?: string[]
  /** 多级分组字段（键=ganttGroup.ts 的分组维度） */
  groupBy?: string[]
  /** 筛选表达式（缺省=不过滤；语法由 ganttFilter 后续定义） */
  filter?: string
  /** 排序 */
  sort?: SchedViewSort
  /** 网络图显示模式（kind=aoa/pdm 时有效） */
  renderMode?: SchedRenderMode
}

export interface SchedProject {
  name: string
  /** 开工日期（YYYY-MM-DD，工作日历推算锚点） */
  startDate: string
  tasks: SchedTask[]
  links: SchedLink[]
  /** 工作日历（缺省=周一~周五） */
  calendar?: SchedCalendar
  /** 基线（v4.116 刀7：保存时的排程快照，缺省=尚未保存；=baselines 中的**活跃**槽，Go 镜像/AI 工具口径不变） */
  baseline?: SchedBaseline | null
  /** 基线槽位列表（v4.137 #11 多基线：FIFO 上限 3，支持签证 1→N 场景切换对比；缺省=单基线旧文件零迁移） */
  baselines?: SchedBaseline[]
  /** 自定义字段列名覆盖（v4.138 #14：key→显示名；缺省用注册表 label。xlsx 导入导出同口径） */
  customLabels?: Record<string, string>
  /** 目标竣工日期（v4.117 刀8：YYYY-MM-DD，倒排校核用，缺省=未设） */
  deadline?: string | null
  /** 资源表（v4.122 资源成本刀1：缺省=无资源维度，旧文件零迁移可读） */
  resources?: SchedResource[]
  /** 任务↔资源分配（(taskId,resourceId) 唯一） */
  assignments?: SchedAssignment[]
  /** 双代号手动布局 pins（v4.123 AOA 刀1：缺省=自动布局） */
  aoaLayout?: AoaLayout
  /**
   * 检视序列（多期进度快照；缺省=旧文件零迁移）。
   * 见 SchedReview 注释：v1 只承载与透传，CPM 与视图暂不消费。
   */
  reviews?: SchedReview[]
  /** 双代号点-箭持久图元（缺省=沿用 aoa.ts 派生口径） */
  aoa?: SchedAoaGraph
  /** 图面参数（缺省=沿用现有渲染口径） */
  networkStyle?: SchedNetworkStyle
  /** 视图快照列表（缺省=无自定义视图） */
  views?: SchedView[]
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
