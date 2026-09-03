/**
 * readingAnnotation — 阅读页「划线 / 想法」回渲染工具（自 ChapterPage 原样搬移）：
 * - planAnnotationMatches：纯函数——在段落全文中为各划线摘录规划不重叠的命中区间
 *   （长摘录优先占位，同摘录取首个无冲突出现位置），返回按起始偏移降序的匹配计划；
 * - renderAnnotationHighlights：DOM 主体——清掉旧 mark[data-ann-id] 后按计划把摘录
 *   文本重新包进 mark，并挂点击回调（点击 mark 打开想法编辑）。
 * 摘录对应文本已被编辑掉的划线在段落中找不到位置、自然不产生 mark（与原 effect 一致）。
 * 原先经组件闭包读 readingScrollRef / annotations / openAnnotation，抽离后全部显式传参。
 */
import type { ReadingAnnotation } from '../../utils/readingAnnotations'

/** 单条划线在段落全文中的命中区间（UTF-16，end 不含） */
export interface AnnotationMatch {
  start: number
  end: number
  ann: ReadingAnnotation
}

/**
 * 在段落全文中为划线摘录规划不重叠命中区间：按摘录长度降序遍历（长摘录优先占位），
 * 已被占用的字符区间跳过；返回按起始偏移降序排列的匹配计划（从后往前套 mark 不串位）。
 */
export function planAnnotationMatches(text: string, anns: ReadingAnnotation[]): AnnotationMatch[] {
  const matches: AnnotationMatch[] = []
  const occupied = new Set<number>()
  for (const ann of [...anns].sort((a, b) => b.text.length - a.text.length)) {
    if (!ann.text) continue
    let idx = text.indexOf(ann.text)
    while (idx !== -1) {
      const end = idx + ann.text.length
      let conflict = false
      for (let k = idx; k < end; k++) {
        if (occupied.has(k)) { conflict = true; break }
      }
      if (!conflict) {
        matches.push({ start: idx, end, ann })
        for (let k = idx; k < end; k++) occupied.add(k)
        break
      }
      idx = text.indexOf(ann.text, idx + 1)
    }
  }
  matches.sort((a, b) => b.start - a.start)
  return matches
}

/**
 * 划线回渲染主体：解掉 root 内旧 mark 并合并文本节点后，按当前章节划线列表
 * 在各阅读段落内重新定位、重建 mark（class novel-reading-mark，data-ann-id/ann-color），
 * mark 点击时回调 onOpen。仅处理纯文本首子节点段落（回渲染前 normalize 保证成立）。
 */
export function renderAnnotationHighlights(
  root: HTMLElement,
  current: ReadingAnnotation[],
  onOpen: (ann: ReadingAnnotation) => void,
): void {
  root.querySelectorAll<HTMLElement>('mark[data-ann-id]').forEach((m) => {
    const parent = m.parentNode
    if (!parent) return
    parent.replaceChild(document.createTextNode(m.textContent ?? ''), m)
  })
  root.normalize()
  if (current.length === 0) return
  const paras = Array.from(root.querySelectorAll<HTMLElement>('.novel-reading-p'))
  for (const p of paras) {
    const text = p.textContent ?? ''
    if (!text) continue
    const matches = planAnnotationMatches(text, current)
    if (matches.length === 0) continue
    for (const m of matches) {
      const node = p.firstChild
      if (!node || node.nodeType !== Node.TEXT_NODE) continue
      try {
        const range = document.createRange()
        range.setStart(node, m.start)
        range.setEnd(node, m.end)
        const mark = document.createElement('mark')
        mark.className = 'novel-reading-mark'
        mark.dataset.annId = m.ann.id
        mark.dataset.annColor = m.ann.color
        mark.addEventListener('click', () => onOpen(m.ann))
        mark.appendChild(range.extractContents())
        range.insertNode(mark)
      } catch { /* ignore */ }
    }
  }
}
