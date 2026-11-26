// CostLibraryPage.test.tsx — 一级板块「造价数据库」页面测试（2026-11-25 推倒重建）。
// 覆盖：①三级行计数与默认清单列表 ②点行看综合单价分析 → 定额码下钻到定额 →
// 含量行下钻到工料机 → 引用清单回链 ③左栏项目筛选与项目封面卡 ④搜索与无匹配
// 清除筛选 ⑤归并视图 ⑥删除：取消不删、确认删并重读、后端拒绝（仍被引用）时
// 原因留在卡上 ⑦导出：选目录后按当前筛选调绑定并显示路径 ⑧体检抽屉列出发现并
// 可跳转 ⑨板块常驻保活下切回即静默重读 ⑩三级全读失败不伪装成空库、重试恢复
// ⑪空库三步引导与去办公板块。桥接只 mock 页面消费的绑定。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import type { CostIndexStatusView, WorkcostBillProject, WorkcostCompose, WorkcostQuota, WorkcostResource } from '../gaea/lib/types'

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
  categoryPath: '材料/混凝土', source: '成本库', supplier: '', region: '乐山', priceDate: '2026年第7期', priceType: '到场价',
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
    { id: 2, resourceId: 101, price: 360, period: '2026年第11期', region: '乐山', priceType: '到场价', source: '造价模式调价', fetchedAt: '2026-11-20T08:00:00Z', note: '' },
    { id: 1, resourceId: 101, price: 345, period: '2026年第7期', region: '乐山', priceType: '到场价', source: '项目导入', fetchedAt: '2026-10-07T08:00:00Z', note: '' },
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
})

describe('造价数据库 · 三级浏览面', () => {
  it('读入三级数据：状态行给出规模，默认落在清单列表，详情空态', async () => {
    await renderReady()
    const status = screen.getByTestId('cost-status')
    expectText(status, '项目')
    expectText(status, '清单')
    expectText(status, '定额')
    expectText(status, '工料机')
    expectAriaSelected(screen.getByTestId('cost-tier-bill'))
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
    expectText(res, '核算取用价')
    expectText(res, 'G RC010001')
    expectText(res, '¥360')
    expectText(res, '¥345') // 基准价
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

  it('左栏项目筛选：只看该项目的清单，详情显示项目封面卡（只读费率）', async () => {
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
    expect(items).toEqual(['测算模版（按库预填）', '清单（当前筛选）', '定额（当前筛选）', '工料机（当前筛选）', '资料条目（全部）'])
    fireEvent.click(Array.from(menu.querySelectorAll('.ant-dropdown-menu-item')).at(-1)!)
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(1))
    const [kind, filter, out] = mocks.CostLibraryExport.mock.calls[0]!
    expect([kind, filter]).toEqual(['entry', EMPTY_FILTER])
    expect(out.startsWith('D:\\新工作\\导出/造价库-资料条目-')).toBe(true)
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

describe('造价数据库 · 维度（专业 / 章节 / 分类 / 地区 / 期）', () => {
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

  it('定额：先选专业才出章节这一级；章节在专业内筛选，换专业时章节筛选跟着清', async () => {
    const sz: WorkcostQuota = { ...quotaWP01, id: 3, code: 'SZ01', origCode: '', title: '沥青路面', specialty: '市政', chapter: '道路工程' }
    const unlabeled: WorkcostQuota = { ...quotaWP01, id: 4, code: 'Q9', origCode: '', title: '没标专业的定额', specialty: '', chapter: '' }
    mocks.WorkcostQuotaList.mockResolvedValue([
      { ...quotaWP01, items: undefined, usageCount: 2 }, { ...quotaFree, items: undefined, chapter: '', usageCount: 0 },
      { ...sz, items: undefined }, { ...unlabeled, items: undefined },
    ])
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-quota'))
    await screen.findByTestId('cost-quota-rows')
    expect(screen.queryByTestId('cost-rail-quota-chapters')).toBeNull() // 没选专业：不出章节
    expect(rowsOf('cost-quota-rows')).toBe(5) // 表头 + 4
    expectText(screen.getByTestId('cost-quota-rows'), '章节') // 列表有章节列
    fireEvent.click(railBtn('cost-rail-quota-specialties', /^土壤修复/))
    const chapters = await screen.findByTestId('cost-rail-quota-chapters')
    expectText(chapters, 'A临建')
    expectText(chapters, '未归章节')
    expect(rowsOf('cost-quota-rows')).toBe(3) // 表头 + 土壤修复的 2 条
    fireEvent.click(railBtn('cost-rail-quota-chapters', /^A临建/))
    expect(rowsOf('cost-quota-rows')).toBe(2)
    expectText(screen.getByTestId('cost-crumb'), 'A临建')
    fireEvent.click(railBtn('cost-rail-quota-specialties', /^市政/))
    expect(textOf(screen.getByTestId('cost-crumb'))).not.toContain('A临建')
    expectText(screen.getByTestId('cost-rail-quota-chapters'), '道路工程')
    expect(rowsOf('cost-quota-rows')).toBe(2) // 市政的 1 条（章节筛选已清）
  })

  it('工料机：类别 → 分类（选上级连带下级）；工具条按地区、期数筛选；列表有地区与期两列', async () => {
    const cq: WorkcostResource = { ...c20, id: 104, code: 'C30', title: 'C30商品混凝土', spec: '泵送', categoryPath: '材料/混凝土/泵送', region: '重庆中心城区', priceDate: '2025-12' }
    const bare: WorkcostResource = { ...c20, id: 105, code: 'SAND', title: '中砂', spec: '', categoryPath: '', region: '', priceDate: '' }
    mocks.WorkcostResourceList.mockResolvedValue([
      { ...c20, region: '成都市区', priceDate: '2026-06' }, { ...l01, region: '', priceDate: '', categoryPath: '' }, { ...cq }, { ...bare },
    ])
    await renderReady()
    fireEvent.click(screen.getByTestId('cost-tier-resource'))
    await screen.findByTestId('cost-resource-rows')
    expect(rowsOf('cost-resource-rows')).toBe(5) // 表头 + 4
    expectText(screen.getByTestId('cost-resource-rows'), '成都市区')
    expectText(screen.getByTestId('cost-resource-rows'), '2026年6月') // 库里存 YYYY-MM，界面显示年月
    const options = (testid: string) => Array.from(screen.getByTestId(testid).querySelectorAll('option')).map((o) => o.textContent)
    expect(options('cost-select-region')).toEqual(['全部地区', '成都市区（1）', '重庆中心城区（1）', '未标注（2）'])
    expect(options('cost-select-period')).toEqual(['全部期', '2026年6月（1）', '2025年12月（1）', '无期（2）'])
    fireEvent.change(screen.getByTestId('cost-select-region'), { target: { value: '重庆中心城区' } })
    expect(rowsOf('cost-resource-rows')).toBe(2)
    expectText(screen.getByTestId('cost-crumb'), '重庆中心城区')
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
      categoryPath: '', region: '重庆中心城区', priceDate: '2026-06',
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

  it('待整理：状态行给出未归类的条数与分项，点一下复制整理提示词', async () => {
    await renderReady()
    const pending = screen.getByTestId('cost-pending')
    expectText(pending, '待整理 3') // 三条清单行都还没标专业
    expect(pending.getAttribute('title')).toContain('清单无专业 3')
    fireEvent.click(pending)
    await waitFor(() => expect(copyToClipboard).toHaveBeenCalledWith(PENDING_PROMPT))
    expectText(pending, '已复制提示词')
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
    fireEvent.change(screen.getByTestId('cost-select-region'), { target: { value: '乐山' } })
    await pickExport(3)
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(2))
    expect(mocks.CostLibraryExport.mock.calls[1]!.slice(0, 2)).toEqual(['resource', { ...EMPTY_FILTER, region: '乐山' }])
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
    fireEvent.change(screen.getByTestId('cost-select-region'), { target: { value: '乐山' } })
    fireEvent.click(within(screen.getByTestId('cost-rail-kinds')).getAllByRole('button', { name: /^材料/ })[0]!)
    await pickExport(0)
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(2))
    expect(mocks.CostLibraryExport.mock.calls[1]!.slice(0, 2)).toEqual(['template', { ...EMPTY_FILTER, kind: '材料', region: '乐山' }])
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
})
