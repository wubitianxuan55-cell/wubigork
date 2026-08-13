import { useState } from 'react'

// 模型中心「置顶/收藏」持久化到 localStorage，跨页面切换与重启保留。
const PIN_KEY = 'gaea.modelcenter.pinnedModels'

export function loadPinnedModels(): string[] {
  try {
    const raw = localStorage.getItem(PIN_KEY)
    const arr = raw ? JSON.parse(raw) : []
    return Array.isArray(arr) ? arr.filter((x): x is string => typeof x === 'string') : []
  } catch {
    return []
  }
}

export function persistPinnedModels(ids: string[]): void {
  try {
    localStorage.setItem(PIN_KEY, JSON.stringify(ids))
  } catch {
    // 隐私模式/配额异常时静默降级，不影响主流程
  }
}

export function togglePinnedModel(id: string): string[] {
  const cur = loadPinnedModels()
  const next = cur.includes(id) ? cur.filter(x => x !== id) : [...cur, id]
  persistPinnedModels(next)
  return next
}

// 供模型分区复用的置顶状态 hook。
export function usePinnedModels(): [string[], (id: string) => void] {
  const [pinned, setPinned] = useState<string[]>(loadPinnedModels)
  const toggle = (id: string) => setPinned(togglePinnedModel(id))
  return [pinned, toggle]
}
