import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'

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

afterEach(() => {
  cleanup()
})
