// proposal-library/logic.ts — 方案库板块的纯逻辑（零 DOM、零绑定）：字数格式、
// 标题树的可见行/折叠/祖先、命中摘要的【】切分、语义索引进度文案。
// 页面组件 ProposalLibraryPage.tsx 只负责取数与渲染。

/** 字数的人读形式：不足一万显示「N 字」，过万折成「x.x 万字」。 */
export function formatChars(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return "0 字"
  if (n < 10000) return `${n} 字`
  return `${(n / 10000).toFixed(1)} 万字`
}

/** 语义索引进度：全部完成不显示，未完成提示还差多少（检索时自动补齐）。 */
export function indexLabel(done: number, total: number): string {
  if (!total || total <= 0) return ""
  if (done >= total) return ""
  if (done <= 0) return "语义索引待建立"
  return `语义索引 ${done}/${total}`
}

export interface TreeNode {
  id: number
  level: number
}

/** 节点 id → 在 outline 里的下标。 */
export function indexById<T extends TreeNode>(outline: T[]): Map<number, number> {
  const m = new Map<number, number>()
  outline.forEach((n, i) => m.set(n.id, i))
  return m
}

/** 某节点是否有直接子节（紧随其后出现更深一级的标题）。 */
export function hasChildren<T extends TreeNode>(outline: T[], id: number): boolean {
  const i = indexOf(outline, id)
  if (i < 0) return false
  return i + 1 < outline.length && outline[i + 1].level === outline[i].level + 1
}

/** 某节点的父节点 id（往前找最近的更高级标题）；一级标题返回 null。 */
export function parentOf<T extends TreeNode>(outline: T[], id: number): number | null {
  const i = indexOf(outline, id)
  if (i < 0) return null
  for (let j = i - 1; j >= 0; j--) {
    if (outline[j].level < outline[i].level) return outline[j].id
  }
  return null
}

function indexOf<T extends TreeNode>(outline: T[], id: number): number {
  for (let i = 0; i < outline.length; i++) if (outline[i].id === id) return i
  return -1
}

/** 某节点的全部祖先 id（自近及远）。 */
export function ancestorsOf<T extends TreeNode>(outline: T[], id: number): number[] {
  const out: number[] = []
  let cur = parentOf(outline, id)
  let guard = 0
  while (cur != null && guard++ < 64) {
    out.push(cur)
    cur = parentOf(outline, cur)
  }
  return out
}

/** 初始折叠集：章（1 级）展开到节，2 级及更深的有子节者收起——几百个节点的
 *  标题树首屏只露两级，不至于一进来就是一面墙。 */
export function initialCollapsed<T extends TreeNode>(outline: T[]): Set<number> {
  const collapsed = new Set<number>()
  for (let i = 0; i < outline.length - 1; i++) {
    if (outline[i].level >= 2 && outline[i + 1].level === outline[i].level + 1) collapsed.add(outline[i].id)
  }
  return collapsed
}

/** 在折叠集基础上展开某节点的全部祖先（定位到深处的小节时要能看见它）。 */
export function expandAncestors<T extends TreeNode>(collapsed: ReadonlySet<number>, outline: T[], id: number): Set<number> {
  const ancestors = ancestorsOf(outline, id)
  if (!ancestors.length) return new Set(collapsed)
  const next = new Set(collapsed)
  for (const a of ancestors) next.delete(a)
  return next
}

export interface VisibleRow<T> {
  node: T
  /** 缩进层数 = level - 1 */
  depth: number
}

/** 标题树的可见行：被折叠节点的全部后代跳过；顺序即文档顺序。 */
export function visibleRows<T extends TreeNode>(outline: T[], collapsed: ReadonlySet<number>): VisibleRow<T>[] {
  const out: VisibleRow<T>[] = []
  let hiddenBelow = 0
  for (const node of outline) {
    if (hiddenBelow && node.level > hiddenBelow) continue
    hiddenBelow = 0
    out.push({ node, depth: node.level - 1 })
    if (collapsed.has(node.id)) hiddenBelow = node.level
  }
  return out
}

/** 键盘上下移：可见行里相对当前行的目标行（到头停住）。 */
export function stepRow<T extends TreeNode>(rows: VisibleRow<T>[], currentId: number | null, delta: 1 | -1): number | null {
  if (!rows.length) return null
  const idx = rows.findIndex((r) => r.node.id === currentId)
  if (idx < 0) return rows[0].node.id
  return rows[Math.min(rows.length - 1, Math.max(0, idx + delta))].node.id
}

export interface SnippetPart {
  text: string
  hit: boolean
}

/** 把后端的命中摘要按【】切成片段：括号内的是命中词，界面渲染成高亮。 */
export function splitSnippet(snippet: string): SnippetPart[] {
  const out: SnippetPart[] = []
  const re = /【([^【】]*)】/g
  let last = 0
  for (let m = re.exec(snippet); m; m = re.exec(snippet)) {
    if (m.index > last) out.push({ text: snippet.slice(last, m.index), hit: false })
    out.push({ text: m[1], hit: true })
    last = m.index + m[0].length
  }
  if (last < snippet.length) out.push({ text: snippet.slice(last), hit: false })
  return out
}

/** 检索回执的模式/备注拼成一句给人看的说明；模型不可用时的降级说明原样带出。 */
export function searchSummary(mode: string, note: string): string {
  const label = mode === "title" ? "标题树" : mode === "hybrid" ? "全文 + 语义" : mode === "keyword" ? "全文" : ""
  return [label, note].filter(Boolean).join(" · ")
}
