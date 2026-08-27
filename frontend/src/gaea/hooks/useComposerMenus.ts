// Composer 拆分产物：/ 命令、/ 参数、@ 文件引用菜单状态机（行为零变化，T6-10.1）
import { useEffect, useMemo, useRef, useState } from 'react'
import { app } from '../lib/bridge'
import { slashQueryOf, atMentionOf } from '../lib/composer'
import type { CommandInfo, DirEntry, FileSearchHit, SlashArgsResult, SlashArgItem, AtEntry } from '../lib/types'
import { loadRecentFiles, recordRecentFile } from '../lib/recentFiles'

export interface UseComposerMenusOptions {
  text: string
  debouncedText: string
  setTextCaretEnd: (next: string) => void
}

export function useComposerMenus({ text, debouncedText, setTextCaretEnd }: UseComposerMenusOptions) {
  const [active, setActive] = useState(0)
  const [dismissed, setDismissed] = useState(false)

  // ── / 命令 ──
  const [commands, setCommands] = useState<CommandInfo[]>([])
  useEffect(() => { app.Commands().then(setCommands).catch(() => {}); }, [])
  const slashQuery = useMemo(() => slashQueryOf(text), [text])
  const slashMatches = useMemo(() => (slashQuery === null ? [] : commands.filter((c) => c.name.toLowerCase().includes(slashQuery)).slice(0, 8)), [slashQuery, commands])

  // ── 命令参数 ──
  const [argRes, setArgRes] = useState<SlashArgsResult | null>(null)
  useEffect(() => {
    if (!text.startsWith("/") || !/\s/.test(text)) { setArgRes(null); return; }
    let live = true
    app.SlashArgs(text).then((r) => {
      if (!live) return
      const useful = (r.items ?? []).filter((it) => text.slice(0, r.from) + it.insert !== text)
      setArgRes(useful.length > 0 ? { items: useful, from: r.from } : null); setActive(0)
    }).catch(() => {})
    return () => { live = false; }
  }, [text])

  // ── @ 文件引用 ──
  const atMention = useMemo(() => atMentionOf(debouncedText), [debouncedText])
  const atRaw = atMention?.raw ?? null
  const atDir = atMention?.dir ?? ""
  const atFrag = atMention?.frag ?? ""
  const atActive = atRaw !== null
  const [entries, setEntries] = useState<DirEntry[]>([])
  const dirCache = useRef<Record<string, DirEntry[]>>({})
  useEffect(() => {
    if (!atActive) return
    const cached = dirCache.current[atDir]
    if (cached) { setEntries(cached); return }
    let live = true
    app.ListDir(atDir).then((es) => { const list = es ?? []; dirCache.current[atDir] = list; if (live) setEntries(list); }).catch(() => {})
    return () => { live = false; }
  }, [atActive, atDir])
  // 工作区跨目录搜索（@ 引用增强：搜一下定位资料）
  const [atHits, setAtHits] = useState<FileSearchHit[]>([])
  useEffect(() => {
    if (atRaw === null) { setAtHits([]); return; }
    let live = true
    app.FileSearch(atFrag, 30).then((h) => { if (live) setAtHits(h ?? []); }).catch(() => {})
    return () => { live = false; }
  }, [atRaw, atFrag])
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
    active, setActive, dismissed, setDismissed,
    pickCommand, pickArg, pickEntry, pickActive,
  }
}
