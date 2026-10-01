/**
 * 章节节点树工具（T6-7.5 从 CreatePage 拆分）：构建带深度/子树的节点树，
 * 并做先序扁平化，供章节树面板渲染。
 */
import type { OutlineNode } from '../../../types'

export interface TreeNode {
  node: OutlineNode
  children: TreeNode[]
  depth: number
}

/** 构建节点树：{ node, children[], depth } */
export function buildTree(nodes: OutlineNode[]): TreeNode[] {
  const map = new Map<string, TreeNode>()
  const roots: TreeNode[] = []
  for (const n of nodes) {
    map.set(n.id, { node: n, children: [], depth: 0 })
  }
  for (const n of nodes) {
    const tn = map.get(n.id)!
    if (n.parent_id && map.has(n.parent_id)) {
      map.get(n.parent_id)!.children.push(tn)
    } else {
      roots.push(tn)
    }
  }
  // 设置深度
  const setDepth = (list: TreeNode[], d: number) => {
    for (const tn of list) {
      tn.depth = d
      setDepth(tn.children, d + 1)
    }
  }
  setDepth(roots, 0)
  return roots
}

/** 先序扁平化（父 → 子） */
export function flattenTree(roots: TreeNode[]): TreeNode[] {
  const result: TreeNode[] = []
  const walk = (list: TreeNode[]) => {
    for (const tn of list) {
      result.push(tn)
      walk(tn.children)
    }
  }
  walk(roots)
  return result
}

/** 章节展示标签：主线「第N章」，分支「第Na章」 */
export function chapterLabel(orderIndex: number, branch?: string): string {
  return branch ? `第${orderIndex}${branch}章` : `第${orderIndex}章`
}

/** 累计指定章节之前（含）的摘要，用于生成前文摘要注入 */
export function buildPrevSummary(outlines: OutlineNode[], upToChapter: number): string {
  const parts: string[] = []
  for (const n of [...outlines].sort((a, b) => (a.order_index || 0) - (b.order_index || 0))) {
    const cn = n.order_index || 0
    if (cn > 0 && cn <= upToChapter && n.summary) {
      parts.push(`第${cn}章：${n.summary.slice(0, 100)}`)
    }
  }
  return parts.join('\n\n')
}

/** 全树收集章节点（order_index>0；卷/分组节点 order 0 不算章），按章号升序。
 *  卷结构：章挂在卷的 children 下——只扫顶层会把卷当章（用户实盘事故）。 */
export function flattenChapters(outlines: OutlineNode[]): OutlineNode[] {
  const out: OutlineNode[] = []
  const walk = (ns: OutlineNode[]) => {
    for (const n of ns) {
      if ((n.order_index || 0) > 0) out.push(n)
      if (n.children?.length) walk(n.children)
    }
  }
  walk(outlines)
  return out.sort((a, b) => (a.order_index || 0) - (b.order_index || 0))
}

/**
 * 下一章=第一条从未生成过的章（status==='planned'）；全写过/无大纲顺延最大章号+1。
 * 两条实弹教训：只扫顶层会撞卷结构（算出 1 反复覆盖第1章）；chapter_file 判「未写」
 * 会误覆盖（ensureChapterNode 生成开始即写入，恒非空）——判据只用 status。
 */
export function nextToWriteChapter(outlines: OutlineNode[]): number {
  const chapters = flattenChapters(outlines)
  for (const c of chapters) {
    if ((c.status || 'planned') === 'planned') return c.order_index
  }
  return (chapters.length ? chapters[chapters.length - 1].order_index || 0 : 0) + 1
}
