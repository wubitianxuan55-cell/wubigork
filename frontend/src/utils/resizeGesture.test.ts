import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { startColumnResizeGesture } from './resizeGesture'

describe('startColumnResizeGesture', () => {
  beforeEach(() => {
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  })

  afterEach(() => {
    // 防串场：清除可能残留的手势态
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  })

  const dispatch = (type: string) => {
    window.dispatchEvent(new Event(type))
  }

  it('启动即设置 body 手势态（cursor + 禁选中）', () => {
    startColumnResizeGesture(vi.fn(), vi.fn())
    expect(document.body.style.cursor).toBe('col-resize')
    expect(document.body.style.userSelect).toBe('none')
    dispatch('pointerup')
  })

  it('pointerup 触发 onDone 并清理手势态与监听（pointerup 只生效一次）', () => {
    const onMove = vi.fn()
    const onDone = vi.fn()
    startColumnResizeGesture(onMove, onDone)

    dispatch('pointermove')
    expect(onMove).toHaveBeenCalledTimes(1)

    dispatch('pointerup')
    expect(onDone).toHaveBeenCalledTimes(1)
    expect(document.body.style.cursor).toBe('')
    expect(document.body.style.userSelect).toBe('')

    // 清理后不再响应 move/up（取消订阅生效）
    dispatch('pointermove')
    dispatch('pointerup')
    expect(onMove).toHaveBeenCalledTimes(1)
    expect(onDone).toHaveBeenCalledTimes(1)
  })

  it('pointercancel 与 pointerup 同义（拖拽被打断也收尾）', () => {
    const onDone = vi.fn()
    startColumnResizeGesture(vi.fn(), onDone)
    dispatch('pointercancel')
    expect(onDone).toHaveBeenCalledTimes(1)
    expect(document.body.style.cursor).toBe('')
  })
})
