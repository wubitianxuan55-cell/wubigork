import { describe, expect, it, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useStatsState } from './useStatsState'

const { getStatsMock } = vi.hoisted(() => ({
  getStatsMock: vi.fn(),
}))

vi.mock('../../../api/engines', () => ({
  getModelCallStats: getStatsMock,
  resetModelCallStats: vi.fn().mockResolvedValue(undefined),
}))

function hourKey(date: Date, h: number): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(h)}:00`
}

describe('useStatsState 趋势聚合', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('7天模式按天聚合，标签只显示日期', async () => {
    const today = new Date()
    const yesterday = new Date(Date.now() - 24 * 3600 * 1000)
    getStatsMock.mockResolvedValue({
      total_calls: 10,
      success_calls: 10,
      fail_calls: 0,
      total_tokens: 1000,
      input_tokens: 800,
      output_tokens: 200,
      total_duration_ms: 100,
      avg_duration_ms: 10,
      total_cost: 0,
      trend: [
        { time: hourKey(yesterday, 9), calls: 1, success_calls: 1, fail_calls: 0, input_tokens: 100, output_tokens: 10, total_tokens: 110, cost: 0 },
        { time: hourKey(yesterday, 10), calls: 1, success_calls: 1, fail_calls: 0, input_tokens: 200, output_tokens: 20, total_tokens: 220, cost: 0 },
        { time: hourKey(today, 9), calls: 1, success_calls: 1, fail_calls: 0, input_tokens: 300, output_tokens: 30, total_tokens: 330, cost: 0 },
      ],
      per_model: [],
    })
    const { result } = renderHook(() => useStatsState(false))
    await waitFor(() => expect(result.current.callStats?.total_calls).toBe(10))

    expect(result.current.trendData.length).toBe(2) // 昨天 + 今天各一条
    expect(result.current.trendData[0].inputTokens).toBe(300) // 昨天两小时聚合
    expect(result.current.trendData[0].label).toMatch(/^\d{2}-\d{2}$/) // 只显示日期
  })

  it('今日模式只保留当天的小时桶', async () => {
    const today = new Date()
    const yesterday = new Date(Date.now() - 24 * 3600 * 1000)
    getStatsMock.mockResolvedValue({
      total_calls: 3,
      success_calls: 3,
      fail_calls: 0,
      total_tokens: 100,
      input_tokens: 80,
      output_tokens: 20,
      total_duration_ms: 30,
      avg_duration_ms: 10,
      total_cost: 0,
      trend: [
        { time: hourKey(yesterday, 10), calls: 1, success_calls: 1, fail_calls: 0, input_tokens: 10, output_tokens: 1, total_tokens: 11, cost: 0 },
        { time: hourKey(today, 9), calls: 1, success_calls: 1, fail_calls: 0, input_tokens: 40, output_tokens: 4, total_tokens: 44, cost: 0 },
        { time: hourKey(today, 10), calls: 1, success_calls: 1, fail_calls: 0, input_tokens: 50, output_tokens: 5, total_tokens: 55, cost: 0 },
      ],
      per_model: [],
    })
    const { result } = renderHook(() => useStatsState(false))
    await waitFor(() => expect(result.current.callStats?.total_calls).toBe(3))

    act(() => result.current.setTrendRange('today'))
    expect(result.current.trendData.length).toBe(2) // 只有今天两小时
    expect(result.current.trendData[0].label).toMatch(/^\d{2}:\d{2}$/)
  })
})
