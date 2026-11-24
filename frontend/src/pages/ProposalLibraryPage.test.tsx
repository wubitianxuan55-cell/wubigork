// ProposalLibraryPage.test.tsx — 一级板块「方案库」页面测试。
// 覆盖：①清单加载后自动选中首份方案、标题树首屏只展开到节、阅读窗落在首个节点
// ②点击节点读取该节、展开折叠节点 ③检索调用绑定并高亮命中词、点击命中跳转
// ④无命中/检索失败 ⑤空库三步引导与去办公板块 ⑥清单读取失败不伪装成空库
// ⑦删除需确认 ⑧窄窗页签切换 ⑨板块常驻保活下切回即静默重读清单、手动刷新、所选方案消失后的改选。桥接只 mock 页面消费的五个 GaeaProposal* 方法。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import type { ProposalHit, ProposalSearchResult, ProposalSectionBody, ProposalSectionNode, ProposalSummary } from '../gaea/lib/types/proposal'

const mocks = vi.hoisted(() => ({
  GaeaProposalList: vi.fn(),
  GaeaProposalOutline: vi.fn(),
  GaeaProposalRead: vi.fn(),
  GaeaProposalSearch: vi.fn(),
  GaeaProposalDelete: vi.fn(),
}))
vi.mock('../gaea/lib/bridge', () => ({ app: mocks }))

import { notifyBoardActive, resetBoardActive } from '../lib/boardActive'
import ProposalLibraryPage from './ProposalLibraryPage'

const p1: ProposalSummary = {
  name: 'n1', title: '技术标 甲', projectType: '技术标', year: '2026', status: '现行',
  summary: '污染土壤修复技术标', source: '甲.docx', sourceFile: 'D:/甲.md', updatedAt: '2026-11-20 09:30',
  sections: 4, chars: 90, indexDone: 5, indexTotal: 5,
}
const p2: ProposalSummary = {
  name: 'n2', title: '调查报告 乙', projectType: '调查报告', year: '2025', status: '录入中',
  summary: '', source: '', sourceFile: '', updatedAt: '2026-11-18 16:02',
  sections: 1, chars: 20, indexDone: 1, indexTotal: 4,
}

const tree1: ProposalSectionNode[] = [
  { id: 1, parentId: 0, level: 1, seq: 0, title: '第一章 施工降水', titlePath: '第一章 施工降水', chars: 30 },
  { id: 2, parentId: 1, level: 2, seq: 1, title: '1.1 沉降控制', titlePath: '第一章 施工降水 › 1.1 沉降控制', chars: 20 },
  { id: 3, parentId: 2, level: 3, seq: 2, title: '1.1.1 回灌措施', titlePath: '第一章 施工降水 › 1.1 沉降控制 › 1.1.1 回灌措施', chars: 15 },
  { id: 4, parentId: 0, level: 1, seq: 3, title: '第二章 安全保证', titlePath: '第二章 安全保证', chars: 25 },
]
const tree2: ProposalSectionNode[] = [
  { id: 11, parentId: 0, level: 1, seq: 0, title: '一 概述', titlePath: '一 概述', chars: 20 },
]

const bodies: Record<number, ProposalSectionBody> = {
  1: { id: 1, proposalTitle: '技术标 甲', status: '现行', title: '第一章 施工降水', titlePath: '第一章 施工降水', chars: 30,
    body: '基坑降水采用管井降水，降水期间监测周边建筑物沉降。', children: [{ id: 2, level: 2, title: '1.1 沉降控制', chars: 20 }] },
  2: { id: 2, proposalTitle: '技术标 甲', status: '现行', title: '1.1 沉降控制', titlePath: '第一章 施工降水 › 1.1 沉降控制', chars: 20,
    body: '如发现沉降超限，立即回灌并停止降水。', children: [{ id: 3, level: 3, title: '1.1.1 回灌措施', chars: 15 }] },
  3: { id: 3, proposalTitle: '技术标 甲', status: '现行', title: '1.1.1 回灌措施', titlePath: '第一章 施工降水 › 1.1 沉降控制 › 1.1.1 回灌措施', chars: 15,
    body: '回灌井布置在基坑外侧。', children: null },
  4: { id: 4, proposalTitle: '技术标 甲', status: '现行', title: '第二章 安全保证', titlePath: '第二章 安全保证', chars: 25,
    body: '边坡失稳的防控措施：分层开挖、及时支护。', children: null },
  11: { id: 11, proposalTitle: '调查报告 乙', status: '录入中', title: '一 概述', titlePath: '一 概述', chars: 20,
    body: '调查目的与范围。', children: null },
}

const hit4: ProposalHit = {
  proposalName: 'n1', proposalTitle: '技术标 甲', projectType: '技术标', year: '2026', status: '现行',
  sectionId: 4, level: 1, title: '第二章 安全保证', titlePath: '第二章 安全保证', chars: 25, childCount: 0,
  snippet: '【边坡失稳】的防控措施：分层开挖、及时支护。', matched: ['边坡', '失稳'],
}
const hit11: ProposalHit = {
  proposalName: 'n2', proposalTitle: '调查报告 乙', projectType: '调查报告', year: '2025', status: '录入中',
  sectionId: 11, level: 1, title: '一 概述', titlePath: '一 概述', chars: 20, childCount: 0,
  snippet: '调查目的与范围。', matched: [],
}

async function renderReady() {
  const utils = render(<ProposalLibraryPage />)
  await screen.findByTestId('proposal-reader')
  return utils
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.GaeaProposalList.mockResolvedValue([p1, p2])
  mocks.GaeaProposalOutline.mockImplementation(async (name: string) => (name === 'n1' ? tree1 : tree2))
  mocks.GaeaProposalRead.mockImplementation(async (id: number) => bodies[id])
  mocks.GaeaProposalSearch.mockResolvedValue({ hits: [hit4, hit11], mode: 'hybrid', note: '已本地精排' } satisfies ProposalSearchResult)
  mocks.GaeaProposalDelete.mockResolvedValue(undefined)
})

describe('ProposalLibraryPage 清单 / 标题树 / 阅读窗', () => {
  it('加载清单后自动选中第一份方案；标题树首屏只展开到节；阅读窗落在首个节点', async () => {
    await renderReady()
    expect(mocks.GaeaProposalOutline).toHaveBeenCalledWith('n1')
    expect(mocks.GaeaProposalRead).toHaveBeenCalledWith(1)
    const tree = screen.getByRole('tree')
    expect(within(tree).getAllByRole('treeitem').map((li) => li.getAttribute('aria-selected'))).toEqual(['true', 'false', 'false'])
    expect(within(tree).getByText('第一章 施工降水')).toBeTruthy()
    expect(within(tree).getByText('1.1 沉降控制')).toBeTruthy()
    expect(within(tree).getByText('第二章 安全保证')).toBeTruthy()
    // 2 级节点默认收起：三级标题不在首屏
    expect(screen.queryByText('1.1.1 回灌措施')).toBeNull()
    const reader = screen.getByTestId('proposal-reader')
    expect(within(reader).getByText('第一章 施工降水', { selector: 'h1' })).toBeTruthy()
    await waitFor(() => expect(reader.textContent).toContain('管井降水'))
    // 状态行：库规模；清单行：类型/年度/录入中标记/语义索引进度
    const status = screen.getByTestId('proposal-search-status')
    expect(status.textContent).toContain('共 2 份方案')
    expect(status.textContent).toContain('5 个标题节点')
    const list = screen.getByTestId('proposal-list')
    expect(within(list).getByText('录入中')).toBeTruthy()
    expect(within(list).getByText('语义索引 1/4')).toBeTruthy()
    expect(within(list).queryByText('语义索引 5/5')).toBeNull()
  })

  it('点击标题树节点读取该节；点击箭头展开被折叠的节点', async () => {
    await renderReady()
    // 点到的是被折叠的父节点：读取它的同时顺手展开，子标题随之出现
    fireEvent.click(within(screen.getByRole('tree')).getByRole('button', { name: /^1\.1 沉降控制/ }))
    await waitFor(() => expect(mocks.GaeaProposalRead).toHaveBeenCalledWith(2))
    expect((await screen.findByTestId('proposal-reader')).textContent).toContain('立即回灌')
    expect(within(screen.getByRole('tree')).getByText('1.1.1 回灌措施')).toBeTruthy()
    // 箭头单独收起/展开，不触发读取
    fireEvent.click(screen.getByRole('button', { name: '收起 1.1 沉降控制' }))
    expect(screen.queryByText('1.1.1 回灌措施')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: '展开 1.1 沉降控制' }))
    expect(screen.getByText('1.1.1 回灌措施')).toBeTruthy()
    expect(mocks.GaeaProposalRead).toHaveBeenCalledTimes(2)
  })

  it('上一节/下一节按文档顺序移动；子节胶囊可直接跳转', async () => {
    await renderReady()
    fireEvent.click(screen.getByRole('button', { name: /下一节.*1\.1 沉降控制/ }))
    await waitFor(() => expect(mocks.GaeaProposalRead).toHaveBeenCalledWith(2))
    fireEvent.click(screen.getByRole('button', { name: /下一节.*1\.1\.1 回灌措施/ }))
    await waitFor(() => expect(mocks.GaeaProposalRead).toHaveBeenCalledWith(3))
    fireEvent.click(screen.getByRole('button', { name: /上一节.*1\.1 沉降控制/ }))
    await waitFor(() => expect(mocks.GaeaProposalRead).toHaveBeenCalledTimes(4))
    expect(mocks.GaeaProposalRead).toHaveBeenLastCalledWith(2)
    fireEvent.click(screen.getByRole('button', { name: /1\.1\.1 回灌措施 · 15 字/ }))
    await waitFor(() => expect((screen.getByTestId('proposal-reader')).textContent).toContain('回灌井布置'))
  })

  it('切换方案会换成它的标题树', async () => {
    await renderReady()
    fireEvent.click(within(screen.getByTestId('proposal-list')).getByRole('button', { name: /^调查报告 乙/ }))
    await waitFor(() => expect(mocks.GaeaProposalOutline).toHaveBeenCalledWith('n2'))
    expect(within(screen.getByRole('tree')).getByText('一 概述')).toBeTruthy()
  })
})

describe('ProposalLibraryPage 检索', () => {
  it('搜索调用检索绑定（全文+语义、不过滤、30 条）；命中词高亮；点击命中跳到对应小节', async () => {
    await renderReady()
    fireEvent.change(screen.getByRole('textbox', { name: '检索方案库' }), { target: { value: '边坡 失稳' } })
    fireEvent.click(screen.getByRole('button', { name: '搜索' }))
    await screen.findByTestId('proposal-hits')
    expect(mocks.GaeaProposalSearch).toHaveBeenCalledWith('边坡 失稳', '', 0, '', '', '', 30)
    const hits = screen.getByTestId('proposal-hits')
    expect(within(hits).getAllByText('边坡失稳').filter((el) => el.tagName === 'MARK')).toHaveLength(1)
    expect(screen.getByTestId('proposal-search-status').textContent).toContain('全文 + 语义 · 已本地精排')
    // 命中在另一份方案里：先换方案再落到该节
    fireEvent.click(within(hits).getByText('一 概述', { selector: '.pl-hit-title' }).closest('button')!)
    await waitFor(() => expect(mocks.GaeaProposalOutline).toHaveBeenCalledWith('n2'))
    await waitFor(() => expect(mocks.GaeaProposalRead).toHaveBeenCalledWith(11))
    // 返回清单
    fireEvent.click(screen.getByRole('button', { name: '返回方案清单' }))
    expect(screen.queryByTestId('proposal-hits')).toBeNull()
    expect(screen.getByTestId('proposal-list')).toBeTruthy()
  })

  it('「只搜标题」把 title 作为范围传给后端；层级/类型筛选随请求带出', async () => {
    await renderReady()
    fireEvent.click(screen.getByRole('button', { name: '只搜标题' }))
    fireEvent.change(screen.getByRole('combobox', { name: '按层级筛选' }), { target: { value: '2' } })
    fireEvent.change(screen.getByRole('combobox', { name: '按类型筛选' }), { target: { value: '调查报告' } })
    fireEvent.change(screen.getByRole('textbox', { name: '检索方案库' }), { target: { value: '沉降' } })
    fireEvent.keyDown(screen.getByRole('textbox', { name: '检索方案库' }), { key: 'Enter' })
    await screen.findByTestId('proposal-hits')
    expect(mocks.GaeaProposalSearch).toHaveBeenCalledWith('沉降', 'title', 2, '调查报告', '', '', 30)
  })

  it('无命中给出明确提示', async () => {
    mocks.GaeaProposalSearch.mockResolvedValue({ hits: [], mode: 'hybrid', note: '' })
    await renderReady()
    fireEvent.change(screen.getByRole('textbox', { name: '检索方案库' }), { target: { value: '查不到' } })
    fireEvent.click(screen.getByRole('button', { name: '搜索' }))
    expect(await screen.findByText('没有命中')).toBeTruthy()
    expect(screen.getByTestId('proposal-search-status').textContent).toContain('命中 0 条')
  })

  it('检索失败如实报错并可重试', async () => {
    mocks.GaeaProposalSearch.mockRejectedValueOnce(new Error('库被占用')).mockResolvedValue({ hits: [hit4], mode: 'keyword', note: '' })
    await renderReady()
    fireEvent.change(screen.getByRole('textbox', { name: '检索方案库' }), { target: { value: '边坡' } })
    fireEvent.click(screen.getByRole('button', { name: '搜索' }))
    expect(await screen.findByText(/检索失败：库被占用/)).toBeTruthy()
    fireEvent.click(screen.getAllByRole('button', { name: /重试/ })[0])
    expect(await screen.findByTestId('proposal-hits')).toBeTruthy()
  })
})

describe('ProposalLibraryPage 空库与失败态', () => {
  it('空库显示三步引导；「去办公板块」发起导航到办公板块', async () => {
    mocks.GaeaProposalList.mockResolvedValue([])
    const listener = vi.fn()
    window.addEventListener('navigate', listener)
    try {
      render(<ProposalLibraryPage />)
      const empty = await screen.findByTestId('proposal-empty')
      expect(within(empty).getByText('方案库还是空的')).toBeTruthy()
      expect(within(empty).getAllByRole('listitem')).toHaveLength(3)
      fireEvent.click(within(empty).getByRole('button', { name: /去办公板块/ }))
      expect(listener).toHaveBeenCalledTimes(1)
      expect((listener.mock.calls[0][0] as CustomEvent).detail).toEqual({ page: 'gaea' })
    } finally {
      window.removeEventListener('navigate', listener)
    }
  })

  it('清单读取失败显示失败态并可重试，不伪装成空库', async () => {
    mocks.GaeaProposalList.mockRejectedValueOnce(new Error('database is locked'))
    render(<ProposalLibraryPage />)
    expect(await screen.findByText('方案库读取失败')).toBeTruthy()
    expect(screen.queryByTestId('proposal-empty')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: /重试/ }))
    expect(await screen.findByTestId('proposal-list')).toBeTruthy()
  })

  it('标题树读取失败如实显示并可重试', async () => {
    mocks.GaeaProposalOutline.mockRejectedValueOnce(new Error('方案库里没有《技术标 甲》'))
    render(<ProposalLibraryPage />)
    expect(await screen.findByText('标题树读取失败')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: /重试/ }))
    await screen.findByRole('tree')
  })
})

describe('ProposalLibraryPage 清单刷新（板块常驻保活）', () => {
  it('切去别的板块再切回来：静默重读清单，新录入的方案出现在清单里', async () => {
    mocks.GaeaProposalList.mockResolvedValueOnce([p1])
    await renderReady()
    expect(screen.getByTestId('proposal-search-status').textContent).toContain('共 1 份方案')
    mocks.GaeaProposalList.mockResolvedValue([p1, p2])
    try {
      // 壳层声明当前可见板块（lib/boardActive）：本板块 → 别处 → 回到本板块
      // 每次切换后让 React 提交一帧（真实壳层里切走与切回之间隔着整段停留时间）
      await act(async () => { notifyBoardActive('proposal') })
      expect(mocks.GaeaProposalList).toHaveBeenCalledTimes(1)
      await act(async () => { notifyBoardActive('gaea') })
      await act(async () => { notifyBoardActive('proposal') })
      await waitFor(() => expect(mocks.GaeaProposalList).toHaveBeenCalledTimes(2))
      await waitFor(() => expect(screen.getByTestId('proposal-search-status').textContent).toContain('共 2 份方案'))
      expect(within(screen.getByTestId('proposal-list')).getByText('调查报告 乙')).toBeTruthy()
      // 重读期间不清空清单：当前选中项与阅读窗原样保留
      expect(mocks.GaeaProposalOutline).toHaveBeenCalledTimes(1)
      expect(screen.getByTestId('proposal-reader').textContent).toContain('管井降水')
    } finally {
      resetBoardActive()
    }
  })

  it('停留在本板块时不重复读取；手动刷新按钮可重读清单', async () => {
    await renderReady()
    expect(mocks.GaeaProposalList).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: '刷新方案清单' }))
    await waitFor(() => expect(mocks.GaeaProposalList).toHaveBeenCalledTimes(2))
    expect(screen.getByTestId('proposal-list')).toBeTruthy()
  })

  it('切回时重读失败：保留旧清单并如实标出「刷新失败」，不伪装成空库', async () => {
    await renderReady()
    try {
      await act(async () => { notifyBoardActive('gaea') })
      mocks.GaeaProposalList.mockRejectedValueOnce(new Error('database is locked'))
      await act(async () => { notifyBoardActive('proposal') })
      await waitFor(() => expect(mocks.GaeaProposalList).toHaveBeenCalledTimes(2))
      expect(await screen.findByText('刷新失败')).toBeTruthy()
      expect(screen.getByTestId('proposal-list')).toBeTruthy()
      expect(screen.queryByText('方案库读取失败')).toBeNull()
      fireEvent.click(screen.getByRole('button', { name: '刷新方案清单' }))
      await waitFor(() => expect(screen.queryByText('刷新失败')).toBeNull())
    } finally {
      resetBoardActive()
    }
  })

  it('选中的方案已不在库里（别处被删）：刷新后改选第一份，不挂着消失的方案', async () => {
    mocks.GaeaProposalList.mockResolvedValueOnce([p1, p2])
    await renderReady()
    fireEvent.click(within(screen.getByTestId('proposal-list')).getByRole('button', { name: /^调查报告 乙/ }))
    await waitFor(() => expect(mocks.GaeaProposalOutline).toHaveBeenCalledWith('n2'))
    mocks.GaeaProposalList.mockResolvedValue([p1])
    try {
      await act(async () => { notifyBoardActive('gaea') })
      await act(async () => { notifyBoardActive('proposal') })
      await waitFor(() => expect(mocks.GaeaProposalList).toHaveBeenCalledTimes(2))
      await waitFor(() => expect(screen.queryByText('调查报告 乙')).toBeNull())
      expect(screen.getByTestId('proposal-reader').textContent).toContain('管井降水')
      expect(within(screen.getByTestId('proposal-list')).queryByText('调查报告 乙')).toBeNull()
    } finally {
      resetBoardActive()
    }
  })
})

describe('ProposalLibraryPage 删除与窄窗', () => {
  it('删除需在弹窗确认；确认后调用删除绑定并刷新清单', async () => {
    await renderReady()
    fireEvent.click(screen.getByRole('button', { name: '删除方案 技术标 甲' }))
    const dialog = await screen.findByRole('dialog')
    expect(dialog.textContent).toContain('确定删除《技术标 甲》吗？')
    expect(mocks.GaeaProposalDelete).not.toHaveBeenCalled()
    fireEvent.click(within(dialog).getByRole('button', { name: /删\s*除/ }))
    await waitFor(() => expect(mocks.GaeaProposalDelete).toHaveBeenCalledWith('n1'))
    await waitFor(() => expect(mocks.GaeaProposalList).toHaveBeenCalledTimes(2))
  })

  it('删除失败时弹窗留在原地并说明原因，清单不变', async () => {
    mocks.GaeaProposalDelete.mockRejectedValueOnce(new Error('只读'))
    await renderReady()
    fireEvent.click(screen.getByRole('button', { name: '删除方案 技术标 甲' }))
    const dialog = await screen.findByRole('dialog')
    fireEvent.click(within(dialog).getByRole('button', { name: /删\s*除/ }))
    expect(await within(dialog).findByText(/删除失败：只读/)).toBeTruthy()
    expect(screen.getByTestId('proposal-list')).toBeTruthy()
  })

  it('窄窗改为页签切换：默认停在目录，页签可切到正文', async () => {
    const spy = vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({ width: 800 } as DOMRect)
    try {
      await renderReady()
      // jsdom 不认容器查询：.pl-panes 的基础样式是 display:none，页签按「含隐藏元素」查询
      const tabs = await screen.findByRole('tablist', { name: '方案库视图', hidden: true })
      const tab = (name: string) => within(tabs).getByRole('tab', { name, hidden: true })
      expect(['方案', '目录', '正文'].map((n) => tab(n).textContent)).toEqual(['方案', '目录', '正文'])
      expect(tab('目录').getAttribute('aria-selected')).toBe('true')
      // 默认停在目录：正文栏带 hidden；切到正文后目录栏隐藏、正文栏显示
      expect(screen.getByTestId('proposal-reader').closest('.pl-col')?.getAttribute('hidden')).not.toBeNull()
      fireEvent.click(tab('正文'))
      expect(screen.getByTestId('proposal-reader').closest('.pl-col')?.getAttribute('hidden')).toBeNull()
      expect(screen.getByRole('tree', { hidden: true }).closest('.pl-col')?.getAttribute('hidden')).not.toBeNull()
    } finally {
      spy.mockRestore()
    }
  })
})
