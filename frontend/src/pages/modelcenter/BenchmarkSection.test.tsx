import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { BenchmarkSection } from './BenchmarkSection'

const { getListMock, getCatalogMock, startMock, exportMock, streamProbeMock } = vi.hoisted(() => ({
  getListMock: vi.fn(),
  getCatalogMock: vi.fn(),
  startMock: vi.fn(),
  exportMock: vi.fn(),
  streamProbeMock: vi.fn(),
}))

vi.mock('../../api/engines', () => ({
  getBenchmarkList: getListMock,
  getHerdsmanCatalog: getCatalogMock,
  startBenchmark: startMock,
  exportBenchmark: exportMock,
  streamProbe: streamProbeMock,
}))

const RUN = {
  id: 'run-abc123',
  created_at: '2026-08-14T07:50:51+08:00',
  finished_at: '2026-08-14T07:51:24+08:00',
  status: 'succeeded',
  model_names: ['Qwen3.6-35B-A3B-HauhauCS-Q4_K_P-2', 'Qwen3.6-35B-A3B-LynnStyle-Q5'],
  variants: ['standard'],
  context_sizes: [4096],
  summary: { total_cases: 2, succeeded: 2, failed: 0, canceled: 0, avg_duration_ms: 304, avg_ttft_ms: 29.5, avg_tps: 65.1 },
}

describe('BenchmarkSection', () => {
  beforeEach(() => {
    getListMock.mockResolvedValue([RUN])
    getCatalogMock.mockResolvedValue({
      models: [{ name: 'qwen3-8b', display_name: 'qwen3-8b', installed: true }],
      total: 1,
      installed: 1,
      running: 0,
      source: 'test',
    })
    startMock.mockResolvedValue('run-new')
    exportMock.mockResolvedValue('C:/tmp/herdsman-benchmark-x.md')
  })

  it('渲染历史运行列表与汇总 KPI', async () => {
    render(<BenchmarkSection />)
    await waitFor(() => expect(screen.getByText(/Qwen3\.6-35B-A3B-HauhauCS-Q4_K_P-2 \+ Qwen3\.6-35B-A3B-LynnStyle-Q5/)).toBeTruthy())
    expect(screen.getByText('succeeded')).toBeTruthy()
    expect(screen.getByText('65.1')).toBeTruthy() // 平均 TPS
    expect(screen.getByText('30 ms')).toBeTruthy() // 平均 TTFT（四舍五入）
  })

  it('未选模型时发起测评给出提示（不调用后端）', async () => {
    render(<BenchmarkSection />)
    await waitFor(() => expect(screen.getByText(/Qwen3\.6-35B-A3B-HauhauCS-Q4_K_P-2/)).toBeTruthy())
    fireEvent.click(screen.getByText('发起受控测评'))
    await waitFor(() => expect(screen.getByText('请至少选择一个已安装模型')).toBeTruthy())
    expect(startMock).not.toHaveBeenCalled()
  })

  it('模型下拉可展开并列出已安装模型', async () => {
    render(<BenchmarkSection />)
    await waitFor(() => expect(screen.getByText(/Qwen3\.6-35B-A3B-HauhauCS-Q4_K_P-2 \+ Qwen3\.6-35B-A3B-LynnStyle-Q5/)).toBeTruthy())
    // 打开模型多选下拉（body 门户），应列出已安装模型
    const selectors = document.querySelectorAll('.ant-select-selector')
    expect(selectors.length).toBeGreaterThan(0)
    fireEvent.mouseDown(selectors[0])
    await waitFor(() => expect(document.querySelector('.ant-select-dropdown')).toBeTruthy())
    const dropdown = document.querySelector('.ant-select-dropdown') as HTMLElement
    expect(within(dropdown).getAllByText('qwen3-8b').length).toBeGreaterThan(0)
  })

  it('渲染快速流式探针区并可触发（D3-4）', async () => {
    render(<BenchmarkSection />)
    await waitFor(() => expect(screen.getByText(/Qwen3\.6-35B-A3B-HauhauCS-Q4_K_P-2/)).toBeTruthy())
    // 流式探针区渲染 + 按钮触发
    expect(screen.getByText('快速流式探针（断流/卡顿观察）')).toBeTruthy()
    const btns = screen.getAllByText('流式探针')
    expect(btns.length).toBeGreaterThan(0)
    streamProbeMock.mockResolvedValue({
      model: 'qwen3-8b', ok: true, ttft_ms: 42, chunks: 3, tokens: 30,
      duration_ms: 500, max_gap_ms: 10, avg_gap_ms: 5, completed: true, interrupted: false,
    })
    fireEvent.click(btns[0])
    await waitFor(() => expect(streamProbeMock).toHaveBeenCalled())
    expect(streamProbeMock.mock.calls[0][0]).toBe('qwen3-8b')
    await waitFor(() => expect(screen.getByText(/TTFT 42ms/)).toBeTruthy())
  })
})
