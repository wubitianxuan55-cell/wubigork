import { afterEach, describe, expect, it } from 'vitest'
import { readReadingProgress, writeReadingProgress } from './readingProgress'

const KEY = 'gaea.novel.reading.C:/books/test'

describe('readingProgress', () => {
  afterEach(() => {
    try { localStorage.removeItem(KEY) } catch { /* ignore */ }
  })

  it('round-trips progress', () => {
    writeReadingProgress('C:/books/test', { nodeId: 'n1', chapterNum: 3, title: '第3章' })
    expect(readReadingProgress('C:/books/test')).toEqual({
      nodeId: 'n1',
      chapterNum: 3,
      title: '第3章',
    })
  })

  it('returns null for missing or invalid data', () => {
    expect(readReadingProgress('C:/books/missing')).toBeNull()
    localStorage.setItem(KEY, 'not-json')
    expect(readReadingProgress('C:/books/test')).toBeNull()
  })
})
