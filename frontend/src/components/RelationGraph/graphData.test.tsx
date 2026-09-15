// graphData.test.tsx — 图谱数据构建纯函数单测（t5 收官 §7.5）。
// 锁：节点三类（角色/组织/职业+分组虚拟）、边四层顺序与强度、
// 组织成员 active 过滤与悬空引用防御、人际 pair 去重、
// 兜底强度（无结构边时人际 0.02→0.3）、副职业无名快照退 career_id。
import { describe, expect, it } from 'vitest'
import { buildGraphData, EDGE_CATEGORY_LABEL } from './graphData'

const colors = { careerMain: 'var(--color-primary)', careerSub: 'var(--color-warning)', group: 'var(--color-text-tertiary)' }

describe('buildGraphData（§7.5 三类节点四层边）', () => {
  const chars = [
    { id: 'c1', name: '林晚', role_type: 'protagonist', main_career_id: '剑修', main_career_stage: 3, sub_careers: [{ career_id: '炼丹师', career_name: '炼丹师', stage: 2 }] },
    { id: 'c2', name: '沈青', role_type: 'antagonist', main_career_id: '剑修', main_career_stage: 5 },
    { id: 'c3', name: '路人', role_type: 'minor' },
  ]
  const orgs = [
    { id: 'o1', name: '青云宗', member_list: [
      { character_id: 'c1', position: '内门弟子', status: 'active' },
      { character_id: 'c2', status: 'retired' }, // 非 active 过滤
      { character_id: 'ghost', status: 'active' }, // 悬空引用防御
    ] },
    { id: 'o2', name: '丹盟' }, // 无 member_list
  ]
  const rels = [
    { from_id: 'c1', to_id: 'c2', relation_type: 'rival' },
    { from_id: 'c2', to_id: 'c1', relation_type: 'rival' }, // 反向重复 → 去重
    { from_id: 'c1', to_id: 'c3', relation_type: 'friend' },
  ]

  it('节点：角色 3+组织 2+主职业 1+副职 1+两个分组虚拟', () => {
    const { nodes } = buildGraphData(chars, orgs, rels, colors)
    const kinds = nodes.map(n => n.kind)
    expect(kinds.filter(k => k === 'character').length).toBe(3)
    expect(kinds.filter(k => k === 'organization').length).toBe(2)
    expect(kinds.filter(k => k === 'career').length).toBe(2)
    expect(kinds.filter(k => k === 'career_group').length).toBe(2)
    const mainCareer = nodes.find(n => n.id === 'career-main-剑修')
    expect(mainCareer?.holders).toEqual(['林晚', '沈青'])
    expect(mainCareer?.stage).toBe(5) // 持有者最高阶
  })

  it('边四层：组织成员(含职位标签)/分组/主副职/人际去重', () => {
    const { edges } = buildGraphData(chars, orgs, rels, colors)
    const orgEdges = edges.filter(e => e.category === 'organization')
    expect(orgEdges.length).toBe(1) // retired 与 ghost 被过滤
    expect(orgEdges[0].label).toBe('组织成员·内门弟子')
    expect(orgEdges[0].dashed).toBe(true)
    expect(orgEdges[0].strength).toBe(0.5)

    expect(edges.filter(e => e.category === 'career_group').length).toBe(2)
    const mains = edges.filter(e => e.category === 'career_main')
    expect(mains.length).toBe(2) // 林晚+沈青 → 剑修
    expect(mains.every(e => e.strength === 0.3)).toBe(true)
    const subs = edges.filter(e => e.category === 'career_sub')
    expect(subs.length).toBe(1)
    expect(subs[0].dashed).toBe(true)
    expect(subs[0].to).toBe('林晚') // career-sub-炼丹师 → 林晚

    const inter = edges.filter(e => e.category === 'interpersonal')
    expect(inter.length).toBe(2) // rival 反向去重后 friend+rival
    expect(inter.every(e => e.strength === 0.02)).toBe(true)
  })

  it('兜底：无组织/职业数据时人际边强度回 0.3 参与布局', () => {
    const { edges, hasStructuralEdges } = buildGraphData(
      [{ id: 'a', name: '甲', role_type: 'protagonist' }, { id: 'b', name: '乙', role_type: 'minor' }],
      [], [{ from_id: 'a', to_id: 'b', relation_type: 'friend' }], colors,
    )
    expect(hasStructuralEdges).toBe(false)
    expect(edges[0].strength).toBe(0.3)
  })

  it('副职业无名快照退 career_id；分类标签表五类齐全', () => {
    const { nodes } = buildGraphData(
      [{ id: 'c1', name: '甲', role_type: 'minor', sub_careers: [{ career_id: 'cd-9', stage: 1 }] }],
      [], [], colors,
    )
    expect(nodes.find(n => n.id === 'career-sub-cd-9')?.name).toBe('cd-9')
    expect(Object.keys(EDGE_CATEGORY_LABEL).length).toBe(5)
  })
})
