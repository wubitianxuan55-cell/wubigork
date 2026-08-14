/**
 * runtimePolyfill.ts — HTTP 环境 runtime 事件 polyfill
 *
 * 在 HTTP 环境下，模拟 window.runtime.EventsOn / EventsOff
 * 流式事件通过 fetch 流式 SSE 连接 `/api/stream` 实现（T6-9.6：
 * EventSource 无法自定义请求头，token 改用 Authorization: Bearer 头携带，
 * 不再拼进 URL 查询参数）。
 * 普通事件通过内存事件总线实现。
 */

import { getHttpToken } from './httpToken'

type EventCallback = (...args: unknown[]) => void

/** window.runtime 在 HTTP 环境补齐的最小事件面（与 types/wails.d.ts RuntimeAPI 对齐） */
interface RuntimeNamespace {
  EventsOn: (eventName: string, callback: EventCallback) => void
  EventsOff: (eventName: string, callback?: EventCallback) => void
  EventsOnce: (eventName: string, callback: EventCallback) => void
  EventsOnMultiple: (eventName: string, callback: EventCallback, maxCallbacks: number) => (() => void) | void
  EventsEmit: (eventName: string, ...args: unknown[]) => void
}

// 活跃的 SSE 连接（fetch 流式，AbortController 关闭）
interface SSEConnection {
  close: () => void
}
const activeSSE = new Map<string, SSEConnection>()

// 桥接探测：HTTP 模式先确认 Go 内核桥接（/api/health）存在，再建立 SSE，
// 避免无后端时对 /api/stream 无限重连刷错误。
let bridgeProbed = false
let bridgeAvailable = false
// 探测完成前订阅的事件先入队，探测成功后一次性全部建连，
// 避免启动时多个 EventsOn 并发订阅只连上第一个事件的竞态。
const pendingSSE = new Set<string>()

// 内存事件总线
const eventBus = new Map<string, Set<EventCallback>>()

// ── SSE 帧解析（纯函数，可单测）────────────────────────────────────────────

/**
 * 一帧解析结果。event 为显式 event 字段；无 event 字段的帧为 ''，
 * 等价 EventSource 的默认 message 事件（onmessage 语义）。
 */
export interface ParsedSSEFrame {
  event: string
  data: string[]
}

/**
 * 解析一帧完整 SSE 帧文本（调用方已按空行切分）。
 * - 多行 data 按 SSE 规范以 \n 拼接（data.join('\n') 即完整 payload）
 * - ':' 开头为注释行（keep-alive），忽略
 * - 无 event 且无 data 的帧（纯注释/空帧）返回 null
 */
export function parseSSEFrame(rawFrame: string): ParsedSSEFrame | null {
  let event = ''
  const data: string[] = []
  for (const rawLine of rawFrame.split('\n')) {
    let line = rawLine
    if (line.endsWith('\r')) line = line.slice(0, -1)
    if (line === '') continue
    if (line.startsWith(':')) continue // 注释行（keep-alive）
    const colon = line.indexOf(':')
    const field = colon === -1 ? line : line.slice(0, colon)
    let value = colon === -1 ? '' : line.slice(colon + 1)
    if (value.startsWith(' ')) value = value.slice(1)
    if (field === 'event') event = value
    else if (field === 'data') data.push(value)
    // id / retry 字段本场景无需处理
  }
  if (event === '' && data.length === 0) return null
  return { event, data }
}

/**
 * 逐块解析 SSE 帧流（纯函数）：任意切分的字符串块（网络 chunk 边界无关
 * 紧要），空行结束一帧并产出；流结束时 flush 未以空行收尾的残留帧。
 */
export async function* parseSSEStream(chunks: AsyncIterable<string> | Iterable<string>): AsyncGenerator<ParsedSSEFrame> {
  let buffer = ''
  let frameLines: string[] = []
  for await (const chunk of chunks) {
    buffer += chunk
    for (;;) {
      const nl = buffer.indexOf('\n')
      if (nl === -1) break
      let line = buffer.slice(0, nl)
      buffer = buffer.slice(nl + 1)
      if (line.endsWith('\r')) line = line.slice(0, -1)
      if (line === '') {
        // 空行 = 帧结束
        const frame = parseSSEFrame(frameLines.join('\n'))
        frameLines = []
        if (frame) yield frame
      } else {
        frameLines.push(line)
      }
    }
  }
  // 流结束：flush 残留——frameLines 里未结束的完整行 + buffer 里未以
  // \n 收尾的尾部行（正常 SSE 以空行收尾，通常无残留）。
  if (buffer !== '') {
    let tail = buffer
    if (tail.endsWith('\r')) tail = tail.slice(0, -1)
    if (tail !== '') frameLines.push(tail)
  }
  if (frameLines.length > 0) {
    const frame = parseSSEFrame(frameLines.join('\n'))
    if (frame) yield frame
  }
}

// ── runtime polyfill ───────────────────────────────────────────────────────

/**
 * 初始化 runtime polyfill
 *
 * 在 App.tsx 最早时机调用 — 确保所有 runtime.EventsOn 调用可用
 */
export function initRuntimePolyfill(): void {
  const win = window as unknown as { __runtime_polyfill_initialized?: boolean; runtime?: RuntimeNamespace }
  // 避免重复初始化
  if (win.__runtime_polyfill_initialized) return
  win.__runtime_polyfill_initialized = true

  // 检查是否已有原生 runtime
  if (win.runtime?.EventsOn) {
    console.log('[runtime] 原生 runtime 已存在，跳过 polyfill')
    return
  }

  console.log('[runtime] 创建 runtime polyfill')

  const runtime: RuntimeNamespace = {
    // EventsOn — 注册事件监听
    EventsOn: (eventName: string, callback: EventCallback) => {
      if (!eventBus.has(eventName)) {
        eventBus.set(eventName, new Set())
      }
      eventBus.get(eventName)!.add(callback)

      // 所有事件都尝试走 Go 内核的 SSE 推送（网页版对齐桌面端）
      ensureBridgeSSE(eventName)
    },

    // EventsOff — 注销事件监听
    EventsOff: (eventName: string, callback?: EventCallback) => {
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
    },

    // EventsOnce — 单次监听（部分代码可能使用）
    EventsOnce: (eventName: string, callback: EventCallback) => {
      const onceWrapper = (...args: unknown[]) => {
        callback(...args)
        runtime.EventsOff(eventName, onceWrapper)
      }
      runtime.EventsOn(eventName, onceWrapper)
    },

    // EventsOnMultiple — 注册监听，最多触发 maxCallbacks 次。
    // wailsjs/runtime/runtime.js 的 EventsOn(-1)/EventsOnce(1) 均通过它实现，
    // HTTP 环境不补齐会导致 "window.runtime.EventsOnMultiple is not a function"。
    EventsOnMultiple: (eventName: string, callback: EventCallback, maxCallbacks: number) => {
      if (maxCallbacks < 0) {
        // -1 = 不限次数，等价 EventsOn
        runtime.EventsOn(eventName, callback)
        return () => runtime.EventsOff(eventName, callback)
      }
      // 正数 = 达到次数后自动注销（1 = 单次，等价 EventsOnce）
      let count = 0
      const wrapper = (...args: unknown[]) => {
        count++
        if (count > maxCallbacks) {
          runtime.EventsOff(eventName, wrapper)
          return
        }
        callback(...args)
      }
      runtime.EventsOn(eventName, wrapper)
      return () => runtime.EventsOff(eventName, wrapper)
    },

    // EventsEmit — 发射事件到本地总线（HTTP 环境无后端推送时使用）
    EventsEmit: (eventName: string, ...args: unknown[]) => {
      const cbs = eventBus.get(eventName)
      if (!cbs) return
      for (const cb of cbs) {
        try {
          cb(...args)
        } catch (e) {
          console.error(`[runtime] EventsEmit ${eventName} 回调异常:`, e)
        }
      }
    },
  }
  win.runtime = runtime
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
 * 将 ReadableStream 的字节块解码为字符串块；abort() 主动关闭时静默结束。
 */
async function* streamChunks(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  decoder: TextDecoder,
): AsyncGenerator<string> {
  try {
    for (;;) {
      let chunk: ReadableStreamReadResult<Uint8Array>
      try {
        chunk = await reader.read()
      } catch {
        return // 连接被 abort() 主动关闭
      }
      if (chunk.done) return
      yield decoder.decode(chunk.value, { stream: true })
    }
  } finally {
    try {
      reader.releaseLock()
    } catch {
      /* ignore */
    }
  }
}

/**
 * 分发一帧：event: done 关闭并移除连接；其余显式 event 帧（connected 等）
 * 只用于生命周期；无 event 帧（message 语义）data 行 JSON.parse 后分发。
 */
function dispatchSSEFrame(eventName: string, frame: ParsedSSEFrame): void {
  if (frame.event === 'done') {
    console.log(`[runtime] SSE 流结束: ${eventName}`)
    activeSSE.get(eventName)?.close()
    activeSSE.delete(eventName)
    return
  }
  if (frame.event !== '') return
  if (frame.data.length === 0) return
  const raw = frame.data.join('\n')
  try {
    const data = JSON.parse(raw)
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

/**
 * 连接 SSE 流（fetch 流式替代 EventSource，T6-9.6）。
 * token 非空时携带 Authorization: Bearer；无 token 不带头（兼容旧版无鉴权后端）。
 * 断线（fetch 失败/连接中断/流意外结束）5 秒后若有订阅则自动重连。
 */
function connectSSE(eventName: string): void {
  const token = getHttpToken()
  const headers: Record<string, string> = {}
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const url = `/api/stream?id=${encodeURIComponent(eventName)}`
  console.log(`[runtime] 连接 SSE: ${url}`)

  const controller = new AbortController()
  const conn: SSEConnection = {
    close: () => controller.abort(),
  }
  // 同步登记，保证 EventsOff / 重复 EventsOn 的语义与旧 EventSource 一致。
  activeSSE.set(eventName, conn)

  void (async () => {
    const resp = await fetch(url, { headers, signal: controller.signal })
    if (!resp.ok) {
      throw new Error(`SSE HTTP ${resp.status}`)
    }
    if (!resp.body) {
      throw new Error('SSE: 响应无 body')
    }
    const frames = parseSSEStream(streamChunks(resp.body.getReader(), new TextDecoder()))
    for await (const frame of frames) {
      dispatchSSEFrame(eventName, frame)
      // done 事件已关闭并删除条目，停止消费本连接。
      if (!activeSSE.has(eventName)) {
        return
      }
    }
    // 流自然结束（服务端断开/网络中断）：视为断线，走重连。
    throw new Error('SSE 流意外结束')
  })().catch((err: unknown) => {
    // 主动关闭（EventsOff / done）不重连。
    if (controller.signal.aborted) {
      return
    }
    console.error(`[runtime] SSE 连接错误: ${eventName}，5 秒后重连:`, err)
    activeSSE.delete(eventName)
    setTimeout(() => {
      if (eventBus.has(eventName) && eventBus.get(eventName)!.size > 0) {
        connectSSE(eventName)
      }
    }, 5000)
  })
}
