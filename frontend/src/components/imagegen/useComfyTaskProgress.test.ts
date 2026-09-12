// useComfyTaskProgress.test.ts — 生成进度轮询：只在 active 时轮询、读失败保留
// 上一帧、非 ComfyUI 后端如实给出「未知进度」（percent = -1）。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'

const apiMock = vi.hoisted(() => ({
  getComfyUITaskProgress: vi.fn(),
  getImageBackendInfo: vi.fn(),
}))

vi.mock('../../api/image', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/image')>()),
  getComfyUITaskProgress: apiMock.getComfyUITaskProgress,
  getImageBackendInfo: apiMock.getImageBackendInfo,
}))

import { useComfyTaskProgress } from './useComfyTaskProgress'

beforeEach(() => {
  apiMock.getComfyUITaskProgress.mockReset().mockResolvedValue({ status: 'running', elapsed: 7, percent: 42, node: 'KSampler' })
  apiMock.getImageBackendInfo.mockReset().mockResolvedValue({ backend: 'comfyui' })
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useComfyTaskProgress', () => {
  it('active 时轮询并回填进度（百分比/节点/已用时）', async () => {
    const { result } = renderHook(() => useComfyTaskProgress(true, 50))
    await waitFor(() => expect(result.current.percent).toBe(42))
    expect(result.current.node).toBe('KSampler')
    expect(result.current.elapsed).toBe(7)
    expect(result.current.status).toBe('running')
    expect(apiMock.getComfyUITaskProgress).toHaveBeenCalled()
  })

  it('inactive 时不轮询且清空视图', async () => {
    const { result } = renderHook(() => useComfyTaskProgress(false, 50))
    await new Promise((r) => setTimeout(r, 80))
    expect(apiMock.getComfyUITaskProgress).not.toHaveBeenCalled()
    expect(result.current.percent).toBe(-1)
    expect(result.current.status).toBe('')
  })

  it('后端未提供 percent 时保持 -1（消费方走不定态，不编造百分比）', async () => {
    apiMock.getComfyUITaskProgress.mockResolvedValue({ status: '', elapsed: 0 })
    const { result } = renderHook(() => useComfyTaskProgress(true, 50))
    await waitFor(() => expect(apiMock.getComfyUITaskProgress).toHaveBeenCalled())
    expect(result.current.percent).toBe(-1)
  })

  it('单次读失败保留上一帧（进度条不闪回）', async () => {
    const { result } = renderHook(() => useComfyTaskProgress(true, 30))
    await waitFor(() => expect(result.current.percent).toBe(42))
    apiMock.getComfyUITaskProgress.mockRejectedValue(new Error('read failed'))
    await new Promise((r) => setTimeout(r, 90))
    expect(result.current.percent).toBe(42)
  })
})
