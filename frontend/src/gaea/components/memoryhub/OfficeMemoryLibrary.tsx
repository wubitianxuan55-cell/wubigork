import { useCallback, useEffect, useMemo, useState } from "react";
import { FileText, RefreshCw } from "../../icons";
import { app } from "../../lib/bridge";
import type { MemoryFact, MemoryView } from "../../lib/types";
import { FactCard } from "../FactCard";
import { EmptyState } from "../EmptyState";

const TYPE_FILTERS = ["all", "user", "feedback", "project", "reference"] as const;

/** OfficeMemoryLibrary 左脑办公记忆库：Hephaestus 的 facts 条目管理。 */
export function OfficeMemoryLibrary() {
  const [view, setView] = useState<MemoryView | null>(null);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [highlight, setHighlight] = useState<string | null>(null);

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

  const handleSave = async (name: string, body: string) => {
    await app.UpdateFact(name, body);
    await load();
  };
  const handleForget = async (name: string) => {
    await app.Forget(name);
    await load();
  };
  const handleChangeType = async (name: string, newType: string) => {
    await app.ChangeFactType(name, newType);
    await load();
  };

  return (
    <div className="h-full flex flex-col">
      {/* 工具条 */}
      <div className="shrink-0 flex items-center gap-2 px-4 pt-3 pb-2">
        <div className="text-fg text-[13px] font-medium">办公记忆</div>
        <span className="text-fg-faint text-[11px]">Hephaestus 跨会话记住的工作事实（Type × Kind）</span>
        <div className="ml-auto flex items-center gap-2">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索记忆…"
            className="w-44 px-3 h-8 rounded-lg border border-border bg-bg text-fg text-[12px] placeholder:text-fg-faint outline-none focus:border-accent transition-colors"
          />
          <button
            className="inline-flex items-center gap-1 px-2.5 h-8 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[12px]"
            onClick={load}
            title="刷新"
          >
            <RefreshCw size={13} />
          </button>
        </div>
      </div>

      {/* 类型筛选 */}
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
        <span className="ml-auto text-fg-faint text-[11px]">{filtered.length} 条</span>
      </div>

      {/* 列表 */}
      <div className="flex-1 min-h-0 overflow-y-auto px-4 pb-4 space-y-2">
        {loading ? (
          <div className="py-10 text-center text-fg-faint text-[13px]">加载中…</div>
        ) : filtered.length === 0 ? (
          <EmptyState
            title={view?.available === false ? "办公记忆不可用" : "暂无记忆"}
            desc={view?.available === false ? "未配置用户目录" : "办公 agent 用 remember 工具保存的事实会出现在这里"}
          />
        ) : (
          filtered.map((f) => (
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
    </div>
  );
}
