/**
 * queue.ts — 绘梦本地生成队列取消语义（纯逻辑，可单测）
 *
 * T6-4.1 取消真实生效后前端适配：
 * - 收到取消（点击取消按钮起）后，本地队列停止继续提交：
 *   已有生成中的项立即标记取消，排队中的项不再启动。
 * - 本地取消标记一直保持，直到用户手动发起新一轮生成（enqueueTask 清除）。
 */

import type { QueueEntry, QueueStatus } from './types'

/** 取消队列：运行中与排队中的条目全部标记为 canceled（已有生成中的项立即标记取消） */
export function markQueueCanceled(entries: QueueEntry[]): QueueEntry[] {
  return entries.map((e) =>
    e.status === 'pending' || e.status === 'running'
      ? { ...e, status: 'canceled' as const }
      : e,
  )
}

/** 队列是否应继续提交下一条：有待执行项，且未被本地取消标记阻断 */
export function shouldSubmitNext(pendingCount: number, cancelArmed: boolean): boolean {
  return pendingCount > 0 && !cancelArmed
}

/** 取消标记生效时条目归为 canceled，否则按执行结果（done/failed） */
export function afterTaskStatus(cancelArmed: boolean, ok: boolean): QueueStatus {
  if (cancelArmed) return 'canceled'
  return ok ? 'done' : 'failed'
}
