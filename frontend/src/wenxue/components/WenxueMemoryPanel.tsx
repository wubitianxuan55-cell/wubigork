// WenxueMemoryPanel — 图文板块「记忆」标签页（2026-11-16 面板优化批）。
// 数据面全锚图文脑（wenxue-brain 独立 DB）：facts+归档清单+生命周期
// （固化/归档/失效恢复）。视觉对齐三 Hub（WenxueHubFrame）。
// 做梦/晨报/建议是办公脑机制，图文引擎不跑——面板如实标注。

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Plus, RefreshCw, Search } from "../icons";
import { app } from "../lib/bridge";
import { subscribe } from "../../events";
import { WenxueHubFrame } from "./WenxueHubFrame";
import type { MemoryView, WenxueMemoryArchivedView } from "../lib/types";

const FACT_TYPE_LABEL: Record<string, string> = {
  user: "用户",
  feedback: "反馈",
  project: "项目",
  reference: "参考",
};

const FACT_TYPES = ["user", "feedback", "project", "reference"] as const;

function fmtTime(v?: string): string {
  if (!v) return "";
  const d = new Date(v);
  return Number.isNaN(d.getTime()) ? v : d.toLocaleString();
}

export function WenxueMemoryPanel() {
  const [view, setView] = useState<MemoryView | null>(null);
  const [archived, setArchived] = useState<WenxueMemoryArchivedView[]>([]);
  const [archivedOpen, setArchivedOpen] = useState(false);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState<"all" | (typeof FACT_TYPES)[number]>("all");
  const [note, setNote] = useState("");
  const [scope, setScope] = useState("project");
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState("");
  const [draft, setDraft] = useState("");
  const reloadRef = useRef(0);

  const load = useCallback(async () => {
    const token = ++reloadRef.current;
    setError("");
    try {
      const [v, arch] = await Promise.all([
        app.Memory(),
        app.WenxueMemoryArchivedList().catch(() => [] as WenxueMemoryArchivedView[]),
      ]);
      if (token !== reloadRef.current) return;
      setView(v);
      setArchived(arch ?? []);
    } catch (err) {
      if (token === reloadRef.current) setError(err instanceof Error ? err.message : String(err));
    }
  }, []);

  useEffect(() => {
    void load();
    return subscribe("wenxue:workspace-changed", () => void load());
  }, [load]);

  const facts = useMemo(() => {
    const all = view?.facts ?? [];
    const q = query.trim().toLowerCase();
    return all
      .filter((f) => (typeFilter === "all" ? true : f.type === typeFilter))
      .filter((f) => (q ? `${f.name} ${f.title ?? ""} ${f.body}`.toLowerCase().includes(q) : true))
      .sort((a, b) => {
        if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
        return (b.updatedAt ?? "").localeCompare(a.updatedAt ?? "");
      });
  }, [view, query, typeFilter]);

  const pinnedCount = (view?.facts ?? []).filter((f) => f.pinned).length;

  const addFact = async () => {
    const text = note.trim();
    if (!text) return;
    setAdding(true);
    try {
      await app.Remember(scope, text);
      setNote("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setAdding(false);
    }
  };

  const saveEdit = async (name: string) => {
    try {
      await app.UpdateFact(name, draft);
      setEditing("");
      setDraft("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const guard = async (action: () => Promise<unknown>) => {
    try {
      await action();
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const scopes = view?.scopes?.length
    ? view.scopes
    : [{ scope: "project", path: "" }, { scope: "user", path: "" }, { scope: "local", path: "" }];

  return (
    <WenxueHubFrame
      tone="memory"
      eyebrow="书脑"
      title="图文记忆"
      description="事实自动注入每个会话；做梦/晨报为办公板块机制，图文不运行。读写只在图文脑（wenxue-brain），与办公记忆物理隔离。"
      metrics={[
        { label: "事实", value: view?.facts?.length ?? 0, hint: view?.available ? undefined : "脑不可用" },
        { label: "固化", value: pinnedCount, hint: "免疫衰减" },
        { label: "归档", value: archived.length, hint: "可恢复" },
      ]}
      actions={[
        {
          key: "refresh",
          label: "刷新",
          icon: RefreshCw,
          onClick: () => void load(),
        },
      ]}
    >
      <div className="flex h-full min-h-0 flex-col" data-testid="wenxue-memory-panel">
        {error && (
          <div className="mx-4 mb-2 rounded-md border border-warning/30 bg-warning/5 px-2 py-1.5 text-[10px] text-warning" role="alert">
            {error}
          </div>
        )}

        {/* 快捷添加（Remember → 图文脑） */}
        <div className="mx-4 mb-2 flex items-center gap-1.5 rounded-md border px-2 py-1.5"
          style={{ borderColor: "var(--md-sys-color-outline-variant)", background: "var(--md-sys-color-surface-container)" }}>
          <select
            className="border-0 bg-transparent text-[11px] outline-none"
            style={{ color: "var(--md-sys-color-text-secondary)" }}
            value={scope}
            onChange={(e) => setScope(e.target.value)}
            aria-label="记忆归属"
          >
            {scopes.map((s) => (
              <option key={s.scope} value={s.scope}>{s.scope}</option>
            ))}
          </select>
          <input
            className="min-w-0 flex-1 border-0 bg-transparent text-[11.5px] outline-none"
            style={{ color: "var(--md-sys-color-text)" }}
            value={note}
            onChange={(e) => setNote(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && note.trim()) void addFact();
            }}
            placeholder="记一条事实（Enter 快捷添加，自动注入后续会话）…"
            aria-label="快捷添加记忆"
          />
          <button
            type="button"
            className="flex cursor-pointer items-center gap-1 rounded border-0 px-2 py-1 text-[10.5px] disabled:opacity-50"
            style={{ background: "var(--md-sys-color-primary-container)", color: "var(--md-sys-color-primary)" }}
            disabled={adding || !note.trim()}
            onClick={() => void addFact()}
          >
            <Plus size={11} aria-hidden /> 记住
          </button>
        </div>

        {/* 搜索 + 类型筛选 */}
        <div className="mx-4 mb-2 flex items-center gap-1.5">
          <div className="flex min-w-0 flex-1 items-center gap-1.5 rounded-md border px-2 py-1.5"
            style={{ borderColor: "var(--md-sys-color-outline-variant)", background: "var(--md-sys-color-surface-container)" }}>
            <Search size={12} aria-hidden />
            <input
              className="min-w-0 flex-1 border-0 bg-transparent text-[11.5px] outline-none"
              style={{ color: "var(--md-sys-color-text)" }}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索事实名/正文…"
              aria-label="搜索记忆"
            />
          </div>
          {(["all", ...FACT_TYPES] as const).map((t) => (
            <button
              key={t}
              type="button"
              className="cursor-pointer rounded-full border-0 px-2 py-1 text-[10px]"
              aria-pressed={typeFilter === t}
              onClick={() => setTypeFilter(t)}
              style={{
                background: typeFilter === t ? "var(--md-sys-color-primary-container)" : "var(--md-sys-color-surface-container)",
                color: typeFilter === t ? "var(--md-sys-color-primary)" : "var(--md-sys-color-text-secondary)",
              }}
            >
              {t === "all" ? "全部" : FACT_TYPE_LABEL[t]}
            </button>
          ))}
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-4">
          {!view ? (
            <div className="py-10 text-center text-[11px] text-fg-faint">正在读取图文脑…</div>
          ) : !view.available ? (
            <div className="py-10 text-center text-[11px] leading-relaxed text-fg-faint">
              图文脑不可用（MemoryUserDir 未配置）。在对话里说「记住…」即可沉淀第一条。
            </div>
          ) : facts.length === 0 ? (
            <div className="py-10 text-center text-[11px] text-fg-faint">
              {(view.facts ?? []).length === 0
                ? "还没有事实。上方记一条，或在对话里说「记住…」。"
                : "无匹配事实。"}
            </div>
          ) : (
            <div className="flex flex-col gap-1.5">
              {facts.map((f) => (
                <div
                  key={f.name}
                  className="rounded-md border px-3 py-2"
                  style={{
                    borderColor: f.invalid ? "var(--md-sys-color-outline-variant)" : undefined,
                    background: "var(--md-sys-color-surface-container)",
                    opacity: f.invalid ? 0.65 : 1,
                  }}
                >
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-[11px] font-semibold" style={{ color: "var(--md-sys-color-primary)" }}>
                      {f.name}
                    </span>
                    {f.title && f.title !== f.name && (
                      <span className="text-[11px] text-fg-faint">{f.title}</span>
                    )}
                    <span className="rounded-full px-1.5 py-px text-[9px]"
                      style={{ background: "color-mix(in srgb, var(--md-sys-color-text) 8%, transparent)" }}>
                      {FACT_TYPE_LABEL[f.type] ?? f.type}
                    </span>
                    {f.pinned && <span className="text-[9px] text-warning">固化</span>}
                    {f.invalid && <span className="text-[9px] text-fg-faint">已失效（不注入）</span>}
                    <span className="ml-auto text-[9px] text-fg-faint">
                      {f.updatedAt ? `写于 ${fmtTime(f.updatedAt)}` : ""}
                    </span>
                  </div>
                  {editing === f.name ? (
                    <div className="mt-1.5 flex flex-col gap-1">
                      <textarea
                        className="min-h-[60px] w-full rounded border bg-transparent p-2 text-[11px] outline-none"
                        style={{ borderColor: "var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text)" }}
                        value={draft}
                        onChange={(e) => setDraft(e.target.value)}
                        aria-label={`编辑事实 ${f.name}`}
                      />
                      <div className="flex gap-1.5">
                        <button
                          type="button"
                          className="cursor-pointer rounded border-0 px-2 py-1 text-[10.5px]"
                          style={{ background: "var(--md-sys-color-primary-container)", color: "var(--md-sys-color-primary)" }}
                          onClick={() => void saveEdit(f.name)}
                        >
                          保存
                        </button>
                        <button
                          type="button"
                          className="cursor-pointer rounded border-0 px-2 py-1 text-[10.5px] text-fg-faint"
                          onClick={() => { setEditing(""); setDraft(""); }}
                        >
                          取消
                        </button>
                      </div>
                    </div>
                  ) : (
                    <>
                      <div className="mt-1 whitespace-pre-wrap text-[11px] leading-relaxed" style={{ color: "var(--md-sys-color-text)" }}>
                        {f.body}
                      </div>
                      <div className="mt-1 flex gap-2">
                        <button
                          type="button"
                          className="cursor-pointer border-0 bg-transparent p-0 text-[9.5px] text-fg-faint hover:opacity-80"
                          onClick={() => void guard(() => app.WenxueMemoryPin(f.name, !f.pinned))}
                        >
                          {f.pinned ? "解固" : "固化"}
                        </button>
                        <button
                          type="button"
                          className="cursor-pointer border-0 bg-transparent p-0 text-[9.5px] text-fg-faint hover:opacity-80"
                          onClick={() => { setEditing(f.name); setDraft(f.body); }}
                        >
                          编辑
                        </button>
                        {!f.invalid ? (
                          <button
                            type="button"
                            className="cursor-pointer border-0 bg-transparent p-0 text-[9.5px] text-fg-faint hover:opacity-80"
                            onClick={() => void guard(() => app.WenxueMemoryInvalidate(f.name, "过时（面板标记）"))}
                            title="事实已被推翻：退出注入面，原文保留可恢复"
                          >
                            失效
                          </button>
                        ) : (
                          <button
                            type="button"
                            className="cursor-pointer border-0 bg-transparent p-0 text-[9.5px] text-fg-faint hover:opacity-80"
                            onClick={() => void guard(() => app.WenxueMemoryRestoreInvalidate(f.name))}
                          >
                            恢复注入
                          </button>
                        )}
                        <button
                          type="button"
                          className="cursor-pointer border-0 bg-transparent p-0 text-[9.5px] text-fg-faint hover:opacity-80"
                          onClick={() => void guard(() => app.WenxueMemoryArchive(f.name))}
                          title="软删：退出注入面，可在归档区恢复"
                        >
                          归档
                        </button>
                        <button
                          type="button"
                          className="cursor-pointer border-0 bg-transparent p-0 text-[9.5px] text-fg-faint hover:text-warning"
                          onClick={() => void guard(() => app.Forget(f.name))}
                          aria-label={`删除事实 ${f.name}`}
                        >
                          删除
                        </button>
                      </div>
                    </>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* 归档区（软删恢复面） */}
          {archived.length > 0 && (
            <div className="mt-3">
              <button
                type="button"
                className="cursor-pointer border-0 bg-transparent p-0 text-[10.5px] text-fg-faint hover:opacity-80"
                onClick={() => setArchivedOpen((v) => !v)}
                aria-expanded={archivedOpen}
              >
                归档 {archived.length} 条 {archivedOpen ? "▲" : "▼"}
              </button>
              {archivedOpen && (
                <div className="mt-1.5 flex flex-col gap-1">
                  {archived.map((a) => (
                    <div key={a.name} className="flex items-start gap-2 rounded-md border px-2.5 py-1.5"
                      style={{ borderColor: "var(--md-sys-color-outline-variant)", opacity: 0.8 }}>
                      <span className="font-mono text-[10px] text-fg-faint">{a.name}</span>
                      <span className="min-w-0 flex-1 truncate text-[10px] text-fg-faint">{a.body}</span>
                      <button
                        type="button"
                        className="cursor-pointer border-0 bg-transparent p-0 text-[9.5px] hover:opacity-80"
                        style={{ color: "var(--md-sys-color-primary)" }}
                        onClick={() => void guard(() => app.WenxueMemoryUnarchive(a.name))}
                      >
                        恢复
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        {view?.storeDir && (
          <div className="border-t px-4 py-1.5 font-mono text-[9px] text-fg-faint"
            style={{ borderColor: "var(--md-sys-color-outline-variant)" }}>
            {view.storeDir} · 文档 {view.docs?.length ?? 0} 篇
          </div>
        )}
      </div>
    </WenxueHubFrame>
  );
}
