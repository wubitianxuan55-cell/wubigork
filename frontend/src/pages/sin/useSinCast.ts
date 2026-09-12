// sin/useSinCast.ts — 原罪 × 角色库（v4.257「原罪能够使用角色库内容」）。
//
// 数据流：角色库列表经既有 CharacterList（CharlibB 门面，全库不分空间——角色库
// 是跨板块共享资产层）；本故事的选择经 SinCastGet/SinCastSet 存原罪自有配置
// （<用户配置目录>/gaea/sin/cast.json，不写办公工作区）。
// 提示词侧：后端 SinStream 会把这些角色的设定注入故事上下文（见 sin_prompt.go）。

import { useCallback, useEffect, useMemo, useState } from 'react'
import { app } from '../../gaea/lib/bridge'

/** 角色库角色最小展示面（对齐 characterlib.Character 的 JSON 字段）。 */
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

function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}

/** 归一化 CharacterList 的条目（缺字段诚实留空，不编造）。 */
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

export interface UseSinCastResult {
  /** 角色库全量（个人库上限 200 条，够用；更多再分页扩展）。 */
  library: SinCastCharacter[]
  libraryLoading: boolean
  libraryError: string
  /** 本故事已选角色（按选择顺序）。 */
  castIds: string[]
  cast: SinCastCharacter[]
  saving: boolean
  /** 保存选择（去重/悬空 id 由后端过滤，返回生效清单）。 */
  saveCast: (ids: string[]) => Promise<void>
  reloadLibrary: () => void
}

export function useSinCast(activeId: string): UseSinCastResult {
  const [library, setLibrary] = useState<SinCastCharacter[]>([])
  const [libraryLoading, setLibraryLoading] = useState(true)
  const [libraryError, setLibraryError] = useState('')
  const [castIds, setCastIds] = useState<string[]>([])
  const [saving, setSaving] = useState(false)
  const [reloadTick, setReloadTick] = useState(0)

  // ── 角色库列表 ──
  useEffect(() => {
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
  }, [reloadTick])

  // ── 当前故事的角色选择（切故事即重读；无故事清空） ──
  useEffect(() => {
    let live = true
    if (!activeId) {
      setCastIds([])
      return
    }
    app.SinCastGet(activeId)
      .then((ids: string[]) => { if (live) setCastIds(Array.isArray(ids) ? ids : []) })
      .catch(() => { if (live) setCastIds([]) })
    return () => { live = false }
  }, [activeId])

  const saveCast = useCallback(async (ids: string[]) => {
    if (!activeId) return
    setSaving(true)
    try {
      const effective = await app.SinCastSet(activeId, ids)
      setCastIds(Array.isArray(effective) ? effective : [])
    } finally {
      setSaving(false)
    }
  }, [activeId])

  const cast = useMemo(
    () => castIds.map((id) => library.find((c) => c.id === id)).filter((c): c is SinCastCharacter => !!c),
    [castIds, library],
  )

  const reloadLibrary = useCallback(() => setReloadTick((n) => n + 1), [])

  return { library, libraryLoading, libraryError, castIds, cast, saving, saveCast, reloadLibrary }
}
