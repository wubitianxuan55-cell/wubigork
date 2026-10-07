// CostLibraryPage.test.tsx — 「造价数据库」全新设计（v4.459 工料法 IA）测试。
// 覆盖：①默认落点=费用汇总工作台 ②导航 6 模块齐全有序（测算项目/复盘笔记
// 退役、成本条目降为价格数据的「资料条目」段）③模块切换渲染对应视图
// ④价格数据四段 ⑤概览工料法仪表盘（资源/定额/资料库规模+四类构成）
// ⑥定额为 0 时的下一步引导 ⑦概览双视图 ⑧空库三步引导 ⑨读取失败态
// ⑩回到概览。
// 子视图以桩模块替换（各自有独立测试），桥接仅 mock 页面级消费的 app.* 方法。

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  WorkcostResourceList: vi.fn(),
  WorkcostQuotaList: vi.fn(),
  CostSearch: vi.fn(),
  PriceSources: vi.fn(),
}))
vi.mock('../gaea/lib/bridge', () => ({ app: mocks }))

// 子视图桩：IA 测试只关心「切换到某模块渲染对应视图」，各视图内部有自己的测试。
vi.mock('../gaea/components/memoryhub/WorkcostBillView', () => ({
  WorkcostBillView: () => <div>清单视图桩</div>,
}))
vi.mock('../gaea/components/memoryhub/WorkcostComposeView', () => ({
  WorkcostComposeView: () => <div>单价分析视图桩</div>,
}))
vi.mock('../gaea/components/memoryhub/WorkcostResourceView', () => ({
  WorkcostResourceView: () => <div>资源库视图桩</div>,
}))
vi.mock('../gaea/components/CostLibraryView', () => ({
  CostLibraryView: () => <div>资料条目视图桩</div>,
}))
vi.mock('../gaea/components/memoryhub/CostIndicatorsView', () => ({
  CostIndicatorsView: () => <div>造价参考视图桩</div>,
}))
vi.mock('../gaea/components/memoryhub/CostGraphView', () => ({
  CostGraphView: () => <div>关联图谱视图桩</div>,
}))
vi.mock('../gaea/components/memoryhub/PriceSourcesPanel', () => ({
  PriceSourcesPanel: () => <div>价格源面板桩</div>,
}))
vi.mock('../gaea/components/memoryhub/PriceSourcesRepository', () => ({
  PriceSourcesRepository: () => <div>价格仓库面板桩</div>,
}))
vi.mock('../gaea/components/memoryhub/CostInquiryPanel', () => ({
  CostInquiryPanel: () => <div>询价库面板桩</div>,
}))

import CostLibraryPage from './CostLibraryPage'

const LOAD = { timeout: 5000 }

// 工料法三层 mock 数据：资源 5 条（人工 1/材料 2/机械 1/外委 1）+ 定额 2 条。
const RESOURCES = [
  { id: 1, code: 'L001', kind: '人工', title: '普通工', unit: '工日', currentPrice: 300, basePrice: 300 },
  { id: 2, code: 'M001', kind: '材料', title: 'C20商品混凝土', unit: 'm³', currentPrice: 345, basePrice: 345 },
  { id: 3, code: 'M002', kind: '材料', title: '级配碎石', unit: 'm³', currentPrice: 78, basePrice: 78 },
  { id: 4, code: 'E001', kind: '机械', title: '挖掘机1.0m³台班', unit: '台班', currentPrice: 1773.3, basePrice: 1773.3 },
  { id: 5, code: 'O001', kind: '外委', title: '水泥窑协同处置', unit: 't', currentPrice: 260, basePrice: 260 },
]
const QUOTAS = [
  { id: 1, code: 'WP01', title: '施工便道', specialty: '土壤修复', chapter: 'A临建', unit: 'm' },
  { id: 2, code: 'WP02', title: '场地平整', specialty: '土壤修复', chapter: 'A临建', unit: 'm²' },
]
const ENTRIES = [
  { name: 'steel-h', title: 'H 型钢', category: '钢材', unit: '吨', price: 5200, status: '现行' },
  { name: 'draft-p', title: '钻孔桩（草稿）', category: '桩基', unit: 'm', price: 0, status: '草稿' },
]

function mockFull() {
  mocks.WorkcostResourceList.mockResolvedValue(RESOURCES)
  mocks.WorkcostQuotaList.mockResolvedValue(QUOTAS)
  mocks.CostSearch.mockResolvedValue(ENTRIES)
  mocks.PriceSources.mockResolvedValue([{ id: 's1', name: '重庆造价信息网' }])
}

async function renderPage() {
  const view = render(<CostLibraryPage />)
  // 默认落点=费用汇总工作台（不再等待概览数据——概览按需拉取）。
  await waitFor(() => expect(screen.getByText('清单视图桩')).toBeTruthy(), LOAD)
  return view
}

async function renderOverview() {
  const view = await renderPage()
  fireEvent.click(screen.getByTitle('工料法规模 · 资料库速览 · 关联图谱'))
  await waitFor(() => expect(screen.getByText('快捷入口')).toBeTruthy(), LOAD)
  return view
}

beforeEach(() => {
  vi.clearAllMocks()
  mockFull()
})

describe('CostLibraryPage 造价数据库全新设计（v4.459 工料法 IA）', () => {
  it('默认落点=分部分项清单（录入数据面），不再是概览', async () => {
    await renderPage()
    expect(screen.getByText('清单视图桩')).toBeTruthy()
    expect(screen.queryByText('数据概览')).toBeNull()
  })

  it('导航 6 模块齐全有序；测算项目/复盘笔记退役，成本条目降为价格数据段', async () => {
    await renderPage()
    const nav = screen.getByRole('navigation', { name: '造价数据库模块' })
    const labels = ['清单库', '企业定额库', '工料机库', '价格数据', '造价参考', '概览']
    let cursor = -1
    for (const label of labels) {
      const btns = Array.from(nav.querySelectorAll('button'))
      const idx = btns.findIndex((b, i) => i > cursor && b.textContent?.includes(label))
      expect(idx, `导航缺少 ${label}`).toBeGreaterThan(cursor)
      cursor = idx
    }
    // 带 hint title 的按钮 = 6 个顶级模块 + 常驻「回到概览」（默认落在费用汇总）。
    expect(nav.querySelectorAll('button[title]').length).toBe(7)
    // 退役模块不再是顶级入口（数据与 Go 绑定保留，仅 UI 退役）。
    expect(nav.textContent).not.toContain('测算项目')
    expect(nav.textContent).not.toContain('复盘笔记')
    expect(nav.textContent).not.toContain('成本条目')
    expect(nav.textContent).not.toContain('价格仓库')
    expect(nav.textContent).not.toContain('询价')
  })

  it('模块切换：费用汇总/单价分析/资源库/造价参考各渲染对应视图', async () => {
    await renderPage()
    for (const [hint, marker] of [
      ['所有项目累计的分部分项清单 · 明细/库项归并双视图', '清单视图桩'],
      ['企业定额积累 · 单价分析（只含人材机）· 导入/导出五表', '单价分析视图桩'],
      ['工料机主数据 · 现行价/基准价 · 与信息价联动 · 调价历史', '资源库视图桩'],
      ['项目案例（导入项目文件）+ 资料条目分位数对标——与分部分项清单是两套数据', '造价参考视图桩'],
    ] as const) {
      fireEvent.click(screen.getByTitle(hint))
      expect(await screen.findByText(marker, {}, LOAD)).toBeTruthy()
    }
  })

  it('价格数据四段：默认价格源，资料条目（旧成本条目）降为第四段', async () => {
    await renderPage()
    fireEvent.click(screen.getByTitle('价格源 · 价格仓库 · 询价库 · 资料条目'))
    const seg = screen.getByRole('tablist', { name: '价格数据子视图' })
    expect(seg.textContent).toContain('价格源')
    expect(seg.textContent).toContain('价格仓库')
    expect(seg.textContent).toContain('询价库')
    expect(seg.textContent).toContain('资料条目')
    expect(await screen.findByText('价格源面板桩', {}, LOAD)).toBeTruthy()

    fireEvent.click(within(seg).getByRole('button', { name: '资料条目' }))
    expect(screen.getByText('资料条目视图桩')).toBeTruthy()
    fireEvent.click(within(seg).getByRole('button', { name: '询价库' }))
    expect(screen.getByText('询价库面板桩')).toBeTruthy()
    fireEvent.click(within(seg).getByRole('button', { name: '价格仓库' }))
    expect(screen.getByText('价格仓库面板桩')).toBeTruthy()
  })

  it('概览=工料法仪表盘：三层规模、四类构成与资料库速览', async () => {
    await renderOverview()
    expect(screen.getByText('5')).toBeTruthy() // 资源总数（hero 大数字）
    expect(screen.getAllByText(/工料机资源/).length).toBeGreaterThan(0)
    expect(screen.getByText('条工料机资源 · 2 条消耗定额')).toBeTruthy()
    expect(screen.getByText('资料条目 2')).toBeTruthy()
    expect(screen.getByText('价格源 1')).toBeTruthy()
    // 四类构成在 hero 右侧速览。
    expect(screen.getByText('人工 1')).toBeTruthy()
    expect(screen.getByText('材料 2')).toBeTruthy()
    expect(screen.getByText('机械 1')).toBeTruthy()
    expect(screen.getByText('外委 1')).toBeTruthy()
    // 定额就绪态指路清单测算。
    expect(screen.getByText('清单数据就绪')).toBeTruthy()
  })

  it('概览引导：有资源但定额为 0 时，指路「去导入项目表」', async () => {
    mocks.WorkcostQuotaList.mockResolvedValue([])
    const view = await renderPage()
    fireEvent.click(screen.getByTitle('工料法规模 · 资料库速览 · 关联图谱'))
    await waitFor(() => expect(screen.getByText('还差消耗定额')).toBeTruthy(), LOAD)
    fireEvent.click(screen.getByText('去导入项目表'))
    expect(await screen.findByText('单价分析视图桩', {}, LOAD)).toBeTruthy()
    expect(view).toBeTruthy()
  })

  it('概览双视图：关联图谱切换可见，数据概览切回', async () => {
    await renderOverview()
    fireEvent.click(screen.getByRole('button', { name: '关联图谱' }))
    expect(screen.getByText('关联图谱视图桩')).toBeTruthy()
    expect(screen.queryByText('快捷入口')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: '数据概览' }))
    expect(await screen.findByText('快捷入口', {}, LOAD)).toBeTruthy()
    expect(screen.queryByText('关联图谱视图桩')).toBeNull()
  })

  it('空库三步引导：导入项目表 → 维护资源库 → 订阅价格源', async () => {
    mocks.WorkcostResourceList.mockResolvedValue([])
    mocks.WorkcostQuotaList.mockResolvedValue([])
    render(<CostLibraryPage />)
    fireEvent.click(screen.getByTitle('工料法规模 · 资料库速览 · 关联图谱'))
    expect(await screen.findByText('工料法成本数据库还是空的', {}, LOAD)).toBeTruthy()
    expect(screen.getByText('去单价分析')).toBeTruthy()
    expect(screen.getByText('去资源库')).toBeTruthy()
    expect(screen.getByText('去配置')).toBeTruthy()
  })

  it('全源读取失败显形为失败态（不装成空库引导）', async () => {
    mocks.WorkcostResourceList.mockRejectedValue(new Error('database is locked'))
    mocks.WorkcostQuotaList.mockRejectedValue(new Error('database is locked'))
    mocks.CostSearch.mockRejectedValue(new Error('database is locked'))
    mocks.PriceSources.mockRejectedValue(new Error('database is locked'))
    render(<CostLibraryPage />)
    fireEvent.click(screen.getByTitle('工料法规模 · 资料库速览 · 关联图谱'))
    const alert = await screen.findByTestId('cost-stats-failed', {}, LOAD)
    expect(alert.textContent).toContain('读取失败')
    expect(screen.getByText('重试')).toBeTruthy()
  })

  it('非概览模块显示回到概览，点击回数据概览', async () => {
    await renderPage()
    fireEvent.click(screen.getByTitle(/项目案例（导入项目文件）/))
    fireEvent.click(screen.getByTitle('回到概览'))
    expect(await screen.findByText('快捷入口', {}, LOAD)).toBeTruthy()
  })
})
