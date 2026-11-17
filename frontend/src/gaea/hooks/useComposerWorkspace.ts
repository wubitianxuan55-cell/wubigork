// Composer 拆分产物：工作区切换菜单状态机（行为零变化，T6-10.1）
import { useEffect, useMemo, useRef, useState } from 'react'
import { app as defaultApp } from '../lib/bridge'
import type { ComposerBridge } from '../lib/composerBridge'
import type { WorkspaceView } from '../lib/types'

export interface UseComposerWorkspaceOptions {
  cwd?: string
  onPickFolder: (path?: string) => Promise<string>
  bridge?: ComposerBridge
  variant?: 'office' | 'wenxue'
}

export function useComposerWorkspace({
  cwd, onPickFolder, bridge: bridgeOverride, variant = 'office',
}: UseComposerWorkspaceOptions) {
  const bridge = bridgeOverride ?? defaultApp
  const [workspaceMenuOpen, setWorkspaceMenuOpen] = useState(false)
  const [workspaceQuery, setWorkspaceQuery] = useState("")
  const [workspaces, setWorkspaces] = useState<WorkspaceView[]>([])
  const [wenxueCwd, setWenxueCwd] = useState("")
  const workspaceAnchorRef = useRef<HTMLDivElement>(null)
  const workspaceMenuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!workspaceMenuOpen) return
    bridge.ListWorkspaces().then(setWorkspaces).catch(() => setWorkspaces([]))
  }, [workspaceMenuOpen, cwd, bridge])
  useEffect(() => {
    if (variant !== 'wenxue') {
      setWenxueCwd("")
      return
    }
    const refresh = () => {
      bridge.ListWorkspaces().then((workspaceList) => {
        const next = workspaceList ?? []
        setWorkspaces(next)
        setWenxueCwd(next.find((workspace) => workspace.current)?.path ?? "")
      }).catch(() => {})
    }
    refresh()
    window.addEventListener("wenxue:workspace-changed", refresh)
    return () => window.removeEventListener("wenxue:workspace-changed", refresh)
  }, [bridge, variant])
  const workspaceName = useMemo(() => {
    const path = variant === 'wenxue' ? (wenxueCwd || cwd) : cwd
    if (!path) return ""
    const parts = path.split(/[/\\]/).filter(Boolean)
    return parts.length > 0 ? parts[parts.length - 1] : path
  }, [cwd, variant, wenxueCwd])
  useEffect(() => {
    if (!workspaceMenuOpen) return
    const close = (e: MouseEvent) => { const tgt = e.target as Node; if (workspaceAnchorRef.current?.contains(tgt) || workspaceMenuRef.current?.contains(tgt)) return; setWorkspaceMenuOpen(false) }
    document.addEventListener("mousedown", close); return () => document.removeEventListener("mousedown", close)
  }, [workspaceMenuOpen])
  const filteredWorkspaces = useMemo(() => { const q = workspaceQuery.trim().toLowerCase(); if (!q) return workspaces; return workspaces.filter((w) => `${w.name} ${w.path}`.toLowerCase().includes(q)); }, [workspaceQuery, workspaces])
  const chooseWorkspace = async (path?: string) => { const next = await onPickFolder(path); if (next) { setWorkspaceMenuOpen(false); setWorkspaceQuery("") } }
  const requestCreateNovel = variant === 'wenxue' ? () => {
    setWorkspaceMenuOpen(false)
    setWorkspaceQuery("")
    window.dispatchEvent(new Event("wenxue:new-novel-requested"))
  } : undefined

  return {
    workspaceName, workspaceMenuOpen, setWorkspaceMenuOpen,
    workspaceQuery, setWorkspaceQuery, filteredWorkspaces,
    workspaceAnchorRef, workspaceMenuRef, chooseWorkspace, requestCreateNovel,
  }
}
