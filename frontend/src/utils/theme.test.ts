import { afterEach, describe, expect, it, vi } from 'vitest'
import { resolveThemeColor } from './theme'

describe('resolveThemeColor', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('返回不含 var()/color-mix() 的原串（探针短路）', () => {
    const spy = vi.spyOn(window, 'getComputedStyle')
    expect(resolveThemeColor('#6b7280')).toBe('#6b7280') // hex-exempt 测试输入样例
    expect(resolveThemeColor('rgb(37, 99, 235)')).toBe('rgb(37, 99, 235)')
    expect(spy).not.toHaveBeenCalled()
  })

  it('var() 串经探针解析为 computed 色', () => {
    const spy = vi.spyOn(window, 'getComputedStyle').mockReturnValue({
      color: 'rgb(37, 99, 235)',
    } as CSSStyleDeclaration)
    expect(resolveThemeColor('var(--color-primary)')).toBe('rgb(37, 99, 235)')
    expect(spy).toHaveBeenCalled()
  })

  it('color-mix() 无 var() 也走探针（v4.390 前字符串手拆实现不支持）', () => {
    vi.spyOn(window, 'getComputedStyle').mockReturnValue({
      color: 'color(srgb 0.5 0.2 0.1 / 0.92)',
    } as CSSStyleDeclaration)
    expect(resolveThemeColor('color-mix(in srgb, var(--color-surface) 92%, #000)'))
      .toBe('color(srgb 0.5 0.2 0.1 / 0.92)')
  })

  it('探针 computed 为空时回退原串（jsdom 对未定义令牌的行为）', () => {
    // 不 mock：jsdom 对 var(--未定义令牌) 的 computed color 为空串
    expect(resolveThemeColor('var(--color-token-undefined-xyz)'))
      .toBe('var(--color-token-undefined-xyz)')
  })

  it('非浏览器环境原样返回', () => {
    vi.stubGlobal('document', undefined)
    expect(resolveThemeColor('var(--color-primary)')).toBe('var(--color-primary)')
  })
})
