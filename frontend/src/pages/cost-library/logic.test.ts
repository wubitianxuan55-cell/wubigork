// cost-library/logic.test.ts — 造价数据库（三级浏览面）纯函数层。
// 覆盖：关键词匹配与挂接码归一、清单筛选（项目/分部/关键词）、库项归并、现算
// 单价（手填优先 / 多定额求和 / 算不出给 null）、定额被谁用的引用索引、定额与
// 工料机筛选、取用价口径、渐进披露、选择键往返。
import { describe, expect, it } from 'vitest'
import type { WorkcostBillItem, WorkcostQuota, WorkcostResource } from '../../gaea/lib/types'
import {
  capRows, divisionsOf, effectivePrice, filterBill, filterQuotas, filterResources, fmtMoney, fmtQty, fmtRate,
  groupBill, kindCounts, linkedQuotas, matchKeyword, parseSelKey, quotaUsageIndex, selKey, specialtiesOf,
  unitPriceOf, type BillRow,
} from './logic'

function row(patch: Partial<BillRow>): BillRow {
  return {
    id: 0, projectId: 1, projectName: 'P1', code: '', title: '', unit: '', division: '', quantity: 0,
    quantityExpr: '', quotaCode: '', feature: '', priceOverride: 0, sort: 0, quotaCodes: [], ...patch,
  }
}

const rows: BillRow[] = [
  row({ id: 11, code: 'WP01', title: '施工便道', unit: 'm', division: 'A临建', feature: '泥结碎石', quotaCode: 'WP01', quotaCodes: ['WP01', 'WP09'] }),
  row({ id: 12, code: 'WP02', title: '场地平整', unit: 'm²', division: 'A临建', quotaCode: 'WP02', quotaCodes: ['WP02'] }),
  row({ id: 13, projectId: 2, projectName: 'P2', code: 'WP01', title: '施工便道', unit: 'm', division: 'A临建', feature: '泥结碎石', quotaCode: 'WP01', quotaCodes: ['WP01'] }),
  // 旧数据：只有单值列，没有引用表聚合。
  row({ id: 14, projectId: 2, projectName: 'P2', code: 'WP07', title: '围挡', unit: 'm', quotaCode: 'WP02' }),
  // 手填行：没有挂定额。
  row({ id: 15, projectId: 1, code: 'M001', title: '外委检测', unit: '项', priceOverride: 5000 }),
]

describe('关键词与挂接码', () => {
  it('空关键词恒命中；否则忽略大小写做包含匹配', () => {
    expect(matchKeyword('', '什么都没有', undefined)).toBe(true)
    expect(matchKeyword('  ', 'x')).toBe(true)
    expect(matchKeyword('C20', 'c20 商品混凝土')).toBe(true)
    expect(matchKeyword('C20', '商品混凝土', undefined, 'C25')).toBe(false)
    expect(matchKeyword('便道', '施工便道', 'WP01')).toBe(true)
  })

  it('挂接码：引用表聚合优先，旧数据回退单值列，空白码剔除', () => {
    const item: Pick<WorkcostBillItem, 'quotaCode' | 'quotaCodes'> = { quotaCode: 'A', quotaCodes: [' A ', 'B'] }
    expect(linkedQuotas(item)).toEqual(['A', 'B'])
    expect(linkedQuotas({ quotaCode: 'A', quotaCodes: [] })).toEqual(['A'])
    expect(linkedQuotas({ quotaCode: 'A', quotaCodes: undefined })).toEqual(['A'])
    expect(linkedQuotas({ quotaCode: '', quotaCodes: undefined })).toEqual([])
  })
})

describe('清单', () => {
  it('按项目 / 分部 / 关键词筛选，累计视图放行全部项目', () => {
    expect(filterBill(rows, { projectId: 0, division: '', keyword: '', grouped: false })).toHaveLength(5)
    expect(filterBill(rows, { projectId: 2, division: '', keyword: '', grouped: false }).map((r) => r.id)).toEqual([13, 14])
    // 分部筛选：三条「A临建」；围挡与手填行没有分部，被筛掉。
    expect(filterBill(rows, { projectId: 0, division: 'A临建', keyword: '', grouped: false }).map((r) => r.id)).toEqual([11, 12, 13])
    // 关键词也搜挂接定额码：只挂在 11 上的 WP09 能把这一行带出来。
    expect(filterBill(rows, { projectId: 0, division: '', keyword: 'wp09', grouped: false }).map((r) => r.id)).toEqual([11])
  })

  it('分部选项去重排序', () => {
    expect(divisionsOf([...rows, row({ id: 99, division: 'B支护' }), row({ id: 98, division: 'A临建' })])).toEqual(['A临建', 'B支护'])
  })

  it('库项归并：同名+同特征+同单位聚成一条，按被多少个项目使用降序，可展开成员', () => {
    const groups = groupBill(rows)
    const road = groups.find((g) => g.title === '施工便道')
    expect(road?.projects).toEqual(['P1', 'P2'])
    expect(road?.items.map((r) => r.id).sort()).toEqual([11, 13])
    expect(road?.quotaCodes.sort()).toEqual(['WP01', 'WP09'])
    // 两个项目使用的库项排在单项目库项前面。
    expect(groups[0]?.title).toBe('施工便道')
    // 特征不同不归并（围挡与场地平整各自成组）。
    expect(groups.map((g) => g.key)).toHaveLength(new Set(groups.map((g) => g.key)).size)
    expect(groups.find((g) => g.title === '围挡')?.items).toHaveLength(1)
  })

  it('现算单价：手填价优先，否则挂接定额的综合单价求和；算不出给 null 不冒充 0', () => {
    const ref = { WP01: 400, WP09: 50 }
    expect(unitPriceOf(rows[0]!, ref)).toBe(450)
    expect(unitPriceOf(rows[1]!, ref)).toBeNull() // 挂 WP02，参考价里没有
    expect(unitPriceOf(rows[4]!, ref)).toBe(5000) // 手填
    expect(unitPriceOf(row({ id: 1 }), {})).toBeNull() // 无挂接
  })

  it('引用索引：定额码 → 引用它的清单行（含旧数据单值列兜底）', () => {
    const index = quotaUsageIndex(rows)
    expect(index.get('WP01')?.map((r) => r.id)).toEqual([11, 13])
    expect(index.get('WP09')?.map((r) => r.id)).toEqual([11])
    expect(index.get('WP02')?.map((r) => r.id)).toEqual([12, 14])
    expect(index.has('NOPE')).toBe(false)
  })
})

describe('定额与工料机', () => {
  const quotas: WorkcostQuota[] = [
    { id: 1, code: 'WP01', title: '施工便道', specialty: '土壤修复', chapter: 'A临建', unit: 'm', categoryPath: '', baseLabor: 0, baseMaterial: 0, baseMachine: 0, source: '', region: '', priceDate: '', note: '', status: '现行' },
    { id: 2, code: 'TL01', origCode: 'TL-1', title: '灌注桩', specialty: '地基处理', chapter: 'B桩基', unit: 'm', categoryPath: '', baseLabor: 0, baseMaterial: 0, baseMachine: 0, source: '', region: '', priceDate: '', note: '', status: '现行' },
  ]
  const resources: WorkcostResource[] = [
    { id: 1, code: 'C20', kind: '材料', title: 'C20商品混凝土', spec: '泵送', unit: 'm³', basePrice: 345, currentPrice: 360, categoryPath: '材料/混凝土', source: '', supplier: '', region: '', priceDate: '', priceType: '', validUntil: '', lossRate: 0, note: '', tags: [], status: '现行' },
    { id: 2, code: 'L01', kind: '人工', title: '普通工', spec: '', unit: '工日', basePrice: 300, currentPrice: 0, categoryPath: '', source: '', supplier: '', region: '', priceDate: '', priceType: '', validUntil: '', lossRate: 0, note: '', tags: [], status: '现行' },
    { id: 3, code: 'Z01', kind: '材料', title: '待询价', spec: '', unit: 't', basePrice: 0, currentPrice: 0, categoryPath: '', source: '', supplier: '', region: '', priceDate: '', priceType: '', validUntil: '', lossRate: 0, note: '', tags: [], status: '现行' },
  ]

  it('定额按专业与关键词筛选；专业选项去重', () => {
    expect(specialtiesOf(quotas)).toEqual(['土壤修复', '地基处理'].sort((a, b) => a.localeCompare(b, 'zh-CN')))
    expect(filterQuotas(quotas, { specialty: '地基处理', keyword: '' }).map((q) => q.code)).toEqual(['TL01'])
    // 原码也能搜到。
    expect(filterQuotas(quotas, { specialty: '', keyword: 'tl-1' }).map((q) => q.code)).toEqual(['TL01'])
    expect(filterQuotas(quotas, { specialty: '', keyword: '桩基' }).map((q) => q.code)).toEqual(['TL01'])
  })

  it('工料机按类别与关键词筛选（名称 / 编码 / 规格 / 分类）；类别计数', () => {
    expect(filterResources(resources, { kind: '人工', keyword: '' }).map((r) => r.code)).toEqual(['L01'])
    expect(filterResources(resources, { kind: '', keyword: '泵送' }).map((r) => r.code)).toEqual(['C20'])
    expect(filterResources(resources, { kind: '', keyword: '混凝土' }).map((r) => r.code)).toEqual(['C20'])
    expect(kindCounts(resources)).toEqual({ 材料: 2, 人工: 1 })
  })

  it('取用价：现行价优先，为 0 回退基准价（与 Go 侧 EffectivePrice 同口径）', () => {
    expect(effectivePrice(resources[0]!)).toBe(360)
    expect(effectivePrice(resources[1]!)).toBe(300)
    expect(effectivePrice(resources[2]!)).toBe(0)
  })
})

describe('展示与小工具', () => {
  it('金额 / 数量 / 费率格式化：未录入不显示成 0', () => {
    expect(fmtMoney(469.2)).toBe('¥469.2')
    expect(fmtMoney(null)).toBe('—')
    expect(fmtMoney(0)).toBe('¥0')
    expect(fmtQty(1.02)).toBe('1.02')
    expect(fmtQty(0)).toBe('—')
    expect(fmtQty(-1)).toBe('—')
    expect(fmtRate(0.03)).toBe('3%')
    expect(fmtRate(0.03333)).toBe('3.33%')
    expect(fmtRate(0)).toBe('不计取')
  })

  it('渐进披露：先给前 cap 行，其余等用户点', () => {
    const all = Array.from({ length: 7 }, (_, i) => i)
    expect(capRows(all, 3)).toEqual({ shown: [0, 1, 2], hidden: 4 })
    expect(capRows(all, 10)).toEqual({ shown: all, hidden: 0 })
    expect(capRows([], 3)).toEqual({ shown: [], hidden: 0 })
  })

  it('选择键往返；非法层级或缺分隔符返回 null', () => {
    for (const tier of ['bill', 'quota', 'resource'] as const) {
      const key = selKey(tier, 'A:01')
      expect(parseSelKey(key)).toEqual({ tier, id: 'A:01' })
    }
    expect(parseSelKey('nope:1')).toBeNull()
    expect(parseSelKey('bill')).toBeNull()
  })
})
