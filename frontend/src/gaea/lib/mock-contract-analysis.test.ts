// t7 分析接线契约层 mock 冒烟：AnalyzeChapter（出 Legacy 转正）+ 
// NovelChapterAnalysisV2 两名（mock 实现见 mock/novel.ts；真实后端见 Go NovelB）。
// 锁定 mock 形状与 ChapterAnalysisPanel 消费字段的对应（先例：mock-contract-prompt.test.ts）。
import { describe, expect, it } from 'vitest'
import { app } from './bridge'

const analysisApi = app as unknown as {
  AnalyzeChapter(chapterNum: number): Promise<Record<string, unknown>>
  NovelChapterAnalysisV2(chapterNum: number): Promise<{
    chapter_num: number
    result: {
      hooks: unknown[]
      foreshadows: unknown[]
      conflict: Record<string, unknown>
      emotional_arc: Record<string, unknown>
      character_states: unknown[]
      organization_states: unknown[]
      plot_points: unknown[]
      scenes: unknown[]
      pacing: string
      scores: { overall: number }
      suggestions: string[]
      summary: string
    }
  }>
}

describe('mock 契约 · 分析接线两名（t7：分析 V2 面板）', () => {
  it('两名均存在于 mock 绑定面', () => {
    for (const n of ['AnalyzeChapter', 'NovelChapterAnalysisV2']) {
      expect(typeof (app as unknown as Record<string, unknown>)[n], n).toBe('function')
    }
  })

  it('V2 返回九维齐全（面板各分节的数据源）', async () => {
    const d = await analysisApi.NovelChapterAnalysisV2(3)
    expect(d.chapter_num).toBe(3)
    const r = d.result
    for (const k of ['hooks', 'foreshadows', 'conflict', 'emotional_arc', 'character_states', 'organization_states', 'plot_points', 'scenes', 'pacing', 'scores', 'suggestions', 'summary'] as const) {
      expect(r[k], k).toBeDefined()
    }
    expect(typeof r.scores.overall).toBe('number')
    expect(Array.isArray(r.suggestions)).toBe(true)
  })

  it('AnalyzeChapter 回 V1 形状（面板忽略返回、真相源=V2 落盘）', async () => {
    const r = await analysisApi.AnalyzeChapter(3)
    for (const k of ['hook', 'conflict', 'qualityScore', 'improvementTips'] as const) {
      expect(k in r, k).toBe(true)
    }
  })
})
