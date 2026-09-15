// PartialRewriteModal.test.tsx — 选段局部重写弹窗关键路径（t4-C3 余项前端消费刀）
// 覆盖：提交载荷形状（rune 偏移 + selected_text + length_mode，custom 带 target_word_count）；
// 结果统计与新选段渲染；应用逐参（Apply + onApplied + onClose）；指令空禁用提交。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      NovelChapterRewrite: vi.fn().mockResolvedValue({
        versionId: 'rw-3-2', status: 'completed', similarity: 72.5,
        change: 20, changePercent: 3.1, originalWordCount: 3000,
        newWordCount: 3020, newContent: '前文。崭新的选段内容。后文。',
        mode: 'partial', selectedWordCount: 6, newSelectedWordCount: 7,
        lengthMode: 'similar', startPos: 3, endPos: 9,
      }),
      NovelApplyRewriteVersion: vi.fn().mockResolvedValue({ applied: true }),
      NovelDiscardRewriteVersion: vi.fn().mockResolvedValue(undefined),
    },
  }
})

import PartialRewriteModal from './PartialRewriteModal'
import { app } from '../../gaea/lib/bridge'

describe('PartialRewriteModal 选段局部重写（t4-C3 余项）', () => {
  const onApplied = vi.fn()
  const onClose = vi.fn()
  const selection = { start: 2, end: 4, text: 'b文' }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('提交载荷形状：mode/source + rune 偏移 + selected_text + length_mode，custom 带 target_word_count', async () => {
    render(<PartialRewriteModal open chapterNum={3} selection={selection} onClose={onClose} onApplied={onApplied} />)
    fireEvent.change(screen.getByPlaceholderText(/把这段对话改得更锋利/), { target: { value: '更口语化' } })
    fireEvent.click(screen.getByRole('radio', { name: '自定义' }))
    fireEvent.change(screen.getByRole('spinbutton'), { target: { value: '800' } })
    fireEvent.click(screen.getByTestId('partial-rewrite-submit'))

    await waitFor(() => expect(app.NovelChapterRewrite).toHaveBeenCalledTimes(1))
    const [num, reqJSON] = vi.mocked(app.NovelChapterRewrite).mock.calls[0] as [number, string]
    expect(num).toBe(3)
    const req = JSON.parse(reqJSON)
    expect(req.mode).toBe('partial')
    expect(req.source).toBe('custom')
    expect(req.custom_instructions).toBe('更口语化')
    expect(req.start_pos).toBe(2)
    expect(req.end_pos).toBe(4)
    expect(req.selected_text).toBe('b文')
    expect(req.length_mode).toBe('custom')
    expect(req.target_word_count).toBe(800)
  })

  it('结果视图：统计行与新选段（按 startPos + newSelectedWordCount 从全文切出）渲染', async () => {
    render(<PartialRewriteModal open chapterNum={3} selection={selection} onClose={onClose} onApplied={onApplied} />)
    fireEvent.change(screen.getByPlaceholderText(/把这段对话改得更锋利/), { target: { value: '收紧节奏' } })
    fireEvent.click(screen.getByTestId('partial-rewrite-submit'))

    expect(await screen.findByTestId('partial-rewrite-result')).toBeTruthy()
    expect(screen.getByText(/相似度 72\.5%/)).toBeTruthy()
    expect(screen.getByText('选段 6 字 → 新 7 字')).toBeTruthy()
    expect(screen.getByText('崭新的选段内容')).toBeTruthy()
  })

  it('应用并写回：逐参调用 Apply 且触发 onApplied + onClose', async () => {
    render(<PartialRewriteModal open chapterNum={3} selection={selection} onClose={onClose} onApplied={onApplied} />)
    fireEvent.change(screen.getByPlaceholderText(/把这段对话改得更锋利/), { target: { value: '收紧节奏' } })
    fireEvent.click(screen.getByTestId('partial-rewrite-submit'))
    fireEvent.click(await screen.findByTestId('partial-rewrite-apply'))

    await waitFor(() => expect(app.NovelApplyRewriteVersion).toHaveBeenCalledWith(3, 'rw-3-2'))
    await waitFor(() => expect(onApplied).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
  })

  it('指令为空：提交按钮禁用且不发起请求', () => {
    render(<PartialRewriteModal open chapterNum={3} selection={selection} onClose={onClose} onApplied={onApplied} />)
    const submit = screen.getByTestId('partial-rewrite-submit')
    expect(submit).toHaveProperty('disabled', true)
    fireEvent.click(submit)
    expect(app.NovelChapterRewrite).not.toHaveBeenCalled()
  })
})
