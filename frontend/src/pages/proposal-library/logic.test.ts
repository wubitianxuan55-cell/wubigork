// logic.test.ts — 方案库板块纯逻辑：折叠/可见行/祖先、命中摘要切分、文案格式。
import { describe, expect, it } from 'vitest'
import {
  ancestorsOf,
  expandAncestors,
  formatChars,
  hasChildren,
  indexLabel,
  initialCollapsed,
  parentOf,
  searchSummary,
  splitSnippet,
  stepRow,
  visibleRows,
} from './logic'

// 一份 1~4 级的小标题树：
//   1 第一章 ─ 2 1.1 ─ 3 1.1.1 ─ 4 1.1.1.1
//   5 第二章 ─ 6 2.1
const tree = [
  { id: 1, level: 1 }, { id: 2, level: 2 }, { id: 3, level: 3 }, { id: 4, level: 4 },
  { id: 5, level: 1 }, { id: 6, level: 2 },
]
const ids = (rows: { node: { id: number } }[]) => rows.map((r) => r.node.id)

describe('标题树的父子与祖先', () => {
  it('hasChildren：紧随其后出现更深一级即为父节点', () => {
    expect(hasChildren(tree, 1)).toBe(true)
    expect(hasChildren(tree, 2)).toBe(true)
    expect(hasChildren(tree, 3)).toBe(true)
    expect(hasChildren(tree, 4)).toBe(false)
    expect(hasChildren(tree, 5)).toBe(true)
    expect(hasChildren(tree, 6)).toBe(false)
    expect(hasChildren(tree, 99)).toBe(false)
  })

  it('parentOf：往前找最近的更高级标题；一级标题与未知 id 为 null', () => {
    expect(parentOf(tree, 2)).toBe(1)
    expect(parentOf(tree, 4)).toBe(3)
    expect(parentOf(tree, 6)).toBe(5)
    expect(parentOf(tree, 1)).toBeNull()
    expect(parentOf(tree, 99)).toBeNull()
  })

  it('ancestorsOf：自近及远，跳级的文档也能找到链', () => {
    expect(ancestorsOf(tree, 4)).toEqual([3, 2, 1])
    expect(ancestorsOf(tree, 6)).toEqual([5])
    expect(ancestorsOf(tree, 1)).toEqual([])
  })
})

describe('折叠与可见行', () => {
  it('初始折叠：章展开到节，2 级及更深的有子节者收起', () => {
    expect([...initialCollapsed(tree)].sort()).toEqual([2, 3])
    // 首屏：第一章、1.1、第二章、2.1
    expect(ids(visibleRows(tree, initialCollapsed(tree)))).toEqual([1, 2, 5, 6])
  })

  it('折叠一级标题隐藏它的全部后代，直到下一个同级或更高级标题', () => {
    expect(ids(visibleRows(tree, new Set([1])))).toEqual([1, 5, 6])
  })

  it('缩进层数 = level - 1', () => {
    const rows = visibleRows(tree, new Set())
    expect(rows.map((r) => r.depth)).toEqual([0, 1, 2, 3, 0, 1])
  })

  it('expandAncestors 展开目标节点的全部祖先且不改动原折叠集', () => {
    const collapsed = new Set([1, 2, 3])
    const next = expandAncestors(collapsed, tree, 4)
    expect([...next]).toEqual([])
    expect([...collapsed]).toEqual([1, 2, 3])
    expect(ids(visibleRows(tree, next))).toEqual([1, 2, 3, 4, 5, 6])
  })

  it('键盘上下移：在可见行里走，到头停住', () => {
    const rows = visibleRows(tree, new Set([2]))
    expect(rows.map((r) => r.node.id)).toEqual([1, 2, 5, 6])
    expect(stepRow(rows, null, 1)).toBe(1)
    expect(stepRow(rows, 2, 1)).toBe(5)
    expect(stepRow(rows, 6, 1)).toBe(6)
    expect(stepRow(rows, 1, -1)).toBe(1)
    expect(stepRow([], 1, 1)).toBeNull()
  })
})

describe('文案', () => {
  it('formatChars：不足一万显示字数，过万折成万字', () => {
    expect(formatChars(0)).toBe('0 字')
    expect(formatChars(9999)).toBe('9999 字')
    expect(formatChars(12000)).toBe('1.2 万字')
    expect(formatChars(285000)).toBe('28.5 万字')
  })

  it('indexLabel：索引完成或没有正文不显示，未完成提示进度', () => {
    expect(indexLabel(12, 12)).toBe('')
    expect(indexLabel(0, 0)).toBe('')
    expect(indexLabel(0, 7)).toBe('语义索引待建立')
    expect(indexLabel(3, 7)).toBe('语义索引 3/7')
  })

  it('searchSummary：模式与备注拼成一句；降级说明原样带出', () => {
    expect(searchSummary('hybrid', '已本地精排')).toBe('全文 + 语义 · 已本地精排')
    expect(searchSummary('title', '')).toBe('标题树')
    expect(searchSummary('keyword', '本地 embedding 不可用，本次只按全文路检索')).toBe(
      '全文 · 本地 embedding 不可用，本次只按全文路检索',
    )
  })
})

describe('命中摘要的【】切分', () => {
  it('括号内是命中词，括号外是普通文本；相邻命中连成一段', () => {
    expect(splitSnippet('基坑【边坡失稳】、扬尘为主要【危险源】。')).toEqual([
      { text: '基坑', hit: false },
      { text: '边坡失稳', hit: true },
      { text: '、扬尘为主要', hit: false },
      { text: '危险源', hit: true },
      { text: '。', hit: false },
    ])
  })

  it('无标记与空串', () => {
    expect(splitSnippet('纯文本')).toEqual([{ text: '纯文本', hit: false }])
    expect(splitSnippet('')).toEqual([])
  })
})
