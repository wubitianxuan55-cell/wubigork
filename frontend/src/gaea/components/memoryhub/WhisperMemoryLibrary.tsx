import { useCallback, useEffect, useMemo, useState } from "react";
import { HeartOutlined } from "@ant-design/icons";
import { RefreshCw } from "../../icons";
import { app } from "../../lib/bridge";
import type { WhisperMemoryView } from "../../lib/types";
import { EmptyState } from "../EmptyState";

// 对齐后端 memory_taxonomy.go 6 domain（与 WhisperMemoryModal 一致）
const DOMAIN_LABELS: Record<string, string> = {
  IDENTITY: "🪪 身份", SOCIAL: "💕 社交", DAILY_LIFE: "🏠 日常",
  PURSUITS: "🎯 追求", INNER_WORLD: "🧘 内心", TEMPORAL: "⏰ 时间",
};
const DOMAIN_ORDER = ["IDENTITY", "SOCIAL", "DAILY_LIFE", "PURSUITS", "INNER_WORLD", "TEMPORAL"];

/** WhisperMemoryLibrary 右脑轻语记忆库（只读）：hermes.db 记忆事实浏览。 */
export function WhisperMemoryLibrary() {
  const [facts, setFacts] = useState<WhisperMemoryView[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    app
      .WhisperMemories()
      .then((list) => {
        setFacts(list);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  useEffect(() => { load(); }, [load]);

  const q = query.trim().toLowerCase();
  const filtered = useMemo(
    () => facts.filter((f) =>
      !q ||
      f.subject.toLowerCase().includes(q) ||
      f.summary.toLowerCase().includes(q) ||
      f.subcategory.toLowerCase().includes(q),
    ),
    [facts, q],
  );

  // 按 6 domain 分组（未识别的归入"其他"）
  const groups = useMemo(() => {
    const byDomain = new Map<string, WhisperMemoryView[]>();
    for (const f of filtered) {
      const key = DOMAIN_ORDER.includes(f.domain) ? f.domain : "其他";
      if (!byDomain.has(key)) byDomain.set(key, []);
      byDomain.get(key)!.push(f);
    }
    const order = [...DOMAIN_ORDER];
    if (byDomain.has("其他")) order.push("其他");
    return order.filter((d) => byDomain.has(d)).map((d) => ({ domain: d, items: byDomain.get(d)! }));
  }, [filtered]);

  return (
    <div className="h-full flex flex-col">
      {/* 工具条 */}
      <div className="shrink-0 flex items-center gap-2 px-4 pt-3 pb-2">
        <div className="text-fg text-[13px] font-medium">轻语记忆</div>
        <span className="text-fg-faint text-[11px]">Hermes 的人格记忆（hermes.db，只读）</span>
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

      {/* 分组列表 */}
      <div className="flex-1 min-h-0 overflow-y-auto px-4 pb-4 space-y-4">
        {loading ? (
          <div className="py-10 text-center text-fg-faint text-[13px]">加载中…</div>
        ) : filtered.length === 0 ? (
          <EmptyState title="暂无轻语记忆" desc="与 Hermes 对话产生的人格记忆会出现在这里（只读浏览）" />
        ) : (
          groups.map((g) => (
            <div key={g.domain}>
              <div className="flex items-center gap-2 mb-1.5">
                <HeartOutlined style={{ fontSize: 13, color: "#f472b6" }} />
                <span className="text-fg-dim text-[12px] font-medium">{DOMAIN_LABELS[g.domain] ?? g.domain}</span>
                <span className="text-fg-faint text-[10.5px]">{g.items.length}</span>
              </div>
              <div className="space-y-1.5">
                {g.items.map((f) => (
                  <div key={f.id} className="p-2.5 rounded-lg border border-border bg-bg-soft/50">
                    <div className="flex items-center gap-2">
                      <span className="text-fg text-[12.5px] font-medium truncate">{f.subject}</span>
                      {f.tier === "core" && (
                        <span className="px-1.5 py-0.5 rounded bg-amber-400/20 text-amber-300 text-[10px]">核心</span>
                      )}
                      {f.status === "retired" && (
                        <span className="px-1.5 py-0.5 rounded bg-bg-elev text-fg-faint text-[10px]">已退役</span>
                      )}
                      <span className="ml-auto shrink-0 text-fg-faint text-[10.5px]">
                        W{f.weight.toFixed(1)} · C{f.confidence.toFixed(1)}
                      </span>
                    </div>
                    <div className="mt-1 text-fg-dim text-[12px] leading-relaxed line-clamp-2">{f.summary}</div>
                  </div>
                ))}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
