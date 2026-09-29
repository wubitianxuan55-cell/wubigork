// sin/useSinCast.ts — 原罪 × 角色库（v4.257「原罪能够使用角色库内容」）。
//
// 数据流：角色库列表经既有 CharacterList（CharlibB 门面，全库不分空间——角色库
// 是跨板块共享资产层）；本故事的选择经 SinCastGet/SinCastSet 存原罪自有配置
// （<用户配置目录>/gaea/sin/cast.json，不写办公工作区）。
// 提示词侧：后端 SinStream 会把这些角色的设定注入故事上下文（见 sin_prompt.go）。

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
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
  /** 保存选择（去重/悬空 id 由后端过滤，返回生效清单）。
   *  返回 ok=false 时 message 是失败原因（调用方透出，别让保存静默假成功）。 */
  saveCast: (ids: string[]) => Promise<{ ok: boolean; message: string }>
  /** 角色库重读（设定卡生成后刷新等）。首拉前调用会直接触发首次加载。 */
  reloadLibrary: () => void
  /** 首次需要展示库列表时调用（面板展开/选择器打开）：懒加载，进板块不拉 200 条。 */
  ensureLibrary: () => void
}

export function useSinCast(activeId: string): UseSinCastResult {
  const [library, setLibrary] = useState<SinCastCharacter[]>([])
  const [libraryLoading, setLibraryLoading] = useState(false)
  const [libraryError, setLibraryError] = useState('')
  const [castIds, setCastIds] = useState<string[]>([])
  const [saving, setSaving] = useState(false)
  const [reloadTick, setReloadTick] = useState(0)
  // 懒加载闸：面板从未展开/选择器从未打开时不拉全量库（进板块零额外请求）
  const [libRequested, setLibRequested] = useState(false)
  // 最新 activeId（saveCast 归属校验用：闭包里的 activeId 是发起时的故事）
  const activeIdRef = useRef(activeId)
  activeIdRef.current = activeId

  // ── 角色库列表（首次 ensureLibrary 或显式 reloadLibrary 后才拉） ──
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

  const saveCast = useCallback(async (ids: string[]): Promise<{ ok: boolean; message: string }> => {
    const storyId = activeId
    if (!storyId) return { ok: false, message: '没有选中的故事' }
    setSaving(true)
    try {
      const effective = await app.SinCastSet(storyId, ids)
      // 归属校验（v4.426）：保存期间切了故事就不再回填——历史实现会把 A 故事
      // 的生效清单写进 B 故事的面板状态，用户再点「带入故事」就把 B 覆盖成 A。
      if (activeIdRef.current !== storyId) {
        return { ok: true, message: '' }
      }
      setCastIds(Array.isArray(effective) ? effective : [])
      return { ok: true, message: '' }
    } catch (err) {
      // 失败如实透出（历史实现无 catch：chip 不动、选择器照关、unhandled rejection，
      // 用户以为保存成功了）
      return { ok: false, message: errText(err, '角色保存失败') }
    } finally {
      setSaving(false)
    }
  }, [activeId])

  const cast = useMemo(
    () => castIds.map((id) => library.find((c) => c.id === id)).filter((c): c is SinCastCharacter => !!c),
    [castIds, library],
  )

  const reloadLibrary = useCallback(() => setReloadTick((n) => n + 1), [])
  const ensureLibrary = useCallback(() => setLibRequested(true), [])

  return { library, libraryLoading, libraryError, castIds, cast, saving, saveCast, reloadLibrary, ensureLibrary }
}
