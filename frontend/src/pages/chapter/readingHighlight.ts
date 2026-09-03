/**
 * readingHighlight — 阅读页「朗读/搜索定位高亮」纯 DOM 工具（自 ChapterPage 原样搬移）：
 * - textNodesOf：收集元素内全部非空文本节点及其全文偏移区间；
 * - clearReadingHighlight：解开指定 class 的高亮容器并把文本合并回原处；
 * - highlightSearchHitAt：按后端段落索引 + 段内 rune 偏移把该处命中词包进高亮 span
 *   （与 applyTextHighlight「全文首个命中」不同，只作用于指定段落内的指定那一处）。
 * 三者仅做 DOM 读/写，无组件状态依赖；原先经 useCallback 闭包读组件内的
 * readingScrollRef，抽离后把滚动根作为显式参数传入，调用点行为保持一致。
 */
import { locateParagraphMatch } from './novelSearchUtils'

/** 元素内一个文本节点及其在元素全文本中的偏移区间（UTF-16，end 不含） */
export interface TextNodeSpan {
  node: Text
  start: number
  end: number
}

/** 按文档序收集 el 内非空文本节点（跳过空节点，偏移连续推进覆盖全文本）。 */
export function textNodesOf(el: HTMLElement): TextNodeSpan[] {
  const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT)
  const out: TextNodeSpan[] = []
  let pos = 0
  let n: Node | null = walker.nextNode()
  while (n) {
    const len = (n.textContent || '').length
    if (len > 0) out.push({ node: n as Text, start: pos, end: pos + len })
    pos += len
    n = walker.nextNode()
  }
  return out
}

/** 解开 root 内指定 class 的高亮容器：子节点移回原位后移除容器并合并相邻文本节点。 */
export function clearReadingHighlight(root: HTMLElement | null, className: string): void {
  if (!root) return
  root.querySelectorAll<HTMLElement>(`.${className}`).forEach((el) => {
    const parent = el.parentNode
    if (!parent) return
    while (el.firstChild) parent.insertBefore(el.firstChild, el)
    parent.removeChild(el)
    parent.normalize()
  })
}

/**
 * 搜索命中定位：按段落索引取段，段内定位命中词并包进高亮 span。
 * 未命中/越界/空查询返回 false 且不动 DOM；定位失败（内容漂移）时由调用方降级。
 */
export function highlightSearchHitAt(
  root: HTMLElement | null,
  paras: HTMLElement[],
  paraIdx: number,
  query: string,
  charOffset: number,
): boolean {
  const p = paras[paraIdx]
  if (!p || !query) return false
  const start = locateParagraphMatch(p.textContent || '', query, charOffset)
  if (start < 0) return false
  const endAt = start + query.length
  const nodes = textNodesOf(p)
  const startHolder = nodes.find((t) => start < t.end)
  const endHolder = nodes.find((t) => endAt <= t.end)
  if (!startHolder || !endHolder) return false
  try {
    clearReadingHighlight(root, 'novel-reading-search-hit')
    const range = document.createRange()
    range.setStart(startHolder.node, Math.max(0, start - startHolder.start))
    range.setEnd(endHolder.node, endAt - endHolder.start)
    const span = document.createElement('span')
    span.className = 'novel-reading-search-hit'
    span.appendChild(range.extractContents())
    range.insertNode(span)
    span.scrollIntoView({ block: 'center', behavior: 'smooth' })
    return true
  } catch { return false }
}
