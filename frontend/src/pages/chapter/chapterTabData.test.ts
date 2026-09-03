// chapterTabData.test.ts — 章节 Tab 数据构造与关闭守卫纯函数测试（自 ChapterPage 搬出）。
import { describe, expect, it } from 'vitest'
import { createTabData, needsCloseConfirm } from './chapterTabData'
import type { ChapterTabData, OutlineNode } from '../../types'

const node = { id: 'ch-1', title: '第一回', order_index: 3 } as unknown as OutlineNode

describe('createTabData', () => {
  it('按大纲节点构造空白数据骨架', () => {
    const tab = createTabData(node)
    expect(tab.node).toBe(node)
    expect(tab.chapterNum).toBe(3)
    expect(tab.scenes).toEqual([''])
    expect(tab.saved).toBe(false)
    expect(tab.messages).toEqual([])
    expect(tab.targetWords).toBe(3000)
    expect(tab.retryStatus).toBeNull()
  })

  it('order_index 缺失时 chapterNum 回落 0（无绑定章节不可保存）', () => {
    const tab = createTabData({ id: 'x', order_index: 0 } as unknown as OutlineNode)
    expect(tab.chapterNum).toBe(0)
  })
})

describe('needsCloseConfirm', () => {
  const tab = (patch: Partial<ChapterTabData>): ChapterTabData =>
    ({ ...createTabData(node), ...patch })

  it('未保存且有内容：需要确认', () => {
    expect(needsCloseConfirm(tab({ saved: false, scenes: ['正文内容'] }))).toBe(true)
  })

  it('已保存 / 无内容 / 纯空白场景：直接关闭', () => {
    expect(needsCloseConfirm(tab({ saved: true, scenes: ['正文内容'] }))).toBe(false)
    expect(needsCloseConfirm(tab({ saved: false, scenes: [''] }))).toBe(false)
    expect(needsCloseConfirm(tab({ saved: false, scenes: ['  ', '\n\t'] }))).toBe(false)
  })

  it('目标不存在（key 未命中）不需要确认', () => {
    expect(needsCloseConfirm(null)).toBe(false)
    expect(needsCloseConfirm(undefined)).toBe(false)
  })
})
