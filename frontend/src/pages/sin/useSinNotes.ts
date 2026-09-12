// sin/useSinNotes.ts — 右栏面板「设定/大纲」数据源（v4.263）。
//
// 数据是 AI 写作的工作底稿：sin_notes（设定集）/ sin_outline（大纲）工具落
// <用户配置目录>/gaea/sin/notes/<故事 id>.json，SinNotesGet 只读同源文件。
// 读取时机：切故事 + 调用方在每个回合结束后 reload（工具可能在回合里写了
// 便签/大纲）；本 hook 只做只读收敛，不做写路径。

import { useCallback, useEffect, useState } from 'react'
import { app } from '../../gaea/lib/bridge'

export interface SinNotesDoc {
  notes: string[]
  outline: string
}

const EMPTY: SinNotesDoc = { notes: [], outline: '' }

function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}

export interface UseSinNotesResult {
  doc: SinNotesDoc
  error: string
  loading: boolean
  /** 回合结束/需要时重读（切故事自动触发，无需手动调）。 */
  reload: () => void
}

export function useSinNotes(activeId: string): UseSinNotesResult {
  const [doc, setDoc] = useState<SinNotesDoc>(EMPTY)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [tick, setTick] = useState(0)

  useEffect(() => {
    let live = true
    if (!activeId) {
      setDoc(EMPTY)
      setError('')
      return
    }
    setLoading(true)
    app.SinNotesGet(activeId)
      .then((res) => {
        if (!live) return
        const notes = Array.isArray(res?.notes)
          ? (res.notes as unknown[]).filter((n): n is string => typeof n === 'string')
          : []
        setDoc({ notes, outline: typeof res?.outline === 'string' ? res.outline : '' })
        setError('')
      })
      .catch((err: unknown) => {
        if (!live) return
        setDoc(EMPTY)
        setError(errText(err, '便签读取失败'))
      })
      .finally(() => { if (live) setLoading(false) })
    return () => { live = false }
  }, [activeId, tick])

  const reload = useCallback(() => setTick((n) => n + 1), [])

  return { doc, error, loading, reload }
}
