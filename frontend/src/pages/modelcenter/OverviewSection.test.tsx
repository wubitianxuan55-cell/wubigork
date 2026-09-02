/**
 * OverviewSection.test.tsx — C 刀「健康巡检」引擎卡状态测试
 *
 * 后端连续 ≥3 次探测失败时 error 带「连续 N 次探测失败」前缀 → 卡片状态
 * tone 用 danger 且 text 显示该 error；last_checked 存在 → 状态行 title
 * 提示「巡检 HH:MM」（克制，不占版面）。写法照 LLMSection.test.tsx。
 */
import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ModelCenterContext, type ModelCenterContextValue } from './context'
import { OverviewSection } from './OverviewSection'
import type { EngineStatus } from '../../api/engines'

function renderOverview(engineStatuses: Record<string, EngineStatus>) {
  const value = {
    engines: [
      { id: 'glm', name: 'GLM', type: 'glm', label: 'GLM 云端', enabled: true, base_url: '', default_model: '', models: [] },
      // 禁用引擎不进卡片网格
      { id: 'ollama', name: 'Ollama', type: 'ollama', enabled: false, base_url: '', default_model: '', models: [] },
    ],
    engineStatuses,
    callStats: null,
    setCategory: vi.fn(),
    activeEngine: '',
  } as unknown as ModelCenterContextValue
  return render(
    <ModelCenterContext.Provider value={value}>
      <OverviewSection />
    </ModelCenterContext.Provider>,
  )
}

const st = (over: Partial<EngineStatus>): EngineStatus => ({
  id: 'glm',
  connected: false,
  model_count: 0,
  error: '',
  ...over,
})

describe('OverviewSection · 引擎卡巡检状态（C 刀）', () => {
  it('error 含「连续」前缀 → danger tone 且 text 显示错误原文', () => {
    const { container } = renderOverview({
      glm: st({ error: '连续 3 次探测失败：connection refused' }),
    })
    const el = container.querySelector('.mc-model-card .mc-status.is-danger')
    expect(el).toBeTruthy()
    expect(el!.textContent).toContain('连续 3 次探测失败')
    expect(el!.textContent).toContain('connection refused')
  })

  it('last_checked 存在 → 状态行 title 提示「巡检 HH:MM」', () => {
    const { container } = renderOverview({
      glm: st({ connected: true, model_count: 5, last_checked: '2026-09-02T10:30:00' }),
    })
    const el = container.querySelector('.mc-model-card .mc-status.is-ok')
    expect(el).toBeTruthy()
    expect(el!.textContent).toContain('连接正常 · 5 个模型')
    expect(el!.querySelector('span')?.getAttribute('title')).toBe('巡检 10:30')
  })

  it('last_checked 缺失 → title 不渲染（不占位）', () => {
    const { container } = renderOverview({
      glm: st({ connected: true, model_count: 2 }),
    })
    const el = container.querySelector('.mc-model-card .mc-status.is-ok span')
    expect(el?.getAttribute('title') ?? '').toBe('')
  })
})
