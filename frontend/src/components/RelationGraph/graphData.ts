// graphData.ts — 关系图谱的数据构建纯函数（t5 收官 §7.5：节点三类+边四层）。
// 从组件抽出以便单测钉死：MuMu :869-969 的分层语义 + gaea active 过滤增量。
// 布局权重映射 d3 strength：组织成员 8 → 0.5，职业分组 4 → 0.35，
// 主职业 3 → 0.3，副职业 2 → 0.2，人际 1 → 0.02（只渲染基本不布局，
// MuMu :971-974「人际边不参与布局」；无组织/职业边时人际退 0.3 兜底布局）。

export interface CareerRefLite {
  career_id: string
  career_name?: string
  stage: number
}

export interface GraphCharacter {
  id: string
  name: string
  role_type: string
  main_career_id?: string
  main_career_stage?: number
  sub_careers?: CareerRefLite[]
}

export interface GraphOrganization {
  id: string
  name: string
  member_list?: { character_id: string; position?: string; status?: string }[]
}

export type EdgeCategory = 'organization' | 'career_group' | 'career_main' | 'career_sub' | 'interpersonal'

export const EDGE_CATEGORY_LABEL: Record<EdgeCategory, string> = {
  organization: '组织成员',
  career_group: '职业分组',
  career_main: '主职业',
  career_sub: '副职业',
  interpersonal: '人际关系',
}

export interface GraphNode {
  id: string
  name: string
  kind: 'character' | 'organization' | 'career' | 'career_group'
  color: string
  // 详情浮层载荷（kind 各异）
  role_type?: string
  main_career_id?: string
  main_career_stage?: number
  sub_careers?: CareerRefLite[]
  members?: string[]          // 组织：成员名列表
  holders?: string[]          // 职业：持有角色名列表
  stage?: number              // 职业主阶（career-main 节点取持有者最大阶）
}

export interface GraphEdge {
  from: string
  to: string
  label: string
  category: EdgeCategory
  dashed?: boolean
  strength: number
}

const GROUP_MAIN = '__career_group_main__'
const GROUP_SUB = '__career_group_sub__'

/** 组织成员：仅 active/缺省语义（与 v4.308 回灌区段同一过滤口径） */
function isActiveMember(status?: string): boolean {
  return !status || status === 'active'
}

/** 构建三类节点+四层边；人际边 pair 去重（MuMu 无向语义）。 */
export function buildGraphData(
  characters: GraphCharacter[],
  organizations: GraphOrganization[],
  relationships: { from_id: string; to_id: string; relation_type: string }[],
  colorOf: { careerMain: string; careerSub: string; group: string },
): { nodes: GraphNode[]; edges: GraphEdge[]; hasStructuralEdges: boolean } {
  const nodes: GraphNode[] = []
  const edges: GraphEdge[] = []
  const charById = new Map(characters.map(c => [c.id, c]))
  const careerMain = new Map<string, { name: string; holders: string[]; maxStage: number }>()
  const careerSub = new Map<string, { name: string; holders: string[] }>()

  // ── 组织成员边（第一层，最优先）──
  for (const org of organizations) {
    nodes.push({ id: org.id, name: org.name, kind: 'organization', color: colorOf.group })
    for (const m of org.member_list || []) {
      if (!isActiveMember(m.status)) continue
      if (!charById.has(m.character_id)) continue // 悬空引用防御
      edges.push({
        from: org.id, to: m.character_id,
        label: m.position ? `组织成员·${m.position}` : '组织成员',
        category: 'organization', dashed: true, strength: 0.5,
      })
    }
  }

  // ── 职业节点与边（第二/三层）──
  for (const c of characters) {
    if (c.main_career_id) {
      const entry = careerMain.get(c.main_career_id) || { name: c.main_career_id, holders: [], maxStage: 1 }
      entry.holders.push(c.name)
      entry.maxStage = Math.max(entry.maxStage, c.main_career_stage || 1)
      careerMain.set(c.main_career_id, entry)
    }
    for (const sc of c.sub_careers || []) {
      const name = sc.career_name || sc.career_id
      const entry = careerSub.get(sc.career_id) || { name, holders: [] }
      entry.holders.push(c.name)
      careerSub.set(sc.career_id, entry)
    }
  }
  if (careerMain.size > 0) {
    nodes.push({ id: GROUP_MAIN, name: '主职业', kind: 'career_group', color: colorOf.group })
  }
  if (careerSub.size > 0) {
    nodes.push({ id: GROUP_SUB, name: '副职业', kind: 'career_group', color: colorOf.group })
  }
  for (const [id, e] of careerMain) {
    const nid = `career-main-${id}`
    nodes.push({ id: nid, name: e.name, kind: 'career', color: colorOf.careerMain, holders: e.holders, stage: e.maxStage })
    edges.push({ from: GROUP_MAIN, to: nid, label: '职业分类·主', category: 'career_group', dashed: true, strength: 0.35 })
    for (const h of e.holders) edges.push({ from: nid, to: h, label: `主职业·${e.name}`, category: 'career_main', strength: 0.3 })
  }
  for (const [id, e] of careerSub) {
    const nid = `career-sub-${id}`
    nodes.push({ id: nid, name: e.name, kind: 'career', color: colorOf.careerSub, holders: e.holders })
    edges.push({ from: GROUP_SUB, to: nid, label: '职业分类·副', category: 'career_group', dashed: true, strength: 0.35 })
    for (const h of e.holders) edges.push({ from: nid, to: h, label: `副职业·${e.name}`, category: 'career_sub', dashed: true, strength: 0.2 })
  }

  // ── 人际边（第四层，最后；pair 去重）──
  const seen = new Set<string>()
  for (const r of relationships) {
    const key = [r.from_id, r.to_id].sort().join('-')
    if (seen.has(key)) continue
    seen.add(key)
    edges.push({ from: r.from_id, to: r.to_id, label: r.relation_type, category: 'interpersonal', strength: 0.02 })
  }

  // ── 角色节点 ──
  for (const c of characters) {
    nodes.push({
      id: c.id, name: c.name, kind: 'character', color: '',
      role_type: c.role_type, main_career_id: c.main_career_id,
      main_career_stage: c.main_career_stage, sub_careers: c.sub_careers,
    })
  }

  const hasStructuralEdges = edges.some(e => e.category !== 'interpersonal')
  // 兜底（MuMu :972）：无组织/职业边时人际边恢复常规强度参与布局
  if (!hasStructuralEdges) {
    for (const e of edges) if (e.category === 'interpersonal') e.strength = 0.3
  }
  return { nodes, edges, hasStructuralEdges }
}
