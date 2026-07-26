import { useState, useCallback } from 'react'
import { message } from 'antd'

/** Branch — 剧情分支数据结构 */
export interface Branch {
  id: string
  title: string
  summary: string
  characters_involved: string[]
  core_conflict: string
  foreshadow_impact: string
  tone: string
}

/** toneColors — 语气 → 颜色映射 */
export const toneColors: Record<string, string> = {
  '紧张': '#f87171', '温暖': '#f59e0b', '悲伤': '#60a5fa',
  '欢乐': '#4ade80', '阴沉': '#9ca3af', '悬疑': '#c084fc',
  '史诗': '#a78bfa',
}

/** usePlotBranch — 剧情分支脑暴 + 应用逻辑 */
export function usePlotBranch(nodeID: string, onApplied?: () => void) {
  const [loading, setLoading] = useState(false)
  const [branches, setBranches] = useState<Branch[]>([])
  const [selected, setSelected] = useState<number>(-1)
  const [applying, setApplying] = useState(false)

  const handleBrainstorm = useCallback(async () => {
    if (!nodeID) return
    setLoading(true)
    setBranches([])
    setSelected(-1)
    try {
      // @ts-ignore
      const result = await window.go.app.App.BrainstormBranches(nodeID)
      if (result?.branches) setBranches(result.branches)
    } catch (err: any) {
      message.error(err.message || '推理失败')
    } finally { setLoading(false) }
  }, [nodeID])

  const handleApply = useCallback(async (manualMode: boolean, manualInput: string) => {
    if (selected < 0 && !manualMode) { message.warning('请选择分支或手工录入'); return }
    setApplying(true)
    try {
      // @ts-ignore
      await window.go.app.App.ApplyBranch(
        nodeID,
        manualMode ? -1 : selected,
        manualMode ? manualInput : '',
      )
      message.success('分支已应用，大纲已更新')
      onApplied?.()
    } catch (err: any) {
      message.error(err.message || '应用分支失败')
    } finally { setApplying(false) }
  }, [nodeID, selected, onApplied])

  return {
    branches, setBranches,
    selected, setSelected,
    loading, setLoading,
    applying, setApplying,
    handleBrainstorm,
    handleApply,
  }
}
