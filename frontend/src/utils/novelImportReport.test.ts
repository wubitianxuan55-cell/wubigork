import { describe, expect, it } from 'vitest'

import {
  formatImportSuccess, formatImportWarnings, SPLIT_STRATEGY_LABELS,
  type ImportReportLike,
} from './novelImportReport'

const base: ImportReportLike = {
  path: 'C:/novels/风雪夜归',
  title: '风雪夜归',
  chapter_count: 12,
  total_words: 34567,
}

describe('formatImportSuccess', () => {
  it('编码 + 策略拼进括号，策略映射中文标签', () => {
    expect(formatImportSuccess({ ...base, encoding: 'gb18030', split_strategy: 'strong' }))
      .toBe('已导入「风雪夜归」：12 章，34,567 字（gb18030 · 标题分章）')
  })

  it('booksource 映射为在线书源（书源线 t2），编码留空则只显策略', () => {
    expect(SPLIT_STRATEGY_LABELS.booksource).toBe('在线书源')
    expect(formatImportSuccess({ ...base, split_strategy: 'booksource' }))
      .toBe('已导入「风雪夜归」：12 章，34,567 字（在线书源）')
  })

  it('无编码无策略时不带括号；未知策略原样透出不吞', () => {
    expect(formatImportSuccess(base)).toBe('已导入「风雪夜归」：12 章，34,567 字')
    expect(formatImportSuccess({ ...base, split_strategy: 'future-x' }))
      .toContain('（future-x）')
  })
})

describe('formatImportWarnings', () => {
  it('无告警返回 null；前 2 条拼接', () => {
    expect(formatImportWarnings(base)).toBeNull()
    expect(formatImportWarnings({
      ...base,
      warnings: [{ message: '第3章过短' }, { message: '第7章过短' }],
    })).toBe('第3章过短；第7章过短')
  })

  it('溢出计数「等 N 条提示」', () => {
    expect(formatImportWarnings({
      ...base,
      warnings: [
        { message: 'a' }, { message: 'b' }, { message: 'c' },
      ],
    })).toBe('a；b；等 3 条提示')
  })
})
