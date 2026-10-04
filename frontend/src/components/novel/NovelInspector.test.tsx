// NovelInspector.test.tsx — 属性检查器「章节体检」区（RunChapterGate 合并报告展示）
// 覆盖 v4.282（oh-story 蒸馏 T3）：确定性两路（写前大纲契约 outlineContract /
// 写后硬信号 deterministic）与 AI 四路并列直显；四路缺省仍按「未启用」诚实降级。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({ RunChapterGate: vi.fn() }))
vi.mock('../../../wailsjs/go/app/NovelB', () => ({ RunChapterGate: mocks.RunChapterGate }))

import NovelInspector from './NovelInspector'
import { useAppStore } from '../../stores/appStore'

/** 派发阅读页上报的当前章节属性（NovelInspector 订阅 novel:chapter-active）。 */
function emitChapterActive(detail: Record<string, unknown>) {
  window.dispatchEvent(new CustomEvent('novel:chapter-active', { detail }))
}

describe('NovelInspector 章节体检（确定性两路 + AI 四路）', () => {
  beforeEach(() => {
    mocks.RunChapterGate.mockReset()
    useAppStore.setState({ projectTitle: '评审测试', projectPath: 'C:/novel/test' })
  })

  it('无当前章：体检按钮不出现（只提示从大纲选章）', () => {
    render(<NovelInspector activeTab="chapter" collapsed={false} onToggleCollapse={vi.fn()} onNavigate={vi.fn()} stats={null} />)
    expect(screen.getByText(/从左侧大纲选择章节后/)).toBeTruthy()
    expect(screen.queryByRole('button', { name: /运行|重跑/ })).toBeNull()
  })

  it('体检报告：写前契约/写后硬信号计数与按严重度排序的条目直显', async () => {
    mocks.RunChapterGate.mockResolvedValue({
      chapterNum: 2,
      outlineContract: [
        { code: 'outline_keypoints_empty', severity: 'S3', message: '关键要点为空：建议列 2-6 条本章必须落地的点' },
        { code: 'outline_summary_empty', severity: 'S2', message: '本章计划（summary）为空' },
      ],
      deterministic: [
        { code: 'period_heavy', severity: 'S3', message: '标点以句号为主（句号 40 / 逗号 6）' },
      ],
      analysis: null,
      review: { score: 7, weaknesses: ['对话偏说明文'] },
      consistency: { total_issues: 0 },
      aiTaste: { score: 42 },
    })
    render(<NovelInspector activeTab="chapter" collapsed={false} onToggleCollapse={vi.fn()} onNavigate={vi.fn()} stats={null} />)
    emitChapterActive({ id: 'n2', chapterNum: 2, title: '第二章', words: 2300, saved: true })

    fireEvent.click(await screen.findByRole('button', { name: /运行/ }))
    await waitFor(() => expect(mocks.RunChapterGate).toHaveBeenCalledWith(2))

    expect(await screen.findByText(/写前契约：/)).toBeTruthy()
    expect(screen.getByText('2 项待补')).toBeTruthy()
    expect(screen.getByText(/写后硬信号：/)).toBeTruthy()
    expect(screen.getByText('1 项')).toBeTruthy()
    // 条目按严重度排序：S2 契约项先于 S3；两路各自取前两条。
    expect(screen.getByText('计划 [S2] 本章计划（summary）为空')).toBeTruthy()
    expect(screen.getByText('正文 [S3] 标点以句号为主（句号 40 / 逗号 6）')).toBeTruthy()
    // AI 四路仍在（章节质量分与 AI 味分）。
    expect(screen.getByText('7')).toBeTruthy()
    expect(screen.getByText('42')).toBeTruthy()
  })

  it('确定性两路缺省（老后端）：显示「齐备 / 未命中」而非报错', async () => {
    mocks.RunChapterGate.mockResolvedValue({ chapterNum: 3, analysis: null, review: null, consistency: null, aiTaste: null })
    render(<NovelInspector activeTab="chapter" collapsed={false} onToggleCollapse={vi.fn()} onNavigate={vi.fn()} stats={null} />)
    emitChapterActive({ id: 'n3', chapterNum: 3, title: '第三章', words: 2100, saved: true })
    fireEvent.click(await screen.findByRole('button', { name: /运行/ }))
    expect(await screen.findByText('写前契约：齐备')).toBeTruthy()
    expect(screen.getByText('写后硬信号：未命中')).toBeTruthy()
  })

  // v4.425 B12 反向守卫：在途体检响应不得挂到已切换的章上。
  it('体检在途换章：旧响应被丢弃，不把第 2 章的报告挂到第 5 章', async () => {
    let resolveGate: (v: unknown) => void = () => {}
    mocks.RunChapterGate.mockImplementationOnce(
      () => new Promise((r) => { resolveGate = r }) as never,
    )
    render(<NovelInspector activeTab="chapter" collapsed={false} onToggleCollapse={vi.fn()} onNavigate={vi.fn()} stats={null} />)
    emitChapterActive({ id: 'n2', chapterNum: 2, title: '第二章', words: 2300, saved: true })
    fireEvent.click(await screen.findByRole('button', { name: /运行/ }))
    await waitFor(() => expect(mocks.RunChapterGate).toHaveBeenCalledWith(2))

    // 第 2 章的响应还在途中时用户切到第 5 章
    emitChapterActive({ id: 'n5', chapterNum: 5, title: '第五章', words: 2600, saved: true })
    await waitFor(() => expect(screen.getByText('第五章')).toBeTruthy())

    await act(async () => {
      resolveGate({ chapterNum: 2, analysis: null, review: { score: 7, weaknesses: ['对话偏说明文'] }, consistency: { total_issues: 0 }, aiTaste: { score: 42 } })
      await Promise.resolve()
    })

    // 报告区不得出现第 2 章的分数（gate 仍为空态）
    expect(screen.queryByText('写前契约：齐备')).toBeNull()
    expect(screen.queryByText('42')).toBeNull()
    expect(screen.getByText(/合并分析\/审查\/一致性\/AI 味四路/)).toBeTruthy()
  })
})
