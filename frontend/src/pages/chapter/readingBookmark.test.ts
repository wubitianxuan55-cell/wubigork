// readingBookmark.test.ts — 阅读页书签列表纯操作族测试：
// 锁 48px 容差去重（toggle 依据）、追加按 scrollTop 升序、移除按 nodeId+scrollTop 身份，
// 以及纯函数不修改入参列表。
import { describe, expect, it } from 'vitest'
import {
  BOOKMARK_NEAR_TOLERANCE, findBookmarkNear, removeBookmarkInList, toggleBookmarkInList,
} from './readingBookmark'
import type { ReadingBookmark } from '../../utils/readingBookmarks'

const bm = (nodeId: string, scrollTop: number): ReadingBookmark => ({
  nodeId, title: `章节${scrollTop}`, scrollTop, pct: 0, text: '摘录', createdAt: scrollTop,
})

describe('findBookmarkNear', () => {
  it('同章内滚动位置相差小于容差的书签被找到', () => {
    const list = [bm('ch-1', 100), bm('ch-1', 300)]
    expect(findBookmarkNear(list, 'ch-1', 140)?.scrollTop).toBe(100)
  })

  it('容差边界：相差 47 视为同一处，相差 48 不算（< 严格小于）', () => {
    const list = [bm('ch-1', 100)]
    expect(BOOKMARK_NEAR_TOLERANCE).toBe(48)
    expect(findBookmarkNear(list, 'ch-1', 147)).toBeDefined()
    expect(findBookmarkNear(list, 'ch-1', 148)).toBeUndefined()
  })

  it('异章书签即使位置相近也不匹配', () => {
    const list = [bm('ch-2', 100)]
    expect(findBookmarkNear(list, 'ch-1', 100)).toBeUndefined()
  })

  it('支持自定义容差：默认容差外命中不了，放宽后命中', () => {
    const list = [bm('ch-1', 100)]
    expect(findBookmarkNear(list, 'ch-1', 160, 48)).toBeUndefined()
    expect(findBookmarkNear(list, 'ch-1', 160, 100)).toBeDefined()
  })
})

describe('toggleBookmarkInList', () => {
  it('无相近书签：追加快照并按 scrollTop 升序排列，原列表不被修改', () => {
    const list = [bm('ch-1', 300), bm('ch-1', 100)]
    const next = toggleBookmarkInList(list, bm('ch-1', 200))
    expect(next.map((b) => b.scrollTop)).toEqual([100, 200, 300])
    // 入参保持原序（纯函数）
    expect(list.map((b) => b.scrollTop)).toEqual([300, 100])
  })

  it('相近书签已存在：按引用摘除该条，其余保留', () => {
    const near = bm('ch-1', 100)
    const other = bm('ch-1', 300)
    const next = toggleBookmarkInList([near, other], bm('ch-1', 120))
    expect(next).toEqual([other])
  })

  it('跨章快照不与本章相近书签互斥（append 语义按 nodeId 区分）', () => {
    const list = [bm('ch-1', 100)]
    const next = toggleBookmarkInList(list, bm('ch-2', 100))
    expect(next.length).toBe(2)
  })
})

describe('removeBookmarkInList', () => {
  it('按 nodeId + scrollTop 精确删除', () => {
    const list = [bm('ch-1', 100), bm('ch-1', 300)]
    expect(removeBookmarkInList(list, bm('ch-1', 100))).toEqual([bm('ch-1', 300)])
  })

  it('同位置异章的书签不受误删', () => {
    const list = [bm('ch-2', 100)]
    expect(removeBookmarkInList(list, bm('ch-1', 100))).toEqual(list)
  })

  it('删除不存在的书签返回等价列表', () => {
    const list = [bm('ch-1', 100)]
    expect(removeBookmarkInList(list, bm('ch-1', 999)).length).toBe(1)
  })
})
