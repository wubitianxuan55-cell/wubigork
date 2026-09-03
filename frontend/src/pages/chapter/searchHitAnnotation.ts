/**
 * searchHitAnnotation — 搜索命中「落为划线」适配纯函数：
 * 全文检索命中（NovelSearchHitData）带的是区间口径 —— paragraph_index + 段内 rune 偏移
 * （char_offset）+ match_len，不含命中原文；而划线标注（ReadingAnnotation）走摘录文本
 * 口径 —— 持久化 text，回渲染时在段落内 text.indexOf 回定位（planAnnotationMatches）。
 * 两者口径不一致，本模块把命中适配为 addHighlight 的既有入参（nodeId / title / 摘录文本），
 * 持久化与回渲染路径零新增。
 * 命中原文从 snippet 内取：后端 snippet = 命中词前后各 40 rune 的窗口（novel_search_handler
 * snippetAround），必含完整命中；大小写不敏感检索时章节内原文大小写可能与查询词不同，
 * 摘录必须用原文才能被回渲染的 text.indexOf 命中。截取长度按查询词 UTF-16 长度推进，
 * 与 highlightSearchHitAt 的 endAt = start + query.length 同一口径。
 */
import { findQueryOccurrences, type NovelSearchHitData } from './novelSearchUtils'

/** 划线标注归属目标（对齐 addHighlight 的 nodeId / title 来源口径） */
export interface SearchHitAnchor {
  nodeId: string
  title: string
  /** 摘录文本（章节内命中原文），即 addHighlight 的 textOverride */
  text: string
}

/**
 * 命中 → 划线入参适配：
 * - 标题命中（paragraph_index < 0）：正文无对应段落、划线无法回渲染，返回 null；
 * - 空查询返回 null（正常交互下命中列表必然来自非空查询，防御旧载荷）；
 * - 摘录取 snippet 内首个查询命中的原文（保留原文大小写）；snippet 异常不含查询词时
 *   退化为查询词本身（正常后端载荷不会走到）。
 */
export function searchHitAnchor(hit: NovelSearchHitData, query: string): SearchHitAnchor | null {
  const q = query.trim()
  if (!q || hit.paragraph_index < 0) return null
  const occ = findQueryOccurrences(hit.snippet, q)
  const text = occ.length > 0 ? hit.snippet.slice(occ[0], occ[0] + q.length) : q
  return { nodeId: hit.node_id, title: hit.title, text }
}
