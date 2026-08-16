/**
 * 阅读书签（按项目持久化，条目归属章节）
 *  - nodeId: 章节大纲节点 id
 *  - scrollTop: 章节内滚动偏移（恢复时定位）
 *  - pct: 章内阅读百分比（列表展示）
 *  - text: 书签处段落摘录（列表预览）
 */
export interface ReadingBookmark {
  nodeId: string
  title: string
  scrollTop: number
  pct: number
  text: string
  createdAt: number
}

const BOOKMARKS_KEY = 'gaea.novel.readingBookmarks.'

export function readBookmarks(projectPath: string): ReadingBookmark[] {
  try {
    if (!projectPath) return []
    const raw = localStorage.getItem(BOOKMARKS_KEY + projectPath)
    if (!raw) return []
    const list = JSON.parse(raw) as ReadingBookmark[]
    if (!Array.isArray(list)) return []
    return list.filter(
      (b) => b && typeof b.nodeId === 'string' && typeof b.scrollTop === 'number',
    )
  } catch {
    return []
  }
}

export function writeBookmarks(projectPath: string, bookmarks: ReadingBookmark[]) {
  try {
    if (!projectPath) return
    localStorage.setItem(BOOKMARKS_KEY + projectPath, JSON.stringify(bookmarks))
  } catch { /* ignore */ }
}
