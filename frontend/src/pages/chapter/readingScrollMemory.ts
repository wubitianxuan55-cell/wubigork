/**
 * readingScrollMemory — 阅读位置记忆的纯工具族（自 ChapterPage 原样搬移）：
 * 每章滚动位置按 key（固定前缀 + nodeId）存 localStorage，下次进入该章自动恢复；
 * 读写均吞异常（隐私模式 / 配额满等场景静默降级为无记忆）。
 * 另含滚动进度百分比计算（进度条与书签 pct 共用，原先两处各写一份公式）。
 */

const READ_SCROLL_PREFIX = 'gaea.novel.reading.scroll.'

/** 某章滚动位置的 localStorage 键 */
export function readScrollKey(nodeId: string): string {
  return READ_SCROLL_PREFIX + nodeId
}

/** 读某章保存的滚动位置；无记录 / 存储异常时返回 0 */
export function readSavedScrollTop(nodeId: string): number {
  try { return Number(localStorage.getItem(readScrollKey(nodeId)) || 0) } catch { return 0 }
}

/** 保存某章滚动位置（静默失败，不打断滚动流程） */
export function saveScrollTop(nodeId: string, scrollTop: number): void {
  try { localStorage.setItem(readScrollKey(nodeId), String(scrollTop)) } catch { /* ignore */ }
}

/**
 * 滚动进度百分比（0-100）：容器无溢出时为 0，向上取整并钳到 100。
 * 原实现进度条处多一个 Math.min(100, …)、书签 pct 处没有——scrollTop 恒 ≤ max，
 * 两者数值本就一致，此处统一为带钳制的版本，调用点行为不变。
 */
export function scrollPct(el: HTMLElement): number {
  const max = el.scrollHeight - el.clientHeight
  return max > 0 ? Math.min(100, Math.round((el.scrollTop / max) * 100)) : 0
}
