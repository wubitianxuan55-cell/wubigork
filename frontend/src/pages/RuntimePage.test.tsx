import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import RuntimePage from './RuntimePage'

const { getRuntimeMonitorMock } = vi.hoisted(() => ({
  getRuntimeMonitorMock: vi.fn(),
}))

vi.mock('../api/engines', () => ({
  getRuntimeMonitor: getRuntimeMonitorMock,
}))

// 轮询门默认开（jsdom visibilityState=visible）；测试不推进假时钟，靠挂载
// 首拍渲染断言。
vi.mock('../hooks/usePollingGate', () => ({
  usePollingGate: () => true,
}))

const SNAPSHOT = {
  time: '2026-10-19 20:00:00',
  system: { cpu: 23, memTotal: 109, memUsed: 42, gpuName: '', vramTotal: 0, vramUsed: 0 },
  engines: [
    {
      id: 'herdsman', name: 'Herdsman 本地', type: 'herdsman', local: true, enabled: true,
      connected: true, latency_ms: 12, default_model: 'Qwen3.6-MTP-r2',
      models: [
        { id: 'Qwen3.6-35B-MTP-r2', kind: 'llm', status: 'running' },
        { id: 'bge-m3', kind: 'embedding', status: 'running' },
        { id: 'edge-tts', kind: 'tts', status: 'stopped' },
      ],
    },
    {
      id: 'tianxuan', name: '天璇 Tianxuan', type: 'tianxuan', local: true, enabled: true,
      connected: true, latency_ms: 8, default_model: 'qwen3.8-flash-next-uncensored',
      models: [{ id: 'qwen3.8-flash-next-uncensored', kind: 'llm', status: 'running' }],
    },
    { id: 'strata', name: 'Strata', type: 'strata', local: true, enabled: false, connected: false, models: [] },
    { id: 'xai', name: 'xAI', type: 'xai', local: false, enabled: true, connected: true, default_model: 'grok-4.7', models: [] },
  ],
  tianxuan: {
    live: { state: 'generating', queued: 0, tok_s: 65.2, elapsed_s: 3.4, generated: 120 },
    engine: { model: 'qwen3.8-flash-next-uncensored' },
    hardware: { cpu: 31, ram_used: 45.2, ram_total: 109, gpu_util: null, gpu_temp: null, disk_read_mb: 12.5 },
  },
  strata: null,
  recent_calls: [
    {
      engine_id: 'tianxuan', model: 'qwen3.8-flash-next-uncensored', feature: 'gaea',
      input_tokens: 1520, output_tokens: 340, cache_hit_tokens: 1024,
      duration_ms: 5400, success: true, finished_at: '2026-10-19 19:59:55',
    },
    {
      engine_id: 'xai', model: 'grok-4.7', feature: 'chat',
      input_tokens: 300, output_tokens: 0, duration_ms: 800, success: false,
      error_message: 'HTTP 429', finished_at: '2026-10-19 19:58:10',
    },
  ],
  totals: { total_calls: 1200, success_calls: 1190, fail_calls: 10, total_tokens: 2400000, total_cost: 3.45 },
}

describe('RuntimePage 运行监测板块', () => {
  beforeEach(() => {
    // mockReset（非 clearAllMocks）：清实现 + Once 队列，防上个用例的持久实现串台
    getRuntimeMonitorMock.mockReset()
  })

  it('快照到达前显示加载态', () => {
    getRuntimeMonitorMock.mockImplementation(() => new Promise(() => {}))
    render(<RuntimePage />)
    expect(screen.getByText('正在读取监测数据…')).toBeTruthy()
    expect(screen.getByText('加载中')).toBeTruthy()
  })

  it('状态条：引擎生成中 + tok/s + 累计口径', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getAllByText(/生成中/).length).toBeGreaterThanOrEqual(1))
    expect(screen.getByText(/65\.2 tok\/s/)).toBeTruthy()
    expect(screen.getByText(/1,200 次调用/)).toBeTruthy()
    expect(screen.getByText(/2,400,000 tokens/)).toBeTruthy()
    expect(screen.getByText(/¥3\.45/)).toBeTruthy()
  })

  it('指标卡：遥测有值显示数值，GPU 无遥测诚实留空', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('生成速度')).toBeTruthy())
    expect(screen.getByText('65.2')).toBeTruthy() // tok/s（引擎 live）
    expect(screen.getByText('31')).toBeTruthy() // CPU（引擎遥测优先）
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(3) // GPU 负载/温度/功率 + 显存（AMD 无遥测）
  })

  it('引擎卡：本地三引擎 + 云端聚合 + 生成中状态徽标', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('Herdsman 本地')).toBeTruthy())
    expect(screen.getByText('天璇 Tianxuan')).toBeTruthy()
    expect(screen.getByText('Strata')).toBeTruthy()
    expect(screen.getByText('云端引擎')).toBeTruthy()
    expect(screen.getByText('xAI')).toBeTruthy()
    // 天璇 live=generating → 状态徽标「生成中」在引擎卡上呈现
    expect(screen.getAllByText('生成中').length).toBeGreaterThanOrEqual(2) // 状态条 + 引擎卡
    // herdsman 模型 chip：过滤条件 kind=llm 或 running（bge-m3 running 保留，edge-tts stopped+tts 过滤）
    expect(screen.getByText('bge-m3')).toBeTruthy()
  })

  it('最近请求表：成功/失败行与缓存命中', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('最近请求（gaea 侧全引擎记账）')).toBeTruthy())
    expect(screen.getAllByText('qwen3.8-flash-next-uncensored').length).toBeGreaterThanOrEqual(2) // 引擎卡 + 请求行
    expect(screen.getByText('成功')).toBeTruthy()
    expect(screen.getByText('失败')).toBeTruthy()
    expect(screen.getByText('1,024')).toBeTruthy() // 缓存命中
  })

  it('空快照：无引擎在线/无调用的诚实空态', async () => {
    getRuntimeMonitorMock.mockResolvedValue({
      time: '', system: {}, engines: [], strata: null, tianxuan: null, recent_calls: [], totals: {},
    })
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('无引擎在线')).toBeTruthy())
    expect(screen.getByText('暂无调用记录')).toBeTruthy()
  })

  it('刷新失败保留已有快照并提示', async () => {
    getRuntimeMonitorMock.mockResolvedValueOnce(SNAPSHOT).mockRejectedValueOnce(new Error('boom'))
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getAllByText(/生成中/).length).toBeGreaterThanOrEqual(1))
    // 触发手动刷新（第 2 次调用 = rejected）
    const btn = screen.getByRole('button', { name: '立即刷新' })
    btn.click()
    await waitFor(() => expect(screen.getByText(/刷新失败：boom/)).toBeTruthy())
    expect(screen.getAllByText(/生成中/).length).toBeGreaterThanOrEqual(1)
  })
})
