import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  clearRecentObjectsForTest,
  loadRecentObjects,
  recordRecentObject,
  subscribeRecentObjects,
} from './recentObjects'

// recentObjects 单源（首页「继续」二期）：记账幂等/每 kind 上限/广播/损坏兜底。
describe('recentObjects 单源', () => {
  beforeEach(() => {
    clearRecentObjectsForTest()
  })
  afterEach(() => {
    clearRecentObjectsForTest()
    vi.restoreAllMocks()
  })

  it('记账去重置顶（同 kind+key 幂等，标题以后写为准）', () => {
    recordRecentObject('novel', 'C:/books/长夜', '长夜')
    recordRecentObject('schedule', 'plans/百锦路.gsp', '百锦路')
    recordRecentObject('novel', 'C:/books/长夜', '长夜（修订版）')
    const list = loadRecentObjects()
    expect(list).toHaveLength(2)
    expect(list[0]).toMatchObject({ kind: 'novel', key: 'C:/books/长夜', title: '长夜（修订版）' })
    expect(list[1]).toMatchObject({ kind: 'schedule', key: 'plans/百锦路.gsp' })
  })

  it('每 kind 只留最近 3 条（防单一板块刷屏）', () => {
    for (let i = 1; i <= 5; i++) recordRecentObject('schedule', `plans/p${i}.gsp`, `p${i}`)
    recordRecentObject('novel', 'C:/books/a', 'a')
    const list = loadRecentObjects()
    expect(list.filter((o) => o.kind === 'schedule')).toHaveLength(3)
    // 最近 3 条置顶保留（p5/p4/p3），p1/p2 被挤出
    expect(list.map((o) => o.key)).toEqual(['C:/books/a', 'plans/p5.gsp', 'plans/p4.gsp', 'plans/p3.gsp'])
  })

  it('写入即广播（同窗口订阅者收到通知）', () => {
    const cb = vi.fn()
    const off = subscribeRecentObjects(cb)
    recordRecentObject('novel', 'C:/books/x', 'x')
    expect(cb).toHaveBeenCalledTimes(1)
    off()
    recordRecentObject('novel', 'C:/books/y', 'y')
    expect(cb).toHaveBeenCalledTimes(1)
  })

  it('空 key 不记账；损坏存储兜底为空数组', () => {
    recordRecentObject('novel', '', '空')
    expect(loadRecentObjects()).toHaveLength(0)
    clearRecentObjectsForTest()
    localStorage.setItem('gaea.atRecentObjects', '{oops')
    expect(loadRecentObjects()).toEqual([])
  })
})
