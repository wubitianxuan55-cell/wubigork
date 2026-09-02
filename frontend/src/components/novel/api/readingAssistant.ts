/**
 * 阅读伴读 API（章节摘要 / 划线提问），只读当前章节本地文本，不写文件。
 * ask 支持多轮连续追问：history 非空时后端结合此前问答作答（空数组 = 单轮，兼容旧行为）。
 */
import * as App from '../../../../src/wailsjsCompat'

export type ReadingAssistantKind = 'summary' | 'ask'

/** 问书历史一轮（对齐后端 internal/app readingTurn）：q=用户问题，a=助手回答 */
export interface ReadingAskTurn {
  q: string
  a: string
}

/** AI 阅读伴读：summary=章节摘要（单轮），ask=针对摘选原文提问（history 非空为连续追问） */
export async function askReadingAssistant(
  kind: ReadingAssistantKind,
  title: string,
  chapterText: string,
  selection: string,
  question: string,
  history: ReadingAskTurn[] = [],
): Promise<string> {
  // 空历史发空串：后端解析失败/空串一律按单轮走，与旧签名行为完全兼容
  const historyJSON = history.length > 0 ? JSON.stringify(history) : ''
  return App.NovelReadingAsk(kind, title, chapterText, selection, question, historyJSON)
}
