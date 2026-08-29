import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ChapterIllustration } from './ChapterIllustration'

// ── 调用层 mock ─────────────────────────────────────────────────────────
// GenerateSceneIllustration 经 wailsjsCompat（NovelB 门面）调用；
// 组件内 PortraitImg 依赖 bridge 的 AttachmentDataURL（本地路径兜底），一并 mock。
const { generateSceneIllustrationMock } = vi.hoisted(() => ({
  generateSceneIllustrationMock: vi.fn(),
}))

vi.mock('../../wailsjsCompat', () => ({
  GenerateSceneIllustration: generateSceneIllustrationMock,
}))

vi.mock('../../gaea/lib/bridge', () => ({
  app: { AttachmentDataURL: vi.fn() },
}))

describe('ChapterIllustration 章节配图', () => {
  beforeEach(() => {
    generateSceneIllustrationMock.mockReset()
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
    expect(generateSceneIllustrationMock).toHaveBeenCalledWith(3)
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
})
