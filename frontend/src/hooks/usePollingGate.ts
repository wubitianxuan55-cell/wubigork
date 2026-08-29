import { useSyncExternalStore } from 'react'
import { isPageVisible } from '../lib/pollingGate'

// 订阅器保持模块级稳定引用，避免每次渲染重订阅
const subscribe = (onStoreChange: () => void) => {
  document.addEventListener('visibilitychange', onStoreChange)
  return () => document.removeEventListener('visibilitychange', onStoreChange)
}

/**
 * 轮询门控 hook：页面不可见时返回 false，轮询执行体应套 `if (!visible) return` 跳过
 * （interval 仍在但空转零成本）。jsdom 默认 visible，现有测试时序不受影响。
 * @param enabled 额外开关（如仅本地后端才轮询）；false 时恒返回 false。
 */
export function usePollingGate(enabled = true): boolean {
  return useSyncExternalStore(subscribe, () => enabled && isPageVisible())
}
