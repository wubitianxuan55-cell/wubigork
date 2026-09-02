/**
 * AI 问书会话纯函数：消息流 ↔ 历史轮归并、历史截断。
 * 截断规则与后端 internal/app/novel_reading_handler.go 对齐
 * （最多保留最近 6 轮，每轮回答截 500 rune，前后端双保险防 prompt 爆炸）。
 */

export type ReadingAskRole = 'user' | 'assistant'

/** 问书弹窗里的一条消息（渲染用，含未成对的尾部提问） */
export interface ReadingAskMessage {
  role: ReadingAskRole
  content: string
}

/** 问书历史一轮（对齐后端 readingTurn，随请求发送） */
export interface ReadingAskTurn {
  q: string
  a: string
}

export const ASK_MAX_HISTORY_TURNS = 6
export const ASK_MAX_ANSWER_RUNES = 500

/** 按 rune（Unicode 码点）截断，截断处附省略号——与后端 truncateRunes 语义一致 */
export function truncateRunes(s: string, max: number): string {
  const r = [...s]
  if (r.length <= max) return s
  return r.slice(0, max - 1).join('') + '…'
}

/**
 * 把消息序列归并成问答轮：user 开启一轮，紧随的 assistant 收尾；
 * 未获回答的尾部 user 消息（发送中/已回滚）与无提问的 assistant 消息不成对、不进历史。
 */
export function deriveAskTurns(messages: ReadingAskMessage[]): ReadingAskTurn[] {
  const turns: ReadingAskTurn[] = []
  let pending: string | null = null
  for (const m of messages) {
    if (m.role === 'user') {
      pending = m.content
    } else if (pending !== null) {
      turns.push({ q: pending, a: m.content })
      pending = null
    }
  }
  return turns
}

/** 防止 prompt 爆炸：只保留最近 N 轮，每轮回答截断（问题保留原文） */
export function trimAskTurns(turns: ReadingAskTurn[]): ReadingAskTurn[] {
  return turns.slice(-ASK_MAX_HISTORY_TURNS).map((t) => ({
    q: t.q,
    a: truncateRunes(t.a, ASK_MAX_ANSWER_RUNES),
  }))
}

/** 由会话消息构建随请求发送的历史（空数组 = 单轮） */
export function buildAskHistory(messages: ReadingAskMessage[]): ReadingAskTurn[] {
  return trimAskTurns(deriveAskTurns(messages))
}
