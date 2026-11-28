// proposal-library/logic.ts — 方案库板块的纯逻辑（零 DOM、零绑定）：字数格式、
// 标题树的可见行/折叠/祖先/筛选、命中摘要的【】切分与检索词高亮、语义索引进度文案、
// 编号与层级核对、结构操作的可用性与回执、拆分预览、命中来源、索引体检结论。
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

// ── 2026-11-28 索引与编辑优化批 ──────────────────────────────────────
// 标题树筛选、编号与层级核对、结构操作的可用性与回执、拆分预览、字数变化、命中来源。

export interface TitledNode extends TreeNode {
  title: string
  titlePath: string
}

/** 把文本按检索词切成片段（命中的词给 hit=true），大小写不敏感；没有命中就是整段一条。 */
export function splitByTerms(text: string, terms: string[]): SnippetPart[] {
  const list = terms.map((t) => t.trim()).filter(Boolean)
  if (!list.length) return text ? [{ text, hit: false }] : []
  const low = text.toLowerCase()
  const spans: { start: number; end: number }[] = []
  for (const t of list) {
    const needle = t.toLowerCase()
    for (let at = low.indexOf(needle); at >= 0; at = low.indexOf(needle, at + needle.length)) {
      spans.push({ start: at, end: at + needle.length })
    }
  }
  spans.sort((a, b) => a.start - b.start || b.end - a.end)
  const out: SnippetPart[] = []
  let last = 0
  for (const s of spans) {
    if (s.end <= last) continue // 被前一个命中盖住的短词不重复标
    if (s.start > last) out.push({ text: text.slice(last, s.start), hit: false })
    out.push({ text: text.slice(s.start, s.end), hit: true })
    last = s.end
  }
  if (last < text.length) out.push({ text: text.slice(last), hit: false })
  return out
}

/** 检索词（空白分开，全部出现才算命中）。 */
export function queryTerms(keyword: string): string[] {
  return keyword.trim().toLowerCase().split(/\s+/).filter(Boolean)
}

/** 标题树筛选：留下标题或标题路径含全部检索词的节点，连同它们的全部祖先（不然深层节点看不见上下文）。 */
export function filterTree<T extends TitledNode>(outline: T[], keyword: string): Set<number> {
  const keep = new Set<number>()
  const terms = queryTerms(keyword)
  if (!terms.length) return keep
  outline.forEach((n, i) => {
    const hay = `${n.title}\n${n.titlePath}`.toLowerCase()
    if (!terms.every((t) => hay.includes(t))) return
    keep.add(n.id)
    let level = n.level
    for (let j = i - 1; j >= 0 && level > 1; j--) {
      if (outline[j].level < level) {
        keep.add(outline[j].id)
        level = outline[j].level
      }
    }
  })
  return keep
}

/** 标题开头编号的层数：点分数字（「1.4.7.2」是 4 级写法）、「第X章 / 篇 / 部分」（1 级）。
 *  没有编号、或「一、」「（一）」这类只在章内起算的序号返回 0——判断不了层级。 */
export function numberDepth(title: string): number {
  const t = title.trim()
  const dotted = t.match(/^(\d{1,4}(?:\.\d{1,4})+)(?![\d.])/)
  if (dotted) return dotted[1].split(".").length
  if (/^第[一二三四五六七八九十百千零〇两\d]+(章|篇|部分|部|卷|编|册)/.test(t)) return 1
  return 0
}

/** 编号与层级不符：调过层级 / 挪过位置后，标题里照录的编号还是原来的层数。 */
export function numberMismatch(title: string, level: number): boolean {
  const depth = numberDepth(title)
  return depth > 0 && depth !== level
}

export function mismatchHint(title: string, level: number): string {
  return `编号与层级不符：标题里的编号是 ${numberDepth(title)} 级写法，现在在第 ${level} 层。可以改标题里的编号，或用「调整结构」改回去`
}

/** nodes[i] 的子树结束下标（不含）：其后第一个层级不深于它的节点。 */
export function subtreeEnd<T extends TreeNode>(outline: T[], i: number): number {
  for (let j = i + 1; j < outline.length; j++) if (outline[j].level <= outline[i].level) return j
  return outline.length
}

/** nodes[i] 上一个同级小节的下标；已是本层第一个时为 -1。 */
export function prevSiblingIndex<T extends TreeNode>(outline: T[], i: number): number {
  for (let k = i - 1; k >= 0; k--) {
    if (outline[k].level === outline[i].level) return k
    if (outline[k].level < outline[i].level) break
  }
  return -1
}

/** 某个节点能做哪些结构操作（与后端 structure.go 的判定同口径；后端会再校验一遍）。 */
export interface StructureFlags {
  canPromote: boolean
  canDemote: boolean
  canMoveUp: boolean
  canMoveDown: boolean
  /** 并入父节只许第一个子节：阅读顺序才不变 */
  canMergeParent: boolean
  canMergePrev: boolean
  canAddChild: boolean
  childCount: number
  subtreeCount: number
  subtreeChars: number
}

const NO_STRUCTURE: StructureFlags = {
  canPromote: false, canDemote: false, canMoveUp: false, canMoveDown: false,
  canMergeParent: false, canMergePrev: false, canAddChild: false,
  childCount: 0, subtreeCount: 0, subtreeChars: 0,
}

export function structureFlags<T extends TreeNode & { chars: number }>(outline: T[], id: number): StructureFlags {
  const i = indexOf(outline, id)
  if (i < 0) return NO_STRUCTURE
  const node = outline[i]
  const end = subtreeEnd(outline, i)
  const prev = prevSiblingIndex(outline, i)
  const subtree = outline.slice(i, end)
  return {
    canPromote: node.level > 1,
    canDemote: node.level < 4 && prev >= 0,
    canMoveUp: prev >= 0,
    canMoveDown: end < outline.length && outline[end].level === node.level,
    canMergeParent: node.level > 1 && prev < 0,
    canMergePrev: prev >= 0,
    canAddChild: node.level < 4,
    childCount: subtree.filter((c) => c.level === node.level + 1).length,
    subtreeCount: end - i,
    subtreeChars: subtree.reduce((n, c) => n + c.chars, 0),
  }
}

export interface SplitPreview {
  /** 交给后端的切点：正文的 Unicode 码点数（后端按 rune 数） */
  offset: number
  head: string
  tail: string
  /** 拆出去那一段的第一行，给新小节标题当底稿 */
  suggest: string
}

/** 拆分预览：光标所在行及之后切成新小节。光标在第一行、或之后没有正文时返回 null。 */
export function splitPreview(body: string, cursor: number): SplitPreview | null {
  let cut = Math.max(0, Math.min(cursor, body.length))
  while (cut > 0 && body[cut - 1] !== "\n") cut -= 1
  if (cut <= 0) return null
  const tail = body.slice(cut).replace(/^\n+/, "")
  if (!tail.trim()) return null
  const first = tail.split("\n").find((line) => line.trim())?.trim() ?? ""
  const suggest = first.replace(/^[#>*+\-\s|]+/, "").slice(0, 40).trim()
  return {
    offset: Array.from(body.slice(0, cut)).length,
    head: body.slice(0, cut).replace(/[\n \t\r]+$/, ""),
    tail,
    suggest,
  }
}

/** 字数变化提示（码点数）：没改返回空串。 */
export function changeLabel(before: string, after: string): string {
  if (before === after) return ""
  const a = Array.from(before).length
  const b = Array.from(after).length
  if (a === b) return "字数没变，内容有改动"
  return b > a ? `+${b - a} 字` : `−${a - b} 字`
}

/** 命中来源的短标签：全文路 / 语义路 / 两路都有。 */
export function sourceLabel(source?: string): string {
  if (source === "both") return "全文+语义"
  if (source === "semantic") return "语义"
  if (source === "keyword") return "全文"
  return ""
}

/** 结构操作回执的一句话。 */
export function structureReceipt(op: string, res: { affected: number; created: number; removed: number }): string {
  // 升降级时受影响的小节里含本节自己；其余跟着改的是标题路径。
  const follow = (op === "promote" || op === "demote") && res.affected > 1 ? `，${res.affected - 1} 个小节的标题路径已同步` : ""
  const numbering = "标题里的编号没有改，与新层级不符时标题旁会有提示"
  switch (op) {
    case "promote":
      return `已升级${follow}。${numbering}`
    case "demote":
      return `已降级，归到上一同级小节名下${follow}。${numbering}`
    case "moveUp":
      return `已上移。${numbering}`
    case "moveDown":
      return `已下移。${numbering}`
    case "split":
      return "已拆出新小节"
    case "mergeParent":
    case "mergePrev":
      return "已并入，正文接在后面"
    case "addChild":
    case "addSibling":
      return res.created > 0 ? "已新增小节，可以开始写正文" : "没有变化"
    case "remove":
      return res.removed > 1 ? `已删除本节及 ${res.removed - 1} 个子节` : "已删除本节"
    default:
      return "已完成"
  }
}

/** 导出用的文件名：去掉文件系统不认的字符。 */
export function safeFileName(title: string): string {
  const name = title.replace(/[\\/:*?"<>|\r\n]+/g, " ").replace(/\s+/g, " ").trim().slice(0, 80)
  return name || "方案"
}

/** 索引体检的一行结论：没问题不显示；有问题列出条目；只剩向量没补齐时说明进度。 */
export function indexSummary(status: {
  vecDone: number; vecTotal: number; orphanFts: number; missingFts: number; staleFts: number; missingChunks: number; staleChunks: number
}): { broken: boolean; lines: string[] } {
  const lines: string[] = []
  if (status.orphanFts > 0) lines.push(`${status.orphanFts} 条全文索引对应的小节已不存在`)
  if (status.missingFts > 0) lines.push(`${status.missingFts} 个小节缺全文索引`)
  if (status.staleFts > 0) lines.push(`${status.staleFts} 条全文索引内容过期`)
  if (status.missingChunks > 0) lines.push(`${status.missingChunks} 个小节缺语义块`)
  if (status.staleChunks > 0) lines.push(`${status.staleChunks} 个语义块的标题路径过期`)
  return { broken: lines.length > 0, lines }
}
