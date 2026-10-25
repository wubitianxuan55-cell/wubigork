import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import RuntimePage from './RuntimePage'

const { getRuntimeMonitorMock, getModelCallStatsMock } = vi.hoisted(() => ({
  getRuntimeMonitorMock: vi.fn(),
  getModelCallStatsMock: vi.fn(),
}))

vi.mock('../api/engines', () => ({
  getRuntimeMonitor: getRuntimeMonitorMock,
  getModelCallStats: getModelCallStatsMock,
}))

// 轮询门默认开（jsdom visibilityState=visible）；测试不推进假时钟，靠挂载
// 首拍渲染断言。
vi.mock('../hooks/usePollingGate', () => ({
  usePollingGate: () => true,
}))

const SNAPSHOT = {
  time: '2026-10-19 20:00:00',
  system: { cpu: 23, memTotal: 109, memUsed: 42, gpuName: 'AMD Radeon 8060S', gpuUsage: 62, vramTotal: 96, vramUsed: 12.2 },
  comfy_running: true,
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
    engine: { model: 'qwen3.8-flash-next-uncensored', max_context: 4096 },
    hardware: { cpu: 31, ram_used: 45.2, ram_total: 109, gpu_util: null, gpu_temp: null, disk_read_mb: 12.5, disk_write_mb: 3.2 },
    requests: [
      {
        time: 1763400000, duration_s: 5.4, finish: 'stop', prompt_tokens: 1520, prompt_total: 1600,
        prompt_read: 1520, reused: 1024, output_tokens: 340, prompt_ms: 1400, decode_tok_s: 65.2, hit_rate: 0.63,
      },
      { time: 1763399000, duration_s: 1.2, finish: 'error', prompt_tokens: 80, output_tokens: 0 },
    ],
    conversation_cache: {
      enabled: true, budget_mib: 4096, slots: 4,
      requests: 12, requests_reused: 8, reused_tokens: 8192, prompt_tokens: 16384,
    },
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
  totals: { total_calls: 1200, success_calls: 1190, fail_calls: 10, total_tokens: 2400000, input_tokens: 1900000, output_tokens: 500000, total_cost: 3.45 },
}

// 调用统计（原模型中心检查器并入）：趋势按小时桶返回，hook 聚合到天。
const CALL_STATS = {
  total_calls: 1200, success_calls: 1190, fail_calls: 10,
  total_tokens: 2400000, input_tokens: 1900000, output_tokens: 500000,
  total_duration_ms: 1000, avg_duration_ms: 1, total_cost: 3.45,
  trend: [
    { time: '2026-10-17 08:00:00', calls: 12, success_calls: 12, fail_calls: 0, input_tokens: 8000, output_tokens: 2000, total_tokens: 10000, cost: 0.1 },
    { time: '2026-10-18 09:00:00', calls: 30, success_calls: 28, fail_calls: 2, input_tokens: 20000, output_tokens: 5000, total_tokens: 25000, cost: 0.4 },
    { time: '2026-10-19 20:00:00', calls: 45, success_calls: 45, fail_calls: 0, input_tokens: 30000, output_tokens: 9000, total_tokens: 39000, cost: 0.6 },
  ],
  per_model: [],
}

describe('RuntimePage 运行监测板块', () => {
  beforeEach(() => {
    // mockReset（非 clearAllMocks）：清实现 + Once 队列，防上个用例的持久实现串台
    getRuntimeMonitorMock.mockReset()
    // 调用统计默认空摘要（无趋势=诚实空态；有趋势用例自行覆写）
    getModelCallStatsMock.mockReset()
    getModelCallStatsMock.mockResolvedValue({ ...CALL_STATS, trend: [] })
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

  it('指标卡：遥测有值显示数值，GPU 吃系统 PDH 兜底，温度/功率诚实留空', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('生成速度')).toBeTruthy())
    expect(screen.getAllByText('65.2').length).toBeGreaterThanOrEqual(1) // tok/s（引擎 live；引擎请求表 decode 列同值）
    expect(screen.getByText('31')).toBeTruthy() // CPU（引擎遥测优先）
    expect(screen.getByText('62')).toBeTruthy() // GPU 负载：引擎遥测 null → 系统 PDH（gpuUsage 62）兜底
    expect(screen.getByText('12.2/96')).toBeTruthy() // 显存（系统侧读数）
    expect(screen.getByText('12.5 / 3.2')).toBeTruthy() // 磁盘 读/写 合并卡
    expect(screen.getByText('磁盘 读/写')).toBeTruthy()
    expect(screen.getByText('GPU 温度')).toBeTruthy() // 无遥测卡仍在（值 — ）
  })

  it('引擎侧遥测：Context 环 + 会话缓存 + 引擎请求表（prefill/decode/VRAM 命中）', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('引擎侧请求（天璇 /metrics）')).toBeTruthy())
    // Context 环：最近请求 prompt_total+输出 1940 / max_context 4096 ≈ 47%
    expect(screen.getByText('47%')).toBeTruthy()
    expect(screen.getByText('1.9K / 4.1K tok')).toBeTruthy()
    // 会话缓存口径
    expect(screen.getByText('预算 4096 MiB · 4 槽')).toBeTruthy()
    expect(screen.getByText(/复用 8 次/)).toBeTruthy()
    expect(screen.getByText(/Prompt 复用 8.2K \/ 16.4K tok（50%）/)).toBeTruthy()
    // 引擎请求行：prefill = prompt_read/prompt_ms = 1520/1.4s ≈ 1086；decode 65.2；VRAM 命中 63%
    expect(screen.getByText('1086')).toBeTruthy()
    expect(screen.getAllByText('65.2').length).toBeGreaterThanOrEqual(2) // 指标卡 + decode 列
    expect(screen.getByText('63%')).toBeTruthy()
    expect(screen.getByText('复用 1.0K')).toBeTruthy() // Prompt 列内复用子标注
    expect(screen.getByText('正常')).toBeTruthy()
    expect(screen.getByText('错误')).toBeTruthy()
  })

  it('引擎卡：本地三引擎 + 云端聚合 + ComfyUI 卡 + 生成中状态徽标', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('Herdsman 本地')).toBeTruthy())
    expect(screen.getByText('天璇 Tianxuan')).toBeTruthy()
    expect(screen.getByText('Strata')).toBeTruthy()
    expect(screen.getByText('云端引擎')).toBeTruthy()
    expect(screen.getByText('xAI')).toBeTruthy()
    // ComfyUI 卡（原模型中心资源检查器 chip 并入）：comfy_running=true 点亮
    expect(screen.getByText('ComfyUI')).toBeTruthy()
    expect(screen.getByText('运行中 · 画室生成队列可用')).toBeTruthy()
    // 天璇 live=generating → 状态徽标「生成中」在引擎卡上呈现
    expect(screen.getAllByText('生成中').length).toBeGreaterThanOrEqual(2) // 状态条 + 引擎卡
    // herdsman 模型 chip：过滤条件 kind=llm 或 running（bge-m3 running 保留，edge-tts stopped+tts 过滤）
    expect(screen.getByText('bge-m3')).toBeTruthy()
  })

  it('ComfyUI 卡：未运行时灰态呈现（快照缺 comfy_running=false）', async () => {
    getRuntimeMonitorMock.mockResolvedValue({ ...SNAPSHOT, comfy_running: false })
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('ComfyUI')).toBeTruthy())
    expect(screen.getByText('未运行')).toBeTruthy()
  })

  it('调用统计段（检查器并入）：趋势双图 + 范围切换 + 入/出拆分 + 明细跳转', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    getModelCallStatsMock.mockResolvedValue(CALL_STATS)
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('调用统计')).toBeTruthy())
    // 双迷你图标题行（峰值=三天最大 45）
    expect(screen.getByText('请求趋势')).toBeTruthy()
    expect(screen.getByText('Token 分布')).toBeTruthy()
    expect(screen.getByText('峰值 45')).toBeTruthy()
    // 入/出拆分（totals 快照口径）
    expect(screen.getByText(/入 1,900,000 \/ 出 500,000/)).toBeTruthy()
    // 范围切换三档
    for (const label of ['今日', '7天', '30天']) {
      expect(screen.getByRole('radio', { name: label })).toBeTruthy()
    }
    // SVG 迷你图渲染（aria-label 由 charts 组件给出）
    expect(screen.getByRole('img', { name: '请求趋势迷你图' })).toBeTruthy()
    expect(screen.getByRole('img', { name: 'Token 分布迷你图' })).toBeTruthy()
    // 跳模型中心入口
    expect(screen.getByRole('button', { name: /按模型明细/ })).toBeTruthy()
  })

  it('调用统计段：无趋势数据诚实空态', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('调用统计')).toBeTruthy())
    expect(screen.getByText(/暂无趋势数据/)).toBeTruthy()
  })

  it('调用统计段：今日/7天/30天切换真实换数（峰值随档位变化）', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    // 趋势含「今天」小时桶 + 前两天各一桶（日期动态生成防跨日脆断；
    // time 格式对齐 Go 小时桶 "YYYY-MM-DDTHH:00"）
    const hourBucket = (daysAgo: number, hour: number) => {
      const dt = new Date()
      dt.setDate(dt.getDate() - daysAgo)
      const p2 = (n: number) => String(n).padStart(2, '0')
      return `${dt.getFullYear()}-${p2(dt.getMonth() + 1)}-${p2(dt.getDate())}T${p2(hour)}:00`
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
    // 默认 7天：三天日聚合，峰值 45
    await waitFor(() => expect(screen.getByText('峰值 45')).toBeTruthy())
    // 切「今日」：只剩今天的小时桶，峰值 12
    fireEvent.click(screen.getByRole('radio', { name: '今日' }))
    await waitFor(() => expect(screen.getByText('峰值 12')).toBeTruthy())
    expect(screen.queryByText('峰值 45')).toBeNull()
    // 切「30天」：全量三天回来，峰值 45
    fireEvent.click(screen.getByRole('radio', { name: '30天' }))
    await waitFor(() => expect(screen.getByText('峰值 45')).toBeTruthy())
    // 切回「7天」：与 30天 同为日聚合（数据窗口内一致），仍是 45
    fireEvent.click(screen.getByRole('radio', { name: '7天' }))
    await waitFor(() => expect(screen.getByText('峰值 45')).toBeTruthy())
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
