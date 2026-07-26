/**
 * 统一日志工具
 * 替代全站散落的 console.error，支持可开关、可分级
 */

const LOG_LEVEL = import.meta.env.DEV ? 'debug' : 'warn'

const levels: Record<string, number> = { debug: 0, info: 1, warn: 2, error: 3, silent: 4 }
const currentLevel = levels[LOG_LEVEL] ?? 2

/** 开发/调试日志 */
export function logDebug(tag: string, ...args: unknown[]) {
  if (currentLevel <= 0) console.debug(`[${tag}]`, ...args)
}

/** 一般信息日志 */
export function logInfo(tag: string, ...args: unknown[]) {
  if (currentLevel <= 1) console.info(`[${tag}]`, ...args)
}

/** 可恢复的错误日志 */
export function logWarn(tag: string, ...args: unknown[]) {
  if (currentLevel <= 2) console.warn(`[${tag}]`, ...args)
}

/** 错误日志（始终输出） */
export function logError(tag: string, error: unknown) {
  console.error(`[${tag}]`, error)
}
