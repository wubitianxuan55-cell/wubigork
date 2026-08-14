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

export interface StatsState {
  callStats: ModelStatsSummary | null
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
  const [statsSort, setStatsSort] = useState<StatsSort>('calls')
  const [trendRange, setTrendRange] = useState<TrendRange>('7d')

  const loadCallStats = useCallback(async () => {
    try {
      const s = await getModelCallStats()
      if (s) setCallStats(s)
    } catch (_) {}
  }, [])

  // 调用统计抽屉打开时定时刷新
  useEffect(() => {
    if (!statsOpen) return
    loadCallStats()
    const timer = window.setInterval(loadCallStats, 15000)
    return () => window.clearInterval(timer)
  }, [statsOpen, loadCallStats])

  const handleResetCallStats = async () => {
    try {
      await resetModelCallStats()
      setCallStats(null)
      message.success('模型调用统计已清空')
      loadCallStats()
    } catch (err: any) {
      message.error(err?.message || '重置失败')
    }
  }

  // 趋势数据：后端按小时返回，按当前范围聚合为小时或天粒度。
  const trendData = useMemo<TrendDatum[]>(() => {
    if (!callStats?.trend?.length) return []
    const agg = new Map<string, TrendDatum>()
    for (const p of callStats.trend) {
      const hourly = trendRange === 'today'
      const key = hourly ? p.time : p.time.slice(0, 10)
      const label = hourly ? p.time.slice(5, 16).replace('T', ' ') : p.time.slice(5)
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
    statsSort,
    setStatsSort,
    trendRange,
    setTrendRange,
    trendData,
    loadCallStats,
    handleResetCallStats,
  }
}
