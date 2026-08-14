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

  const workspaceName = useMemo(() => { if (!cwd) return ""; const parts = cwd.split(/[/\\]/).filter(Boolean); return parts.length > 0 ? parts[parts.length - 1] : cwd; }, [cwd])
  const loadWorkspaces = () => { app.ListWorkspaces().then(setWorkspaces).catch(() => setWorkspaces([])) }
  useEffect(() => { if (workspaceMenuOpen) loadWorkspaces(); }, [workspaceMenuOpen, cwd])
  useEffect(() => {
    if (!workspaceMenuOpen) return
    const close = (e: MouseEvent) => { const tgt = e.target as Node; if (workspaceAnchorRef.current?.contains(tgt) || workspaceMenuRef.current?.contains(tgt)) return; setWorkspaceMenuOpen(false) }
    document.addEventListener("mousedown", close); return () => document.removeEventListener("mousedown", close)
  }, [workspaceMenuOpen])
  const filteredWorkspaces = useMemo(() => { const q = workspaceQuery.trim().toLowerCase(); if (!q) return workspaces; return workspaces.filter((w) => `${w.name} ${w.path}`.toLowerCase().includes(q)); }, [workspaceQuery, workspaces])
  const chooseWorkspace = async (path?: string) => { const next = await onPickFolder(path); if (next) { setWorkspaceMenuOpen(false); setWorkspaceQuery("") } }

  return {
    workspaceName, workspaceMenuOpen, setWorkspaceMenuOpen,
    workspaceQuery, setWorkspaceQuery, filteredWorkspaces,
    workspaceAnchorRef, workspaceMenuRef, chooseWorkspace,
  }
}
