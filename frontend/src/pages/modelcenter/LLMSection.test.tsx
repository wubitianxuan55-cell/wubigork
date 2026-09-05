/**
 * LLMSection.test.tsx — B 刀「模型元数据徽标」渲染测试
 *
 * 有 meta 的模型显示上下文/能力/价格徽标（free 走 ok 绿色）；
 * 无 meta 的模型不显示徽标也不占位。
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ModelCenterContext, type ModelCenterContextValue } from './context'
import { LLMSection } from './LLMSection'
import type { ModelCardData } from './utils'

const card = (over: Partial<ModelCardData>): ModelCardData => ({
  modelId: 'glm-5.3',
  modelName: 'glm-5.3',
  engineId: 'glm',
  engineName: 'GLM 云端',
  engineType: 'glm',
  engineEnabled: true,
  status: 'running',
  ...over,
})

function renderLLM(models: ModelCardData[], over: Record<string, unknown> = {}) {
  const value = {
    engines: [
      { id: 'glm', name: 'GLM', type: 'glm', enabled: true, base_url: '', default_model: '', models: [] },
    ],
    llmModels: models,
    engineStatuses: {},
    testingEngine: null,
    hubLoadingIds: [] as string[],
    handleTestConnection: vi.fn(),
    handleRefreshModels: vi.fn(),
    handleStartModel: vi.fn(),
    isModelActive: () => false,
    ...over,
  } as unknown as ModelCenterContextValue
  return render(
    <ModelCenterContext.Provider value={value}>
      <LLMSection />
    </ModelCenterContext.Provider>,
  )
}

/** 取指定模型卡片的芯片文案列表（引擎名 + kindChip 为基线，元数据芯片在其后） */
function chipTexts(root: HTMLElement, modelName: string): string[] {
  const el = Array.from(root.querySelectorAll('.mc-model-card'))
    .find(c => c.querySelector('.mc-model-name')?.textContent === modelName)
  expect(el).toBeTruthy()
  return Array.from(el!.querySelectorAll('.mc-chip')).map(c => c.textContent || '')
}

describe('LLMSection · 模型元数据徽标（B 刀）', () => {
  it('有 meta 的模型显示上下文/能力徽标，free 价格走绿色「免费」', () => {
    renderLLM([card({ meta: { context_length: 200_000, free: true, caps: ['vision', 'tools'] } })])
    expect(screen.getByText('200K')).toBeTruthy()
    expect(screen.getByText('视觉')).toBeTruthy()
    expect(screen.getByText('工具')).toBeTruthy()
    const freeChip = screen.getByText('免费').closest('.mc-chip')
    expect(freeChip?.className).toContain('is-ok')
    expect(screen.getByText('200K').closest('.mc-chip')?.getAttribute('title')).toBe('上下文长度')
  })

  it('计价模型显示组合价格徽标，price_note 进 title', () => {
    renderLLM([card({ meta: { price_in: 1.4, price_out: 4.4, currency: 'CNY', price_note: '以官网为准' } })])
    const chip = screen.getByText('¥1.4·¥4.4/M').closest('.mc-chip')
    expect(chip).toBeTruthy()
    expect(chip!.getAttribute('title')).toBe('以官网为准')
  })

  it('未收录的能力键回退原文透传（search/json 契约键直接显示）', () => {
    renderLLM([card({ meta: { caps: ['search', 'json'] } })])
    expect(screen.getByText('搜索')).toBeTruthy()
    expect(screen.getByText('结构化')).toBeTruthy()
  })

  it('无 meta 的模型不显示元数据徽标也不占位（只有类型芯片）', () => {
    const { container } = renderLLM([
      card({ modelId: 'plain-model', modelName: 'plain-model' }),
      card({ meta: { context_length: 8_192 } }),
    ])
    expect(chipTexts(container, 'plain-model')).toEqual(['GLM 云端'])
    expect(chipTexts(container, 'glm-5.3')).toEqual(['GLM 云端', '8K'])
  })
})

describe('LLMSection · MH2/MH3（modelhub 加载态与 UD 档位）', () => {
  const hubCard = (over: Partial<ModelCardData>): ModelCardData => card({
    engineId: 'modelhub', engineName: 'Model Hub 本地', ...over,
  })

  function renderHub(models: ModelCardData[], over: Record<string, unknown> = {}) {
    return renderLLM(models, {
      engines: [
        { id: 'modelhub', name: 'Model Hub', type: 'modelhub', enabled: true, base_url: '', default_model: '', models: [] },
      ],
      ...over,
    })
  }

  it('UD 变体显示「推荐」徽标并置顶，普通量化名不显示', () => {
    const { container } = renderHub([
      hubCard({ modelId: 'qwen3.6:Q4_K_P', modelName: 'Qwen3.6:Q4_K_P' }),
      hubCard({ modelId: 'qwen3.6:UD-Q4_K_XL', modelName: 'Qwen3.6:UD-Q4_K_XL' }),
    ])
    expect(screen.getByText('推荐').closest('.mc-chip')?.getAttribute('title')).toContain('动态量化')
    const cards = Array.from(container.querySelectorAll('.mc-model-name')).map(el => el.textContent)
    expect(cards[0]).toBe('Qwen3.6:UD-Q4_K_XL')
    expect(chipTexts(container, 'Qwen3.6:Q4_K_P')).toEqual(['Model Hub 本地'])
  })

  it('加载中模型：状态与按钮显示「加载中」且按钮禁用', () => {
    renderHub(
      [hubCard({ modelId: 'qwen3.6:UD-Q4_K_XL', modelName: 'Qwen3.6:UD-Q4_K_XL', status: 'stopped' })],
      { hubLoadingIds: ['qwen3.6:UD-Q4_K_XL'] },
    )
    expect(screen.getAllByText('加载中').length).toBeGreaterThanOrEqual(1)
    // antd 按钮可访问名称含图标 aria-label（"loading 加载中"），用正则匹配文字部分
    const btn = screen.getByRole('button', { name: /加载中/ }) as HTMLButtonElement
    expect(btn.disabled).toBe(true)
  })

  it('非 modelhub 引擎的 UD 字样不触发「推荐」徽标（作用域守护）', () => {
    renderLLM([card({ modelId: 'SUD-Model', modelName: 'SUD-Model' })])
    expect(screen.queryByText('推荐')).toBeNull()
  })
})
