// pendingSessionResume.ts — 会话级回源通道（7.3-1 欠账收口，规格
// 进度计划/gaea-stage7-audit-resume-2026-09-17.md §1）：
// 任务收件箱「源会话」精确回源（Session 字段落库=会话路径），但 resume
// seam（resumeRecentSession）活在 gaea App 树内，收件箱在首页/各板拿不到。
// gaea 板块 keepAlive=true → 两态通道：
//   已挂载态：requestSessionResume 派发 window CustomEvent，App 监听直达；
//   冷启态：事件已丢，App 挂载 effect consume pending 兜底。
// 模块级单例（不进 React 状态）——跨树、跨挂载周期稳定。

export const RESUME_SESSION_EVENT = 'gaea:resume-session'

let pending: string | null = null

/** 发起回源：记 pending + 派发事件（两态通道同调；调用方自行导航到 gaea 板块）。 */
export function requestSessionResume(path: string): void {
  if (!path) return
  pending = path
  try {
    window.dispatchEvent(new CustomEvent(RESUME_SESSION_EVENT, { detail: { path } }))
  } catch {
    // 非 DOM 环境（测试桩）静默——pending 仍在
  }
}

/** 取走 pending（一次性消费；无则 null）。 */
export function consumePendingSessionResume(): string | null {
  const p = pending
  pending = null
  return p
}

/** 清空（测试/取消用）。 */
export function clearPendingSessionResume(): void {
  pending = null
}
