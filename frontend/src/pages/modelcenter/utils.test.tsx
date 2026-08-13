import { describe, expect, it } from 'vitest'
import {
  COMFY_IMAGE_MODELS,
  imageModelDefaultFor,
  imageModelOptionsFor,
  isImageModel,
} from './utils'
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
