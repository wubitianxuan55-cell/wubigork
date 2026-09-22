import React from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'
import { CompanionAvatar } from './CompanionAvatar'

// ── canvas 桩：记录 fillStyle 与渐变 stop（jsdom 无 2d 实现）──
function makeCtxSpy() {
  const fills: string[] = []
  const stops: string[] = []
  const target: Record<string, unknown> = {
    measureText: () => ({ width: 10 }),
    createRadialGradient: () => ({ addColorStop: (_o: number, c: string) => { stops.push(c) } }),
    createLinearGradient: () => ({ addColorStop: (_o: number, c: string) => { stops.push(c) } }),
  }
  const ctx = new Proxy(target, {
    get(t, prop) {
      if (prop === 'fillStyle') return fills.at(-1) ?? ''
      if (prop in t) return t[prop as string]
      return () => {}
    },
    set(t, prop, v) {
      if (prop === 'fillStyle') fills.push(String(v))
      else t[prop as string] = v
      return true
    },
  })
  return { ctx: ctx as unknown as CanvasRenderingContext2D, fills, stops }
}

describe('CompanionAvatar 令牌色派生', () => {
  const offsetDesc = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'offsetParent')
  // jsdom offsetParent 恒 null 会触发 v4.365 后台空转挂起（不绘制），测试强制可见
  const forceVisible = () => Object.defineProperty(HTMLElement.prototype, 'offsetParent', {
    configurable: true, get() { return document.body },
  })

  afterEach(() => {
    vi.restoreAllMocks()
    if (offsetDesc) Object.defineProperty(HTMLElement.prototype, 'offsetParent', offsetDesc)
  })

  it('var() 令牌经探针解析为 rgba() 派生色（v4.390：集中解析+RGBA 拼接替代 hex 后缀）', async () => {
    forceVisible()
    const { ctx, fills, stops } = makeCtxSpy()
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(ctx)
    const realGCS = window.getComputedStyle.bind(window)
    // 探针（display:none span）返回 rgb() 串——hex 后缀拼接时代会拼出非法色
    vi.spyOn(window, 'getComputedStyle').mockImplementation(((el: Element, ...rest: unknown[]) => {
      const html = el as HTMLElement
      if (html.tagName === 'SPAN' && html.style?.display === 'none' && html.style?.color) {
        return { color: 'rgb(232, 83, 136)' } as CSSStyleDeclaration
      }
      return realGCS(el, ...rest as [])
    }) as typeof window.getComputedStyle)

    const { unmount } = render(
      <CompanionAvatar size={120} state="speaking" emotionColor="var(--gaea-glow, #e85388)" />,
    )
    // draw() 同步首帧
    await Promise.resolve()
    const derived = [...fills, ...stops].filter((c) => c.startsWith('rgba(232, 83, 136'))
    expect(derived.length).toBeGreaterThan(0)
    // 不再出现 hex 后缀拼接产物
    expect([...fills, ...stops].some((c) => /^rgb\(232, 83, 136\)[0-9a-f]{2}$/i.test(c))).toBe(false)
    unmount()
  })

  it('纯 hex emotionColor 仍可解析派生（默认值路径）', async () => {
    forceVisible()
    const { ctx, fills, stops } = makeCtxSpy()
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(ctx)
    const { unmount } = render(<CompanionAvatar size={120} state="idle" />)
    await Promise.resolve()
    const derived = [...fills, ...stops].filter((c) => c.startsWith('rgba(232, 83, 136'))
    expect(derived.length).toBeGreaterThan(0)
    unmount()
  })
})
