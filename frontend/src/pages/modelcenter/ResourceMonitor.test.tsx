import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ResourceMonitor } from './ResourceMonitor'

const { getMonitorMock } = vi.hoisted(() => ({
  getMonitorMock: vi.fn(),
}))

vi.mock('../../api/engines', () => ({
  getModelMonitor: getMonitorMock,
}))

const SNAPSHOT = {
  engines: [
    { engine: 'herdsman', name: 'Herdsman 本地', model: 'qwen3-8b', isLocal: true },
    { engine: 'xai', name: 'xAI 云端', model: 'grok-4.20', isLocal: false },
  ],
  stats: {
    cpu: 42,
    memTotal: 32,
    memUsed: 16,
    gpuName: 'Radeon 8060S',
    gpuUsage: 0,
    vramTotal: 16,
    vramUsed: 8,
  },
  comfyRunning: true,
}

describe('ResourceMonitor 右侧资源块', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('首次加载完成前显示加载态，不隐藏资源块', () => {
    getMonitorMock.mockImplementation(() => new Promise(() => {}))
    render(<ResourceMonitor />)
    expect(screen.getByText('本地资源')).toBeTruthy()
    expect(screen.getByText('正在读取系统资源…')).toBeTruthy()
  })

  it('加载成功后展示 CPU/内存/显存占用与本地引擎/ComfyUI 状态', async () => {
    getMonitorMock.mockResolvedValue(SNAPSHOT)
    render(<ResourceMonitor />)
    await waitFor(() => expect(screen.getByText('42%')).toBeTruthy()) // CPU
    expect(screen.getAllByText('50%').length).toBeGreaterThanOrEqual(3) // 内存/GPU/显存
    expect(screen.getByText('herdsman·qwen3-8b')).toBeTruthy()
    expect(screen.getByText('ComfyUI 运行中')).toBeTruthy()
    expect(screen.getByText('Radeon 8060S')).toBeTruthy()
  })

  it('加载失败显示错误与重试，重试成功后恢复数据', async () => {
    getMonitorMock.mockRejectedValueOnce(new Error('backend down'))
    render(<ResourceMonitor />)
    await waitFor(() => expect(screen.getByText(/资源加载失败/)).toBeTruthy())

    getMonitorMock.mockResolvedValueOnce(SNAPSHOT)
    fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }))
    await waitFor(() => expect(screen.getByText('42%')).toBeTruthy())
    expect(screen.queryByText(/资源加载失败/)).toBeNull()
  })
})
