import { useEffect } from 'react'

/**
 * useWailsEvent — 统一 Wails 事件监听 hook
 * 自动在组件卸载时调用 EventsOff 清理，避免内存泄漏
 *
 * @example
 * useWailsEvent('chapter-stream', (ev) => {
 *   console.log('stream chunk:', ev)
 * })
 */
export function useWailsEvent(eventName: string, handler: (data: any) => void) {
  useEffect(() => {
    // @ts-ignore
    if (!window.runtime?.EventsOn) return
    // @ts-ignore
    window.runtime.EventsOn(eventName, handler)
    return () => {
      try {
        // @ts-ignore
        window.runtime?.EventsOff?.(eventName)
      } catch (e) {
        // 忽略销毁时的错误
      }
    }
  }, [eventName])
}
