import { useMemo, useState } from "react";
import { Pencil, Search, Trash2, Check, X, MessageSquare, Clock } from "../icons";
import { t, useT } from "../lib/i18n";
import type { SessionMeta } from "../lib/types";
import { DrawerHeader, DrawerTitle } from "./DrawerHeader";
import { ResizableDrawer } from "./ResizableDrawer";

// v3「星枢」面板语言：会话列表 hover 柔光、当前会话 = 主色容器 + 左缘光条 +
// 状态徽标（语义色 + 文字）令牌化；搜索框聚焦光晕。
export function HistoryPanel({
  sessions,
  onResume,
  onDelete,
  onRename,
  onClose,
}: {
  sessions: SessionMeta[];
  onResume: (path: string) => void;
  onDelete: (path: string) => void;
  onRename: (path: string, title: string) => void;
  onClose: () => void;
}) {
  const tr = useT();
  const [editing, setEditing] = useState<string | null>(null);
  const [draft, setDraft] = useState("");
  const [confirming, setConfirming] = useState<string | null>(null);
  const [query, setQuery] = useState("");

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return sessions;
    return sessions.filter((s) => {
      const text = [(s.title || s.preview || ""), s.path].join(" ").toLowerCase();
      return text.includes(q);
    });
  }, [sessions, query]);

  const startRename = (s: SessionMeta) => {
    setConfirming(null);
    setEditing(s.path);
    setDraft(s.title || s.preview || "");
  };
  const commitRename = (path: string) => {
    const title = draft.trim();
    if (title) onRename(path, title);
    setEditing(null);
  };

  // 按日期分组
  const groups: { label: string; items: SessionMeta[] }[] = [];
  for (const s of filtered) {
    const label = dayLabel(s.modTime);
    const last = groups[groups.length - 1];
    if (last && last.label === label) last.items.push(s);
    else groups.push({ label, items: [s] });
  }

  const hasSessions = sessions.length > 0;

  return (
    <ResizableDrawer onClose={onClose}>
      <DrawerHeader onClose={onClose}>
        <DrawerTitle text={tr("history.title")} />
        {hasSessions && (
          <span
            className="rounded-full px-2 py-0.5 font-mono text-[11px]"
            style={{
              background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
              color: "var(--gaea-glow)",
              border: "1px solid color-mix(in srgb, var(--gaea-glow) 26%, transparent)",
            }}
          >
            {sessions.length}
          </span>
        )}
      </DrawerHeader>

      {/* ── 搜索栏（聚焦光晕 = --gaea-glow 柔光） ── */}
      <div className="shrink-0 px-4 py-3 border-b" style={{ borderBottom: "var(--v3-split)", background: "color-mix(in srgb, var(--md-sys-color-surface-container) 55%, transparent)" }}>
        <label className="flex items-center gap-1.5 px-2.5 h-8 rounded-md border transition-[border-color,box-shadow] duration-200 focus-within:border-[color:color-mix(in_srgb,var(--gaea-glow)_45%,var(--md-sys-color-outline-variant))] focus-within:shadow-[0_0_0_2px_color-mix(in_srgb,var(--gaea-glow)_14%,transparent)]"
          style={{ borderColor: "var(--md-sys-color-outline-variant)", background: "var(--md-sys-color-surface-container)", color: "var(--md-sys-color-text-secondary)" }}
        >
          <Search size={14} aria-hidden />
          <input
            className="flex-1 border-0 outline-none bg-transparent text-[13px] placeholder:text-(color:--md-sys-color-text-secondary)"
            style={{ color: "var(--md-sys-color-text)" }}
            type="search"
            placeholder={tr("history.searchPlaceholder")}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            aria-label={tr("history.searchPlaceholder")}
          />
          {query && (
            <button
              className="shrink-0 w-5 h-5 flex items-center justify-center border-0 rounded bg-transparent cursor-pointer transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
              style={{ color: "var(--md-sys-color-text-secondary)" }}
              onClick={() => setQuery("")}
              title="清除搜索"
              aria-label="清除搜索"
            >
              <X size={13} />
            </button>
          )}
        </label>
      </div>

      {/* ── 列表 ── */}
      <div className="flex-1 min-h-0 overflow-y-auto">
        {!hasSessions ? (
          <div className="flex flex-col items-center justify-center gap-3 py-12" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            <MessageSquare size={32} aria-hidden className="opacity-20" />
            <div className="text-[13px]">{tr("history.empty")}</div>
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-3 py-12" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            <Search size={32} aria-hidden className="opacity-20" />
            <div className="text-[13px]">{tr("history.noMatches")}</div>
            <button
              className="px-3 py-1 rounded-md text-[11px] bg-transparent cursor-pointer transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
              style={{ border: "1px solid var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text-secondary)" }}
              onClick={() => setQuery("")}
            >
              清除搜索
            </button>
          </div>
        ) : (
          <div className="px-4 py-3.5 flex flex-col">
            {groups.map((g) => (
              <section className="mb-4" key={g.label}>
                <div className="flex items-center gap-2 px-2 pb-1.5 text-[10px] font-semibold uppercase tracking-wider" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                  <span className="w-1 h-1 rounded-full" style={{ background: "var(--md-sys-color-text-secondary)" }} />
                  {g.label}
                  <span className="font-mono font-normal normal-case tracking-normal" style={{ color: "var(--md-sys-color-text-secondary)" }}>{g.items.length}</span>
                </div>
                {g.items.map((s) => (
                  <div
                    className={`group flex items-start gap-1 px-2 py-2.5 rounded-[var(--radius-md)] border-l-[3px] transition-all duration-200 ${
                      s.current
                        ? "bg-(color:--md-sys-color-primary-container) border-l-(color:--gaea-glow) shadow-[var(--v3-glow-faint)]"
                        : "border-l-transparent hover:bg-(color:--md-sys-color-surface-container-high) hover:shadow-[var(--v3-glow-faint)]"
                    }`}
                    key={s.path}
                  >
                    {editing === s.path ? (
                      <input
                        className="flex-1 rounded-md px-2 py-1 text-[13px] outline-none transition-[border-color,box-shadow] duration-200 focus:shadow-[0_0_0_2px_color-mix(in_srgb,var(--gaea-glow)_14%,transparent)]"
                        style={{
                          background: "var(--md-sys-color-surface-container)",
                          border: "1px solid color-mix(in srgb, var(--gaea-glow) 45%, var(--md-sys-color-outline-variant))",
                          color: "var(--md-sys-color-text)",
                        }}
                        autoFocus
                        value={draft}
                        onChange={(e) => setDraft(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") commitRename(s.path);
                          if (e.key === "Escape") setEditing(null);
                        }}
                        onBlur={() => { if (editing === s.path) commitRename(s.path); }}
                        placeholder={tr("history.namePlaceholder")}
                      />
                    ) : (
                      <button
                        className="flex-1 min-w-0 flex flex-col gap-1 bg-transparent border-0 text-left cursor-pointer"
                        onClick={() => onResume(s.path)}
                        title={s.path}
                      >
                        <div
                          className="text-[13px] leading-snug font-medium truncate"
                          style={{ color: s.current ? "var(--md-sys-color-on-primary-container)" : "var(--md-sys-color-text-secondary)" }}
                        >
                          {s.title || s.preview || tr("history.emptySession")}
                        </div>
                        <div className="flex items-center gap-1.5 text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                          {s.current && (
                            <span
                              className="text-[10px] px-1.5 py-px rounded font-medium"
                              style={{
                                background: "color-mix(in srgb, var(--gaea-glow) 16%, transparent)",
                                color: "var(--gaea-glow)",
                              }}
                            >
                              {tr("history.current")}
                            </span>
                          )}
                          <span className="flex items-center gap-1">
                            <MessageSquare size={11} aria-hidden className="opacity-50" />
                            {tr(s.turns === 1 ? "history.turnOne" : "history.turnOther", { n: s.turns })}
                          </span>
                          <span className="opacity-40">·</span>
                          <span className="flex items-center gap-1">
                            <Clock size={11} aria-hidden className="opacity-50" />
                            {timeLabel(s.modTime)}
                          </span>
                        </div>
                      </button>
                    )}

                    {editing !== s.path && (
                      <div className="hidden group-hover:flex items-center gap-0.5 shrink-0">
                        {confirming === s.path ? (
                          <>
                            <button
                              className="w-7 h-7 flex items-center justify-center border-0 rounded-md bg-transparent cursor-pointer transition-colors hover:bg-[color:color-mix(in_srgb,var(--md-sys-color-destructive)_12%,transparent)]"
                              style={{ color: "var(--md-sys-color-destructive)" }}
                              title={tr("history.confirmDelete")}
                              aria-label={tr("history.confirmDelete")}
                              onClick={() => { onDelete(s.path); setConfirming(null); }}
                            ><Check size={14} /></button>
                            <button
                              className="w-7 h-7 flex items-center justify-center border-0 rounded-md bg-transparent cursor-pointer transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
                              style={{ color: "var(--md-sys-color-text-secondary)" }}
                              title={tr("common.cancel")}
                              aria-label={tr("common.cancel")}
                              onClick={() => setConfirming(null)}
                            ><X size={14} /></button>
                          </>
                        ) : (
                          <>
                            <button
                              className="w-7 h-7 flex items-center justify-center border-0 rounded-md bg-transparent cursor-pointer transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
                              style={{ color: "var(--md-sys-color-text-secondary)" }}
                              title={tr("history.rename")}
                              aria-label={tr("history.rename")}
                              onClick={() => startRename(s)}
                            ><Pencil size={13} /></button>
                            {!s.current && (
                              <button
                                className="w-7 h-7 flex items-center justify-center border-0 rounded-md bg-transparent cursor-pointer transition-colors hover:bg-[color:color-mix(in_srgb,var(--md-sys-color-destructive)_12%,transparent)]"
                                style={{ color: "var(--md-sys-color-text-secondary)" }}
                                title={tr("common.delete")}
                                aria-label={tr("common.delete")}
                                onClick={() => setConfirming(s.path)}
                              ><Trash2 size={13} /></button>
                            )}
                          </>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </section>
            ))}
          </div>
        )}
      </div>
    </ResizableDrawer>
  );
}

/** 日期分组标签 */
function dayLabel(ms: number): string {
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const days = Math.round((startOfDay(new Date()) - startOfDay(new Date(ms))) / 86_400_000);
  if (days <= 0) return t("history.today");
  if (days === 1) return t("history.yesterday");
  return new Date(ms).toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function timeLabel(ms: number): string {
  return new Date(ms).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}
