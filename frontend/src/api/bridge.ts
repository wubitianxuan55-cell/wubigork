/**
 * bridge.ts — 移动端 RPC 桥接层
 *
 * 在 HTTP 环境下（非 Wails 原生），将 window.go.app.App.xxx() 调用
 * 透明代理为 fetch('/api/rpc') 调用，使现有前端代码无需修改即可在移动端运行。
 */

const API_BASE = ''

interface RPCResponse {
  result?: unknown
  error?: string
}

/**
 * 检测是否运行在 Wails 原生环境中
 */
function isWailsNative(): boolean {
  return !!(window as any).go?.app?.App
}

/**
 * 移动端 RPC 调用 — 通过 HTTP POST /api/rpc 调用后端方法
 */
async function rpcCall(method: string, ...args: unknown[]): Promise<unknown> {
  const res = await fetch(`${API_BASE}/api/rpc`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ method, args }),
  })
  if (!res.ok) {
    throw new Error(`RPC 请求失败: ${res.status} ${res.statusText}`)
  }
  const data: RPCResponse = await res.json()
  if (data.error) {
    throw new Error(data.error)
  }
  return data.result
}

/**
 * 创建 App 代理对象 — 拦截所有方法调用并转发到 RPC
 */
function createAppProxy(): Record<string, (...args: unknown[]) => Promise<unknown>> {
  return new Proxy(
    {},
    {
      get(_target, prop: string) {
        return (...args: unknown[]) => rpcCall(prop, ...args)
      },
    },
  ) as Record<string, (...args: unknown[]) => Promise<unknown>>
}

/**
 * 初始化桥接层 — 在 App.tsx 最早时机调用
 *
 * 效果：
 *  - 非 Wails 环境：创建 window.go.app.App 的 HTTP 代理
 *  - Wails 环境：不做任何事（已有原生绑定）
 */
export function initBridge(): void {
  // 避免重复初始化
  if ((window as any).__bridge_initialized) return
  ;(window as any).__bridge_initialized = true

  if (isWailsNative()) {
    console.log('[bridge] Wails 原生环境，跳过桥接')
    return
  }

  console.log('[bridge] 移动端 HTTP 模式，创建 RPC 代理')

  // 构建 window.go.app.App 代理
  if (!(window as any).go) {
    ;(window as any).go = {}
  }
  if (!(window as any).go.app) {
    ;(window as any).go.app = {}
  }
  ;(window as any).go.app.App = createAppProxy()
}
