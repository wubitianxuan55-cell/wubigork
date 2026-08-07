/**
 * runtimePolyfill.ts — HTTP 环境 runtime 事件 polyfill
 *
 * 在 HTTP 环境下，模拟 window.runtime.EventsOn / EventsOff
 * 流式事件通过 EventSource 连接 `/api/stream` 实现。
 * 普通事件通过内存事件总线实现。
 */

type EventCallback = (...args: unknown[]) => void

// 活跃的 EventSource 连接
const activeSSE = new Map<string, EventSource>()

// 桥接探测：HTTP 模式先确认 Go 内核桥接（/api/health）存在，再建立 SSE，
// 避免无后端时对 /api/stream 无限重连刷错误。
let bridgeProbed = false
let bridgeAvailable = false
// 探测完成前订阅的事件先入队，探测成功后一次性全部建连，
// 避免启动时多个 EventsOn 并发订阅只连上第一个事件的竞态。
const pendingSSE = new Set<string>()

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

    // 所有事件都尝试走 Go 内核的 SSE 推送（网页版对齐桌面端）
    ensureBridgeSSE(eventName)
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

  // EventsOnMultiple — 注册监听，最多触发 maxCallbacks 次。
  // wailsjs/runtime/runtime.js 的 EventsOn(-1)/EventsOnce(1) 均通过它实现，
  // HTTP 环境不补齐会导致 "window.runtime.EventsOnMultiple is not a function"。
  ;(window as any).runtime.EventsOnMultiple = (eventName: string, callback: EventCallback, maxCallbacks: number) => {
    if (maxCallbacks < 0) {
      // -1 = 不限次数，等价 EventsOn
      ;(window as any).runtime.EventsOn(eventName, callback)
      return () => (window as any).runtime.EventsOff(eventName, callback)
    }
    // 正数 = 达到次数后自动注销（1 = 单次，等价 EventsOnce）
    let count = 0
    const wrapper = (...args: unknown[]) => {
      count++
      if (count > maxCallbacks) {
        ;(window as any).runtime.EventsOff(eventName, wrapper)
        return
      }
      callback(...args)
    }
    ;(window as any).runtime.EventsOn(eventName, wrapper)
    return () => (window as any).runtime.EventsOff(eventName, wrapper)
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
 * 探测 Go 内核桥接；可用时为指定事件建立 SSE 连接。
 */
function ensureBridgeSSE(eventName: string): void {
  if (activeSSE.has(eventName)) return
  if (bridgeAvailable) {
    connectSSE(eventName)
    return
  }
  pendingSSE.add(eventName)
  if (bridgeProbed) {
    // 上次探测失败（或桥接后启动）：下一次订阅时重新探测，
    // 让浏览器页面在桥接就绪后也能自动建连，无需刷新。
    bridgeProbed = false
  } else {
    bridgeProbed = true
    fetch('/api/health')
      .then((r) => {
        bridgeAvailable = r.ok
        if (bridgeAvailable) {
          for (const name of pendingSSE) {
            if (!activeSSE.has(name)) connectSSE(name)
          }
        }
        pendingSSE.clear()
      })
      .catch(() => {
        bridgeAvailable = false
        pendingSSE.clear()
      })
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
