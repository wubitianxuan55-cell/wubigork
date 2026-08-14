import { describe, expect, it } from 'vitest'
import { parseCreateChapterEvent } from './chapterStreamTypes'

describe('parseCreateChapterEvent 判别联合分发（T6-7.5）', () => {
  it('phase 事件（writing / continuing 续写）', () => {
    const e = parseCreateChapterEvent({ type: 'phase', phase: 'continuing', attempt: 2, current: 1200, target: 5000 })
    expect(e).not.toBeNull()
    if (e && e.type === 'phase') {
      expect(e.phase).toBe('continuing')
      expect(e.attempt).toBe(2)
      expect(e.current).toBe(1200)
      expect(e.target).toBe(5000)
    }
    const w = parseCreateChapterEvent({ type: 'phase', phase: 'writing', target: 5000 })
    expect(w?.type).toBe('phase')
  })

  it('chunk 事件携带正文增量', () => {
    const e = parseCreateChapterEvent({ type: 'chunk', content: '正文片段', total: 42 })
    expect(e).not.toBeNull()
    if (e && e.type === 'chunk') {
      expect(e.content).toBe('正文片段')
      expect(e.total).toBe(42)
    }
  })

  it('done / error 事件', () => {
    const done = parseCreateChapterEvent({ type: 'done', chapterNum: 3, branch: '', total: 5120, content: '...' })
    expect(done?.type).toBe('done')
    const err = parseCreateChapterEvent({ type: 'error', error: 'boom' })
    expect(err?.type).toBe('error')
    if (err && err.type === 'error') expect(err.error).toBe('boom')
  })

  it('cancelled 事件（后端批 1 新增：携带已落盘部分正文）', () => {
    const e = parseCreateChapterEvent({ type: 'cancelled', chapterNum: 2, branch: 'a', nodeId: 'n1', total: 900, content: '部分正文' })
    expect(e).not.toBeNull()
    if (e && e.type === 'cancelled') {
      expect(e.chapterNum).toBe(2)
      expect(e.branch).toBe('a')
      expect(e.content).toBe('部分正文')
      expect(e.total).toBe(900)
    }
    // content 缺省（部分为空或落盘失败）：仍解析为合法 cancelled 事件
    const bare = parseCreateChapterEvent({ type: 'cancelled', chapterNum: 1, branch: '', nodeId: 'n2' })
    expect(bare?.type).toBe('cancelled')
  })

  it('畸形/未知负载返回 null（不静默抛错）', () => {
    expect(parseCreateChapterEvent(null)).toBeNull()
    expect(parseCreateChapterEvent('str')).toBeNull()
    expect(parseCreateChapterEvent(42)).toBeNull()
    expect(parseCreateChapterEvent({})).toBeNull()
    expect(parseCreateChapterEvent({ type: 'unknown' })).toBeNull()
    expect(parseCreateChapterEvent({ type: 'phase' })).toBeNull() // 缺 phase 字段
    expect(parseCreateChapterEvent({ type: 'chunk' })).toBeNull() // 缺 content 字段
  })
})
