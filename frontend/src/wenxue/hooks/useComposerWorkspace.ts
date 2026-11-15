// Composer 拆分产物：工作区切换菜单状态机（行为零变化，T6-10.1）
import { useEffect, useMemo, useRef, useState } from 'react'
import { app } from '../lib/bridge'
import type { WorkspaceView } from '../lib/types'

export interface UseComposerWorkspaceOptions {
  cwd?: string
  onPickFolder: (path?: string) => Promise<string>
}

export function useComposerWorkspace({ cwd, onPickFolder }: UseComposerWorkspaceOptions) {
  const [workspaceMenuOpen, setWorkspaceMenuOpen] = useState(false)
  const [workspaceQuery, setWorkspaceQuery] = useState("")
  const [workspaces, setWorkspaces] = useState<WorkspaceView[]>([])
  const workspaceAnchorRef = useRef<HTMLDivElement>(null)
  const workspaceMenuRef = useRef<HTMLDivElement>(null)

  const loadWorkspaces = () => { app.ListWorkspaces().then(setWorkspaces).catch(() => setWorkspaces([])) }
  useEffect(() => { if (workspaceMenuOpen) loadWorkspaces(); }, [workspaceMenuOpen, cwd])
  // 图文工作空间变更（左栏《新建小说》或小说切换广播）：重拉列表保证菜单现值。
  // 工作区名取 WenxueListWorkspaces 现值（tuwen 口径），不用壳层 cwd（=办公 Meta，另一工作空间）。
  const [wenxueCwd, setWenxueCwd] = useState("")
  useEffect(() => {
    const refresh = () => {
      app.ListWorkspaces().then((ws) => {
        setWorkspaces(ws ?? [])
        const cur = (ws ?? []).find((w) => w.current)
        setWenxueCwd(cur?.path ?? "")
      }).catch(() => {})
    }
    refresh()
    window.addEventListener("wenxue:workspace-changed", refresh)
    return () => window.removeEventListener("wenxue:workspace-changed", refresh)
  }, [])
  const workspaceName = useMemo(() => {
    const base = wenxueCwd || cwd
    if (!base) return ""
    const parts = base.split(/[/\\]/).filter(Boolean)
    return parts.length > 0 ? parts[parts.length - 1] : base
  }, [wenxueCwd, cwd])
  useEffect(() => {
    if (!workspaceMenuOpen) return
    const close = (e: MouseEvent) => { const tgt = e.target as Node; if (workspaceAnchorRef.current?.contains(tgt) || workspaceMenuRef.current?.contains(tgt)) return; setWorkspaceMenuOpen(false) }
    document.addEventListener("mousedown", close); return () => document.removeEventListener("mousedown", close)
  }, [workspaceMenuOpen])
  const filteredWorkspaces = useMemo(() => { const q = workspaceQuery.trim().toLowerCase(); if (!q) return workspaces; return workspaces.filter((w) => `${w.name} ${w.path}`.toLowerCase().includes(q)); }, [workspaceQuery, workspaces])
  const chooseWorkspace = async (path?: string) => { const next = await onPickFolder(path); if (next) { setWorkspaceMenuOpen(false); setWorkspaceQuery("") } }
  const requestCreateNovel = () => {
    setWorkspaceMenuOpen(false)
    setWorkspaceQuery("")
    window.dispatchEvent(new Event("wenxue:new-novel-requested"))
  }

  return {
    workspaceName, workspaceMenuOpen, setWorkspaceMenuOpen,
    workspaceQuery, setWorkspaceQuery, filteredWorkspaces,
    workspaceAnchorRef, workspaceMenuRef, chooseWorkspace, requestCreateNovel,
  }
}
