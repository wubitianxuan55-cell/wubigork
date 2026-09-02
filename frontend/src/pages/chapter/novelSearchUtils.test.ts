import { describe, expect, it } from 'vitest'
import {
  findQueryOccurrences,
  locateParagraphMatch,
  runeOffsetToUtf16,
  splitSnippet,
  summarizeSearch,
  type NovelSearchHitData,
} from './novelSearchUtils'

const hit = (over: Partial<NovelSearchHitData>): NovelSearchHitData => ({
  node_id: 'n1',
  title: '第一章',
  chapter_num: 1,
  snippet: '……目标词……',
  title_hit: false,
  match_index: 1,
  paragraph_index: 0,
  char_offset: 0,
  match_len: 3,
  total_hits: 1,
  chapter_count: 1,
  ...over,
})

describe('findQueryOccurrences（段内命中偏移，大小写不敏感）', () => {
  it('返回全部命中的 UTF-16 起始偏移', () => {
    expect(findQueryOccurrences('剑修山水剑修', '剑修')).toEqual([0, 4])
  })
  it('大小写不敏感', () => {
    expect(findQueryOccurrences('AbC abc', 'abc')).toEqual([0, 4])
  })
  it('按整词长推进（不重叠）', () => {
    expect(findQueryOccurrences('aaaa', 'aa')).toEqual([0, 2])
  })
  it('空查询 / 无命中返回空数组', () => {
    expect(findQueryOccurrences('正文', '')).toEqual([])
    expect(findQueryOccurrences('正文', '不存在')).toEqual([])
  })
})

describe('runeOffsetToUtf16（rune → UTF-16 偏移）', () => {
  it('BMP 中文（常见路径）rune 偏移与 UTF-16 偏移一致', () => {
    expect(runeOffsetToUtf16('山门外的少年，一心想成为剑修。', 12)).toBe(12)
  })
  it('非 BMP 字符（如 emoji）按 1 个 rune 计', () => {
    const text = '🐉🐉甲乙'
    // 每个 emoji 是 1 个 rune、2 个 UTF-16 单元
    expect(runeOffsetToUtf16(text, 2)).toBe(4)
    expect(runeOffsetToUtf16(text, 3)).toBe(5)
  })
  it('边界：0 / 超出长度', () => {
    expect(runeOffsetToUtf16('正文', 0)).toBe(0)
    expect(runeOffsetToUtf16('正文', 99)).toBe(2)
  })
})

describe('locateParagraphMatch（定位段落内目标命中）', () => {
  it('char_offset 精确命中', () => {
    // 后端：段 "另一位剑修路过。" 中第二处命中 rune 偏移 3
    expect(locateParagraphMatch('另一位剑修路过。', '剑修', 3)).toBe(3)
  })
  it('内容漂移时退化为最近的命中', () => {
    // 期望偏移 12，但内容已变，只有偏移 2 一处 → 就近取 2
    expect(locateParagraphMatch('前缀剑修后缀', '剑修', 12)).toBe(2)
  })
  it('段内无命中返回 -1', () => {
    expect(locateParagraphMatch('没有目标', '剑修', 0)).toBe(-1)
  })
})

describe('splitSnippet（结果列表命中词切分）', () => {
  it('前后文 + 命中词三段', () => {
    expect(splitSnippet('他是剑修，号的剑修。', '剑修')).toEqual([
      { text: '他是', match: false },
      { text: '剑修', match: true },
      { text: '，号的', match: false },
      { text: '剑修', match: true },
      { text: '。', match: false },
    ])
  })
  it('无命中 / 空查询返回单段', () => {
    expect(splitSnippet('正文', '剑修')).toEqual([{ text: '正文', match: false }])
    expect(splitSnippet('正文', '')).toEqual([{ text: '正文', match: false }])
  })
})

describe('summarizeSearch（共 N 处 · M 章）', () => {
  it('优先读后端冗余汇总字段', () => {
    const hits = [
      hit({ total_hits: 320, chapter_count: 12 }),
      hit({ node_id: 'n2', chapter_num: 2 }),
    ]
    expect(summarizeSearch(hits)).toEqual({ total: 320, chapters: 12, shown: 2 })
  })
  it('旧载荷缺汇总字段时本地统计兜底', () => {
    const legacy = { node_id: 'n1', title: 't', chapter_num: 1, snippet: 's' } as unknown as NovelSearchHitData
    const hits = [legacy, { ...legacy, node_id: 'n2' }]
    expect(summarizeSearch(hits)).toEqual({ total: 2, chapters: 2, shown: 2 })
  })
  it('空结果为零', () => {
    expect(summarizeSearch([])).toEqual({ total: 0, chapters: 0, shown: 0 })
  })
})
