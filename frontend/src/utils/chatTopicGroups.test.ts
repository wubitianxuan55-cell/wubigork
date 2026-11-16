import { describe, expect, it } from 'vitest'
import { groupChatTopics, monthKey } from './chatTopicGroups'

// G1 时间分组测试（chat-demo sessionGroups.ts 语义对齐）：
// 今天 / 昨天 / 7 天内 / 30 天内 / 按月；相邻同组合并、顺序不重排。
const NOW = new Date('2026-11-16T15:00:00').getTime()

interface T { id: string; updatedAt?: number; createdAt: number }
const t = (id: string, updatedAt?: number, createdAt = updatedAt ?? NOW): T => ({ id, updatedAt, createdAt })

describe('groupChatTopics', () => {
  it('边界：今天/昨天/7天内/30天内/按月', () => {
    const groups = groupChatTopics([
      t('a', new Date('2026-11-16T09:00:00').getTime()),          // 今天
      t('b', new Date('2026-11-15T23:00:00').getTime()),          // 昨天
      t('c', new Date('2026-11-15T08:00:00').getTime()),          // 昨天（跨日 diff=1）
      t('d', new Date('2026-11-11T12:00:00').getTime()),          // 5 天前 → 7 天内
      t('e', new Date('2026-11-01T12:00:00').getTime()),          // 15 天前 → 30 天内
      t('f', new Date('2026-09-20T12:00:00').getTime()),          // 57 天前 → 9 月
      t('g', new Date('2026-09-02T12:00:00').getTime()),          // 75 天前 → 9 月（与 f 同组合并）
    ], NOW)
    expect(groups.map(g => g.label)).toEqual(['今天', '昨天', '7 天内', '30 天内', '2026-09'])
    expect(groups[1].topics.map(x => x.id)).toEqual(['b', 'c'])
    expect(groups[4].topics.map(x => x.id)).toEqual(['f', 'g'])
  })

  it('分组顺序 = 首现顺序（不重排；乱序时间各自成组）', () => {
    const groups = groupChatTopics([
      t('old', new Date('2026-09-20T12:00:00').getTime()),
      t('new', new Date('2026-11-16T09:00:00').getTime()),
    ], NOW)
    expect(groups.map(g => g.label)).toEqual(['2026-09', '今天'])
  })

  it('updatedAt 缺省回退 createdAt', () => {
    const groups = groupChatTopics([{ id: 'x', createdAt: new Date('2026-11-16T10:00:00').getTime() }], NOW)
    expect(groups[0].label).toBe('今天')
  })

  it('空列表空分组', () => {
    expect(groupChatTopics([], NOW)).toEqual([])
  })

  it('monthKey 两位补零', () => {
    expect(monthKey(new Date('2026-03-05').getTime())).toBe('2026-03')
  })
})
