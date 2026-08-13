/**
 * bridge.ts — 移动端 RPC 桥接层
 *
 * 在 HTTP 环境下（非 Wails 原生），将 window.go.app.App.xxx() 调用
 * 透明代理为 fetch('/api/rpc') 调用，使现有前端代码无需修改即可在移动端运行。
 */

import { getHttpToken } from './httpToken'

const API_BASE = ''

interface RPCResponse {
  result?: unknown
  error?: string
}

/**
 * 检测是否运行在 Wails 原生环境中。
 * S2-3 绑定面拆分后 window.go.app 下有多个板块门面（CoreB/OfficeB/...），
 * 只要存在任一绑定对象即视为原生环境。
 */
function isWailsNative(): boolean {
  const goApp = (window as any).go?.app
  return !!goApp && typeof goApp === 'object' && Object.keys(goApp).length > 0
}

/**
 * 旧代码直接调用 window.go.app.App.Xxx()（gaea 拆分前形态）。
 * S2-3 后原生绑定对象为各板块门面——这里为兼容层补一个 window.go.app.App
 * 代理，按方法名路由到对应门面，旧调用点零改动。
 */
function ensureLegacyAppProxy(): void {
  const goApp = (window as any).go?.app
  if (!goApp || typeof goApp !== 'object') return
  if (goApp.App) return
  goApp.App = new Proxy(
    {},
    {
      get(_t, prop: string) {
        if (prop === 'then') return undefined // 避免被误判为 Promise
        for (const ns of Object.values(goApp)) {
          if (ns === goApp.App || ns === null || typeof ns !== 'object') continue
          const v = (ns as Record<string, unknown>)[prop]
          if (typeof v === 'function') return (v as (...a: unknown[]) => unknown).bind(ns)
        }
        return undefined
      },
    },
  )
}

/**
 * 移动端 RPC 调用 — 通过 HTTP POST /api/rpc 调用后端方法
 * 后端启用一次性 token 鉴权时（S2-2），随请求携带 Authorization: Bearer。
 */
async function rpcCall(method: string, ...args: unknown[]): Promise<unknown> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = getHttpToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const res = await fetch(`${API_BASE}/api/rpc`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ method, args }),
  })
  if (res.status === 401) {
    throw new Error('桥接鉴权失败：请携带正确的 token（URL ?token= / __GAEA_HTTP_TOKEN / localStorage gaea_http_token）')
  }
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
    // S2-3：原生绑定为多个板块门面，为旧调用点补 window.go.app.App 兼容代理。
    ensureLegacyAppProxy()
    console.log('[bridge] Wails 原生环境，已就绪板块门面路由')
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
