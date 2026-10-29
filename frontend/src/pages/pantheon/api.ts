import { app } from '../../gaea/lib/bridge'
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

function downloadFile(filename: string, text: string, type: string): void {
  const blob = new Blob([text], { type })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  URL.revokeObjectURL(url)
}

export function downloadMarkdown(filename: string, text: string): void {
  downloadFile(filename, text, 'text/markdown;charset=utf-8')
}

export function downloadJSON(filename: string, value: unknown): void {
  downloadFile(filename, JSON.stringify(value, null, 2), 'application/json;charset=utf-8')
}
