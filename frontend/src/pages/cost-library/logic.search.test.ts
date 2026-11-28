// logic.search.test.ts — 造价数据库纯函数层（2026-11-28 索引优化）。
// 覆盖：多词搜索（空白分词、全含）、分面计数随搜索联动、状态筛选、定额地区筛选、
// 「待整理」逐项点开的落点、关键词高亮切分。
import { describe, expect, it } from 'vitest'
import type { WorkcostQuota, WorkcostResource } from '../../gaea/lib/types'
import {
  NONE, OUTSIDE, STATE_LABELS, billSpecialties, categoryTreeOf, chaptersOf, divisionsOf, emptyQuotaFilters, emptyResourceFilters,
  filterBill, filterQuotas, filterResources, keywordTerms, kindsOf, matchKeyword, pendingFilter, periodsOf, projectCounts,
  quotaRegionsOf, regionsOf, splitByKeyword, specialtiesOf, type BillRow, type FilterContext,
} from './logic'

function row(patch: Partial<BillRow>): BillRow {
  return {
    id: 0, projectId: 1, projectName: 'P1', code: '', title: '', unit: '', division: '', quantity: 0,
    quantityExpr: '', quotaCode: '', feature: '', priceOverride: 0, sort: 0, quotaCodes: [], ...patch,
  }
}

const rows: BillRow[] = [
  row({ id: 11, code: 'WP01', title: 'C25 混凝土垫层', division: 'A基础', quotaCode: 'Q1', quotaCodes: ['Q1', 'Q9'], specialty: '土壤修复' }),
  row({ id: 12, code: 'WP02', title: '场地平整', division: 'A临建', quotaCode: 'Q3', quotaCodes: ['Q3'], specialty: '土壤修复' }),
  row({ id: 13, projectId: 2, projectName: 'P2', code: 'WP03', title: 'C25 混凝土垫层', division: 'A基础', quotaCode: 'Q1', quotaCodes: ['Q1'], specialty: '市政' }),
  row({ id: 14, projectId: 2, code: 'M001', title: '外委检测', division: 'D外委' }), // 手填行、专业未标注
]

const quotas = [
  { code: 'Q1', title: '混凝土垫层', specialty: '土壤修复', chapter: '基础', region: '四川省' },
  { code: 'Q2', title: '砌砖墙', specialty: '建筑工程', chapter: '', region: '' },
  { code: 'Q3', title: '场地平整', specialty: '土壤修复', chapter: '基础', region: '' },
].map((q, i) => ({ id: i + 1, unit: 'm3', ...q })) as unknown as WorkcostQuota[]

const resources = [
  { code: 'C25', kind: '材料', title: 'C25 商品混凝土', spec: '泵送', categoryPath: '材料/混凝土', region: '四川省', priceDate: '2026-11', basePrice: 345, currentPrice: 360 },
  { code: 'SAND', kind: '材料', title: '中砂', spec: '', categoryPath: '材料/砂石', region: '四川省', priceDate: '2026-11', basePrice: 120, currentPrice: 0 },
  { code: 'ZERO', kind: '材料', title: '待询价', spec: '', categoryPath: '', region: '', priceDate: '', basePrice: 0, currentPrice: 0 },
  { code: 'W1', kind: '措施费', title: '说明行', spec: '', categoryPath: '', region: '', priceDate: '', basePrice: 1, currentPrice: 1 },
].map((r, i) => ({ id: i + 1, ...r })) as unknown as WorkcostResource[]

const noBill = { projectId: 0, division: '', specialty: '', keyword: '', grouped: false }
const noQuota = { specialty: '', chapter: '', keyword: '' }
const noRes = { kind: '', category: '', region: '', period: '', keyword: '' }
const names = (list: { name: string }[]): string[] => list.map((f) => f.name)

describe('多词搜索（空白分词，全含）', () => {
  it('词条：去首尾空白、忽略大小写、去重、保序', () => {
    expect(keywordTerms('  C25   混凝土 c25 ')).toEqual(['c25', '混凝土'])
    expect(keywordTerms('')).toEqual([])
  })

  it('多个词要求每个词都命中，一个词落在任一字段里算命中', () => {
    expect(matchKeyword('C25 混凝土', 'C25商品混凝土')).toBe(true)
    expect(matchKeyword('C25 混凝土', 'c25', undefined, '商品混凝土')).toBe(true) // 词分别落在编码与名称
    expect(matchKeyword('C25 混凝土', 'C25 砂浆')).toBe(false)
    expect(matchKeyword('混凝土', 'C25 混凝土')).toBe(true) // 单词照旧是子串匹配
  })

  it('高亮切分：命中片段与未命中片段交替给出，拼回去是原文', () => {
    const parts = splitByKeyword('c25 混凝土', 'C25商品混凝土垫层')
    expect(parts.filter((p) => p.hit).map((p) => p.text)).toEqual(['C25', '混凝土'])
    expect(parts.map((p) => p.text).join('')).toBe('C25商品混凝土垫层')
    expect(splitByKeyword('', '随便')).toEqual([{ text: '随便', hit: false }])
    expect(splitByKeyword('没有的词', '随便')).toEqual([{ text: '随便', hit: false }])
    expect(splitByKeyword('词', undefined)).toEqual([{ text: '', hit: false }])
  })
})

describe('分面计数随搜索联动', () => {
  it('清单：专业与分部的计数只数命中关键词的行；项目列表给每个项目剩多少', () => {
    // 同数时按名称排序（这里不关心先后，只看是哪两个）
    expect(names(billSpecialties(rows, { keyword: '混凝土' })).sort()).toEqual(['市政', '土壤修复'].sort())
    expect(billSpecialties(rows, { keyword: '混凝土' }).map((f) => f.count)).toEqual([1, 1])
    // 没搜的时候「未标注」单列一桶
    expect(names(billSpecialties(rows))).toEqual(['土壤修复', '市政', NONE])
    expect(divisionsOf(rows, { keyword: '混凝土' })).toEqual(['A基础'])
    expect([...projectCounts(rows, { keyword: '混凝土' }).entries()]).toEqual([[1, 1], [2, 1]])
    expect(projectCounts(rows, { keyword: '外委' }).has(1)).toBe(false)
  })

  it('这一维自己的筛选不套进计数：选了一个专业，别的专业仍可选', () => {
    // billSpecialties 只套项目 / 分部 / 关键词 / 状态，专业筛选值不进参数
    expect(names(billSpecialties(rows, { keyword: '' }))).toEqual(['土壤修复', '市政', NONE])
    // 分部只在专业内有意义：土壤修复下的两条都在「基础」，没有未归的
    expect(names(chaptersOf(quotas, '土壤修复'))).toEqual(['基础'])
    expect(names(chaptersOf(quotas))).toEqual(['基础', NONE])
    expect(names(chaptersOf(quotas, '土壤修复', '垫层'))).toEqual(['基础'])
  })

  it('定额：专业与地区计数随关键词走', () => {
    expect(names(specialtiesOf(quotas, { keyword: '垫层' }))).toEqual(['土壤修复'])
    expect(names(specialtiesOf(quotas))).toEqual(['土壤修复', '建筑工程'])
    expect(names(quotaRegionsOf(quotas))).toEqual(['四川省', NONE])
    expect(quotaRegionsOf(quotas).find((f) => f.name === NONE)?.count).toBe(2)
    expect(names(quotaRegionsOf(quotas, { keyword: '砖' }))).toEqual([NONE])
  })

  it('工料机：类别、分类、地区、期数的计数随关键词走', () => {
    expect(names(regionsOf(resources, { keyword: '砂' }))).toEqual(['四川省'])
    expect(names(regionsOf(resources))).toEqual(['四川省', NONE])
    expect(periodsOf(resources, { keyword: '砂' }).map((f) => f.name)).toEqual(['2026-11'])
    expect(kindsOf(resources, { keyword: '说明' }).map((k) => k.kind)).toEqual([OUTSIDE])
    const tree = categoryTreeOf(resources, '材料', { keyword: '混凝土' })
    expect(tree.tree.map((n) => n.label)).toEqual(['材料'])
    expect(tree.tree[0]?.children[0]?.label).toBe('混凝土')
  })
})

describe('状态筛选', () => {
  it('清单：未套定额 = 手填行；外专业挂接要借定额专业表判断', () => {
    expect(filterBill(rows, { ...noBill, state: 'manual' }).map((r) => r.id)).toEqual([14])
    const ctx: FilterContext = { quotaSpecialty: new Map([['Q9', '市政']]) }
    expect(filterBill(rows, { ...noBill, state: 'cross' }, ctx).map((r) => r.id)).toEqual([11])
    // 同专业的挂接不算外专业；任一方未标注也不算
    expect(filterBill(rows, { ...noBill, state: 'cross' }).map((r) => r.id)).toEqual([])
  })

  it('定额：没有被清单引用；地区筛选（未标地区单列）', () => {
    expect(filterQuotas(quotas, { ...noQuota, state: 'unused' }, { quotaUsage: new Map([['Q1', 2]]) }).map((q) => q.code)).toEqual(['Q2', 'Q3'])
    expect(filterQuotas(quotas, { ...noQuota, region: NONE }).map((q) => q.code)).toEqual(['Q2', 'Q3'])
    expect(filterQuotas(quotas, { ...noQuota, region: '四川省' }).map((q) => q.code)).toEqual(['Q1'])
  })

  it('工料机：缺价、没有被定额引用', () => {
    expect(filterResources(resources, { ...noRes, state: 'noprice' }).map((r) => r.code)).toEqual(['ZERO'])
    // 现行价为 0 但基准价有值的不算缺价（取用价回退基准价）
    const ctx: FilterContext = { usedResourceCodes: new Set(['C25', 'ZERO']) }
    expect(filterResources(resources, { ...noRes, state: 'unused' }, ctx).map((r) => r.code)).toEqual(['SAND', 'W1'])
  })

  it('每一级的状态选项都有中文标签', () => {
    for (const tier of ['bill', 'quota', 'resource'] as const) {
      expect(STATE_LABELS[tier].length).toBeGreaterThan(0)
      for (const s of STATE_LABELS[tier]) expect(s.label.length).toBeGreaterThan(0)
    }
  })
})

describe('「待整理」逐项点开的落点', () => {
  it('切到对应那一级，只留这一项的筛选', () => {
    expect(pendingFilter('quotaChapter')).toEqual({ tier: 'quota', filters: { ...emptyQuotaFilters, chapter: NONE } })
    expect(pendingFilter('billSpecialty').filters).toMatchObject({ specialty: NONE })
    expect(pendingFilter('resourceKind')).toEqual({ tier: 'resource', filters: { ...emptyResourceFilters, kind: OUTSIDE } })
    expect(pendingFilter('resourcePeriod').filters).toMatchObject({ period: NONE })
  })

  it('落点的筛选确实筛出那一批行', () => {
    expect(filterQuotas(quotas, { ...noQuota, chapter: NONE }).map((q) => q.code)).toEqual(['Q2'])
    expect(filterQuotas(quotas, { ...noQuota, region: NONE }).map((q) => q.code)).toEqual(['Q2', 'Q3'])
  })
})
