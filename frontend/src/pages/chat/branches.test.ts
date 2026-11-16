import { describe, expect, it } from 'vitest'
import { buildVersions, buildVisiblePath, btOf, dbIdOf } from './branches'
import type { ChatMsg } from './types'

// A 批分支树纯函数测试（Go internal/chat/branches_test.go 同构镜像）。

function feRow(key: string, role: 'user' | 'assistant', content: string, streaming = false): ChatMsg {
  return { key, role, content, createdAt: '', ...(streaming ? { streaming: true } : {}) }
}

function dbRow(id: number, role: 'user' | 'assistant', content: string, bt?: { p: number; c: number }): ChatMsg {
  return {
    key: `db_${id}`, role, content, createdAt: '2026-01-01T00:00:00Z',
    ...(bt ? { extra: { bt } } : {}),
  }
}

function ids(path: ChatMsg[]): number[] {
  return path.map(m => dbIdOf(m.key) as number)
}

describe('branches 基础', () => {
  it('dbIdOf 只认 db_ 前缀', () => {
    expect(dbIdOf('db_42')).toBe(42)
    expect(dbIdOf('new_x1')).toBeNull()
  })
  it('btOf 缺失/非法返回 null', () => {
    expect(btOf(dbRow(1, 'user', 'x'))).toBeNull()
    expect(btOf({ key: 'db_2', role: 'user', content: '', createdAt: '', extra: { bt: 'oops' } })).toBeNull()
    expect(btOf(dbRow(3, 'user', '', { p: 1, c: 0 }))).toEqual({ p: 1, c: 0 })
  })
})

describe('buildVisiblePath', () => {
  it('线形话题直通（无 bt 行 + 本地行追加尾）', () => {
    const msgs = [dbRow(1, 'user', 'a'), dbRow(2, 'assistant', 'b'),
      feRow('new_x', 'assistant', 'c', true)]
    // db 行直通 + 本地行追加尾（路径含流式行，ids() 对本地行返回 null 属预期）
    expect(buildVisiblePath(msgs).map(m => m.key)).toEqual(['db_1', 'db_2', 'new_x'])
  })

  it('树形：叶子回溯走选中链，隐藏分支不进路径', () => {
    // q1 →(c=10) a1a →(c=2) q2 →(c=20) a2；a1b 隐藏
    const msgs = [
      dbRow(1, 'user', 'q1', { p: 0, c: 10 }),
      dbRow(10, 'assistant', 'a1a', { p: 1, c: 2 }),
      dbRow(11, 'assistant', 'a1b', { p: 1, c: 0 }),
      dbRow(2, 'user', 'q2', { p: 10, c: 20 }),
      dbRow(20, 'assistant', 'a2', { p: 2, c: 0 }),
    ]
    expect(ids(buildVisiblePath(msgs))).toEqual([1, 10, 2, 20])
  })

  it('切版本：父 c 指向另一子后路径切换', () => {
    const msgs = [
      dbRow(1, 'user', 'q1', { p: 0, c: 11 }),
      dbRow(10, 'assistant', 'a1a', { p: 1, c: 2 }),
      dbRow(11, 'assistant', 'a1b', { p: 1, c: 0 }),
      dbRow(2, 'user', 'q2', { p: 10, c: 20 }),
      dbRow(20, 'assistant', 'a2', { p: 2, c: 0 }),
    ]
    expect(ids(buildVisiblePath(msgs))).toEqual([1, 11])
  })

  it('分页窗口截断：父缺失按可见处理', () => {
    const msgs = [
      dbRow(10, 'assistant', 'a1a', { p: 1, c: 2 }),
      dbRow(11, 'assistant', 'a1b', { p: 1, c: 0 }),
      dbRow(2, 'user', 'q2', { p: 10, c: 20 }),
      dbRow(20, 'assistant', 'a2', { p: 2, c: 0 }),
    ]
    expect(ids(buildVisiblePath(msgs))).toEqual([10, 2, 20])
  })

  it('分支锚点：流式乐观行插入锚后并截断旧链', () => {
    const msgs = [
      dbRow(1, 'user', 'q1', { p: 0, c: 10 }),
      dbRow(10, 'assistant', 'a1a', { p: 1, c: 0 }),
      dbRow(11, 'assistant', 'a1b', { p: 1, c: 0 }),
      feRow('new_am', 'assistant', '', true),
    ]
    const path = buildVisiblePath(msgs, 'db_1')
    expect(path.map(m => m.key)).toEqual(['db_1', 'new_am'])
    // 锚点不存在（分页外）→ 退化为尾部追加
    const fallback = buildVisiblePath(msgs, 'db_99')
    expect(fallback[fallback.length - 1].key).toBe('new_am')
  })
})

describe('buildVersions', () => {
  it('按 bt.p 分组兄弟，index 为本行位次；单元素组不触发切换器', () => {
    const msgs = [
      dbRow(1, 'user', 'q1', { p: 0, c: 11 }),
      dbRow(10, 'assistant', 'a1a', { p: 1, c: 0 }),
      dbRow(11, 'assistant', 'a1b', { p: 1, c: 0 }),
    ]
    const v = buildVersions(msgs)
    expect(v.get('db_10')).toEqual({ ids: [10, 11], index: 0 })
    expect(v.get('db_11')).toEqual({ ids: [10, 11], index: 1 })
    expect(v.get('db_1')).toEqual({ ids: [1], index: 0 })
  })
})
