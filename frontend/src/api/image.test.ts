// image.test.ts — getSystemStats Partial 归一化（v4.415.0 走查根修的回归锚）。
// 背景：绑定契约是 Partial<SystemStats>（faucade 侧类型即如此），此前直透消费方，
// ControlPanel 的 memUsed.toFixed 在字段缺失时抛 TypeError → 绘梦整页崩进 ErrorBoundary。
import { beforeEach, describe, expect, it, vi } from 'vitest'

const bindings = vi.hoisted(() => ({
  GetSystemStats: vi.fn(),
}))

vi.mock('../gaea/lib/bridge', () => ({
  app: {
    GetSystemStats: (...args: unknown[]) => bindings.GetSystemStats(...args),
  },
}))

import { getSystemStats } from './image'

beforeEach(() => {
  bindings.GetSystemStats.mockReset()
})

describe('getSystemStats Partial → 全字段归一化', () => {
  it('缺字段补安全默认值（数值 0 / 字符串空串）', async () => {
    bindings.GetSystemStats.mockResolvedValue({ cpu: 42, memTotal: 32 })
    const s = await getSystemStats()
    expect(s).toEqual({
      cpu: 42, memTotal: 32, memUsed: 0, gpuName: '', gpuUsage: 0, vramUsed: 0, vramTotal: 0,
    })
  })

  it('全字段齐全时透传不篡改', async () => {
    const full = { cpu: 11, memTotal: 32, memUsed: 16, gpuName: 'GPU A', gpuUsage: 55, vramUsed: 4, vramTotal: 8 }
    bindings.GetSystemStats.mockResolvedValue({ ...full })
    await expect(getSystemStats()).resolves.toEqual(full)
  })

  it('非数值字段收 0；非 string 的 gpuName 收空串', async () => {
    bindings.GetSystemStats.mockResolvedValue({ cpu: '71', memUsed: Number.NaN, gpuName: 3 })
    const s = await getSystemStats()
    expect(s?.cpu).toBe(71)
    expect(s?.memUsed).toBe(0)
    expect(s?.gpuName).toBe('')
  })

  it('null / 非对象返回值 → null', async () => {
    bindings.GetSystemStats.mockResolvedValue(null)
    await expect(getSystemStats()).resolves.toBeNull()
    bindings.GetSystemStats.mockResolvedValue('nope')
    await expect(getSystemStats()).resolves.toBeNull()
  })

  it('绑定抛错 → null（既有契约不变）', async () => {
    bindings.GetSystemStats.mockRejectedValue(new Error('boom'))
    await expect(getSystemStats()).resolves.toBeNull()
  })
})
