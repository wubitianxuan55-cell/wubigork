// ProposalLibraryPage.indexedit.test.tsx — 方案库板块「索引与编辑优化批」（2026-11-28）的页面测试：
// 检索范围 / 分面收窄 / 排序 / 加载更多 / 多词放宽与语义模型不可用提示、命中定位与节内查找、
// 标题树筛选、编号层级提示、结构修正（升降级 / 删除 / 新增 / 拆分）、未保存守卫与快捷键、
// 编辑预览、批量改类型年度、导出、索引体检与重建、同题对照。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import type { ProposalHit, ProposalIndexStatus, ProposalSectionBody, ProposalSectionNode, ProposalSummary, ProposalTwin } from '../gaea/lib/types/proposal'

const healthy: ProposalIndexStatus = { vecDone: 5, vecTotal: 5, orphanFts: 0, missingFts: 0, staleFts: 0, missingChunks: 0, staleChunks: 0, model: 'test-embed' }

const mocks = vi.hoisted(() => ({
  GaeaProposalList: vi.fn(),
  GaeaProposalOutline: vi.fn(),
  GaeaProposalRead: vi.fn(),
  GaeaProposalSearch: vi.fn(),
  GaeaProposalDelete: vi.fn(),
  GaeaProposalUpdateMeta: vi.fn(),
  GaeaProposalUpdateSection: vi.fn(),
  GaeaProposalSectionOp: vi.fn(),
  GaeaProposalTwins: vi.fn(),
  GaeaProposalIndexStatus: vi.fn(),
  GaeaProposalReindex: vi.fn(),
  GaeaProposalExport: vi.fn(),
  SaveFileAs: vi.fn(),
}))
vi.mock('../gaea/lib/bridge', () => ({ app: mocks }))

import ProposalLibraryPage from './ProposalLibraryPage'

const p1: ProposalSummary = {
  name: 'n1', title: '技术标 甲', projectType: '技术标', year: '2026', status: '现行',
  summary: '污染土壤修复技术标', source: '甲.docx', sourceFile: 'D:/甲.md', updatedAt: '2026-11-20 09:30',
  sections: 4, chars: 90, indexDone: 5, indexTotal: 5,
}
const p2: ProposalSummary = {
  name: 'n2', title: '调查报告 乙', projectType: '调查报告', year: '2025', status: '现行',
  summary: '', source: '', sourceFile: '', updatedAt: '2026-11-18 16:02',
  sections: 1, chars: 20, indexDone: 4, indexTotal: 4,
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
  11: { id: 11, proposalTitle: '调查报告 乙', status: '现行', title: '一 概述', titlePath: '一 概述', chars: 20,
    body: '调查目的与范围。', children: null },
}

const hit4: ProposalHit = {
  proposalName: 'n1', proposalTitle: '技术标 甲', projectType: '技术标', year: '2026', status: '现行',
  sectionId: 4, level: 1, title: '第二章 安全保证', titlePath: '第二章 安全保证', chars: 25, childCount: 0,
  snippet: '【边坡失稳】的防控措施：分层开挖、及时支护。', matched: ['边坡', '失稳'], source: 'both',
}
const hit11: ProposalHit = {
  proposalName: 'n2', proposalTitle: '调查报告 乙', projectType: '调查报告', year: '2025', status: '现行',
  sectionId: 11, level: 1, title: '一 概述', titlePath: '一 概述', chars: 20, childCount: 0,
  snippet: '【监测】目的与范围。', matched: ['监测'], source: 'keyword',
}
const facets = [
  { key: 'n1', label: '技术标 甲', count: 18 },
  { key: 'n2', label: '调查报告 乙', count: 6 },
]

async function renderReady() {
  const utils = render(<ProposalLibraryPage />)
  await screen.findByTestId('proposal-reader')
  return utils
}

async function search(query: string) {
  fireEvent.change(screen.getByRole('textbox', { name: '检索方案库' }), { target: { value: query } })
  fireEvent.click(screen.getByRole('button', { name: '搜索' }))
  await screen.findByTestId('proposal-hits')
}

// 展开 1.1 沉降控制 并打开它的子节 1.1.1 回灌措施
async function openLeaf() {
  const tree = screen.getByRole('tree')
  fireEvent.click(within(tree).getByRole('button', { name: '展开 1.1 沉降控制' }))
  fireEvent.click(within(tree).getByRole('button', { name: /^1\.1\.1 回灌措施/ }))
  await waitFor(() => expect(mocks.GaeaProposalRead).toHaveBeenLastCalledWith(3))
}

function enterEdit() {
  fireEvent.click(within(screen.getByTestId('proposal-reader')).getByTestId('proposal-section-edit-open'))
  return screen.getByTestId('proposal-section-edit')
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.GaeaProposalList.mockResolvedValue([p1, p2])
  mocks.GaeaProposalOutline.mockImplementation(async (name: string) => (name === 'n1' ? tree1 : tree2))
  mocks.GaeaProposalRead.mockImplementation(async (id: number) => bodies[id])
  mocks.GaeaProposalSearch.mockResolvedValue({ hits: [hit4, hit11], total: 2, mode: 'hybrid', note: '已本地精排' })
  mocks.GaeaProposalUpdateMeta.mockResolvedValue('n1')
  mocks.GaeaProposalUpdateSection.mockResolvedValue(undefined)
  mocks.GaeaProposalTwins.mockResolvedValue([])
  mocks.GaeaProposalIndexStatus.mockResolvedValue(healthy)
  mocks.GaeaProposalReindex.mockResolvedValue(healthy)
  mocks.GaeaProposalSectionOp.mockResolvedValue({ op: 'promote', focusId: 3, affected: 1, created: 0, removed: 0, removedChars: 0, missingAssets: [] })
  mocks.SaveFileAs.mockResolvedValue('')
})

describe('检索：范围、分面、翻页、排序', () => {
  it('「仅本方案」把当前方案名带给后端；章分面点一下就限定在该章之下；点别的方案的 chip 切过去', async () => {
    mocks.GaeaProposalSearch.mockImplementation(async (_q: string, _s: string, _l: number, _t: string, _y: string, within: string) => (
      within === 'n1'
        ? { hits: [hit4], total: 18, byProposal: facets, byChapter: [{ key: '4', label: '第二章 安全保证', count: 6 }, { key: '1', label: '第一章 施工降水', count: 12 }], mode: 'keyword', note: '' }
        : { hits: [hit4, hit11], total: 24, byProposal: facets, byChapter: [], mode: 'keyword', note: '' }
    ))
    await renderReady()
    await search('监测')
    expect(screen.getByTestId('proposal-search-status').textContent).toContain('共 24 处命中')
    expect(screen.getAllByTestId('proposal-facet-proposal')).toHaveLength(2)
    expect(screen.queryAllByTestId('proposal-facet-chapter')).toHaveLength(0) // 没限定方案就没有「章」
    // 命中行标出来源
    expect(within(screen.getByTestId('proposal-hits')).getByText('全文+语义')).toBeTruthy()

    fireEvent.click(screen.getByTestId('proposal-range-proposal'))
    await waitFor(() => expect(mocks.GaeaProposalSearch).toHaveBeenLastCalledWith('监测', '', 0, '', '', 'n1', 0, 0, '', 30))
    const chapters = await screen.findAllByTestId('proposal-facet-chapter')
    expect(chapters).toHaveLength(2)
    fireEvent.click(chapters[0])
    await waitFor(() => expect(mocks.GaeaProposalSearch).toHaveBeenLastCalledWith('监测', '', 0, '', '', 'n1', 4, 0, '', 30))
    expect(screen.getByTestId('proposal-under-clear').textContent).toContain('第二章 安全保证')
    fireEvent.click(screen.getByTestId('proposal-under-clear'))
    await waitFor(() => expect(mocks.GaeaProposalSearch).toHaveBeenLastCalledWith('监测', '', 0, '', '', 'n1', 0, 0, '', 30))

    fireEvent.click(screen.getAllByTestId('proposal-facet-proposal')[1])
    await waitFor(() => expect(mocks.GaeaProposalOutline).toHaveBeenLastCalledWith('n2'))
    await waitFor(() => expect(mocks.GaeaProposalSearch).toHaveBeenLastCalledWith('监测', '', 0, '', '', 'n2', 0, 0, '', 30))
  })

  it('排序切到「文档顺序」随请求带出', async () => {
    await renderReady()
    await search('监测')
    fireEvent.click(screen.getByRole('button', { name: '文档顺序' }))
    await waitFor(() => expect(mocks.GaeaProposalSearch).toHaveBeenLastCalledWith('监测', '', 0, '', '', '', 0, 0, 'doc', 30))
  })

  it('命中超过一页：报总数、出「加载更多」，点一下接上下一页', async () => {
    mocks.GaeaProposalSearch.mockImplementation(async (_q: string, _s: string, _l: number, _t: string, _y: string, _w: string, _u: number, offset: number) => (
      offset === 0 ? { hits: [hit4], total: 2, mode: 'keyword', note: '' } : { hits: [hit11], total: 2, mode: 'keyword', note: '' }
    ))
    await renderReady()
    await search('监测')
    expect(screen.getByTestId('proposal-search-status').textContent).toContain('共 2 处命中')
    expect(within(screen.getByTestId('proposal-hits')).getAllByRole('button')).toHaveLength(1)
    expect(screen.getByTestId('proposal-load-more').textContent).toContain('还有 1 条')
    fireEvent.click(screen.getByTestId('proposal-load-more'))
    await waitFor(() => expect(mocks.GaeaProposalSearch).toHaveBeenLastCalledWith('监测', '', 0, '', '', '', 0, 1, '', 30))
    await waitFor(() => expect(within(screen.getByTestId('proposal-hits')).getAllByRole('button')).toHaveLength(2))
    await waitFor(() => expect(screen.queryByTestId('proposal-load-more')).toBeNull())
  })

  it('多词放宽会说明；语义模型不可用时出提示并可重试', async () => {
    mocks.GaeaProposalSearch.mockResolvedValue({ hits: [hit4], total: 1, mode: 'keyword', note: '本地 embedding 不可用，本次只按全文路检索', relaxed: true, searchError: '本地 embedding 不可用' })
    await renderReady()
    await search('降水 扬尘')
    expect(screen.getByTestId('proposal-search-status').textContent).toContain('已放宽为任一词')
    const notice = screen.getByTestId('proposal-search-degraded')
    expect(notice.textContent).toContain('只按关键词匹配')
    fireEvent.click(within(notice).getByRole('button', { name: /重试/ }))
    await waitFor(() => expect(mocks.GaeaProposalSearch).toHaveBeenCalledTimes(2))
  })
})

describe('命中定位与节内查找', () => {
  it('点命中跳到小节后：查找条打开、落在命中那一段并高亮', async () => {
    await renderReady()
    await search('边坡')
    fireEvent.click(screen.getByTestId('proposal-hits').querySelector('.pl-hit')!)
    await waitFor(() => expect(mocks.GaeaProposalRead).toHaveBeenLastCalledWith(4))
    const find = await screen.findByTestId('proposal-find')
    expect((within(find).getByRole('textbox') as HTMLInputElement).value).toBe('边坡')
    expect(find.textContent).toContain('第 1 处，共 1 处')
  })

  it('阅读窗里查词：逐处翻，关掉即清', async () => {
    await renderReady()
    fireEvent.click(screen.getByTestId('proposal-find-open'))
    const find = await screen.findByTestId('proposal-find')
    fireEvent.change(within(find).getByRole('textbox', { name: '在本节中查找' }), { target: { value: '降水' } })
    expect(find.textContent).toContain('第 1 处，共 3 处')
    fireEvent.click(within(find).getByRole('button', { name: '下一处' }))
    expect(find.textContent).toContain('第 2 处，共 3 处')
    fireEvent.click(within(find).getByRole('button', { name: '上一处' }))
    expect(find.textContent).toContain('第 1 处，共 3 处')
    fireEvent.click(within(find).getByRole('button', { name: '关闭查找' }))
    await waitFor(() => expect(screen.queryByTestId('proposal-find')).toBeNull())
  })
})

describe('标题树：筛选与编号提示', () => {
  it('筛选只留匹配的标题和它的上级，命中词高亮；清除后恢复', async () => {
    await renderReady()
    const tree = screen.getByRole('tree')
    fireEvent.change(screen.getByRole('textbox', { name: '筛选标题' }), { target: { value: '回灌' } })
    expect(within(tree).getAllByRole('treeitem').map((li) => li.getAttribute('aria-level'))).toEqual(['1', '2', '3'])
    expect(within(tree).getByText('回灌', { selector: 'mark' })).toBeTruthy()
    expect(within(tree).queryByText('第二章 安全保证')).toBeNull()
    expect(screen.getByText('3 项')).toBeTruthy()
    fireEvent.click(screen.getByTestId('proposal-tree-filter-clear'))
    await waitFor(() => expect(within(tree).queryByText('1.1.1 回灌措施')).toBeNull()) // 回到默认折叠
    expect(within(tree).getByText('第二章 安全保证')).toBeTruthy()
  })

  it('调过层级而编号没改：标题旁出「编号与层级不符」提示', async () => {
    mocks.GaeaProposalOutline.mockImplementation(async (name: string) => (name === 'n1' ? tree1.map((n) => (n.id === 2 ? { ...n, level: 3 } : n)) : tree2))
    await renderReady()
    const warn = screen.getByTestId('proposal-num-warn')
    expect(warn.getAttribute('aria-label')).toBe('编号与层级不符：标题里的编号是 2 级写法，现在在第 3 层。可以改标题里的编号，或用「调整结构」改回去')
  })
})

describe('结构修正', () => {
  it('菜单按可用态置灰并说明原因；升级后调结构绑定、重取标题树、给回执', async () => {
    await renderReady()
    await openLeaf()
    const before = mocks.GaeaProposalOutline.mock.calls.length
    fireEvent.click(screen.getByTestId('proposal-structure-open'))
    expect((screen.getByTestId('proposal-structure-promote') as HTMLButtonElement).disabled).toBe(false)
    expect((screen.getByTestId('proposal-structure-demote') as HTMLButtonElement).disabled).toBe(true)
    expect(screen.getByTestId('proposal-structure-demote').getAttribute('title')).toContain('前面没有同级小节')
    fireEvent.click(screen.getByTestId('proposal-structure-promote'))
    await waitFor(() => expect(mocks.GaeaProposalSectionOp).toHaveBeenCalledWith(3, 'promote', 0, ''))
    await waitFor(() => expect(mocks.GaeaProposalOutline.mock.calls.length).toBe(before + 1))
    expect((await screen.findByTestId('proposal-structure-msg')).textContent).toContain('已升级')
    expect(screen.getByTestId('proposal-structure-msg').textContent).toContain('编号没有改')
  })

  it('删除小节要先确认；新增子节问标题，新小节直接进编辑态', async () => {
    await renderReady()
    await openLeaf()
    mocks.GaeaProposalSectionOp.mockImplementation(async (_id: number, op: string) => (op === 'remove'
      ? { op, focusId: 2, affected: 0, created: 0, removed: 1, removedChars: 15, missingAssets: [] }
      : { op, focusId: 99, affected: 0, created: 1, removed: 0, removedChars: 0, missingAssets: [] }))
    const grown = [...tree1, { id: 99, parentId: 3, level: 4, seq: 4, title: '新小节', titlePath: '第一章 施工降水 › 1.1 沉降控制 › 1.1.1 回灌措施 › 新小节', chars: 0 }]
    mocks.GaeaProposalOutline.mockImplementation(async (name: string) => (name === 'n1' ? grown : tree2))
    mocks.GaeaProposalRead.mockImplementation(async (id: number) => (id === 99
      ? { id: 99, proposalTitle: '技术标 甲', status: '现行', title: '新小节', titlePath: grown[4].titlePath, chars: 0, body: '', children: null }
      : bodies[id]))

    fireEvent.click(screen.getByTestId('proposal-structure-open'))
    expect(screen.getByTestId('proposal-structure-remove').textContent).toContain('删除本节')
    fireEvent.click(screen.getByTestId('proposal-structure-remove'))
    const dialog = await screen.findByRole('dialog')
    expect(dialog.textContent).toContain('确定删除《1.1.1 回灌措施》')
    expect(mocks.GaeaProposalSectionOp).not.toHaveBeenCalled()
    fireEvent.click(within(dialog).getByRole('button', { name: '取 消' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())

    fireEvent.click(screen.getByTestId('proposal-structure-open'))
    fireEvent.click(screen.getByTestId('proposal-structure-addChild'))
    const titleBox = await screen.findByTestId('proposal-new-title')
    fireEvent.change(titleBox, { target: { value: '新小节' } })
    fireEvent.keyDown(titleBox, { key: 'Enter' })
    await waitFor(() => expect(mocks.GaeaProposalSectionOp).toHaveBeenCalledWith(3, 'addChild', 0, '新小节'))
    const edit = await screen.findByTestId('proposal-section-edit')
    expect((within(edit).getByTestId('proposal-edit-title') as HTMLInputElement).value).toBe('新小节')
  })

  it('拆分：光标在第一行不能拆；有未保存改动时先保存草稿再拆', async () => {
    await renderReady()
    mocks.GaeaProposalRead.mockImplementation(async (id: number) => (id === 2 ? { ...bodies[2], body: '第一段。\n\n第二段讲回灌。' } : bodies[id]))
    const tree = screen.getByRole('tree')
    fireEvent.click(within(tree).getByRole('button', { name: /^1\.1 沉降控制/ }))
    await waitFor(() => expect(mocks.GaeaProposalRead).toHaveBeenLastCalledWith(2))
    await screen.findByText('第二段讲回灌。')
    const edit = enterEdit()
    const area = within(edit).getByTestId('proposal-edit-body') as HTMLTextAreaElement
    area.setSelectionRange(2, 2)
    fireEvent.click(within(edit).getByTestId('proposal-split'))
    expect((within(edit).getByTestId('proposal-edit-msg')).textContent).toContain('光标要放在')

    fireEvent.change(within(edit).getByTestId('proposal-edit-title'), { target: { value: '1.1 沉降控制（改）' } })
    area.setSelectionRange(6, 6)
    fireEvent.click(within(edit).getByTestId('proposal-split'))
    const titleBox = await screen.findByTestId('proposal-new-title')
    expect((titleBox as HTMLInputElement).value).toBe('第二段讲回灌。')
    fireEvent.click(screen.getByRole('button', { name: '拆 分' }))
    await waitFor(() => expect(mocks.GaeaProposalUpdateSection).toHaveBeenCalledWith(2, '1.1 沉降控制（改）', true, '第一段。\n\n第二段讲回灌。', false))
    await waitFor(() => expect(mocks.GaeaProposalSectionOp).toHaveBeenCalledWith(2, 'split', 6, '第二段讲回灌。'))
  })

  it('结构操作失败：原因留在阅读窗，标题树不动', async () => {
    await renderReady()
    const before = mocks.GaeaProposalOutline.mock.calls.length
    mocks.GaeaProposalSectionOp.mockRejectedValueOnce(new Error('这份方案还在分批录入、尚未收尾，收尾后才能调整结构'))
    fireEvent.click(screen.getByTestId('proposal-structure-open'))
    fireEvent.click(screen.getByTestId('proposal-structure-moveDown'))
    const msg = await screen.findByTestId('proposal-structure-msg')
    expect(msg.getAttribute('role')).toBe('alert')
    expect(msg.textContent).toContain('尚未收尾')
    expect(mocks.GaeaProposalOutline.mock.calls.length).toBe(before)
  })
})

describe('编辑体验：守卫、快捷键、预览', () => {
  it('有未保存改动时换节先问；不让走就留在编辑态', async () => {
    await renderReady()
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    try {
      const edit = enterEdit()
      fireEvent.change(within(edit).getByTestId('proposal-edit-body'), { target: { value: '改了正文。' } })
      expect(edit.querySelector('.pl-edit-change')?.textContent).toBe('−20 字') // 改动量提示
      const reads = mocks.GaeaProposalRead.mock.calls.length
      fireEvent.click(within(screen.getByRole('tree')).getByRole('button', { name: /^1\.1 沉降控制/ }))
      expect(confirm).toHaveBeenCalledTimes(1)
      expect(screen.getByTestId('proposal-section-edit')).toBeTruthy()
      expect(mocks.GaeaProposalRead.mock.calls.length).toBe(reads)
      confirm.mockReturnValue(true)
      fireEvent.click(within(screen.getByRole('tree')).getByRole('button', { name: /^1\.1 沉降控制/ }))
      await waitFor(() => expect(screen.queryByTestId('proposal-section-edit')).toBeNull())
    } finally {
      confirm.mockRestore()
    }
  })

  it('Ctrl+S 保存；Esc 有改动先问，不让放弃就留在编辑态', async () => {
    await renderReady()
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    try {
      const edit = enterEdit()
      fireEvent.change(within(edit).getByTestId('proposal-edit-body'), { target: { value: '新正文。' } })
      fireEvent.keyDown(window, { key: 'Escape' })
      expect(screen.getByTestId('proposal-section-edit')).toBeTruthy()
      fireEvent.keyDown(window, { key: 's', ctrlKey: true })
      await waitFor(() => expect(mocks.GaeaProposalUpdateSection).toHaveBeenCalledWith(1, '第一章 施工降水', false, '新正文。', true))
      await waitFor(() => expect(screen.queryByTestId('proposal-section-edit')).toBeNull())
    } finally {
      confirm.mockRestore()
    }
  })

  it('保存改了标题的小节后标题树立即重取（不被未保存守卫拦）', async () => {
    await renderReady()
    const before = mocks.GaeaProposalOutline.mock.calls.length
    const edit = enterEdit()
    fireEvent.change(within(edit).getByTestId('proposal-edit-title'), { target: { value: '第一章 降水施工' } })
    fireEvent.click(within(edit).getByTestId('proposal-edit-save'))
    await waitFor(() => expect(mocks.GaeaProposalOutline.mock.calls.length).toBe(before + 1))
  })

  it('编辑 ⇄ 预览：预览里能看到渲染后的正文', async () => {
    await renderReady()
    const edit = enterEdit()
    fireEvent.change(within(edit).getByTestId('proposal-edit-body'), { target: { value: '**加粗**的正文' } })
    fireEvent.click(within(edit).getByTestId('proposal-edit-preview'))
    const preview = within(edit).getByTestId('proposal-edit-preview-body')
    expect(preview.querySelector('strong')?.textContent).toBe('加粗')
    expect(screen.queryByTestId('proposal-edit-body')).toBeNull()
  })
})

describe('元数据：可选可填、批量改、导出', () => {
  it('类型与年度可选已有写法；没摘要的方案在清单上标「摘要待补」', async () => {
    await renderReady()
    const list = screen.getByTestId('proposal-list')
    expect(within(list).getByText('摘要待补')).toBeTruthy()
    fireEvent.click(within(list).getByRole('button', { name: '编辑方案 技术标 甲' }))
    await screen.findByTestId('proposal-meta-type')
    const typeBox = screen.getByTestId('proposal-meta-type') as HTMLInputElement
    expect(typeBox.getAttribute('list')).toBe('proposal-type-options')
    expect(document.getElementById('proposal-type-options')?.querySelectorAll('option')).toHaveLength(2)
  })

  it('勾选多份方案批量改类型：没填的项保持原值，逐份保存', async () => {
    await renderReady()
    fireEvent.click(screen.getByRole('checkbox', { name: '选择方案 技术标 甲' }))
    fireEvent.click(screen.getByRole('checkbox', { name: '选择方案 调查报告 乙' }))
    expect(screen.getByTestId('proposal-batch-bar').textContent).toContain('已选 2 份')
    fireEvent.click(screen.getByTestId('proposal-batch-open'))
    fireEvent.change(await screen.findByTestId('proposal-batch-type'), { target: { value: '修复方案' } })
    fireEvent.click(screen.getByRole('button', { name: '应 用' }))
    await waitFor(() => expect(mocks.GaeaProposalUpdateMeta).toHaveBeenCalledWith('n1', '技术标 甲', '修复方案', '2026', '污染土壤修复技术标'))
    await waitFor(() => expect(mocks.GaeaProposalUpdateMeta).toHaveBeenCalledWith('n2', '调查报告 乙', '修复方案', '2025', ''))
    await waitFor(() => expect(screen.queryByTestId('proposal-batch-bar')).toBeNull())
  })

  it('导出：另存对话框选路径后由后端写盘；取消另存不导出', async () => {
    await renderReady()
    mocks.SaveFileAs.mockResolvedValueOnce('D:/导出/技术标 甲.md')
    mocks.GaeaProposalExport.mockResolvedValue({ path: 'D:/导出/技术标 甲.md', assetsDir: 'D:/导出/技术标 甲-assets', sections: 4, chars: 90, images: 2, missing: 0 })
    fireEvent.click(screen.getByRole('button', { name: '导出方案 技术标 甲' }))
    await waitFor(() => expect(mocks.SaveFileAs).toHaveBeenCalledWith('技术标 甲.md', ''))
    await waitFor(() => expect(mocks.GaeaProposalExport).toHaveBeenCalledWith('n1', 'D:/导出/技术标 甲.md'))
    expect((await screen.findByTestId('proposal-export-msg')).textContent).toContain('已导出到')
    expect(screen.getByTestId('proposal-export-msg').textContent).toContain('图片 2 张')
    fireEvent.click(screen.getByRole('button', { name: '导出方案 调查报告 乙' }))
    await waitFor(() => expect(mocks.SaveFileAs).toHaveBeenLastCalledWith('调查报告 乙.md', ''))
    expect(mocks.GaeaProposalExport).toHaveBeenCalledTimes(1)
  })

  it('导出失败如实提示', async () => {
    await renderReady()
    mocks.SaveFileAs.mockResolvedValueOnce('D:/x.md')
    mocks.GaeaProposalExport.mockRejectedValueOnce(new Error('写入导出文件失败'))
    fireEvent.click(screen.getByRole('button', { name: '导出方案 技术标 甲' }))
    expect((await screen.findByTestId('proposal-export-msg')).textContent).toContain('写入导出文件失败')
  })
})

describe('索引体检与同题对照', () => {
  it('索引有问题时状态钮转「待修复」；体检列出条目；修复并重建后回到就绪', async () => {
    mocks.GaeaProposalIndexStatus.mockResolvedValue({ ...healthy, orphanFts: 3, vecDone: 3, vecTotal: 5 })
    await renderReady()
    const button = screen.getByTestId('proposal-index-status')
    expect(button.textContent).toContain('索引 待修复')
    fireEvent.click(button)
    const pop = await screen.findByTestId('proposal-index-pop')
    expect(pop.textContent).toContain('3 条全文索引对应的小节已不存在')
    expect(pop.textContent).toContain('3/5')
    fireEvent.click(within(pop).getByTestId('proposal-reindex'))
    await waitFor(() => expect(mocks.GaeaProposalReindex).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(screen.getByTestId('proposal-index-status').textContent).toContain('语义索引 就绪'))
  })

  it('重建失败：原因留在体检浮层里', async () => {
    await renderReady()
    mocks.GaeaProposalReindex.mockRejectedValueOnce(new Error('语义索引正在重建，请稍候再试'))
    fireEvent.click(screen.getByTestId('proposal-index-status'))
    const pop = await screen.findByTestId('proposal-index-pop')
    fireEvent.click(within(pop).getByTestId('proposal-reindex'))
    expect((await within(pop).findByRole('alert')).textContent).toContain('请稍候再试')
  })

  it('别的方案里有同题小节：阅读窗列出，点开对照，可跳过去', async () => {
    const twin: ProposalTwin = { proposalName: 'n2', proposalTitle: '调查报告 乙', projectType: '调查报告', year: '2025', sectionId: 11, level: 1, title: '一 概述', titlePath: '一 概述', chars: 20 }
    mocks.GaeaProposalTwins.mockResolvedValue([twin])
    await renderReady()
    const chip = await screen.findByTestId('proposal-twin')
    expect(chip.textContent).toContain('调查报告 乙 › 一 概述')
    fireEvent.click(chip)
    expect(await screen.findByText('调查目的与范围。')).toBeTruthy()
    fireEvent.click(screen.getByTestId('proposal-twin-jump'))
    await waitFor(() => expect(mocks.GaeaProposalOutline).toHaveBeenLastCalledWith('n2'))
  })
})
