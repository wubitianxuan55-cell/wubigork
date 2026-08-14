// CapabilitiesPanel 拆分产物：MCP 服务器列表区（行为零变化，T6-10.1）
import { useState } from "react";
import { X } from "../../icons";
import { useT } from "../../lib/i18n";
import { summarizeServerError } from "../../lib/capabilities";
import type { MCPServerInput, ServerView } from "../../lib/types";

export function ServerGroup({
  servers,
  expanded,
  busy,
  confirming,
  onConfirm,
  onCancelConfirm,
  onRemove,
  onRetry,
  onToggle,
  onToggleDetails,
}: {
  servers: ServerView[];
  expanded: Set<string>;
  busy: boolean;
  confirming: string | null;
  onConfirm: (name: string) => void;
  onCancelConfirm: () => void;
  onRemove: (name: string) => void;
  onRetry: (name: string) => void;
  onToggle: (name: string, on: boolean) => void;
  onToggleDetails: (name: string) => void;
}) {
  if (servers.length === 0) return null
  return (
    <div className="flex flex-col mt-3">
      {servers.map((s) => (
        <ServerRow
          key={s.name}
          s={s}
          expanded={expanded.has(s.name)}
          busy={busy}
          confirming={confirming === s.name}
          onConfirm={() => onConfirm(s.name)}
          onCancelConfirm={onCancelConfirm}
          onRemove={() => onRemove(s.name)}
          onRetry={() => onRetry(s.name)}
          onToggle={(on) => onToggle(s.name, on)}
          onToggleDetails={() => onToggleDetails(s.name)}
        />
      ))}
    </div>
  )
}

export function FailedServersNotice({
  servers,
  expanded,
  busy,
  confirming,
  onToggle,
  onRetry,
  onConfirm,
  onCancelConfirm,
  onRemove,
}: {
  servers: ServerView[];
  expanded: Set<string>;
  busy: boolean;
  confirming: string | null;
  onToggle: (name: string) => void;
  onRetry: (name: string) => void;
  onConfirm: (name: string) => void;
  onCancelConfirm: () => void;
  onRemove: (name: string) => void;
}) {
  const t = useT()
  return (
    <div className="mb-3 p-3 border border-err/20 rounded-lg" role="status" style={{background: "var(--ds-danger-soft)"}}>
      <div className="flex items-center justify-between mb-2">
        <div>
          <div className="text-err text-sm font-semibold">{t("caps.failureTitle", { failed: servers.length })}</div>
          <div className="text-fg-faint text-[11px]">{t("caps.failureHint")}</div>
        </div>
      </div>
      <div className="flex flex-col gap-2">
        {servers.map((s) => {
          const open = expanded.has(s.name)
          const error = s.error || t("caps.failed")
          return (
            <div className="border border-border-soft rounded-lg overflow-hidden" key={s.name}>
              <div className="flex items-center gap-2 px-3 py-2">
                <span className="w-2 h-2 rounded-full bg-err shrink-0" />
                <div className="flex-1 min-w-0">
                  <div className="text-fg text-[13px] font-medium">{s.name}</div>
                  <div className="text-fg-faint text-[11px] truncate">{summarizeServerError(error)}</div>
                </div>
              </div>
              <div className="flex items-center gap-1 px-3 pb-2">
                {confirming === s.name ? (
                  <>
                    <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={() => onRemove(s.name)}>{t("caps.confirmRemove")}</button>
                    <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={onCancelConfirm}>{t("common.cancel")}</button>
                  </>
                ) : (
                  <>
                    <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={() => onRetry(s.name)}>{t("caps.retry")}</button>
                    <button className="px-2.5 py-1 text-xs border border-border-soft rounded bg-transparent text-fg-dim cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors" onClick={() => void navigator.clipboard?.writeText(error)}>{t("common.copy")}</button>
                    <button className="px-2.5 py-1 text-xs" onClick={() => onToggle(s.name)} aria-expanded={open}>{open ? t("common.collapse") : t("caps.showLog")}</button>
                    <button className="px-2.5 py-1 text-xs border border-border-soft rounded bg-transparent text-fg-dim cursor-pointer hover:text-err hover:bg-bg-soft transition-colors" disabled={busy} onClick={() => onConfirm(s.name)} title={t("caps.remove")}><X size={13} /></button>
                  </>
                )}
              </div>
              {open && <pre className="m-0 p-3 bg-bg text-fg-dim text-xs leading-relaxed whitespace-pre-wrap border-t border-border-soft max-h-[200px] overflow-y-auto">{error}</pre>}
            </div>
          )
        })}
      </div>
    </div>
  )
}

function ServerRow({
  s,
  expanded,
  busy,
  confirming,
  onConfirm,
  onCancelConfirm,
  onRemove,
  onRetry,
  onToggle,
  onToggleDetails,
}: {
  s: ServerView;
  expanded: boolean;
  busy: boolean;
  confirming: boolean;
  onConfirm: () => void;
  onCancelConfirm: () => void;
  onRemove: () => void;
  onRetry: () => void;
  onToggle: (on: boolean) => void;
  onToggleDetails: () => void;
}) {
  const t = useT()
  const actionLabel = serverActionLabel(s, t)
  const tools = s.toolList ?? []
  const hasTools = tools.length > 0
  const sub =
    s.status === "failed"
      ? s.error || t("caps.failed")
      : s.status === "disabled"
        ? t("caps.disabled")
        : t("caps.counts", { tools: s.tools, prompts: s.prompts, resources: s.resources })
  return (
    <div className={`border border-border-soft rounded-lg ${s.status === "disabled" ? "opacity-60" : ""}`}>
      <div className="flex items-center gap-2 px-3 py-2" title={s.error || undefined}>
        <button
          className="w-5 h-5 border-0 bg-transparent text-fg-faint cursor-pointer flex items-center justify-center text-sm disabled:opacity-30 disabled:cursor-default"
          disabled={!hasTools}
          aria-expanded={hasTools ? expanded : undefined}
          onClick={onToggleDetails}
          title={hasTools ? (expanded ? t("caps.collapseTools") : t("caps.expandTools")) : t("caps.noToolDetails")}
        >
          {hasTools ? (expanded ? "⌄" : "›") : ""}
        </button>
        <span className={`w-2 h-2 rounded-full shrink-0 ${s.status === "connected" ? "bg-ok" : s.status === "failed" ? "bg-err" : "bg-fg-faint"}`} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-fg text-[13px] font-medium">{s.name}</span>
            <span className="text-fg-faint text-[11px] font-mono">{s.transport}</span>
          </div>
          <div className={`text-[11px] truncate ${s.status === "disabled" ? "text-fg-faint opacity-60" : "text-fg-faint"}`}>{sub}</div>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {confirming ? (
            <>
              <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={onRemove}>{t("caps.confirmRemove")}</button>
              <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={onCancelConfirm}>{t("common.cancel")}</button>
            </>
          ) : (
            <>
              {s.status === "failed" ? (
                <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={onRetry}>{actionLabel}</button>
              ) : (
                <label className="inline-flex cursor-pointer no-drag" title={s.status === "connected" ? t("caps.disable") : t("caps.enable")}>
                  <input type="checkbox" className="peer absolute opacity-0 w-0 h-0" checked={s.status === "connected"} disabled={busy} onChange={(e) => onToggle(e.target.checked)} />
                  <span className="relative w-[30px] h-[17px] rounded-full bg-border transition-colors duration-[var(--dur-base)] peer-checked:bg-ok peer-disabled:opacity-50 peer-checked:[&>span]:translate-x-[13px]">
                    <span className="absolute top-0.5 left-0.5 w-[13px] h-[13px] rounded-full bg-bg-elev transition-transform duration-[var(--dur-base)]" />
                  </span>
                </label>
              )}
              <button className="px-2.5 py-1 text-xs border border-border-soft rounded bg-transparent text-fg-dim cursor-pointer hover:text-err hover:bg-bg-soft transition-colors" disabled={busy} onClick={onConfirm} title={t("caps.remove")}><X size={13} /></button>
            </>
          )}
        </div>
      </div>
      {hasTools && expanded && (
        <div className="border-t border-border-soft px-3 py-2">
          <div className="text-fg-faint text-[11px] font-medium mb-1">{t("caps.tools")}</div>
          {tools.map((tool) => (
            <div className="flex items-center gap-2 px-2 py-1" key={tool.name}>
              <span className="font-mono text-fg text-[13px]">{tool.name}</span>
              {tool.description && <span className="text-fg-faint text-[11px] truncate">{tool.description}</span>}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function serverActionLabel(s: ServerView, t: ReturnType<typeof useT>): string {
  const err = (s.error || "").toLowerCase()
  if (err.includes("401") || err.includes("unauthorized")) return t("caps.reauthorize")
  if (
    err.includes("command not found") ||
    err.includes("executable file not found") ||
    err.includes("no such file") ||
    err.includes("enoent")
  ) {
    return t("caps.checkCommand")
  }
  return t("caps.retry")
}

export function AddServerForm({
  busy,
  onCancel,
  onAdd,
}: {
  busy: boolean;
  onCancel: () => void;
  onAdd: (input: MCPServerInput) => void;
}) {
  const t = useT()
  const [name, setName] = useState("")
  const [transport, setTransport] = useState("stdio")
  const [command, setCommand] = useState("")
  const [url, setUrl] = useState("")
  const [env, setEnv] = useState("")

  const isStdio = transport === "stdio"
  const ready = name.trim() !== "" && (isStdio ? command.trim() !== "" : url.trim() !== "")

  const submit = () => {
    const parts = command.trim().split(/\s+/).filter(Boolean)
    const envMap: Record<string, string> = {}
    for (const line of env.split("\n")) {
      const eq = line.indexOf("=")
      if (eq > 0) envMap[line.slice(0, eq).trim()] = line.slice(eq + 1).trim()
    }
    onAdd({
      name: name.trim(),
      transport,
      command: isStdio ? (parts[0] ?? "") : "",
      args: isStdio ? parts.slice(1) : [],
      url: isStdio ? "" : url.trim(),
      env: envMap,
    })
  }

  return (
    <div className="flex flex-col gap-2 p-3 border border-border-soft rounded-lg mb-2">
      <input className="flex-1 bg-bg-soft border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent" placeholder={t("caps.namePlaceholder")} value={name} onChange={(e) => setName(e.target.value)} />
      <label className="text-fg-dim text-[13px] shrink-0">{t("caps.transport")}</label>
      <select className="bg-bg-soft border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none focus:border-accent" value={transport} onChange={(e) => setTransport(e.target.value)}>
        <option value="stdio">stdio</option>
        <option value="http">http</option>
        <option value="sse">sse</option>
      </select>
      {isStdio ? (
        <input className="flex-1 bg-bg-soft border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent" placeholder={t("caps.commandPlaceholder")} value={command} onChange={(e) => setCommand(e.target.value)} />
      ) : (
        <input className="flex-1 bg-bg-soft border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent" placeholder={t("caps.urlPlaceholder")} value={url} onChange={(e) => setUrl(e.target.value)} />
      )}
      <label className="text-fg-dim text-[13px] shrink-0">{t("caps.envLabel")}</label>
      <textarea className="bg-bg-soft border border-border-soft rounded-md text-fg text-[13px] p-2 outline-none resize-y min-h-[60px] focus:border-accent" value={env} onChange={(e) => setEnv(e.target.value)} placeholder={t("caps.envPlaceholder")} spellCheck={false} />
      <div className="flex justify-end gap-2 mt-2">
        <button className="px-2.5 py-1 text-xs border border-border-soft rounded bg-transparent text-fg-dim cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors" onClick={onCancel} disabled={busy}>
          {t("common.cancel")}
        </button>
        <button className="btn--primary" onClick={submit} disabled={busy || !ready}>
          {t("caps.add")}
        </button>
      </div>
    </div>
  )
}
