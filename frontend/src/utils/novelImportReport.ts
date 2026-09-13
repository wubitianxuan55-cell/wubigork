/**
 * novelImportReport — 导入成品小说的报告文案（v4.279 报告面的共享收口）。
 *
 * 文件导入（ImportNovelBook）与书源在线导入（NovelBookSourceImport done 事件）
 * 共用同一条落库链，成功文案与告警摘要也共用：切分策略 → 中文标签（书源线
 * 补 booksource 映射，规格 docs/gaea-novel-booksource-import-2026-09.md §8.1）、
 * 编码 + 策略拼「（…）」、告警取前 2 条 + 溢出计数。
 */

/** 导入结果报告的最小结构（NovelImportResult 线上形状，camelCase/snake_case 混用保持线上原样）。 */
export interface ImportReportLike {
  path: string
  title: string
  chapter_count: number
  total_words: number
  encoding?: string
  split_strategy?: string
  warnings?: Array<{ code?: string; message: string; level?: string }>
}

/** 切分策略 → 中文标签（booksource = 书源在线导入：章节来自引擎直下，非文件解析）。 */
export const SPLIT_STRATEGY_LABELS: Record<string, string> = {
  strong: '标题分章',
  weak: '短标题分章',
  window: '窗口分章',
  single: '单章',
  epub: 'EPUB',
  booksource: '在线书源',
}

/** 成功文案主体：「已导入《…》：N 章，M 字（编码 · 策略）」。 */
export function formatImportSuccess(res: ImportReportLike): string {
  const how = [
    res.encoding,
    res.split_strategy ? (SPLIT_STRATEGY_LABELS[res.split_strategy] ?? res.split_strategy) : '',
  ]
    .filter(Boolean)
    .join(' · ')
  return `已导入「${res.title}」：${res.chapter_count} 章，${res.total_words.toLocaleString()} 字${
    how ? `（${how}）` : ''
  }`
}

/** 告警摘要：前 2 条消息 + 溢出计数；无告警返回 null。 */
export function formatImportWarnings(res: ImportReportLike): string | null {
  const warns = res.warnings ?? []
  if (warns.length === 0) return null
  const head = warns.slice(0, 2).map((w) => w.message).join('；')
  return warns.length > 2 ? `${head}；等 ${warns.length} 条提示` : head
}
