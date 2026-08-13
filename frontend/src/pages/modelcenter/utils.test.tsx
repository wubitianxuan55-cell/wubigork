import { describe, expect, it } from 'vitest'
import {
  COMFY_IMAGE_MODELS,
  imageModelDefaultFor,
  imageModelOptionsFor,
  isImageModel,
  modelAvailability,
  featureState,
  routeSourceLabel,
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
