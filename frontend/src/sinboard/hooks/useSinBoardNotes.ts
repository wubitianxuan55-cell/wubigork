// useSinBoardNotes.ts — 原罪右栏「大纲/设定」卡数据源（刀6a fork useSinNotes）：
// 键=书 slug（SinBoardNotes* 板载绑定，无话题守卫）；冲突口径与 legacy 同串。
import { useCallback, useEffect, useRef, useState } from 'react'
import { errText } from '../../utils/errText'
import { app } from '../../gaea/lib/bridge'

export interface SinBoardNotesDoc {
  notes: string[]
  outline: string
}

/** 后端冲突错误固定前缀（与 Go 侧 sinConflictPrefix 同串）。 */
export const SIN_CONFLICT_PREFIX = '底稿冲突：'

const EMPTY: SinBoardNotesDoc = { notes: [], outline: '' }

export interface UseSinBoardNotesResult {
  doc: SinBoardNotesDoc
  error: string
  loading: boolean
  reload: () => void
  save: (baseline: SinBoardNotesDoc, outline: string, notes: string[], force: boolean) => Promise<{ ok: boolean; conflict: boolean; message: string }>
}

export function useSinBoardNotes(slug: string): UseSinBoardNotesResult {
  const [doc, setDoc] = useState<SinBoardNotesDoc>(EMPTY)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [tick, setTick] = useState(0)
  const slugRef = useRef(slug)
  slugRef.current = slug

  useEffect(() => {
    let live = true
    if (!slug) {
      setDoc(EMPTY)
      setError('')
      return
    }
    setLoading(true)
    app.SinBoardNotesGet(slug)
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
  }, [slug, tick])

  const reload = useCallback(() => setTick((n) => n + 1), [])

  const save = useCallback(async (baseline: SinBoardNotesDoc, outline: string, notes: string[], force: boolean) => {
    const key = slug
    try {
      const res = await app.SinBoardNotesSave(
        key,
        JSON.stringify({ notes: baseline.notes, outline: baseline.outline }),
        outline,
        JSON.stringify(notes),
        force,
      )
      if (slugRef.current !== key) {
        return { ok: true, conflict: false, message: '' }
      }
      const nextNotes = Array.isArray(res?.notes)
        ? (res.notes as unknown[]).filter((n): n is string => typeof n === 'string')
        : []
      setDoc({ notes: nextNotes, outline: typeof res?.outline === 'string' ? res.outline : '' })
      return { ok: true, conflict: false, message: '' }
    } catch (err) {
      const message = errText(err, '保存失败')
      return { ok: false, conflict: message.startsWith(SIN_CONFLICT_PREFIX), message }
    }
  }, [slug])

  return { doc, error, loading, reload, save }
}
