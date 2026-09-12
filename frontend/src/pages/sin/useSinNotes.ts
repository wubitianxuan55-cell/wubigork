// sin/useSinNotes.ts — 右栏面板「设定/大纲」数据源（v4.263 只读；v4.266 加编辑保存）。
//
// 数据是 AI 写作的工作底稿：sin_notes（设定集）/ sin_outline（大纲）工具落
// <用户配置目录>/gaea/sin/notes/<故事 id>.json，SinNotesGet 只读同源文件。
// 读取时机：切故事 + 调用方在每个回合结束后 reload（工具可能在回合里写了
// 便签/大纲）。写路径（v4.266）：SinNotesSave 整包写回，锁内基线比对防他端
// 覆盖（冲突返回 conflict=true，调用方确认后 force 重试）。

import { useCallback, useEffect, useState } from 'react'
import { app } from '../../gaea/lib/bridge'

export interface SinNotesDoc {
  notes: string[]
  outline: string
}

/** 后端冲突错误固定前缀（与 Go 侧 sinConflictPrefix 同串）。 */
export const SIN_CONFLICT_PREFIX = '底稿冲突：'

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
  /**
   * 面板编辑保存（v4.266，整包写回大纲+设定集）。baseline=开始编辑时的快照；
   * 返回 conflict=true 表示底稿已被他端更新，调用方可让用户确认后 force 重试。
   */
  save: (baseline: SinNotesDoc, outline: string, notes: string[], force: boolean) => Promise<{ ok: boolean; conflict: boolean; message: string }>
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

  const save = useCallback(async (baseline: SinNotesDoc, outline: string, notes: string[], force: boolean) => {
    try {
      const res = await app.SinNotesSave(
        activeId,
        JSON.stringify({ notes: baseline.notes, outline: baseline.outline }),
        outline,
        JSON.stringify(notes),
        force,
      )
      const nextNotes = Array.isArray(res?.notes)
        ? (res.notes as unknown[]).filter((n): n is string => typeof n === 'string')
        : []
      setDoc({ notes: nextNotes, outline: typeof res?.outline === 'string' ? res.outline : '' })
      return { ok: true, conflict: false, message: '' }
    } catch (err) {
      const message = errText(err, '保存失败')
      return { ok: false, conflict: message.startsWith(SIN_CONFLICT_PREFIX), message }
    }
  }, [activeId])

  return { doc, error, loading, reload, save }
}
