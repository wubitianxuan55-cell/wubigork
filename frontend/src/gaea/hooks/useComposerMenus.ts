// Composer 拆分产物：/ 命令、/ 参数、@ 文件引用菜单状态机（行为零变化，T6-10.1）
import { useEffect, useMemo, useRef, useState } from 'react'
import { app as defaultApp } from '../lib/bridge'
import type { ComposerBridge } from '../lib/composerBridge'
import { slashQueryOf, atMentionOf } from '../lib/composer'
import type { CommandInfo, DirEntry, FileSearchHit, SlashArgsResult, SlashArgItem, AtEntry } from '../lib/types'
import { loadRecentFiles, recordRecentFile } from '../lib/recentFiles'
import { loadFileTreePrefs } from '../lib/fileTreePrefs'

export interface UseComposerMenusOptions {
  text: string
  debouncedText: string
  setTextCaretEnd: (next: string) => void
  cwd?: string
  bridge?: ComposerBridge
  filterCommand?: (name: string) => boolean
}

export function useComposerMenus({
  text, debouncedText, setTextCaretEnd, cwd, bridge = defaultApp, filterCommand,
}: UseComposerMenusOptions) {
  const [active, setActive] = useState(0)
  const [dismissed, setDismissed] = useState(false)
  const filterCommandRef = useRef(filterCommand)
  filterCommandRef.current = filterCommand

  // ── / 命令 ──
  const [commands, setCommands] = useState<CommandInfo[]>([])
  useEffect(() => {
    let live = true
    bridge.Commands().then((list) => {
      if (!live) return
      const filter = filterCommandRef.current
      setCommands(filter ? list.filter((command) => filter(command.name)) : list)
    }).catch(() => {})
    return () => { live = false }
  }, [bridge])
  const slashQuery = useMemo(() => slashQueryOf(text), [text])
  const slashMatches = useMemo(() => (slashQuery === null ? [] : commands.filter((c) => c.name.toLowerCase().includes(slashQuery)).slice(0, 8)), [slashQuery, commands])

  // ── 命令参数 ──
  const slashArgQuery = text.startsWith("/") && /\s/.test(text) ? text : null
  const [slashArgState, setSlashArgState] = useState<{
    query: string | null
    waiting: boolean
    result: SlashArgsResult | null
  }>({ query: null, waiting: false, result: null })
  useEffect(() => {
    if (slashArgQuery === null) {
      setSlashArgState({ query: null, waiting: false, result: null })
      return
    }
    const query = slashArgQuery
    setSlashArgState((prev) => ({
      query,
      waiting: true,
      result: prev.query === query ? prev.result : null,
    }))
    let live = true
    bridge.SlashArgs(query).then((r) => {
      if (!live) return
      const useful = (r.items ?? []).filter((it) => query.slice(0, r.from) + it.insert !== query)
      setSlashArgState({
        query,
        waiting: false,
        result: useful.length > 0 ? { items: useful, from: r.from } : null,
      })
      setActive(0)
    }).catch(() => {
      if (live) setSlashArgState({ query, waiting: false, result: null })
    })
    return () => { live = false; }
  }, [slashArgQuery, bridge])
  const argRes = slashArgState.query === text ? slashArgState.result : null
  const slashArgWaiting = slashArgState.query === text && slashArgState.waiting

  // ── @ 文件引用 ──
  const atMention = useMemo(() => atMentionOf(debouncedText), [debouncedText])
  const atRaw = atMention?.raw ?? null
  const atDir = atMention?.dir ?? ""
  const atFrag = atMention?.frag ?? ""
  const atActive = atRaw !== null
  const excludes = useMemo(() => loadFileTreePrefs(cwd).excludes, [cwd])
  const [entries, setEntries] = useState<DirEntry[]>([])
  const dirCache = useRef<Record<string, DirEntry[]>>({})
  useEffect(() => {
    if (!atActive) return
    const cacheKey = `${atDir}\0${excludes.join("\0")}`
    const cached = dirCache.current[cacheKey]
    if (cached) { setEntries(cached); return }
    let live = true
    const batch = bridge.GaeaListDirBatch
    const request = typeof batch === "function"
      ? batch([atDir], excludes).then((levels) => (levels.find((level) => level.path === atDir)?.entries ?? []) as DirEntry[])
      : bridge.ListDir(atDir).then((es) => es ?? [])
    request.then((list) => { dirCache.current[cacheKey] = list; if (live) setEntries(list) }).catch(() => {})
    return () => { live = false; }
  }, [atActive, atDir, excludes, bridge])
  // 工作区跨目录搜索（@ 引用增强：搜一下定位资料）
  const atSearchToken = atRaw === null ? null : `${atRaw}\0${excludes.join("\0")}`
  const [atSearchState, setAtSearchState] = useState<{
    token: string | null
    waiting: boolean
    hits: FileSearchHit[]
  }>({ token: null, waiting: false, hits: [] })
  useEffect(() => {
    if (atSearchToken === null) {
      setAtSearchState({ token: null, waiting: false, hits: [] })
      return
    }
    let live = true
    setAtSearchState({ token: atSearchToken, waiting: true, hits: [] })
    bridge.FileSearch(atFrag, 30, excludes).then((hits) => {
      if (live) setAtSearchState({ token: atSearchToken, waiting: false, hits: hits ?? [] })
    }).catch(() => {
      if (live) setAtSearchState({ token: atSearchToken, waiting: false, hits: [] })
    })
    return () => { live = false; }
  }, [atSearchToken, atFrag, excludes, bridge])
  const atHits = useMemo(
    () => atSearchState.token === atSearchToken ? atSearchState.hits : [],
    [atSearchState, atSearchToken],
  )
  const atSearchWaiting = atSearchState.token === atSearchToken && atSearchState.waiting
  // 最近使用文件（@ 选择过的文件，本地持久化 —— 与「最近文件」快捷区共用单源）
  const [recent, setRecent] = useState<AtEntry[]>(() => loadRecentFiles())
  // 统一 @ 条目：目录内浏览（路径前缀）或 最近使用 + 工作区搜索 + 当前目录
  const atItems: AtEntry[] = useMemo(() => {
    if (atRaw === null) return []
    const out: AtEntry[] = []
    const seen = new Set<string>()
    const push = (e: AtEntry) => { if (!seen.has(e.path) && out.length < 12) { seen.add(e.path); out.push(e); } }
    if (atDir !== "") {
      for (const e of entries) {
        if (!e.name.toLowerCase().includes(atFrag)) continue
        push({ path: atDir + e.name + (e.isDir ? "/" : ""), name: e.name, isDir: e.isDir })
      }
      return out
    }
    for (const r of recent) if (r.name.toLowerCase().includes(atFrag)) push(r)
    for (const h of atHits) {
      if (!h.name.toLowerCase().includes(atFrag)) continue
      push({ path: h.isDir ? h.path + "/" : h.path, name: h.name, isDir: h.isDir, size: h.size })
    }
    for (const e of entries) {
      if (!e.name.toLowerCase().includes(atFrag)) continue
      push({ path: e.name + (e.isDir ? "/" : ""), name: e.name, isDir: e.isDir })
    }
    return out
  }, [atRaw, atDir, atFrag, entries, atHits, recent])

  // ── 菜单状态 ──
  const menuMode: "slash" | "slasharg" | "at" | null =
    slashMatches.length > 0 && !dismissed ? "slash"
    : argRes && argRes.items.length > 0 && !dismissed ? "slasharg"
    : atItems.length > 0 && !dismissed ? "at"
    : null
  const menuCount = menuMode === "slash" ? slashMatches.length : menuMode === "slasharg" ? argRes!.items.length : menuMode === "at" ? atItems.length : 0
  useEffect(() => { setActive(0); setDismissed(false); }, [slashQuery, atRaw])

  const pickCommand = (c: CommandInfo) => setTextCaretEnd("/" + c.name + " ")
  const pickEntry = (e: AtEntry) => {
    const atPos = text.length - (atRaw?.length ?? 0) - 1
    const prefix = text.slice(0, atPos)
    if (e.isDir) {
      setTextCaretEnd(prefix + "@" + e.path)
      return
    }
    recordRecentFile(e.path, e.name)
    setRecent(loadRecentFiles())
    setTextCaretEnd(prefix + "@" + e.path + " ")
  }
  const pickArg = (it: SlashArgItem) => { if (!argRes) return; setTextCaretEnd(text.slice(0, argRes.from) + it.insert) }
  const pickActive = () => {
    if (menuMode === "slash") pickCommand(slashMatches[active])
    else if (menuMode === "slasharg" && argRes) pickArg(argRes.items[active])
    else if (menuMode === "at") pickEntry(atItems[active])
  }

  return {
    slashMatches, argRes, atItems, menuMode, menuCount,
    menuWaiting: slashArgWaiting || atSearchWaiting,
    active, setActive, dismissed, setDismissed,
    pickCommand, pickArg, pickEntry, pickActive,
  }
}
