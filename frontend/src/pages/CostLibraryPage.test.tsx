// CostLibraryPage.test.tsx — 一级板块「造价数据库」页面测试（2026-11-25 推倒重建）。
// 覆盖：①三级行计数与默认清单列表 ②点行看综合单价分析 → 定额码下钻到定额 →
// 含量行下钻到工料机 → 引用清单回链 ③左栏项目筛选与项目封面卡 ④搜索与无匹配
// 清除筛选 ⑤归并视图 ⑥删除：取消不删、确认删并重读、后端拒绝（仍被引用）时
// 原因留在卡上 ⑦导出：选目录后按当前筛选调绑定并显示路径 ⑧体检抽屉列出发现并
// 可跳转 ⑨板块常驻保活下切回即静默重读 ⑩三级全读失败不伪装成空库、重试恢复
// ⑪空库三步引导与去办公板块。桥接只 mock 页面消费的绑定。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import type { WorkcostBillProject, WorkcostCompose, WorkcostQuota, WorkcostResource } from '../gaea/lib/types'

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
}))
vi.mock('../gaea/lib/bridge', () => ({ app: mocks }))
vi.mock('../gaea/lib/clipboard', () => ({ copyToClipboard: vi.fn().mockResolvedValue('plain') }))

import { notifyBoardActive, resetBoardActive } from '../lib/boardActive'
import CostLibraryPage from './CostLibraryPage'

const p1: WorkcostBillProject = {
  id: 1, name: '旺平矿业修复', fileName: '旺平成本测算表.xlsx', source: '', location: '乐山', duration: '180 天',
  pricing: '综合单价只含人材机', managementRate: 0.1, regulatoryRate: 0.02, profitRate: 0.07, taxRate: 0.09,
  profitIncludesRegulatory: false, controlPrice: 0, itemCount: 2,
}
const p2: WorkcostBillProject = { ...p1, id: 2, name: '什邡地块治理', itemCount: 1, location: '德阳' }

const row11 = { id: 11, projectId: 1, projectName: p1.name, code: 'WP01', title: '施工便道（铺筑压实）', unit: 'm', division: 'A临建', quantity: 850, quantityExpr: '长850×宽1', quotaCode: 'WP01', feature: '泥结碎石', priceOverride: 0, listPrice: 420, sort: 1, quotaCodes: ['WP01'] }
const row12 = { id: 12, projectId: 1, projectName: p1.name, code: 'M001', title: '外委检测', unit: '项', division: 'D外运处置', quantity: 1, quantityExpr: '', quotaCode: '', feature: '', priceOverride: 5000, sort: 2, quotaCodes: [] }
const row13 = { id: 13, projectId: 2, projectName: p2.name, code: 'WP01', title: '施工便道（铺筑压实）', unit: 'm', division: 'A临建', quantity: 300, quantityExpr: '', quotaCode: 'WP01', feature: '泥结碎石', priceOverride: 0, listPrice: 430, sort: 1, quotaCodes: ['WP01'] }

const quotaWP01: WorkcostQuota = {
  id: 1, code: 'WP01', origCode: 'WP-1', title: '施工便道', specialty: '土壤修复', chapter: 'A临建', unit: 'm',
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
  id: 101, code: 'C20', kind: '材料', title: 'C20商品混凝土', spec: '泵送到场', unit: 'm³', basePrice: 345, currentPrice: 360,
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
  mocks.CostLibraryExport.mockImplementation(async (_k: string, _p: number, _q: string, _e: string, out: string) => out)
  mocks.PickDirectory.mockResolvedValue('D:\\新工作\\导出')
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
    expect((await within(bill).findAllByText('¥162')).length).toBeGreaterThan(0) // 0.45 × 360
    expect(mocks.WorkcostQuotaGet).toHaveBeenCalledWith('WP01')

    // 定额码下钻：切到定额级并选中该定额。
    fireEvent.click(within(bill).getByRole('button', { name: '下钻到定额 WP01' }))
    const quota = await screen.findByTestId('cost-detail-quota')
    expectAriaSelected(screen.getByTestId('cost-tier-quota'))
    expectText(quota, '施工便道')
    expectText(quota, '原码 WP-1')
    expectText(quota, '被 2 条清单项引用')
    expect(mocks.WorkcostQuotaCompose).toHaveBeenCalledWith('WP01', null)

    // 含量行下钻到工料机。
    fireEvent.click(within(quota).getByRole('button', { name: '下钻到工料机 C20商品混凝土' }))
    const res = await screen.findByTestId('cost-detail-resource')
    expectAriaSelected(screen.getByTestId('cost-tier-resource'))
    expectText(res, '核算取用价')
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
    expect(items).toEqual(['清单（当前筛选）', '定额（当前筛选）', '工料机（当前筛选）', '资料条目（全部）'])
    fireEvent.click(Array.from(menu.querySelectorAll('.ant-dropdown-menu-item')).at(-1)!)
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(1))
    const [kind, projectId, keyword, extra, out] = mocks.CostLibraryExport.mock.calls[0]!
    expect([kind, projectId, keyword, extra]).toEqual(['entry', 0, '', ''])
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
    fireEvent.click(Array.from(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item')).at(0)!)
    await waitFor(() => expect(mocks.CostLibraryExport).toHaveBeenCalledTimes(1))
    expect(mocks.CostLibraryExport.mock.calls[0]!.slice(0, 4)).toEqual(['bill', 0, '', 'A临建'])
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
    expectText(empty, '导入造价库')
    expect(screen.queryByRole('searchbox')).toBeNull()
    const nav = vi.fn()
    window.addEventListener('navigate', nav)
    fireEvent.click(within(empty).getByRole('button', { name: /去办公板块/ }))
    expect(nav).toHaveBeenCalledTimes(1)
    window.removeEventListener('navigate', nav)
  })
})
