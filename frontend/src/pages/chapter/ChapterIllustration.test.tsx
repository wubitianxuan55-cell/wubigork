import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ChapterIllustration } from './ChapterIllustration'

// ── 调用层 mock ─────────────────────────────────────────────────────────
// GenerateSceneIllustration 经 gaea/lib/bridge 的 app 调用（NovelB 门面代理）；
// 组件内 PortraitImg 依赖 bridge 的 AttachmentDataURL（本地路径兜底），一并 mock。
// 配图 v2（刀 D）：GetCharacters 拉角色名单（getCharacters 内部 try/catch 回 []，
// 桩回两名样本走真实映射）。
const { generateSceneIllustrationMock, getCharactersBridgeMock } = vi.hoisted(() => ({
  generateSceneIllustrationMock: vi.fn(),
  getCharactersBridgeMock: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', () => ({
  app: {
    AttachmentDataURL: vi.fn(),
    GenerateSceneIllustration: generateSceneIllustrationMock,
    GetCharacters: getCharactersBridgeMock,
  },
}))

const EMPTY_OPTS = JSON.stringify({ characterIds: [], style: '' })

describe('ChapterIllustration 章节配图', () => {
  beforeEach(() => {
    generateSceneIllustrationMock.mockReset()
    getCharactersBridgeMock.mockReset()
    getCharactersBridgeMock.mockResolvedValue({ characters: [
      { id: 'c1', name: '林昭' },
      { id: 'c2', name: '苏晚' },
    ] })
  })

  it('打开即加载：成功后展示图片 URL 与 revised_prompt 说明', async () => {
    let resolve!: (v: unknown) => void
    generateSceneIllustrationMock.mockReturnValue(new Promise((r) => { resolve = r }))
    render(<ChapterIllustration chapterNum={3} onClose={() => {}} />)

    // 打开即加载（spinner + 提示）
    expect(screen.getByText('正在为第3章生成配图…')).toBeTruthy()

    await act(async () => {
      resolve({ url: 'https://img.example.com/scene-1.png', revised_prompt: '夜色下的竹林，月光透过竹叶洒落' })
    })

    // 图片 URL + revised_prompt 说明
    expect(await screen.findByRole('img', { name: '第3章配图' })).toBeTruthy()
    expect(screen.getByText('夜色下的竹林，月光透过竹叶洒落')).toBeTruthy()
    expect(generateSceneIllustrationMock).toHaveBeenCalledWith(3, EMPTY_OPTS)
  })

  it('失败时展示错误，重试可再次调用并成功', async () => {
    generateSceneIllustrationMock.mockRejectedValueOnce(new Error('图像后端不可用'))
    render(<ChapterIllustration chapterNum={2} onClose={() => {}} />)

    const alert = await screen.findByRole('alert')
    expect(alert.textContent).toContain('图像后端不可用')

    // 重试 → 再次调用后端并成功展示图片
    generateSceneIllustrationMock.mockResolvedValueOnce({ url: 'https://img.example.com/retry.png' })
    fireEvent.click(screen.getByText('重试'))

    await waitFor(() => expect(generateSceneIllustrationMock).toHaveBeenCalledTimes(2))
    expect(await screen.findByAltText('第2章配图')).toBeTruthy()
  })

  it('章节号为空时展示守卫错误，不调用后端', async () => {
    render(<ChapterIllustration chapterNum={0} onClose={() => {}} />)
    const alert = await screen.findByRole('alert')
    expect(alert.textContent).toContain('无章节号')
    expect(generateSceneIllustrationMock).not.toHaveBeenCalled()
  })

  it('配图 v2：风格槽与角色勾选进入 opts，重试带最新选项；refNote 展示', async () => {
    generateSceneIllustrationMock.mockResolvedValue({
      url: 'https://img.example.com/scene-v2.png',
      revised_prompt: '水墨夜景',
      refNote: '已附 1 张角色参考图（img2img）',
    })
    render(<ChapterIllustration chapterNum={5} onClose={() => {}} />)
    await waitFor(() => expect(generateSceneIllustrationMock).toHaveBeenCalledTimes(1))
    // refNote 展示为浅色说明行
    await waitFor(() => expect(screen.getByTestId('scene-ref-note').textContent).toContain('已附 1 张'))

    // 填风格 + 勾第一个角色 → 重试带最新选项
    fireEvent.change(screen.getByTestId('scene-style-input'), { target: { value: '水墨国风' } })
    const boxes = await screen.findAllByRole('checkbox')
    fireEvent.click(boxes[0])
    fireEvent.click(screen.getByText('重试'))
    await waitFor(() => expect(generateSceneIllustrationMock).toHaveBeenCalledTimes(2))
    const [num, optsJSON] = generateSceneIllustrationMock.mock.calls[1]
    expect(num).toBe(5)
    expect(JSON.parse(optsJSON as string)).toEqual({ characterIds: ['c1'], style: '水墨国风' })
  })

  it('配图 v2：无 refNote 不渲染说明行', async () => {
    generateSceneIllustrationMock.mockResolvedValue({
      url: 'https://img.example.com/scene-plain.png',
      revised_prompt: '',
      refNote: '',
    })
    render(<ChapterIllustration chapterNum={4} onClose={() => {}} />)
    await waitFor(() => expect(generateSceneIllustrationMock).toHaveBeenCalled())
    await waitFor(() => expect(screen.queryByTestId('scene-ref-note')).toBeNull())
  })
})
