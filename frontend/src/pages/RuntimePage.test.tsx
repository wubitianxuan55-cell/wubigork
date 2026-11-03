import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import RuntimePage from './RuntimePage'

const { getRuntimeMonitorMock, getModelCallStatsMock } = vi.hoisted(() => ({
  getRuntimeMonitorMock: vi.fn(),
  getModelCallStatsMock: vi.fn(),
}))

vi.mock('../api/engines', () => ({
  getRuntimeMonitor: getRuntimeMonitorMock,
  getModelCallStats: getModelCallStatsMock,
}))

vi.mock('../hooks/usePollingGate', () => ({
  usePollingGate: () => true,
}))

const SNAPSHOT = {
  time: '2026-11-03 22:00:00',
  system: {
    cpu: 23,
    memTotal: 109,
    memUsed: 42,
    gpuName: 'AMD Radeon 8060S',
    gpuUsage: 62,
    vramTotal: 96,
    vramUsed: 12.2,
  },
  comfy_running: true,
  engines: [
    {
      id: 'herdsman',
      name: 'Herdsman 本地',
      type: 'herdsman',
      local: true,
      enabled: true,
      connected: true,
      latency_ms: 12,
      default_model: 'Qwen3.6-MTP-r2',
      models: [
        { id: 'Qwen3.6-35B-MTP-r2', kind: 'llm', status: 'running' },
        { id: 'bge-m3', kind: 'embedding', status: 'running' },
        { id: 'edge-tts', kind: 'tts', status: 'stopped' },
      ],
    },
    {
      id: 'tianxuan',
      name: '天璇 Tianxuan',
      type: 'tianxuan',
      local: true,
      enabled: true,
      connected: true,
      latency_ms: 8,
      default_model: 'qwen3.8-flash-next-uncensored',
      models: [{ id: 'qwen3.8-flash-next-uncensored', kind: 'llm', status: 'running' }],
    },
    {
      id: 'strata',
      name: 'Strata',
      type: 'strata',
      local: true,
      enabled: false,
      connected: false,
      models: [],
    },
    {
      id: 'xai',
      name: 'xAI',
      type: 'xai',
      local: false,
      enabled: true,
      connected: true,
      default_model: 'grok-4.7',
      models: [],
    },
  ],
  tianxuan: {
    live: {
      state: 'generating',
      queued: 0,
      tok_s: 65.2,
      elapsed_s: 3.4,
      generated: 120,
    },
    engine: { model: 'qwen3.8-flash-next-uncensored', max_context: 4096 },
    hardware: {
      cpu: 31,
      ram_used: 45.2,
      ram_total: 109,
      gpu_util: null,
      gpu_temp: null,
      disk_read_mb: 12.5,
      disk_write_mb: 3.2,
    },
    requests: [
      {
        time: 1763400000,
        duration_s: 5.4,
        finish: 'stop',
        prompt_tokens: 1520,
        prompt_total: 1600,
        prompt_read: 1520,
        reused: 1024,
        output_tokens: 340,
        prompt_ms: 1400,
        decode_tok_s: 65.2,
        hit_rate: 0.63,
      },
      {
        time: 1763399000,
        duration_s: 1.2,
        finish: 'error',
        prompt_tokens: 80,
        output_tokens: 0,
      },
    ],
    conversation_cache: {
      enabled: true,
      budget_mib: 4096,
      slots: 4,
      requests: 12,
      requests_reused: 8,
      reused_tokens: 8192,
      prompt_tokens: 16384,
    },
  },
  strata: null,
  recent_calls: [
    {
      engine_id: 'tianxuan',
      model: 'qwen3.8-flash-next-uncensored',
      feature: 'gaea',
      input_tokens: 1520,
      output_tokens: 340,
      cache_hit_tokens: 1024,
      duration_ms: 5400,
      success: true,
      finished_at: '2026-11-03 21:59:55',
    },
    {
      engine_id: 'xai',
      model: 'grok-4.7',
      feature: 'chat',
      input_tokens: 300,
      output_tokens: 0,
      duration_ms: 800,
      success: false,
      error_message: 'HTTP 429',
      finished_at: '2026-11-03 21:58:10',
    },
    {
      engine_id: 'herdsman',
      model: 'Qwen3.6-35B-MTP-r2',
      feature: 'novel',
      input_tokens: 900,
      output_tokens: 500,
      duration_ms: 2100,
      success: true,
      finished_at: '2026-11-03 21:57:40',
    },
  ],
  totals: {
    total_calls: 1200,
    success_calls: 1190,
    fail_calls: 10,
    total_tokens: 2400000,
    input_tokens: 1900000,
    output_tokens: 500000,
    total_cost: 3.45,
  },
}

const CALL_STATS = {
  total_calls: 1200,
  success_calls: 1190,
  fail_calls: 10,
  total_tokens: 2400000,
  input_tokens: 1900000,
  output_tokens: 500000,
  total_duration_ms: 1000,
  avg_duration_ms: 1,
  total_cost: 3.45,
  trend: [
    {
      time: '2026-11-01 08:00:00',
      calls: 12,
      success_calls: 12,
      fail_calls: 0,
      input_tokens: 8000,
      output_tokens: 2000,
      total_tokens: 10000,
      cost: 0.1,
    },
    {
      time: '2026-11-02 09:00:00',
      calls: 30,
      success_calls: 28,
      fail_calls: 2,
      input_tokens: 20000,
      output_tokens: 5000,
      total_tokens: 25000,
      cost: 0.4,
    },
    {
      time: '2026-11-03 20:00:00',
      calls: 45,
      success_calls: 45,
      fail_calls: 0,
      input_tokens: 30000,
      output_tokens: 9000,
      total_tokens: 39000,
      cost: 0.6,
    },
  ],
  per_model: [],
}

const PER_MODEL_STATS = [
  {
    engine_id: 'tianxuan',
    model: 'qwen3.8-flash-next-uncensored',
    call_count: 80,
    success_count: 80,
    fail_count: 0,
    input_tokens: 80000,
    output_tokens: 20000,
    total_tokens: 100000,
    total_duration_ms: 240000,
    estimated_cost: 0,
    currency: 'CNY',
  },
  {
    engine_id: 'herdsman',
    model: 'Qwen3.6-35B-MTP-r2',
    call_count: 20,
    success_count: 20,
    fail_count: 0,
    input_tokens: 4000,
    output_tokens: 1000,
    total_tokens: 5000,
    total_duration_ms: 44000,
    estimated_cost: 0,
    currency: 'CNY',
  },
  {
    engine_id: 'xai',
    model: 'grok-4.7',
    call_count: 20,
    success_count: 16,
    fail_count: 4,
    input_tokens: 6000,
    output_tokens: 2000,
    total_tokens: 8000,
    total_duration_ms: 50000,
    estimated_cost: 0.5,
    currency: 'USD',
    last_error: 'HTTP 429',
    last_called_at: '2026-11-03 21:58:10',
  },
]

describe('RuntimePage 运行监测', () => {
  beforeEach(() => {
    getRuntimeMonitorMock.mockReset()
    getModelCallStatsMock.mockReset()
    getModelCallStatsMock.mockResolvedValue({ ...CALL_STATS, trend: [] })
  })

  it('首拍显示监测与统计加载态', () => {
    getRuntimeMonitorMock.mockImplementation(() => new Promise(() => {}))
    getModelCallStatsMock.mockImplementation(() => new Promise(() => {}))
    render(<RuntimePage />)

    expect(screen.getByText('正在读取监测数据…')).toBeTruthy()
    expect(screen.getByText('等待首轮同步')).toBeTruthy()
    expect(screen.getByLabelText('正在读取调用统计')).toBeTruthy()
  })

  it('顶部健康摘要呈现推理状态、在线服务与累计成功率', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)

    await waitFor(() => expect(screen.getAllByText('生成中').length).toBeGreaterThanOrEqual(1))
    expect(screen.getByText(/累计 1,200 次调用，成功率 99\.2%/)).toBeTruthy()
    expect(screen.getByText('3 / 3')).toBeTruthy()
    expect(screen.getAllByText('65.2 tok/s').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('本轮 3.4s')).toBeTruthy()
  })

  it('调用统计以所选范围为口径呈现 KPI、趋势与质量分析', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    getModelCallStatsMock.mockResolvedValue(CALL_STATS)
    render(<RuntimePage />)

    await waitFor(() => expect(screen.getByTestId('metric-calls')).toBeTruthy())
    expect(within(screen.getByTestId('metric-calls')).getByText('87')).toBeTruthy()
    expect(within(screen.getByTestId('metric-success')).getByText('97.7%')).toBeTruthy()
    expect(within(screen.getByTestId('metric-tokens')).getByText('74.0K')).toBeTruthy()
    expect(within(screen.getByTestId('metric-cost')).getByText('¥1.10')).toBeTruthy()

    expect(screen.getByRole('img', { name: '调用趋势图' })).toBeTruthy()
    expect(screen.getByRole('img', { name: 'Token 用量趋势图' })).toBeTruthy()
    expect(screen.getByText('峰值').parentElement?.textContent).toContain('45')
    expect(screen.getByText('调用结果')).toBeTruthy()
    expect(screen.getByText('近期请求耗时')).toBeTruthy()
    expect(screen.getByText('近期错误')).toBeTruthy()
    expect(screen.getAllByText('HTTP 429').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('查看图表明细数据')).toBeTruthy()
  })

  it('今日、7 天、30 天切换会重算 KPI 与图表', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    const hourBucket = (daysAgo: number, hour: number) => {
      const date = new Date()
      date.setDate(date.getDate() - daysAgo)
      const pad = (value: number) => String(value).padStart(2, '0')
      return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(hour)}:00`
    }
    getModelCallStatsMock.mockResolvedValue({
      ...CALL_STATS,
      trend: [
        { time: hourBucket(0, 8), calls: 12, success_calls: 12, fail_calls: 0, input_tokens: 8000, output_tokens: 2000, total_tokens: 10000, cost: 0.1 },
        { time: hourBucket(1, 9), calls: 30, success_calls: 28, fail_calls: 2, input_tokens: 20000, output_tokens: 5000, total_tokens: 25000, cost: 0.4 },
        { time: hourBucket(2, 10), calls: 45, success_calls: 45, fail_calls: 0, input_tokens: 30000, output_tokens: 9000, total_tokens: 39000, cost: 0.6 },
      ],
    })
    render(<RuntimePage />)

    await waitFor(() => expect(within(screen.getByTestId('metric-calls')).getByText('87')).toBeTruthy())
    fireEvent.click(screen.getByRole('radio', { name: '今日' }))
    await waitFor(() => expect(within(screen.getByTestId('metric-calls')).getByText('12')).toBeTruthy())
    expect(within(screen.getByTestId('metric-cost')).getByText('¥0.10')).toBeTruthy()

    fireEvent.click(screen.getByRole('radio', { name: '最近 30 天' }))
    await waitFor(() => expect(within(screen.getByTestId('metric-calls')).getByText('87')).toBeTruthy())
  })

  it('模型明细支持搜索、范围筛选与异常排序信息', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    getModelCallStatsMock.mockResolvedValue({ ...CALL_STATS, per_model: PER_MODEL_STATS })
    render(<RuntimePage />)

    const modelSection = screen.getByText('模型调用明细').closest('section')
    expect(modelSection).toBeTruthy()
    await waitFor(() => expect(within(modelSection!).getByText('grok-4.7')).toBeTruthy())
    expect(within(modelSection!).getByText('80.0%')).toBeTruthy()
    expect(within(modelSection!).getByText('¥3.60')).toBeTruthy()
    expect(screen.getByRole('radio', { name: '需关注' })).toBeTruthy()

    const search = within(modelSection!).getByPlaceholderText('搜索模型或引擎')
    fireEvent.change(search, { target: { value: 'grok' } })
    await waitFor(() => expect(within(modelSection!).queryByText('qwen3.8-flash-next-uncensored')).toBeNull())
    expect(within(modelSection!).getByText('grok-4.7')).toBeTruthy()

    fireEvent.change(search, { target: { value: '' } })
    fireEvent.click(within(modelSection!).getByRole('radio', { name: '本地' }))
    await waitFor(() => expect(within(modelSection!).queryByText('grok-4.7')).toBeNull())
    expect(within(modelSection!).getByText('qwen3.8-flash-next-uncensored')).toBeTruthy()
  })

  it('模型明细空态与统计加载态可区分', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)

    await waitFor(() => expect(screen.getByText(/暂无模型调用记录/)).toBeTruthy())
  })

  it('最近请求可切换为失败请求，并保留错误原因', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)

    await waitFor(() => expect(screen.getByText('最近请求')).toBeTruthy())
    const section = screen.getByText('最近请求').closest('section')
    expect(section).toBeTruthy()
    expect(within(section!).getByText('HTTP 429')).toBeTruthy()

    fireEvent.click(within(section!).getByRole('radio', { name: '失败 1' }))
    await waitFor(() => expect(within(section!).queryByText('Qwen3.6-35B-MTP-r2')).toBeNull())
    expect(within(section!).getByText('grok-4.7')).toBeTruthy()
    expect(within(section!).getByText('HTTP 429')).toBeTruthy()
  })

  it('运行状态呈现上下文、缓存、资源与引擎遥测', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)

    await waitFor(() => expect(screen.getByText('运行状态')).toBeTruthy())
    expect(screen.getByRole('img', { name: '上下文用量' })).toBeTruthy()
    expect(screen.getByText('1.9K / 4.1K Token')).toBeTruthy()
    expect(screen.getByText('8 / 12')).toBeTruthy()
    expect(screen.getAllByText('Prompt 复用 50%').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText(/12\.5 \/ 3\.2 MB\/s/)).toBeTruthy()
    expect(screen.getByText('ComfyUI')).toBeTruthy()
  })

  it('引擎侧请求对齐 Prompt、Prefill、Decode 与缓存口径', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)

    await waitFor(() => expect(screen.getByText('引擎侧请求')).toBeTruthy())
    expect(screen.getByText('1086 t/s')).toBeTruthy()
    expect(screen.getByText('65.2 t/s')).toBeTruthy()
    expect(screen.getByText('63%')).toBeTruthy()
    expect(screen.getByText('复用 1.0K')).toBeTruthy()
    expect(screen.getAllByText('正常').length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText('错误').length).toBeGreaterThanOrEqual(1)
  })

  it('统计失败提供范围就近的重试入口，成功后恢复图表', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    getModelCallStatsMock.mockRejectedValueOnce(new Error('统计服务不可用')).mockResolvedValue(CALL_STATS)
    render(<RuntimePage />)

    await waitFor(() => expect(screen.getByText(/调用统计加载失败：统计服务不可用/)).toBeTruthy())
    fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }))
    await waitFor(() => expect(screen.getByTestId('metric-calls')).toBeTruthy())
  })

  it('空快照保持诚实空态', async () => {
    getRuntimeMonitorMock.mockResolvedValue({
      time: '',
      system: {},
      engines: [],
      strata: null,
      tianxuan: null,
      recent_calls: [],
      totals: {},
    })
    render(<RuntimePage />)

    await waitFor(() => expect(screen.getAllByText('无引擎在线').length).toBeGreaterThanOrEqual(1))
    expect(screen.getByText('暂无调用记录')).toBeTruthy()
    expect(screen.getByText('引擎暂无已完成请求')).toBeTruthy()
  })

  it('实时快照刷新失败时保留已有数据并提示恢复入口', async () => {
    getRuntimeMonitorMock.mockResolvedValueOnce(SNAPSHOT).mockRejectedValueOnce(new Error('boom'))
    render(<RuntimePage />)

    await waitFor(() => expect(screen.getAllByText('生成中').length).toBeGreaterThanOrEqual(1))
    fireEvent.click(screen.getByRole('button', { name: /刷新全部/ }))
    await waitFor(() => expect(screen.getByText(/实时快照刷新失败：boom/)).toBeTruthy())
    expect(screen.getAllByText('生成中').length).toBeGreaterThanOrEqual(1)
  })
})
