/**
 * 小说全文搜索升级（v4.3h）前端配套纯函数：
 * - 调用后端 NovelSearch（每章全部命中 + 段落/偏移位置 + 全书汇总）；
 * - 结果列表「共 N 处 · M 章」汇总；
 * - snippet 命中词切分（列表高亮）；
 * - 段内命中定位（rune 偏移 → JS UTF-16 偏移，供阅读页 scrollIntoView + 短暂高亮）。
 */
import * as App from '../../wailsjsCompat'
import type { NovelSearchHitData } from '../../types/wails'

export type { NovelSearchHitData }

/** 调用后端全文检索；空查询直接返回 []，后端异常向上抛出由调用方提示。 */
export async function searchNovelAll(query: string): Promise<NovelSearchHitData[]> {
  const q = query.trim()
  if (!q) return []
  // 生成的 wailsjs 模型尚未包含新增字段，经 unknown 收窄到最新载荷类型。
  const hits = (await App.NovelSearch(q)) as unknown as NovelSearchHitData[] | null
  return hits ?? []
}

export interface SearchSummary {
  /** 全书总命中数（共 N 处） */
  total: number
  /** 命中章节总数（M 章） */
  chapters: number
  /** 实际返回条数（可能因上限 300 截断） */
  shown: number
}

/** 汇总「共 N 处 · M 章」：优先读后端冗余汇总字段，旧载荷（缺字段）时退化为本地统计。 */
export function summarizeSearch(hits: NovelSearchHitData[]): SearchSummary {
  const first = hits[0]
  const total = first && Number.isFinite(first.total_hits) && first.total_hits > 0
    ? first.total_hits
    : hits.length
  const chapters = first && Number.isFinite(first.chapter_count) && first.chapter_count > 0
    ? first.chapter_count
    : new Set(hits.map((h) => h.node_id)).size
  return { total, chapters, shown: hits.length }
}

/**
 * 段内查询所有命中的起始偏移（大小写不敏感，UTF-16 偏移）。
 * 命中互不重叠、按整词长推进 —— 与后端扫描推进方式一致，保证 match_index 可对齐。
 */
export function findQueryOccurrences(text: string, query: string): number[] {
  if (!query) return []
  const t = text.toLowerCase()
  const q = query.toLowerCase()
  if (!q) return []
  const out: number[] = []
  let from = 0
  for (;;) {
    const i = t.indexOf(q, from)
    if (i < 0) break
    out.push(i)
    from = i + q.length
  }
  return out
}

/** rune（码点）偏移 → JS 字符串 UTF-16 偏移（后端 char_offset 按 rune 计）。 */
export function runeOffsetToUtf16(text: string, runeOffset: number): number {
  if (runeOffset <= 0) return 0
  const cps = Array.from(text)
  if (runeOffset >= cps.length) return text.length
  return cps.slice(0, runeOffset).join('').length
}

/**
 * 在段落内定位目标命中词的 UTF-16 起始偏移：
 * 优先精确匹配后端 char_offset；内容漂移（章节已被编辑）时退化为最近的命中；无命中返回 -1。
 */
export function locateParagraphMatch(paragraphText: string, query: string, charOffsetRunes: number): number {
  const occ = findQueryOccurrences(paragraphText, query)
  if (occ.length === 0) return -1
  const target = runeOffsetToUtf16(paragraphText, charOffsetRunes)
  let best = occ[0]
  for (const o of occ) {
    if (o === target) return o
    if (Math.abs(o - target) < Math.abs(best - target)) best = o
  }
  return best
}

export interface SnippetSeg {
  text: string
  match: boolean
}

/** snippet 按命中词切段（大小写不敏感），供结果列表高亮渲染。 */
export function splitSnippet(snippet: string, query: string): SnippetSeg[] {
  if (!query) return [{ text: snippet, match: false }]
  const occ = findQueryOccurrences(snippet, query)
  if (occ.length === 0) return [{ text: snippet, match: false }]
  const segs: SnippetSeg[] = []
  let pos = 0
  for (const o of occ) {
    if (o > pos) segs.push({ text: snippet.slice(pos, o), match: false })
    segs.push({ text: snippet.slice(o, o + query.length), match: true })
    pos = o + query.length
  }
  if (pos < snippet.length) segs.push({ text: snippet.slice(pos), match: false })
  return segs
}
