/**
 * usageTypes.ts — 资源使用视图数据契约（v4.137 #12，类型与实现分离）
 *
 * 单独成文件的原因：computeUsage 实现（usage.ts）与 UsageView 组件并行开发，
 * 两边只共享这份不可变契约——类型先行钉死口径，实现与渲染互不阻塞。
 *
 * 口径（蒸馏 ProjectLibre E5+U6，适配本项目双工作制语义）：
 *  - 时间轴=项目工作日序号 wd（0..duration-1），与 CPM 的 es/ef 同一口径；
 *  - 负载 workload(资源, wd)=Σ 该资源在 wd 当天**在工**任务的 units（缺省 1；
 *    在工=es ≤ wd < ef，叶任务，auto/manual 均按 CPM 排程结果）；
 *  - 可用性 availability(资源, wd)=maxUnits(缺省 1)；若资源设置了个人日历
 *    （v4.137 #13）且 wd 对应的自然日不是其工作日 → 0（休假/停工不可用）；
 *  - 超载 overloaded = workload > availability + 1e-9；
 *  - 只有 work（工时）型资源进 rows——材料按总量计价、成本按金额计价，
 *    无「按天投入」语义，不参与按天负载/超载（诚实呈现，不做假分布）。
 */
import type { ResourceType } from './types'

/** 单资源单工作日格子 */
export interface UsageDayCell {
  /** 项目工作日序号（与 TaskCpm.es/ef 同口径） */
  wd: number
  /** 当天负载 = Σ 在工任务 units */
  workload: number
  /** 当天可用上限（资源个人日历休假=0） */
  availability: number
  /** workload > availability */
  overloaded: boolean
}

/** 在工任务（供视图 tooltip：这天是谁把负载顶起来的） */
export interface UsageActiveTask {
  taskId: string
  name: string
  /** 在工区间（工作日序号，含头不含尾） */
  es: number
  ef: number
  /** 每日投入强度（缺省 1） */
  units: number
}

/** 单资源使用行 */
export interface UsageResourceRow {
  resourceId: string
  name: string
  type: ResourceType
  /** 该资源被分配的在工任务清单（按 es 排序） */
  tasks: UsageActiveTask[]
  /** 逐工作日格子，长度=days */
  cells: UsageDayCell[]
  /** 峰值负载（max workload） */
  peakWorkload: number
  /** 超载工作日数 */
  overloadDays: number
  /** 累计投入（Σ workload，工日） */
  totalWorkload: number
}

/** 使用视图计算结果 */
export interface UsageResult {
  ok: boolean
  /** 循环依赖等错误（ok=false 时有值） */
  error?: string
  /** 时间轴长度（=cpm.duration，工作日数） */
  days: number
  /** 工时资源行（按名称排序，无分配的工时资源也保留=全零行） */
  rows: UsageResourceRow[]
  /** 全项目超载总数（Σ rows.overloadDays，状态栏/汇总用） */
  totalOverloadDays: number
}
