import { describe, expect, it } from 'vitest'
import { computeResourceSnapshot, fmtGB, resourceLevel } from './resource'

describe('computeResourceSnapshot', () => {
  it('空数据返回全 0 安全快照', () => {
    const s = computeResourceSnapshot(null)
    expect(s.cpu).toBe(0)
    expect(s.memPct).toBe(0)
    expect(s.vramPct).toBe(0)
    expect(s.localEngines).toEqual([])
    expect(s.comfyRunning).toBe(false)
  })

  it('按 GB 计算内存/显存百分比并保留本地引擎', () => {
    const s = computeResourceSnapshot({
      engines: [
        { engine: 'herdsman', model: 'qwen3-8b', isLocal: true },
        { engine: 'xai', model: 'grok-4.20', isLocal: false },
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
    })
    expect(s.memPct).toBe(50)
    expect(s.vramPct).toBe(50)
    expect(s.gpuPct).toBe(50) // gpuUsage 为 0 时回退显存占用
    expect(s.localEngines).toEqual(['herdsman·qwen3-8b'])
    expect(s.comfyRunning).toBe(true)
  })
})

describe('resourceLevel', () => {
  it('按阈值分级', () => {
    expect(resourceLevel(0)).toBe('ok')
    expect(resourceLevel(74)).toBe('ok')
    expect(resourceLevel(75)).toBe('warn')
    expect(resourceLevel(89)).toBe('warn')
    expect(resourceLevel(90)).toBe('high')
  })
})

describe('fmtGB', () => {
  it('格式化为一位小数 GB', () => {
    expect(fmtGB(12.34)).toBe('12.3 GB')
    expect(fmtGB(0)).toBe('--')
  })
})
