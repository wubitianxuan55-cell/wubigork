import { describe, expect, it } from 'vitest'
import {
  BACKEND_OPTIONS, backendLabel, isLocalBackend,
  templateSizeToPreset,
} from './meta'

describe('引擎枚举（单源化）', () => {
  it('枚举覆盖 5 个后端，GLM 标 txt2imgOnly（百炼已下线）', () => {
    const values = BACKEND_OPTIONS.map((o) => o.value)
    expect(values).toEqual(['xai', 'comfyui', 'herdsman', 'ollama', 'glm'])
    const glm = BACKEND_OPTIONS.find((o) => o.value === 'glm')
    expect(glm?.txt2imgOnly).toBe(true)
    // 百炼（dashscope）v4.45 下线后不再出现在引擎枚举
    expect(BACKEND_OPTIONS.some((o) => o.value === 'dashscope')).toBe(false)
  })

  it('backendLabel：云端/本地与未知回退', () => {
    expect(backendLabel('xai')).toBe('xAI')
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
  })
})

describe('templateSizeToPreset（模板推荐画幅落地）', () => {
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
