// 章节分析 V2 面板（t7 前端接线收官刀）：九维渲染/缺档空态+分析闭环/标注列表
// 与高亮视图（rune→code-unit 换算+交叠裁剪+锚点定位）/无章空态。
// 直调 app.* 的 mock 手法同 PromptWorkshopPanel.test（bridge 桩 + saveFile/pickFile
// 无关；这里只桩 bridge）。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  v2: vi.fn(),
  annotations: vi.fn().mockResolvedValue([]),
  analyze: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      NovelChapterAnalysisV2: mocks.v2,
      NovelChapterAnnotations: mocks.annotations,
      AnalyzeChapter: mocks.analyze,
    },
  }
})

import ChapterAnalysisPanel from './ChapterAnalysisPanel'
import type { ChapterAnalysisV2View } from '../../gaea/lib/bridge/novel'

const mkV2 = (): ChapterAnalysisV2View => ({
  chapter_num: 2,
  analyzed_at: '2026-09-16T10:00:00Z',
  engine: 'herdsman',
  model: 'test-model',
  result: {
    hooks: [{ type: '悬念', content: '夜半敲门声', strength: 8, position: '开头' }],
    foreshadows: [{ title: '褪色的信物', content: '袖中信物纹路一致。', type: 'planted', strength: 7, is_long_term: true, estimated_resolve_chapter: 14 }],
    conflict: { types: ['人与人'], level: 7, description: '进门与守门的拉锯。', resolution_progress: 0.4 },
    emotional_arc: { primary_emotion: '警觉', intensity: 7, curve: '缓起-陡升' },
    character_states: [{ name: '林昭', old_state: '旁观', new_state: '入局', psychological_change: '直面' }],
    organization_states: [{ org_name: '守门会', power_value: 60, member_changes: [{ character_name: '林昭', change_type: 'joined' }] }],
    plot_points: [{ content: '信物现世。', type: 'revelation', importance: 0.8 }],
    scenes: [{ location: '旧宅门廊', atmosphere: '湿冷', duration: '一刻钟' }],
    pacing: 'moderate',
    dialogue_ratio: 0.32,
    description_ratio: 0.41,
    scores: { pacing: 7.5, engagement: 8, coherence: 8, overall: 7.8, score_justification: '开篇扎实。' },
    plot_stage: '第一幕',
    suggestions: ['中段可收紧'],
    summary: '信物将主角推入局中。',
  },
})

const open = (chapterNum: number | null, content = '') =>
  render(<ChapterAnalysisPanel open onClose={vi.fn()} chapterNum={chapterNum} content={content} />)

describe('ChapterAnalysisPanel 章节分析 V2（t7 收官）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.annotations.mockResolvedValue([])
  })

  it('九维渲染：综合分/三维 Tag/chips/各分节/建议/小结 + meta', async () => {
    mocks.v2.mockResolvedValue(mkV2() as never)
    open(2)
    await waitFor(() => expect(screen.getByTestId('analysis-body')).toBeTruthy())
    expect(screen.getByText('7.8')).toBeTruthy()
    expect(screen.getByText('节奏 7.5')).toBeTruthy()
    expect(screen.getByText(/开篇扎实/)).toBeTruthy()
    expect(screen.getByText('节奏 适中')).toBeTruthy()
    expect(screen.getByText(/对话 32%/)).toBeTruthy() // chips 之一
    expect(screen.getByText(/夜半敲门声/)).toBeTruthy()
    expect(screen.getByText(/褪色的信物/)).toBeTruthy()
    expect(screen.getByText(/预计第 14 章/)).toBeTruthy()
    expect(screen.getAllByText(/林昭/).length).toBeGreaterThanOrEqual(2) // 角色变化 + 组织成员变更
    expect(screen.getByText(/守门会/)).toBeTruthy()
    expect(screen.getByText(/中段可收紧/)).toBeTruthy()
    expect(screen.getByText(/信物将主角推入局中/)).toBeTruthy()
    expect(screen.getByText(/herdsman/)).toBeTruthy()
    // 打开只拉一次 V2 + 标注
    expect(mocks.v2).toHaveBeenCalledTimes(1)
    expect(mocks.annotations).toHaveBeenCalledWith(2)
  })

  it('缺档空态：尚未分析引导 + 分析本章按钮闭环（AnalyzeChapter → 重拉 V2+标注）', async () => {
    mocks.v2.mockRejectedValueOnce(new Error('第 2 章尚未分析（先运行「分析本章」）') as never)
      .mockResolvedValueOnce(mkV2() as never)
    open(2)
    await waitFor(() => expect(screen.getByTestId('analysis-empty')).toBeTruthy())
    const btn = screen.getByTestId('analysis-run-btn')
    mocks.analyze.mockResolvedValueOnce({} as never)
    fireEvent.click(btn)
    await waitFor(() => expect(mocks.analyze).toHaveBeenCalledWith(2))
    await waitFor(() => expect(mocks.v2).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(screen.getByTestId('analysis-body')).toBeTruthy())
  })

  it('标注列表与高亮视图：rune 换算+交叠裁剪+锚点切换', async () => {
    mocks.v2.mockResolvedValue(mkV2() as never)
    // 正文 10 rune；两段交叠（pos 2/len 3 与 pos 3/len 5 → 第二段起点钳 5）+ 一条未命中（pos -1）
    mocks.annotations.mockResolvedValue([
      { type: 'hook', content: '第一段', pos: 2, length: 3 },
      { type: 'foreshadow', content: '第二段', pos: 3, length: 5 },
      { type: 'suggestion', content: '未命中不渲染', pos: -1 },
    ] as never)
    const content = '零一二三四五六七八九'
    open(2, content)
    await waitFor(() => expect(screen.getByTestId('analysis-annotations')).toBeTruthy())
    // 列表三行（未命中也在列表）
    expect(screen.getAllByTestId('analysis-ann-row')).toHaveLength(3)
    expect(screen.getByText('未命中不渲染')).toBeTruthy()
    // 切高亮视图：两段 mark（交叠裁剪后），尾段文本完整
    fireEvent.click(screen.getByText('高亮视图'))
    const marks = screen.getAllByTestId('analysis-highlight')
    expect(marks).toHaveLength(2)
    expect(marks[0].textContent).toBe(content.slice(2, 5)) // 二三四
    expect(marks[1].textContent).toBe(content.slice(5, 8)) // 五六七（起点钳 5）
  })

  it('列表行点击：切高亮视图并滚动锚点（requestAnimationFrame 桩）', async () => {
    mocks.v2.mockResolvedValue(mkV2() as never)
    mocks.annotations.mockResolvedValue([
      { type: 'conflict', content: '冲突段', pos: 1, length: 2 },
    ] as never)
    open(2, '零一二三')
    await waitFor(() => expect(screen.getByTestId('analysis-ann-row')).toBeTruthy())
    const scrollIntoView = vi.fn()
    Element.prototype.scrollIntoView = scrollIntoView
    fireEvent.click(screen.getByTestId('analysis-ann-row'))
    await waitFor(() => expect(scrollIntoView).toHaveBeenCalled())
    expect(screen.getByTestId('analysis-highlight-view')).toBeTruthy()
  })

  it('无激活章：空态引导，不触发拉取', () => {
    open(null)
    expect(screen.getByText('先在左侧选择一章')).toBeTruthy()
    expect(mocks.v2).not.toHaveBeenCalled()
  })
})

describe('标注定位编辑器光标（t7 观察池：onLocate 入口）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.v2.mockResolvedValue(mkV2() as never)
    mocks.annotations.mockResolvedValue([
      { type: 'conflict', content: '冲突段', pos: 1, length: 2 },
      { type: 'suggestion', content: '未命中不锚定', pos: -1 },
    ] as never)
  })

  it('提供 onLocate 且锚定：行内出「编辑器定位」，点击回传标注且不触发行内高亮跳转', async () => {
    const onLocate = vi.fn()
    render(<ChapterAnalysisPanel open onClose={vi.fn()} chapterNum={2} content="零一二三" onLocate={onLocate} />)
    await waitFor(() => expect(screen.getAllByTestId('analysis-ann-row')).toHaveLength(2))
    fireEvent.click(screen.getByTestId('analysis-ann-locate'))
    expect(onLocate).toHaveBeenCalledTimes(1)
    expect(onLocate).toHaveBeenCalledWith(expect.objectContaining({ pos: 1, length: 2 }))
    // stopPropagation：行点击副作用（切高亮视图）不触发
    expect(screen.queryByTestId('analysis-highlight-view')).toBeNull()
  })

  it('仅未锚定标注（pos=-1）时定位入口不出现', async () => {
    mocks.annotations.mockResolvedValue([{ type: 'suggestion', content: '未命中不锚定', pos: -1 }] as never)
    render(<ChapterAnalysisPanel open onClose={vi.fn()} chapterNum={2} content="零一二三" onLocate={vi.fn()} />)
    await waitFor(() => expect(screen.getByTestId('analysis-ann-row')).toBeTruthy())
    expect(screen.queryByTestId('analysis-ann-locate')).toBeNull()
  })

  it('未提供 onLocate：入口整体隐藏（gate 跳他章形态）', async () => {
    render(<ChapterAnalysisPanel open onClose={vi.fn()} chapterNum={2} content="零一二三" />)
    await waitFor(() => expect(screen.getAllByTestId('analysis-ann-row')).toHaveLength(2))
    expect(screen.queryByTestId('analysis-ann-locate')).toBeNull()
  })
})

describe('章际对比（t7 多章对比最小形态）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.v2.mockResolvedValue(mkV2() as never)
    mocks.annotations.mockResolvedValue([])
  })

  it('未提供 chapterOptions：对比区不渲染', async () => {
    open(2, '零一二三')
    await waitFor(() => expect(screen.getByTestId('analysis-body')).toBeTruthy())
    expect(screen.queryByTestId('analysis-compare')).toBeNull()
  })

  it('提供 chapterOptions：下拉排除本章；选对比章拉其 V2 渲染差值表', async () => {
    const cmp = mkV2()
    cmp.result!.scores!.overall = 6.0
    mocks.v2.mockImplementation(async (num: number) => {
      if (num === 3) return cmp as never
      return mkV2() as never
    })
    render(<ChapterAnalysisPanel open onClose={vi.fn()} chapterNum={2} content="零一二三" chapterOptions={[1, 2, 3, 4]} />)
    await waitFor(() => expect(screen.getByTestId('analysis-body')).toBeTruthy())
    fireEvent.mouseDown(screen.getByRole('combobox'))
    // 本章（2）被排除；选项含 1/3/4（dropdown 渲染在 portal，等它出现）
    expect(await screen.findByTitle('第 3 章')).toBeTruthy()
    expect(screen.queryByTitle('第 2 章')).toBeNull()
    fireEvent.click(screen.getByTitle('第 3 章'))
    // 差值表：本章 7.8 vs 对比章 6.0 → Δ -1.8
    expect(await screen.findByTestId('analysis-compare-table')).toBeTruthy()
    expect(screen.getByText('6.0')).toBeTruthy()
    expect(screen.getByText('-1.8')).toBeTruthy()
    expect(mocks.v2).toHaveBeenCalledWith(3)
  })

  it('对比章尚未分析：行内提示不报 toast', async () => {
    mocks.v2.mockImplementation(async (num: number) => {
      if (num === 3) throw new Error('第 3 章尚未分析')
      return mkV2() as never
    })
    render(<ChapterAnalysisPanel open onClose={vi.fn()} chapterNum={2} content="零一二三" chapterOptions={[3]} />)
    await waitFor(() => expect(screen.getByTestId('analysis-body')).toBeTruthy())
    fireEvent.mouseDown(screen.getByRole('combobox'))
    fireEvent.click(await screen.findByTitle('第 3 章'))
    expect(await screen.findByText(/尚未分析——先在章节页运行分析后再对比/)).toBeTruthy()
    expect(screen.queryByTestId('analysis-compare-table')).toBeNull()
  })
})
