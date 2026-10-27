import { useState } from 'react'
import { GetEntityRelations } from '../../../wailsjs/go/app/NovelB'

// useCreateGraph — 创作页全文脑图域（2026-10-27 大道至简分域拆件：
// CreatePage 67 个 useState 的第四族——实体关系弹窗的数据族 + 自包含加载函数
// openGraph；窄化函数与类型随迁，页面 EntityGraphSvg 渲染件不动）。

export interface EntityGraphNode { id?: string; name?: string; type?: string; group?: string | number }
export interface EntityGraphEdge { from?: string; to?: string; type?: string }
export interface EntityGraph { nodes: EntityGraphNode[]; edges: EntityGraphEdge[] }

/** 把后端未知负载窄化为 EntityGraph（畸形/非对象 → 空图）。 */
export function toEntityGraph(value: unknown): EntityGraph {
  if (typeof value === 'object' && value !== null) {
    const rec = value as Record<string, unknown>
    const nodes = Array.isArray(rec.nodes)
      ? rec.nodes.map((n) => (typeof n === 'object' && n !== null ? n as EntityGraphNode : {}))
      : []
    const edges = Array.isArray(rec.edges)
      ? rec.edges.map((e) => (typeof e === 'object' && e !== null ? e as EntityGraphEdge : {}))
      : []
    return { nodes, edges }
  }
  return { nodes: [], edges: [] }
}

export function useCreateGraph() {
  const [graphOpen, setGraphOpen] = useState(false)
  const [graphBusy, setGraphBusy] = useState(false)
  const [graphData, setGraphData] = useState<EntityGraph | null>(null)
  const [graphMsg, setGraphMsg] = useState('')

  const openGraph = async () => {
    setGraphOpen(true)
    setGraphBusy(true)
    setGraphMsg('')
    try {
      const g = toEntityGraph(await GetEntityRelations())
      setGraphData(g)
      setGraphMsg(g.nodes.length > 0 ? `共 ${g.nodes.length} 个实体 · ${g.edges.length} 条关系` : '')
    } catch (err: unknown) {
      setGraphData(null)
      setGraphMsg(err instanceof Error ? err.message : '加载实体关系失败')
    } finally {
      setGraphBusy(false)
    }
  }

  return { graphOpen, setGraphOpen, graphBusy, graphData, graphMsg, openGraph }
}
