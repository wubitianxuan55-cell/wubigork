/**
 * StatsSection.test.tsx — 「详细统计」抽屉重设计测试
 *
 * - 工具条：口径元信息芯片（统计自 since / 价格目录 catalog_version），不再有重复大标题
 * - KPI 带：五格语义化图标 + 成功率按阈值着色（100% → 成功色）
 * - 云端/本地分流带：占比条 aria-label 按口算占比、KV 命中率紧凑行（getUsageOverview 打桩）
 * - 空态：total_calls=0 走 EmptyState
 */
import { render, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ModelCenterActionsContext, ModelCenterContext, ModelCenterStateContext, type ModelCenterContextValue } from './context'
import { StatsSection } from './StatsSection'
import { getUsageOverview, type ModelStatsSummary, type UsageOverview } from '../../api/engines'

vi.mock('../../api/engines', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../api/engines')>()
  return { ...actual, getUsageOverview: vi.fn() }
})

const stats = (over: Partial<ModelStatsSummary> = {}): ModelStatsSummary => ({
  total_calls: 100,
  success_calls: 100,
  fail_calls: 0,
  total_tokens: 50000,
  input_tokens: 40000,
  output_tokens: 10000,
  total_duration_ms: 300000,
  avg_duration_ms: 3000,
  total_cost: 3.5,
  trend: [],
  per_model: [{ engine_id: 'glm', model: 'glm-4.7', call_count: 100, success_count: 100, fail_count: 0, input_tokens: 4, output_tokens: 1, total_tokens: 5, total_duration_ms: 3 }],
  since: '2026-09-01T14:22:00+08:00',
  catalog_version: 'v3',
  catalog_source: 'bigmodel',
  ...over,
})

const overview: UsageOverview = {
  cloud: { calls: 12, success_calls: 12, fail_calls: 0, input_tokens: 8000, output_tokens: 2000, total_tokens: 10000, total_duration_ms: 1, cost: 0.5, engines: ['glm'] },
  local: { calls: 30, success_calls: 30, fail_calls: 0, input_tokens: 50000, output_tokens: 10000, total_tokens: 60000, total_duration_ms: 1, cost: 0, engines: ['herdsman'] },
  savings: { ref_price_per_mtok: 8, would_cost_cloud: 0.48, saved: 0.48, note: '本地分流节省说明' },
  cache_hit_tokens: 100,
  cache_miss_tokens: 100,
  cache_hit_rate: 0.5,
}

function renderStats(over: Partial<ModelCenterContextValue> = {}) {
  const value = {
    engines: [],
    callStats: stats(),
    statsSort: 'calls',
    trendRange: '7d',
    trendData: [],
    loadError: null,
    setStatsSort: vi.fn(),
    setTrendRange: vi.fn(),
    loadCallStats: vi.fn(),
    handleResetCallStats: vi.fn(),
    openStatsDrawer: vi.fn(),
    ...over,
  } as unknown as ModelCenterContextValue
  return render(
    <ModelCenterStateContext.Provider value={value}><ModelCenterActionsContext.Provider value={value}><ModelCenterContext.Provider value={value}>
      <StatsSection />
    </ModelCenterContext.Provider></ModelCenterActionsContext.Provider></ModelCenterStateContext.Provider>,
  )
}

describe('StatsSection · 详细统计抽屉（重设计）', () => {
  it('工具条渲染口径元信息芯片（since / 价格目录），不重复渲染块标题', () => {
    const { container } = renderStats()
    const chips = Array.from(container.querySelectorAll('.mc-meta-chip')).map(el => el.textContent)
    expect(chips.some(c => c!.includes('2026-09-01'))).toBe(true)
    expect(chips.some(c => c!.includes('bigmodel') && c!.includes('v3'))).toBe(true)
    expect(container.querySelector('.mc-stats-toolbar')).toBeTruthy()
  })

  it('KPI 带：五格 + 成功率 100% 着成功色 + Token hint 紧凑格式（窄卡不截断）', () => {
    const { container } = renderStats()
    const labels = Array.from(container.querySelectorAll('.mc-kpi-label')).map(el => el.textContent)
    expect(labels).toEqual(['总调用', 'Token 用量', '估算费用', '成功率', '平均耗时'])
    const rate = container.querySelectorAll('.mc-kpi-value')[3]
    expect(rate?.textContent).toBe('100.0%')
    expect((rate?.firstElementChild as HTMLElement)?.style.color).toBe('var(--mc-ok)')
    // 入/出改 k/M 缩写（旧全数字 hint 在 KPI 卡内被省略号截断）
    const tokenHint = container.querySelectorAll('.mc-kpi-hint')[1]
    expect(tokenHint?.textContent).toBe('入 40.0k / 出 10.0k')
  })

  it('云端/本地分流带：占比条按 token 口算占比 + KV 命中率紧凑行', async () => {
    ;(getUsageOverview as ReturnType<typeof vi.fn>).mockResolvedValue(overview)
    const { container } = renderStats()
    await waitFor(() => {
      expect(container.querySelector('.mc-splitbar')).toBeTruthy()
    })
    // 10000 / 70000 ≈ 14%
    expect(container.querySelector('.mc-splitbar')?.getAttribute('aria-label')).toContain('云端 14%')
    expect(container.querySelector('.mc-splitbar')?.getAttribute('aria-label')).toContain('本地 86%')
    const cacheRow = container.querySelector('.mc-cache-row')
    expect(cacheRow?.textContent).toContain('全局 50.0%')
    expect(cacheRow?.textContent).toContain('云端')
    expect(cacheRow?.textContent).toContain('本地')
    // 三列读出（云端费用 / 本地用量 / 已节省；图标在 label 内，textContent 带前导空格）
    const colLabels = Array.from(container.querySelectorAll('.mc-split-col-label')).map(el => el.textContent?.trim())
    expect(colLabels).toEqual(['云端费用', '本地用量', '已节省'])
  })

  it('空态（total_calls=0）：EmptyState 且不渲染 KPI 带', () => {
    const { container } = renderStats({ callStats: stats({ total_calls: 0, success_calls: 0, per_model: [] }) })
    expect(container.querySelector('.mc-overview-grid')).toBeNull()
    expect(container.textContent).toContain('暂无调用记录')
  })
})
