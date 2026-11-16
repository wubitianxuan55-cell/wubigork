// chatTopicGroups — 会话列表时间分组（G1，chat-demo sessionGroups.ts 语义对齐：
// 今天 / 昨天 / 7 天内 / 30 天内 / 按月 YYYY-MM）。topics 须已按 updatedAt 降序
// （ChatPage topicList 契约）；分组顺序 = 首现顺序，不重排。

export interface TopicGroupLike {
  /** 最近活跃时间（ms）；缺省回退 createdAt 由调用方保证 */
  updatedAt?: number
  createdAt: number
}

export interface TopicGroup<T> {
  key: string
  label: string
  topics: T[]
}

const DAY = 86_400_000

const startOfDay = (time: number): number => {
  const d = new Date(time)
  d.setHours(0, 0, 0, 0)
  return d.getTime()
}

const pad2 = (n: number): string => String(n).padStart(2, '0')

/** YYYY-MM 月份标签（30 天外的兜底分组键） */
export function monthKey(time: number): string {
  const d = new Date(time)
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}`
}

/**
 * 按最近活跃时间分组：今天 / 昨天 / 7 天内 / 30 天内 / 按月。
 * now 可注入（测试钉死时钟）。
 */
export function groupChatTopics<T extends TopicGroupLike>(topics: T[], now: number): TopicGroup<T>[] {
  const groups: TopicGroup<T>[] = []
  for (const topic of topics) {
    const t = topic.updatedAt || topic.createdAt
    const diffDays = Math.floor((startOfDay(now) - startOfDay(t)) / DAY)
    let key: string
    let label: string
    if (diffDays <= 0) {
      key = 'today'
      label = '今天'
    } else if (diffDays <= 1) {
      key = 'yesterday'
      label = '昨天'
    } else if (diffDays < 7) {
      key = '7days'
      label = '7 天内'
    } else if (diffDays < 30) {
      key = '30days'
      label = '30 天内'
    } else {
      key = monthKey(t)
      label = key
    }
    const last = groups[groups.length - 1]
    if (last && last.key === key) {
      last.topics.push(topic)
    } else {
      groups.push({ key, label, topics: [topic] })
    }
  }
  return groups
}
