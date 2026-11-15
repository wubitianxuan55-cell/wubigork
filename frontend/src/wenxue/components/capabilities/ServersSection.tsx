// CapabilitiesPanel 拆分产物：MCP 服务器列表区（行为零变化，T6-10.1）
// v3「星枢」面板语言：服务器卡实底收敛（状态点 + 名称/传输 + 计数 + 启停），
// 失败区 = 破坏语义色容器；开关/按钮全部令牌化。
import { useState, type ReactNode } from "react";
import { ChevronDown, ChevronRight, X } from "../../icons";
import { useT } from "../../lib/i18n";
import { summarizeServerError } from "../../lib/capabilities";
import type { MCPServerInput, ServerView } from "../../lib/types";

const textBtn =
  "px-2.5 py-1 text-xs rounded-md cursor-pointer transition-colors disabled:opacity-40 disabled:cursor-default";

function OutlinedBtn({ onClick, disabled, children, title }: { onClick: () => void; disabled?: boolean; children: ReactNode; title?: string }) {
  return (
    <button
      className={textBtn}
      onClick={onClick}
      disabled={disabled}
      title={title}
      style={{
        border: "1px solid var(--md-sys-color-outline-variant)",
        background: "transparent",
        color: "var(--md-sys-color-text-secondary)",
      }}
    >
      {children}
    </button>
  )
}

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
    <div className="flex flex-col mt-3 gap-2">
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
    <div
      className="mb-3 p-3 rounded-[var(--radius-md)]"
      role="status"
      style={{
        background: "color-mix(in srgb, var(--md-sys-color-destructive) 8%, transparent)",
        border: "1px solid color-mix(in srgb, var(--md-sys-color-destructive) 26%, transparent)",
      }}
    >
      <div className="flex items-center justify-between mb-2">
        <div>
          <div className="text-sm font-semibold" style={{ color: "var(--md-sys-color-destructive)" }}>{t("caps.failureTitle", { failed: servers.length })}</div>
          <div className="text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{t("caps.failureHint")}</div>
        </div>
      </div>
      <div className="flex flex-col gap-2">
        {servers.map((s) => {
          const open = expanded.has(s.name)
          const error = s.error || t("caps.failed")
          return (
            <div
              className="rounded-[var(--radius-sm)] overflow-hidden"
              key={s.name}
              style={{
                background: "var(--md-sys-color-surface-container)",
                border: "1px solid var(--md-sys-color-outline-variant)",
              }}
            >
              <div className="flex items-center gap-2 px-3 py-2">
                <span
                  className="w-2 h-2 rounded-full shrink-0"
                  style={{ background: "var(--md-sys-color-destructive)", boxShadow: "0 0 6px var(--md-sys-color-destructive)" }}
                />
                <div className="flex-1 min-w-0">
                  <div className="text-[13px] font-medium" style={{ color: "var(--md-sys-color-text)" }}>{s.name}</div>
                  <div className="text-[11px] truncate" style={{ color: "var(--md-sys-color-text-secondary)" }}>{summarizeServerError(error)}</div>
                </div>
              </div>
              <div className="flex items-center gap-1 px-3 pb-2">
                {confirming === s.name ? (
                  <>
                    <button className={textBtn} disabled={busy} onClick={() => onRemove(s.name)} style={{ border: "1px solid color-mix(in srgb, var(--md-sys-color-destructive) 40%, transparent)", color: "var(--md-sys-color-destructive)", background: "transparent" }}>{t("caps.confirmRemove")}</button>
                    <button className={textBtn} disabled={busy} onClick={onCancelConfirm} style={{ border: "1px solid var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text-secondary)", background: "transparent" }}>{t("common.cancel")}</button>
                  </>
                ) : (
                  <>
                    <button className={textBtn} disabled={busy} onClick={() => onRetry(s.name)} style={{ border: "1px solid color-mix(in srgb, var(--gaea-glow) 40%, transparent)", color: "var(--gaea-glow)", background: "color-mix(in srgb, var(--gaea-glow) 7%, transparent)" }}>{t("caps.retry")}</button>
                    <OutlinedBtn onClick={() => void navigator.clipboard?.writeText(error)}>{t("common.copy")}</OutlinedBtn>
                    <button className={textBtn} onClick={() => onToggle(s.name)} aria-expanded={open} style={{ border: "1px solid var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text-secondary)", background: "transparent" }}>{open ? t("common.collapse") : t("caps.showLog")}</button>
                    <OutlinedBtn onClick={() => onConfirm(s.name)} disabled={busy} title={t("caps.remove")}>
                      <X size={13} aria-hidden />
                    </OutlinedBtn>
                  </>
                )}
              </div>
              {open && (
                <pre
                  className="m-0 p-3 text-xs leading-relaxed whitespace-pre-wrap max-h-[200px] overflow-y-auto"
                  style={{
                    background: "var(--md-sys-color-surface)",
                    color: "var(--md-sys-color-text-secondary)",
                    borderTop: "1px solid var(--md-sys-color-outline-variant)",
                  }}
                >
                  {error}
                </pre>
              )}
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
  const dotColor =
    s.status === "connected" ? "var(--md-sys-color-success)"
    : s.status === "failed" ? "var(--md-sys-color-destructive)"
    : "var(--md-sys-color-text-secondary)"
  return (
    <div
      className={`rounded-[var(--radius-md)] transition-all duration-200 ${s.status === "disabled" ? "opacity-60" : ""}`}
      style={{
        background: "var(--md-sys-color-surface-container)",
        border: "1px solid var(--md-sys-color-outline-variant)",
        boxShadow: "inset 0 1px 0 color-mix(in srgb, var(--md-sys-color-text) 5%, transparent)",
      }}
    >
      <div className="flex items-center gap-2 px-3 py-2" title={s.error || undefined}>
        <button
          className="w-5 h-5 border-0 bg-transparent cursor-pointer flex items-center justify-center disabled:opacity-30 disabled:cursor-default"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
          disabled={!hasTools}
          aria-expanded={hasTools ? expanded : undefined}
          onClick={onToggleDetails}
          title={hasTools ? (expanded ? t("caps.collapseTools") : t("caps.expandTools")) : t("caps.noToolDetails")}
          aria-label={hasTools ? (expanded ? t("caps.collapseTools") : t("caps.expandTools")) : t("caps.noToolDetails")}
        >
          {hasTools ? (expanded ? <ChevronDown size={13} aria-hidden /> : <ChevronRight size={13} aria-hidden />) : null}
        </button>
        <span
          className="w-2 h-2 rounded-full shrink-0"
          style={{ background: dotColor, boxShadow: `0 0 6px ${dotColor}` }}
        />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-[13px] font-medium" style={{ color: "var(--md-sys-color-text)" }}>{s.name}</span>
            <span className="text-[11px] font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>{s.transport}</span>
          </div>
          <div
            className="text-[11px] truncate"
            style={{ color: "var(--md-sys-color-text-secondary)" }}
          >
            {sub}
          </div>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {confirming ? (
            <>
              <button className={textBtn} disabled={busy} onClick={onRemove} style={{ border: "1px solid color-mix(in srgb, var(--md-sys-color-destructive) 40%, transparent)", color: "var(--md-sys-color-destructive)", background: "transparent" }}>{t("caps.confirmRemove")}</button>
              <button className={textBtn} disabled={busy} onClick={onCancelConfirm} style={{ border: "1px solid var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text-secondary)", background: "transparent" }}>{t("common.cancel")}</button>
            </>
          ) : (
            <>
              {s.status === "failed" ? (
                <button className={textBtn} disabled={busy} onClick={onRetry} style={{ border: "1px solid color-mix(in srgb, var(--gaea-glow) 40%, transparent)", color: "var(--gaea-glow)", background: "color-mix(in srgb, var(--gaea-glow) 7%, transparent)" }}>{actionLabel}</button>
              ) : (
                <label className="inline-flex cursor-pointer no-drag" title={s.status === "connected" ? t("caps.disable") : t("caps.enable")}>
                  <input type="checkbox" className="peer absolute opacity-0 w-0 h-0" checked={s.status === "connected"} disabled={busy} onChange={(e) => onToggle(e.target.checked)} />
                  <span
                    className="relative w-[30px] h-[17px] rounded-full transition-colors duration-200 peer-checked:bg-(color:--md-sys-color-success) peer-disabled:opacity-50 peer-checked:[&>span]:translate-x-[13px]"
                    style={{ background: "var(--md-sys-color-outline-variant)" }}
                  >
                    <span
                      className="absolute top-0.5 left-0.5 w-[13px] h-[13px] rounded-full transition-transform duration-200"
                      style={{ background: "var(--md-sys-color-surface-container-highest)" }}
                    />
                  </span>
                </label>
              )}
              <OutlinedBtn onClick={onConfirm} disabled={busy} title={t("caps.remove")}>
                <X size={13} aria-hidden />
              </OutlinedBtn>
            </>
          )}
        </div>
      </div>
      {hasTools && expanded && (
        <div
          className="px-3 py-2"
          style={{ borderTop: "1px solid var(--md-sys-color-outline-variant)" }}
        >
          <div className="text-[11px] font-medium mb-1" style={{ color: "var(--md-sys-color-text-secondary)" }}>{t("caps.tools")}</div>
          {tools.map((tool) => (
            <div className="flex items-center gap-2 px-2 py-1" key={tool.name}>
              <span className="font-mono text-[13px]" style={{ color: "var(--md-sys-color-text)" }}>{tool.name}</span>
              {tool.description && <span className="text-[11px] truncate" style={{ color: "var(--md-sys-color-text-secondary)" }}>{tool.description}</span>}
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

const fieldCls =
  "rounded-lg border outline-none text-[13px] transition-[border-color,box-shadow] duration-200 focus:border-[color:color-mix(in_srgb,var(--gaea-glow)_45%,var(--md-sys-color-outline-variant))] focus:shadow-[0_0_0_2px_color-mix(in_srgb,var(--gaea-glow)_14%,transparent)] placeholder:text-(color:--md-sys-color-text-secondary) border-(color:--md-sys-color-outline-variant) bg-(color:--md-sys-color-surface-container) text-(color:--md-sys-color-text)";

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
    <div
      className="flex flex-col gap-2 p-3 rounded-[var(--radius-md)] mb-2"
      style={{
        background: "var(--md-sys-color-surface-container)",
        border: "1px solid var(--md-sys-color-outline-variant)",
      }}
    >
      <input className={`flex-1 px-2.5 py-1.5 ${fieldCls}`} placeholder={t("caps.namePlaceholder")} value={name} onChange={(e) => setName(e.target.value)} />
      <label className="text-[13px] shrink-0" style={{ color: "var(--md-sys-color-text-secondary)" }}>{t("caps.transport")}</label>
      <select className={`px-2.5 py-1.5 ${fieldCls}`} value={transport} onChange={(e) => setTransport(e.target.value)}>
        <option value="stdio">stdio</option>
        <option value="http">http</option>
        <option value="sse">sse</option>
      </select>
      {isStdio ? (
        <input className={`flex-1 px-2.5 py-1.5 ${fieldCls}`} placeholder={t("caps.commandPlaceholder")} value={command} onChange={(e) => setCommand(e.target.value)} />
      ) : (
        <input className={`flex-1 px-2.5 py-1.5 ${fieldCls}`} placeholder={t("caps.urlPlaceholder")} value={url} onChange={(e) => setUrl(e.target.value)} />
      )}
      <label className="text-[13px] shrink-0" style={{ color: "var(--md-sys-color-text-secondary)" }}>{t("caps.envLabel")}</label>
      <textarea className={`resize-y min-h-[60px] p-2 ${fieldCls}`} style={{ minHeight: 60 }} value={env} onChange={(e) => setEnv(e.target.value)} placeholder={t("caps.envPlaceholder")} spellCheck={false} />
      <div className="flex justify-end gap-2 mt-2">
        <OutlinedBtn onClick={onCancel} disabled={busy}>
          {t("common.cancel")}
        </OutlinedBtn>
        <button className="btn--primary" onClick={submit} disabled={busy || !ready}>
          {t("caps.add")}
        </button>
      </div>
    </div>
  )
}
