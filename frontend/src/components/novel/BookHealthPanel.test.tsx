// 全书体检面板（GenerationGate 闭环收口）：聚合卡/最差 AI 味告警/逐章表越线
// 红标/伏笔 findings/空书空态/打开即跑。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  run: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      RunBookHealthCheck: mocks.run,
    },
  }
})

import BookHealthPanel from './BookHealthPanel'

const REPORT = {
  totalChapters: 2,
  chapters: [
    { chapterNum: 1, words: 3200, outlineIssues: 0, outlineErrors: 0, qualityIssues: 0, qualityErrors: 0, aiTasteScore: 12 },
    { chapterNum: 2, words: 4100, outlineIssues: 3, outlineErrors: 1, qualityIssues: 2, qualityErrors: 0, aiTasteScore: 74 },
  ],
  contractIssueChapters: 1,
  qualityIssueChapters: 1,
  worstAiTaste: { chapterNum: 2, words: 4100, outlineIssues: 3, outlineErrors: 1, qualityIssues: 2, qualityErrors: 0, aiTasteScore: 74 },
  analyzedChapters: 1,
  foreshadow: { totalChapters: 2, items: 4, planted: 3, hinted: 1, revealed: 0, longTerm: 2, findings: [{ code: 'target-past', message: '「褪色的信物」目标回收章已过当前进度' }] },
}

describe('BookHealthPanel 全书体检（GenerationGate 闭环）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('打开即跑：聚合卡 + 逐章表 + 伏笔 findings + 最差 AI 味告警', async () => {
    mocks.run.mockResolvedValue(REPORT as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    await waitFor(() => expect(screen.getByTestId('book-health-body')).toBeTruthy())
    expect(mocks.run).toHaveBeenCalledTimes(1)
    // 聚合卡
    expect(screen.getByText('共 2 章')).toBeTruthy()
    expect(screen.getByText('契约问题章 1')).toBeTruthy()
    expect(screen.getByText('质量问题章 1')).toBeTruthy()
    expect(screen.getByText('最差 AI 味 74（第 2 章）')).toBeTruthy()
    expect(screen.getByText('分析覆盖 1/2')).toBeTruthy()
    // 最差 AI 味告警（≥60）
    expect(screen.getByText('第 2 章 AI 味 74 分')).toBeTruthy()
    // 逐章表：契约/质量计数与阻断级标注
    expect(screen.getByText('3 项（1 阻断级）')).toBeTruthy()
    expect(screen.getByText('2 项')).toBeTruthy()
    // 伏笔 findings
    expect(screen.getByText(/褪色的信物/)).toBeTruthy()
  })

  it('重新体检按钮再跑一次', async () => {
    mocks.run.mockResolvedValue(REPORT as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    await waitFor(() => expect(screen.getByTestId('book-health-body')).toBeTruthy())
    fireEvent.click(screen.getByRole('button', { name: /重新体检/ }))
    await waitFor(() => expect(mocks.run).toHaveBeenCalledTimes(2))
  })

  it('零章空书：表格空态不炸', async () => {
    mocks.run.mockResolvedValue({
      totalChapters: 0, chapters: [], contractIssueChapters: 0, qualityIssueChapters: 0, analyzedChapters: 0, foreshadow: { totalChapters: 0, items: 0, planted: 0, hinted: 0, revealed: 0, longTerm: 0, findings: [] },
    } as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    await waitFor(() => expect(screen.getByTestId('book-health-body')).toBeTruthy())
    expect(screen.getByText('共 0 章')).toBeTruthy()
    expect(screen.getByText('还没有已写章节')).toBeTruthy()
  })

  it('失败：错误提示 + 空态', async () => {
    mocks.run.mockRejectedValue(new Error('请先打开项目') as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    await waitFor(() => expect(screen.getByText(/请先打开项目/)).toBeTruthy())
    expect(screen.getByText('没有体检结果')).toBeTruthy()
  })
})
