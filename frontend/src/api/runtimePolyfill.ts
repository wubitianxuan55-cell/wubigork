/**
 * runtimePolyfill.ts — HTTP 环境 runtime 事件 polyfill
 *
 * 在 HTTP 环境下，模拟 window.runtime.EventsOn / EventsOff
 * 流式事件通过 EventSource 连接 `/api/stream` 实现。
 * 普通事件通过内存事件总线实现。
 */

type EventCallback = (...args: unknown[]) => void

// 已知流式事件 — 这些事件需要 SSE 连接
// 必须与 internal/rpc.go 中 streamEventMap 的值一致
const STREAM_EVENTS = new Set([
  'chapter-stream',
  'ghost-stream',
  'beat-prose-stream',
  'tts-stream',
  'xai-output',
])

// 活跃的 EventSource 连接
const activeSSE = new Map<string, EventSource>()

// 内存事件总线
const eventBus = new Map<string, Set<EventCallback>>()

/**
 * 初始化 runtime polyfill
 *
 * 在 App.tsx 最早时机调用 — 确保所有 runtime.EventsOn 调用可用
 */
export function initRuntimePolyfill(): void {
  // 避免重复初始化
  if ((window as any).__runtime_polyfill_initialized) return
  ;(window as any).__runtime_polyfill_initialized = true

  // 检查是否已有原生 runtime
  if ((window as any).runtime?.EventsOn) {
    console.log('[runtime] 原生 runtime 已存在，跳过 polyfill')
    return
  }

  console.log('[runtime] 创建 runtime polyfill')

  // 创建 runtime 命名空间
  if (!(window as any).runtime) {
    ;(window as any).runtime = {}
  }

  // EventsOn — 注册事件监听
  ;(window as any).runtime.EventsOn = (eventName: string, callback: EventCallback) => {
    if (!eventBus.has(eventName)) {
      eventBus.set(eventName, new Set())
    }
    eventBus.get(eventName)!.add(callback)

    // 流式事件 — 建立 SSE 连接
    if (STREAM_EVENTS.has(eventName) && !activeSSE.has(eventName)) {
      connectSSE(eventName)
    }
  }

  // EventsOff — 注销事件监听
  ;(window as any).runtime.EventsOff = (eventName: string, callback?: EventCallback) => {
    if (callback) {
      eventBus.get(eventName)?.delete(callback)
    } else {
      eventBus.delete(eventName)
    }

    // 清理 SSE 连接
    const sse = activeSSE.get(eventName)
    if (sse) {
      sse.close()
      activeSSE.delete(eventName)
    }
  }

  // EventsOnce — 单次监听（部分代码可能使用）
  ;(window as any).runtime.EventsOnce = (eventName: string, callback: EventCallback) => {
    const onceWrapper = (...args: unknown[]) => {
      callback(...args)
      ;(window as any).runtime.EventsOff(eventName, onceWrapper)
    }
    ;(window as any).runtime.EventsOn(eventName, onceWrapper)
  }

  // EventsEmit — 发射事件到本地总线（HTTP 环境无后端推送时使用）
  ;(window as any).runtime.EventsEmit = (eventName: string, ...args: unknown[]) => {
    const cbs = eventBus.get(eventName)
    if (!cbs) return
    for (const cb of cbs) {
      try {
        cb(...args)
      } catch (e) {
        console.error(`[runtime] EventsEmit ${eventName} 回调异常:`, e)
      }
    }
  }
}

/**
 * 连接 SSE 流
 */
function connectSSE(eventName: string): void {
  const url = `/api/stream?id=${eventName}`
  console.log(`[runtime] 连接 SSE: ${url}`)

  const es = new EventSource(url)

  es.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      // 分发到注册的回调
      const cbs = eventBus.get(eventName)
      if (cbs) {
        for (const cb of cbs) {
          cb(data)
        }
      }
    } catch (e) {
      console.error(`[runtime] SSE parse error for ${eventName}:`, e)
    }
  }

  es.addEventListener('done', () => {
    console.log(`[runtime] SSE 流结束: ${eventName}`)
    es.close()
    activeSSE.delete(eventName)
  })

  es.onerror = () => {
    console.error(`[runtime] SSE 连接错误: ${eventName}，5 秒后重连`)
    es.close()
    activeSSE.delete(eventName)
    setTimeout(() => {
      if (eventBus.has(eventName) && eventBus.get(eventName)!.size > 0) {
        connectSSE(eventName)
      }
    }, 5000)
  }

  activeSSE.set(eventName, es)
}
