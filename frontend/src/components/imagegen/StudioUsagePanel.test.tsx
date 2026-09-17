// 画室消耗折叠面板（绘梦阶段一刀 E）：标题聚合（张数+合计）/按模型行/
// 免费与未定价注记/空态/失败静默。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  usage: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      ImageHubMonthlyUsage: mocks.usage,
    },
  }
})

import StudioUsagePanel from './StudioUsagePanel'

const USAGE = {
  month: '2026-09',
  space: 'play',
  total: 9,
  freeCount: 7,
  unpriced: 1,
  estimated: '0.36 CNY（另有 1 张未定价）',
  byModel: [
    { model: 'krea2', backend: 'comfyui', count: 7, unitCost: '0', estCost: '0 CNY' },
    { model: 'qwen-image-3.0-pro', backend: 'openai', count: 2, unitCost: '0.18 CNY/张', estCost: '0.36 CNY' },
  ],
}

describe('StudioUsagePanel 画室消耗（刀 E）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('挂载即拉 play 空间聚合；标题带总数与合计', async () => {
    mocks.usage.mockResolvedValue(USAGE as never)
    render(<StudioUsagePanel />)
    await waitFor(() => expect(mocks.usage).toHaveBeenCalledWith('play'))
    await waitFor(() => expect(screen.getByTestId('studio-usage-title').textContent).toContain('9 张'))
    expect(screen.getByTestId('studio-usage-title').textContent).toContain('0.36 CNY')
  })

  it('展开：按模型行（张数/估算/未定价如实）+口径脚注', async () => {
    mocks.usage.mockResolvedValue(USAGE as never)
    render(<StudioUsagePanel />)
    await waitFor(() => expect(screen.getByTestId('studio-usage-title').textContent).toContain('9 张'))
    fireEvent.click(screen.getByTestId('studio-usage-title'))
    await waitFor(() => expect(screen.getByTestId('studio-usage-body')).toBeTruthy())
    expect(screen.getByText('krea2')).toBeTruthy()
    expect(screen.getByText('qwen-image-3.0-pro')).toBeTruthy()
    expect(screen.getAllByText('7 张').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('0.36 CNY')).toBeTruthy()
    expect(screen.getAllByText(/另有 1 张未定价/).length).toBeGreaterThanOrEqual(1) // 标题+脚注两处
  })

  it('空月：标题零张 + 展开引导文案', async () => {
    mocks.usage.mockResolvedValue({ ...USAGE, total: 0, estimated: '本月还没有创作记录', byModel: [], unpriced: 0 } as never)
    render(<StudioUsagePanel />)
    await waitFor(() => expect(screen.getByTestId('studio-usage-title').textContent).toContain('0 张'))
    fireEvent.click(screen.getByTestId('studio-usage-title'))
    await waitFor(() => expect(screen.getByText('本月还没有创作记录')).toBeTruthy())
  })

  it('失败静默收起（标题不带数据）', async () => {
    mocks.usage.mockRejectedValue(new Error('x') as never)
    render(<StudioUsagePanel />)
    await waitFor(() => expect(mocks.usage).toHaveBeenCalled())
    expect(screen.getByTestId('studio-usage-title').textContent).not.toContain('张')
  })
})
