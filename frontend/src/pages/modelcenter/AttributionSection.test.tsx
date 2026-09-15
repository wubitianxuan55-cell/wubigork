/**
 * AttributionSection.test.tsx — 模型中心「成本归因」分区测试（阶段七 7.1-2 C 线）
 *
 * 断言焦点：KPI 行与功能分组表（含「未标记」桶排最后/无数据功能键空态）、
 * 建议卡渲染与采纳闭环（apply 调用 + 重拉后卡消失 + 「已改绑」提示）、
 * 忽略闭环（ignore 调用 + 卡消失）、读取失败诚实错误条 + 重试恢复、全空空态。
 * api/engines.ts 新增四函数按既有 mock factory 惯例打桩。
 */
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { message } from 'antd'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AttributionSection } from './AttributionSection'

const mocks = vi.hoisted(() => ({
  getRouteLedger: vi.fn(),
  getRouteSuggestions: vi.fn(),
  applyRouteSuggestion: vi.fn(),
  ignoreRouteSuggestion: vi.fn(),
}))

vi.mock('../../api/engines', () => ({
  getRouteLedger: mocks.getRouteLedger,
  getRouteSuggestions: mocks.getRouteSuggestions,
  applyRouteSuggestion: mocks.applyRouteSuggestion,
  ignoreRouteSuggestion: mocks.ignoreRouteSuggestion,
}))

const ledger = {
  generated_at: '2026-09-15T10:00:00Z',
  total: { calls: 100, tokens_in: 12000, tokens_out: 8000, cost_cny: 12.34, avg_ms: 1500, success_rate: 0.95 },
  features: [
    { feature: 'chat', engine_id: 'glm', model: 'glm-5.3', calls: 60, tokens_in: 10000, tokens_out: 6000, cost_cny: 9.5, avg_ms: 1200, success_rate: 0.98 },
    { feature: 'chat', engine_id: 'ollama', model: 'qwen3-8b', calls: 10, tokens_in: 500, tokens_out: 300, cost_cny: 0, avg_ms: 3000, success_rate: 1 },
    { feature: 'novel', engine_id: 'herdsman', model: 'glm-air-local', calls: 5, tokens_in: 1000, tokens_out: 500, cost_cny: 0, avg_ms: 800, success_rate: 1 },
    // 「未标记」桶：升级前无功能维度的历史数据（feature=""）
    { feature: '', engine_id: 'xai', model: 'grok-4.20', calls: 40, tokens_in: 2000, tokens_out: 2000, cost_cny: 2.84, avg_ms: 2100, success_rate: 0.9 },
  ],
}

const suggestion = {
  id: 'novel:glm/glm-5.3>herdsman/glm-air-local',
  feature: 'novel',
  reason: '本地候选质量与成本综合评分更高',
  evidence: '成功率 0.98 vs 0.95；成本 ¥0 vs ¥9.5（近 7 天 60 次调用）',
  from: { engine_id: 'glm', model: 'glm-5.3' },
  to: { engine_id: 'herdsman', model: 'glm-air-local', is_local: true, success_rate: 0.98, cost_cny: 0 },
  score_gap: 0.35,
}

describe('模型中心「成本归因」分区', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getRouteLedger.mockResolvedValue(ledger)
    mocks.getRouteSuggestions.mockResolvedValue({ generated_at: '2026-09-15T10:00:00Z', suggestions: [] })
  })

  it('渲染 KPI 行与功能分组表：已知功能键空态常显、「未标记」桶排最后、组内按成本降序', async () => {
    render(<AttributionSection />)
    await waitFor(() => expect(screen.getByTestId('mc-attribution-kpi')).toBeTruthy())

    // KPI：总成本（两位小数）/ 总 token（入+出千分位）/ 调用 / 成功率
    const kpi = screen.getByTestId('mc-attribution-kpi').textContent || ''
    expect(kpi).toContain('¥12.34')
    expect(kpi).toContain((20000).toLocaleString())
    expect(kpi).toContain('100')
    expect(kpi).toContain('95.0%')

    // 功能分组：聊天（2 行，glm 成本高排前）、小说、无数据功能键空态、未标记桶
    const chat = screen.getByTestId('mc-attribution-row-chat')
    expect(chat.textContent).toContain('GLM 云端')
    expect(chat.textContent).toContain('glm-5.3')
    expect(chat.textContent).toContain('¥9.50')
    const chatRows = within(chat).getAllByRole('row')
    expect(chatRows[1].textContent).toContain('glm-5.3') // 组内成本降序：¥9.5 的 glm 在首行
    expect(chatRows[2].textContent).toContain('qwen3-8b')

    expect(screen.getByTestId('mc-attribution-row-novel').textContent).toContain('Herdsman 本地')
    expect(screen.getByTestId('mc-attribution-row-office').textContent).toContain('暂无调用记录')

    // 「未标记」桶存在且排在所有功能节最后
    const unmarked = screen.getByTestId('mc-attribution-row-unmarked')
    expect(unmarked.textContent).toContain('未标记')
    expect(unmarked.textContent).toContain('grok-4.20')
    const sections = document.querySelectorAll('[data-testid^="mc-attribution-row-"]')
    expect(sections[sections.length - 1]).toBe(unmarked)

    // 成功率/均时长列渲染
    expect(chat.textContent).toContain('98.0%')
    expect(chat.textContent).toContain(`${(1200).toLocaleString()} ms`)
  })

  it('建议卡渲染 + 采纳闭环：apply 调用 → 重拉后卡消失 → 「已改绑」提示', async () => {
    const successSpy = vi.spyOn(message, 'success')
    mocks.getRouteSuggestions
      .mockResolvedValueOnce({ generated_at: '2026-09-15T10:00:00Z', suggestions: [suggestion] })
      .mockResolvedValue({ generated_at: '2026-09-15T10:00:00Z', suggestions: [] })
    render(<AttributionSection />)

    const card = await screen.findByTestId('mc-suggest-novel')
    expect(card.textContent).toContain('评分差 +0.35')
    expect(card.textContent).toContain(suggestion.reason)
    expect(card.textContent).toContain(suggestion.evidence)
    expect(card.textContent).toContain('Herdsman 本地 · glm-air-local')
    expect(card.textContent).toContain('本地')

    fireEvent.click(screen.getByTestId('mc-suggest-novel-apply'))
    await waitFor(() => expect(mocks.applyRouteSuggestion).toHaveBeenCalledWith(suggestion.id))
    await waitFor(() => expect(mocks.getRouteSuggestions).toHaveBeenCalledTimes(2)) // 动作后重拉建议
    await waitFor(() => expect(screen.queryByTestId('mc-suggest-novel')).toBeNull()) // 卡消失
    expect(successSpy).toHaveBeenCalledWith('已改绑')
  })

  it('忽略闭环：ignore 调用 → 重拉后卡消失', async () => {
    mocks.getRouteSuggestions
      .mockResolvedValueOnce({ generated_at: '2026-09-15T10:00:00Z', suggestions: [suggestion] })
      .mockResolvedValue({ generated_at: '2026-09-15T10:00:00Z', suggestions: [] })
    render(<AttributionSection />)

    await screen.findByTestId('mc-suggest-novel')
    fireEvent.click(screen.getByTestId('mc-suggest-novel-ignore'))
    await waitFor(() => expect(mocks.ignoreRouteSuggestion).toHaveBeenCalledWith(suggestion.id))
    await waitFor(() => expect(screen.queryByTestId('mc-suggest-novel')).toBeNull())
  })

  it('读取失败 → 诚实错误条，重试成功后恢复账本视图', async () => {
    mocks.getRouteLedger.mockRejectedValueOnce(new Error('绑定不可用（旧后端）'))
    render(<AttributionSection />)

    const err = await screen.findByTestId('mc-attribution-error')
    expect(err.textContent).toContain('绑定不可用（旧后端）')

    fireEvent.click(screen.getByText('重试'))
    await waitFor(() => expect(screen.getByTestId('mc-attribution-kpi')).toBeTruthy())
    expect(screen.queryByTestId('mc-attribution-error')).toBeNull()
  })

  it('全空 → EmptyState 空态，不渲染 KPI 行', async () => {
    mocks.getRouteLedger.mockResolvedValue({
      generated_at: '2026-09-15T10:00:00Z',
      total: { calls: 0, tokens_in: 0, tokens_out: 0, cost_cny: 0, avg_ms: 0, success_rate: 0 },
      features: [],
    })
    render(<AttributionSection />)

    await waitFor(() => expect(screen.getByText('暂无归因数据')).toBeTruthy())
    expect(screen.queryByTestId('mc-attribution-kpi')).toBeNull()
    expect(screen.queryByTestId('mc-attribution-row-unmarked')).toBeNull()
  })
})
