import { app } from '../../gaea/lib/bridge'
import { saveExportBlob } from '../../gaea/lib/saveFile'
import { subscribeWailsEvent } from '../../gaea/lib/wailsEvents'
import type { PantheonApi, PantheonEvent } from './types'

export const pantheonApi: PantheonApi = app

export function subscribePantheonEvents(handler: (event: PantheonEvent) => void): () => void {
  const rt = (window as unknown as {
    runtime?: { EventsOn?: (name: string, cb: (payload: unknown) => void) => unknown }
  }).runtime
  if (!rt?.EventsOn) return () => {}
  return subscribeWailsEvent(
    { EventsOn: (name, cb) => rt.EventsOn?.(name, cb) },
    'pantheon-event',
    (payload) => {
      if (payload && typeof payload === 'object') handler(payload as PantheonEvent)
    },
  )
}

export async function copyMarkdown(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}

/**
 * 导出统一走 saveFile 中立层双门：壳内 <a download> 不落盘（v4.162 实证，
 * 2026-10-29 圆桌裁决复核），必须经 GaeaSaveFileAs 系统「另存为」。
 * 返回 false = 用户在另存为对话框取消。
 */
export function downloadMarkdown(filename: string, text: string): Promise<boolean> {
  return saveExportBlob(new Blob([text], { type: 'text/markdown;charset=utf-8' }), filename)
}

export function downloadJSON(filename: string, value: unknown): Promise<boolean> {
  return saveExportBlob(
    new Blob([JSON.stringify(value, null, 2)], { type: 'application/json;charset=utf-8' }),
    filename,
  )
}
