// sin/sinBookDownload.ts — 书源下载会话（模块级单例，v4.426）。
//
// SinBookSourcePanel 挂在手风琴卡里：点其它卡即卸载——下载状态放组件 state 时，
// 切换卡会丢进度、丢终态提示、丢 jobId（v4.423 观察池立案项）。状态与事件订阅
// 挂模块单例（先例：同目录 illustrationQueue），面板重挂载即恢复显示，终态 toast
// 照常弹出。面板布局不动：五卡形态维持（一次聚焦一张卡），只把会话状态外提。

import { message } from 'antd'
import { subscribe, sinBooksourceChannel } from '../../events'

export type BookDownloadState =
  | { kind: 'idle' }
  | { kind: 'downloading'; jobId: string; done: number; total: number }

export type BookDownloadOutcome = 'done' | 'error' | 'cancel'

let downloadState: BookDownloadState = { kind: 'idle' }
let downloadUnsub: (() => void) | null = null
// 终态序号（done/error/取消都自增）：面板据此刷新成书清单（哪怕终态时面板不在挂载中）
let downloadTick = 0
// 终态去向：面板据此前进组件态（done 清选中；error 回填行内原因）
let downloadOutcome: BookDownloadOutcome | null = null
let downloadLastError = ''
const downloadSubs = new Set<() => void>()

function notifyDownloadSubs() {
  for (const cb of downloadSubs) cb()
}

function setDownloadState(next: BookDownloadState) {
  downloadState = next
  notifyDownloadSubs()
}

function finishTracking(outcome: BookDownloadOutcome) {
  downloadUnsub?.()
  downloadUnsub = null
  downloadOutcome = outcome
  downloadTick++
  if (downloadState.kind !== 'idle') setDownloadState({ kind: 'idle' })
}

/** 开局：登记 jobId 并接管该 job 的事件订阅（终态收尾，模块级长驻）。 */
export function beginBookDownload(jobId: string, total: number) {
  downloadUnsub?.()
  downloadOutcome = null
  downloadLastError = ''
  setDownloadState({ kind: 'downloading', jobId, done: 0, total })
  downloadUnsub = subscribe(sinBooksourceChannel(jobId), (data: unknown) => {
    const payload = (data ?? {}) as { type?: string; done?: number; total?: number; error?: string; result?: { chapters?: number } }
    if (payload.type === 'progress') {
      if (downloadState.kind === 'downloading') {
        setDownloadState({
          ...downloadState,
          done: payload.done ?? downloadState.done,
          total: payload.total ?? downloadState.total,
        })
      }
      return
    }
    if (payload.type === 'done') {
      // 完成提示走模块级 toast：终态落在面板卸载窗口（用户切了卡）也能被看见
      message.success(`成书完成${payload.result?.chapters ? `：${payload.result.chapters} 章` : ''}，已收进成书清单`)
      finishTracking('done')
      return
    }
    if (payload.type === 'error') {
      downloadLastError = payload.error || '下载失败'
      finishTracking('error')
    }
  })
}

/** 本地立即收尾（取消语义：后端中止，成书不落盘）。 */
export function cancelBookDownloadTracking() {
  finishTracking('cancel')
}

/** 当前会话快照 + 终态序号（面板用快照渲染、用 tick 触发清单刷新）。 */
export function bookDownloadSnapshot(): { state: BookDownloadState; tick: number } {
  return { state: downloadState, tick: downloadTick }
}

/** 终态去向与行内错误原因（面板在 tick 变化时读取）。 */
export function bookDownloadOutcome(): { outcome: BookDownloadOutcome | null; error: string } {
  return { outcome: downloadOutcome, error: downloadLastError }
}

/** 订阅会话变化（进度推进与终态都会通知；返回退订函数）。 */
export function subscribeBookDownload(cb: () => void): () => void {
  downloadSubs.add(cb)
  return () => { downloadSubs.delete(cb) }
}

/** 测试隔离：清空模块会话状态（生产不使用）。 */
export function __resetBookDownloadForTest(): void {
  downloadUnsub?.()
  downloadUnsub = null
  downloadState = { kind: 'idle' }
  downloadTick = 0
  downloadOutcome = null
  downloadLastError = ''
  downloadSubs.clear()
}
