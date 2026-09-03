/**
 * SchedulingSection.test.tsx — C 刀「故障转移」开关卡测试
 *
 * 第三张开关卡照保活/自动预载卡逐字样式：读取渲染、切换调用
 * setEngineFailover、读取失败显示「未知」并禁用开关。
 * bridge app facade 与 api/engines 均按既有 mock factory 惯例打桩。
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { SchedulingSection } from './SchedulingSection'

const mocks = vi.hoisted(() => ({
  KeepWarmGet: vi.fn(),
  KeepWarmSet: vi.fn(),
  PreloadPlanGet: vi.fn(),
  PreloadPlanSet: vi.fn(),
  getHerdsmanCatalog: vi.fn(),
  getEngineFailover: vi.fn(),
  setEngineFailover: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', () => ({
  app: {
    KeepWarmGet: mocks.KeepWarmGet,
    KeepWarmSet: mocks.KeepWarmSet,
    PreloadPlanGet: mocks.PreloadPlanGet,
    PreloadPlanSet: mocks.PreloadPlanSet,
  },
}))

vi.mock('../../api/engines', () => ({
  getHerdsmanCatalog: mocks.getHerdsmanCatalog,
  getEngineFailover: mocks.getEngineFailover,
  setEngineFailover: mocks.setEngineFailover,
}))

/** 按「故障转移」标题定位所在开关卡容器 */
function failoverCard(): HTMLElement {
  const el = screen.getByText('故障转移').closest('.mc-bind-card')
  expect(el).toBeTruthy()
  return el as HTMLElement
}

describe('SchedulingSection · 故障转移开关卡（C 刀）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.KeepWarmGet.mockResolvedValue(true)
    mocks.PreloadPlanGet.mockResolvedValue(false)
    mocks.getHerdsmanCatalog.mockResolvedValue({ models: [], running: 1 })
  })

  it('读取后渲染开关状态：开启 → 「已开启」芯片 + switch 勾选可用', async () => {
    mocks.getEngineFailover.mockResolvedValue(true)
    render(<SchedulingSection />)
    const card = await waitFor(() => {
      const c = failoverCard()
      expect(c.querySelector('.mc-chip')?.textContent).toContain('已开启')
      return c
    })
    const sw = card.querySelector('button[role="switch"]')
    expect(sw?.getAttribute('aria-checked')).toBe('true')
    expect(sw?.hasAttribute('disabled')).toBe(false)
  })

  it('切换开关调用 setEngineFailover(true)，成功后芯片更新为「已开启」', async () => {
    mocks.getEngineFailover.mockResolvedValue(false)
    render(<SchedulingSection />)
    const sw = await waitFor(() => {
      const s = failoverCard().querySelector('button[role="switch"]')
      expect(s).toBeTruthy()
      return s as HTMLElement
    })
    fireEvent.click(sw)
    await waitFor(() => {
      expect(mocks.setEngineFailover).toHaveBeenCalledWith(true)
    })
    await waitFor(() => {
      expect(failoverCard().querySelector('.mc-chip')?.textContent).toContain('已开启')
    })
  })

  it('读取失败显示「未知」并禁用开关（keepWarm===null 先例）', async () => {
    mocks.getEngineFailover.mockRejectedValue(new Error('backend unavailable'))
    render(<SchedulingSection />)
    await waitFor(() => {
      expect(failoverCard().querySelector('.mc-chip')?.textContent).toContain('未知')
    })
    const sw = failoverCard().querySelector('button[role="switch"]')
    expect(sw?.hasAttribute('disabled')).toBe(true)
  })

  // B 线欠账「dev mock 补 Get/SetEngineFailover」：mock 可读布尔后开关不再
  // 恒「未知」禁用——false（默认关）同样渲染为可切换的「已关闭」卡。
  it('mock 返回 false 时开关可用：「已关闭」芯片 + switch 未禁用未勾选', async () => {
    mocks.getEngineFailover.mockResolvedValue(false)
    render(<SchedulingSection />)
    const sw = await waitFor(() => {
      const s = failoverCard().querySelector('button[role="switch"]')
      expect(s).toBeTruthy()
      return s as HTMLElement
    })
    expect(failoverCard().querySelector('.mc-chip')?.textContent).toContain('已关闭')
    expect(sw.hasAttribute('disabled')).toBe(false)
    expect(sw.getAttribute('aria-checked')).toBe('false')
  })
})
