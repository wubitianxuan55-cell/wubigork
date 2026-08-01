import { useState, useEffect, useCallback } from 'react'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

/**
 * 功能模型 hook — 读取并实时监听某功能的绑定模型（持久化，重启不丢）。
 * feature: 'chat' | 'whisper' | 'novel' | 'office'
 */
export function useFeatureModel(feature: string): { engine: string; model: string } {
  const [m, setM] = useState<{ engine: string; model: string }>({ engine: '', model: '' })

  const refresh = useCallback(async () => {
    try {
      const r: any = await App.GetFeatureModel(feature)
      setM({ engine: r?.engine || '', model: r?.model || '' })
    } catch (_) {}
  }, [feature])

  useEffect(() => {
    refresh()
    let unsub: any
    try {
      unsub = EventsOn('feature-model-changed', (d: any) => {
        if (d?.feature === feature) setM({ engine: d.engine || '', model: d.model || '' })
      })
    } catch (_) {}
    return () => {
      try { if (typeof unsub === 'function') unsub() } catch (_) {}
    }
  }, [refresh, feature])

  return m
}
