// CostLibraryPage.test.tsx — 「造价数据库」化繁为简重构图（v4.50）测试。
// 覆盖 IA 重组零删减：①导航 6 模块齐全有序（旧平级 价格源/价格仓库/知识图谱
// 收编，不再作为顶级模块）②概览双视图（数据概览/关联图谱）③价格数据三段
// （价格源/价格仓库/询价库——询价库从成本条目隐藏 icon 视图升格一等子页）
// ④模块切换渲染对应视图 ⑤概览统计（库规模/健康/专业分部）⑥快捷入口收敛
// ⑦回到概览 ⑧空库引导态 ⑨新建条目/导入文件快捷动作链路。
// 子视图以桩模块替换（各自有独立测试），桥接仅 mock 页面级消费的 app.* 方法。

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  CostSearch: vi.fn(),
  CostCategories: vi.fn(),
  PriceSources: vi.fn(),
  PickFiles: vi.fn(),
}))
vi.mock('../gaea/lib/bridge', () => ({ app: mocks }))

// 子视图桩：IA 测试只关心「切换到某模块渲染对应视图」，各视图内部有自己的测试。
vi.mock('../gaea/components/CostLibraryView', () => ({
  CostLibraryView: () => <div>成本条目视图桩</div>,
}))
vi.mock('../gaea/components/memoryhub/CostProjectsView', () => ({
  CostProjectsView: () => <div>测算项目视图桩</div>,
}))
vi.mock('../gaea/components/memoryhub/CostIndicatorsView', () => ({
  CostIndicatorsView: () => <div>造价参考视图桩</div>,
}))
vi.mock('../gaea/components/memoryhub/CostNotesView', () => ({
  CostNotesView: () => <div>复盘笔记视图桩</div>,
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
vi.mock('../gaea/components/memoryhub/CostEntryModal', () => ({
  CostEntryModal: ({ open }: { open: boolean }) => (open ? <div>新建条目弹窗桩</div> : null),
}))
vi.mock('../gaea/components/memoryhub/CostImportModal', () => ({
  CostImportModal: () => <div>导入弹窗桩</div>,
}))

import { CostLibraryPage } from './CostLibraryPage'

const LOAD = { timeout: 5000 }

// 两条覆盖三种健康态：A 现行可引用；B 草稿且缺单价（同时进待补/草稿/待处理）。
const ENTRIES = [
  {
    name: 'steel-h', title: 'H 型钢', category: '钢材', categoryPath: '综合单价/市政/钢材',
    unit: '吨', price: 5200, spec: '', source: '', tags: [], status: '现行', updatedAt: '2026-09-01T00:00:00Z',
  },
  {
    name: 'draft-p', title: '钻孔桩（草稿）', category: '桩基', categoryPath: '综合单价/市政/桩基',
    unit: 'm', price: 0, spec: '', source: '', tags: [], status: '草稿', updatedAt: '2026-09-02T00:00:00Z',
  },
]

const CAT_TREE = [
  {
    id: 1, parentId: 0, name: '综合单价', sort: 0, count: 0,
    children: [
      { id: 11, parentId: 1, name: '市政', sort: 0, count: 0, children: [
        { id: 111, parentId: 11, name: '钢材', sort: 0, count: 1 },
        { id: 112, parentId: 11, name: '桩基', sort: 0, count: 1 },
      ] },
      { id: 12, parentId: 1, name: '建筑', sort: 0, count: 0, children: [] },
    ],
  },
]

function mockFull() {
  mocks.CostSearch.mockResolvedValue(ENTRIES)
  mocks.CostCategories.mockResolvedValue(CAT_TREE)
  mocks.PriceSources.mockResolvedValue([{ id: 's1', name: '重庆造价信息网' }])
}

async function renderPage() {
  const view = render(<CostLibraryPage />)
  await waitFor(() => expect(screen.getByText('数据健康')).toBeTruthy(), LOAD)
  return view
}

beforeEach(() => {
  vi.clearAllMocks()
  mockFull()
  mocks.PickFiles.mockResolvedValue([])
})

describe('CostLibraryPage 造价数据库化繁为简（8→6 模块）', () => {
  it('导航六模块齐全有序，旧平级 价格源/价格仓库/知识图谱/询价 不再是顶级模块', async () => {
    await renderPage()
    const nav = screen.getByRole('navigation', { name: '造价数据库模块' })
    const labels = ['概览', '成本条目', '测算项目', '价格数据', '造价参考', '复盘笔记']
    let cursor = -1
    for (const label of labels) {
      const btns = Array.from(nav.querySelectorAll('button'))
      const idx = btns.findIndex((b, i) => i > cursor && b.textContent?.includes(label))
      expect(idx, `导航缺少 ${label}`).toBeGreaterThan(cursor)
      cursor = idx
    }
    // 顶级模块按钮恰好 6 个（带 hint title；概览右侧 数据概览/关联图谱 胶囊不带 title）
    expect(nav.querySelectorAll('button[title]').length).toBe(6)
    expect(nav.textContent).not.toContain('知识图谱')
    expect(nav.textContent).not.toContain('价格仓库')
    expect(nav.textContent).not.toContain('询价')
  })

  it('概览默认渲染库规模统计、人材机构成与数据健康', async () => {
    await renderPage()
    expect(screen.getByText('条综合单价子目')).toBeTruthy()
    expect(screen.getByText('待补单价')).toBeTruthy()
    expect(screen.getByText('0 条')).toBeTruthy() // 可放心引用 = 总2 - 缺价1 - 草稿1（同一草稿缺价条目）
    expect(screen.getByText('专业 2')).toBeTruthy()
    expect(screen.getByText('分部 2')).toBeTruthy() // 市政→钢材|桩基；建筑无分部
  })

  it('概览双视图：关联图谱切换可见，数据概览切回', async () => {
    await renderPage()
    fireEvent.click(screen.getByRole('button', { name: '关联图谱' }))
    expect(screen.getByText('关联图谱视图桩')).toBeTruthy()
    expect(screen.queryByText('数据健康')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: '数据概览' }))
    expect(await screen.findByText('数据健康', {}, LOAD)).toBeTruthy()
    expect(screen.queryByText('关联图谱视图桩')).toBeNull()
  })

  it('模块切换：成本条目/测算项目/造价参考/复盘笔记各渲染对应视图', async () => {
    await renderPage()
    for (const [hint, marker] of [
      ['分类树 + 列表/表格管理', '成本条目视图桩'],
      ['报价/测算工作 · 版本留痕 · 沉淀回库', '测算项目视图桩'],
      ['案例分位数对标（不落表实时聚合）', '造价参考视图桩'],
      ['结论/边界/风险/证据沉淀判断', '复盘笔记视图桩'],
    ] as const) {
      fireEvent.click(screen.getByTitle(hint))
      expect(await screen.findByText(marker, {}, LOAD)).toBeTruthy()
    }
  })

  it('价格数据三段：默认价格源，询价库升格一等子页，价格仓库可切', async () => {
    await renderPage()
    fireEvent.click(screen.getByTitle('价格源 · 价格仓库 · 询价库'))
    const seg = screen.getByRole('tablist', { name: '价格数据子视图' })
    expect(seg.textContent).toContain('价格源')
    expect(seg.textContent).toContain('价格仓库')
    expect(seg.textContent).toContain('询价库')
    expect(await screen.findByText('价格源面板桩', {}, LOAD)).toBeTruthy()

    fireEvent.click(within(seg).getByRole('button', { name: '询价库' }))
    expect(screen.getByText('询价库面板桩')).toBeTruthy()
    fireEvent.click(within(seg).getByRole('button', { name: '价格仓库' }))
    expect(screen.getByText('价格仓库面板桩')).toBeTruthy()
  })

  it('快捷入口收敛为 4 项，价格数据卡可跳转对应模块', async () => {
    await renderPage()
    const quickSection = screen.getByText('快捷入口').closest('section')!
    const quickText = quickSection.textContent ?? ''
    // 只留动作与高频去处；造价参考/复盘笔记 走导航（导航内可达即不丢功能）
    expect(quickText).toContain('导入资料')
    expect(quickText).toContain('新建条目')
    expect(quickText).toContain('测算项目')
    expect(quickText).not.toContain('案例分位数对标')
    expect(quickText).not.toContain('复盘笔记')
    fireEvent.click(within(quickSection).getByText('价格数据'))
    expect(await screen.findByText('价格源面板桩', {}, LOAD)).toBeTruthy()
  })

  it('非概览模块显示回到概览，点击回数据概览', async () => {
    await renderPage()
    fireEvent.click(screen.getByTitle('报价/测算工作 · 版本留痕 · 沉淀回库'))
    fireEvent.click(screen.getByTitle('回到概览'))
    expect(await screen.findByText('数据健康', {}, LOAD)).toBeTruthy()
  })

  it('空库态：GettingStarted 三步引导', async () => {
    mocks.CostSearch.mockResolvedValue([])
    render(<CostLibraryPage />)
    expect(await screen.findByText('造价数据库还是空的', {}, LOAD)).toBeTruthy()
    expect(screen.getByText('选择文件')).toBeTruthy()
    expect(screen.getByText('订阅价格源')).toBeTruthy()
  })

  it('顶栏快捷动作：新建条目开弹窗；导入文件走 PickFiles 开导入弹窗', async () => {
    mocks.PickFiles.mockResolvedValue([{ path: 'C:/报价单.xlsx', name: '报价单.xlsx' }])
    await renderPage()
    fireEvent.click(screen.getByTitle('新建一条综合单价子目'))
    expect(screen.getByText('新建条目弹窗桩')).toBeTruthy()
    fireEvent.click(screen.getByTitle('导入 Excel/CSV/PDF/图片报价单，预览确认后入库'))
    expect(await screen.findByText('导入弹窗桩', {}, LOAD)).toBeTruthy()
    expect(mocks.PickFiles).toHaveBeenCalled()
  })
})
