// RewriteModal.test.tsx — 整章重写弹窗关键路径（t4-C3 前端消费刀）
// 覆盖：建议加载渲染；建议驱动提交参数逐参对齐；结果统计渲染；
// 应用逐参（写回正文回调触发）；应用后恢复入口。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      NovelChapterSuggestions: vi.fn().mockResolvedValue([]),
      NovelChapterRewrite: vi.fn().mockResolvedValue({
        versionId: 'rw-2-1', status: 'completed', similarity: 66.6,
        change: -120, changePercent: -4.2, originalWordCount: 3000,
        newWordCount: 2880, newContent: '重写后的正文全文。',
      }),
      NovelApplyRewriteVersion: vi.fn().mockResolvedValue({ applied: true }),
      NovelDiscardRewriteVersion: vi.fn().mockResolvedValue(undefined),
      NovelRestoreRewriteVersion: vi.fn().mockResolvedValue({ restored: true }),
    },
  }
})

import RewriteModal from './RewriteModal'
import { app } from '../../gaea/lib/bridge'

describe('RewriteModal 整章重写（t4-C3）', () => {
  const onApplied = vi.fn()
  const onClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('打开时加载该章分析建议并渲染勾选清单', async () => {
    vi.mocked(app.NovelChapterSuggestions).mockResolvedValue(['节奏拖沓', '情感不足'])
    render(<RewriteModal open chapterNum={2} onClose={onClose} onApplied={onApplied} />)

    expect(await screen.findByText('2. 情感不足')).toBeTruthy()
    expect(screen.getByText('1. 节奏拖沓')).toBeTruthy()
    expect(app.NovelChapterSuggestions).toHaveBeenCalledWith(2)
  })

  it('无分析结果：提示先分析，自定义指令可驱动重写（参数逐参对齐）', async () => {
    render(<RewriteModal open chapterNum={2} onClose={onClose} onApplied={onApplied} />)
    await screen.findByText(/该章暂无分析结果/)

    fireEvent.change(screen.getByPlaceholderText(/收紧中段节奏/), { target: { value: '收紧节奏' } })
    fireEvent.click(screen.getByRole('button', { name: /开始重写/ }))

    await waitFor(() => expect(app.NovelChapterRewrite).toHaveBeenCalledTimes(1))
    const [num, reqJSON] = vi.mocked(app.NovelChapterRewrite).mock.calls[0] as [number, string]
    expect(num).toBe(2)
    const req = JSON.parse(reqJSON)
    expect(req.source).toBe('custom')
    expect(req.custom_instructions).toBe('收紧节奏')
  })

  it('结果视图：统计与新全文渲染；应用逐参 + 回调刷新', async () => {
    render(<RewriteModal open chapterNum={2} onClose={onClose} onApplied={onApplied} />)
    fireEvent.change(screen.getByPlaceholderText(/收紧中段节奏/), { target: { value: '收紧节奏' } })
    fireEvent.click(screen.getByRole('button', { name: /开始重写/ }))

    expect(await screen.findByTestId('rewrite-result')).toBeTruthy()
    expect(screen.getByText(/相似度 66\.6%/)).toBeTruthy()
    expect(screen.getByText('3000 → 2880 字（-120）')).toBeTruthy()
    expect(screen.getByText('重写后的正文全文。')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: /应用并写回正文/ }))
    await waitFor(() => expect(app.NovelApplyRewriteVersion).toHaveBeenCalledWith(2, 'rw-2-1'))
    await waitFor(() => expect(onApplied).toHaveBeenCalled())
    expect(await screen.findByText('已应用')).toBeTruthy()
  })

  it('应用后出现恢复原文入口并逐参调用', async () => {
    render(<RewriteModal open chapterNum={2} onClose={onClose} onApplied={onApplied} />)
    fireEvent.change(screen.getByPlaceholderText(/收紧中段节奏/), { target: { value: '收紧节奏' } })
    fireEvent.click(screen.getByRole('button', { name: /开始重写/ }))
    fireEvent.click(await screen.findByRole('button', { name: /应用并写回正文/ }))

    const restore = await screen.findByRole('button', { name: /恢复原文/ })
    fireEvent.click(restore)
    await waitFor(() => expect(app.NovelRestoreRewriteVersion).toHaveBeenCalledWith(2, 'rw-2-1'))
    await waitFor(() => expect(onApplied).toHaveBeenCalledTimes(2))
  })

  it('放弃版本：逐参调用并关闭弹窗', async () => {
    render(<RewriteModal open chapterNum={2} onClose={onClose} onApplied={onApplied} />)
    fireEvent.change(screen.getByPlaceholderText(/收紧中段节奏/), { target: { value: 'x' } })
    fireEvent.click(screen.getByRole('button', { name: /开始重写/ }))
    fireEvent.click(await screen.findByRole('button', { name: /放弃版本/ }))

    await waitFor(() => expect(app.NovelDiscardRewriteVersion).toHaveBeenCalledWith(2, 'rw-2-1'))
    await waitFor(() => expect(onClose).toHaveBeenCalled())
  })
})
