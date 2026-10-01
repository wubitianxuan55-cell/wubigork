// 全书体检面板（GenerationGate 闭环收口）：聚合卡/最差 AI 味告警/逐章表越线
// 红标/伏笔 findings/空书空态/打开即跑。
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  run: vi.fn(),
  v2: vi.fn(),
  evalSnap: vi.fn(),
  evalBase: vi.fn(),
  evalCmp: vi.fn(),
  evalList: vi.fn(),
  evalDelete: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      RunBookHealthCheck: mocks.run,
      NovelChapterAnalysisV2: mocks.v2,
      NovelEvalSnapshot: mocks.evalSnap,
      NovelEvalBaselineSet: mocks.evalBase,
      NovelEvalCompare: mocks.evalCmp,
      NovelEvalSnapshotsList: mocks.evalList,
      NovelEvalSnapshotDelete: mocks.evalDelete,
    },
  }
})

import BookHealthPanel from './BookHealthPanel'
import { useAppStore } from '../../stores/appStore'
import { Modal, message as antdMessage } from 'antd'

afterEach(() => {
  Modal.destroyAll()
  antdMessage.destroy()
})

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
    // 坑复训：vi.clearAllMocks() 不清 mockImplementation——跨用例必须复位，
    // 否则「失败的实现」会泄漏给后续用例（本文件先前已踩过一次集体假红）。
    mocks.run.mockReset()
    mocks.run.mockResolvedValue(REPORT as never)
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

  it('失败：错误横幅（不是「没有体检结果」空态）+ 重试按钮可再跑', async () => {
    // 第一次失败、重试成功（mockReset 已在 beforeEach 统一做，此处只排队这一次的实现）
    mocks.run.mockRejectedValueOnce(new Error('请先打开项目') as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    await waitFor(() => expect(screen.getByTestId('book-health-error')).toBeTruthy())
    // B10 反向守卫：失败必须与「确实没有结果」区分开，且给出重试入口
    expect(screen.queryByText('没有体检结果')).toBeNull()
    expect(screen.getByText('全书体检失败')).toBeTruthy()
    // 原子串同时出现在「失败原因」与 toast 文案里 → 用 getAllByText，不用 getByText
    expect(screen.getAllByText(/请先打开项目/).length).toBeGreaterThan(0)

    // 弹窗 footer 另有「重新体检」——这里用更精确的名字定位横幅按钮
    fireEvent.click(screen.getByRole('button', { name: /^\s*重\s*试\s*$/ }))
    await waitFor(() => expect(screen.getByTestId('book-health-body')).toBeTruthy())
    expect(mocks.run).toHaveBeenCalledTimes(2)
  })

  it('后端返回 null（确实没有结果）：走空态而非错误横幅', async () => {
    mocks.run.mockResolvedValue(null as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    await waitFor(() => expect(screen.getByText('没有体检结果')).toBeTruthy())
    expect(screen.queryByTestId('book-health-error')).toBeNull()
  })

  it('切换项目：清掉上一本的报告态（不额外发请求）', async () => {
    useAppStore.setState({ projectPath: 'C:/novel/book-a' })
    mocks.run.mockResolvedValue(REPORT as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    await waitFor(() => expect(screen.getByTestId('book-health-body')).toBeTruthy())
    const callsBefore = mocks.run.mock.calls.length

    act(() => { useAppStore.setState({ projectPath: 'C:/novel/book-b' }) })

    // 报告属上一本 → 清空；且只是清态，不触发新请求（重开弹窗时才重拉）
    await waitFor(() => expect(screen.queryByTestId('book-health-body')).toBeNull())
    expect(screen.queryByText('最差 AI 味 74（第 2 章）')).toBeNull()
    expect(mocks.run.mock.calls.length).toBe(callsBefore)
  })
})

// 情感曲线（t7 观察池）：按需逐章拉分析 V2 情感弧线，SVG 折线；未分析章跳过计数。
describe('BookHealthPanel 情感曲线', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.run.mockResolvedValue(REPORT as never)
  })

  it('默认不拉 V2；点「生成曲线」逐章读取并渲染折线点与 tooltip', async () => {
    mocks.v2.mockImplementation(async (num: number) => ({
      chapter_num: num,
      result: { emotional_arc: { primary_emotion: num === 1 ? '警觉' : '释然', intensity: num === 1 ? 7 : 3 } },
    }) as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    await waitFor(() => expect(screen.getByTestId('book-health-body')).toBeTruthy())
    // 未点按钮前零 V2 调用
    expect(mocks.v2).not.toHaveBeenCalled()
    fireEvent.click(screen.getByTestId('health-curve-run'))
    expect(await screen.findByTestId('health-curve-svg')).toBeTruthy()
    expect(mocks.v2).toHaveBeenCalledTimes(2) // 逐章读取
    const pts = screen.getAllByTestId('health-curve-point')
    expect(pts).toHaveLength(2)
    expect(pts[0].querySelector('title')?.textContent).toContain('第 1 章 · 警觉（强度 7）')
    expect(pts[1].querySelector('title')?.textContent).toContain('第 2 章 · 释然（强度 3）')
  })

  it('两章有数据：两个折线点', async () => {
    mocks.v2.mockImplementation(async (num: number) => ({
      chapter_num: num,
      result: { emotional_arc: { primary_emotion: num === 1 ? '警觉' : '释然', intensity: num === 1 ? 7 : 3 } },
    }) as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    await waitFor(() => expect(screen.getByTestId('book-health-body')).toBeTruthy())
    fireEvent.click(screen.getByTestId('health-curve-run'))
    await waitFor(() => expect(screen.getAllByTestId('health-curve-point')).toHaveLength(2))
  })

  it('可绘制章不足（1 章有弧线 + 1 章跳过）给不足提示并计跳过数', async () => {
    mocks.v2.mockImplementation(async (num: number) => {
      if (num === 1) {
        return { chapter_num: 1, result: { emotional_arc: { primary_emotion: '警觉', intensity: 7 } } } as never
      }
      throw new Error('尚未分析')
    })
    render(<BookHealthPanel open onClose={vi.fn()} />)
    await waitFor(() => expect(screen.getByTestId('book-health-body')).toBeTruthy())
    fireEvent.click(screen.getByTestId('health-curve-run'))
    expect(await screen.findByText(/可绘制的章不足/)).toBeTruthy()
    expect(screen.getByText(/1 章跳过/)).toBeTruthy()
    expect(screen.queryByTestId('health-curve-svg')).toBeNull()
  })
})

// ── v4.429 观察池#3：情感曲线在途循环的代际守卫 ──
// 旧缺陷：buildCurve 逐章 await 无 token——重跑体检/切书后旧循环完成时仍
// setCurve，把上一本书/上一轮的点写进新上下文。
describe('BookHealthPanel 情感曲线代际守卫（观察池#3）', () => {
  it('曲线生成在途重跑体检：迟到的旧轮结果不落（svg 不出现）', async () => {
    vi.clearAllMocks()
    mocks.run.mockResolvedValue(REPORT as never)
    useAppStore.setState({ projectPath: 'C:/novel/book-a' })
    // 第一章的 V2 挂起（在途）；旧轮循环停在第一章 await 上
    let resolveV2!: (v: unknown) => void
    mocks.v2.mockImplementation((num: number) => {
      if (num === 1) return new Promise((r) => { resolveV2 = r }) as never
      return Promise.resolve({
        chapter_num: num,
        result: { emotional_arc: { primary_emotion: '释然', intensity: 3 } },
      }) as never
    })
    render(<BookHealthPanel open onClose={vi.fn()} />)
    await waitFor(() => expect(screen.getByTestId('book-health-body')).toBeTruthy())
    fireEvent.click(screen.getByTestId('health-curve-run'))
    await waitFor(() => expect(mocks.v2).toHaveBeenCalled())

    // 重跑体检：作废在途循环（seq++）。report 会被新结果替换（非空）——旧轮
    // 迟到结果若未被守卫丢弃，svg 会渲染出来（这正是反向验证能红的前提：
    // 切书场景 report 清空后断言恒真，测不出守卫缺失）。
    fireEvent.click(screen.getByRole('button', { name: /重新体检/ }))
    await waitFor(() => expect(mocks.run).toHaveBeenCalledTimes(2))

    // 旧轮的迟到结果到达——不得渲染出曲线
    resolveV2({ chapter_num: 1, result: { emotional_arc: { primary_emotion: '警觉', intensity: 9 } } })
    await act(async () => { await new Promise((r) => setTimeout(r, 50)) })
    expect(screen.queryByTestId('health-curve-svg')).toBeNull()
  })
})

describe('BookHealthPanel 评测基线区（v4.446）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.run.mockReset()
    mocks.run.mockResolvedValue(REPORT as never)
    mocks.evalSnap.mockReset()
    mocks.evalBase.mockReset()
    mocks.evalCmp.mockReset()
    mocks.evalList.mockReset()
  })

  it('生成快照：确定性指标行渲染（含张力块覆盖>0 才显示）', async () => {
    mocks.evalSnap.mockResolvedValue({
      chapters: 2, chars: 7300,
      taste: { mean: 43, max: 74, p90: 74, worstChapter: 2 },
      quality: { s1: 0, s2: 1, s3: 2 },
      foreshadow: { items: 4, recall: 0, findings: 1 },
      tension: { mean: 5.5, p90: 10, swing: 8, covered: 2 },
      contextRunes: 620,
    } as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    fireEvent.click(await screen.findByTestId('health-eval-snap'))
    expect(await screen.findByText('AI 味均值 43（P90 74）')).toBeTruthy()
    expect(screen.getByText(/张力均值 5.5 · 极差 8（覆盖 2 章）$/)).toBeTruthy()
    expect(screen.getByText(/伏笔回收率 0$/)).toBeTruthy()
  })

  it('与基线对比 stale：横幅如实提示且不出对比表（旧基线不可比）', async () => {
    mocks.evalCmp.mockResolvedValue({
      stale: true,
      items: [{ metric: 'AI 味均值', from: 30, to: 45, delta: 15, dir: 'worse' }],
    } as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    fireEvent.click(await screen.findByTestId('health-eval-cmp'))
    expect(await screen.findByText(/prompts 已变更，基线过期/)).toBeTruthy()
    expect(screen.queryByText('AI 味均值')).toBeNull()
  })

  it('对比非 stale：方向着色（恶化红/改善绿）', async () => {
    mocks.evalCmp.mockResolvedValue({
      stale: false,
      items: [
        { metric: 'AI 味均值', from: 30, to: 45, delta: 15, dir: 'worse' },
        { metric: '伏笔回收率', from: 0, to: 0.8, delta: 0.8, dir: 'better' },
      ],
    } as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    fireEvent.click(await screen.findByTestId('health-eval-cmp'))
    expect(await screen.findByText(/恶化/)).toBeTruthy()
    expect(screen.getByText(/改善/)).toBeTruthy()
  })
})

describe('BookHealthPanel 历史快照（v4.447）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.run.mockReset()
    mocks.run.mockResolvedValue(REPORT as never)
    mocks.evalSnap.mockReset()
    mocks.evalBase.mockReset()
    mocks.evalCmp.mockReset()
    mocks.evalList.mockReset()
    mocks.evalDelete.mockReset()
  })

  it('生成快照走 persist 落盘并回显 savedAs；历史表按新→旧渲染、坏档行不出现', async () => {
    mocks.evalSnap.mockResolvedValue({
      chapters: 30, chars: 15000,
      taste: { mean: 40, p90: 74 },
      quality: { s1: 0, s2: 1, s3: 2 },
      foreshadow: { recall: 0.8 },
      tension: { mean: 5.5, swing: 8, covered: 30 },
      savedAs: '20261002T120000.000Z.json',
    } as never)
    mocks.evalList.mockResolvedValue([
      { name: '20261002T120000.000Z.json', chapters: 30, tasteMean: 40, s1: 0, s2: 1, s3: 2, recall: 0.8, tensionMean: 5.5, tensionCover: 30 },
      { name: '20261002T110000.000Z.json', chapters: 29, tasteMean: 52, s1: 1, s2: 0, s3: 0, recall: 0.4, tensionMean: 3, tensionCover: 20 },
    ] as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    fireEvent.click(await screen.findByTestId('health-eval-snap'))
    expect(await screen.findByText(/已存 20261002T120000/)).toBeTruthy()
    // persist 语义：生成快照必须请求落盘（persist=true）
    expect(mocks.evalSnap).toHaveBeenCalledWith(true)

    fireEvent.click(screen.getByText('查看历史快照'))
    expect(await screen.findByText('20261002T120000.000Z.json')).toBeTruthy()
    expect(screen.getByText('20261002T110000.000Z.json')).toBeTruthy()
    expect(screen.getByText(/2 份（eval\/snapshots\/，新→旧）/)).toBeTruthy()
  })

  it('无历史：空态提示指向生成入口', async () => {
    mocks.evalList.mockResolvedValue([] as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    fireEvent.click(await screen.findByText('查看历史快照'))
    expect(await screen.findByText(/还没有持久化快照/)).toBeTruthy()
  })
})

describe('BookHealthPanel 历史快照删除（v4.448）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.run.mockReset()
    mocks.run.mockResolvedValue(REPORT as never)
    mocks.evalSnap.mockReset()
    mocks.evalBase.mockReset()
    mocks.evalCmp.mockReset()
    mocks.evalList.mockReset()
    mocks.evalDelete.mockReset()
  })

  it('Popconfirm 确认后调删除绑定并刷新索引；取消不删', async () => {
    mocks.evalList.mockResolvedValue([
      { name: '20261002T120000.000Z.json', chapters: 30, tasteMean: 40, s1: 0, s2: 1, s3: 2, recall: 0.8, tensionMean: 5.5, tensionCover: 30 },
    ] as never)
    mocks.evalDelete.mockResolvedValue(undefined as never)
    render(<BookHealthPanel open onClose={vi.fn()} />)
    fireEvent.click(await screen.findByText('查看历史快照'))
    const del = await screen.findByRole('button', { name: '删除快照 20261002T120000.000Z.json' })
    fireEvent.click(del)
    // Popconfirm 取消：不删
    fireEvent.click(await screen.findByText(/取\s*消/))
    expect(mocks.evalDelete).not.toHaveBeenCalled()
    // Popconfirm 确认：删 + 刷新索引
    fireEvent.click(del)
    const confirmBtn = screen
      .getAllByText(/删\s*除/)
      .map((el) => el.closest('button'))
      .find((b) => b && b.closest('.ant-popover'))
    fireEvent.click(confirmBtn!)
    await waitFor(() => expect(mocks.evalDelete).toHaveBeenCalledWith('20261002T120000.000Z.json'))
    await waitFor(() => expect(mocks.evalList).toHaveBeenCalledTimes(2))
  })
})
