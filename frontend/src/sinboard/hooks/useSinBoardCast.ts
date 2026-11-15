// useSinBoardCast.ts — 原罪右栏「角色」卡数据源（刀6a fork useSinCast）：
// 键=书 slug（SinBoardCast* 板载绑定，无话题守卫）；角色库列表走共享
// CharacterList（跨板块资产层）。
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { errText } from '../../utils/errText'
import { app } from '../../gaea/lib/bridge'

export interface SinCastCharacter {
  id: string
  name: string
  gender?: string
  age?: string
  tags?: string[]
  roleType?: string
  portraitUrl?: string
  personality?: string
  appearance?: string
  background?: string
}

const LIBRARY_PAGE_SIZE = 200

function toCharacter(raw: Record<string, unknown>): SinCastCharacter {
  const str = (v: unknown) => (typeof v === 'string' ? v : undefined)
  return {
    id: str(raw.id) ?? '',
    name: str(raw.name) ?? '',
    gender: str(raw.gender),
    age: str(raw.age),
    tags: Array.isArray(raw.tags) ? (raw.tags as unknown[]).filter((t): t is string => typeof t === 'string') : undefined,
    roleType: str(raw.roleType),
    portraitUrl: str(raw.portraitUrl),
    personality: str(raw.personality),
    appearance: str(raw.appearance),
    background: str(raw.background),
  }
}

export interface UseSinBoardCastResult {
  library: SinCastCharacter[]
  libraryLoading: boolean
  libraryError: string
  castIds: string[]
  cast: SinCastCharacter[]
  saving: boolean
  saveCast: (ids: string[]) => Promise<{ ok: boolean; message: string }>
  reloadLibrary: () => void
  ensureLibrary: () => void
}

export function useSinBoardCast(slug: string): UseSinBoardCastResult {
  const [library, setLibrary] = useState<SinCastCharacter[]>([])
  const [libraryLoading, setLibraryLoading] = useState(false)
  const [libraryError, setLibraryError] = useState('')
  const [castIds, setCastIds] = useState<string[]>([])
  const [saving, setSaving] = useState(false)
  const [reloadTick, setReloadTick] = useState(0)
  const [libRequested, setLibRequested] = useState(false)
  const slugRef = useRef(slug)
  slugRef.current = slug

  useEffect(() => {
    if (!libRequested && reloadTick === 0) return
    let live = true
    setLibraryLoading(true)
    app.CharacterList('', '', false, 1, LIBRARY_PAGE_SIZE)
      .then((res: unknown) => {
        if (!live) return
        const items = (res as { items?: unknown[] } | null)?.items
        const list = Array.isArray(items)
          ? items.map((it) => toCharacter(it as Record<string, unknown>)).filter((c) => c.id !== '')
          : []
        setLibrary(list)
        setLibraryError('')
      })
      .catch((err: unknown) => {
        if (!live) return
        setLibrary([])
        setLibraryError(errText(err, '角色库读取失败'))
      })
      .finally(() => { if (live) setLibraryLoading(false) })
    return () => { live = false }
  }, [reloadTick, libRequested])

  useEffect(() => {
    let live = true
    if (!slug) {
      setCastIds([])
      return
    }
    app.SinBoardCastGet(slug)
      .then((ids: string[]) => { if (live) setCastIds(Array.isArray(ids) ? ids : []) })
      .catch(() => { if (live) setCastIds([]) })
    return () => { live = false }
  }, [slug])

  const saveCast = useCallback(async (ids: string[]): Promise<{ ok: boolean; message: string }> => {
    const key = slug
    if (!key) return { ok: false, message: '未绑定书工作空间' }
    setSaving(true)
    try {
      const effective = await app.SinBoardCastSet(key, ids)
      if (slugRef.current !== key) {
        return { ok: true, message: '' }
      }
      setCastIds(Array.isArray(effective) ? effective : [])
      return { ok: true, message: '' }
    } catch (err) {
      return { ok: false, message: errText(err, '角色保存失败') }
    } finally {
      setSaving(false)
    }
  }, [slug])

  const cast = useMemo(
    () => castIds.map((id) => library.find((c) => c.id === id)).filter((c): c is SinCastCharacter => !!c),
    [castIds, library],
  )

  const reloadLibrary = useCallback(() => setReloadTick((n) => n + 1), [])
  const ensureLibrary = useCallback(() => setLibRequested(true), [])

  return { library, libraryLoading, libraryError, castIds, cast, saving, saveCast, reloadLibrary, ensureLibrary }
}
