/**
 * bookSearchHistory — 在线搜书关键字历史（t3 余项；localStorage 本地数据，
 * 会话间保留）。最近在前、去重、封顶（默认 8 条）。
 */

const KEY = 'gaea.booksearch.history'

export function loadSearchHistory(): string[] {
  try {
    const raw = localStorage.getItem(KEY)
    const arr = raw ? (JSON.parse(raw) as unknown) : []
    if (!Array.isArray(arr)) return []
    return arr.filter((x): x is string => typeof x === 'string' && x.trim() !== '')
  } catch {
    return [] // 损坏当空，不抛错
  }
}

/** 记录一次搜索：trim、非空去重（最近在前）、封顶。返回新列表。 */
export function pushSearchHistory(keyword: string, cap = 8): string[] {
  const kw = keyword.trim()
  if (!kw) return loadSearchHistory()
  const next = [kw, ...loadSearchHistory().filter((x) => x !== kw)].slice(0, cap)
  try {
    localStorage.setItem(KEY, JSON.stringify(next))
  } catch {
    // 存储满/禁用：历史尽力而为，不阻塞搜索
  }
  return next
}

export function clearSearchHistory(): void {
  try {
    localStorage.removeItem(KEY)
  } catch {
    // 同上
  }
}
