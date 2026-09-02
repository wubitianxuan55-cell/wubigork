import { describe, expect, it } from 'vitest'
import {
  COMFY_IMAGE_MODELS,
  capLabels,
  estimatePoints,
  findModelMeta,
  formatCtx,
  formatPrice,
  hasPointsCoef,
  imageModelDefaultFor,
  imageModelOptionsFor,
  isImageModel,
  modelAvailability,
  modelMeta,
  featureState,
  routeSourceLabel,
  filterModelsBySearch,
  sortModelsPinnedFirst,
  modelOptionsForEngine,
  filterEnginesByEnabled,
  glmEndpointFamily,
  glmAliasNote,
  billingModeLabel,
} from './utils'
import type { ModelCardData } from './utils'
import type { EngineConfig } from '../../api/engines'

const engines: EngineConfig[] = [
  {
    id: 'herdsman', name: 'Herdsman', type: 'herdsman', base_url: '', enabled: true, default_model: '',
    models: [
      { id: 'qwen3-8b', owned_by: 'local', status: 'running', kind: 'llm' },
      { id: 'flux-dev', owned_by: 'local', status: 'running', kind: 'image' },
      { id: 'bge-m3', owned_by: 'local', status: 'running', kind: 'embedding' },
    ],
  },
  {
    id: 'ollama', name: 'Ollama', type: 'ollama', base_url: '', enabled: false, default_model: '',
    models: [],
  },
]

describe('模型中心 utils', () => {
  it('xAI 后端只提供 Grok Imagine 系列模型', () => {
    const opts = imageModelOptionsFor('xai', engines)
    expect(opts.map(o => o.value)).toEqual(['grok-imagine-image-quality', 'grok-imagine-image'])
    expect(opts.some(o => o.value === 'krea2')).toBe(false)
  })

  it('ComfyUI 后端提供内置模型，并兜底保留自定义当前模型', () => {
    const opts = imageModelOptionsFor('comfyui', engines)
    expect(opts.map(o => o.value)).toEqual(COMFY_IMAGE_MODELS.map(m => m.modelId))

    const withCustom = imageModelOptionsFor('comfyui', engines, 'flux-workflow')
    expect(withCustom.some(o => o.value === 'flux-workflow')).toBe(true)
  })

  it('引擎后端只列出该引擎的图片模型', () => {
    const opts = imageModelOptionsFor('herdsman', engines)
    expect(opts.map(o => o.value)).toEqual(['flux-dev'])
  })

  it('引擎无图片模型时回退保留当前模型，避免表单悬空', () => {
    const opts = imageModelOptionsFor('ollama', engines, 'qwen-image')
    expect(opts).toEqual([{ value: 'qwen-image', label: 'qwen-image' }])
    expect(imageModelOptionsFor('ollama', engines)).toEqual([])
  })

  it('切换后端返回合理默认模型', () => {
    expect(imageModelDefaultFor('xai', engines)).toBe('grok-imagine-image-quality')
    expect(imageModelDefaultFor('comfyui', engines)).toBe('krea2')
    expect(imageModelDefaultFor('herdsman', engines)).toBe('flux-dev')
    expect(imageModelDefaultFor('ollama', engines)).toBe('')
  })

  it('isImageModel 优先使用后端 kind，缺失时回退名称启发式', () => {
    expect(isImageModel({ id: 'flux-dev', kind: 'image' })).toBe(true)
    expect(isImageModel({ id: 'qwen3-8b', kind: 'llm' })).toBe(false)
    expect(isImageModel({ id: 'krea2' })).toBe(true)
    expect(isImageModel({ id: 'grok-imagine-image-quality' })).toBe(true)
    expect(isImageModel({ id: 'grok-4.20' })).toBe(false)
  })
})

describe('模型中心 modelAvailability', () => {
  const card = (status: string): ModelCardData => ({
    modelId: 'm', modelName: 'M', engineId: 'e', engineName: 'E', engineType: 'xai', engineEnabled: true, status,
  })

  it('引擎禁用 → disabled', () => {
    expect(modelAvailability(card('running'), false, true)).toBe('disabled')
  })

  it('连接失败 → disconnected', () => {
    expect(modelAvailability(card('running'), true, false)).toBe('disconnected')
  })

  it('模型停止 → stopped', () => {
    expect(modelAvailability(card('stopped'), true, true)).toBe('stopped')
  })

  it('连接未知/正常 → ready', () => {
    expect(modelAvailability(card('running'), true, undefined)).toBe('ready')
    expect(modelAvailability(card('running'), true, true)).toBe('ready')
  })
})

describe('模型中心 featureState / routeSourceLabel', () => {
  it('绑定 + 启用 → bound-active', () => {
    expect(featureState(true, true)).toBe('bound-active')
  })

  it('绑定 + 停用 → bound-disabled', () => {
    expect(featureState(true, false)).toBe('bound-disabled')
  })

  it('未绑定 → fallback（无论启停）', () => {
    expect(featureState(false, true)).toBe('fallback')
    expect(featureState(false, false)).toBe('fallback')
  })

  it('路由来源文案映射', () => {
    expect(routeSourceLabel('feature')).toBe('功能绑定')
    expect(routeSourceLabel('global')).toBe('全局默认')
    expect(routeSourceLabel('fallback')).toBe('兜底')
    expect(routeSourceLabel(undefined)).toBe('-')
  })
})

describe('模型中心 搜索 / 置顶排序', () => {
  const models: ModelCardData[] = [
    { modelId: 'grok-4.20', modelName: 'Grok 4.20', engineId: 'xai', engineName: 'xAI', engineType: 'xai', engineEnabled: true, status: 'running' },
    { modelId: 'qwen3-8b', modelName: 'Qwen3 8B', engineId: 'herdsman', engineName: 'Herdsman', engineType: 'herdsman', engineEnabled: true, status: 'running' },
    { modelId: 'flux-dev', modelName: 'Flux Dev', engineId: 'herdsman', engineName: 'Herdsman', engineType: 'herdsman', engineEnabled: true, status: 'running' },
  ]

  it('filterModelsBySearch 按名称或 ID 匹配', () => {
    expect(filterModelsBySearch(models, 'qwen').map(m => m.modelId)).toEqual(['qwen3-8b'])
    expect(filterModelsBySearch(models, 'flux').map(m => m.modelId)).toEqual(['flux-dev'])
    expect(filterModelsBySearch(models, '')).toHaveLength(3)
  })

  it('sortModelsPinnedFirst 把置顶模型排前且不丢失其余', () => {
    const sorted = sortModelsPinnedFirst(models, ['flux-dev'])
    expect(sorted[0].modelId).toBe('flux-dev')
    expect(sorted.map(m => m.modelId).sort()).toEqual(['flux-dev', 'grok-4.20', 'qwen3-8b'])
  })
})

describe('模型中心 modelOptionsForEngine', () => {
  const models: ModelCardData[] = [
    { modelId: 'grok-4.20', modelName: 'Grok 4.20', engineId: 'xai', engineName: 'xAI', engineType: 'xai', engineEnabled: true, status: 'running' },
    { modelId: 'qwen3-8b', modelName: 'Qwen3 8B', engineId: 'herdsman', engineName: 'Herdsman', engineType: 'herdsman', engineEnabled: true, status: 'running' },
  ]

  it('只列出指定引擎的模型', () => {
    expect(modelOptionsForEngine('xai', models).map(o => o.value)).toEqual(['grok-4.20'])
  })

  it('当前模型不在候选时兜底补一条', () => {
    const opts = modelOptionsForEngine('xai', models, 'grok-4.6')
    expect(opts.some(o => o.value === 'grok-4.6')).toBe(true)
    expect(opts.length).toBe(2)
  })

  it('空引擎返回空列表', () => {
    expect(modelOptionsForEngine('', models)).toEqual([])
  })
})

describe('模型中心 filterEnginesByEnabled', () => {
  const engines: EngineConfig[] = [
    { id: 'xai', name: 'xAI', type: 'xai', base_url: '', enabled: true, default_model: '', models: [] },
    { id: 'ollama', name: 'Ollama', type: 'ollama', base_url: '', enabled: false, default_model: '', models: [] },
  ]

  it('onlyEnabled=true 只保留已启用引擎', () => {
    expect(filterEnginesByEnabled(engines, true).map(e => e.id)).toEqual(['xai'])
  })

  it('onlyEnabled=false 保留全部引擎', () => {
    expect(filterEnginesByEnabled(engines, false).map(e => e.id)).toEqual(['xai', 'ollama'])
  })
})

describe('模型中心 GLM 生图与端点家族', () => {
  const glmEngines: EngineConfig[] = [
    {
      id: 'glm', name: 'GLM (智谱)', type: 'glm',
      base_url: 'https://open.bigmodel.cn/api/paas/v4', enabled: true, default_model: 'glm-5.3',
      models: [
        { id: 'glm-5.3', owned_by: 'glm', status: 'running', kind: 'llm' },
        { id: 'glm-5-turbo', owned_by: 'glm', status: 'running', kind: 'llm' },
        { id: 'cogview-4-250304', owned_by: 'glm', status: 'running', kind: 'image' },
        { id: 'glm-image', owned_by: 'glm', status: 'running', kind: 'image' },
      ],
    },
  ]

  it('GLM 后端只列生图模型（glm-5-turbo 不混入）', () => {
    const opts = imageModelOptionsFor('glm', glmEngines)
    expect(opts.map(o => o.value)).toEqual(['cogview-4-250304', 'glm-image'])
  })

  it('GLM 后端默认取第一个生图模型', () => {
    expect(imageModelDefaultFor('glm', glmEngines)).toBe('cogview-4-250304')
  })

  it('glmEndpointFamily：coding 端点识别，其余归 std', () => {
    expect(glmEndpointFamily('https://open.bigmodel.cn/api/paas/v4')).toBe('std')
    expect(glmEndpointFamily('https://open.bigmodel.cn/api/coding/paas/v4')).toBe('coding')
    expect(glmEndpointFamily('')).toBe('std')
    expect(glmEndpointFamily(undefined)).toBe('std')
  })
})

describe('模型中心 glmAliasNote / billingModeLabel', () => {
  it('glmAliasNote：有 alias_of 返回切换说明，含调用名与实际模型', () => {
    expect(glmAliasNote({ id: 'glm-5.2', alias_of: 'glm-5.3' }))
      .toBe('服务端自动切换：调用 glm-5.2 实际按 glm-5.3 服务（编码套餐）')
    expect(glmAliasNote({ id: 'glm-5-turbo', alias_of: 'glm-5.3-flash' }))
      .toBe('服务端自动切换：调用 glm-5-turbo 实际按 glm-5.3-flash 服务（编码套餐）')
  })

  it('glmAliasNote：无 alias_of / 空串 / 缺参返回空', () => {
    expect(glmAliasNote({ id: 'glm-5.3' })).toBe('')
    expect(glmAliasNote({ id: 'glm-5.3', alias_of: '' })).toBe('')
    expect(glmAliasNote(undefined)).toBe('')
  })

  it('billingModeLabel：coding_points 返回积分口径标签', () => {
    expect(billingModeLabel('coding_points')).toBe('编码套餐 · 积分口径（不计价）')
  })

  it('billingModeLabel：空/未知口径返回空', () => {
    expect(billingModeLabel('')).toBe('')
    expect(billingModeLabel(undefined)).toBe('')
    expect(billingModeLabel('usage')).toBe('')
  })
})

// ── B 刀：模型元数据徽标 + 积分口径 ──────────────────────────

describe('模型中心 formatCtx（上下文徽标）', () => {
  it('≥1M 用 M 一位小数去尾 0', () => {
    expect(formatCtx(1_000_000)).toBe('1M')
    expect(formatCtx(1_500_000)).toBe('1.5M')
    expect(formatCtx(2_000_000)).toBe('2M')
  })

  it('≥1K 用 K 整数', () => {
    expect(formatCtx(200_000)).toBe('200K')
    expect(formatCtx(131_072)).toBe('131K')
    expect(formatCtx(8_192)).toBe('8K')
  })

  it('空值/非法值返回空串（徽标不占位）', () => {
    expect(formatCtx(undefined)).toBe('')
    expect(formatCtx(0)).toBe('')
    expect(formatCtx(-1)).toBe('')
    expect(formatCtx(Number.NaN)).toBe('')
  })
})

describe('模型中心 formatPrice（价格徽标）', () => {
  it('free → 免费', () => {
    expect(formatPrice({ free: true })).toBe('免费')
  })

  it('unit=call/minute 按次/分计价', () => {
    expect(formatPrice({ unit: 'call', price_in: 0.1, currency: 'CNY' })).toBe('¥0.1/次')
    expect(formatPrice({ unit: 'call', price_in: 0.02, currency: 'USD' })).toBe('$0.02/次')
    expect(formatPrice({ unit: 'minute', price_in: 0.18, currency: 'CNY' })).toBe('¥0.18/分')
  })

  it('默认每百万 tokens：CNY/USD 前缀与 in·out 组合', () => {
    expect(formatPrice({ price_in: 1.4, currency: 'CNY' })).toBe('¥1.4/M')
    expect(formatPrice({ price_in: 1.4, price_out: 4.4, currency: 'USD' })).toBe('$1.4·$4.4/M')
    expect(formatPrice({ price_out: 4.4, currency: 'USD' })).toBe('$4.4/M')
  })

  it('空 meta / 未计价返回空串', () => {
    expect(formatPrice(undefined)).toBe('')
    expect(formatPrice({})).toBe('')
    expect(formatPrice({ unit: 'call' })).toBe('')
  })
})

describe('模型中心 capLabels / modelMeta', () => {
  it('capLabels 覆盖契约能力键', () => {
    expect(capLabels).toMatchObject({ vision: '视觉', tools: '工具', reasoning: '推理', search: '搜索', json: '结构化' })
  })

  it('modelMeta：任一字段存在才构造，全空返回 undefined（卡片不占位）', () => {
    expect(modelMeta({ id: 'm', owned_by: 'x', status: 'ok' })).toBeUndefined()
    expect(modelMeta({ id: 'm', owned_by: 'x', status: 'ok', context_length: 128_000 }))
      .toMatchObject({ context_length: 128_000 })
    expect(modelMeta({ id: 'm', owned_by: 'x', status: 'ok', points_in: 2.5 }))
      .toMatchObject({ points_in: 2.5 })
  })

  it('findModelMeta：按引擎 + 模型定位元数据，找不到返回 undefined', () => {
    const engines: EngineConfig[] = [
      {
        id: 'glm', name: 'GLM', type: 'glm', base_url: '', enabled: true, default_model: '',
        models: [{ id: 'glm-5.3', owned_by: 'glm', status: 'ok', kind: 'llm', context_length: 200_000 }],
      },
    ]
    expect(findModelMeta(engines, 'glm', 'glm-5.3')).toMatchObject({ context_length: 200_000 })
    expect(findModelMeta(engines, 'glm', 'nope')).toBeUndefined()
    expect(findModelMeta(engines, 'xai', 'glm-5.3')).toBeUndefined()
  })
})

describe('模型中心 积分口径（coding 套餐估算）', () => {
  it('estimatePoints 按 (入×in + 缓存×cached + 出×out)/10000', () => {
    expect(estimatePoints(50_000, 30_000, 20_000, { points_in: 2, points_cached: 0.4, points_out: 4 }))
      .toBeCloseTo(19.2)
  })

  it('缺失系数按 0 计', () => {
    expect(estimatePoints(10_000, 5_000, 2_000, { points_in: 1 })).toBe(1)
    expect(estimatePoints(10_000, 0, 0, {})).toBe(0)
  })

  it('hasPointsCoef：具备主系数才算可估算（仅 peak 不算）', () => {
    expect(hasPointsCoef({ points_in: 1 })).toBe(true)
    expect(hasPointsCoef({ points_cached: 0.4 })).toBe(true)
    expect(hasPointsCoef({ points_peak: 2 })).toBe(false)
    expect(hasPointsCoef(undefined)).toBe(false)
  })
})
