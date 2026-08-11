import { useCallback, useEffect, useMemo, useState } from "react";
import { Modal } from "antd";
import { FileText, RefreshCw, Sparkles } from "../../icons";
import { app } from "../../lib/bridge";
import type { MemoryDuplicateView, MemoryFact, MemoryView } from "../../lib/types";
import { FactCard } from "../FactCard";
import { DocEditor } from "../DocEditor";
import { ArchivesSection } from "../ArchivesSection";
import { EmptyState } from "../EmptyState";

const TYPE_FILTERS = ["all", "user", "feedback", "project", "reference"] as const;
type TabKey = "facts" | "docs" | "archives";

const TABS: { key: TabKey; label: string }[] = [
  { key: "facts", label: "记忆" },
  { key: "docs", label: "文档" },
  { key: "archives", label: "归档" },
];

/** OfficeMemoryLibrary 左脑办公记忆库：gaea 的 facts/docs/archives 管理。 */
export function OfficeMemoryLibrary() {
  const [view, setView] = useState<MemoryView | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [tab, setTab] = useState<TabKey>("facts");
  const [query, setQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [highlight, setHighlight] = useState<string | null>(null);
  const [dupOpen, setDupOpen] = useState(false);
  const [dups, setDups] = useState<MemoryDuplicateView[]>([]);
  const [dupMsg, setDupMsg] = useState<string | null>(null);
  const [merging, setMerging] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    app
      .Memory()
      .then((v) => {
        setView(v);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const facts = view?.facts ?? [];
  const factNames = useMemo(() => new Set(facts.map((f) => f.name)), [facts]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return facts.filter((f) => {
      if (typeFilter !== "all" && f.type !== typeFilter) return false;
      if (!q) return true;
      return (
        f.name.toLowerCase().includes(q) ||
        f.description.toLowerCase().includes(q) ||
        f.body.toLowerCase().includes(q)
      );
    });
  }, [facts, query, typeFilter]);

  const toggleExpanded = (name: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const refresh = useCallback(async () => {
    await load();
    setBusy(false);
  }, [load]);

  const handleSave = async (name: string, body: string) => {
    await app.UpdateFact(name, body);
    await refresh();
  };
  const handleForget = async (name: string) => {
    await app.Forget(name);
    await refresh();
  };
  const handleChangeType = async (name: string, newType: string) => {
    await app.ChangeFactType(name, newType);
    await refresh();
  };
  const handleSaveDoc = async (path: string, body: string) => {
    setBusy(true);
    await app.SaveDoc(path, body);
    await refresh();
  };

  const openDuplicates = useCallback(async () => {
    const hits = await app.MemoryDuplicates(0.55).catch(() => [] as MemoryDuplicateView[]);
    setDups(hits ?? []);
    setDupMsg(null);
    setDupOpen(true);
  }, []);

  const doMergePair = useCallback(async (keep: string, dup: string) => {
    setMerging(keep);
    try {
      await app.MemoryMerge(keep, [dup]);
      setDups((prev) => prev.filter((d) => !(d.keep === keep && d.dup === dup)));
      setDupMsg(`已合并「${dup}」→「${keep}」`);
      await refresh();
    } finally {
      setMerging(null);
    }
  }, [refresh]);

  const doMergeAll = useCallback(async () => {
    setMerging("all");
    try {
      for (const d of [...dups]) {
        await app.MemoryMerge(d.keep, [d.dup]).catch(() => {});
      }
      setDups([]);
      setDupMsg(`已合并 ${dups.length} 对重复记忆`);
      await refresh();
    } finally {
      setMerging(null);
    }
  }, [dups, refresh]);

  return (
    <div className="h-full flex flex-col">
      {/* 工具条 */}
      <div className="shrink-0 flex items-center gap-2 px-4 pt-3 pb-2">
        <div className="text-fg text-[13px] font-medium">办公记忆</div>
        <span className="text-fg-faint text-[11px]">gaea 跨会话记忆（facts/docs/归档）</span>
        <div className="ml-auto flex items-center gap-2">
          {tab === "facts" && (
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索记忆…"
              className="w-44 px-3 h-8 rounded-lg border border-border bg-bg text-fg text-[12px] placeholder:text-fg-faint outline-none focus:border-accent transition-colors"
            />
          )}
          {tab === "facts" && (
            <button
              className="inline-flex items-center gap-1 px-2.5 h-8 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[12px]"
              onClick={() => void openDuplicates()}
              title="查重：合并疑似重复的记忆事实"
            >
              <Sparkles size={13} className="text-accent" />
              查重合并
            </button>
          )}
          <button
            className="inline-flex items-center gap-1 px-2.5 h-8 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[12px]"
            onClick={load}
            title="刷新"
          >
            <RefreshCw size={13} />
          </button>
        </div>
      </div>

      {/* tabs */}
      <div className="shrink-0 flex items-center gap-1 px-4 pb-2">
        {TABS.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-3 h-7 rounded-full text-[12px] transition-colors ${
              tab === t.key
                ? "bg-accent text-white"
                : "text-fg-faint hover:text-fg hover:bg-bg-soft"
            }`}
          >
            {t.label}
            {t.key === "facts" && (
              <span className="ml-1 opacity-70">{filtered.length}</span>
            )}
            {t.key === "archives" && (
              <span className="ml-1 opacity-70">{view?.archives?.length ?? 0}</span>
            )}
          </button>
        ))}
      </div>

      {/* facts tab */}
      {tab === "facts" && (
        <>
          <div className="shrink-0 flex items-center gap-1.5 px-4 pb-2 flex-wrap">
            {TYPE_FILTERS.map((t) => (
              <button
                key={t}
                onClick={() => setTypeFilter(t)}
                className={`px-2.5 h-7 rounded-full text-[11.5px] transition-colors ${
                  typeFilter === t
                    ? "bg-accent text-white"
                    : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
                }`}
              >
                {t === "all" ? "全部" : t}
              </button>
            ))}
          </div>
          <div className="flex-1 min-h-0 overflow-y-auto px-4 pb-4 space-y-2">
            {loading ? (
              <div className="py-10 text-center text-fg-faint text-[13px]">加载中…</div>
            ) : filtered.length === 0 ? (
              <EmptyState message={view?.available === false ? "办公记忆不可用 — 未配置用户目录" : "暂无记忆 — 办公 agent 用 remember 工具保存的事实会出现在这里"} />
            ) : (
              filtered.map((f: MemoryFact) => (
                <FactCard
                  key={f.name}
                  fact={f}
                  factNames={factNames}
                  expanded={expanded.has(f.name)}
                  highlight={highlight === f.name}
                  onToggle={() => toggleExpanded(f.name)}
                  onJump={(name) => {
                    setHighlight(name);
                    toggleExpanded(name);
                    setTimeout(() => setHighlight(null), 1500);
                  }}
                  onSave={handleSave}
                  onForget={() => handleForget(f.name)}
                  onChangeType={handleChangeType}
                />
              ))
            )}
          </div>
        </>
      )}

      {/* docs tab */}
      {tab === "docs" && (
        <div className="flex-1 min-h-0 overflow-y-auto px-4 pb-4">
          {loading ? (
            <div className="py-10 text-center text-fg-faint text-[13px]">加载中…</div>
          ) : (
            <DocEditor docs={view?.docs ?? []} onSaveDoc={handleSaveDoc} busy={busy} />
          )}
        </div>
      )}

      {/* archives tab */}
      {tab === "archives" && (
        <div className="flex-1 min-h-0 overflow-y-auto px-4 pb-4">
          <ArchivesSection archives={view?.archives ?? []} />
        </div>
      )}

      {/* 查重合并弹窗 */}
      <Modal
        title="查重合并：疑似重复记忆"
        open={dupOpen}
        onCancel={() => setDupOpen(false)}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        footer={
          <div className="flex items-center gap-2">
            <span className="mr-auto text-[11px] text-fg-faint">{dups.length} 对疑似重复</span>
            <button className="px-3 h-8 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft text-[12px]" onClick={() => setDupOpen(false)} type="button">关闭</button>
            {dups.length > 0 && (
              <button
                className="px-3 h-8 rounded-lg bg-accent text-white text-[12px] hover:opacity-90 disabled:opacity-50"
                onClick={() => void doMergeAll()}
                disabled={merging !== null}
                type="button"
              >
                {merging === "all" ? "合并中…" : `全部合并（${dups.length} 对）`}
              </button>
            )}
          </div>
        }
        width={620}
      >
        {dupMsg && <div className="mb-2 px-3 py-2 rounded-lg bg-bg-elev text-accent text-[11.5px]">{dupMsg}</div>}
        {dups.length === 0 ? (
          <div className="py-6 text-center text-fg-faint text-[12px]">
            {dupMsg ? dupMsg : "暂无疑似重复记忆"}
          </div>
        ) : (
          <div className="space-y-1.5 max-h-[46vh] overflow-auto">
            {dups.map((d) => (
              <div key={`${d.keep}-${d.dup}`} className="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-bg-soft/40 text-[12px]">
                <span className="min-w-0 flex-1 truncate">
                  <span className="text-fg">{d.keepTitle}</span>
                  <span className="text-fg-faint"> ⇄ </span>
                  <span className="text-fg-dim">{d.dupTitle}</span>
                </span>
                <span className="shrink-0 text-fg-faint text-[10.5px]">{Math.round(d.score * 100)}%</span>
                <button
                  className="shrink-0 px-2 h-6 rounded-md bg-accent/15 text-accent text-[11px] cursor-pointer hover:bg-accent/25 disabled:opacity-50"
                  disabled={merging !== null}
                  onClick={() => void doMergePair(d.keep, d.dup)}
                  type="button"
                  title={`把「${d.dupTitle}」合并进「${d.keepTitle}」（标签并集、来源删除）`}
                >
                  {merging === d.keep ? "合并中…" : "合并"}
                </button>
              </div>
            ))}
          </div>
        )}
      </Modal>
    </div>
  );
}
