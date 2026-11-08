import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import ProgrammingPage, {
  DSH_INSTALL_COMMAND,
  DSH_LOCAL_URL,
  DSH_REPO_URL,
  DSH_VERSION,
  REA_REPO_URL,
} from './ProgrammingPage'
import type {
  ProgrammingWebPreflight,
  ProgrammingWebStatus,
} from '../gaea/lib/types'

const mocks = vi.hoisted(() => ({
  status: vi.fn(),
  preflight: vi.fn(),
  start: vi.fn(),
  stop: vi.fn(),
  logs: vi.fn(),
  openExternal: vi.fn(),
}))

vi.mock('../gaea/lib/bridge', () => ({
  app: {
    GetProgrammingWebStatus: mocks.status,
    GetProgrammingWebPreflight: mocks.preflight,
    StartProgrammingWeb: mocks.start,
    StopProgrammingWeb: mocks.stop,
    ProgrammingWebLogTail: mocks.logs,
  },
}))

vi.mock('../gaea/lib/bridge/proxy', () => ({
  openExternal: mocks.openExternal,
}))

const running: ProgrammingWebStatus = {
  running: true,
  owned: true,
  starting: false,
  pid: 4242,
  url: `${DSH_LOCAL_URL}/?token=test-token`,
  root: String.raw`C:\AI\dsh`,
  home: String.raw`C:\AI\dsh\home`,
  version: DSH_VERSION,
  log: String.raw`C:\AI\dsh\home\dsh-web.log`,
  uptime_s: 12,
}

const idle: ProgrammingWebStatus = {
  ...running,
  running: false,
  owned: false,
  pid: 0,
  url: `${DSH_LOCAL_URL}/`,
  uptime_s: 0,
}

const ready: ProgrammingWebPreflight = {
  node_found: true,
  node_path: 'node',
  install_ready: true,
  profile_ready: true,
  plugin_ready: true,
  rea_ready: true,
  isolated: true,
  version_ok: true,
  port_free: true,
  all_ready: true,
  root: String.raw`C:\AI\dsh`,
  home: String.raw`C:\AI\dsh\home`,
  agents_home: String.raw`C:\AI\dsh\home\agents`,
  base: String.raw`C:\AI\dsh\node_modules\@deepseek-ai`,
  version: DSH_VERSION,
  expected: DSH_VERSION,
}

beforeEach(() => {
  vi.clearAllMocks()
  mocks.stop.mockResolvedValue(undefined)
  mocks.logs.mockResolvedValue({ exists: false, path: '', lines: [], error: '' })
})

describe('ProgrammingPage', () => {
  it('运行中时直接内嵌 DSH Web，并保留“编程”H1', async () => {
    mocks.status.mockResolvedValue(running)
    mocks.preflight.mockResolvedValue(ready)

    render(<ProgrammingPage />)

    expect(screen.getByRole('heading', { level: 1, name: '编程' })).toBeTruthy()
    const frame = await screen.findByTitle('DeepSeek Harness Web') as HTMLIFrameElement
    expect(frame.src).toBe(running.url)
    expect(screen.getByText(`运行中 · ${DSH_VERSION}`)).toBeTruthy()
  })

  it('空闲时自动启动隔离 DSH Web，再切换为内嵌页面', async () => {
    mocks.status
      .mockResolvedValueOnce(idle)
      .mockResolvedValueOnce(idle)
      .mockResolvedValue(running)
    mocks.preflight.mockResolvedValue(ready)
    mocks.start.mockResolvedValue(undefined)

    render(<ProgrammingPage />)

    const frame = await screen.findByTitle('DeepSeek Harness Web') as HTMLIFrameElement
    expect(mocks.start).toHaveBeenCalledTimes(1)
    expect(frame.src).toBe(running.url)
  })

  it('环境不匹配时展示原因，并保留官方安装入口', async () => {
    mocks.status.mockResolvedValue(idle)
    mocks.preflight.mockResolvedValue({
      ...ready,
      version_ok: false,
      version: '0.2.1-rc.5',
    })

    render(<ProgrammingPage />)

    expect(await screen.findByText(/适配目标为 0\.2\.2-rc\.1/)).toBeTruthy()
    const install = screen.getByRole('link', { name: /打开官方网页版安装说明/ })
    expect(install.getAttribute('href')).toBe(DSH_REPO_URL)
    fireEvent.click(install)
    expect(mocks.openExternal).toHaveBeenCalledWith(DSH_REPO_URL)
    expect(screen.getByText(DSH_INSTALL_COMMAND)).toBeTruthy()
    const rea = screen.getByRole('link', { name: /REA 逆向工程工具/ })
    expect(rea.getAttribute('href')).toBe(REA_REPO_URL)
    expect(screen.getByText(/REA MCP 6\.8\.0 · 229 tools/)).toBeTruthy()
    expect(screen.getByText(String.raw`C:\AI\dsh\home\agents`)).toBeTruthy()
  })

  it('展示可直接执行的隔离安装命令与复制按钮', async () => {
    mocks.status.mockResolvedValue(idle)
    mocks.preflight.mockResolvedValue({ ...ready, version_ok: false })

    render(<ProgrammingPage />)
    expect(await screen.findByLabelText('复制安装命令')).toBeTruthy()
    expect(screen.getByText(DSH_INSTALL_COMMAND)).toBeTruthy()
  })
})
