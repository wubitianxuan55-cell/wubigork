import { describe, expect, it } from 'vitest'
import {
  ASK_MAX_ANSWER_RUNES, ASK_MAX_HISTORY_TURNS,
  buildAskHistory, deriveAskTurns, trimAskTurns, truncateRunes,
  type ReadingAskMessage,
} from './readingAskSession'

const msg = (role: ReadingAskMessage['role'], content: string): ReadingAskMessage => ({ role, content })

describe('truncateRunes', () => {
  it('短串原样返回', () => {
    expect(truncateRunes('你好', 10)).toBe('你好')
  })
  it('超长按 rune 截断并附省略号（中文按字符计）', () => {
    const out = truncateRunes('长'.repeat(600), 500)
    expect([...out].length).toBe(500)
    expect(out.endsWith('…')).toBe(true)
  })
  it('代理对（emoji）按 1 个 rune 计', () => {
    expect(truncateRunes('😀'.repeat(10), 5)).toBe('😀'.repeat(4) + '…')
  })
})

describe('deriveAskTurns', () => {
  it('按 user→assistant 相邻对归并成轮', () => {
    const turns = deriveAskTurns([
      msg('user', '他是谁'), msg('assistant', '主角'),
      msg('user', '那他后来呢'), msg('assistant', '远行了'),
    ])
    expect(turns).toEqual([
      { q: '他是谁', a: '主角' },
      { q: '那他后来呢', a: '远行了' },
    ])
  })
  it('尾部未回答的 user 消息不成对（发送中）', () => {
    const turns = deriveAskTurns([msg('user', 'q1'), msg('assistant', 'a1'), msg('user', 'q2')])
    expect(turns).toEqual([{ q: 'q1', a: 'a1' }])
  })
  it('无前置提问的 assistant 消息被忽略', () => {
    const turns = deriveAskTurns([msg('assistant', '孤立回答'), msg('user', 'q'), msg('assistant', 'a')])
    expect(turns).toEqual([{ q: 'q', a: 'a' }])
  })
  it('空会话得空历史', () => {
    expect(deriveAskTurns([])).toEqual([])
  })
})

describe('trimAskTurns', () => {
  it('超过上限只保留最近 N 轮', () => {
    const turns = Array.from({ length: 8 }, (_, i) => ({ q: `问题${i}`, a: `回答${i}` }))
    const trimmed = trimAskTurns(turns)
    expect(trimmed.length).toBe(ASK_MAX_HISTORY_TURNS)
    expect(trimmed[0].q).toBe('问题2')
    expect(trimmed[trimmed.length - 1].q).toBe('问题7')
  })
  it('长回答截断到 500 rune，问题不截断', () => {
    const trimmed = trimAskTurns([{ q: '问'.repeat(600), a: '答'.repeat(600) }])
    expect([...trimmed[0].a].length).toBe(ASK_MAX_ANSWER_RUNES)
    expect([...trimmed[0].q].length).toBe(600)
  })
})

describe('buildAskHistory', () => {
  it('端到端：消息流 → 截断后的历史轮', () => {
    const messages: ReadingAskMessage[] = [
      msg('user', '他的动机是什么'),
      msg('assistant', '复仇。'.repeat(200)),
      msg('user', '那他后来呢'),
    ] // 尾部 user 无回答，不进历史
    const history = buildAskHistory(messages)
    expect(history.length).toBe(1)
    expect(history[0].q).toBe('他的动机是什么')
    expect(history[0].a.endsWith('…')).toBe(true)
  })
  it('空会话 → 空数组（单轮）', () => {
    expect(buildAskHistory([])).toEqual([])
  })
})
