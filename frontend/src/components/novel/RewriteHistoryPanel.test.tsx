// 重写版本历史面板（t4-C3 余项）：列表渲染/状态门控镜像/详情懒拉/动作闭环。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      NovelListRewriteVersions: vi.fn().mockResolvedValue([]),
      NovelGetRewriteVersion: vi.fn().mockResolvedValue({}),
      NovelApplyRewriteVersion: vi.fn().mockResolvedValue({ applied: true }),
      NovelDiscardRewriteVersion: vi.fn().mockResolvedValue(undefined),
      NovelRestoreRewriteVersion: vi.fn().mockResolvedValue({ restored: true }),
    },
  }
})

import RewriteHistoryPanel from './RewriteHistoryPanel'
import { app } from '../../gaea/lib/bridge'

const mkRow = (id: string, mode: string, status: string, similarity = 60) => ({
  id, chapterNum: 2, mode, status, similarity, createdAt: '2026-09-16T00:00:00Z',
})

describe('RewriteHistoryPanel 重写版本历史（t4-C3 余项）', () => {
  const onClose = vi.fn()
  const onApplied = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    onApplied.mockClear()
  })

  it('列表渲染：模式/状态 Tag 与相似度', async () => {
    vi.mocked(app.NovelListRewriteVersions).mockResolvedValue([
      mkRow('v1', 'whole', 'applied', 62.5),
      mkRow('v2', 'partial', 'completed', 81.2),
    ] as never)
    render(<RewriteHistoryPanel open chapterNum={2} onClose={onClose} onApplied={onApplied} />)
    await waitFor(() => expect(screen.getAllByTestId('rewrite-history-row')).toHaveLength(2))
    expect(screen.getByText('整章')).toBeTruthy()
    expect(screen.getByText('局部')).toBeTruthy()
    expect(screen.getByText('已应用')).toBeTruthy()
    expect(screen.getByText('已完成')).toBeTruthy()
    expect(screen.getByText('相似度 63%')).toBeTruthy()
    expect(screen.getByText('相似度 81%')).toBeTruthy()
  })

  it('空态：无版本时给引导文案', async () => {
    vi.mocked(app.NovelListRewriteVersions).mockResolvedValue([] as never)
    render(<RewriteHistoryPanel open chapterNum={2} onClose={onClose} />)
    await waitFor(() => expect(screen.getByText(/该章还没有重写版本/)).toBeTruthy())
  })

  it('详情懒拉：点行一次 GetRewriteVersion，再点收起不重复拉', async () => {
    vi.mocked(app.NovelListRewriteVersions).mockResolvedValue([mkRow('v2', 'whole', 'completed')] as never)
    vi.mocked(app.NovelGetRewriteVersion).mockResolvedValue({
      originalWordCount: 3000, newWordCount: 2880, similarity: 66.6,
      beforeAIScore: 41, afterAIScore: 26,
      originalContent: '原文快照。', newContent: '新文预览。',
    } as never)
    render(<RewriteHistoryPanel open chapterNum={2} onClose={onClose} />)
    await waitFor(() => expect(screen.getAllByTestId('rewrite-history-row')).toHaveLength(1))
    fireEvent.click(screen.getByText('详情'))
    await waitFor(() => expect(screen.getByTestId('rewrite-history-metrics')).toBeTruthy())
    expect(screen.getByText(/原文 3000 字 → 新文 2880 字/)).toBeTruthy()
    expect(screen.getByText(/AI 味分 41 → 26/)).toBeTruthy()
    expect(screen.getByTestId('rewrite-history-new').textContent).toContain('新文预览。')
    expect(app.NovelGetRewriteVersion).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByText('收起'))
    expect(screen.queryByTestId('rewrite-history-metrics')).toBeNull()
  })

  it('状态门控镜像：completed 有应用+放弃；applied 只有恢复；discarded 只有应用', async () => {
    vi.mocked(app.NovelListRewriteVersions).mockResolvedValue([
      mkRow('vc', 'whole', 'completed'),
      mkRow('va', 'whole', 'applied'),
      mkRow('vd', 'whole', 'discarded'),
    ] as never)
    vi.mocked(app.NovelGetRewriteVersion).mockResolvedValue({
      originalWordCount: 10, newWordCount: 12, originalContent: 'o', newContent: 'n',
    } as never)
    render(<RewriteHistoryPanel open chapterNum={2} onClose={onClose} />)
    await waitFor(() => expect(screen.getAllByTestId('rewrite-history-row')).toHaveLength(3))
    // 逐行展开：首行点开后其按钮变「收起」，剩余「详情」依次是下一行
    fireEvent.click(screen.getAllByText('详情')[0])
    fireEvent.click(screen.getAllByText('详情')[0])
    fireEvent.click(screen.getAllByText('详情')[0])
    await waitFor(() => expect(screen.getAllByTestId('rewrite-history-apply')).toHaveLength(2)) // completed+discarded
    expect(screen.getAllByText('放弃版本')).toHaveLength(1) // 仅 completed
    expect(screen.getAllByTestId('rewrite-history-restore')).toHaveLength(1) // 仅 applied
  })

  it('应用闭环：Popconfirm 确认后调用 + onApplied + 列表刷新', async () => {
    vi.mocked(app.NovelListRewriteVersions)
      .mockResolvedValueOnce([mkRow('v2', 'whole', 'completed')] as never)
      .mockResolvedValueOnce([mkRow('v2', 'whole', 'applied')] as never)
    vi.mocked(app.NovelGetRewriteVersion).mockResolvedValue({
      originalWordCount: 10, newWordCount: 12, originalContent: 'o', newContent: 'n',
    } as never)
    render(<RewriteHistoryPanel open chapterNum={2} onClose={onClose} onApplied={onApplied} />)
    await waitFor(() => expect(screen.getAllByTestId('rewrite-history-row')).toHaveLength(1))
    fireEvent.click(screen.getByText('详情'))
    await waitFor(() => expect(screen.getByTestId('rewrite-history-apply')).toBeTruthy())
    fireEvent.click(screen.getByTestId('rewrite-history-apply'))
    // antd 两字中文按钮自动插空格（「确 定」），按 role 匹配
    const okBtn = await screen.findByRole('button', { name: /确\s*定/ }, { timeout: 4000 })
    fireEvent.click(okBtn)
    await waitFor(() => expect(app.NovelApplyRewriteVersion).toHaveBeenCalledWith(2, 'v2'))
    await waitFor(() => expect(onApplied).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(app.NovelListRewriteVersions).toHaveBeenCalledTimes(2))
  })
})
