/**
 * useAttributionState — 模型中心「成本归因」状态 Hook（阶段七 7.1-2 C 线）
 *
 * 加载归因账本（ledger）与改绑建议（suggestions，Promise.all 并行），
 * 承接采纳/忽略两个用户动作：成功后 message 提示并重拉建议列表；
 * 采纳在后端经 SetFeatureModel 即时重建，绑定面靠既有 feature-model-changed
 * 事件自动刷新，前端不手动刷绑定区（建议制红线：不自动改绑，采纳=用户动作）。
 */
import { useCallback, useEffect, useState } from 'react'
import { message } from 'antd'
import {
  applyRouteSuggestion,
  getRouteLedger,
  getRouteSuggestions,
  ignoreRouteSuggestion,
  type RouteLedgerView,
  type RouteSuggestion,
} from '../../../api/engines'

/** 提取错误消息（unknown 收窄；无 message 用 fallback） */
function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}

export interface AttributionState {
  ledger: RouteLedgerView | null
  suggestions: RouteSuggestion[]
  loading: boolean
  loadError: string | null
  /** 采纳/忽略进行中的建议 ID（建议卡按钮 loading 态） */
  actingId: string | null
  reload: () => Promise<void>
  handleApply: (id: string) => Promise<void>
  handleIgnore: (id: string) => Promise<void>
}

export function useAttributionState(): AttributionState {
  const [ledger, setLedger] = useState<RouteLedgerView | null>(null)
  const [suggestions, setSuggestions] = useState<RouteSuggestion[]>([])
  const [loading, setLoading] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [actingId, setActingId] = useState<string | null>(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setLoadError(null)
    try {
      const [l, s] = await Promise.all([getRouteLedger(), getRouteSuggestions()])
      setLedger(l ?? null)
      setSuggestions(s?.suggestions ?? [])
    } catch (err: unknown) {
      setLedger(null)
      setSuggestions([])
      setLoadError(errText(err, '加载成本归因数据失败'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void reload() }, [reload])

  // 动作后仅重拉建议列表（账本是历史聚合，改绑不改写它）
  const refreshSuggestions = useCallback(async () => {
    try {
      const s = await getRouteSuggestions()
      setSuggestions(s?.suggestions ?? [])
    } catch { /* 静默降级：建议刷新失败不打扰账本展示 */ }
  }, [])

  const handleApply = useCallback(async (id: string) => {
    setActingId(id)
    try {
      await applyRouteSuggestion(id)
      message.success('已改绑')
      await refreshSuggestions()
    } catch (err: unknown) {
      message.error(errText(err, '改绑失败'))
    } finally {
      setActingId(null)
    }
  }, [refreshSuggestions])

  const handleIgnore = useCallback(async (id: string) => {
    setActingId(id)
    try {
      await ignoreRouteSuggestion(id)
      message.success('已忽略该建议')
      await refreshSuggestions()
    } catch (err: unknown) {
      message.error(errText(err, '忽略失败'))
    } finally {
      setActingId(null)
    }
  }, [refreshSuggestions])

  return {
    ledger,
    suggestions,
    loading,
    loadError,
    actingId,
    reload,
    handleApply,
    handleIgnore,
  }
}
