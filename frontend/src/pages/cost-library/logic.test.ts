// cost-library/logic.test.ts — 造价数据库（三级浏览面）纯函数层。
// 覆盖：关键词匹配与挂接码归一、清单筛选（项目/分部/专业/关键词）、库项归并、现算
// 单价（手填优先 / 多定额求和 / 算不出给 null）、消耗量、定额被谁用的引用索引、
// 核算行与含量行对齐、定额与工料机筛选（专业/分部/类别/分类/地区/期）、分类树、
// 期数排序与显示、待整理计数、取用价口径、渐进披露、选择键往返。
import { describe, expect, it } from 'vitest'
import type { WorkcostBillItem, WorkcostComposeLineView, WorkcostQuota, WorkcostQuotaItem, WorkcostResource } from '../../gaea/lib/types'
import {
  NONE, OUTSIDE, alignItems, billSpecialties, capRows, categoryTreeOf, chaptersOf, consumptionOf, dimLabel, divisionsOf, effectivePrice,
  facetMatch, filterBill, filterQuotas, filterResources, fmtMoney, fmtPeriod, fmtQty, fmtRate, groupBill, isCrossSpecialty,
  isKnownKind, kindCounts, kindLabel, kindsOf, linkedQuotas, matchKeyword, parseSelKey, pendingOf, pendingSummary, periodKey, periodsOf, quotaFactor,
  quotaUsageIndex, regionsOf, selKey, specialtiesOf, unitPriceOf, type BillRow,
} from './logic'

function row(patch: Partial<BillRow>): BillRow {
  return {
    id: 0, projectId: 1, projectName: 'P1', code: '', title: '', unit: '', division: '', quantity: 0,
    quantityExpr: '', quotaCode: '', feature: '', priceOverride: 0, sort: 0, quotaCodes: [], ...patch,
  }
}

const rows: BillRow[] = [
  row({ id: 11, code: 'WP01', title: '施工便道', unit: 'm', division: 'A临建', feature: '泥结碎石', quotaCode: 'WP01', quotaCodes: ['WP01', 'WP09'], specialty: '土壤修复' }),
  row({ id: 12, code: 'WP02', title: '场地平整', unit: 'm²', division: 'A临建', quotaCode: 'WP02', quotaCodes: ['WP02'], specialty: '土壤修复' }),
  row({ id: 13, projectId: 2, projectName: 'P2', code: 'WP01', title: '施工便道', unit: 'm', division: 'A临建', feature: '泥结碎石', quotaCode: 'WP01', quotaCodes: ['WP01'], specialty: '市政' }),
  // 旧数据：只有单值列，没有引用表聚合；专业还没标注。
  row({ id: 14, projectId: 2, projectName: 'P2', code: 'WP07', title: '围挡', unit: 'm', quotaCode: 'WP02' }),
  // 手填行：没有挂定额；专业还没标注。
  row({ id: 15, projectId: 1, code: 'M001', title: '外委检测', unit: '项', priceOverride: 5000 }),
]
const noBillFilter = { projectId: 0, division: '', specialty: '', keyword: '', grouped: false }

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

describe('规范值筛选', () => {
  it('筛选为空=不过滤；NONE=只要空值（未标注）；否则精确相等', () => {
    expect(facetMatch('', undefined)).toBe(true)
    expect(facetMatch('', '市政')).toBe(true)
    expect(facetMatch(NONE, '')).toBe(true)
    expect(facetMatch(NONE, undefined)).toBe(true)
    expect(facetMatch(NONE, '土壤修复')).toBe(false)
    expect(facetMatch('市政', ' 市政 ')).toBe(true)
    expect(facetMatch('市政', '土壤修复')).toBe(false)
  })

  it('维度值的显示：空值与哨兵都显示占位名，其余原样', () => {
    expect(dimLabel('', '未标注')).toBe('未标注')
    expect(dimLabel(undefined, '未标注')).toBe('未标注')
    expect(dimLabel(NONE, '未细分')).toBe('未细分')
    expect(dimLabel(' 土壤修复 ')).toBe('土壤修复')
  })
})

describe('清单', () => {
  it('按项目 / 分部 / 关键词筛选，累计视图放行全部项目', () => {
    expect(filterBill(rows, noBillFilter)).toHaveLength(5)
    expect(filterBill(rows, { ...noBillFilter, projectId: 2 }).map((r) => r.id)).toEqual([13, 14])
    // 分部筛选：三条「A临建」；围挡与手填行没有分部，被筛掉。
    expect(filterBill(rows, { ...noBillFilter, division: 'A临建' }).map((r) => r.id)).toEqual([11, 12, 13])
    // 关键词也搜挂接定额码：只挂在 11 上的 WP09 能把这一行带出来。
    expect(filterBill(rows, { ...noBillFilter, keyword: 'wp09' }).map((r) => r.id)).toEqual([11])
  })

  it('按专业筛选：「未标注」是一桶；专业分布按条数降序、未标注排最后', () => {
    expect(filterBill(rows, { ...noBillFilter, specialty: '土壤修复' }).map((r) => r.id)).toEqual([11, 12])
    expect(filterBill(rows, { ...noBillFilter, specialty: NONE }).map((r) => r.id)).toEqual([14, 15])
    expect(billSpecialties(rows, { projectId: 0, division: '' })).toEqual([
      { name: '土壤修复', count: 2 }, { name: '市政', count: 1 }, { name: NONE, count: 2 },
    ])
    // 专业分布受上游筛选约束：只看 P2 时只剩市政与未标注。
    expect(billSpecialties(rows, { projectId: 2, division: '' }).map((f) => f.name).sort()).toEqual([NONE, '市政'].sort())
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

  it('定额工程量与消耗量：挂接明细给了用它，否则跟随清单工程量；非正不显示', () => {
    const item = row({ id: 1, quantity: 100, quotaCode: 'A', quotaCodes: ['A', 'B'], linkQuantities: { B: 30 } })
    expect(quotaFactor(item, 'A')).toBe(100)
    expect(quotaFactor(item, 'B')).toBe(30)
    expect(quotaFactor(item, '没有的码')).toBe(100)
    expect(consumptionOf(100, 1.02)).toBeCloseTo(102)
    expect(consumptionOf(0, 1.02)).toBeNull()
    expect(consumptionOf(100, 0)).toBeNull()
    expect(consumptionOf(Number.POSITIVE_INFINITY, 1)).toBeNull()
  })

  it('引用索引：定额码 → 引用它的清单行（含旧数据单值列兜底）', () => {
    const index = quotaUsageIndex(rows)
    expect(index.get('WP01')?.map((r) => r.id)).toEqual([11, 13])
    expect(index.get('WP09')?.map((r) => r.id)).toEqual([11])
    expect(index.get('WP02')?.map((r) => r.id)).toEqual([12, 14])
    expect(index.has('NOPE')).toBe(false)
  })

  it('外专业判定：清单专业与定额专业都对得上不算；任一方未标注不算', () => {
    expect(isCrossSpecialty('土壤修复', '市政')).toBe(true)
    expect(isCrossSpecialty('土壤修复', '土壤修复')).toBe(false)
    expect(isCrossSpecialty('土壤修复', '')).toBe(false)
    expect(isCrossSpecialty('', '市政')).toBe(false)
    expect(isCrossSpecialty('土壤修复', undefined)).toBe(false)
  })
})

describe('定额与工料机', () => {
  const quotas: WorkcostQuota[] = [
    { id: 1, code: 'WP01', title: '施工便道', specialty: '土壤修复', chapter: 'A临建', unit: 'm', categoryPath: '', baseLabor: 0, baseMaterial: 0, baseMachine: 0, source: '', region: '', priceDate: '', note: '', status: '现行' },
    { id: 2, code: 'TL01', origCode: 'TL-1', title: '灌注桩', specialty: '地基处理', chapter: 'B桩基', unit: 'm', categoryPath: '', baseLabor: 0, baseMaterial: 0, baseMachine: 0, source: '', region: '', priceDate: '', note: '', status: '现行' },
    { id: 3, code: 'TL02', title: '冠梁', specialty: '地基处理', chapter: '', unit: 'm', categoryPath: '', baseLabor: 0, baseMaterial: 0, baseMachine: 0, source: '', region: '', priceDate: '', note: '', status: '现行' },
    { id: 4, code: 'ZZ01', title: '没标专业的定额', specialty: '', chapter: '', unit: 'm', categoryPath: '', baseLabor: 0, baseMaterial: 0, baseMachine: 0, source: '', region: '', priceDate: '', note: '', status: '现行' },
  ]
  const noQuotaFilter = { specialty: '', chapter: '', keyword: '' }
  const resources: WorkcostResource[] = [
    { id: 1, code: 'C20', kind: '材料', title: 'C20商品混凝土', spec: '泵送', unit: 'm³', basePrice: 345, currentPrice: 360, categoryPath: '材料/混凝土', source: '', supplier: '', region: '四川省', priceDate: '2026-06', priceType: '', validUntil: '', lossRate: 0, note: '', tags: [], status: '现行' },
    { id: 2, code: 'L01', kind: '人工', title: '普通工', spec: '', unit: '工日', basePrice: 300, currentPrice: 0, categoryPath: '', source: '', supplier: '', region: '', priceDate: '', priceType: '', validUntil: '', lossRate: 0, note: '', tags: [], status: '现行' },
    { id: 3, code: 'Z01', kind: '材料', title: '待询价', spec: '', unit: 't', basePrice: 0, currentPrice: 0, categoryPath: '', source: '', supplier: '', region: '重庆市', priceDate: '2026年11月', priceType: '', validUntil: '', lossRate: 0, note: '', tags: [], status: '现行' },
    { id: 4, code: 'C30', kind: '材料', title: 'C30商品混凝土', spec: '泵送', unit: 'm³', basePrice: 380, currentPrice: 390, categoryPath: '材料/混凝土/泵送', source: '', supplier: '', region: '重庆市', priceDate: '2025-12', priceType: '', validUntil: '', lossRate: 0, note: '', tags: [], status: '现行' },
    { id: 5, code: 'REB', kind: '材料', title: '螺纹钢筋', spec: 'HRB400', unit: 't', basePrice: 3200, currentPrice: 3300, categoryPath: '材料/钢材', source: '', supplier: '', region: '四川省', priceDate: '2026年6月', priceType: '', validUntil: '', lossRate: 0, note: '', tags: [], status: '现行' },
  ]
  const noResourceFilter = { kind: '', category: '', region: '', period: '', keyword: '' }

  it('定额按专业 / 分部 / 关键词筛选；专业与分部分布带条数、未标注排最后', () => {
    expect(specialtiesOf(quotas)).toEqual([
      { name: '地基处理', count: 2 }, { name: '土壤修复', count: 1 }, { name: NONE, count: 1 },
    ])
    expect(filterQuotas(quotas, { ...noQuotaFilter, specialty: '地基处理' }).map((q) => q.code)).toEqual(['TL01', 'TL02'])
    // 原码也能搜到。
    expect(filterQuotas(quotas, { ...noQuotaFilter, keyword: 'tl-1' }).map((q) => q.code)).toEqual(['TL01'])
    expect(filterQuotas(quotas, { ...noQuotaFilter, keyword: '桩基' }).map((q) => q.code)).toEqual(['TL01'])
    // 分部只在专业内比：地基处理下一条有分部、一条未归。
    expect(chaptersOf(quotas, '地基处理')).toEqual([{ name: 'B桩基', count: 1 }, { name: NONE, count: 1 }])
    expect(filterQuotas(quotas, { ...noQuotaFilter, specialty: '地基处理', chapter: NONE }).map((q) => q.code)).toEqual(['TL02'])
    expect(filterQuotas(quotas, { ...noQuotaFilter, specialty: NONE }).map((q) => q.code)).toEqual(['ZZ01'])
  })

  it('工料机按类别 / 分类 / 地区 / 期 / 关键词筛选；类别计数', () => {
    expect(filterResources(resources, { ...noResourceFilter, kind: '人工' }).map((r) => r.code)).toEqual(['L01'])
    expect(filterResources(resources, { ...noResourceFilter, keyword: '泵送' }).map((r) => r.code)).toEqual(['C20', 'C30'])
    expect(filterResources(resources, { ...noResourceFilter, keyword: '混凝土' }).map((r) => r.code)).toEqual(['C20', 'C30'])
    expect(kindCounts(resources)).toEqual({ 材料: 4, 人工: 1 })
    // 分类：选上级连带下级。
    expect(filterResources(resources, { ...noResourceFilter, category: '材料/混凝土' }).map((r) => r.code)).toEqual(['C20', 'C30'])
    expect(filterResources(resources, { ...noResourceFilter, category: '材料/混凝土/泵送' }).map((r) => r.code)).toEqual(['C30'])
    expect(filterResources(resources, { ...noResourceFilter, category: '材料/混凝' }).map((r) => r.code)).toEqual([]) // 按段匹配，不做前缀子串
    expect(filterResources(resources, { ...noResourceFilter, category: NONE }).map((r) => r.code)).toEqual(['L01', 'Z01'])
    // 地区与期：未标注是一桶；期数按库里的规范写法比对。
    expect(filterResources(resources, { ...noResourceFilter, region: '四川省' }).map((r) => r.code)).toEqual(['C20', 'REB'])
    expect(filterResources(resources, { ...noResourceFilter, region: NONE }).map((r) => r.code)).toEqual(['L01'])
    expect(filterResources(resources, { ...noResourceFilter, period: '2026-06' }).map((r) => r.code)).toEqual(['C20'])
    expect(filterResources(resources, { ...noResourceFilter, period: NONE }).map((r) => r.code)).toEqual(['L01'])
  })

  it('类别待整理：类别不在四类之内的行在类别列表末尾单列一桶，筛出来只有它们；没有这种行就不出这一桶', () => {
    expect(kindsOf(resources).map((k) => [k.kind, k.count])).toEqual([['人工', 1], ['材料', 4]]) // 四类按固定顺序
    expect(kindsOf([])).toEqual([])
    const note: WorkcostResource = { ...resources[0]!, id: 9, code: 'NOTE-1', kind: '一段备注：地磅设备摊销为估价', title: '错记成工料机的说明行', categoryPath: '' }
    const odd: WorkcostResource = { ...resources[1]!, id: 8, code: 'X', kind: '  ' } // 类别是空白：同样在四类之外
    const all = [...resources, note, odd]
    expect(isKnownKind('材料')).toBe(true)
    expect(isKnownKind('材料 ')).toBe(true)
    expect(isKnownKind('一段备注')).toBe(false)
    expect(isKnownKind(undefined)).toBe(false)
    expect(kindLabel(OUTSIDE)).toBe('类别待整理')
    expect(kindLabel('机械')).toBe('机械')
    expect(kindsOf(all).map((k) => [k.kind, k.count])).toEqual([['人工', 1], ['材料', 4], [OUTSIDE, 2]])
    expect(filterResources(all, { ...noResourceFilter, kind: OUTSIDE }).map((r) => r.code)).toEqual(['NOTE-1', 'X'])
    // 四类内的行不受这一桶影响；四类外有分类路径的也不混进「材料」。
    expect(filterResources(all, { ...noResourceFilter, kind: '材料' })).toHaveLength(4)
    expect(categoryTreeOf(all, OUTSIDE)).toEqual({ tree: [], uncategorized: 2 })
  })

  it('分类树：路径的每一段一级、条数含下级；没分类的另计', () => {
    const { tree, uncategorized } = categoryTreeOf(resources, '材料')
    expect(uncategorized).toBe(1) // Z01
    expect(tree.map((n) => [n.path, n.count])).toEqual([['材料', 3]])
    const concrete = tree[0]!.children.find((n) => n.path === '材料/混凝土')!
    expect(concrete.count).toBe(2) // C20 与 C30
    expect(concrete.children.map((n) => [n.path, n.count])).toEqual([['材料/混凝土/泵送', 1]])
    expect(tree[0]!.children.map((n) => n.path).sort()).toEqual(['材料/混凝土', '材料/钢材'])
    // 换个类别：人工下没有分类，全是未细分。
    expect(categoryTreeOf(resources, '人工')).toEqual({ tree: [], uncategorized: 1 })
  })

  it('地区与期数分布：期数新的在前、无期的在最后', () => {
    expect(regionsOf(resources)).toEqual([
      { name: '重庆市', count: 2 }, { name: '四川省', count: 2 }, { name: NONE, count: 1 },
    ])
    expect(periodsOf(resources).map((f) => f.name)).toEqual(['2026年11月', '2026-06', '2026年6月', '2025-12', NONE])
    expect(periodsOf(resources).map((f) => f.count)).toEqual([1, 1, 1, 1, 1])
  })

  it('期数的键与显示：认得的年月写法归一；认不出的键排最后、显示原样', () => {
    expect(periodKey('2026-06')).toBe(202606)
    expect(periodKey('2026年6月')).toBe(202606)
    expect(periodKey('2025.12')).toBe(202512)
    expect(periodKey('')).toBe(-1)
    expect(periodKey('一季度')).toBe(-1)
    expect(fmtPeriod('2026-06')).toBe('2026年6月')
    expect(fmtPeriod('2026年11月')).toBe('2026年11月')
    expect(fmtPeriod('2026年第11期')).toBe('2026年第11期')
    expect(fmtPeriod('')).toBe('')
  })

  it('取用价：现行价优先，为 0 回退基准价（与 Go 侧 EffectivePrice 同口径）', () => {
    expect(effectivePrice(resources[0]!)).toBe(360)
    expect(effectivePrice(resources[1]!)).toBe(300)
    expect(effectivePrice(resources[2]!)).toBe(0)
  })
})

describe('核算行与含量行对齐', () => {
  const line = (kind: string, title: string): WorkcostComposeLineView =>
    ({ kind: kind as WorkcostComposeLineView['kind'], title, unit: '', quantity: 1, price: 1, lossRate: 0, amount: 1, sharePct: 0 })
  const item = (kind: string, title: string, resourceCode: string): WorkcostQuotaItem =>
    ({ quotaCode: 'Q', resourceCode, kind: kind as WorkcostQuotaItem['kind'], title, unit: '', quantity: 1, resourcePrice: 0, lossRate: 0, note: '', sort: 0 })

  it('后端把核算行按 人工→材料 重排、含量行是工作簿原序：按类别+名称对上，不钻错', () => {
    const lines = [line('人工', '普通工'), line('材料', 'C20商品混凝土')]
    const items = [item('材料', 'C20商品混凝土', 'C20'), item('人工', '普通工', 'L01')]
    expect(alignItems(lines, items).map((it) => it?.resourceCode)).toEqual(['L01', 'C20'])
  })

  it('顺序本来就一致时同样对上；同类多条按名称区分，名称对不上退回同下标', () => {
    const lines = [line('材料', '中砂'), line('材料', '碎石'), line('机械', '挖机')]
    const items = [item('材料', '碎石', 'S1'), item('材料', '中砂', 'S2'), item('机械', '别的名字', 'M1')]
    expect(alignItems(lines, items).map((it) => it?.resourceCode)).toEqual(['S2', 'S1', 'M1'])
  })

  it('对不上的位置留空，不乱点鸳鸯', () => {
    const lines = [line('材料', '中砂'), line('材料', '碎石')]
    expect(alignItems(lines, [item('人工', '普通工', 'L01')]).map((it) => it?.resourceCode)).toEqual([undefined, undefined])
    expect(alignItems([], [])).toEqual([])
  })
})

describe('待整理', () => {
  it('不合规矩的条数逐项列出，总数 = 各项之和；说明只列非零项', () => {
    const quota: WorkcostQuota = {
      id: 1, code: 'Q1', title: '定额', specialty: '', chapter: '', unit: 'm', categoryPath: '', baseLabor: 0, baseMaterial: 0,
      baseMachine: 0, source: '', region: '', priceDate: '', note: '', status: '现行',
    }
    const res: WorkcostResource = {
      id: 1, code: 'R1', kind: '材料', title: '材料', spec: '', unit: 't', basePrice: 1, currentPrice: 1, categoryPath: '',
      source: '', supplier: '', region: '', priceDate: '', priceType: '', validUntil: '', lossRate: 0, note: '', tags: [], status: '现行',
    }
    // 类别写成别的东西的存量行：其余维度都齐，只差类别。
    const odd: WorkcostResource = {
      ...res, id: 2, code: 'R2', kind: '一段说明', categoryPath: '其他', region: '四川省', priceDate: '2026-06',
    }
    const p = pendingOf(rows, [quota], [res, odd])
    expect(p.billSpecialty).toBe(2)
    expect(p.quotaSpecialty).toBe(1)
    expect(p.quotaChapter).toBe(1)
    expect(p.resourceKind).toBe(1)
    expect(p.resourceRegion).toBe(1)
    expect(p.resourceCategory).toBe(1)
    expect(p.resourcePeriod).toBe(1)
    // 总数把列出来的每一项都算进去（入口上的数要和它自己的说明对得上）。
    expect(p.total).toBe(2 + 1 + 1 + 1 + 1 + 1 + 1)
    expect(pendingSummary(p)).toBe('清单无专业 2 · 定额无专业 1 · 定额未归分部 1 · 工料机类别待整理 1 · 工料机未标地区 1 · 工料机未细分 1 · 工料机无期数 1')
    const clean = pendingOf([row({ specialty: '土壤修复' })], [], [])
    expect(clean.total).toBe(0)
    expect(pendingSummary(clean)).toBe('')
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
