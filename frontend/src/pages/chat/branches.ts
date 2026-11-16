// branches.ts — 消息分支版本树前端纯函数（A 批，chat-demo branches.ts 语义对齐；
// 设计文档 docs/gaea-chat-branch-tree-design-2026-11.md，Go 侧同构实现
// internal/chat/branches.go——两侧算法须保持一致，改一处必对另一处）。
//
// 数据：分支字段在 Message.extra.bt（p=父消息 id，c=选中子指针挂父上）；
// db 行 key 形如 db_<id>（Go 落库回填），本地乐观行（流式对）无 db id——
// 作为「pending 行」插在锚点（branchAnchorKey）之后并截断旧链（编辑提问/
// 重生成的流中视图），无锚点时追加在路径尾（普通发送续叶）。
import type { ChatMsg } from './types'

export interface MsgBT { p: number; c: number }
export interface MsgVersions { ids: number[]; index: number }

/** db_ 前缀行 → Go 消息 id；本地行返回 null */
export function dbIdOf(key: string): number | null {
  const m = /^db_(\d+)$/.exec(key)
  return m ? Number(m[1]) : null
}

/** extra.bt 读取；缺失/非法返回 null（=线形语义） */
export function btOf(msg: ChatMsg): MsgBT | null {
  const bt = msg.extra?.bt
  if (typeof bt !== 'object' || bt === null) return null
  const p = (bt as { p?: unknown }).p
  const c = (bt as { c?: unknown }).c
  if (typeof p !== 'number' || !Number.isSafeInteger(p) || p < 0) return null
  if (typeof c !== 'number' || !Number.isSafeInteger(c) || c < 0) return null
  return { p, c }
}

/** 线形话题判定：任一 db 行缺 bt（Go BTNeeded 同构） */
function btNeeded(rows: ChatMsg[]): boolean {
  return rows.some(m => btOf(m) === null)
}

/**
 * 可见路径（叶子回溯，Go VisiblePath 同构）：
 * 候选叶子 = 无子（bt.c===0）的 db 行，按 seq 新→旧取第一个回溯链全过
 * 选中校验的；线形话题直通。FE 行（流式/乐观）按 branchAnchorKey 插入
 * 或追加在尾。
 */
export function buildVisiblePath(messages: ChatMsg[], branchAnchorKey?: string | null): ChatMsg[] {
  const dbRows = messages.filter(m => dbIdOf(m.key) !== null)
  const feRows = messages.filter(m => dbIdOf(m.key) === null)
  let path: ChatMsg[]
  if (btNeeded(dbRows)) {
    path = dbRows
  } else {
    const byId = new Map<number, ChatMsg>()
    for (const m of dbRows) byId.set(dbIdOf(m.key) as number, m)
    // 回溯链全过选中校验：链上每个节点的父 bt.c 都指向它；父缺失（更早分页）
    // 按可见截断。
    const chainVisible = (cur: number): boolean => {
      for (let i = 0; cur !== 0 && i <= dbRows.length; i++) {
        const m = byId.get(cur)
        if (!m) return true
        const bt = btOf(m)
        if (!bt || bt.p === 0) return true
        const parent = byId.get(bt.p)
        if (!parent) return true
        const pbt = btOf(parent)
        if (!pbt || pbt.c !== cur) return false
        cur = bt.p
      }
      return true
    }
    let leafId = 0
    for (let i = dbRows.length - 1; i >= 0; i--) {
      const bt = btOf(dbRows[i])
      if (!bt || bt.c !== 0) continue
      if (chainVisible(dbIdOf(dbRows[i].key) as number)) {
        leafId = dbIdOf(dbRows[i].key) as number
        break
      }
    }
    path = []
    let cur = leafId
    while (cur !== 0) {
      const m = byId.get(cur)
      if (!m) break
      path.push(m)
      const bt = btOf(m)
      if (!bt) break
      cur = bt.p
    }
    path.reverse()
  }
  if (branchAnchorKey) {
    const idx = path.findIndex(m => m.key === branchAnchorKey)
    if (idx >= 0) return [...path.slice(0, idx + 1), ...feRows]
    return [...path, ...feRows] // 锚点不在已加载窗口（更早分页）→ 退化为尾部追加
  }
  return [...path, ...feRows]
}

/**
 * 兄弟版本分组（Go/DeepSeek buildMessageVersions 同构）：db 行按 bt.p 分组
 * （缺 bt 行自成单元素组，永不触发切换器），组内按 id 升序 ≈ 创建序。
 * key → 该行的 {ids 兄弟消息 id 列表, index 本行位次}。
 */
export function buildVersions(messages: ChatMsg[]): Map<string, MsgVersions> {
  const groups = new Map<string, number[]>()
  const idOf = new Map<string, number>()
  for (const m of messages) {
    const id = dbIdOf(m.key)
    if (id === null) continue
    idOf.set(m.key, id)
    const bt = btOf(m)
    const key = bt ? `p${bt.p}` : `solo-${id}`
    const arr = groups.get(key)
    if (arr) arr.push(id)
    else groups.set(key, [id])
  }
  const out = new Map<string, MsgVersions>()
  for (const [key, id] of idOf) {
    const bt = btOf(messages.find(m => m.key === key) as ChatMsg)
    const gkey = bt ? `p${bt.p}` : `solo-${id}`
    const ids = [...(groups.get(gkey) ?? [id])].sort((a, b) => a - b)
    out.set(key, { ids, index: ids.indexOf(id) })
  }
  return out
}
