// readingScrollMemory.test.ts — 阅读位置记忆纯工具族测试：
// localStorage 键格式、保存/恢复回路、无记录回落 0（存储实现见 src/test/setup.ts polyfill），
// 以及 scrollPct 的百分比/钳制/无溢出语义（scrollHeight 等用实例属性打桩——jsdom 恒为 0）。
import { afterEach, describe, expect, it } from 'vitest'
import { readSavedScrollTop, readScrollKey, saveScrollTop, scrollPct } from './readingScrollMemory'

/** jsdom 未实现滚动几何，用实例属性打桩模拟滚动容器 */
function stubScrollBox(scrollTop: number, scrollHeight: number, clientHeight: number): HTMLElement {
  const el = document.createElement('div')
  Object.defineProperty(el, 'scrollTop', { value: scrollTop, configurable: true })
  Object.defineProperty(el, 'scrollHeight', { value: scrollHeight, configurable: true })
  Object.defineProperty(el, 'clientHeight', { value: clientHeight, configurable: true })
  return el
}

afterEach(() => {
  localStorage.clear()
})

describe('滚动位置记忆', () => {
  it('键格式：固定前缀 + nodeId', () => {
    expect(readScrollKey('ch-1')).toBe('gaea.novel.reading.scroll.ch-1')
  })

  it('保存后可读回（字符串数值）', () => {
    saveScrollTop('ch-1', 240)
    expect(localStorage.getItem(readScrollKey('ch-1'))).toBe('240')
    expect(readSavedScrollTop('ch-1')).toBe(240)
  })

  it('无记录返回 0（|| 0 兜底）', () => {
    expect(readSavedScrollTop('ch-none')).toBe(0)
  })

  it('保存 0 也写库（读到 0 与「无记录」读数一致，恢复逻辑以 >0 判断）', () => {
    saveScrollTop('ch-1', 0)
    expect(localStorage.getItem(readScrollKey('ch-1'))).toBe('0')
  })
})

describe('scrollPct', () => {
  it('按比例返回进度并四舍五入', () => {
    expect(scrollPct(stubScrollBox(250, 2000, 1000))).toBe(25)
    expect(scrollPct(stubScrollBox(333, 2000, 1000))).toBe(33)
  })

  it('超界时钳到 100', () => {
    expect(scrollPct(stubScrollBox(1500, 2000, 1000))).toBe(100)
    expect(scrollPct(stubScrollBox(99999, 2000, 1000))).toBe(100)
  })

  it('容器无溢出（max ≤ 0）返回 0', () => {
    expect(scrollPct(stubScrollBox(0, 500, 500))).toBe(0)
    expect(scrollPct(stubScrollBox(100, 300, 500))).toBe(0)
  })
})
