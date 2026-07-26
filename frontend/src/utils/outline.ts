import type { OutlineNode } from '../types'

/** 大纲树通用工具 */

/** 按 order_index 递归排序大纲树 */
export function sortNodes(nodes: OutlineNode[]): OutlineNode[] {
  return [...nodes].sort((a, b) => a.order_index - b.order_index).map((n) => ({
    ...n,
    children: n.children ? sortNodes(n.children) : undefined,
  }))
}

/** 递归提取所有叶子节点（按 order_index 排序） */
export function findAllLeaves(nodes: OutlineNode[]): OutlineNode[] {
  const result: OutlineNode[] = []
  function walk(list: OutlineNode[]) {
    for (const n of list) {
      if (n.children && n.children.length > 0) {
        walk(n.children)
      } else {
        result.push(n)
      }
    }
  }
  walk(nodes)
  return result.sort((a, b) => (a.order_index || 0) - (b.order_index || 0))
}
