/**
 * httpToken.ts — HTTP 调试桥接的一次性 token（S2-2 安全收敛）
 *
 * 后端在 GAEA_HTTP_PORT 启用桥接时，/api/rpc 与 /api/stream 要求携带
 * 一次性 token（GAEA_HTTP_TOKEN 或每次启动自动生成并打印在日志）。
 * 前端 token 来源优先级：
 *   1. URL 查询参数 ?token=（网页端调试最直观）
 *   2. window.__GAEA_HTTP_TOKEN（宿主注入，如调试页内嵌）
 *   3. localStorage['gaea_http_token']（操作者在 DevTools 手动设置）
 * 找不到时返回空串——后端未启用鉴权（旧版本）时无需携带，鉴权启用时
 * 请求会得到 401 并提示检查 token。
 */

export function getHttpToken(): string {
  if (typeof window === 'undefined') return ''
  try {
    const fromQuery = new URLSearchParams(window.location.search).get('token')
    if (fromQuery) return fromQuery
  } catch {
    /* ignore malformed query */
  }
  const injected = (window as unknown as Record<string, unknown>).__GAEA_HTTP_TOKEN
  if (typeof injected === 'string' && injected !== '') return injected
  try {
    return window.localStorage.getItem('gaea_http_token') ?? ''
  } catch {
    return ''
  }
}
