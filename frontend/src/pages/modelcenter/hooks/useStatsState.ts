/**
 * useStatsState — 模型中心「调用统计」状态 Hook（T6-6.4 UI 拆分）
 *
 * 归集模型调用统计抽屉的全部状态（汇总、排序、趋势范围）与加载/重置逻辑。
 * 调用统计抽屉打开时每 15s 定时刷新（statsOpen 由顶层传入）。
 */
import { useCallback, useEffect, useMemo, useState } from 'react'
import { message } from 'antd'
import { getModelCallStats, resetModelCallStats, type ModelStatsSummary } from '../../../api/engines'
import { type StatsSort, type TrendDatum, type TrendRange } from '../charts'
import { usePollingGate } from '../../../hooks/usePollingGate'

export interface StatsState {
  callStats: ModelStatsSummary | null
  loadError: string | null
  statsSort: StatsSort
  setStatsSort: (v: StatsSort) => void
  trendRange: TrendRange
  setTrendRange: (v: TrendRange) => void
  trendData: TrendDatum[]
  loadCallStats: () => Promise<void>
  handleResetCallStats: () => Promise<void>
}

export function useStatsState(statsOpen: boolean): StatsState {
  const [callStats, setCallStats] = useState<ModelStatsSummary | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [statsSort, setStatsSort] = useState<StatsSort>('calls')
  const [trendRange, setTrendRange] = useState<TrendRange>('7d')
  // v4.5.2：统计轮询接入系统级后台轮询门控（页面不可见时空转零成本）
  const gate = usePollingGate()

  const loadCallStats = useCallback(async () => {
    try {
      const s = await getModelCallStats()
      setCallStats(s ?? null)
      setLoadError(null)
    } catch (err: unknown) {
      setLoadError(err instanceof Error ? err.message : '加载调用统计失败')
    }
  }, [])

  // 3.0「引擎控制台」：右侧统计检查器常显，进入页面即加载；
  // 页面 keep-alive 常驻，统计需自行定期刷新——基础 30s 轮询，
  // 统计抽屉打开期间加快到 15s（原实现仅在抽屉打开时轮询，面板数据会过期）。
  useEffect(() => {
    const tick = () => { if (gate) void loadCallStats() }
    tick()
    const timer = window.setInterval(tick, statsOpen ? 15000 : 30000)
    return () => window.clearInterval(timer)
  }, [statsOpen, loadCallStats, gate])

  const handleResetCallStats = async () => {
    try {
      await resetModelCallStats()
      setCallStats(null)
      message.success('模型调用统计已清空')
      loadCallStats()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '重置失败')
    }
  }

  // 趋势数据：后端按小时返回；「今日」只取当天的小时桶，
  // 「7天/30天」按天聚合（标签只显示日期，不再带小时）。
  const trendData = useMemo<TrendDatum[]>(() => {
    if (!callStats?.trend?.length) return []
    const now = new Date()
    const todayKey = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
    const agg = new Map<string, TrendDatum>()
    for (const p of callStats.trend) {
      const hourly = trendRange === 'today'
      if (hourly && !p.time.startsWith(todayKey)) continue // 今日只显示当天的数据
      const key = hourly ? p.time : p.time.slice(0, 10)
      const label = hourly ? p.time.slice(11, 16) : p.time.slice(5, 10)
      const cur = agg.get(key)
      if (cur) {
        cur.calls += p.calls
        cur.successCalls += p.success_calls
        cur.failCalls += p.fail_calls
        cur.inputTokens += p.input_tokens
        cur.outputTokens += p.output_tokens
        cur.cost += p.cost
      } else {
        agg.set(key, {
          key,
          label,
          calls: p.calls,
          successCalls: p.success_calls,
          failCalls: p.fail_calls,
          inputTokens: p.input_tokens,
          outputTokens: p.output_tokens,
          cost: p.cost,
        })
      }
    }
    const list = Array.from(agg.values()).sort((a, b) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0))
    const limit = trendRange === 'today' ? 24 : trendRange === '7d' ? 7 : 30
    return list.slice(-limit)
  }, [callStats, trendRange])

  return {
    callStats,
    loadError,
    statsSort,
    setStatsSort,
    trendRange,
    setTrendRange,
    trendData,
    loadCallStats,
    handleResetCallStats,
  }
}
