// 蒙版笔刷编辑器（阶段二刀 B）：涂抹状态驱动/清除回调/坐标比例映射。
// jsdom 无 2d ctx——导出路径走守卫返回 null（onMaskChange 收 null），
// 渲染层 ctx null 直接跳过不崩；坐标映射经 getBoundingClientRect mock 触达。
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'
import type { Mock } from 'vitest'

import MaskBrushEditor from './MaskBrushEditor'
import { renderMaskDataURL } from './ui'

describe('MaskBrushEditor 蒙版笔刷（阶段二刀 B）', () => {
  beforeEach(() => {
    // CSS 缩放显示下的坐标映射：画布显示为 100x100、自然尺寸 200x100
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
      width: 100, height: 100, left: 0, top: 0, right: 100, bottom: 100, x: 0, y: 0, toJSON: () => ({}),
    } as DOMRect)
    vi.spyOn(HTMLImageElement.prototype, 'naturalWidth', 'get').mockReturnValue(200)
    vi.spyOn(HTMLImageElement.prototype, 'naturalHeight', 'get').mockReturnValue(100)
  })
  afterEach(() => {
    vi.restoreAllMocks()
    cleanup()
  })

  it('初始态：引导文案 + 笔刷滑杆 + 清除按钮在位', () => {
    const onMaskChange = vi.fn()
    render(<MaskBrushEditor src="data:image/png;base64,SRC" onMaskChange={onMaskChange} />)
    expect(screen.getByTestId('mask-brush-status').textContent).toContain('在图上涂抹')
    expect(screen.getByTestId('mask-brush-size')).toBeTruthy()
    expect(screen.getByTestId('mask-brush-clear')).toBeTruthy()
    expect(onMaskChange).not.toHaveBeenCalled()
  })

  it('涂抹一笔：状态计数更新 + onMaskChange 回调（jsdom 无 ctx → 导出守卫 null）', () => {
    const onMaskChange = vi.fn()
    render(<MaskBrushEditor src="data:image/png;base64,SRC" onMaskChange={onMaskChange} />)
    const canvas = screen.getByTestId('mask-brush-canvas')
    // 显示 100x100 映射自然 200x100：clientX 50 → 自然 x 100
    fireEvent.pointerDown(canvas, { clientX: 50, clientY: 50, pointerId: 1 })
    fireEvent.pointerMove(canvas, { clientX: 60, clientY: 55, pointerId: 1 })
    fireEvent.pointerUp(canvas, { clientX: 60, clientY: 55, pointerId: 1 })
    expect(screen.getByTestId('mask-brush-status').textContent).toContain('已涂选 1 笔')
    expect(onMaskChange).toHaveBeenCalled()
    // jsdom 无 2d ctx：renderMaskDataURL 守卫返回 null（真机浏览器产出 data URL）
    expect((onMaskChange as Mock).mock.calls.at(-1)?.[0]).toBeNull()
  })

  it('清除：回到初始文案 + onMaskChange(null)', () => {
    const onMaskChange = vi.fn()
    render(<MaskBrushEditor src="data:image/png;base64,SRC" onMaskChange={onMaskChange} />)
    const canvas = screen.getByTestId('mask-brush-canvas')
    fireEvent.pointerDown(canvas, { clientX: 10, clientY: 10, pointerId: 1 })
    fireEvent.pointerUp(canvas, { clientX: 10, clientY: 10, pointerId: 1 })
    fireEvent.click(screen.getByTestId('mask-brush-clear'))
    expect(screen.getByTestId('mask-brush-status').textContent).toContain('在图上涂抹')
    expect((onMaskChange as Mock).mock.calls.at(-1)?.[0]).toBeNull()
  })

  it('renderMaskDataURL：空笔画返回 null（纯函数契约）', () => {
    expect(renderMaskDataURL([], 200, 100)).toBeNull()
  })
})
