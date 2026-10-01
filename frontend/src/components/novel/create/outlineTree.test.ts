import { describe, expect, it } from 'vitest'
import { flattenChapters, nextToWriteChapter } from './outlineTree'
import type { OutlineNode } from '../../../types'

const node = (o: Partial<OutlineNode> & { order_index: number }): OutlineNode => ({
  id: `n${o.order_index}-${Math.random().toString(36).slice(2, 6)}`,
  title: `第${o.order_index}章`,
  summary: '',
  status: 'planned',
  ...o,
} as OutlineNode)

describe('nextToWriteChapter（卷结构章号两次实弹事故的回归锚）', () => {
  it('卷结构：章挂在卷 children 下，只扫顶层会算出 1 反复覆盖第1章——必须全树展平', () => {
    const outlines = [
      node({ order_index: 0, title: '卷一·雾都', status: 'done', children: [
        node({ order_index: 1, status: 'done', parent_id: 'vol1' }),
      ] }),
    ]
    // 卷内第1章已写、无 planned → 顺延卷内最大章号+1 = 第2章（而不是顶层的 1）
    expect(nextToWriteChapter(outlines)).toBe(2)
  })

  it('卷结构：卷内有 planned 章 → 生成那条（第2章 planned 补写本人）', () => {
    const outlines = [
      node({ order_index: 0, title: '卷一', status: 'done', children: [
        node({ order_index: 1, status: 'done' }),
        node({ order_index: 2, status: 'planned' }),
      ] }),
    ]
    expect(nextToWriteChapter(outlines)).toBe(2)
  })

  it('平铺大纲（旧式顶层章）：行为不变——planned 优先，全写过顺延 max+1', () => {
    expect(nextToWriteChapter([
      node({ order_index: 1, status: 'done' }),
      node({ order_index: 2, status: 'planned' }),
    ])).toBe(2)
    expect(nextToWriteChapter([
      node({ order_index: 1, status: 'done' }),
      node({ order_index: 2, status: 'done' }),
    ])).toBe(3)
  })

  it('空大纲 → 第1章；writing/done 不算未写（中断残留走「重新生成」显式覆盖）', () => {
    expect(nextToWriteChapter([])).toBe(1)
    expect(nextToWriteChapter([
      node({ order_index: 1, status: 'writing' }),
      node({ order_index: 2, status: 'done' }),
    ])).toBe(3)
  })

  it('flattenChapters：卷节点（order 0）不入列，先序收集按章号排序', () => {
    const flat = flattenChapters([
      node({ order_index: 0, title: '卷一', status: 'done', children: [
        node({ order_index: 2, status: 'planned' }),
        node({ order_index: 1, status: 'done' }),
      ] }),
    ])
    expect(flat.map(n => n.order_index)).toEqual([1, 2])
  })
})
