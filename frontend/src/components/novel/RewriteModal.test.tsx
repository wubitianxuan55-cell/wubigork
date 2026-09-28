// RewriteModal.test.tsx — 整章重写弹窗关键路径（t4-C3 前端消费刀）
// 覆盖：建议加载渲染；建议驱动提交参数逐参对齐；结果统计渲染；
// 应用逐参（写回正文回调触发）；应用后恢复入口。
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { Modal } from 'antd'

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

  afterEach(async () => {
    Modal.destroyAll()
    await new Promise((r) => setTimeout(r, 0))
  })

  // v4.421.0：运行中关闭策略——允许关闭但先确认（该调用没有取消接口，不能假装能取消；
  // 结果进版本库，「重写历史」可查看）。旧实现点 ✕ 直接关闭且无任何说明。
  it('运行中点关闭先确认：取消则不关闭，确认才关闭', async () => {
    // mockImplementationOnce：一次性挂起（重写只调一次），避免污染后续用例的实现
    vi.mocked(app.NovelChapterRewrite).mockImplementationOnce(() => new Promise(() => {}) as never)
    const close = vi.fn()
    render(<RewriteModal open chapterNum={2} onClose={close} onApplied={onApplied} />)
    await screen.findByText(/该章暂无分析结果/)
    fireEvent.change(screen.getByPlaceholderText(/收紧中段节奏/), { target: { value: '收紧节奏' } })
    fireEvent.click(screen.getByRole('button', { name: /开始重写/ }))
    await screen.findByTestId('rewrite-running')

    fireEvent.click(document.querySelector('.ant-modal-close') as Element)
    await waitFor(() => expect(document.querySelectorAll('.ant-modal-confirm').length).toBeGreaterThan(0))
    const all = document.querySelectorAll<HTMLElement>('.ant-modal-confirm')
    const scoped = within(all[all.length - 1])
    // antd 标题父子双匹配：用 getAllByText 断言存在
    expect(scoped.getAllByText(/重写仍在进行/).length).toBeGreaterThan(0)
    fireEvent.click(scoped.getByRole('button', { name: /取\s*消/ }))
    await new Promise((r) => setTimeout(r, 0))
    expect(close).not.toHaveBeenCalled()

    // 同一个确认弹窗仍在：直接确认关闭
    fireEvent.click(scoped.getByRole('button', { name: /关闭窗口/ }))
    await waitFor(() => expect(close).toHaveBeenCalledTimes(1))
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
