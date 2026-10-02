// SinCastPanel 生成设定卡（v4.403，sin 侧入口）：chip 级一键三视图→存回角色库。
// v4.408：生成期进度行（节点中文+用时/排队）+行尾取消钮（全局 CancelImageGeneration）。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { SinCastPanel } from './SinCastPanel'
import { __resetIllustrationQueueForTest, enqueueIllustration } from './illustrationQueue'
import type { SinCastCharacter } from './useSinCast'

const apiMock = vi.hoisted(() => ({
  getComfyUITaskProgress: vi.fn(),
  getImageBackendInfo: vi.fn(),
  cancelImageGeneration: vi.fn(),
}))

vi.mock('../../api/image', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/image')>()),
  getComfyUITaskProgress: apiMock.getComfyUITaskProgress,
  getImageBackendInfo: apiMock.getImageBackendInfo,
  cancelImageGeneration: apiMock.cancelImageGeneration,
}))

const cast: SinCastCharacter[] = [
  { id: 'c1', name: '林晚', portraitUrl: 'data:image/png;base64,P1' },
  { id: 'c2', name: '沈砚', portraitUrl: '' },
]

const setup = (onGenerateSheet: (id: string) => Promise<void>) =>
  render(
    <SinCastPanel
      cast={cast}
      saving={false}
      onOpenPicker={() => {}}
      onRemove={() => {}}
      onGenerateSheet={onGenerateSheet}
    />,
  )

beforeEach(() => {
  __resetIllustrationQueueForTest()
  apiMock.getComfyUITaskProgress.mockReset()
  // 默认空快照：busy 期瞬时轮询不至于拿到 undefined（未在生成的用例也安全）
  apiMock.getComfyUITaskProgress.mockResolvedValue({ status: '', elapsed: 0, node: '' })
  // useComfyTaskProgress（FE6-06 起共用）挂载时解析图像后端类型
  apiMock.getImageBackendInfo.mockReset().mockResolvedValue({ backend: 'comfyui' })
  apiMock.cancelImageGeneration.mockReset()
})

describe('SinCastPanel 生成设定卡（sin 侧入口）', () => {
  it('chip 设定卡按钮：点击按角色 id 调用，成功后复位', async () => {
    const onGenerateSheet = vi.fn().mockResolvedValue(undefined)
    setup(onGenerateSheet)
    const btn = screen.getByLabelText('生成 林晚 的设定卡')
    fireEvent.click(btn)
    expect(onGenerateSheet).toHaveBeenCalledWith('c1')
    await waitFor(() => expect(btn.hasAttribute('disabled')).toBe(false))
  })

  it('单飞：进行中禁用其他 chip 的设定卡按钮；失败同样复位', async () => {
    let resolve1!: (v: undefined) => void
    const onGenerateSheet = vi
      .fn()
      .mockImplementationOnce(() => new Promise<undefined>((r) => (resolve1 = r)))
      .mockRejectedValueOnce(new Error('仅 ComfyUI 本地档'))
    apiMock.getComfyUITaskProgress.mockResolvedValue({ status: '', elapsed: 0, node: '' })
    setup(onGenerateSheet)
    const b1 = screen.getByLabelText('生成 林晚 的设定卡')
    const b2 = screen.getByLabelText('生成 沈砚 的设定卡')
    fireEvent.click(b1)
    await waitFor(() => expect(b2.hasAttribute('disabled')).toBe(true))
    resolve1(undefined)
    await waitFor(() => expect(b2.hasAttribute('disabled')).toBe(false))
    // 失败路径：调用后按钮复位（错误由页面层 message 透出）
    fireEvent.click(b2)
    await waitFor(() => expect(b2.hasAttribute('disabled')).toBe(false))
    expect(onGenerateSheet).toHaveBeenCalledTimes(2)
  })

  it('未传 onGenerateSheet：不渲染设定卡按钮（向后兼容）', () => {
    render(
      <SinCastPanel cast={cast} saving={false} onOpenPicker={() => {}} onRemove={() => {}} />,
    )
    expect(screen.queryByLabelText('生成 林晚 的设定卡')).toBeNull()
  })

  it('生成中进度行：显示当前节点中文与用时，取消钮调用全局取消', async () => {
    let resolveGen!: (v: undefined) => void
    const onGenerateSheet = vi.fn().mockImplementation(() => new Promise<undefined>((r) => (resolveGen = r)))
    apiMock.getComfyUITaskProgress.mockResolvedValue({ status: 'running', elapsed: 42, node: 'UNETLoader' })
    setup(onGenerateSheet)
    fireEvent.click(screen.getByLabelText('生成 林晚 的设定卡'))
    const row = await screen.findByTestId('sin-cast-progress')
    expect(row.textContent).toContain('加载模型')
    expect(row.textContent).toContain('已用时 42s')
    fireEvent.click(screen.getByTestId('sin-cast-cancel'))
    await waitFor(() => expect(apiMock.cancelImageGeneration).toHaveBeenCalledTimes(1))
    resolveGen(undefined)
    await waitFor(() => expect(screen.queryByTestId('sin-cast-progress')).toBeNull())
  })

  it('排队中文案；结束后进度行消失', async () => {
    let resolveGen!: (v: undefined) => void
    const onGenerateSheet = vi.fn().mockImplementation(() => new Promise<undefined>((r) => (resolveGen = r)))
    apiMock.getComfyUITaskProgress.mockResolvedValue({ status: 'queued', elapsed: 3, node: 'queue' })
    setup(onGenerateSheet)
    fireEvent.click(screen.getByLabelText('生成 沈砚 的设定卡'))
    const row = await screen.findByTestId('sin-cast-progress')
    expect(row.textContent).toContain('排队中（前有任务）')
    resolveGen(undefined)
    await waitFor(() => expect(screen.queryByTestId('sin-cast-progress')).toBeNull())
  })

  it('进度读取失败：不渲染进度行也不阻断生成', async () => {
    let resolveGen!: (v: undefined) => void
    const onGenerateSheet = vi.fn().mockImplementation(() => new Promise<undefined>((r) => (resolveGen = r)))
    apiMock.getComfyUITaskProgress.mockRejectedValue(new Error('bridge down'))
    setup(onGenerateSheet)
    fireEvent.click(screen.getByLabelText('生成 林晚 的设定卡'))
    await waitFor(() => expect(apiMock.getComfyUITaskProgress).toHaveBeenCalled())
    expect(screen.queryByTestId('sin-cast-progress')).toBeNull()
    resolveGen(undefined)
    await waitFor(() => expect(screen.getByLabelText('生成 林晚 的设定卡').hasAttribute('disabled')).toBe(false))
  })
})

describe('SinCastPanel 一致性评分（v4.410 sin 侧快路径）', () => {
  const setupScore = (
    onGenerateSheet: ((id: string) => Promise<void>) | undefined,
    onScore: (id: string) => Promise<void>,
  ) =>
    render(
      <SinCastPanel
        cast={cast}
        saving={false}
        onOpenPicker={() => {}}
        onRemove={() => {}}
        onGenerateSheet={onGenerateSheet}
        onScore={onScore}
      />,
    )

  it('评分钮：点击按角色 id 调用，成功后复位', async () => {
    const onScore = vi.fn().mockResolvedValue(undefined)
    setupScore(undefined, onScore)
    const btn = screen.getByLabelText('评分 林晚 的参考图')
    fireEvent.click(btn)
    expect(onScore).toHaveBeenCalledWith('c1')
    await waitFor(() => expect(btn.hasAttribute('disabled')).toBe(false))
  })

  it('互斥：评分进行中禁设定卡钮；生成进行中禁评分钮', async () => {
    let resolveScore!: (v: undefined) => void
    const onScore = vi.fn().mockImplementation(() => new Promise<undefined>((r) => (resolveScore = r)))
    let resolveGen!: (v: undefined) => void
    const onGenerateSheet = vi.fn().mockImplementation(() => new Promise<undefined>((r) => (resolveGen = r)))
    apiMock.getComfyUITaskProgress.mockResolvedValue({ status: '', elapsed: 0, node: '' })
    setupScore(onGenerateSheet, onScore)
    fireEvent.click(screen.getByLabelText('评分 林晚 的参考图'))
    await waitFor(() =>
      expect(screen.getByLabelText('生成 林晚 的设定卡').hasAttribute('disabled')).toBe(true),
    )
    resolveScore(undefined)
    await waitFor(() =>
      expect(screen.getByLabelText('生成 林晚 的设定卡').hasAttribute('disabled')).toBe(false),
    )
    fireEvent.click(screen.getByLabelText('生成 林晚 的设定卡'))
    await waitFor(() =>
      expect(screen.getByLabelText('评分 林晚 的参考图').hasAttribute('disabled')).toBe(true),
    )
    resolveGen(undefined)
    await waitFor(() =>
      expect(screen.getByLabelText('评分 林晚 的参考图').hasAttribute('disabled')).toBe(false),
    )
  })

  it('未传 onScore：不渲染评分钮（向后兼容）', () => {
    render(
      <SinCastPanel cast={cast} saving={false} onOpenPicker={() => {}} onRemove={() => {}} />,
    )
    expect(screen.queryByLabelText('评分 林晚 的参考图')).toBeNull()
  })
})

describe('SinCastPanel 设定卡入串行队列（v4.427：与流内插图/画廊重生成同队列）', () => {
  it('生成走 illustrationQueue：与流内插图串行（后入者等先入者完成）', async () => {
    const order: string[] = []
    let releaseFirst!: () => void
    const first = new Promise<void>((r) => (releaseFirst = r))
    // 先占住队列（模拟流内插图在跑）
    const { promise: occupying } = enqueueIllustration(() => first.then(() => order.push('illu')))
    const onGenerateSheet = vi.fn().mockImplementation(async () => { order.push('sheet') })
    setup(onGenerateSheet)
    fireEvent.click(screen.getByLabelText('生成 林晚 的设定卡'))
    // 占位未释放：设定卡不得开跑
    await new Promise((r) => setTimeout(r, 10))
    expect(onGenerateSheet).not.toHaveBeenCalled()
    releaseFirst()
    await occupying
    await waitFor(() => expect(onGenerateSheet).toHaveBeenCalledWith('c1'))
    expect(order).toEqual(['illu', 'sheet'])
  })

  it('排队位次显示：被占位时显示「排队中」，取消只摘自己的队位', async () => {
    let release!: () => void
    const first = new Promise<void>((r) => (release = r))
    const { promise: occupying } = enqueueIllustration(() => first)
    const onGenerateSheet = vi.fn().mockResolvedValue(undefined)
    setup(onGenerateSheet)
    fireEvent.click(screen.getByLabelText('生成 林晚 的设定卡'))
    await screen.findByText(/排队中/)
    fireEvent.click(screen.getByTestId('sin-cast-cancel'))
    release()
    await occupying
    // 队位已摘：设定卡不再执行（取消语义，不是失败样式）
    await new Promise((r) => setTimeout(r, 10))
    expect(onGenerateSheet).not.toHaveBeenCalled()
  })
})
