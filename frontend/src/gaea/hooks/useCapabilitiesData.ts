// CapabilitiesPanel 拆分产物：能力快照数据 hook（行为零变化，T6-10.1）
import { useEffect, useMemo, useState } from 'react'
import { app } from '../lib/bridge'
import { useT } from '../lib/i18n'
import type { CapabilitiesView } from '../lib/types'

export function useCapabilitiesData() {
  const t = useT()
  const [view, setView] = useState<CapabilitiesView | null>(null)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [reloading, setReloading] = useState(false)
  const [adding, setAdding] = useState(false)
  const [addingContext7, setAddingContext7] = useState(false)
  const [confirming, setConfirming] = useState<string | null>(null)

  const reload = async () =>
    setView(await app.Capabilities().catch(() => ({ servers: [], skills: [] })))
  useEffect(() => {
    void reload()
  }, [])

  // reloadEngine 热加载办公引擎：重建 controller 后重新拉取能力快照，
  // 外部编辑的技能/工具/插件目录无需重启桌面端即可被引擎感知。
  const reloadEngine = async () => {
    setReloading(true)
    setNotice(null)
    setErr(null)
    try {
      const res = await app.Reload()
      await reload()
      setNotice(t("caps.reloaded", { tools: res.tools, skills: res.skills }))
    } catch (e) {
      setErr(String((e as Error)?.message ?? e))
    } finally {
      setReloading(false)
    }
  }

  // mutate runs an MCP edit, re-reads the snapshot, and surfaces any failure as an
  // inline banner (a connect error, a missing binary, a bad URL).
  const mutate = async (fn: () => Promise<unknown>) => {
    setBusy(true)
    setErr(null)
    try {
      await fn()
      await reload()
      return true
    } catch (e) {
      setErr(String((e as Error)?.message ?? e))
      return false
    } finally {
      setBusy(false)
    }
  }

  const addContext7 = async () => {
    setAddingContext7(true)
    setErr(null)
    try {
      await app.AddMCPServer({
        name: "context7",
        transport: "streamable-http",
        command: "",
        args: [],
        url: "https://mcp.context7.com/mcp",
        env: {},
      })
      await reload()
    } catch (e) {
      setErr(String((e as Error)?.message ?? e))
    } finally {
      setAddingContext7(false)
    }
  }

  const summary = useMemo(() => {
    if (!view) return ""
    return t("caps.summary", {
      connected: view.servers.filter((s) => s.status === "connected").length,
      failed: view.servers.filter((s) => s.status === "failed").length,
      skills: view.skills.length,
    })
  }, [view, t])

  const serverGroups = useMemo(() => {
    const servers = view?.servers ?? []
    return {
      failed: servers.filter((s) => s.status === "failed"),
      active: servers.filter((s) => s.status !== "failed"),
    }
  }, [view])

  return {
    view, busy, err, notice, reloading, adding, addingContext7, confirming,
    setConfirming, setAdding, reloadEngine, mutate, addContext7, summary, serverGroups,
  }
}
