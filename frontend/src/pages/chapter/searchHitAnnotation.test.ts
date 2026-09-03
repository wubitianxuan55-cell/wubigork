// searchHitAnnotation.test.ts — 搜索命中「落为划线」适配纯函数：
// 锁命中（区间口径）→ 划线入参（摘录文本口径）的转换规则与边界（标题命中/空查询/原文大小写）。
import { describe, expect, it } from 'vitest'
import { searchHitAnchor } from './searchHitAnnotation'
import type { NovelSearchHitData } from './novelSearchUtils'

const hit = (over: Partial<NovelSearchHitData> = {}): NovelSearchHitData => ({
  node_id: 'ch-1',
  title: '第一回 风雪夜归人',
  chapter_num: 1,
  snippet: '夜色沉沉，雨落在窗台上。',
  title_hit: false,
  match_index: 1,
  paragraph_index: 0,
  char_offset: 4,
  match_len: 1,
  total_hits: 1,
  chapter_count: 1,
  ...over,
})

describe('searchHitAnchor（命中 → addHighlight 入参适配）', () => {
  it('正文命中：归属命中章节，摘录取 snippet 内命中原文', () => {
    expect(searchHitAnchor(hit(), '，')).toEqual({
      nodeId: 'ch-1',
      title: '第一回 风雪夜归人',
      text: '，',
    })
  })

  it('大小写不敏感检索：摘录保留章节内原文大小写（回渲染 text.indexOf 依赖原文）', () => {
    const h = hit({ snippet: 'he whispered Alpha secretly', paragraph_index: 2, char_offset: 13 })
    expect(searchHitAnchor(h, 'alpha')?.text).toBe('Alpha')
  })

  it('标题命中（paragraph_index < 0）：无正文段落可回渲染，返回 null', () => {
    expect(searchHitAnchor(hit({ title_hit: true, paragraph_index: -1, char_offset: -1 }), '夜色')).toBeNull()
  })

  it('空查询 / 纯空白查询：返回 null', () => {
    expect(searchHitAnchor(hit(), '')).toBeNull()
    expect(searchHitAnchor(hit(), '   ')).toBeNull()
  })

  it('snippet 异常不含查询词（旧载荷防御）：退化为查询词本身', () => {
    const h = hit({ snippet: '（截断异常的上下文）' })
    expect(searchHitAnchor(h, '夜色')).toEqual({ nodeId: 'ch-1', title: '第一回 风雪夜归人', text: '夜色' })
  })

  it('取 snippet 内首个命中（后端窗口必含完整命中，多命中时对齐第一处）', () => {
    const h = hit({ snippet: '剑修遇剑，剑鸣不止' })
    expect(searchHitAnchor(h, '剑')?.text).toBe('剑')
  })
})
