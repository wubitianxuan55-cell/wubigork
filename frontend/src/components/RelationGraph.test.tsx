import React from 'react'
import { act } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import RelationGraph from './RelationGraph'
import { useAppStore } from '../stores/appStore'

// ── canvas 2d context 记录桩（jsdom 无 2d 实现；Proxy 兜底任意方法 noop）──
function makeCtxSpy() {
  const fillStyles: string[] = []
  const target: Record<string, unknown> = {
    measureText: () => ({ width: 10 }),
    createRadialGradient: () => ({ addColorStop: () => {} }),
    createLinearGradient: () => ({ addColorStop: () => {} }),
  }
  const ctx = new Proxy(target, {
    get(t, prop) {
      if (prop === 'fillStyle') return fillStyles.at(-1) ?? ''
      if (prop in t) return t[prop as string]
      return () => {}
    },
    set(t, prop, v) {
      if (prop === 'fillStyle') fillStyles.push(String(v))
      else t[prop as string] = v
      return true
    },
  })
  return { ctx: ctx as unknown as CanvasRenderingContext2D, fillStyles }
}

// ── 探针拦截：resolveThemeColor 的 span（display:none + var/color-mix color）
//    按当前主题返回 computed 色；其余 getComputedStyle 调用透传真实现 ──
const THEME_COMPUTED = {
  dark: 'rgb(10, 10, 10)',
  light: 'rgb(245, 245, 245)',
}

describe('RelationGraph canvas 调色板', () => {
  const offsetDesc = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'offsetParent')
  // 首绘走 requestAnimationFrame 调度，断言前等帧 flush
  const flushRaf = () => new Promise((r) => setTimeout(r, 40))

  afterEach(() => {
    vi.restoreAllMocks()
    // jsdom offsetParent 恒 null 会触发 v4.365 后台空转挂起（不绘制），测试强制可见
    if (offsetDesc) Object.defineProperty(HTMLElement.prototype, 'offsetParent', offsetDesc)
  })

  it('darkMode 切换后画布底色重解析（v4.390：模块级求值改组件 memo）', async () => {
    Object.defineProperty(HTMLElement.prototype, 'offsetParent', {
      configurable: true, get() { return document.body },
    })
    const { ctx, fillStyles } = makeCtxSpy()
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(ctx)
    const realGCS = window.getComputedStyle.bind(window)
    let theme: keyof typeof THEME_COMPUTED = 'dark'
    vi.spyOn(window, 'getComputedStyle').mockImplementation(((el: Element, ...rest: unknown[]) => {
      const html = el as HTMLElement
      if (html.tagName === 'SPAN' && html.style?.display === 'none' && html.style?.color) {
        return { color: THEME_COMPUTED[theme] } as CSSStyleDeclaration
      }
      return realGCS(el, ...rest as [])
    }) as typeof window.getComputedStyle)

    useAppStore.setState({ darkMode: true })
    render(
      <RelationGraph
        characters={[{ id: 'c1', name: '林晚', role_type: 'protagonist' } as never]}
        organizations={[{ id: 'o1', name: '青岚宗' } as never]}
        relationships={[{ from_id: 'c1', to_id: 'o1', relation_type: 'member' }]}
      />,
    )
    expect(fillStyles.length).toBeGreaterThanOrEqual(0)
    await act(async () => { await flushRaf() })
    // 初始（暗）palette.bg 已解析为暗色
    expect(fillStyles).toContain(THEME_COMPUTED.dark)

    theme = 'light'
    act(() => { useAppStore.setState({ darkMode: false }) })
    // 主题切换 → palette memo 重建 → 重绘 fillStyle 出现亮色解析值
    await act(async () => { await flushRaf() })
    expect(fillStyles).toContain(THEME_COMPUTED.light)
  })

  it('图例色块直接消费 var() 令牌串（DOM 活解析，随主题级联）', () => {
    const { ctx } = makeCtxSpy()
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(ctx)
    useAppStore.setState({ darkMode: true })
    render(
      <RelationGraph
        characters={[{ id: 'c1', name: '林晚', role_type: 'protagonist' } as never]}
        organizations={[]}
        relationships={[]}
      />,
    )
    const swatch = screen.getByText('主角').previousElementSibling as HTMLElement
    expect(swatch.style.background).toContain('var(')
    expect(swatch.style.background).not.toContain('rgb(')
  })
})
