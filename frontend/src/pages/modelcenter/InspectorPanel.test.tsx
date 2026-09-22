/**
 * InspectorPanel.test.tsx — 右栏统计块「遥测读出式」重设计测试
 *
 * 三态（有数据/空态/错误）与右栏→抽屉联动入口：
 * - 有数据：三格主指标 + Token 读数行 + 请求迷你图（窄柱专用 svg）+ 「查看详细统计」按钮
 * - 空态（total_calls=0）：走 EmptyState，不渲染读数区
 * - 入口按钮：点击调 openStatsDrawer（context 动作，v4.389 候选新增）
 * ResourceMonitor 打桩：与统计块无关（真实实现轮询 GetModelMonitor）。
 */
import { fireEvent, render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ModelCenterActionsContext, ModelCenterContext, ModelCenterStateContext, type ModelCenterContextValue } from './context'
import { InspectorPanel } from './InspectorPanel'
import type { ModelStatsSummary } from '../../api/engines'
import type { TrendDatum } from './charts'

vi.mock('./ResourceMonitor', () => ({ ResourceMonitor: () => null }))

const stats = (over: Partial<ModelStatsSummary> = {}): ModelStatsSummary => ({
  total_calls: 1234,
  success_calls: 1210,
  fail_calls: 24,
  total_tokens: 123456,
  input_tokens: 100234,
  output_tokens: 23222,
  total_duration_ms: 600000,
  avg_duration_ms: 486,
  total_cost: 12.34,
  trend: [],
  per_model: [{ engine_id: 'glm', model: 'glm-4.7', call_count: 1234, success_count: 1210, fail_count: 24, input_tokens: 1, output_tokens: 1, total_tokens: 2, total_duration_ms: 1 }],
  ...over,
})

const trend: TrendDatum[] = [
  { key: 'd1', label: '09-20', calls: 10, successCalls: 10, failCalls: 0, inputTokens: 100, outputTokens: 20, cost: 0 },
  { key: 'd2', label: '09-21', calls: 45, successCalls: 40, failCalls: 5, inputTokens: 400, outputTokens: 80, cost: 0 },
  { key: 'd3', label: '09-22', calls: 30, successCalls: 30, failCalls: 0, inputTokens: 300, outputTokens: 60, cost: 0 },
]

function renderInspector(overrides: { state?: Partial<ModelCenterContextValue>; actions?: Partial<ModelCenterContextValue> } = {}) {
  const openStatsDrawer = vi.fn()
  const value = {
    callStats: stats(),
    trendData: trend,
    trendRange: '7d',
    loadError: null,
    engines: [],
    setTrendRange: vi.fn(),
    loadCallStats: vi.fn(),
    handleResetCallStats: vi.fn(),
    openStatsDrawer,
    ...overrides.state,
    ...overrides.actions,
  } as unknown as ModelCenterContextValue
  const utils = render(
    <ModelCenterStateContext.Provider value={value}><ModelCenterActionsContext.Provider value={value}><ModelCenterContext.Provider value={value}>
      <InspectorPanel />
    </ModelCenterContext.Provider></ModelCenterActionsContext.Provider></ModelCenterStateContext.Provider>,
  )
  return { ...utils, openStatsDrawer }
}

describe('InspectorPanel · 右栏统计块（遥测读出式重设计）', () => {
  it('有数据：三格主指标 + Token 读数行 + 两张窄柱迷你图', () => {
    const { container } = renderInspector()
    const hero = container.querySelector('.mc-stat-hero')
    expect(hero).toBeTruthy()
    const values = Array.from(container.querySelectorAll('.mc-stat-value')).map(el => el.textContent)
    expect(values[0]).toBe('1,234')
    expect(values[1]).toBe('98.1%')
    expect(values[2]).toBe('¥12.34')
    const strip = container.querySelector('.mc-stat-strip')
    expect(strip?.textContent).toContain('Token')
    expect(strip?.textContent).toContain('入 100.2k')
    expect(strip?.textContent).toContain('出 23.2k')
    // 迷你图按窄柱视口设计，替代压扁的全尺寸图
    expect(container.querySelector('svg[aria-label="请求趋势迷你图"]')).toBeTruthy()
    expect(container.querySelector('svg[aria-label="Token 分布迷你图"]')).toBeTruthy()
    // 峰值读数来自 trendData
    expect(container.querySelector('.mc-mini-chart-meta')?.textContent).toContain('峰值 45')
  })

  it('「查看详细统计」入口调 openStatsDrawer（右栏 → 抽屉联动）', () => {
    const { container, openStatsDrawer } = renderInspector()
    const btn = container.querySelector<HTMLButtonElement>('.mc-stat-more')
    expect(btn).toBeTruthy()
    fireEvent.click(btn!)
    expect(openStatsDrawer).toHaveBeenCalledTimes(1)
  })

  it('空态（total_calls=0）：读数区不渲染，走 EmptyState', () => {
    const { container } = renderInspector({ state: { callStats: stats({ total_calls: 0, success_calls: 0, fail_calls: 0, per_model: [] }), trendData: [] } })
    expect(container.querySelector('.mc-stat-hero')).toBeNull()
    expect(container.querySelector('svg[aria-label="请求趋势迷你图"]')).toBeNull()
    expect(container.textContent).toContain('暂无调用记录')
  })

  it('加载失败：错误条 + 重试按钮', () => {
    const { container } = renderInspector({ state: { loadError: 'boom' } })
    const err = container.querySelector('.mc-inspector-error')
    expect(err?.textContent).toContain('boom')
    // antd Button 对两字中文标签自动插空格（「重 试」），按按钮存在性断言
    expect(err?.querySelector('button')).toBeTruthy()
  })
})
