/**
 * readingHighlight — 阅读页「朗读/搜索定位高亮」纯 DOM 工具（自 ChapterPage 原样搬移）：
 * - textNodesOf：收集元素内全部非空文本节点及其全文偏移区间；
 * - clearReadingHighlight：解开指定 class 的高亮容器并把文本合并回原处；
 * - highlightSearchHitAt：按后端段落索引 + 段内 rune 偏移把该处命中词包进高亮 span
 *   （与 applyTextHighlight「全文首个命中」不同，只作用于指定段落内的指定那一处）；
 * - applyTextHighlight：从根元素顺序找首个含目标文本的段落并整体包亮（TTS 逐句跟随 /
 *   搜索定位兜底共用），仅 readMode 为真且根已挂载时生效；
 * - paraOf：向上找节点所属的 .novel-reading-p 阅读段落（划线/选区限制用）；
 * - textAtScrollTop：按滚动位置取容差内最后到达段落的摘录文本（书签预览用）。
 * 这些函数仅做 DOM 读/写或纯计算，无组件状态依赖；原先经闭包读组件内的
 * readingScrollRef / readMode，抽离后把滚动根等作为显式参数传入，调用点行为保持一致。
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

/**
 * 全文首个命中定位：从 root 顶部顺序扫描阅读段落，把第一个含目标文本的片段
 * 包进 className 高亮 span 并滚动到可视中央（TTS 逐句跟随 / 搜索定位兜底共用）。
 * 仅 root 已挂载且 readMode 为真时生效；空文本/未命中返回 false 且不产生高亮 DOM。
 * 自 ChapterPage 原样搬移：滚动根与 readMode 由组件闭包读取改为显式传参，逐行逻辑不变；
 * 调用点保留薄包装 + ref 以维持流式调用入口身份恒定（读取时机与原闭包一致）。
 */
export function applyTextHighlight(
  root: HTMLElement | null,
  readMode: boolean,
  rawText: string,
  className: string,
): boolean {
  if (!root || !readMode) return false
  const target = rawText.trim()
  if (!target) return false
  clearReadingHighlight(root, className)
  const paras = Array.from(root.querySelectorAll<HTMLElement>('.novel-reading-p'))
  for (const p of paras) {
    const full = p.textContent ?? ''
    const idx = full.indexOf(target)
    if (idx === -1) continue
    const nodes = textNodesOf(p)
    const endAt = idx + target.length
    const startHolder = nodes.find((t) => idx < t.end)
    const endHolder = nodes.find((t) => endAt <= t.end)
    if (!startHolder || !endHolder) break
    try {
      const range = document.createRange()
      range.setStart(startHolder.node, Math.max(0, idx - startHolder.start))
      range.setEnd(endHolder.node, endAt - endHolder.start)
      const span = document.createElement('span')
      span.className = className
      span.appendChild(range.extractContents())
      range.insertNode(span)
      span.scrollIntoView({ block: 'center', behavior: 'smooth' })
      return true
    } catch { /* ignore */ }
  }
  return false
}

/** 向上查找 node 所属的阅读段落（.novel-reading-p）；不在任何段落内时返回 null。 */
export function paraOf(node: Node): HTMLElement | null {
  let el: HTMLElement | null = node instanceof HTMLElement ? node : node.parentElement
  while (el && !el.classList.contains('novel-reading-p')) el = el.parentElement
  return el
}

/**
 * 书签摘录：取滚动位置 +48px 容差内最后到达段落的文本，去首尾空白后截断到 48 字。
 * 纯 DOM 计算，自 ChapterPage 原样搬移（el 参数放宽为 HTMLElement，调用点传滚动根不变）。
 */
export function textAtScrollTop(el: HTMLElement, scrollTop: number): string {
  const paras = el.querySelectorAll<HTMLElement>('.novel-reading-p')
  let best: HTMLElement | null = null
  for (const p of Array.from(paras)) {
    if (p.offsetTop <= scrollTop + 48) best = p
    else break
  }
  return (best?.textContent || '').trim().slice(0, 48)
}
