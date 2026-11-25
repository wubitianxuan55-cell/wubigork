// WenxueMemoryLibrary — 记忆中枢「图文记忆」库（2026-11-16）。
// 图文脑（wenxue-brain 独立 DB）的只读浏览面：物理隔离不动——读写面仍只在
// 图文引擎记忆工具（remember / memory_search；forget、memory_get、
// promote_session_facts 已于 2026-11-25 并入这两件），中枢同右脑聊天记忆
// 「只读」先例做跨书全览。
// 数据面 = GaeaWenxueMemoryBrowse（facts 全量，List 不过滤项目=跨书）。

import { useEffect, useMemo, useState } from "react";
import { BookOpen, RefreshCw } from "../../icons";
import { app } from "../../lib/bridge";
import type { WenxueMemoryFactView } from "../../lib/types";

const FACT_TYPE_LABEL: Record<string, string> = {
  user: "用户",
  feedback: "反馈",
  project: "项目",
  reference: "参考",
};

function fmtTime(v: string): string {
  if (!v) return "";
  const d = new Date(v);
  return Number.isNaN(d.getTime()) ? v : d.toLocaleString();
}

export function WenxueMemoryLibrary() {
  const [facts, setFacts] = useState<WenxueMemoryFactView[]>([]);
  const [storeDir, setStoreDir] = useState("");
  const [available, setAvailable] = useState(false);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    app
      .WenxueMemoryBrowse()
      .then((view) => {
        if (!alive) return;
        setFacts(view.facts ?? []);
        setStoreDir(view.storeDir ?? "");
        setAvailable(view.available);
      })
      .catch(() => {
        if (alive) setAvailable(false);
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, [reloadKey]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return facts;
    return facts.filter((f) =>
      `${f.name} ${f.title ?? ""} ${f.body}`.toLowerCase().includes(q),
    );
  }, [facts, query]);

  const invalidCount = facts.filter((f) => f.invalid).length;

  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="wenxue-memory-library">
      <div className="flex items-center gap-2 px-4 py-3">
        <BookOpen size={14} aria-hidden style={{ color: "var(--md-sys-color-primary)" }} />
        <span className="text-[13px] font-semibold">图文记忆</span>
        <span className="text-[10.5px] text-fg-faint">
          图文脑跨书全览 · 只读（读写只在图文引擎记忆工具）
        </span>
        <span className="ml-auto font-mono text-[10.5px] text-fg-faint">{facts.length} 条</span>
        <button
          type="button"
          className="cursor-pointer border-0 bg-transparent p-1 leading-none hover:opacity-80"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
          onClick={() => setReloadKey((v) => v + 1)}
          title="刷新"
          aria-label="刷新图文记忆"
        >
          <RefreshCw size={12} />
        </button>
      </div>

      {invalidCount > 0 && (
        <div className="mx-4 mb-2 rounded-md border border-warning/30 bg-warning/5 px-2 py-1.5 text-[10px] text-warning">
          {invalidCount} 条带失效标记（9.3 生命周期：原文保留可查，注入面已排除）
        </div>
      )}

      <div className="mx-4 mb-2 flex items-center gap-1.5 rounded-md border px-2 py-1.5"
        style={{ borderColor: "var(--md-sys-color-outline-variant)", background: "var(--md-sys-color-surface-container)" }}>
        <input
          className="min-w-0 flex-1 border-0 bg-transparent text-[11.5px] outline-none"
          style={{ color: "var(--md-sys-color-text)" }}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜索事实名/正文…"
          aria-label="搜索图文记忆"
        />
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-4">
        {loading ? (
          <div className="py-10 text-center text-[11px] text-fg-faint">正在读取图文脑…</div>
        ) : !available ? (
          <div className="py-10 text-center text-[11px] leading-relaxed text-fg-faint">
            图文脑不可用（MemoryUserDir 未配置或图文引擎尚未创建记忆库）。
            <br />
            在文学板块对话里让 agent 记住事实后会自动建立。
          </div>
        ) : filtered.length === 0 ? (
          <div className="py-10 text-center text-[11px] text-fg-faint">
            {facts.length === 0 ? "图文脑还没有事实。在文学板块对话里说「记住…」即可沉淀。" : "无匹配事实。"}
          </div>
        ) : (
          <div className="flex flex-col gap-1.5">
            {filtered.map((f) => (
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
                  {f.space && (
                    <span className="rounded-full px-1.5 py-px font-mono text-[9px] text-fg-faint"
                      style={{ background: "color-mix(in srgb, var(--md-sys-color-text) 6%, transparent)" }}>
                      {f.space}
                    </span>
                  )}
                  {f.pinned && <span className="text-[9px] text-warning">固化</span>}
                  {f.invalid && <span className="text-[9px] text-fg-faint">已失效</span>}
                  <span className="ml-auto text-[9px] text-fg-faint">
                    {f.updatedAt ? `写于 ${fmtTime(f.updatedAt)}` : ""}
                  </span>
                </div>
                {f.description && (
                  <div className="mt-0.5 text-[10.5px] text-fg-faint">{f.description}</div>
                )}
                <div className="mt-1 whitespace-pre-wrap text-[11px] leading-relaxed" style={{ color: "var(--md-sys-color-text)" }}>
                  {f.body}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {storeDir && (
        <div className="border-t px-4 py-1.5 font-mono text-[9px] text-fg-faint"
          style={{ borderColor: "var(--md-sys-color-outline-variant)" }}>
          {storeDir}
        </div>
      )}
    </div>
  );
}
