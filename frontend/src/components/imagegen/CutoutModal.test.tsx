// 抠图弹窗（阶段二刀 E）：涂选（MaskBrushEditor 复用）→ 导出 mask 透传；
// 未涂 warning；成功态路径显示。jsdom 无 ctx → 桩编辑器产 mask。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react'
import type { Mock } from 'vitest'

const mocks = vi.hoisted(() => ({
  imageCutout: vi.fn(),
}))

vi.mock('../../api/image', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../api/image')>()
  return { ...actual, imageCutout: mocks.imageCutout }
})

// 蒙版编辑器桩：真组件 jsdom 导不出蒙版——按钮回调模拟「已涂抹」
vi.mock('./MaskBrushEditor', () => ({
  default: ({ onMaskChange }: { onMaskChange: (m: string | null) => void }) => (
    <div data-testid="mask-editor-stub">
      <button data-testid="stub-set-mask" onClick={() => onMaskChange('data:image/png;base64,MASK')}>set</button>
    </div>
  ),
}))

import CutoutModal from './CutoutModal'
import type { GenResult } from './types'

const SOURCE: GenResult = {
  image: 'data:image/png;base64,SRC', seed: 1, time: 1, prompt: 'p', model: 'm', size: '2x2',
}

describe('CutoutModal 抠图透明底导出（阶段二刀 E）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    cleanup()
  })

  it('未涂抹：warning 不触导出', () => {
    render(<CutoutModal open source={SOURCE} onClose={vi.fn()} />)
    fireEvent.click(screen.getByTestId('cutout-run'))
    expect(mocks.imageCutout).not.toHaveBeenCalled()
  })

  it('涂抹后导出：mask 透传；成功态显示路径', async () => {
    mocks.imageCutout.mockResolvedValue({ path: 'C:/Pictures/gaea/cutout-x.png', asset_id: 'ih-1' })
    render(<CutoutModal open source={SOURCE} onClose={vi.fn()} />)
    fireEvent.click(screen.getByTestId('stub-set-mask'))
    fireEvent.click(screen.getByTestId('cutout-run'))
    await waitFor(() => expect(mocks.imageCutout).toHaveBeenCalledTimes(1))
    const args = (mocks.imageCutout as Mock).mock.calls[0]
    expect(args[0]).toBe(SOURCE.image)
    expect(args[1]).toBe('data:image/png;base64,MASK')
    expect(screen.getByTestId('cutout-saved').textContent).toContain('cutout-x.png')
  })

  it('错误态如实展示', async () => {
    mocks.imageCutout.mockResolvedValue({ error: '蒙版与原图尺寸不一致' })
    render(<CutoutModal open source={SOURCE} onClose={vi.fn()} />)
    fireEvent.click(screen.getByTestId('stub-set-mask'))
    fireEvent.click(screen.getByTestId('cutout-run'))
    await waitFor(() => expect(screen.getByTestId('cutout-error').textContent).toContain('尺寸不一致'))
  })
})
