// WenxueMemoryPanel — 图文板块「记忆」标签页（2026-11-16 面板恢复批）。
// 与旧面板（办公脑做梦/晨报/建议窗，已退役）的本质差异：数据面全锚图文脑
// （wenxue-brain 独立 DB）——facts/docs/scopes 同源，写侧五件（Remember/
// Forget/UpdateFact/ChangeFactType/SaveDoc）经 wx 控制器落图文脑。
// 做梦/晨报/建议是办公脑机制，图文引擎有意不挂：面板如实展示空态说明。

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Brain, Plus, RefreshCw, Search, Trash2 } from "../icons";
import { app } from "../lib/bridge";
import { subscribe } from "../../events";
import type { MemoryView } from "../lib/types";

const FACT_TYPE_LABEL: Record<string, string> = {
  user: "用户",
  feedback: "反馈",
  project: "项目",
  reference: "参考",
};

export function WenxueMemoryPanel() {
  const [view, setView] = useState<MemoryView | null>(null);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
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
      const v = await app.Memory();
      if (token !== reloadRef.current) return;
      setView(v);
    } catch (err) {
      if (token === reloadRef.current) setError(err instanceof Error ? err.message : String(err));
    }
  }, []);

  useEffect(() => {
    void load();
    // agent 回写（remember/forget 工具）经 filetree 通道不覆盖记忆库——记忆
    // 是 DB 面；轮询外再给一个手动刷新钮即可。
    return subscribe("wenxue:workspace-changed", () => void load());
  }, [load]);

  const facts = useMemo(() => {
    const all = view?.facts ?? [];
    const q = query.trim().toLowerCase();
    if (!q) return all;
    return all.filter((f) => `${f.name} ${f.title ?? ""} ${f.body}`.toLowerCase().includes(q));
  }, [view, query]);

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

  const forget = async (name: string) => {
    try {
      await app.Forget(name);
      if (editing === name) setEditing("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const scopes = view?.scopes?.length
    ? view.scopes
    : [{ scope: "project", path: "" }, { scope: "user", path: "" }, { scope: "local", path: "" }];

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="wenxue-memory-panel">
      <div className="flex items-center gap-2 px-4 py-3">
        <Brain size={14} aria-hidden style={{ color: "var(--md-sys-color-primary)" }} />
        <span className="text-[13px] font-semibold">图文脑记忆</span>
        <span className="text-[10.5px] text-fg-faint">跨会话事实 · 做梦/晨报是办公脑机制，图文脑不跑</span>
        <span className="ml-auto font-mono text-[10.5px] text-fg-faint">{view?.facts?.length ?? 0} 条</span>
        <button
          type="button"
          className="cursor-pointer border-0 bg-transparent p-1 leading-none hover:opacity-80"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
          onClick={() => void load()}
          title="刷新"
          aria-label="刷新图文脑记忆"
        >
          <RefreshCw size={12} />
        </button>
      </div>

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
          placeholder="记一条事实（Enter 快捷添加，agent 写作时自动召回）…"
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

      <div className="mx-4 mb-2 flex items-center gap-1.5 rounded-md border px-2 py-1.5"
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

      <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-4">
        {!view ? (
          <div className="py-10 text-center text-[11px] text-fg-faint">正在读取图文脑…</div>
        ) : !view.available ? (
          <div className="py-10 text-center text-[11px] leading-relaxed text-fg-faint">
            图文脑不可用（MemoryUserDir 未配置）。在对话里说「记住…」即可沉淀第一条。
          </div>
        ) : facts.length === 0 ? (
          <div className="py-10 text-center text-[11px] text-fg-faint">
            还没有事实。上方记一条，或在对话里说「记住…」。
          </div>
        ) : (
          <div className="flex flex-col gap-1.5">
            {facts.map((f) => (
              <div
                key={f.name}
                className="rounded-md border px-3 py-2"
                style={{
                  borderColor: "var(--md-sys-color-outline-variant)",
                  background: "var(--md-sys-color-surface-container)",
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
                        onClick={() => { setEditing(f.name); setDraft(f.body); }}
                      >
                        编辑
                      </button>
                      <button
                        type="button"
                        className="flex cursor-pointer items-center gap-0.5 border-0 bg-transparent p-0 text-[9.5px] text-fg-faint hover:text-warning"
                        onClick={() => void forget(f.name)}
                        aria-label={`删除事实 ${f.name}`}
                      >
                        <Trash2 size={10} aria-hidden /> 删除
                      </button>
                    </div>
                  </>
                )}
              </div>
            ))}
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
  );
}
