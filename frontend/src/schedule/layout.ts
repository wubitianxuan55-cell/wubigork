/**
 * schedule/layout.ts — 时标分层布局（纯函数，横道图之外的两种网络图共用）
 *
 * 以任务/事件最早时间（es）为列（时间轴对齐，天然形成「时标网络图」口径），
 * 列内行序按相邻节点重心（barycentric）两轮松弛，减少连线交叉。
 */
export interface LayoutItem {
  id: string
  /** 时间列（最早时间，天） */
  es: number
}

export interface LayoutEdge {
  from: string
  to: string
}

export interface LayoutPos {
  col: number
  row: number
}

/**
 * 分层布局：返回每个 item 的（列, 行）。列 = es 去重升序序号；
 * 行 = 列内重心排序后的序号。孤立节点（无边）按列内出现顺序排。
 */
export function layerByTime(items: LayoutItem[], edges: LayoutEdge[]): Map<string, LayoutPos> {
  const pos = new Map<string, LayoutPos>()
  if (items.length === 0) return pos

  // 列划分：es 去重升序
  const esList = [...new Set(items.map((it) => it.es))].sort((a, b) => a - b)
  const colOfEs = new Map(esList.map((es, i) => [es, i]))

  const cols: string[][] = esList.map(() => [])
  const itemIds = new Set(items.map((it) => it.id))
  for (const it of items) cols[colOfEs.get(it.es)!].push(it.id)

  const inOf = new Map<string, string[]>()
  const outOf = new Map<string, string[]>()
  for (const id of itemIds) { inOf.set(id, []); outOf.set(id, []) }
  for (const e of edges) {
    if (!itemIds.has(e.from) || !itemIds.has(e.to) || e.from === e.to) continue
    outOf.get(e.from)!.push(e.to)
    inOf.get(e.to)!.push(e.from)
  }

  // 行序初始化：列内按入边源节点列号（重心初值），无边按原顺序
  const rowKey = new Map<string, number>()
  for (const col of cols) col.forEach((id, i) => rowKey.set(id, i))

  // 两轮重心松弛（左→右、右→左各一轮）
  for (let pass = 0; pass < 2; pass++) {
    const order = pass === 0 ? cols : [...cols].reverse()
    for (const col of order) {
      const keyed = col.map((id) => {
        const nbrs = [...(inOf.get(id) ?? []), ...(outOf.get(id) ?? [])]
        if (nbrs.length === 0) return { id, key: rowKey.get(id) ?? 0, tie: 0 }
        const avg = nbrs.reduce((s, n) => s + (rowKey.get(n) ?? 0), 0) / nbrs.length
        return { id, key: avg, tie: 0 }
      })
      keyed.sort((a, b) => a.key - b.key || a.tie - b.tie)
      keyed.forEach((k, i) => rowKey.set(k.id, i))
    }
  }

  for (const col of cols) {
    const ordered = [...col].sort((a, b) => (rowKey.get(a) ?? 0) - (rowKey.get(b) ?? 0))
    ordered.forEach((id, row) => pos.set(id, { col: colOfEs.get(items.find((it) => it.id === id)!.es)!, row }))
  }
  return pos
}
