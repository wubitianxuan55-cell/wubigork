// ProgrammingPage.test.tsx — 编程板块「DeepSeek Harness 编程工作台」测试。
// 桥接经 vi.mock("../gaea/lib/bridge") 注入确定性数据：未运行引导视图
// （前置条件清单/启动/日志）、启动中动画视图、运行中内嵌工作台
// （iframe/运行时长/归属/停止守卫）。

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// 屏蔽 bridge seam（vi.hoisted 避免 mock 提升导致的初始化顺序问题）
const mocks = vi.hoisted(() => ({
  GetProgrammingWebStatus: vi.fn(),
  GetProgrammingWebPreflight: vi.fn(),
  ProgrammingWebLogTail: vi.fn(),
  StartProgrammingWeb: vi.fn(),
  StopProgrammingWeb: vi.fn(),
}))
vi.mock('../gaea/lib/bridge', () => ({
  app: {
    GetProgrammingWebStatus: mocks.GetProgrammingWebStatus,
    GetProgrammingWebPreflight: mocks.GetProgrammingWebPreflight,
    ProgrammingWebLogTail: mocks.ProgrammingWebLogTail,
    StartProgrammingWeb: mocks.StartProgrammingWeb,
    StopProgrammingWeb: mocks.StopProgrammingWeb,
  },
}))

import ProgrammingPage from './ProgrammingPage'

const idleStatus = {
  running: false, owned: false, pid: 0,
  url: 'http://127.0.0.1:3080',
  root: 'C:\\AI\\deepseek-harness',
  log: 'C:\\tmp\\gaea-dsh-web.log',
  uptime_s: 0,
}
const readyPreflight = {
  harness_valid: true, pnpm_found: true, deps_ready: true,
  build_ready: true, port_free: true, all_ready: true,
  root: 'C:\\AI\\deepseek-harness',
}

const PREFLIGHT_LABELS = ['Harness 目录有效', 'pnpm 可用', '依赖已安装', 'Web 构建产物就绪', '端口 3080 空闲']

beforeEach(() => {
  vi.clearAllMocks()
  mocks.GetProgrammingWebStatus.mockResolvedValue(idleStatus)
  mocks.GetProgrammingWebPreflight.mockResolvedValue(readyPreflight)
  mocks.ProgrammingWebLogTail.mockResolvedValue({
    exists: true, path: 'C:\\tmp\\gaea-dsh-web.log',
    lines: ['[info] boot dsh web', '[info] listening on :3080'],
    error: '',
  })
  mocks.StartProgrammingWeb.mockResolvedValue(undefined)
  mocks.StopProgrammingWeb.mockResolvedValue(undefined)
})

describe('ProgrammingPage 未运行：启动引导视图', () => {
  it('渲染启动面板 + 5 项前置条件清单（全部就绪 → 启动按钮可用）', async () => {
    render(<ProgrammingPage />)
    expect(await screen.findByText('编程工作台')).toBeTruthy()
    expect(screen.getByRole('button', { name: /启动编程工作台/ })).toBeTruthy()
    for (const label of PREFLIGHT_LABELS) {
      expect(await screen.findByText(label)).toBeTruthy()
    }
    expect(await screen.findByText(/全部就绪，可一键启动/)).toBeTruthy()
    await waitFor(() => {
      const btn = screen.getByRole('button', { name: /启动编程工作台/ }) as HTMLButtonElement
      expect(btn.disabled).toBe(false)
    })
  })

  it('前置条件未满足：红叉条目 + 提示 + 启动按钮禁用', async () => {
    mocks.GetProgrammingWebPreflight.mockResolvedValue({
      ...readyPreflight, build_ready: false, pnpm_found: false, all_ready: false,
    })
    render(<ProgrammingPage />)
    expect(await screen.findByText(/存在未满足项/)).toBeTruthy()
    // 未满足项条目带 is-fail 标记
    await waitFor(() => {
      const items = screen.getAllByText('Web 构建产物就绪')
      expect(items.length).toBeGreaterThan(0)
    })
    expect(screen.getByRole('button', { name: /启动编程工作台/ })).toBeTruthy()
    const btn = screen.getByRole('button', { name: /启动编程工作台/ }) as HTMLButtonElement
    expect(btn.disabled).toBe(true)
  })

  it('点击「启动编程工作台」调用 StartProgrammingWeb', async () => {
    render(<ProgrammingPage />)
    await screen.findByText(/全部就绪/)
    fireEvent.click(screen.getByRole('button', { name: /启动编程工作台/ }))
    await waitFor(() => expect(mocks.StartProgrammingWeb).toHaveBeenCalled())
  })

  it('点击「重新检查」再次拉取前置条件', async () => {
    render(<ProgrammingPage />)
    await screen.findByText(/全部就绪/)
    fireEvent.click(screen.getByRole('button', { name: /重新检查前置条件/ }))
    await waitFor(() => expect(mocks.GetProgrammingWebPreflight).toHaveBeenCalledTimes(2))
  })

  it('启动中：显示启动动画视图（计时），就绪后切入内嵌工作台', async () => {
    let resolveStart!: () => void
    mocks.StartProgrammingWeb.mockReturnValue(new Promise<void>((res) => { resolveStart = res }))
    render(<ProgrammingPage />)
    await screen.findByText(/全部就绪/)
    fireEvent.click(screen.getByRole('button', { name: /启动编程工作台/ }))
    // 启动动画视图（而非 iframe）
    expect(await screen.findByText('正在启动编程工作台')).toBeTruthy()
    expect(screen.getByText(/已等待/)).toBeTruthy()
    expect(screen.queryByTitle('DeepSeek Harness 编程工作台')).toBeNull()
    // 启动完成 → 轮询返回运行中 → 切入 iframe 工作台
    mocks.GetProgrammingWebStatus.mockResolvedValue({
      ...idleStatus, running: true, owned: true, pid: 9, uptime_s: 3,
    })
    resolveStart()
    const frame = await screen.findByTitle('DeepSeek Harness 编程工作台')
    expect(frame.getAttribute('src')).toBe('http://127.0.0.1:3080')
  })

  it('启动失败：回到引导视图 + 错误提示 + 自动展开日志', async () => {
    mocks.StartProgrammingWeb.mockRejectedValue(new Error('端口 3080 已被其他进程占用（非 gaea 自启实例）'))
    render(<ProgrammingPage />)
    await screen.findByText(/全部就绪/)
    fireEvent.click(screen.getByRole('button', { name: /启动编程工作台/ }))
    // 回到引导视图并显示错误
    expect(await screen.findByText(/端口 3080 已被其他进程占用/)).toBeTruthy()
    expect(await screen.findByText(/全部就绪，可一键启动/)).toBeTruthy()
    // 日志面板自动展开并拉取
    await waitFor(() => expect(mocks.ProgrammingWebLogTail).toHaveBeenCalled())
    expect(await screen.findByText(/listening on :3080/)).toBeTruthy()
  })
})

describe('ProgrammingPage 运行中：桌面内嵌工作台', () => {
  it('渲染 iframe 工作台 + 自启实例运行时长芯片', async () => {
    mocks.GetProgrammingWebStatus.mockResolvedValue({
      ...idleStatus, running: true, owned: true, pid: 99, uptime_s: 65,
    })
    render(<ProgrammingPage />)
    const frame = await screen.findByTitle('DeepSeek Harness 编程工作台')
    expect(frame.getAttribute('src')).toBe('http://127.0.0.1:3080')
    expect(await screen.findByText('1 分 5 秒')).toBeTruthy()
    // 自启实例可停止
    const stopBtn = screen.getByRole('button', { name: '停止服务' }) as HTMLButtonElement
    expect(stopBtn.disabled).toBe(false)
  })

  it('外部实例：外部实例芯片 + 停止按钮禁用（不误杀）', async () => {
    mocks.GetProgrammingWebStatus.mockResolvedValue({
      ...idleStatus, running: true, owned: false,
    })
    render(<ProgrammingPage />)
    expect(await screen.findByText('外部实例')).toBeTruthy()
    const stopBtn = screen.getByRole('button', { name: '停止服务' }) as HTMLButtonElement
    expect(stopBtn.disabled).toBe(true)
  })

  it('点击停止调用 StopProgrammingWeb（仅自启实例）', async () => {
    mocks.GetProgrammingWebStatus.mockResolvedValue({
      ...idleStatus, running: true, owned: true, pid: 7, uptime_s: 120,
    })
    render(<ProgrammingPage />)
    await screen.findByTitle('DeepSeek Harness 编程工作台')
    fireEvent.click(screen.getByRole('button', { name: '停止服务' }))
    await waitFor(() => expect(mocks.StopProgrammingWeb).toHaveBeenCalled())
  })
})

describe('ProgrammingPage 启动日志', () => {
  it('展开日志面板：读取尾部并渲染日志行', async () => {
    render(<ProgrammingPage />)
    fireEvent.click(await screen.findByRole('button', { name: /启动日志/ }))
    expect(await screen.findByText(/listening on :3080/)).toBeTruthy()
    expect(mocks.ProgrammingWebLogTail).toHaveBeenCalledWith(100)
  })

  it('日志尚未生成：显示提示文案空态', async () => {
    mocks.ProgrammingWebLogTail.mockResolvedValue({
      exists: false, path: 'x.log', lines: [], error: '日志文件尚未生成（第一次启动后出现）',
    })
    render(<ProgrammingPage />)
    fireEvent.click(await screen.findByRole('button', { name: /启动日志/ }))
    expect(await screen.findByText('日志文件尚未生成（第一次启动后出现）')).toBeTruthy()
  })
})
