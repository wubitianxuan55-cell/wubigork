import { message } from 'antd'

/**
 * handleError — 统一错误处理，替换全站 ~60+ 处 try/catch{message.error} 模式
 * @param context 错误上下文描述（如「加载角色」）
 * @param err     捕获的异常
 * @param fallbackMsg 兜底消息（未指定时自动从 context 生成）
 */
export function handleError(context: string, err: unknown, fallbackMsg?: string) {
  const msg = fallbackMsg || `${context}失败`
  const detail = err instanceof Error ? err.message : typeof err === 'string' ? err : String(err)
  console.error(`[${context}]`, err)
  message.error(`${msg}${detail ? ': ' + detail : ''}`)
}

/**
 * wrapAsync — 包装异步函数，自动处理错误并调用 message.error
 * @param context 错误上下文
 * @param fn      异步函数
 */
export function wrapAsync<T>(context: string, fn: () => Promise<T>): Promise<T | undefined> {
  return fn().catch((err) => {
    handleError(context, err)
    return undefined
  })
}
