/**
 * 阅读伴读 API（章节摘要 / 划线提问），只读当前章节本地文本，不写文件。
 */
import * as App from '../../../../src/wailsjsCompat'

export type ReadingAssistantKind = 'summary' | 'ask'

/** AI 阅读伴读：summary=章节摘要，ask=针对摘选原文提问 */
export async function askReadingAssistant(
  kind: ReadingAssistantKind,
  title: string,
  chapterText: string,
  selection: string,
  question: string,
): Promise<string> {
  return App.NovelReadingAsk(kind, title, chapterText, selection, question)
}
