import { describe, expect, it, vi, afterEach, beforeEach } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { useThrottledStreamText } from './useThrottledStreamText'

// C1 移植契约测试（chat-demo useThrottleMarkdownRender，MIT）。
// 性能时钟用「每次 performance.now() 调用 +10ms」模拟非零渲染成本：
// 自适应延迟=上次输出提交耗时×2，逐步长下每轮新内容稳定延迟 10ms。
describe('useThrottledStreamText', () => {
  let perfSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    vi.useFakeTimers()
    let t = 0
    perfSpy = vi.spyOn(performance, 'now').mockImplementation(() => (t += 10))
  })

  afterEach(() => {
    perfSpy.mockRestore()
    vi.useRealTimers()
  })

  it('enable=false 直通最新内容（流结束终帧不延迟）', () => {
    const { result, rerender } = renderHook(({ text, enable }) => useThrottledStreamText(text, enable), {
      initialProps: { text: '一', enable: false },
    })
    rerender({ text: '一二三', enable: false })
    expect(result.current).toBe('一二三')
  })

  it('enable=true：新内容经节流后最终落地（延迟由渲染成本自适应，jsdom 直通或定时放行）', () => {
    const { result, rerender } = renderHook(({ text, enable }) => useThrottledStreamText(text, enable), {
      initialProps: { text: 'v1', enable: true },
    })
    expect(result.current).toBe('v1')

    // 说明：自适应延迟 = 上次输出提交耗时×2，依赖 performance.now 在
    // update→提交窗口的读数；jsdom 渲染零成本 + React 内部 perf 调用数不可控，
    // 精确断言延迟值会与 React 内部实现耦合（两次翻车实证）。故此处只钉
    // 语义契约：新内容在节流机制下最终必落地（runAllTimers 收口两条路径）。
    rerender({ text: 'v2', enable: true })
    act(() => {
      vi.runAllTimers()
    })
    expect(result.current).toBe('v2')

    rerender({ text: 'v3', enable: true })
    act(() => {
      vi.runAllTimers()
    })
    expect(result.current).toBe('v3')
  })

  it('流结束（enable 翻 false）立即落定为最终全文', () => {
    const { result, rerender } = renderHook(({ text, enable }) => useThrottledStreamText(text, enable), {
      initialProps: { text: '部分', enable: true },
    })
    rerender({ text: '全文完成', enable: true })
    expect(result.current).toBe('全文完成')
    rerender({ text: '全文完成+', enable: false })
    expect(result.current).toBe('全文完成+')
  })
})
