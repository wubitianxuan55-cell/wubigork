import { describe, expect, it } from 'vitest'
import {
  BACKEND_OPTIONS, backendLabel, isLocalBackend,
  DASHSCOPE_EDIT_MODELS, DASHSCOPE_DEFAULT_MODEL,
  templateSizeToPreset,
} from './meta'

describe('引擎枚举（刀2 单源化）', () => {
  it('枚举覆盖 6 个后端，含 GLM（txt2imgOnly）与百炼改图（img2imgOnly）', () => {
    const values = BACKEND_OPTIONS.map((o) => o.value)
    expect(values).toEqual(['xai', 'comfyui', 'herdsman', 'ollama', 'glm', 'dashscope'])
    const glm = BACKEND_OPTIONS.find((o) => o.value === 'glm')
    const ds = BACKEND_OPTIONS.find((o) => o.value === 'dashscope')
    expect(glm?.txt2imgOnly).toBe(true)
    expect(glm?.img2imgOnly).toBeUndefined()
    expect(ds?.img2imgOnly).toBe(true)
    expect(ds?.txt2imgOnly).toBeUndefined()
  })

  it('backendLabel：云端/本地与未知回退', () => {
    expect(backendLabel('xai')).toBe('xAI')
    expect(backendLabel('dashscope')).toBe('百炼改图')
    expect(backendLabel('glm')).toBe('GLM')
    expect(backendLabel('comfyui')).toBe('ComfyUI')
    expect(backendLabel('weird')).toBe('weird')
  })

  it('isLocalBackend：本地引擎判定与绘梦启停一致', () => {
    expect(isLocalBackend('comfyui')).toBe(true)
    expect(isLocalBackend('herdsman')).toBe(true)
    expect(isLocalBackend('ollama')).toBe(true)
    expect(isLocalBackend('xai')).toBe(false)
    expect(isLocalBackend('glm')).toBe(false)
    expect(isLocalBackend('dashscope')).toBe(false)
  })
})

describe('百炼编辑模型常量（刀1 前端）', () => {
  it('三档官方模型 + 默认 = plus', () => {
    expect(DASHSCOPE_EDIT_MODELS).toEqual(['qwen-image-edit', 'qwen-image-edit-plus', 'qwen-image-edit-max'])
    expect(DASHSCOPE_DEFAULT_MODEL).toBe('qwen-image-edit-plus')
  })
})

describe('templateSizeToPreset（刀3 模板推荐画幅落地）', () => {
  it('文生图：比例标签映射为画幅预设', () => {
    expect(templateSizeToPreset('1:1', 'txt2img')).toEqual({ size: '1024x1024' })
    expect(templateSizeToPreset('16:9', 'txt2img')).toEqual({ size: '1024x576' })
    expect(templateSizeToPreset('9:16', 'txt2img')).toEqual({ size: '576x1024' })
    expect(templateSizeToPreset('3:4', 'txt2img')).toEqual({ size: '768x1024' })
    expect(templateSizeToPreset('4:3', 'txt2img')).toEqual({ size: '1024x768' })
  })

  it('文生图 2:3 立绘：无预置档走自定义 768×1152', () => {
    expect(templateSizeToPreset('2:3', 'txt2img')).toEqual({ size: 'custom', customWidth: 768, customHeight: 1152 })
  })

  it('非文生图模式不套模板画幅（图生图随参考图 / 视频独立画幅）', () => {
    expect(templateSizeToPreset('16:9', 'img2img')).toBeNull()
    expect(templateSizeToPreset('16:9', 't2v')).toBeNull()
    expect(templateSizeToPreset(undefined, 'txt2img')).toBeNull()
    expect(templateSizeToPreset('', 'txt2img')).toBeNull()
    expect(templateSizeToPreset('unknown-ratio', 'txt2img')).toBeNull()
  })
})
