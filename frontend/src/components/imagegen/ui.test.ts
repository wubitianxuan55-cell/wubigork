import { describe, expect, it } from 'vitest'
import { estimateImageTime } from './ui'

describe('estimateImageTime', () => {
  it('estimates text-to-video by duration × 4', () => {
    expect(estimateImageTime('comfyui', 'ltx-video', 1, 't2v', 97, 8)).toBe(49)
    expect(estimateImageTime('comfyui', 'ltx-video', 1, 't2v', 49, 8)).toBe(25)
  })

  it('estimates img2img as 12s per image', () => {
    expect(estimateImageTime('comfyui', 'krea2', 4, 'img2img', 0, 0)).toBe(48)
  })

  it('applies backend- and model-specific pacing', () => {
    expect(estimateImageTime('xai', 'grok-imagine-image', 2, 'txt2img', 0, 0)).toBe(10)
    expect(estimateImageTime('comfyui', 'z-image-turbo', 1, 'txt2img', 0, 0)).toBe(20)
    expect(estimateImageTime('comfyui', 'krea2', 1, 'txt2img', 0, 0)).toBe(300)
  })

  it('falls back to 60s per image', () => {
    expect(estimateImageTime('herdsman', 'some-model', 3, 'txt2img', 0, 0)).toBe(180)
  })
})
