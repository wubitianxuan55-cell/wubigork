// SinIllustration.progress.test.tsx — 插图卡的进度动画（排队 → 生成 → 百分比/阶段）。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

const apiMock = vi.hoisted(() => ({
  getComfyUITaskProgress: vi.fn(),
  getImageBackendInfo: vi.fn(),
  cancelImageGeneration: vi.fn(),
  readFileAsDataURL: vi.fn(),
}))

const bridgeMock = vi.hoisted(() => ({
  SinIllustrate: vi.fn(),
}))

vi.mock('../../api/image', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/image')>()),
  getComfyUITaskProgress: apiMock.getComfyUITaskProgress,
  getImageBackendInfo: apiMock.getImageBackendInfo,
  cancelImageGeneration: apiMock.cancelImageGeneration,
  readFileAsDataURL: apiMock.readFileAsDataURL,
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../gaea/lib/bridge')>()),
  app: new Proxy({} as Record<string, unknown>, {
    get(_t, prop: string) {
      return (bridgeMock as unknown as Record<string, unknown>)[prop]
    },
  }),
}))

import { SinIllustration } from './SinIllustration'
import { __resetIllustrationQueueForTest } from './illustrationQueue'

beforeEach(() => {
  __resetIllustrationQueueForTest()
  apiMock.getImageBackendInfo.mockReset().mockResolvedValue({ backend: 'comfyui' })
  apiMock.getComfyUITaskProgress.mockReset().mockResolvedValue({ status: 'running', elapsed: 3, percent: 55, node: 'KSampler' })
  apiMock.readFileAsDataURL.mockReset().mockResolvedValue('data:image/png;base64,AAA')
  bridgeMock.SinIllustrate.mockReset()
})

describe('SinIllustration 生成进度', () => {
  it('生成中显示进度条：百分比 + 中文阶段名 + 已用时', async () => {
    bridgeMock.SinIllustrate.mockImplementation(() => new Promise(() => {})) // 挂住 = 生成中
    render(
      <SinIllustration
        storyId="sin_1" messageId={2} cueKey="0" prompt="雨夜站台" ready
        onGenerated={vi.fn()}
      />,
    )
    await waitFor(() => expect(screen.getByText('全部: 55%')).toBeTruthy())
    expect(screen.getByText('当前节点: 采样中')).toBeTruthy()
    expect(screen.getByText(/已用时/)).toBeTruthy()
    expect(screen.getByText('取消')).toBeTruthy()
  })

  it('后端无实时进度时不编造百分比：显示不定态光带', async () => {
    apiMock.getImageBackendInfo.mockResolvedValue({ backend: 'xai' })
    apiMock.getComfyUITaskProgress.mockResolvedValue({ status: '', elapsed: 0 })
    bridgeMock.SinIllustrate.mockImplementation(() => new Promise(() => {}))
    const { container } = render(
      <SinIllustration
        storyId="sin_1" messageId={2} cueKey="0" prompt="雨夜站台" ready
        onGenerated={vi.fn()}
      />,
    )
    await waitFor(() => expect(screen.getByText('生成中…')).toBeTruthy())
    const fill = container.querySelector('.ig-progress-fill') as HTMLElement
    expect(fill.className).toContain('is-indeterminate')
  })

  it('已有产物直接渲染图片，不进入生成态', async () => {
    render(
      <SinIllustration
        storyId="sin_1" messageId={2} cueKey="0" prompt="雨夜站台" ready path="C:/tmp/a.png"
        onGenerated={vi.fn()}
      />,
    )
    await waitFor(() => expect(apiMock.readFileAsDataURL).toHaveBeenCalledWith('C:/tmp/a.png'))
    expect(bridgeMock.SinIllustrate).not.toHaveBeenCalled()
  })

  it('角色参考图命中：caption 显示「角色参考：林晚」与外观锚点徽标', async () => {
    bridgeMock.SinIllustrate.mockResolvedValue({
      path: 'C:/tmp/a.png',
      ref_used: true,
      ref_characters: ['林晚'],
      ref_reason: '',
      ref_fallback: false,
      anchor_added: ['林晚'],
    })
    render(
      <SinIllustration
        storyId="sin_1" messageId={2} cueKey="0" prompt="雨夜站台，林晚侧身抓拍" ready
        onGenerated={vi.fn()}
      />,
    )
    await waitFor(() => expect(screen.getByText('角色参考：林晚')).toBeTruthy())
    expect(screen.getByText('已补外观锚点')).toBeTruthy()
  })

  it('参考图回退：如实显示「参考图不可用 · 已按纯文本生成」', async () => {
    bridgeMock.SinIllustrate.mockResolvedValue({
      path: 'C:/tmp/a.png',
      ref_used: false,
      ref_characters: [],
      ref_reason: '参考图不可用（fake）',
      ref_fallback: true,
      anchor_added: [],
    })
    render(
      <SinIllustration
        storyId="sin_1" messageId={2} cueKey="0" prompt="雨夜站台" ready
        onGenerated={vi.fn()}
      />,
    )
    await waitFor(() => expect(screen.getByText('参考图不可用 · 已按纯文本生成')).toBeTruthy())
  })
})

describe('SinIllustration 外部换图采纳（v4.265 画廊重新生成）', () => {
  const base = { storyId: 'sin_1', messageId: 2, cueKey: '0', prompt: '雨夜站台', ready: true } as const

  it('画廊重新生成回写后消息重载带来新 path：组件换读新图', async () => {
    const { rerender } = render(
      <SinIllustration {...base} path="C:/tmp/a.png" onGenerated={vi.fn()} />,
    )
    await waitFor(() => expect(apiMock.readFileAsDataURL).toHaveBeenCalledWith('C:/tmp/a.png'))
    rerender(<SinIllustration {...base} path="C:/tmp/a2.png" onGenerated={vi.fn()} />)
    await waitFor(() => expect(apiMock.readFileAsDataURL).toHaveBeenCalledWith('C:/tmp/a2.png'))
    expect(bridgeMock.SinIllustrate).not.toHaveBeenCalled() // 外部换图不触发再生成
  })

  it('自己生成中不采纳外部换图（防把在途产物顶掉）', async () => {
    bridgeMock.SinIllustrate.mockImplementation(() => new Promise(() => {})) // 挂住 = 生成中
    const { rerender } = render(<SinIllustration {...base} onGenerated={vi.fn()} />)
    await waitFor(() => expect(screen.getByText('取消')).toBeTruthy())
    apiMock.readFileAsDataURL.mockClear()
    rerender(<SinIllustration {...base} path="C:/tmp/ext.png" onGenerated={vi.fn()} />)
    expect(apiMock.readFileAsDataURL).not.toHaveBeenCalled()
  })
})
