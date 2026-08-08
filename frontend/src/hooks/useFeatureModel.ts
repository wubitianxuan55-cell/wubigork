import { useState, useEffect, useCallback } from 'react'
import * as App from '../../wailsjs/go/app/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

/**
 * 功能模型 hook — 读取并实时监听某功能的绑定模型（持久化，重启不丢）。
 * feature: 'chat' | 'whisper' | 'novel' | 'office' | 'gaea' | 'characterlib'
 * enabled: 功能级启停（FeatureModelBar 启停语义，默认启用）
 */
export function useFeatureModel(feature: string): { engine: string; model: string; enabled: boolean } {
  const [m, setM] = useState<{ engine: string; model: string; enabled: boolean }>({ engine: '', model: '', enabled: true })

  const refresh = useCallback(async () => {
    try {
      const r: any = await App.GetFeatureModel(feature)
      let enabled = true
      try { enabled = !!(await App.GetFeatureModelEnabled(feature)) } catch (_) {}
      setM({ engine: r?.engine || '', model: r?.model || '', enabled })
    } catch (_) {}
  }, [feature])

  useEffect(() => {
    refresh()
    let unsub: any
    try {
      unsub = EventsOn('feature-model-changed', (d: any) => {
        if (d?.feature !== feature) return
        setM(prev => ({
          engine: d.engine ?? prev.engine,
          model: d.model ?? prev.model,
          enabled: d.enabled !== undefined ? !!d.enabled : prev.enabled,
        }))
      })
    } catch (_) {}
    return () => {
      try { if (typeof unsub === 'function') unsub() } catch (_) {}
    }
  }, [refresh, feature])

  return m
}
