// ForeshadowPanel.test.tsx — 伏笔登记表面板关键路径
// 覆盖：手工登记→全量写回→列表出现；状态流转写回 hinted；保存失败回滚提示；删除（confirm）；
// 一致性体检（LintForeshadows）：概要行渲染 / findings Tag+说明+章节引用 / 空 findings 提示。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go。
// 组件经 gaea/lib/bridge 的 app 调用 GetForeshadows / SaveForeshadows / LintForeshadows。
vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      GetForeshadows: vi.fn().mockResolvedValue({ items: [] }),
      SaveForeshadows: vi.fn().mockResolvedValue(undefined),
      LintForeshadows: vi.fn().mockResolvedValue(null),
      // t1-P4 增量绑定：默认空/不可用（面板降级），用例内按需覆写
      GetForeshadowStats: vi.fn().mockResolvedValue(null),
      GetLastForeshadowSync: vi.fn().mockRejectedValue(new Error('尚未分析')),
      DeleteChapterForeshadows: vi.fn().mockResolvedValue({ deleted: 3 }),
      CleanChapterAnalysisForeshadows: vi.fn().mockResolvedValue({ deleted: 1, rolledBack: 2 }),
      ClearProjectForeshadowsForReset: vi.fn().mockResolvedValue({ deleted: 4, resetManual: 1 }),
    },
  }
})

import ForeshadowPanel from './ForeshadowPanel'
import { app } from '../../gaea/lib/bridge'
import type { ForeshadowItemData } from '../../types'
import type { ForeshadowLintReport } from '../../gaea/lib/bridge/novel'

const EXISTING: ForeshadowItemData = {
  id: 'plot_001_abc',
  category: 'character',
  description: '主角左臂旧伤',
  planted_in: '001.md',
  status: 'planted',
  is_long_term: false,
}

describe('ForeshadowPanel 行 memo（v4.352）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(app.GetForeshadows).mockResolvedValue({
      items: [
        EXISTING,
        { ...EXISTING, id: 'plot_002_def', description: '城主的佩剑来历', planted_in: '002.md' },
        { ...EXISTING, id: 'plot_003_ghi', description: '雨夜血案的目击者', planted_in: '003.md' },
      ],
    })
    vi.mocked(app.SaveForeshadows).mockResolvedValue(undefined)
  })

  it('行内编辑键入仍可保存（memo 化后行为不回归）', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText('主角左臂旧伤')
    fireEvent.click(screen.getByLabelText('编辑伏笔：主角左臂旧伤'))
    const box = await screen.findByRole('textbox')
    fireEvent.change(box, { target: { value: '主角左臂旧伤（三处刀口）' } })
    fireEvent.click(screen.getByText('保存'))
    await waitFor(() => expect(vi.mocked(app.SaveForeshadows)).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(app.SaveForeshadows).mock.calls[0][0] as string) as ForeshadowItemData[]
    expect(payload.find((x) => x.id === 'plot_001_abc')?.description).toBe('主角左臂旧伤（三处刀口）')
    expect(payload).toHaveLength(3)
  })

  it('编辑取消不写回；其余行不受影响', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText('城主的佩剑来历')
    fireEvent.click(screen.getByLabelText('编辑伏笔：城主的佩剑来历'))
    fireEvent.change(await screen.findByRole('textbox'), { target: { value: 'x' } })
    fireEvent.click(screen.getByText('取消'))
    await waitFor(() => expect(screen.getByText('城主的佩剑来历')).toBeTruthy())
    expect(vi.mocked(app.SaveForeshadows)).not.toHaveBeenCalled()
  })
})

describe('ForeshadowPanel 手工登记闭环', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [EXISTING] })
    vi.mocked(app.SaveForeshadows).mockResolvedValue(undefined)
  })

  it('登记伏笔：提交后 SaveForeshadows 全量写回（保留既有条目 + manual_ 新条目），列表出现', async () => {
    render(<ForeshadowPanel />)
    expect(await screen.findByText('主角左臂旧伤')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: /登记伏笔/ }))
    fireEvent.change(screen.getByPlaceholderText(/伏笔描述/), { target: { value: '神秘铜匣的钥匙' } })
    fireEvent.click(screen.getByRole('button', { name: /登记$/ }))

    await waitFor(() => expect(vi.mocked(app.SaveForeshadows)).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(app.SaveForeshadows).mock.calls[0][0] as string) as ForeshadowItemData[]
    expect(payload).toHaveLength(2)
    // 既有 AI 条目原样保留（全量写回不丢数据）
    expect(payload[0]).toEqual(EXISTING)
    // 新条目：manual_ 前缀 ID、默认 planted、章节号补零
    expect(payload[1].id).toMatch(/^manual_\d+$/)
    expect(payload[1].status).toBe('planted')
    expect(payload[1].description).toBe('神秘铜匣的钥匙')
    expect(payload[1].category).toBe('plot')
    expect(payload[1].planted_in).toBe('001.md')
    expect(payload[1].is_long_term).toBe(false)

    // 列表出现新条目 + 成功提示
    expect(await screen.findByText('神秘铜匣的钥匙')).toBeTruthy()
    expect(await screen.findByText('伏笔已登记')).toBeTruthy()
  })

  it('描述为空时拒登记，不触发写回', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText('主角左臂旧伤')
    fireEvent.click(screen.getByRole('button', { name: /登记伏笔/ }))
    fireEvent.click(screen.getByRole('button', { name: /登记$/ }))
    expect(await screen.findByText('请填写伏笔描述')).toBeTruthy()
    expect(vi.mocked(app.SaveForeshadows)).not.toHaveBeenCalled()
  })

  it('状态流转：标记暗示 → 乐观更新徽标并写回 hinted', async () => {
    render(<ForeshadowPanel />)
    expect(await screen.findByText('主角左臂旧伤')).toBeTruthy()
    expect(screen.getByText('已埋设')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: '标记暗示' }))

    await waitFor(() => expect(vi.mocked(app.SaveForeshadows)).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(app.SaveForeshadows).mock.calls[0][0] as string) as ForeshadowItemData[]
    expect(payload[0].status).toBe('hinted')
    expect(await screen.findByText('已暗示')).toBeTruthy()
  })

  it('保存失败：回滚列表并提示', async () => {
    vi.mocked(app.SaveForeshadows).mockRejectedValueOnce(new Error('磁盘已满'))
    render(<ForeshadowPanel />)
    await screen.findByText('主角左臂旧伤')

    fireEvent.click(screen.getByRole('button', { name: '标记暗示' }))

    expect(await screen.findByText(/伏笔保存失败，已回滚/)).toBeTruthy()
    // 回滚后状态徽标恢复 planted
    expect(await screen.findByText('已埋设')).toBeTruthy()
  })

  it('删除：confirm 后全量写回剩余条目', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText('主角左臂旧伤')

    fireEvent.click(screen.getByRole('button', { name: '删除伏笔：主角左臂旧伤' }))
    // antd Popconfirm 双字按钮会插空格（「删 除」）
    fireEvent.click(await screen.findByRole('button', { name: /^删\s*除$/ }))

    await waitFor(() => expect(vi.mocked(app.SaveForeshadows)).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(app.SaveForeshadows).mock.calls[0][0] as string) as ForeshadowItemData[]
    expect(payload).toEqual([])
    expect(await screen.findByText(/还没有伏笔登记/)).toBeTruthy()
  })
})

describe('ForeshadowPanel 一致性体检', () => {
  const LINT_REPORT: ForeshadowLintReport = {
    totalChapters: 12,
    items: 5,
    planted: 3,
    hinted: 1,
    revealed: 1,
    longTerm: 1,
    findings: [
      {
        code: 'ordering',
        severity: 'high',
        foreshadowId: 'manual_demo_1',
        itemDesc: '神秘铜匣的钥匙',
        message: '回收章 001.md 早于埋设章 002.md，登记序颠倒',
        chapter: '001.md',
      },
      {
        code: 'stale',
        severity: 'medium',
        foreshadowId: 'manual_demo_2',
        itemDesc: '主角左臂旧伤的来历',
        message: '已悬置 11 章未回收（第 1 章埋设），考虑回收或标记长线',
        chapter: '001.md',
      },
    ],
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(app.SaveForeshadows).mockResolvedValue(undefined)
    vi.mocked(app.LintForeshadows).mockResolvedValue(LINT_REPORT)
  })

  it('点击「一致性体检」→ 概要行渲染统计数字', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText(/还没有伏笔登记/)

    fireEvent.click(screen.getByRole('button', { name: /一致性体检/ }))

    expect(await screen.findByText(/全书 12 章 · 登记 5 条/)).toBeTruthy()
    expect(screen.getByText(/已埋 3 \/ 暗示 1 \/ 回收 1（长线 1）/)).toBeTruthy()
    expect(vi.mocked(app.LintForeshadows)).toHaveBeenCalledTimes(1)
  })

  it('findings 渲染：severity Tag（重/中）+ 说明文本 + 章节引用', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText(/还没有伏笔登记/)

    fireEvent.click(screen.getByRole('button', { name: /一致性体检/ }))

    expect(await screen.findByText('重')).toBeTruthy()
    expect(screen.getByText('中')).toBeTruthy()
    expect(screen.getByText('回收章 001.md 早于埋设章 002.md，登记序颠倒')).toBeTruthy()
    expect(screen.getByText('已悬置 11 章未回收（第 1 章埋设），考虑回收或标记长线')).toBeTruthy()
    expect(screen.getAllByText('→ 001.md')).toHaveLength(2)
  })

  it('findings 为空 → 「未发现一致性问题」提示', async () => {
    vi.mocked(app.LintForeshadows).mockResolvedValue({ ...LINT_REPORT, findings: [] })
    render(<ForeshadowPanel />)
    await screen.findByText(/还没有伏笔登记/)

    fireEvent.click(screen.getByRole('button', { name: /一致性体检/ }))

    expect(await screen.findByText('未发现一致性问题')).toBeTruthy()
  })
})

describe('ForeshadowPanel 调度可视面（t1-P4）', () => {
  const URGENT: ForeshadowItemData = {
    id: 'plot_001_urg',
    category: 'plot',
    description: '祠堂地砖下的铜匣',
    planted_in: '001.md',
    status: 'planted',
    is_long_term: false,
    urgency: { level: 3, remainingChapters: -2, overdueChapters: 2, resolveStatus: 'overdue', mustResolve: false },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [URGENT], currentChapter: 5 })
    vi.mocked(app.SaveForeshadows).mockResolvedValue(undefined)
    vi.mocked(app.GetForeshadowStats).mockResolvedValue({
      total: 1, pending: 0, planted: 1, hinted: 0, resolved: 0,
      partiallyResolved: 0, abandoned: 0, longTermCount: 0,
      overdueCount: 1, currentChapter: 5,
    })
    vi.mocked(app.GetLastForeshadowSync).mockResolvedValue({
      plantedCount: 1, resolvedCount: 2, createdCount: 1,
      updatedIds: ['a'], createdIds: ['b'], matchedByContent: 1,
      skippedResolveCount: 1,
      skippedReasons: [
        { kind: 'invalid_reference', refId: 'gone', message: 'reference_stable_id 无效或已删除' },
        { kind: 'no_match', message: '未找到匹配的已埋入伏笔，跳过回收（不创建新记录）' },
        { kind: 'already_resolved', message: '已在本章回收过' },
      ],
      errors: [],
    })
  })

  it('后端统计行渲染（含待规划口径）+ 超期红 Tag', async () => {
    render(<ForeshadowPanel />)
    expect(await screen.findByText(/埋设 1 · 暗示 0 · 回收 0 · 部分 0 · 待规划 0 · 废弃 0 · 长线 0/)).toBeTruthy()
    expect(screen.getByText('超期 1')).toBeTruthy()
    expect(vi.mocked(app.GetForeshadowStats)).toHaveBeenCalledWith(0)
  })

  it('紧急度 Badge 用后端投影渲染（已超期）', async () => {
    render(<ForeshadowPanel />)
    expect(await screen.findByText('祠堂地砖下的铜匣')).toBeTruthy()
    expect(screen.getByText('已超期')).toBeTruthy()
  })

  it('写回载荷剥离 urgency（A5：运行时投影不落库）', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText('祠堂地砖下的铜匣')

    fireEvent.click(screen.getByRole('button', { name: '标记暗示' }))

    await waitFor(() => expect(vi.mocked(app.SaveForeshadows)).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(app.SaveForeshadows).mock.calls[0][0] as string) as Array<ForeshadowItemData & { urgency?: unknown }>
    expect(payload[0].urgency).toBeUndefined()
  })

  it('上次分析同步：计数与跳过原因直显（D3 不静默）', async () => {
    render(<ForeshadowPanel />)
    expect(await screen.findByText(/上次分析同步：回收 2 · 新埋 1 · 内容匹配 1 · 跳过 1/)).toBeTruthy()
    expect(screen.getByText('· reference_stable_id 无效或已删除')).toBeTruthy()
    expect(screen.getByText(/其余 1 条略/)).toBeTruthy()
  })

  it('清理工具：删除本章伏笔 confirm → 绑定带章节文件名与只删分析开关', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText('祠堂地砖下的铜匣')

    fireEvent.click(screen.getByRole('button', { name: /清理/ }))
    fireEvent.click(screen.getByRole('button', { name: /删除本章伏笔/ }))
    // antd 两字按钮插空格（「删 除」）
    fireEvent.click(await screen.findByRole('button', { name: /^删\s*除$/ }))

    await waitFor(() => expect(vi.mocked(app.DeleteChapterForeshadows)).toHaveBeenCalledTimes(1))
    expect(vi.mocked(app.DeleteChapterForeshadows)).toHaveBeenCalledWith('001.md', true)
  })

  it('清理工具：重分析前清理 → CleanChapterAnalysisForeshadows', async () => {
    render(<ForeshadowPanel />)
    await screen.findByText('祠堂地砖下的铜匣')

    fireEvent.click(screen.getByRole('button', { name: /清理/ }))
    fireEvent.click(screen.getByRole('button', { name: /重分析前清理/ }))
    fireEvent.click(await screen.findByRole('button', { name: /^清\s*理$/ }))

    await waitFor(() => expect(vi.mocked(app.CleanChapterAnalysisForeshadows)).toHaveBeenCalledTimes(1))
    expect(vi.mocked(app.CleanChapterAnalysisForeshadows)).toHaveBeenCalledWith('001.md')
  })
})
