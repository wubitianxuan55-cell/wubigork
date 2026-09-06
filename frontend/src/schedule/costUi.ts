/**
 * schedule/costUi.ts — 资源成本 UI 共享口径（v4.124 刀3）
 *
 * 甘特成本列与状态栏总成本段共用的显隐判据与金额格式化。
 * 独立成纯模块：引擎口径在 cost.ts（勿混），此处只放视图层约定。
 */
import type { SchedProject } from './types'

/**
 * 计划是否含有任何资源/成本数据（甘特成本列与状态栏总成本段的同一显隐口径）：
 * resources/assignments 任一非空，或存在非零 fixedCost（0 元与未设同为「无成本数据」）。
 * 无数据时成本列留空、状态栏不出「总成本」段——诚实呈现，不留 ¥0 假象。
 */
export function hasCostData(p: SchedProject): boolean {
  return (p.resources?.length ?? 0) > 0
    || (p.assignments?.length ?? 0) > 0
    || p.tasks.some((t) => (t.fixedCost ?? 0) > 0)
}

/** 金额文本：最多 2 位小数（引擎已四舍五入到分），整数不带小数点 */
export function fmtCost(n: number): string {
  return String(Math.round(n * 100) / 100)
}
