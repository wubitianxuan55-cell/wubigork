import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// StorySpinePanel 经 gaea/lib/bridge 的 app 调用刀3 绑定（NovelB 门面）。
const mocks = vi.hoisted(() => ({
  SpineGet: vi.fn(),
  SpineSave: vi.fn(),
  StoryHealth: vi.fn(),
  SpinePropose: vi.fn(),
}))
vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: new Proxy(
      {
        NovelStorySpineGet: mocks.SpineGet,
        NovelStorySpineSave: mocks.SpineSave,
        NovelStoryHealth: mocks.StoryHealth,
        NovelStorySpinePropose: mocks.SpinePropose,
      },
      {
        get(target, prop) {
          if (prop in target) return Reflect.get(target, prop)
          return (actual.app as unknown as Record<string, unknown>)[String(prop)]
        },
      },
    ),
  }
})

import StorySpinePanel from './StorySpinePanel'

describe('StorySpinePanel 故事骨架（长篇刀3）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.SpineGet.mockResolvedValue({})
    mocks.SpineSave.mockResolvedValue(undefined)
  })

  it('打开即拉取；空骨架给引导态', async () => {
    render(<StorySpinePanel open onClose={vi.fn()} />)
    expect(await screen.findByTestId('spine-body')).toBeTruthy()
    expect(mocks.SpineGet).toHaveBeenCalledTimes(1)
    expect(screen.getByText(/还没有骨架/)).toBeTruthy()
  })

  it('编辑主题+加支线后保存：payload 走整表 JSON', async () => {
    render(<StorySpinePanel open onClose={vi.fn()} />)
    await screen.findByTestId('spine-body')
    fireEvent.change(screen.getByTestId('spine-idea'), { target: { value: '自由以责任为价' } })
    fireEvent.click(screen.getByTestId('spine-add-thread'))
    fireEvent.click(screen.getByTestId('spine-save'))
    await waitFor(() => expect(mocks.SpineSave).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(mocks.SpineSave.mock.calls[0][0] as string)
    expect(payload.theme.controlling_idea).toBe('自由以责任为价')
    expect(payload.threads).toHaveLength(1)
    expect(payload.threads[0].status).toBe('active')
  })

  it('AI 提炼：提案只填表单不落盘（Save 零调用）', async () => {
    mocks.SpinePropose.mockResolvedValue({
      controlling_idea: '自由以责任为价', counter_idea: '秩序才是善',
      arcs: [{ name: '林昭', want: '逃离', need: '承担', misbelief: '离开即自由', first_beat: '' }],
      beats: [{ id: 'midpoint', name: '中点', chapter: 10 }],
      threads: [{ name: '母亲的下落', mice_type: 'query', open_chapter: 2, status: 'active', note: '' }],
      questions: [{ question: '母亲去了哪里？', raised_chapter: 2, status: 'open' }],
    })
    render(<StorySpinePanel open onClose={vi.fn()} />)
    await screen.findByTestId('spine-body')
    fireEvent.click(screen.getByTestId('spine-propose'))
    await waitFor(() => expect(screen.getByDisplayValue('自由以责任为价')).toBeTruthy())
    expect((screen.getByDisplayValue('母亲的下落') as HTMLInputElement).value).toBe('母亲的下落')
    // 提案未落盘：保存零调用；提示走「未落盘」口径
    expect(mocks.SpineSave).not.toHaveBeenCalled()
    expect(await screen.findByText(/未落盘/)).toBeTruthy()
  })

  it('结构体检：findings 渲染严重度与定位', async () => {
    mocks.StoryHealth.mockResolvedValue({
      current_chapter: 16,
      findings: [
        { code: 'thread_stale', severity: 'S1', ref: '母亲的下落', message: '支线已 13 章未推进' },
        { code: 'theme_missing', severity: 'S3', ref: 'theme', message: '控制理念为空' },
      ],
    })
    render(<StorySpinePanel open onClose={vi.fn()} />)
    await screen.findByTestId('spine-body')
    fireEvent.click(screen.getByTestId('spine-check'))
    const report = await screen.findByTestId('spine-report')
    expect(report.textContent).toContain('当前第 16 章')
    expect(report.textContent).toContain('支线已 13 章未推进')
    expect(screen.getByText('S1')).toBeTruthy()
  })

  it('保存失败（校验拒绝）如实报错不清表单', async () => {
    mocks.SpineSave.mockRejectedValue(new Error('故事骨架保存失败: 支线「x」状态非法 "bogus"'))
    render(<StorySpinePanel open onClose={vi.fn()} />)
    await screen.findByTestId('spine-body')
    fireEvent.click(screen.getByTestId('spine-save'))
    expect(await screen.findByText(/状态非法/)).toBeTruthy()
    expect(screen.getByTestId('spine-body')).toBeTruthy()
  })
})
