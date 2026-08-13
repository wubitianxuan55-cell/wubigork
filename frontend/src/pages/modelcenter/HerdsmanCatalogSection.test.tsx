import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { HerdsmanCatalogSection } from './HerdsmanCatalogSection'

const { getHerdsmanCatalogMock, getPresetsMock, getStatsMock, startMock, stopMock, downloadMock, uninstallMock } = vi.hoisted(() => ({
  getHerdsmanCatalogMock: vi.fn(),
  getPresetsMock: vi.fn(),
  getStatsMock: vi.fn(),
  startMock: vi.fn(),
  stopMock: vi.fn(),
  downloadMock: vi.fn(),
  uninstallMock: vi.fn(),
}))

vi.mock('../../api/engines', () => ({
  getHerdsmanCatalog: getHerdsmanCatalogMock,
  getHerdsmanLaunchPresets: getPresetsMock,
  getHerdsmanModelStats: getStatsMock,
  startHerdsmanModel: startMock,
  stopHerdsmanModel: stopMock,
  downloadHerdsmanModel: downloadMock,
  uninstallHerdsmanModel: uninstallMock,
}))

const FIXTURE = {
  models: [
    {
      name: 'bge-m3',
      display_name: 'bge-m3',
      type: 'embedding',
      runtime: 'llama.cpp',
      installed: true,
      running: true,
      status: 'installed',
      quantization: 'Q4_K_M',
      parameter_count: 0.568,
      file_size: 437778496,
      llama_cpp_variants: ['standard'],
      hint: '本地语义向量（embedding），驱动语义召回与文件索引',
    },
    {
      name: 'Hunyuan-MT:7B',
      display_name: '混元 MT 7B 翻译模型',
      type: 'text-generation',
      runtime: 'llama.cpp',
      capabilities: ['translation'],
      installed: false,
      running: false,
      status: 'uninstalled',
      quantization: 'Q4_K_M',
      parameter_count: 7,
      llama_cpp_variants: ['standard', 'phison-aicache'],
    },
    {
      name: 'zimage-turbo',
      display_name: 'Z-Image-Turbo',
      type: 'image-generation',
      runtime: 'sd-cpp',
      capabilities: ['text-to-image', 'image-to-image'],
      installed: true,
      running: false,
      status: 'installed',
      quantization: 'Q4_K',
      file_size: 20027974026,
    },
  ],
  total: 3,
  installed: 2,
  running: 1,
  source: 'herdsman-cli',
  installed_bytes: 437778496 + 20027974026,
  disk_total: 2 ** 40,
  disk_free: 200 * 2 ** 30,
}

describe('HerdsmanCatalogSection 模型库', () => {
  it('渲染 KPI 与模型卡片（含能力/量化/大小徽标）', async () => {
    getHerdsmanCatalogMock.mockResolvedValue(FIXTURE)
    getPresetsMock.mockResolvedValue([])
    getStatsMock.mockResolvedValue({
      total: 1,
      since: '2026-08-13T09:00:00+08:00',
      source: 'herdsman-model-stats',
      per_model: [{
        model: 'bge-m3',
        type: 'embedding',
        runtime: 'llama.cpp',
        calls: 12,
        succeeded: 11,
        failed: 1,
        input_tokens: 100,
        output_tokens: 50,
        total_duration_ms: 12000,
        avg_duration_ms: 1000,
        avg_ttft_ms: 200,
        avg_prompt_tps: 300,
        avg_predicted_tps: 150,
        last_called_at: '',
      }],
    })
    render(<HerdsmanCatalogSection />)

    expect(await screen.findByText('混元 MT 7B 翻译模型')).toBeTruthy()
    expect(screen.getByText('translation')).toBeTruthy()
    expect(screen.getAllByText('Q4_K_M').length).toBeGreaterThan(0)
    expect(screen.getByText('18.7 GB')).toBeTruthy() // zimage-turbo 大小
    expect(screen.getByText('Herdsman 本地调用统计')).toBeTruthy()
    expect(screen.getByText('300.0 / 150.0')).toBeTruthy() // Prompt/预测 TPS
    expect(screen.getByText('已安装 · 未运行')).toBeTruthy()
    expect(screen.getAllByText('运行中').length).toBeGreaterThan(0)
    expect(screen.getAllByText('未安装').length).toBeGreaterThan(0)
    // 用途建议（H0-5：受控测评结论上卡片）
    expect(screen.getByText('本地语义向量（embedding），驱动语义召回与文件索引')).toBeTruthy()
    // 磁盘治理 KPI（E1-4）：已装空间 = bge-m3 + zimage 之和 ≈ 19.1GB（唯一文案；
    // zimage 单独为 18.7 GB 徽标）、磁盘余量 200GB/1TB
    expect(screen.getByText('19.1 GB')).toBeTruthy()
    const vals = Array.from(document.querySelectorAll('.mc-kpi-value')).map(n => n.textContent)
    expect(vals.join(' | ')).toContain('200.0 GB / 1.0 TB')
    expect(screen.getByText('200.0 GB / 1.0 TB')).toBeTruthy()
  })

  it('按运行状态过滤', async () => {
    getHerdsmanCatalogMock.mockResolvedValue(FIXTURE)
    getPresetsMock.mockResolvedValue([])
    getStatsMock.mockResolvedValue({ total: 0, since: '', per_model: [], source: 'x' })
    render(<HerdsmanCatalogSection />)
    await screen.findByText('混元 MT 7B 翻译模型')

    fireEvent.click(screen.getByRole('button', { name: '运行中' }))
    expect(screen.queryByText('混元 MT 7B 翻译模型')).toBeNull()
    expect(screen.queryByText('Z-Image-Turbo')).toBeNull()
    expect(screen.getByText('bge-m3')).toBeTruthy()
  })

  it('按能力关键词搜索', async () => {
    getHerdsmanCatalogMock.mockResolvedValue(FIXTURE)
    getPresetsMock.mockResolvedValue([])
    getStatsMock.mockResolvedValue({ total: 0, since: '', per_model: [], source: 'x' })
    render(<HerdsmanCatalogSection />)
    await screen.findByText('混元 MT 7B 翻译模型')

    fireEvent.change(screen.getByPlaceholderText('搜索模型名称 / 能力（如 translation、voice-clone）'), {
      target: { value: 'translation' },
    })
    expect(screen.getByText('混元 MT 7B 翻译模型')).toBeTruthy()
    expect(screen.queryByText('bge-m3')).toBeNull()
  })

  it('目录不可用时展示错误空态', async () => {
    getHerdsmanCatalogMock.mockRejectedValue(new Error('herdsman 未运行'))
    getPresetsMock.mockResolvedValue([])
    getStatsMock.mockResolvedValue({ total: 0, since: '', per_model: [], source: 'x' })
    render(<HerdsmanCatalogSection />)
    await waitFor(() => expect(screen.getByText('模型目录不可用')).toBeTruthy())
    expect(screen.getByText(/herdsman 未运行/)).toBeTruthy()
  })

  it('生命周期操作：运行中可停止，已安装可启动/卸载，未安装可下载', async () => {
    getHerdsmanCatalogMock.mockResolvedValue(FIXTURE)
    getPresetsMock.mockResolvedValue([
      {
        model: 'zimage-turbo',
        engine: 'sd-cpp',
        port: 8080,
        started_at: '2026-08-13T10:00:00+08:00',
        options: { context_size: 262144, gpu_layers: 99 },
      },
    ])
    getStatsMock.mockResolvedValue({ total: 0, since: '', per_model: [], source: 'x' })
    startMock.mockResolvedValue({ ok: true, status: 'completed', message: '' })
    stopMock.mockResolvedValue({ ok: true, status: 'completed', message: '' })
    downloadMock.mockResolvedValue({ ok: true, status: 'completed', message: '' })
    uninstallMock.mockResolvedValue({ ok: true, status: 'completed', message: '' })
    render(<HerdsmanCatalogSection />)

    await screen.findByText('混元 MT 7B 翻译模型')
    expect(screen.getAllByRole('button', { name: /停\s*止/ }).length).toBe(1) // bge-m3 运行中
    expect(screen.getAllByRole('button', { name: /启\s*动/ }).length).toBe(1) // zimage-turbo 已安装
    expect(screen.getByRole('button', { name: /下\s*载/ })).toBeTruthy() // Hunyuan 未安装
    expect(screen.getByText('启动预设')).toBeTruthy() // zimage-turbo 有 launch_records

    fireEvent.click(screen.getByRole('button', { name: /启\s*动/ }))
    await waitFor(() => expect(startMock).toHaveBeenCalledWith('zimage-turbo'))

    fireEvent.click(screen.getByRole('button', { name: /停\s*止/ }))
    await waitFor(() => expect(stopMock).toHaveBeenCalledWith('bge-m3'))

    fireEvent.click(screen.getByRole('button', { name: /下\s*载/ }))
    await waitFor(() => expect(downloadMock).toHaveBeenCalledWith('Hunyuan-MT:7B'))
  })
})
