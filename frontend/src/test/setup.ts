import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'

// react-window v2 的 List/Grid 内部用 ResizeObserver 测量容器与动态行高，
// jsdom 未实现 → 空实现 polyfill（不触发回调，组件回落 defaultHeight/
// defaultRowHeight；真实浏览器由原生实现接管）。
if (typeof window !== 'undefined' && typeof (window as unknown as Record<string, unknown>).ResizeObserver !== 'function') {
  class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  Object.defineProperty(window, 'ResizeObserver', {
    writable: true,
    value: ResizeObserverStub,
  })
}

// Node 25 / vitest v4 jsdom：全局 localStorage 是缺 clear 等方法的最小对象，
// 导致测试 beforeEach 中 localStorage.clear() 抛 "localStorage.clear is not a function"
// （2026-08-15 chat 板块测试定位的基线环境问题）。此处补完整存储实现。
// 若宿主环境已有完整实现（正常 Node 版本），保持原生不动。
if (typeof window !== 'undefined') {
  const buildStorage = () => {
    const store = new Map<string, string>()
    return {
      get length() { return store.size },
      clear: () => store.clear(),
      getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
      key: (i: number) => Array.from(store.keys())[i] ?? null,
      removeItem: (k: string) => { store.delete(k) },
      setItem: (k: string, v: string) => { store.set(k, String(v)) },
    }
  }
  for (const name of ['localStorage', 'sessionStorage'] as const) {
    const cur = (window as unknown as Record<string, unknown>)[name]
    const isBroken = !cur || typeof (cur as Storage).clear !== 'function' || typeof (cur as Storage).getItem !== 'function'
    if (isBroken) {
      Object.defineProperty(window, name, { writable: true, value: buildStorage() })
    }
  }
}

// antd Modal/Form 内部依赖 Grid.useBreakpoint → window.matchMedia；
// jsdom 未实现，统一 polyfill（静默返回不匹配，测试无需响应式行为）。
if (typeof window !== 'undefined' && typeof window.matchMedia !== 'function') {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }),
  })
}

// 轨迹/transcript 定位滚动（v4.20 消息定位 / v4.24 工具调用定位）依赖
// scrollIntoView；jsdom 未实现 → 空实现（真实浏览器由原生接管）。
if (typeof window !== 'undefined' && typeof Element !== 'undefined' && typeof (Element.prototype as { scrollIntoView?: unknown }).scrollIntoView !== 'function') {
  (Element.prototype as { scrollIntoView: () => void }).scrollIntoView = () => {}
}

afterEach(() => {
  cleanup()
})
