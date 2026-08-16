import { afterEach, describe, expect, it } from 'vitest'
import {
  ANNOTATION_COLORS, readAnnotations, writeAnnotations,
  type ReadingAnnotation,
} from './readingAnnotations'

const KEY = 'gaea.novel.readingAnnotations.C:/books/test'

const a1: ReadingAnnotation = {
  id: 'ann_1', nodeId: 'n1', title: '第1章', color: 'yellow',
  text: '剑修不问红尘。', note: '名场面', createdAt: 1,
}
const a2: ReadingAnnotation = {
  id: 'ann_2', nodeId: 'n2', title: '第2章', color: 'blue',
  text: '山雨欲来', note: '', createdAt: 2,
}

describe('readingAnnotations', () => {
  afterEach(() => {
    try { localStorage.removeItem(KEY) } catch { /* ignore */ }
  })

  it('round-trips 批注列表', () => {
    writeAnnotations('C:/books/test', [a1, a2])
    expect(readAnnotations('C:/books/test')).toEqual([a1, a2])
  })

  it('颜色映射覆盖四种高亮色', () => {
    expect(Object.keys(ANNOTATION_COLORS)).toEqual(['yellow', 'green', 'blue', 'pink'])
  })

  it('缺失/损坏/空摘录数据被过滤', () => {
    expect(readAnnotations('C:/books/none')).toEqual([])
    localStorage.setItem(KEY, 'not-json')
    expect(readAnnotations('C:/books/test')).toEqual([])
    localStorage.setItem(KEY, JSON.stringify([{ id: 'x', nodeId: 'n1', text: '' }]))
    expect(readAnnotations('C:/books/test')).toEqual([])
  })
})
