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
    expect(screen.getByText('12.2/96.0')).toBeTruthy() // 显存（系统侧读数；分母与分子同一位一位小数）
    expect(screen.getByText('12.5 / 3.2')).toBeTruthy() // 磁盘 读/写 合并卡
    expect(screen.getByText('磁盘 读/写')).toBeTruthy()
    expect(screen.getByText('GPU 温度')).toBeTruthy() // 无遥测卡仍在（值 — ）
    // HUD 主轴结构（2026-10-27 技能重设计）：LIVE 徽标 + 四弹道仪表 + 三副读数
    expect(screen.getByText('LIVE')).toBeTruthy()
    expect(document.querySelectorAll('.rt-gauge').length).toBe(4)
    expect(document.querySelectorAll('.rt-gauge-fill').length).toBe(4)
    expect(document.querySelectorAll('.rt-readout').length).toBe(3)
    // 仪表条宽=满条比例（GPU 62%；显存 12.2/96→12.708…% 未取整=精确比例）
    const fills = document.querySelectorAll('.rt-gauge-fill') as NodeListOf<HTMLElement>
    expect(fills[0].style.width).toBe('62%')
    expect(fills[3].style.width).toBe('12.708333333333332%')
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
    // 档位内汇总读数（默认 7天=三桶合计：调用 87、Token 74K、入 58K/出 16K、¥1.10——
    // 累计口径只在头部 totals 行，本面板数字随档位走）
    expect(screen.getByText('调用 87')).toBeTruthy()
    expect(screen.getByText(/Token 74\.0K · 入 58\.0K \/ 出 16\.0K/)).toBeTruthy()
    expect(screen.getByText('¥1.10')).toBeTruthy()
    // 范围切换三档
    for (const label of ['今日', '7天', '30天']) {
      expect(screen.getByRole('radio', { name: label })).toBeTruthy()
    }
    // HUD 图表（2026-10-27 重画）：平滑曲线 + 堆叠柱（CALL_STATS 3 桶 → 3 组柱）
    expect(screen.getByRole('img', { name: '请求趋势图' })).toBeTruthy()
    expect(screen.getByRole('img', { name: 'Token 分布图' })).toBeTruthy()
    expect(document.querySelectorAll('.rt-hchart-grid').length).toBeGreaterThanOrEqual(4) // 双图各 2 条虚线
    expect(document.querySelectorAll('.rt-tok-bar').length).toBe(6) // 3 桶 × 入/出两段
    expect(document.querySelectorAll('.rt-tok-last').length).toBe(1) // 末柱点亮
    expect(document.querySelectorAll('.rt-tok-dot').length).toBe(2) // 入/出双色图例点
    // 抽屉跳钮已随拍板移除（明细常驻本页=唯一入口；反断言防回流）
    expect(screen.queryByRole('button', { name: /按模型明细/ })).toBeNull()
  })

  it('调用统计段：无趋势数据诚实空态', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('调用统计')).toBeTruthy())
    expect(screen.getByText(/暂无趋势数据/)).toBeTruthy()
  })

  it('调用统计段：首拍加载中与无数据区分（callStats 未到=正在读取）', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    getModelCallStatsMock.mockImplementation(() => new Promise(() => {}))
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('正在读取统计…')).toBeTruthy())
    expect(screen.queryByText(/暂无趋势数据/)).toBeNull()
  })

  it('调用统计段：加载失败出重试钮，重试成功后图表恢复', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    getModelCallStatsMock.mockRejectedValueOnce(new Error('boom')).mockResolvedValue(CALL_STATS)
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText(/统计加载失败：boom/)).toBeTruthy())
    fireEvent.click(screen.getByRole('button', { name: /重\s*试/ })) // antd 两字按钮插空格（既定坑）
    await waitFor(() => expect(screen.getByText('峰值 45')).toBeTruthy())
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
    // 默认 7天：三天日聚合，峰值 45、调用 87
    await waitFor(() => expect(screen.getByText('峰值 45')).toBeTruthy())
    expect(screen.getByText('调用 87')).toBeTruthy()
    // 切「今日」：只剩今天的小时桶，峰值 12、调用 12、费用 0.10
    fireEvent.click(screen.getByRole('radio', { name: '今日' }))
    await waitFor(() => expect(screen.getByText('峰值 12')).toBeTruthy())
    expect(screen.queryByText('峰值 45')).toBeNull()
    expect(screen.getByText('调用 12')).toBeTruthy()
    expect(screen.getByText('¥0.10')).toBeTruthy()
    // 切「30天」：全量三天回来，峰值 45、调用 87
    fireEvent.click(screen.getByRole('radio', { name: '30天' }))
    await waitFor(() => expect(screen.getByText('峰值 45')).toBeTruthy())
    expect(screen.getByText('调用 87')).toBeTruthy()
    // 切回「7天」：与 30天 同为日聚合（数据窗口内一致），仍是 45
    fireEvent.click(screen.getByRole('radio', { name: '7天' }))
    await waitFor(() => expect(screen.getByText('峰值 45')).toBeTruthy())
    expect(screen.getByText('调用 87')).toBeTruthy()
  })

  it('按模型明细常驻板：引擎分组平铺 + 占比条 + 成功率着色 + 失败徽标（累计口径）', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    getModelCallStatsMock.mockResolvedValue({
      ...CALL_STATS,
      per_model: [
        {
          engine_id: 'tianxuan', model: 'qwen3.8-flash-next-uncensored',
          call_count: 80, success_count: 80, fail_count: 0,
          input_tokens: 80000, output_tokens: 20000, total_tokens: 100000,
          total_duration_ms: 1000, estimated_cost: 0,
        },
        {
          engine_id: 'tianxuan', model: 'qwen3.6-35b',
          call_count: 20, success_count: 20, fail_count: 0,
          input_tokens: 4000, output_tokens: 1000, total_tokens: 5000,
          total_duration_ms: 300, estimated_cost: 0,
        },
        {
          engine_id: 'xai', model: 'grok-4.7',
          call_count: 20, success_count: 18, fail_count: 2,
          input_tokens: 6000, output_tokens: 2000, total_tokens: 8000,
          total_duration_ms: 500, estimated_cost: 0.5, currency: 'USD', last_error: 'HTTP 429',
        },
      ],
    })
    render(<RuntimePage />)
    // 分组头：引擎标签（engineLabel 映射）+ 模型数 + 累计小计
    await waitFor(() => expect(screen.getByText('天璇引擎 本地')).toBeTruthy())
    expect(screen.getByText('xAI 云端')).toBeTruthy()
    expect(screen.getByText('1 个模型')).toBeTruthy() // xai 组（天璇组=2 个模型）
    // 行内字段：模型名（引擎卡 chip/请求行/明细行多处出现）/ 成功率（组小计与行各一枚）
    expect(screen.getAllByText('grok-4.7').length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText('100%').length).toBeGreaterThanOrEqual(2)
    const failRates = screen.getAllByText('90%')
    expect(failRates.every((el) => el.className.includes('rt-fail'))).toBe(true)
    // USD 0.5 × 缺省 7.2 → ¥3.60
    expect(screen.getByText('¥3.60')).toBeTruthy()
    // 占比条：组内归一——天璇组 80 满格、20/80=25%；xai 组单模型 100%
    const bars = document.querySelectorAll('.rt-mb-bar')
    expect(bars.length).toBe(3)
    expect((bars[0] as HTMLElement).style.width).toBe('100%')
    expect((bars[1] as HTMLElement).style.width).toBe('25%')
    expect((bars[2] as HTMLElement).style.width).toBe('100%')
    // 失败徽标（Tooltip title=last_error；精确串避免撞趋势图 SVG title 的「失败 2」）
    expect(screen.getByText('败 2')).toBeTruthy()
    // 累计口径节标明示（与右列趋势档位两套口径）
    expect(screen.getByText('累计口径 · 全引擎平铺')).toBeTruthy()
  })

  it('按模型明细常驻板：空数据诚实空态', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<RuntimePage />) // beforeEach 默认 per_model: []
    await waitFor(() => expect(screen.getByText(/暂无模型调用记录/)).toBeTruthy())
  })

  it('按模型明细常驻板：统计未到显示明细加载中（与右列「正在读取统计…」文案分立）', async () => {
    getRuntimeMonitorMock.mockResolvedValue(SNAPSHOT)
    getModelCallStatsMock.mockImplementation(() => new Promise(() => {}))
    render(<RuntimePage />)
    await waitFor(() => expect(screen.getByText('明细加载中…')).toBeTruthy())
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
