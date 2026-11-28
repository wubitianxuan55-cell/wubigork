// CostLibraryPage.test.tsx — 一级板块「造价数据库」页面测试（2026-11-25 推倒重建）。
// 覆盖：①三级行计数与默认清单列表 ②点行看综合单价分析 → 定额码下钻到定额 →
// 含量行下钻到工料机 → 引用清单回链 ③左栏项目筛选与项目封面卡 ④搜索与无匹配
// 清除筛选 ⑤归并视图 ⑥删除：取消不删、确认删并重读、后端拒绝（仍被引用）时
// 原因留在卡上 ⑦导出：选目录后按当前筛选调绑定并显示路径 ⑧体检抽屉列出发现并
// 可跳转 ⑨板块常驻保活下切回即静默重读 ⑩三级全读失败不伪装成空库、重试恢复
// ⑪空库三步引导与去办公板块。桥接只 mock 页面消费的绑定。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import type { CostIndexStatusView, WorkcostBillItem, WorkcostBillProject, WorkcostCompose, WorkcostQuota, WorkcostResource } from '../gaea/lib/types'

const mocks = vi.hoisted(() => ({
  WorkcostBillProjects: vi.fn(),
  WorkcostBillItems: vi.fn(),
  WorkcostQuotaList: vi.fn(),
  WorkcostQuotaGet: vi.fn(),
  WorkcostQuotaCompose: vi.fn(),
  WorkcostQuotaComposeMany: vi.fn(),
  WorkcostQuotaDelete: vi.fn(),
  WorkcostResourceList: vi.fn(),
  WorkcostResourcePrices: vi.fn(),
  WorkcostResourceUsage: vi.fn(),
  WorkcostResourceDelete: vi.fn(),
  WorkcostBillItemDelete: vi.fn(),
  WorkcostBillProjectDelete: vi.fn(),
  CostHealthScan: vi.fn(),
  CostLibraryExport: vi.fn(),
  PickDirectory: vi.fn(),
  CostIndexStatus: vi.fn(),
  CostIndexBackfill: vi.fn(),
  WorkcostBillSearch: vi.fn(),
  // 库页编辑面（2026-11-27）：直接修正录入数据的保存入口。
  WorkcostBillItemSave: vi.fn(),
  WorkcostQuotaSave: vi.fn(),
  WorkcostResourceSave: vi.fn(),
  WorkcostResourceSetPrice: vi.fn(),
  WorkcostBillQuotaLinks: vi.fn(),
  WorkcostBillQuotaAttach: vi.fn(),
  WorkcostBillQuotaDetach: vi.fn(),
  WorkcostBillProjectRatesSave: vi.fn(),
  WorkcostBillProjectProfileSave: vi.fn(),
  // 库维护（2026-11-28）：字典读写、定额解除全部挂接、定额地区回填。
  WorkcostDictList: vi.fn(),
  WorkcostDictSet: vi.fn(),
  WorkcostQuotaDetachEverywhere: vi.fn(),
  WorkcostQuotaRegionBackfill: vi.fn(),
}))
vi.mock('../gaea/lib/bridge', () => ({ app: mocks }))
vi.mock('../gaea/lib/clipboard', () => ({ copyToClipboard: vi.fn().mockResolvedValue('plain') }))

import { notifyBoardActive, resetBoardActive } from '../lib/boardActive'
import { copyToClipboard } from '../gaea/lib/clipboard'
import { PENDING_PROMPT } from './cost-library/logic'
import CostLibraryPage from './CostLibraryPage'

const p1: WorkcostBillProject = {
  id: 1, name: '旺平矿业修复', fileName: '旺平成本测算表.xlsx', source: '', location: '乐山', duration: '180 天',
  pricing: '综合单价只含人材机', specialty: '土壤修复', region: '', pricePeriod: '', managementRate: 0.1, regulatoryRate: 0.02, profitRate: 0.07, taxRate: 0.09,
  profitIncludesRegulatory: false, controlPrice: 0, itemCount: 2,
}
const p2: WorkcostBillProject = { ...p1, id: 2, name: '什邡地块治理', itemCount: 1, location: '德阳' }

/** 导出入参：与页面同一份「零值=不过滤」的筛选结构体。 */
const EMPTY_FILTER = { projectId: 0, keyword: '', specialty: '', division: '', chapter: '', kind: '', category: '', region: '', period: '' }

const row11 = { id: 11, projectId: 1, projectName: p1.name, code: 'WP01', gcode: 'Q0001001', title: '施工便道（铺筑压实）', unit: 'm', division: 'A临建', quantity: 850, quantityExpr: '长850×宽1', quotaCode: 'WP01', feature: '泥结碎石', priceOverride: 0, listPrice: 420, sort: 1, quotaCodes: ['WP01'] }
const row12 = { id: 12, projectId: 1, projectName: p1.name, code: 'M001', title: '外委检测', unit: '项', division: 'D外运处置', quantity: 1, quantityExpr: '', quotaCode: '', feature: '', priceOverride: 5000, sort: 2, quotaCodes: [] }
const row13 = { id: 13, projectId: 2, projectName: p2.name, code: 'WP01', gcode: 'Q0001002', title: '施工便道（铺筑压实）', unit: 'm', division: 'A临建', quantity: 300, quantityExpr: '', quotaCode: 'WP01', feature: '泥结碎石', priceOverride: 0, listPrice: 430, sort: 1, quotaCodes: ['WP01'] }

const quotaWP01: WorkcostQuota = {
  id: 1, code: 'WP01', gcode: 'D0101001', origCode: 'WP-1', title: '施工便道', specialty: '土壤修复', chapter: 'A临建', unit: 'm',
  categoryPath: '综合单价/A临建', baseLabor: 0, baseMaterial: 0, baseMachine: 0, source: '项目导入：旺平成本测算表.xlsx',
  region: '', priceDate: '', note: '', status: '现行',
  items: [
    { id: 1, quotaCode: 'WP01', resourceCode: 'C20', kind: '材料', title: 'C20商品混凝土', spec: '泵送到场', unit: 'm³', quantity: 0.45, resourcePrice: 0, lossRate: 0, note: '', sort: 0 },
    { id: 2, quotaCode: 'WP01', resourceCode: 'L01', kind: '人工', title: '普通工', spec: '', unit: '工日', quantity: 0.03, resourcePrice: 0, lossRate: 0, note: '', sort: 1 },
  ],
}
const quotaFree: WorkcostQuota = { ...quotaWP01, id: 2, code: 'ENT-01', origCode: '', title: '企业定额（未引用）', items: [] }

const composeWP01: WorkcostCompose = {
  laborFee: 9, materialFee: 162, machineFee: 0, outsourcedFee: 0, otherFee: 0, subtotal: 171, compositePrice: 171,
  zeroLines: 0, warnings: [],
  lines: [
    { kind: '材料', title: 'C20商品混凝土', unit: 'm³', quantity: 0.45, price: 360, lossRate: 0, amount: 162, sharePct: 94.7 },
    { kind: '人工', title: '普通工', unit: '工日', quantity: 0.03, price: 300, lossRate: 0, amount: 9, sharePct: 5.3 },
  ],
}

const c20: WorkcostResource = {
  id: 101, code: 'C20', gcode: 'RC010001', kind: '材料', title: 'C20商品混凝土', spec: '泵送到场', unit: 'm³', basePrice: 345, currentPrice: 360,
  categoryPath: '材料/混凝土', source: '成本库', supplier: '', region: '四川省', priceDate: '2026年第7期', priceType: '到场价',
  validUntil: '', lossRate: 0, note: '', tags: [], status: '现行',
}
const l01: WorkcostResource = { ...c20, id: 102, code: 'L01', kind: '人工', title: '普通工', spec: '', unit: '工日', basePrice: 300, currentPrice: 0 }
const zero: WorkcostResource = { ...c20, id: 103, code: 'Z01', title: '待询价材料', spec: '', unit: 't', basePrice: 0, currentPrice: 0 }

// 仓库未装 jest-dom 匹配器：文本断言统一走 textContent 归一后的包含判断。
function textOf(el: HTMLElement | null | undefined): string {
  return (el?.textContent ?? '').replace(/\s+/g, ' ')
}
function expectText(el: HTMLElement | null | undefined, part: string): void {
  expect(textOf(el)).toContain(part)
}
function expectAriaSelected(el: HTMLElement | null): void {
  expect(el?.getAttribute('aria-selected')).toBe('true')
}

async function renderReady() {
  const utils = render(<CostLibraryPage />)
  await screen.findByTestId('cost-bill-rows')
  return utils
}

beforeEach(() => {
  vi.clearAllMocks()
  resetBoardActive()
  mocks.WorkcostBillProjects.mockResolvedValue([{ ...p1 }, { ...p2 }])
  mocks.WorkcostBillItems.mockImplementation(async (projectId: number) =>
    projectId === 1 ? [{ ...row11 }, { ...row12 }] : [{ ...row13 }])
  mocks.WorkcostQuotaList.mockResolvedValue([{ ...quotaWP01, items: undefined, usageCount: 2 }, { ...quotaFree, items: undefined, usageCount: 0 }])
  mocks.WorkcostQuotaGet.mockImplementation(async (code: string) => (code === 'WP01' ? quotaWP01 : { ...quotaFree }))
  mocks.WorkcostQuotaCompose.mockImplementation(async (code: string) => (code === 'WP01' ? composeWP01 : { ...composeWP01, lines: [], subtotal: 0, compositePrice: 0 }))
  mocks.WorkcostQuotaComposeMany.mockResolvedValue({ WP01: composeWP01 })
  mocks.WorkcostResourceList.mockResolvedValue([{ ...c20 }, { ...l01 }, { ...zero }])
  mocks.WorkcostResourcePrices.mockResolvedValue([
    { id: 2, resourceId: 101, price: 360, period: '2026年第11期', region: '四川省', priceType: '到场价', source: '造价模式调价', fetchedAt: '2026-11-20T08:00:00Z', note: '' },
    { id: 1, resourceId: 101, price: 345, period: '2026年第7期', region: '四川省', priceType: '到场价', source: '项目导入', fetchedAt: '2026-10-07T08:00:00Z', note: '' },
  ])
  mocks.WorkcostResourceUsage.mockImplementation(async (code: string) =>
    code === 'C20' ? [{ id: 1, code: 'WP01', title: '施工便道', unit: 'm', quantity: 0.45 }] : [])
  mocks.WorkcostBillItemDelete.mockResolvedValue(undefined)
  mocks.WorkcostQuotaDelete.mockResolvedValue(undefined)
  mocks.WorkcostResourceDelete.mockResolvedValue(undefined)
  mocks.WorkcostBillProjectDelete.mockResolvedValue(1)
  mocks.CostHealthScan.mockResolvedValue({
    zeroPrice: [{ id: 103, code: 'Z01', title: '待询价材料', kind: '材料', unit: 't', detail: '现行价与基准价均为 0' }],
    quotaNoPrice: [], brokenRef: [], duplicates: [], codeMismatch: [], totalFindings: 1,
  })
  mocks.CostLibraryExport.mockImplementation(async (_k: string, _f: unknown, out: string) => out)
  mocks.PickDirectory.mockResolvedValue('D:\\新工作\\导出')
  // 语义索引默认「未建」（条数 0）：页面不出去调语义补充，本地过滤照常。
  const unbuilt = {
    model: 'bge-m3', running: true,
    kinds: [
      { kind: 'wc_quota', label: '定额', total: 0, count: 0, updatedAt: '' },
      { kind: 'wc_res', label: '工料机', total: 0, count: 0, updatedAt: '' },
      { kind: 'wc_bill', label: '清单', total: 0, count: 0, updatedAt: '' },
    ],
  }
  mocks.CostIndexStatus.mockResolvedValue(unbuilt)
  mocks.CostIndexBackfill.mockResolvedValue(unbuilt)
  mocks.WorkcostBillSearch.mockResolvedValue([])
  // 编辑面默认成功；入参断言在各用例里做。
  mocks.WorkcostBillItemSave.mockImplementation(async (it: WorkcostBillItem) => it)
  mocks.WorkcostQuotaSave.mockImplementation(async (q: WorkcostQuota) => q)
  mocks.WorkcostResourceSave.mockImplementation(async (r: WorkcostResource) => r)
  mocks.WorkcostResourceSetPrice.mockImplementation(async (_id: number, price: number) => ({ ...c20, currentPrice: price }))
  mocks.WorkcostBillQuotaLinks.mockResolvedValue([{ id: 5, billItemId: 11, quotaCode: 'WP01', quantity: 0, sort: 1 }])
  mocks.WorkcostBillQuotaAttach.mockImplementation(async (_id: number, code: string, quantity: number) =>
    ({ id: 5, billItemId: 11, quotaCode: code, quantity, sort: 1 }))
  mocks.WorkcostBillQuotaDetach.mockResolvedValue(undefined)
  mocks.WorkcostBillProjectRatesSave.mockResolvedValue(undefined)
  mocks.WorkcostBillProjectProfileSave.mockResolvedValue(undefined)
  // 库维护默认：字典读得到但是空的、回填没有待办、解除挂接影响 0 条（各用例按需覆盖）。
  mocks.WorkcostDictList.mockResolvedValue([])
  mocks.WorkcostDictSet.mockResolvedValue(undefined)
  mocks.WorkcostQuotaDetachEverywhere.mockResolvedValue(0)
  mocks.WorkcostQuotaRegionBackfill.mockResolvedValue([])
})

describe('造价数据库 · 三级浏览面', () => {
  it('读入三级数据：状态行给出规模，默认落在清单列表，详情空态', async () => {
    await renderReady()
    const status = screen.getByTestId('cost-status')
    expectText(status, '项目')
    expectText(status, '清单')
    expectText(status, '定额')
    expectText(status, '工料机')
    const billTier = screen.getByTestId('cost-tier-bill')
    expectAriaSelected(billTier)
    expect(billTier.querySelector('[data-cost-tier-icon="bill"]')).toBeTruthy()
    expect(screen.getByTestId('cost-tier-quota').querySelector('[data-cost-tier-icon="quota"]')).toBeTruthy()
    expect(screen.getByTestId('cost-tier-resource').querySelector('[data-cost-tier-icon="resource"]')).toBeTruthy()
    // 累计清单：两个项目的行都在，带项目列。
    expect(within(screen.getByTestId('cost-bill-rows')).getAllByRole('row')).toHaveLength(4) // 表头 + 3 行
    expect(screen.getAllByText('什邡地块治理').length).toBeGreaterThanOrEqual(1) // 左栏项目 + 表格项目列
    expect(screen.getAllByText('¥171').length).toBe(2) // 两个项目各一行现算单价（参考价）
    expect(screen.getByText('手填')).toBeTruthy() // 未套定额的手填行
  })

  it('下钻：清单项综合单价分析 → 定额码 → 定额含量行 → 工料机价格卡；引用清单可回链', async () => {
    await renderReady()
    fireEvent.click(screen.getAllByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' })[0]!)
    const bill = await screen.findByTestId('cost-detail-bill')
    expectText(bill, '原表单价')
    expectText(bill, 'G Q0001001') // 库内自有码与来源码并显（SchemaV39）
    expect((await within(bill).findAllByText('¥162')).length).toBeGreaterThan(0) // 0.45 × 360
    const fees = within(bill).getByTestId('cost-fee-summary')
    expect(Array.from(fees.querySelectorAll('.cl-fee-item'), (el) => el.textContent)).toEqual([
      '人工 ¥9', '材料 ¥162', '机械 ¥0',
    ])
    expect(mocks.WorkcostQuotaGet).toHaveBeenCalledWith('WP01')

    // 定额码下钻：切到定额级并选中该定额。
    fireEvent.click(within(bill).getByRole('button', { name: '下钻到定额 WP01' }))
    const quota = await screen.findByTestId('cost-detail-quota')
    expectAriaSelected(screen.getByTestId('cost-tier-quota'))
    expectText(quota, '施工便道')
    expectText(quota, '原码 WP-1')
    expectText(quota, 'G D0101001')
    expectText(quota, '被 2 条清单项引用')
    expect(mocks.WorkcostQuotaCompose).toHaveBeenCalledWith('WP01', null)

    // 含量行下钻到工料机。
    fireEvent.click(within(quota).getByRole('button', { name: '下钻到工料机 C20商品混凝土' }))
    const res = await screen.findByTestId('cost-detail-resource')
    expectAriaSelected(screen.getByTestId('cost-tier-resource'))
    expect(res.className).toContain('cl-detail-resource')
    expectText(res, '核算取用价')
    expectText(res, 'G RC010001')
    expectText(res, '¥360')
    expectText(res, '¥345') // 基准价
    expectText(res, '核算价格')
    expectText(res, '现行价优先')
    expect(await within(res).findByText('2026年第11期')).toBeTruthy()
    expectText(within(res).getByTestId('cost-resource-usage'), 'WP01')

    // 回链：工料机的引用方 → 定额 → 引用它的清单项。
    fireEvent.click(within(res).getByRole('button', { name: /施工便道.*WP01/ }))
    const quota2 = await screen.findByTestId('cost-detail-quota')
    const usage = await within(quota2).findByTestId('cost-quota-usage')
    const items = within(usage).getAllByRole('button')
    expect(items).toHaveLength(2)
    fireEvent.click(items[1]!)
    const bill2 = await screen.findByTestId('cost-detail-bill')
    expectAriaSelected(screen.getByTestId('cost-tier-bill'))
    expectText(bill2, '什邡地块治理')
  })

  it('同级点一行看详情：这一行在当前筛选结果里，筛选与当前位置都保留', async () => {
    await renderReady()
    fireEvent.click(screen.getByRole('button', { name: '什邡地块治理1' }))
    await waitFor(() => expect(within(screen.getByTestId('cost-bill-rows')).getAllByRole('row')).toHaveLength(2))
    fireEvent.click(screen.getByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' }))
    const bill = await screen.findByTestId('cost-detail-bill')
    expectText(bill, '什邡地块治理')
    // 筛选没被冲掉：还是只看什邡的 1 条，当前位置还在。
    expectText(screen.getByTestId('cost-list-count'), '匹配 1 / 3')
    expectText(screen.getByTestId('cost-crumb'), '什邡地块治理')
    expect(within(screen.getByTestId('cost-bill-rows')).getAllByRole('row')).toHaveLength(2)
  })

  it('跨级跳转：目标行过不了那一级的当前筛选时才清空，保证它在列表里可见', async () => {
    await renderReady()
    // 先在定额级留一个会把目标筛掉的关键词。
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    fireEvent.change(screen.getByRole('textbox', { name: '搜索定额' }), { target: { value: '企业' } })
    expect(within(screen.getByTestId('cost-quota-rows')).getAllByRole('row')).toHaveLength(2) // 表头 + 企业定额
    // 回清单级，从详情点定额码下钻：WP01 被「企业」筛掉 → 清空筛选，WP01 出现在列表里。
    fireEvent.click(screen.getByTestId('cost-tier-bill'))
    await screen.findByTestId('cost-bill-rows')
    fireEvent.click(screen.getAllByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' })[0]!)
    const bill = await screen.findByTestId('cost-detail-bill')
    fireEvent.click(within(bill).getByRole('button', { name: '下钻到定额 WP01' }))
    const quota = await screen.findByTestId('cost-detail-quota')
    expectText(quota, '施工便道')
    expect((screen.getByRole('textbox', { name: '搜索定额' }) as HTMLInputElement).value).toBe('')
    expect(within(screen.getByTestId('cost-quota-rows')).getAllByRole('row')).toHaveLength(3) // 表头 + 2 条定额
    // 目标本就过筛时不清：定额级搜「便道」，再从清单详情下钻到 WP01。
    fireEvent.change(screen.getByRole('textbox', { name: '搜索定额' }), { target: { value: '便道' } })
    fireEvent.click(screen.getByTestId('cost-tier-bill'))
    await screen.findByTestId('cost-bill-rows')
    fireEvent.click(screen.getAllByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' })[0]!)
    await screen.findByTestId('cost-detail-bill')
    fireEvent.click(within(screen.getByTestId('cost-detail-bill')).getByRole('button', { name: '下钻到定额 WP01' }))
    await screen.findByTestId('cost-detail-quota')
    expect((screen.getByRole('textbox', { name: '搜索定额' }) as HTMLInputElement).value).toBe('便道')
    expect(within(screen.getByTestId('cost-quota-rows')).getAllByRole('row')).toHaveLength(2) // 表头 + WP01
  })

  it('左栏项目筛选：只看该项目的清单，详情显示项目封面卡（可编辑口径与费率）', async () => {
    await renderReady()
    fireEvent.click(screen.getByRole('button', { name: '什邡地块治理1' }))
    await waitFor(() => expect(within(screen.getByTestId('cost-bill-rows')).getAllByRole('row')).toHaveLength(2))
    expectText(screen.getByTestId('cost-list-count'), '匹配 1 / 3') // 该项目 1 条，占累计 3 条
    const card = screen.getByTestId('cost-project-card')
    expectText(card, '什邡地块治理')
    expectText(card, '10%')
    expectText(card, '7%')
  })

  it('搜索过滤当前级；无匹配时给出清除筛选', async () => {
    await renderReady()
    const search = screen.getByRole('textbox', { name: '搜索清单' })
    fireEvent.change(search, { target: { value: '外委' } })
    expect(within(screen.getByTestId('cost-bill-rows')).getAllByRole('row')).toHaveLength(2)
    fireEvent.change(search, { target: { value: '查无此项' } })
    const empty = screen.getByTestId('cost-tier-empty')
    expectText(empty, '没有匹配的行')
    fireEvent.click(within(empty).getByRole('button', { name: '清除筛选' }))
    expect(within(screen.getByTestId('cost-bill-rows')).getAllByRole('row')).toHaveLength(4)
  })

  it('表头排序：三级列表都支持升序 / 降序 / 恢复默认，空值不乱序', async () => {
    await renderReady()
    const bill = screen.getByTestId('cost-bill-rows')
    const billPrice = within(bill).getByRole('button', { name: '按现算单价排序' })
    fireEvent.click(billPrice)
    expect(billPrice.closest('th')?.getAttribute('aria-sort')).toBe('ascending')
    expectText(within(bill).getAllByRole('row')[1], '施工便道')
    fireEvent.click(billPrice)
    expect(billPrice.closest('th')?.getAttribute('aria-sort')).toBe('descending')
    expectText(within(bill).getAllByRole('row')[1], '外委检测')
    fireEvent.click(billPrice)
    expect(billPrice.closest('th')?.getAttribute('aria-sort')).toBe('none')

    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    const quota = await screen.findByTestId('cost-quota-rows')
    const quotaCode = within(quota).getByRole('button', { name: '按编码排序' })
    fireEvent.click(quotaCode)
    expectText(within(quota).getAllByRole('row')[1], 'ENT-01')
    fireEvent.click(quotaCode)
    expectText(within(quota).getAllByRole('row')[1], 'WP01')

    fireEvent.click(screen.getByTestId('cost-tier-resource'))
    const resource = await screen.findByTestId('cost-resource-rows')
    const resourcePrice = within(resource).getByRole('button', { name: '按取用价排序' })
    fireEvent.click(resourcePrice)
    expectText(within(resource).getAllByRole('row')[1], '普通工')
    fireEvent.click(resourcePrice)
    expectText(within(resource).getAllByRole('row')[1], 'C20商品混凝土')
  })

  it('归并视图：同名同特征同单位聚成库项，展开见各项目的成员行', async () => {
    await renderReady()
    fireEvent.click(screen.getByRole('button', { name: '按库项归并' }))
    const grouped = screen.getByTestId('cost-bill-grouped')
    expectText(grouped, '被 2 个项目使用')
    expect(within(grouped).queryByText('什邡地块治理')).toBeNull() // 收起时列表里只见库项（左栏项目名不算）
    fireEvent.click(within(grouped).getByRole('button', { name: /施工便道/ }))
    expectText(grouped, '什邡地块治理')
    expectText(grouped, '¥430') // 什邡行的原表单价
  })
})

describe('造价数据库 · 删除', () => {
  it('取消不删；确认后逐条删除并静默重读', async () => {
    await renderReady()
    const row = screen.getByText('外委检测').closest('tr')!
    fireEvent.click(within(row).getByRole('checkbox', { name: '选择 外委检测' }))
    fireEvent.click(screen.getByTestId('cost-delete-checked'))
    const dialog = await screen.findByRole('dialog')
    expectText(dialog, '删除清单项')
    expectText(dialog, '「外委检测」')
    fireEvent.click(within(dialog).getByRole('button', { name: /取\s*消/ }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    expect(mocks.WorkcostBillItemDelete).not.toHaveBeenCalled()

    // 取消后勾选保留，直接再发起。
    fireEvent.click(screen.getByTestId('cost-delete-checked'))
    const dialog2 = await screen.findByRole('dialog')
    const before = mocks.WorkcostBillProjects.mock.calls.length
    fireEvent.click(within(dialog2).getByRole('button', { name: '删除 1 条' }))
    await waitFor(() => expect(mocks.WorkcostBillItemDelete).toHaveBeenCalledWith(12))
    // 确认后静默重读（不回加载态）。
    await waitFor(() => expect(mocks.WorkcostBillProjects.mock.calls.length).toBeGreaterThan(before))
    expect(screen.queryByTestId('cost-loading')).toBeNull()
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  })

  it('后端拒绝（定额仍被清单引用）：原因留在确认卡上，不假装成功', async () => {
    await renderReady()
    mocks.WorkcostQuotaDelete.mockRejectedValue(new Error('该定额仍被 2 条清单项引用，不能删除'))
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    fireEvent.click(screen.getByRole('button', { name: '删除定额「施工便道」' }))
    const dialog = await screen.findByRole('dialog')
    expectText(dialog, '仍被清单项引用的定额会被拒绝')
    fireEvent.click(within(dialog).getByRole('button', { name: '删除 1 条' }))
    expectText(await within(dialog).findByTestId('cost-delete-error'), '仍被 2 条清单项引用')
    expectText(screen.getByTestId('cost-quota-rows'), '施工便道') // 行还在
  })

  it('整体删除项目：确认卡写明规模与后果，确认后调用并重读', async () => {
    await renderReady()
    fireEvent.click(screen.getByRole('button', { name: '删除项目 什邡地块治理' }))
    const dialog = await screen.findByRole('dialog')
    expectText(dialog, '全部 1 条分部分项清单')
    fireEvent.click(within(dialog).getByRole('button', { name: '删除项目' }))
    await waitFor(() => expect(mocks.WorkcostBillProjectDelete).toHaveBeenCalledWith(2))
  })
})

describe('造价数据库 · 导出与体检', () => {
  it('导出：选目录后按当前筛选调绑定，回显写出路径；取消选目录不算失败', async () => {
    await renderReady()
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: '导出' })) })
    const menu = await waitFor(() => {
      const el = Array.from(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu')).at(-1)
      if (!el) throw new Error('导出菜单未打开')
      return el
    })
    const items = Array.from(menu.querySelectorAll('.ant-dropdown-menu-item')).map((i) => i.textContent?.trim())
    expect(items).toEqual(['测算模版（样表 + 使用说明书）', '清单（当前筛选）', '定额（当前筛选）', '工料机（当前筛选）'])
    fireEvent.click(Array.from(menu.querySelectorAll('.ant-dropdown-menu-item')).at(-1)!)
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(1))
    const [kind, filter, out] = mocks.CostLibraryExport.mock.calls[0]!
    expect([kind, filter]).toEqual(['resource', EMPTY_FILTER])
    expect(out.startsWith('D:\\新工作\\导出/造价库-工料机-')).toBe(true)
    expect(out.endsWith('.xlsx')).toBe(true)
    expectText(await screen.findByTestId('cost-export-note'), '已导出')

    // 取消选目录：不调导出，也不报错。
    mocks.PickDirectory.mockResolvedValueOnce('')
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: '导出' })) })
    await waitFor(() => expect(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden)').length).toBeGreaterThan(0))
    fireEvent.click(Array.from(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item')).at(0)!)
    await waitFor(() => expect(mocks.PickDirectory).toHaveBeenCalledTimes(2))
    expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(1)
    expectText(screen.getByTestId('cost-export-note'), '已导出')
  })

  it('导出测算模版：回执点名旁边绑定的使用说明书', async () => {
    await renderReady()
    mocks.PickDirectory.mockResolvedValue('D:/导出')
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: '导出' })) })
    await waitFor(() => expect(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item').length).toBeGreaterThan(0))
    fireEvent.click(Array.from(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item')).at(0)!)
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(1))
    expect(mocks.CostLibraryExport.mock.calls[0]!.slice(0, 2)).toEqual(['template', EMPTY_FILTER])
    expectText(await screen.findByTestId('cost-export-note'), '使用说明书：同目录同名 -使用说明.md')
  })

  it('导出按当前筛选传参：清单带项目与分部，定额带专业', async () => {
    await renderReady()
    mocks.PickDirectory.mockResolvedValue('D:/导出')
    fireEvent.change(screen.getByRole('combobox', { name: '筛选分部' }), { target: { value: 'A临建' } })
    mocks.CostLibraryExport.mockRejectedValueOnce(new Error('磁盘已满'))
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: '导出' })) })
    await waitFor(() => expect(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item').length).toBeGreaterThan(0))
    fireEvent.click(Array.from(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item')).at(1)!)
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(1))
    expect(mocks.CostLibraryExport.mock.calls[0]!.slice(0, 2)).toEqual(['bill', { ...EMPTY_FILTER, division: 'A临建' }])
    expectText(screen.getByTestId('cost-export-note'), '导出失败：磁盘已满')
  })

  it('体检抽屉：列出发现，点条目跳到对应行', async () => {
    await renderReady()
    fireEvent.click(screen.getByRole('button', { name: '体检' }))
    const report = await screen.findByTestId('cost-health-report')
    expectText(report, '共 1 项发现')
    expectText(screen.getByTestId('cost-health-zeroPrice'), '待询价材料')
    fireEvent.click(within(screen.getByTestId('cost-health-zeroPrice')).getByRole('button', { name: /待询价材料/ }))
    const res = await screen.findByTestId('cost-detail-resource')
    expectAriaSelected(screen.getByTestId('cost-tier-resource'))
    expectText(res, '核算取用价')
    expectText(res, '现行价与基准价均为 0')
  })
})

describe('造价数据库 · 常驻与四态', () => {
  it('板块 keepAlive：切走再切回时静默重读（不清空正在看的内容）', async () => {
    await renderReady()
    fireEvent.click(screen.getAllByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' })[0]!)
    await screen.findByTestId('cost-detail-bill')
    const before = mocks.WorkcostBillProjects.mock.calls.length
    expect(mocks.WorkcostBillProjects).toHaveBeenCalledTimes(before)
    await act(async () => { notifyBoardActive('gaea') })
    expect(mocks.WorkcostBillProjects).toHaveBeenCalledTimes(before)
    await act(async () => { notifyBoardActive('cost') })
    await waitFor(() => expect(mocks.WorkcostBillProjects.mock.calls.length).toBe(before + 1))
    expect(screen.getByTestId('cost-detail-bill')).toBeTruthy()
    expect(screen.queryByTestId('cost-loading')).toBeNull()
  })

  it('三级全部读失败：显示读取失败与重试，不伪装成空库', async () => {
    mocks.WorkcostBillProjects.mockRejectedValue(new Error('database is locked'))
    mocks.WorkcostQuotaList.mockRejectedValue(new Error('database is locked'))
    mocks.WorkcostResourceList.mockRejectedValue(new Error('database is locked'))
    render(<CostLibraryPage />)
    const failed = await screen.findByTestId('cost-failed')
    expectText(failed, '造价库读取失败')
    expectText(failed, 'database is locked')
    expect(screen.queryByTestId('cost-empty')).toBeNull()
    mocks.WorkcostBillProjects.mockResolvedValue([{ ...p1 }, { ...p2 }])
    mocks.WorkcostQuotaList.mockResolvedValue([{ ...quotaWP01 }])
    mocks.WorkcostResourceList.mockResolvedValue([{ ...c20 }])
    fireEvent.click(within(failed).getByRole('button', { name: /重试/ }))
    expect(await screen.findByTestId('cost-bill-rows')).toBeTruthy()
  })

  it('部分来源读失败：列表照常，状态行点名哪一路失败', async () => {
    mocks.WorkcostQuotaList.mockRejectedValue(new Error('定额表读失败'))
    await renderReady()
    expectText(screen.getByTestId('cost-status'), '定额读取失败')
    expect(screen.getByTestId('cost-bill-rows')).toBeTruthy()
  })

  it('空库：三步引导、可复制的示例提示词、去办公板块', async () => {
    mocks.WorkcostBillProjects.mockResolvedValue([])
    mocks.WorkcostBillItems.mockResolvedValue([])
    mocks.WorkcostQuotaList.mockResolvedValue([])
    mocks.WorkcostResourceList.mockResolvedValue([])
    mocks.WorkcostQuotaComposeMany.mockResolvedValue({})
    render(<CostLibraryPage />)
    const empty = await screen.findByTestId('cost-empty')
    expectText(empty, '造价库还是空的')
    expectText(empty, '在办公板块切到「造价」运行模式')
    expectText(empty, '录入造价库')
    expectText(empty, '造价资料格式不限')
    expect(screen.queryByRole('searchbox')).toBeNull()
    const nav = vi.fn()
    window.addEventListener('navigate', nav)
    fireEvent.click(within(empty).getByRole('button', { name: /去办公板块/ }))
    expect(nav).toHaveBeenCalledTimes(1)
    window.removeEventListener('navigate', nav)
  })
})

describe('造价数据库 · 维度（专业 / 分部 / 分类 / 地区 / 期）', () => {
  function railBtn(testid: string, name: RegExp): HTMLElement {
    return within(screen.getByTestId(testid)).getByRole('button', { name })
  }
  function rowsOf(testid: string): number {
    return within(screen.getByTestId(testid)).getAllByRole('row').length
  }
  async function pickExport(index: number): Promise<void> {
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: '导出' })) })
    await waitFor(() => expect(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item').length).toBeGreaterThan(0))
    fireEvent.click(Array.from(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item')).at(index)!)
  }

  it('清单按专业：左栏专业 → 项目随专业变，「未标注」单列一桶，当前位置显示专业', async () => {
    mocks.WorkcostBillItems.mockImplementation(async (projectId: number) => projectId === 1
      ? [{ ...row11, specialty: '土壤修复' }, { ...row12 }]
      : [{ ...row13, specialty: '市政' }])
    await renderReady()
    expectText(screen.getByTestId('cost-rail-bill-specialties'), '未标注专业')
    expectText(screen.getByTestId('cost-bill-rows'), '工程量') // 列表有专业与工程量列
    fireEvent.click(railBtn('cost-rail-bill-specialties', /^市政/))
    await waitFor(() => expect(rowsOf('cost-bill-rows')).toBe(2))
    expectText(screen.getByTestId('cost-list-count'), '匹配 1 / 3')
    expectText(screen.getByTestId('cost-crumb'), '市政')
    // 项目列表随专业变：这个专业只有什邡的行。
    const projects = screen.getByTestId('cost-rail-projects')
    expect(within(projects).queryByText('旺平矿业修复')).toBeNull()
    fireEvent.click(within(projects).getByRole('button', { name: /^什邡地块治理/ }))
    expect(textOf(screen.getByTestId('cost-crumb'))).toContain('什邡')
    // 先选了项目再换专业：回到全部项目，不会被卡在空列表里。
    fireEvent.click(railBtn('cost-rail-bill-specialties', /^土壤修复/))
    expect(textOf(screen.getByTestId('cost-crumb'))).not.toContain('什邡')
    expect(within(screen.getByTestId('cost-rail-projects')).getByText('旺平矿业修复')).toBeTruthy()
    fireEvent.click(railBtn('cost-rail-bill-specialties', /未标注专业/))
    expectText(screen.getByTestId('cost-list-count'), '匹配 1 / 3')
  })

  it('定额：先选专业才出分部这一级；分部在专业内筛选，换专业时分部筛选跟着清；列表有地区列', async () => {
    const sz: WorkcostQuota = { ...quotaWP01, id: 3, code: 'SZ01', origCode: '', title: '沥青路面', specialty: '市政', chapter: '道路工程' }
    const unlabeled: WorkcostQuota = { ...quotaWP01, id: 4, code: 'Q9', origCode: '', title: '没标专业的定额', specialty: '', chapter: '' }
    mocks.WorkcostQuotaList.mockResolvedValue([
      { ...quotaWP01, items: undefined, usageCount: 2 }, { ...quotaFree, items: undefined, chapter: '', usageCount: 0 },
      { ...sz, items: undefined }, { ...unlabeled, items: undefined },
    ])
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    expect(screen.queryByTestId('cost-rail-quota-chapters')).toBeNull() // 没选专业：不出分部
    expect(rowsOf('cost-quota-rows')).toBe(5) // 表头 + 4
    expectText(screen.getByTestId('cost-quota-rows'), '分部') // 列表有分部列
    expectText(screen.getByTestId('cost-quota-rows'), '地区') // 列表有地区列（随上游项目挂钩，空=未标地区）
    fireEvent.click(railBtn('cost-rail-quota-specialties', /^土壤修复/))
    const chapters = await screen.findByTestId('cost-rail-quota-chapters')
    expectText(chapters, 'A临建')
    expectText(chapters, '未归分部')
    expect(rowsOf('cost-quota-rows')).toBe(3) // 表头 + 土壤修复的 2 条
    fireEvent.click(railBtn('cost-rail-quota-chapters', /^A临建/))
    expect(rowsOf('cost-quota-rows')).toBe(2)
    expectText(screen.getByTestId('cost-crumb'), 'A临建')
    fireEvent.click(railBtn('cost-rail-quota-specialties', /^市政/))
    expect(textOf(screen.getByTestId('cost-crumb'))).not.toContain('A临建')
    expectText(screen.getByTestId('cost-rail-quota-chapters'), '道路工程')
    expect(rowsOf('cost-quota-rows')).toBe(2) // 市政的 1 条（分部筛选已清）
  })

  it('工料机：类别 → 分类（选上级连带下级）；工具条按地区、期数筛选；列表有地区与期两列', async () => {
    const cq: WorkcostResource = { ...c20, id: 104, code: 'C30', title: 'C30商品混凝土', spec: '泵送', categoryPath: '材料/混凝土/泵送', region: '重庆市', priceDate: '2025-12' }
    const bare: WorkcostResource = { ...c20, id: 105, code: 'SAND', title: '中砂', spec: '', categoryPath: '', region: '', priceDate: '' }
    mocks.WorkcostResourceList.mockResolvedValue([
      { ...c20, region: '四川省', priceDate: '2026-06' }, { ...l01, region: '', priceDate: '', categoryPath: '' }, { ...cq }, { ...bare },
    ])
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-resource'))
    await screen.findByTestId('cost-resource-rows')
    expect(rowsOf('cost-resource-rows')).toBe(5) // 表头 + 4
    expectText(screen.getByTestId('cost-resource-rows'), '四川省')
    expectText(screen.getByTestId('cost-resource-rows'), '2026年6月') // 库里存 YYYY-MM，界面显示年月
    const options = (testid: string) => Array.from(screen.getByTestId(testid).querySelectorAll('option')).map((o) => o.textContent)
    expect(options('cost-select-region')).toEqual(['全部地区', '重庆市（1）', '四川省（1）', '未标注（2）'])
    expect(options('cost-select-period')).toEqual(['全部期', '2026年6月（1）', '2025年12月（1）', '无期（2）'])
    fireEvent.change(screen.getByTestId('cost-select-region'), { target: { value: '重庆市' } })
    expect(rowsOf('cost-resource-rows')).toBe(2)
    expectText(screen.getByTestId('cost-crumb'), '重庆市')
    fireEvent.change(screen.getByTestId('cost-select-period'), { target: { value: '__none__' } })
    expect(screen.getByTestId('cost-tier-empty')).toBeTruthy() // 重庆那条有期：加上「无期」就无匹配
    fireEvent.click(within(screen.getByTestId('cost-tier-empty')).getByRole('button', { name: '清除筛选' }))
    expect(rowsOf('cost-resource-rows')).toBe(5)
    expect(screen.queryByTestId('cost-rail-categories')).toBeNull() // 没选类别：不出分类
    fireEvent.click(railBtn('cost-rail-kinds', /^材料/))
    const cats = await screen.findByTestId('cost-rail-categories')
    expectText(cats, '未细分')
    expect(rowsOf('cost-resource-rows')).toBe(4) // 表头 + 3 条材料
    fireEvent.click(within(cats).getAllByRole('button', { name: /^混凝土/ })[0]!)
    expect(rowsOf('cost-resource-rows')).toBe(3) // 选上级连带下级：C20 与 C30
    fireEvent.click(within(cats).getByRole('button', { name: /^泵送/ }))
    expect(rowsOf('cost-resource-rows')).toBe(2)
    expect(textOf(screen.getByTestId('cost-crumb'))).toContain('材料›混凝土›泵送')
    expectText(screen.getByTestId('cost-resource-rows'), '2025年12月')
  })

  it('类别待整理：类别不在四类之内的工料机在左栏单列一桶，能筛出来，导出同口径；当前位置里的期数显示成年月', async () => {
    const note: WorkcostResource = {
      ...c20, id: 106, code: 'NOTE-1', kind: '一段备注：地磅设备摊销为估价', title: '错记成工料机的说明行', spec: '', unit: '项',
      categoryPath: '', region: '重庆市', priceDate: '2026-06',
    }
    mocks.WorkcostResourceList.mockResolvedValue([{ ...c20 }, { ...note }])
    mocks.PickDirectory.mockResolvedValue('D:/导出')
    await renderReady()
    expectText(screen.getByTestId('cost-pending'), '待整理 5') // 三条清单无专业 + 这一条类别待整理 + 它还没细分
    expect(screen.getByTestId('cost-pending').getAttribute('title')).toContain('工料机类别待整理 1')
    fireEvent.click(screen.getByTestId('cost-tier-resource'))
    await screen.findByTestId('cost-resource-rows')
    expect(rowsOf('cost-resource-rows')).toBe(3) // 表头 + 2
    expect(within(screen.getByTestId('cost-rail-kinds')).queryByRole('button', { name: /^人工/ })).toBeNull()
    fireEvent.click(railBtn('cost-rail-kinds', /^类别待整理/))
    expect(rowsOf('cost-resource-rows')).toBe(2) // 只剩那条
    expectText(screen.getByTestId('cost-resource-rows'), '错记成工料机的说明行')
    expectText(screen.getByTestId('cost-crumb'), '类别待整理')
    expect(screen.queryByTestId('cost-rail-categories')).toBeNull() // 这一桶下面没有分类一级
    fireEvent.change(screen.getByTestId('cost-select-period'), { target: { value: '2026-06' } })
    expect(textOf(screen.getByTestId('cost-crumb'))).toContain('2026年6月')
    expect(textOf(screen.getByTestId('cost-crumb'))).not.toContain('2026-06')
    await pickExport(3) // 工料机（当前筛选）；第 0 位是测算模版
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(1))
    expect(mocks.CostLibraryExport.mock.calls[0]!.slice(0, 2)).toEqual(['resource', { ...EMPTY_FILTER, kind: '__outside__', period: '2026-06' }])
  })

  it('待整理：状态行给出未归类的条数；点开逐项列出，点一项直接筛出对应的行；整理提示词仍可复制', async () => {
    await renderReady()
    const pending = screen.getByTestId('cost-pending')
    expectText(pending, '待整理 3') // 三条清单行都还没标专业
    expect(pending.getAttribute('title')).toContain('清单无专业 3')
    fireEvent.click(pending)
    const item = await screen.findByTestId('cost-pending-billSpecialty')
    expectText(item, '清单无专业')
    expectText(item, '3')
    // 点一项 = 落到清单级、只留「未标注专业」这一项的筛选；三条都在
    fireEvent.click(item)
    await waitFor(() => expect(screen.queryByTestId('cost-pending-pop')).toBeNull())
    expect(within(screen.getByTestId('cost-bill-rows')).getAllByRole('row')).toHaveLength(4) // 表头 + 3 条
    // 整理提示词仍可复制给造价模式
    fireEvent.click(screen.getByTestId('cost-pending'))
    const copy = await screen.findByTestId('cost-pending-copy')
    fireEvent.click(copy)
    await waitFor(() => expect(copyToClipboard).toHaveBeenCalledWith(PENDING_PROMPT))
    expectText(await screen.findByTestId('cost-pending-copy'), '已复制提示词')
  })

  it('索引优化（2026-11-28）· 多词搜索：空白分开的词要求每个都命中', async () => {
    await renderReady()
    const search = screen.getByRole('textbox', { name: '搜索清单' })
    fireEvent.change(search, { target: { value: '便道 泥结' } })
    // 「施工便道」两条（项目特征都含「泥结碎石」）；「外委检测」没有「便道」
    expect(within(screen.getByTestId('cost-bill-rows')).getAllByRole('row')).toHaveLength(3)
    fireEvent.change(search, { target: { value: '便道 外委' } })
    await screen.findByTestId('cost-tier-empty')
    expectText(screen.getByTestId('cost-tier-empty'), '没有匹配的行')
  })

  it('索引优化（2026-11-28）· 分面联动：搜索时左栏计数只数命中的行，项目列表给每个项目剩多少', async () => {
    await renderReady()
    expectText(screen.getByTestId('cost-rail-bill-specialties'), '未标注专业')
    expect(screen.getByTestId('cost-rail-bill-specialties').textContent).toContain('3')
    fireEvent.change(screen.getByRole('textbox', { name: '搜索清单' }), { target: { value: '便道' } })
    await waitFor(() => expect(screen.getByTestId('cost-rail-bill-specialties').textContent).toContain('2'))
    // 两个项目各剩一条；「外委检测」所在的旺平项目也只剩一条
    const projects = screen.getByTestId('cost-rail-projects').textContent ?? ''
    expect(projects).toContain('1')
    expect(within(screen.getByTestId('cost-bill-rows')).getAllByRole('row')).toHaveLength(3)
  })

  it('索引优化（2026-11-28）· 状态筛选：清单选「未套定额」只剩手填行；当前位置点明', async () => {
    await renderReady()
    const state = screen.getByTestId('cost-select-state')
    fireEvent.change(state, { target: { value: 'manual' } })
    const rows = within(screen.getByTestId('cost-bill-rows')).getAllByRole('row')
    expect(rows).toHaveLength(2) // 表头 + 外委检测
    expectText(screen.getByTestId('cost-bill-rows'), '外委检测')
    expectText(screen.getByTestId('cost-crumb'), '未套定额（手填）')
    // 清除筛选回到全部
    fireEvent.change(state, { target: { value: '' } })
    expect(within(screen.getByTestId('cost-bill-rows')).getAllByRole('row')).toHaveLength(4)
  })

  it('索引优化（2026-11-28）· 定额级地区筛选：左栏与工具条都能按地区筛，没标的单列一桶', async () => {
    mocks.WorkcostQuotaList.mockResolvedValue([
      { ...quotaWP01, region: '四川省', items: undefined, usageCount: 2 },
      { ...quotaFree, region: '', items: undefined, usageCount: 0 },
    ])
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    const rail = screen.getByTestId('cost-rail-quota-regions')
    expectText(rail, '四川省')
    expectText(rail, '未标地区')
    fireEvent.change(screen.getByTestId('cost-select-quota-region'), { target: { value: '__none__' } })
    const rows = within(screen.getByTestId('cost-quota-rows')).getAllByRole('row')
    expect(rows).toHaveLength(2) // 表头 + 企业定额（未引用）
    expectText(screen.getByTestId('cost-quota-rows'), '企业定额（未引用）')
  })

  it('清单详情：标注专业；每条定额的消耗量 = 该定额工程量 × 含量；挂了外专业定额要点名', async () => {
    mocks.WorkcostBillItems.mockImplementation(async (projectId: number) => projectId === 1
      ? [{ ...row11, specialty: '土壤修复', linkQuantities: { WP01: 100 } }]
      : [{ ...row13, specialty: '土壤修复' }])
    const other: WorkcostQuota = { ...quotaWP01, specialty: '市政' }
    mocks.WorkcostQuotaList.mockResolvedValue([{ ...other, items: undefined, usageCount: 1 }, { ...quotaFree, items: undefined, usageCount: 0 }])
    mocks.WorkcostQuotaGet.mockResolvedValue(other)
    await renderReady()
    fireEvent.click(screen.getAllByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' })[0]!)
    const bill = await screen.findByTestId('cost-detail-bill')
    expectText(within(bill).getByTestId('cost-bill-specialty'), '土壤修复')
    expectText(within(bill).getByTestId('cost-bill-cross-specialty'), '外专业 · 市政')
    expectText(within(bill).getByTestId('cost-bill-cross-note'), '1 条定额属于别的专业')
    // 定额工程量取挂接明细的 100（不是清单工程量 850）：0.45×100、0.03×100。
    expect(within(bill).getAllByTestId('cost-consumption').map((c) => textOf(c))).toEqual(['45 m³', '3 工日'])
    expectText(bill, '消耗量')
  })

  it('导出带上维度筛选：未标注哨兵、地区、期数原样传给后端', async () => {
    mocks.PickDirectory.mockResolvedValue('D:/导出')
    await renderReady()
    fireEvent.click(railBtn('cost-rail-bill-specialties', /未标注专业/))
    await pickExport(1)
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(1))
    expect(mocks.CostLibraryExport.mock.calls[0]!.slice(0, 2)).toEqual(['bill', { ...EMPTY_FILTER, specialty: '__none__' }])

    fireEvent.click(screen.getByTestId('cost-tier-resource'))
    await screen.findByTestId('cost-resource-rows')
    fireEvent.change(screen.getByTestId('cost-select-region'), { target: { value: '四川省' } })
    await pickExport(3)
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(2))
    expect(mocks.CostLibraryExport.mock.calls[1]!.slice(0, 2)).toEqual(['resource', { ...EMPTY_FILTER, region: '四川省' }])
  })

  it('导出测算模版：kind=template，口径取当前专业与工料机级筛选，「未标注 / 类别待整理」哨兵不当口径', async () => {
    mocks.PickDirectory.mockResolvedValue('D:/导出')
    await renderReady()
    // 清单级：专业未筛选 → 模版专业口径为空。
    await pickExport(0)
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(1))
    expect(mocks.CostLibraryExport.mock.calls[0]!.slice(0, 2)).toEqual(['template', EMPTY_FILTER])

    // 工料机级：地区成为模版口径；「类别待整理」这一桶对模版=没给类别。
    fireEvent.click(screen.getByTestId('cost-tier-resource'))
    await screen.findByTestId('cost-resource-rows')
    fireEvent.change(screen.getByTestId('cost-select-region'), { target: { value: '四川省' } })
    fireEvent.click(within(screen.getByTestId('cost-rail-kinds')).getAllByRole('button', { name: /^材料/ })[0]!)
    await pickExport(0)
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(2))
    expect(mocks.CostLibraryExport.mock.calls[1]!.slice(0, 2)).toEqual(['template', { ...EMPTY_FILTER, kind: '材料', region: '四川省' }])
    expect(String(mocks.CostLibraryExport.mock.calls[1]![2]).startsWith('D:/导出/造价库-测算模版-')).toBe(true)
  })

  it('体检多一类「不合规矩的数据」：字典外的取值点名，工料机的可跳转', async () => {
    mocks.CostHealthScan.mockResolvedValue({
      zeroPrice: [], quotaNoPrice: [], brokenRef: [], duplicates: [], codeMismatch: [], totalFindings: 2,
      ruleIssues: [
        { tier: 'resource', id: 103, code: 'Z01', title: '待询价材料', field: 'region', value: '不在字典的地区', fix: 'x' },
        { tier: 'bill', id: 11, code: 'WP01', title: '施工便道', field: 'specialty', value: '不在字典的专业' },
      ],
    })
    await renderReady()
    fireEvent.click(screen.getByRole('button', { name: '体检' }))
    const report = await screen.findByTestId('cost-health-report')
    expectText(report, '共 2 项发现')
    const section = screen.getByTestId('cost-health-ruleIssues')
    expectText(section, '工料机 Z01「待询价材料」的地区是「不在字典的地区」')
    expectText(section, '清单 WP01「施工便道」的专业是「不在字典的专业」')
    expect(within(section).getAllByRole('button')).toHaveLength(1) // 清单那一行在体检里没有可跳的入口
    fireEvent.click(within(section).getByRole('button', { name: /Z01/ }))
    await screen.findByTestId('cost-detail-resource')
    expectAriaSelected(screen.getByTestId('cost-tier-resource'))
  })
})

describe('造价数据库 · 语义索引（2026-11-26 造价库接入本地模型索引）', () => {
  const readyStatus = (): CostIndexStatusView => ({
    model: 'bge-m3', running: true,
    kinds: [
      { kind: 'wc_quota', label: '定额', total: 2, count: 2, updatedAt: '2026-11-26T08:00:00Z' },
      { kind: 'wc_res', label: '工料机', total: 3, count: 3, updatedAt: '' },
      { kind: 'wc_bill', label: '清单', total: 3, count: 3, updatedAt: '' },
    ],
  })

  it('对账数与向量数都是 0：没有可索引的东西，不出状态钮；也不出语义补充段', async () => {
    await renderReady()
    // 默认对账数与向量数都是 0：没有可索引的东西，不出钮
    expect(screen.queryByTestId('cost-index-status')).toBeNull()
    expect(mocks.WorkcostQuotaList).toHaveBeenCalledTimes(1) // 只有挂载那一次全量读取
  })

  it('索引未建但库里有条目：状态钮置灰写「未建」，搜索照旧只做本地过滤', async () => {
    mocks.CostIndexStatus.mockResolvedValue({
      model: 'bge-m3', running: true,
      kinds: [
        { kind: 'wc_quota', label: '定额', total: 2, count: 0, updatedAt: '' },
        { kind: 'wc_res', label: '工料机', total: 3, count: 0, updatedAt: '' },
        { kind: 'wc_bill', label: '清单', total: 3, count: 0, updatedAt: '' },
      ],
    })
    await renderReady()
    const btn = screen.getByTestId('cost-index-status')
    expect(btn.getAttribute('data-tone')).toBe('empty')
    expectText(btn, '语义索引 未建')
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    fireEvent.change(screen.getByRole('textbox', { name: '搜索定额' }), { target: { value: '企业' } })
    await new Promise((r) => setTimeout(r, 400))
    expect(screen.queryByTestId('cost-semantic')).toBeNull()
    expect(mocks.WorkcostQuotaList).toHaveBeenCalledTimes(1)
  })

  it('索引就绪：状态钮写「就绪」；点击调补齐、重读状态并出提示', async () => {
    mocks.CostIndexStatus.mockResolvedValue(readyStatus())
    mocks.CostIndexBackfill.mockResolvedValue(readyStatus())
    await renderReady()
    const btn = screen.getByTestId('cost-index-status')
    expect(btn.getAttribute('data-tone')).toBe('ok')
    expectText(btn, '语义索引 就绪')
    fireEvent.click(btn)
    await waitFor(() => expect(mocks.CostIndexBackfill).toHaveBeenCalledWith(''))
    await waitFor(() => expect(screen.getByTestId('cost-notice').textContent).toContain('语义索引已补齐'))
    // 挂载读一次 + 补齐后重读一次
    expect(mocks.CostIndexStatus).toHaveBeenCalledTimes(2)
  })

  it('模型不可用：状态钮转警示色；补齐失败把原因留在提示里', async () => {
    const st = readyStatus()
    st.kinds[0] = { ...st.kinds[0], count: 1, lastError: 'embedding 模型不可用' }
    mocks.CostIndexStatus.mockResolvedValue(st)
    mocks.CostIndexBackfill.mockRejectedValue(new Error('定额索引补齐失败: embedding 模型不可用'))
    await renderReady()
    const btn = screen.getByTestId('cost-index-status')
    expect(btn.getAttribute('data-tone')).toBe('warn')
    expectText(btn, '语义索引 待补齐')
    expect(btn.getAttribute('title')).toContain('embedding 模型不可用')
    fireEvent.click(btn)
    await waitFor(() => expect(screen.getByTestId('cost-notice').textContent).toContain('补齐失败：定额索引补齐失败'))
  })

  it('搜索定额：本地匹配在前，「名称不含关键词、意思相近」的条目进补充段', async () => {
    mocks.CostIndexStatus.mockResolvedValue(readyStatus())
    // 混合检索返回：本地已匹配的「施工便道」+ 名称不含「便道」的「场地平整」
    const similar: WorkcostQuota = { ...quotaWP01, id: 3, code: 'Q9', gcode: 'D0100009', origCode: '', title: '场地平整', items: undefined, usageCount: 0 }
    mocks.WorkcostQuotaList.mockResolvedValue([{ ...quotaWP01, items: undefined, usageCount: 2 }, similar])
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    fireEvent.change(screen.getByRole('textbox', { name: '搜索定额' }), { target: { value: '便道' } })
    const sem = await screen.findByTestId('cost-semantic')
    expectText(sem, '名称不含「便道」')
    expectText(sem, '场地平整')
    // 补充段里不该重复本地已匹配的「施工便道」
    expect(sem.textContent).not.toContain('施工便道')
    const tables = screen.getAllByTestId('cost-quota-rows')
    expect(tables).toHaveLength(2) // 主表 + 补充段各一张
    expect(within(tables[0]).getAllByRole('row')).toHaveLength(2) // 表头 + 施工便道
    expect(mocks.WorkcostQuotaList).toHaveBeenLastCalledWith('', '便道')
    // 清空关键词：补充段随之消失
    fireEvent.change(screen.getByRole('textbox', { name: '搜索定额' }), { target: { value: '' } })
    await waitFor(() => expect(screen.queryByTestId('cost-semantic')).toBeNull())
  })

  it('本地零匹配但语义有命中：不出「没有匹配的行」空态，直接给补充段', async () => {
    mocks.CostIndexStatus.mockResolvedValue(readyStatus())
    const similar: WorkcostQuota = { ...quotaWP01, id: 3, code: 'Q9', gcode: 'D0100009', origCode: '', title: '场地平整', items: undefined, usageCount: 0 }
    mocks.WorkcostQuotaList.mockResolvedValue([similar])
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    // 「平地」不是「场地平整」的子串：本地零匹配；语义路把「场地平整」带进来
    fireEvent.change(screen.getByRole('textbox', { name: '搜索定额' }), { target: { value: '平地' } })
    const sem = await screen.findByTestId('cost-semantic')
    expectText(sem, '场地平整')
    expect(screen.queryByTestId('cost-tier-empty')).toBeNull()
    expect(within(sem).getAllByRole('row')).toHaveLength(2) // 表头 + 场地平整
  })

  it('索引优化（2026-11-28）· 搜索呈现：主列表按相关度排、命中词条高亮；补充段的行标「语义」', async () => {
    mocks.CostIndexStatus.mockResolvedValue(readyStatus())
    // 两条名称含「便道」+ 一条意思相近的；后台混合检索给的相关度序：人工便道最靠前
    const manual: WorkcostQuota = { ...quotaWP01, id: 3, code: 'Q-A', gcode: 'D0100003', origCode: '', title: '施工便道（人工）', items: undefined, usageCount: 0 }
    const machine: WorkcostQuota = { ...quotaWP01, id: 4, code: 'Q-B', gcode: 'D0100004', origCode: '', title: '施工便道（机械）', items: undefined, usageCount: 0 }
    const similar: WorkcostQuota = { ...quotaWP01, id: 5, code: 'Q-C', gcode: 'D0100005', origCode: '', title: '场地平整', items: undefined, usageCount: 0 }
    mocks.WorkcostQuotaList.mockResolvedValue([manual, similar, machine])
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    fireEvent.change(screen.getByRole('textbox', { name: '搜索定额' }), { target: { value: '便道' } })
    await screen.findByTestId('cost-semantic')
    const tables = screen.getAllByTestId('cost-quota-rows')
    const main = tables[0].querySelectorAll('tbody tr')
    expect(main).toHaveLength(2)
    expectText(main[0] as HTMLElement, '施工便道（人工）') // 相关度靠前的排在前面
    expectText(main[1] as HTMLElement, '施工便道（机械）')
    // 命中的词条高亮
    const marks = tables[0].querySelectorAll('mark.cl-hit')
    expect(marks.length).toBeGreaterThan(0)
    expect(marks[0]?.textContent).toBe('便道')
    // 补充段的行标「语义」来源；主列表不标
    expect(within(screen.getByTestId('cost-semantic')).getAllByText('语义').length).toBeGreaterThan(0)
    expect(tables[0].textContent).not.toContain('语义')
  })

  it('本地语义模型不可用：搜索后搜索框旁明示、原因在悬停明细；状态钮转警示', async () => {
    // 2026-11-27 的故障：向量模型进程的显卡设备丢了，语义检索静默只剩关键词、20 小时无人知道
    const st = readyStatus()
    const reason = '本地向量模型请求失败：embedding HTTP 500: decode() failed: vk::Queue::submit: ErrorDeviceLost'
    st.kinds[0] = { ...st.kinds[0], searchError: reason }
    mocks.CostIndexStatus.mockResolvedValue(st)
    mocks.WorkcostQuotaList.mockResolvedValue([{ ...quotaWP01, items: undefined, usageCount: 2 }])
    await renderReady()
    expectText(screen.getByTestId('cost-index-status'), '语义模型 不可用')
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    expect(screen.queryByTestId('cost-semantic-down')).toBeNull() // 还没搜：不出提示
    fireEvent.change(screen.getByRole('textbox', { name: '搜索定额' }), { target: { value: '便道' } })
    const warn = await screen.findByTestId('cost-semantic-down')
    expectText(warn, '本地语义模型不可用，只按关键词匹配')
    expect(warn.getAttribute('title')).toBe(reason)
    // 关键词命中照常在，只是没有「意思相近」
    expect(screen.getAllByTestId('cost-quota-rows')[0].textContent).toContain('施工便道')
    expect(screen.queryByTestId('cost-semantic')).toBeNull()
    // 清空关键词：提示随之消失
    fireEvent.change(screen.getByRole('textbox', { name: '搜索定额' }), { target: { value: '' } })
    await waitFor(() => expect(screen.queryByTestId('cost-semantic-down')).toBeNull())
  })

  it('模型恢复后点「重试」：重新检索一次，提示消失', async () => {
    const st = readyStatus()
    st.kinds[0] = { ...st.kinds[0], searchError: '本地向量模型请求失败：embedding HTTP 500' }
    mocks.CostIndexStatus.mockResolvedValue(st)
    mocks.WorkcostQuotaList.mockResolvedValue([{ ...quotaWP01, items: undefined, usageCount: 2 }])
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    fireEvent.change(screen.getByRole('textbox', { name: '搜索定额' }), { target: { value: '便道' } })
    await screen.findByTestId('cost-semantic-down')
    // 模型重载好了（后端下一次检索的结果恢复正常）
    mocks.CostIndexStatus.mockResolvedValue(readyStatus())
    const before = mocks.WorkcostQuotaList.mock.calls.length
    fireEvent.click(screen.getByRole('button', { name: '重试语义检索' }))
    await waitFor(() => expect(screen.queryByTestId('cost-semantic-down')).toBeNull())
    expect(mocks.WorkcostQuotaList.mock.calls.length).toBeGreaterThan(before)
  })
})

describe('造价数据库 · 库页编辑面（2026-11-27）', () => {
  it('清单行编辑：改工程量与特征后保存，绑定收到改动字段并静默重读', async () => {
    await renderReady()
    fireEvent.click(screen.getAllByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' })[0]!)
    const bill = await screen.findByTestId('cost-detail-bill')
    fireEvent.click(within(bill).getByTestId('cost-bill-edit-open'))
    const edit = await screen.findByTestId('cost-bill-edit')
    fireEvent.change(within(edit).getByTestId('cost-edit-bill-quantity'), { target: { value: '999' } })
    fireEvent.click(within(edit).getByTestId('cost-edit-save'))
    await waitFor(() => expect(mocks.WorkcostBillItemSave).toHaveBeenCalledTimes(1))
    const saved = mocks.WorkcostBillItemSave.mock.calls[0][0] as WorkcostBillItem
    expect(saved.id).toBe(11)
    expect(saved.quantity).toBe(999)
    expect(saved.title).toBe('施工便道（铺筑压实）')
    // 保存后：提示 + 静默重读（不清空正在看的内容）
    expect(screen.getByTestId('cost-notice').textContent).toContain('已保存')
    await waitFor(() => expect(mocks.WorkcostBillProjects.mock.calls.length).toBeGreaterThanOrEqual(2))
    // 退出编辑态回只读视图
    await screen.findByTestId('cost-detail-bill')
    expect(screen.queryByTestId('cost-bill-edit')).toBeNull()
  })

  it('清单行挂接编辑：读引用表、解挂与改量走幂等挂接、按编码新挂', async () => {
    await renderReady()
    fireEvent.click(screen.getAllByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' })[0]!)
    const bill = await screen.findByTestId('cost-detail-bill')
    fireEvent.click(within(bill).getByTestId('cost-bill-edit-open'))
    const links = await screen.findByTestId('cost-edit-links')
    expectText(links, 'WP01')
    expect(mocks.WorkcostBillQuotaLinks).toHaveBeenCalledWith(11)
    // 改量：跟随清单工程量(0) → 300
    fireEvent.change(screen.getByTestId('cost-edit-link-qty-WP01'), { target: { value: '300' } })
    fireEvent.click(within(links).getByRole('button', { name: '更新' }))
    await waitFor(() => expect(mocks.WorkcostBillQuotaAttach).toHaveBeenCalledWith(11, 'WP01', 300))
    // 解挂
    fireEvent.click(within(links).getByRole('button', { name: /解挂/ }))
    await waitFor(() => expect(mocks.WorkcostBillQuotaDetach).toHaveBeenCalledWith(11, 5))
    // 按编码新挂
    fireEvent.change(screen.getByLabelText('挂接定额编码'), { target: { value: 'WP02' } })
    fireEvent.click(screen.getByTestId('cost-edit-link-attach'))
    await waitFor(() => expect(mocks.WorkcostBillQuotaAttach).toHaveBeenCalledWith(11, 'WP02', 0))
  })

  it('定额编辑：改含量行数量整组落库；换专业联动分部面', async () => {
    await renderReady()
    fireEvent.click(screen.getAllByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' })[0]!)
    const bill = await screen.findByTestId('cost-detail-bill')
    // 详情里的定额码芯片（列表表格里的同名列不是详情的下钻入口，注意圈定范围）
    fireEvent.click(within(bill).getByRole('button', { name: '下钻到定额 WP01' }))
    const quota = await screen.findByTestId('cost-detail-quota')
    fireEvent.click(within(quota).getByTestId('cost-quota-edit-open'))
    const edit = await screen.findByTestId('cost-quota-edit')
    fireEvent.change(within(edit).getByTestId('cost-edit-item-qty-C20'), { target: { value: '0.5' } })
    // 删掉人工行（整组替换语义）
    fireEvent.click(within(edit).getAllByRole('button', { name: /删行/ })[1]!)
    fireEvent.click(within(edit).getByTestId('cost-edit-save'))
    await waitFor(() => expect(mocks.WorkcostQuotaSave).toHaveBeenCalledTimes(1))
    const saved = mocks.WorkcostQuotaSave.mock.calls[0][0] as WorkcostQuota
    expect(saved.code).toBe('WP01')
    expect(saved.items!.map((it) => it.resourceCode)).toEqual(['C20'])
    expect(saved.items![0].quantity).toBe(0.5)
    expect(screen.getByTestId('cost-notice').textContent).toContain('已保存')
  })

  it('工料机编辑：按期调价留在编辑态并写历史；改现行价保存走 SaveResource', async () => {
    await renderReady()
    fireEvent.click(screen.getAllByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' })[0]!)
    const bill = await screen.findByTestId('cost-detail-bill')
    fireEvent.click(within(bill).getByRole('button', { name: '下钻到定额 WP01' }))
    const quota = await screen.findByTestId('cost-detail-quota')
    // 含量行下钻到工料机（分析行的资源链接，label 用资源名）
    fireEvent.click(within(quota).getByRole('button', { name: '下钻到工料机 C20商品混凝土' }))
    const res = await screen.findByTestId('cost-detail-resource')
    fireEvent.click(within(res).getByTestId('cost-resource-edit-open'))
    const edit = await screen.findByTestId('cost-resource-edit')
    // 按期调价：独立登记动作（写调价历史并推进现行价），登记后留在编辑态
    fireEvent.change(within(edit).getByTestId('cost-edit-res-setprice-price'), { target: { value: '395' } })
    fireEvent.click(within(edit).getByTestId('cost-edit-res-setprice'))
    await waitFor(() => expect(mocks.WorkcostResourceSetPrice).toHaveBeenCalledWith(101, 395, '', '四川省', '信息价', '造价库界面', ''))
    expect(screen.getByTestId('cost-notice').textContent).toContain('已调价')
    expect(screen.getByTestId('cost-resource-edit')).toBeTruthy()
    // 主数据保存：改现行价走 SaveResource，成功后退出编辑态
    fireEvent.change(within(edit).getByTestId('cost-edit-res-price'), { target: { value: '380' } })
    fireEvent.click(within(edit).getByTestId('cost-edit-save'))
    await waitFor(() => expect(mocks.WorkcostResourceSave).toHaveBeenCalledTimes(1))
    const saved = mocks.WorkcostResourceSave.mock.calls[0][0] as WorkcostResource
    expect(saved.id).toBe(101)
    expect(saved.currentPrice).toBe(380)
    await waitFor(() => expect(screen.queryByTestId('cost-resource-edit')).toBeNull())
  })

  it('项目封面编辑：口径与费率分档保存（费率按小数落库），空值保持原口径', async () => {
    await renderReady()
    fireEvent.click(screen.getByRole('button', { name: '什邡地块治理1' }))
    const card = await screen.findByTestId('cost-project-card')
    fireEvent.click(within(card).getByTestId('cost-project-edit-open'))
    const edit = await screen.findByTestId('cost-project-edit')
    fireEvent.change(within(edit).getByTestId('cost-edit-proj-specialty'), { target: { value: '土壤修复' } })
    fireEvent.click(within(edit).getByTestId('cost-edit-proj-profile-save'))
    await waitFor(() => expect(mocks.WorkcostBillProjectProfileSave).toHaveBeenCalledWith(2, '土壤修复', '', ''))
    // 两档独立保存：存完口径留在编辑态，接着存费率（Go RateSet 是小数口径）
    fireEvent.click(within(edit).getByTestId('cost-edit-proj-rates-save'))
    await waitFor(() => expect(mocks.WorkcostBillProjectRatesSave).toHaveBeenCalledTimes(1))
    const rates = mocks.WorkcostBillProjectRatesSave.mock.calls[0]
    expect(rates[0]).toBe(2)
    expect(rates[1]).toEqual({ managementRate: 0.1, regulatoryRate: 0.02, profitRate: 0.07, taxRate: 0.09 })
  })

  it('编辑失败如实报错在表单里，不退出编辑态', async () => {
    mocks.WorkcostBillItemSave.mockRejectedValueOnce(new Error('清单项需要归属项目'))
    await renderReady()
    fireEvent.click(screen.getAllByRole('button', { name: '查看「外委检测」的综合单价分析' })[0]!)
    const bill = await screen.findByTestId('cost-detail-bill')
    fireEvent.click(within(bill).getByTestId('cost-bill-edit-open'))
    const edit = await screen.findByTestId('cost-bill-edit')
    fireEvent.click(within(edit).getByTestId('cost-edit-save'))
    const err = await within(edit).findByTestId('cost-edit-error')
    expectText(err, '清单项需要归属项目')
    // 编辑态还在，可改完再存
    expect(screen.getByTestId('cost-bill-edit')).toBeTruthy()
  })
})

describe('造价数据库 · 索引与编辑优化（2026-11-28）', () => {
  // 进到定额 WP01 的详情（等含量行现算出来 = 完整定额已读到）
  const openQuotaDetail = async (): Promise<HTMLElement> => {
    fireEvent.click(screen.getAllByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' })[0]!)
    const bill = await screen.findByTestId('cost-detail-bill')
    fireEvent.click(within(bill).getByRole('button', { name: '下钻到定额 WP01' }))
    const quota = await screen.findByTestId('cost-detail-quota')
    await within(quota).findByTestId('cost-quota-analysis')
    return quota
  }
  const optionTexts = (select: HTMLElement): (string | null)[] => Array.from(select.querySelectorAll('option')).map((o) => o.textContent)
  /** 字典 mock：可增删，读得到最新（默认 give 的那几条）。 */
  const useDict = (initial: { id: number; kind: string; name: string; parent?: string }[]) => {
    const dict = [...initial]
    mocks.WorkcostDictList.mockImplementation(async (kind: string) => dict.filter((d) => kind === '' || d.kind === kind).map((d) => ({ ...d })))
    mocks.WorkcostDictSet.mockImplementation(async (item: { kind: string; name: string; remove?: boolean }, remove?: boolean) => {
      if (remove) {
        const at = dict.findIndex((d) => d.kind === item.kind && d.name === item.name)
        if (at >= 0) dict.splice(at, 1)
        return
      }
      if (!dict.some((d) => d.kind === item.kind && d.name === item.name)) dict.push({ ...item, id: dict.length + 10 })
    })
    return dict
  }

  it('规范值读字典：规划了但还没行在用的专业也列得出；没有合适的就地新增并选中', async () => {
    useDict([
      { id: 1, kind: 'specialty', name: '土壤修复' }, { id: 2, kind: 'specialty', name: '市政工程' }, { id: 3, kind: 'region', name: '四川省' },
    ])
    await renderReady()
    const quota = await openQuotaDetail()
    fireEvent.click(within(quota).getByTestId('cost-quota-edit-open'))
    const edit = await screen.findByTestId('cost-quota-edit')
    const select = within(edit).getByTestId('cost-edit-quota-specialty') as HTMLSelectElement
    // 库里的定额全是「土壤修复」，字典里另有「市政工程」——此前下拉只列得出前者（字典值在前，按字典顺序）
    expect(optionTexts(select)).toEqual(['未标注', '土壤修复', '市政工程', '＋ 新增…'])
    expect(select.value).toBe('土壤修复')
    fireEvent.change(select, { target: { value: '__add__' } })
    fireEvent.change(await within(edit).findByLabelText('新增定额专业'), { target: { value: '水利水电工程' } })
    fireEvent.click(within(edit).getByTestId('cost-edit-quota-specialty-confirm'))
    await waitFor(() => expect(mocks.WorkcostDictSet).toHaveBeenCalledWith({ id: 0, kind: 'specialty', name: '水利水电工程', parent: '' }, false))
    await waitFor(() => expect((within(edit).getByTestId('cost-edit-quota-specialty') as HTMLSelectElement).value).toBe('水利水电工程'))
  })

  it('期数可以手填新月份：工料机的价格期数是输入框（已有期给候选），保存带上去', async () => {
    await renderReady()
    fireEvent.click(screen.getAllByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' })[0]!)
    const bill = await screen.findByTestId('cost-detail-bill')
    fireEvent.click(within(bill).getByRole('button', { name: '下钻到定额 WP01' }))
    const quota = await screen.findByTestId('cost-detail-quota')
    fireEvent.click(await within(quota).findByRole('button', { name: '下钻到工料机 C20商品混凝土' }))
    const res = await screen.findByTestId('cost-detail-resource')
    fireEvent.click(within(res).getByTestId('cost-resource-edit-open'))
    const edit = await screen.findByTestId('cost-resource-edit')
    const period = within(edit).getByTestId('cost-edit-res-period') as HTMLInputElement
    expect(period.tagName).toBe('INPUT')
    expect(period.getAttribute('list')).toBeTruthy() // 候选
    fireEvent.change(period, { target: { value: '2027-01' } })
    fireEvent.click(within(edit).getByTestId('cost-edit-save'))
    await waitFor(() => expect(mocks.WorkcostResourceSave).toHaveBeenCalledTimes(1))
    expect((mocks.WorkcostResourceSave.mock.calls[0][0] as WorkcostResource).priceDate).toBe('2027-01')
  })

  it('未保存守卫与快捷键：改了没存就切走要先问；Ctrl+S 保存、Esc 取消', async () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    await renderReady()
    fireEvent.click(screen.getAllByRole('button', { name: '查看「外委检测」的综合单价分析' })[0]!)
    const bill = await screen.findByTestId('cost-detail-bill')
    fireEvent.click(within(bill).getByTestId('cost-bill-edit-open'))
    let edit = await screen.findByTestId('cost-bill-edit')
    fireEvent.change(within(edit).getByTestId('cost-edit-bill-title'), { target: { value: '外委检测（改）' } })
    // 没保存就点另一行：先问；答「否」就留在编辑态、选中不变
    fireEvent.click(screen.getAllByRole('button', { name: /「施工便道（铺筑压实）」的综合单价分析/ })[1]!)
    expect(confirm).toHaveBeenCalledTimes(1)
    expect(screen.getByTestId('cost-bill-edit')).toBeTruthy()
    expect((within(edit).getByTestId('cost-edit-bill-title') as HTMLInputElement).value).toBe('外委检测（改）')
    // 有未保存改动时按 Esc 也要问
    fireEvent.keyDown(window, { key: 'Escape', code: 'Escape' })
    expect(confirm).toHaveBeenCalledTimes(2)
    expect(screen.getByTestId('cost-bill-edit')).toBeTruthy()
    // Ctrl+S 保存（带着改动）
    fireEvent.keyDown(window, { key: 's', code: 'KeyS', ctrlKey: true })
    await waitFor(() => expect(mocks.WorkcostBillItemSave).toHaveBeenCalledTimes(1))
    expect((mocks.WorkcostBillItemSave.mock.calls[0][0] as WorkcostBillItem).title).toBe('外委检测（改）')
    await waitFor(() => expect(screen.queryByTestId('cost-bill-edit')).toBeNull())
    // 没有改动时按 Esc 直接退出，不问
    fireEvent.click(within(bill).getByTestId('cost-bill-edit-open'))
    edit = await screen.findByTestId('cost-bill-edit')
    expect(edit).toBeTruthy()
    fireEvent.keyDown(window, { key: 'Escape', code: 'Escape' })
    await waitFor(() => expect(screen.queryByTestId('cost-bill-edit')).toBeNull())
    expect(confirm).toHaveBeenCalledTimes(2)
    confirm.mockRestore()
  })

  it('新建定额：板块里直接建（新编码、含量行从零），保存后重读并提示', async () => {
    mocks.WorkcostQuotaSave.mockImplementation(async (q: WorkcostQuota) => ({ ...q, id: 99 }))
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    fireEvent.click(screen.getByTestId('cost-create-btn'))
    const pane = await screen.findByTestId('cost-create')
    expectText(pane, '新建一条定额')
    const edit = within(pane).getByTestId('cost-quota-edit')
    // 没给编码不许存
    fireEvent.change(within(edit).getByTestId('cost-edit-quota-title'), { target: { value: '新定额' } })
    fireEvent.click(within(edit).getByTestId('cost-edit-save'))
    expectText(await within(edit).findByTestId('cost-edit-error'), '新定额需要编码')
    fireEvent.change(within(edit).getByTestId('cost-edit-quota-code'), { target: { value: 'NEW-1' } })
    fireEvent.click(within(edit).getByTestId('cost-edit-save'))
    await waitFor(() => expect(mocks.WorkcostQuotaSave).toHaveBeenCalledTimes(1))
    const saved = mocks.WorkcostQuotaSave.mock.calls[0][0] as WorkcostQuota
    expect(saved.id).toBe(0)
    expect(saved.code).toBe('NEW-1')
    expect(saved.title).toBe('新定额')
    expect(saved.source).toBe('造价库界面新建')
    expect(saved.items).toEqual([])
    await waitFor(() => expect(screen.queryByTestId('cost-create')).toBeNull())
    expectText(screen.getByTestId('cost-notice'), '已新建定额「新定额」')
  })

  it('复制定额另存：以原定额为底（含量行照带、含量行挂到新编码），编码默认顺延', async () => {
    await renderReady()
    const quota = await openQuotaDetail()
    fireEvent.click(within(quota).getByTestId('cost-quota-duplicate'))
    const pane = await screen.findByTestId('cost-create')
    expectText(pane, '以「施工便道」为底新建一条')
    const edit = within(pane).getByTestId('cost-quota-edit')
    expect((within(edit).getByTestId('cost-edit-quota-code') as HTMLInputElement).value).toBe('WP01-2')
    fireEvent.click(within(edit).getByTestId('cost-edit-save'))
    await waitFor(() => expect(mocks.WorkcostQuotaSave).toHaveBeenCalledTimes(1))
    const saved = mocks.WorkcostQuotaSave.mock.calls[0][0] as WorkcostQuota
    expect(saved.id).toBe(0)
    expect(saved.code).toBe('WP01-2')
    expect(saved.source).toBe('造价库界面复制自 WP01')
    expect(saved.items?.map((it) => it.resourceCode)).toEqual(['C20', 'L01'])
    expect(saved.items?.every((it) => it.quotaCode === 'WP01-2')).toBe(true)
  })

  it('勾选批量修改：选维度给值，逐行保存并回报；一行失败不影响其余', async () => {
    mocks.WorkcostQuotaSave.mockImplementation(async (q: WorkcostQuota) => {
      if (q.code === 'ENT-01') throw new Error('身份冲突')
      return q
    })
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    fireEvent.click(screen.getByRole('checkbox', { name: '选择定额 施工便道' }))
    fireEvent.click(screen.getByRole('checkbox', { name: '选择定额 企业定额（未引用）' }))
    fireEvent.click(screen.getByTestId('cost-batch-open'))
    const dialog = await screen.findByTestId('cost-batch-dialog')
    fireEvent.change(within(dialog).getByTestId('cost-batch-field'), { target: { value: 'chapter' } })
    fireEvent.change(within(dialog).getByTestId('cost-batch-value'), { target: { value: 'A基础' } })
    fireEvent.click(screen.getByRole('button', { name: '把这 2 条改掉' }))
    await waitFor(() => expect(mocks.WorkcostQuotaSave).toHaveBeenCalledTimes(2))
    // 定额先取完整（含量行）再整组保存：含量行原样带回
    expect(mocks.WorkcostQuotaGet).toHaveBeenCalledWith('WP01')
    const saved = mocks.WorkcostQuotaSave.mock.calls[0][0] as WorkcostQuota
    expect(saved.code).toBe('WP01')
    expect(saved.chapter).toBe('A基础')
    expect(saved.items).toHaveLength(2)
    expectText(await screen.findByTestId('cost-batch-failures'), '身份冲突')
    expectText(screen.getByTestId('cost-notice'), '成功 1 条，失败 1 条')
  })

  it('删除定额：仍被清单引用的，卡上多一条「解除挂接后删除」，先摘挂接再删', async () => {
    mocks.WorkcostQuotaDetachEverywhere.mockResolvedValue(2)
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    fireEvent.click(screen.getByRole('button', { name: '删除定额「施工便道」' }))
    const dialog = await screen.findByRole('dialog')
    expectText(dialog, '其中 1 条仍被清单项引用（共 2 条挂接）')
    fireEvent.click(within(dialog).getByTestId('cost-delete-force'))
    await waitFor(() => expect(mocks.WorkcostQuotaDelete).toHaveBeenCalledWith('WP01'))
    expect(mocks.WorkcostQuotaDetachEverywhere).toHaveBeenCalledWith('WP01')
    expect(mocks.WorkcostQuotaDetachEverywhere.mock.invocationCallOrder[0]!).toBeLessThan(mocks.WorkcostQuotaDelete.mock.invocationCallOrder[0]!)
  })

  it('定额地区回填：先预览（只被一个省引用的可回填），确认后才写', async () => {
    mocks.WorkcostQuotaRegionBackfill.mockImplementation(async (dryRun: boolean) => [
      { code: 'WP01', title: '施工便道', region: '四川省', links: 2, applied: !dryRun },
      { code: 'ENT-01', title: '企业定额', links: 0, reason: '没有被清单引用，没有上游项目可依据', applied: false },
    ])
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    fireEvent.click(screen.getByTestId('cost-region-open'))
    const dialog = await screen.findByTestId('cost-region-dialog')
    expectText(dialog, '可以回填')
    expectText(dialog, '四川省 1 条')
    expectText(dialog, '没有被清单引用，没有上游项目可依据（1 条）')
    expect(mocks.WorkcostQuotaRegionBackfill).toHaveBeenCalledTimes(1) // 只预览，没写
    expect(mocks.WorkcostQuotaRegionBackfill).toHaveBeenLastCalledWith(true)
    fireEvent.click(screen.getByTestId('cost-region-apply'))
    await waitFor(() => expect(mocks.WorkcostQuotaRegionBackfill).toHaveBeenLastCalledWith(false))
    expectText(screen.getByTestId('cost-notice'), '已回填 1 条定额的地区')
  })

  it('检索别名：搜索框旁管理（砼 ↔ 混凝土），增删走字典', async () => {
    useDict([{ id: 1, kind: 'specialty', name: '土壤修复' }])
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-alias-open'))
    const dialog = await screen.findByTestId('cost-alias-dialog')
    fireEvent.change(within(dialog).getByLabelText('别名'), { target: { value: '砼' } })
    fireEvent.change(within(dialog).getByLabelText('标准说法'), { target: { value: '混凝土' } })
    fireEvent.click(within(dialog).getByTestId('cost-alias-add'))
    await waitFor(() => expect(mocks.WorkcostDictSet).toHaveBeenCalledWith({ id: 0, kind: 'alias', name: '砼', parent: '混凝土' }, false))
    await waitFor(() => expectText(screen.getByTestId('cost-alias-list'), '砼'))
    fireEvent.click(screen.getByRole('button', { name: '删除别名 砼' }))
    await waitFor(() => expect(mocks.WorkcostDictSet).toHaveBeenLastCalledWith(expect.objectContaining({ kind: 'alias', name: '砼' }), true))
    await waitFor(() => expect(screen.queryByTestId('cost-alias-list')).toBeNull())
  })

  it('挂接定额可以搜出来再选：输入即出候选，点候选即填入编码', async () => {
    await renderReady()
    fireEvent.click(screen.getAllByRole('button', { name: '查看「施工便道（铺筑压实）」的综合单价分析' })[0]!)
    const bill = await screen.findByTestId('cost-detail-bill')
    fireEvent.click(within(bill).getByTestId('cost-bill-edit-open'))
    await screen.findByTestId('cost-bill-edit')
    fireEvent.change(screen.getByLabelText('挂接定额编码'), { target: { value: '企业' } })
    const candidate = await screen.findByRole('option', { name: /企业定额/ })
    fireEvent.click(within(candidate).getByRole('button'))
    expect((screen.getByLabelText('挂接定额编码') as HTMLInputElement).value).toBe('ENT-01')
    fireEvent.click(screen.getByTestId('cost-edit-link-attach'))
    await waitFor(() => expect(mocks.WorkcostBillQuotaAttach).toHaveBeenCalledWith(11, 'ENT-01', 0))
  })
})
