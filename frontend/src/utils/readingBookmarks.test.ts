import { afterEach, describe, expect, it } from 'vitest'
import { readBookmarks, writeBookmarks, type ReadingBookmark } from './readingBookmarks'

const KEY = 'gaea.novel.readingBookmarks.C:/books/test'

const b1: ReadingBookmark = { nodeId: 'n1', title: '第1章', scrollTop: 120, pct: 8, text: '第一章开头…', createdAt: 1 }
const b2: ReadingBookmark = { nodeId: 'n2', title: '第2章', scrollTop: 300, pct: 20, text: '第二章内容…', createdAt: 2 }

describe('readingBookmarks', () => {
  afterEach(() => {
    try { localStorage.removeItem(KEY) } catch { /* ignore */ }
  })

  it('round-trips 书签列表', () => {
    writeBookmarks('C:/books/test', [b1, b2])
    expect(readBookmarks('C:/books/test')).toEqual([b1, b2])
  })

  it('缺失/损坏数据返回空列表', () => {
    expect(readBookmarks('C:/books/none')).toEqual([])
    localStorage.setItem(KEY, 'not-json')
    expect(readBookmarks('C:/books/test')).toEqual([])
    localStorage.setItem(KEY, JSON.stringify([{ foo: 1 }]))
    expect(readBookmarks('C:/books/test')).toEqual([])
  })

  it('空项目路径不读写', () => {
    writeBookmarks('', [b1])
    expect(readBookmarks('')).toEqual([])
  })
})
