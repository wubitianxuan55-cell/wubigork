import { describe, expect, it, beforeEach, vi } from 'vitest'

import { clearSearchHistory, loadSearchHistory, pushSearchHistory } from './bookSearchHistory'

beforeEach(() => {
  localStorage.clear()
})

describe('bookSearchHistory（t3 搜索历史）', () => {
  it('空/损坏数据当空数组', () => {
    expect(loadSearchHistory()).toEqual([])
    localStorage.setItem('gaea.booksearch.history', '{bad json')
    expect(loadSearchHistory()).toEqual([])
    localStorage.setItem('gaea.booksearch.history', JSON.stringify([1, null, 'ok', '  ']))
    expect(loadSearchHistory()).toEqual(['ok'])
  })

  it('push 去重最近在前并持久化', () => {
    pushSearchHistory(' 诡秘之主 ')
    pushSearchHistory('诡秘之主') // trim 后去重，提前
    pushSearchHistory('大道朝天')
    expect(loadSearchHistory()).toEqual(['大道朝天', '诡秘之主'])
    expect(loadSearchHistory()).toEqual(loadSearchHistory()) // 读稳定
  })

  it('封顶默认 8 条', () => {
    for (const kw of ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i']) pushSearchHistory(kw)
    const list = loadSearchHistory()
    expect(list.length).toBe(8)
    expect(list[0]).toBe('i')
    expect(list).not.toContain('a')
  })

  it('空关键字不记录', () => {
    pushSearchHistory('   ')
    expect(loadSearchHistory()).toEqual([])
  })

  it('clear 清空', () => {
    pushSearchHistory('x')
    clearSearchHistory()
    expect(loadSearchHistory()).toEqual([])
  })

  it('setItem 抛错不阻塞（存储禁用兜底）', () => {
    const spy = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('quota')
    })
    expect(() => pushSearchHistory('y')).not.toThrow()
    spy.mockRestore()
  })
})
