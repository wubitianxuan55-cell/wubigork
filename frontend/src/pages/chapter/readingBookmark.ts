/**
 * readingBookmark — 阅读页书签列表的纯操作族（自 ChapterPage 原样搬移）：
 * - findBookmarkNear：找同章内滚动位置相近的已有书签（toggle 去重判定）；
 * - toggleBookmarkInList：相近书签已存在则移除之，否则追加快照并按位置排序；
 * - removeBookmarkInList：按 nodeId + scrollTop 精确删除列表项。
 * 持久化（writeBookmarks）与 DOM 读数（scrollTop/pct/text 摘录）仍由组件接线完成，
 * 本模块只做列表计算，方便对去重/排序规则直接做单元断言。
 */
import type { ReadingBookmark } from '../../utils/readingBookmarks'

/** 同位置去重容差（px）：滚动位置相差小于该值视为同一书签 */
export const BOOKMARK_NEAR_TOLERANCE = 48

/** 新增书签所需的章节位置快照（各字段由调用点现算后整体传入） */
export type BookmarkSnapshot = ReadingBookmark

/** 找同章内与 scrollTop 相差小于容差的已有书签 */
export function findBookmarkNear(
  list: ReadingBookmark[],
  nodeId: string,
  scrollTop: number,
  tolerance: number = BOOKMARK_NEAR_TOLERANCE,
): ReadingBookmark | undefined {
  return list.find((b) => b.nodeId === nodeId && Math.abs(b.scrollTop - scrollTop) < tolerance)
}

/** toggle：相近书签已存在 → 按引用摘除；否则追加快照并按 scrollTop 升序排好 */
export function toggleBookmarkInList(list: ReadingBookmark[], snap: BookmarkSnapshot): ReadingBookmark[] {
  const hit = findBookmarkNear(list, snap.nodeId, snap.scrollTop)
  if (hit) return list.filter((b) => b !== hit)
  return [...list, snap].sort((a, b) => a.scrollTop - b.scrollTop)
}

/** 按 nodeId + scrollTop 精确删除一条书签（书签列表项无独立 id，二者合成身份） */
export function removeBookmarkInList(list: ReadingBookmark[], b: ReadingBookmark): ReadingBookmark[] {
  return list.filter((x) => x.nodeId !== b.nodeId || x.scrollTop !== b.scrollTop)
}
