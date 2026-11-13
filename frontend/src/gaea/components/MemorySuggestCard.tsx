import { useMemo, useState } from "react";
import { Brain, ChevronRight, Check, X, Archive } from "../icons";
import { app } from "../lib/bridge";
import type { MemorySuggestion, MemorySuggestionsView } from "../lib/types/memory";
import { useToast } from "./Toast";
import { dreamSuggestCount } from "../lib/memorySuggest";

// MemorySuggestCard — 提炼记忆卡（波3.5 实弹反馈第四条）：
// 「本轮提炼出 N 条记忆建议」由 notice 文本行升级为结构化卡片（对话流内，
// 脑图标+计数+待确认徽标）；点击弹出建议详情弹窗——逐条展开 body、
// 接受（写入长期记忆）/忽略（出队丢弃）、溢出归档条目标「已归档」徽标
//（队列满被挤出的历史建议同弹窗可查看——修复「无法弹窗查看」）。
// 数据源=GaeaMemorySuggestions（记忆面板建议 tab 同源，memories+overflow 合并）。

type SuggestionRow = MemorySuggestion & { overflow?: boolean };

export function MemorySuggestCard({ text }: { text: string }) {
  const count = dreamSuggestCount(text);
  const overflowHint = text.includes("溢出归档");
  const [open, setOpen] = useState(false);
  const [rows, setRows] = useState<SuggestionRow[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [accepted, setAccepted] = useState<Set<string>>(new Set());
  const [dismissed, setDismissed] = useState<Set<string>>(new Set());
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const toast = useToast();

  const load = async () => {
    setLoading(true);
    try {
      const v = await app.MemorySuggestions();
      // overflow 字段由 Go 侧 MemorySuggestionsView 携带（波3.5 溢出归档）；
      // wails models 再生前 wire 类型暂无该键——本地扩展断言（再生后仍兼容）。
      const vv = v as MemorySuggestionsView & { overflow?: MemorySuggestion[] };
      const pending = vv.memories ?? [];
      const overflow = (vv.overflow ?? []).map((r) => ({ ...r, overflow: true as const }));
      // 弹窗语义=「这一轮提炼的建议」：队列里最新的 count 条优先（队列尾=最新），
      // 溢出归档条目补充在后（标已归档）。
      const latest = pending.slice(-Math.max(count, 0));
      const merged: SuggestionRow[] = [];
      const seen = new Set<string>();
      for (const r of [...latest, ...overflow]) {
        const k = r.id || r.name;
        if (seen.has(k)) continue;
        seen.add(k);
        merged.push(r);
      }
      setRows(merged);
    } catch {
      setRows([]);
      toast.show("读取记忆建议失败", "warn");
    } finally {
      setLoading(false);
    }
  };

  const openModal = () => {
    setOpen(true);
    if (rows === null) void load();
  };

  const accept = async (r: SuggestionRow) => {
    try {
      await app.AcceptMemorySuggestion({
        id: r.id, name: r.name, description: r.description,
        type: r.type, body: r.body, reason: r.reason, evidence: r.evidence ?? [],
      });
      setAccepted((s) => new Set(s).add(r.id || r.name));
      toast.show("已写入长期记忆", "info");
    } catch (e) {
      toast.show(`写入失败：${String(e).slice(0, 80)}`, "warn");
    }
  };

  const dismiss = async (r: SuggestionRow) => {
    try {
      if (!r.overflow) await app.DismissMemorySuggestion(r.id);
      // 溢出归档条目不在待确认队列——dismiss 仅从弹窗视图移除（本地状态）。
      setDismissed((s) => new Set(s).add(r.id || r.name));
    } catch (e) {
      toast.show(`忽略失败：${String(e).slice(0, 80)}`, "warn");
    }
  };

  const visible = useMemo(
    () => (rows ?? []).filter((r) => !dismissed.has(r.id || r.name)),
    [rows, dismissed],
  );

  return (
    <>
      <button
        type="button"
        data-testid="memory-suggest-card"
        onClick={openModal}
        className="flex items-center gap-2 w-full text-left rounded-lg px-3 py-2 mt-1 transition-colors cursor-pointer"
        style={{
          border: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 55%, transparent)",
          background: "color-mix(in srgb, var(--md-sys-color-surface-container) 40%, transparent)",
        }}
        title="点击查看本条回复提炼的记忆建议"
      >
        <Brain size={13} style={{ color: "var(--gaea-glow)" }} className="shrink-0" />
        <span className="text-[12px] truncate" style={{ color: "var(--md-sys-color-text)" }}>
          本轮提炼出 {count} 条记忆建议
        </span>
        <span
          className="shrink-0 rounded-full px-1.5 py-px text-[9px] leading-none"
          style={{
            color: overflowHint ? "var(--md-sys-color-warning)" : "var(--md-sys-color-text-secondary)",
            border: `1px solid ${overflowHint ? "var(--md-sys-color-warning)" : "var(--md-sys-color-outline-variant)"}`,
          }}
        >
          {overflowHint ? "队列已满 · 有归档" : "待确认"}
        </span>
        <ChevronRight size={13} className="shrink-0 ml-auto opacity-60" />
      </button>
      {open && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center"
          style={{ background: "color-mix(in srgb, var(--md-sys-color-scrim) 45%, transparent)" }}
          onClick={() => setOpen(false)}
          role="presentation"
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-label="提炼记忆建议"
            className="max-w-[640px] w-[calc(100%-48px)] max-h-[76vh] flex flex-col rounded-xl overflow-hidden"
            style={{
              background: "var(--md-sys-color-surface-container)",
              border: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 60%, transparent)",
              boxShadow: "0 12px 40px color-mix(in srgb, var(--md-sys-color-scrim) 30%, transparent)",
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <div
              className="flex items-center gap-2 px-4 py-3"
              style={{ borderBottom: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 45%, transparent)" }}
            >
              <Brain size={14} style={{ color: "var(--gaea-glow)" }} />
              <span className="text-[13px] font-medium flex-1" style={{ color: "var(--md-sys-color-text)" }}>
                提炼记忆建议（{visible.length} 条{overflowHint ? " · 含溢出归档" : ""}）
              </span>
              <button
                type="button"
                className="text-[11px] cursor-pointer rounded-md px-2 py-1"
                style={{ color: "var(--md-sys-color-text-secondary)", border: "1px solid var(--md-sys-color-outline-variant)" }}
                onClick={() => setOpen(false)}
              >
                关闭
              </button>
            </div>
            <div className="flex-1 overflow-auto px-3 py-2 flex flex-col gap-2">
              {loading && <div className="text-[12px] text-fg-faint px-1 py-2">读取中…</div>}
              {!loading && visible.length === 0 && (
                <div className="text-[12px] text-fg-faint px-1 py-3">
                  没有待确认建议（可能已被处理，或已在记忆面板「建议」区操作）。
                </div>
              )}
              {visible.map((r) => {
                const k = r.id || r.name;
                const isOpen = expanded.has(k);
                const isAccepted = accepted.has(k);
                return (
                  <div
                    key={k}
                    className="rounded-lg px-3 py-2"
                    style={{
                      border: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 50%, transparent)",
                      opacity: isAccepted ? 0.6 : undefined,
                    }}
                  >
                    <div className="flex items-center gap-1.5">
                      <span
                        className="shrink-0 rounded-full px-1.5 py-px text-[9px] font-mono leading-none"
                        style={{ color: "var(--md-sys-color-text-secondary)", border: "1px solid var(--md-sys-color-outline-variant)" }}
                      >
                        {r.type}{r.overflow ? " · 已归档" : ""}
                      </span>
                      <span className="text-[12.5px] font-medium truncate flex-1" style={{ color: "var(--md-sys-color-text)" }}>
                        {r.title || r.name}
                      </span>
                      {r.overflow && (
                        <Archive size={11} className="shrink-0 opacity-60" aria-label="溢出归档" />
                      )}
                    </div>
                    {r.description && (
                      <div className="text-[11px] mt-0.5" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                        {r.description}
                      </div>
                    )}
                    <button
                      type="button"
                      className="flex items-center gap-1 mt-1 text-[11px] cursor-pointer"
                      style={{ color: "var(--md-sys-color-text-secondary)" }}
                      onClick={() =>
                        setExpanded((s) => {
                          const n = new Set(s);
                          if (n.has(k)) n.delete(k);
                          else n.add(k);
                          return n;
                        })
                      }
                      aria-expanded={isOpen}
                    >
                      <ChevronRight size={11} className={isOpen ? "rotate-90 transition-transform" : "transition-transform"} />
                      {isOpen ? "收起正文" : "展开正文"}
                    </button>
                    {isOpen && (
                      <pre
                        className="m-0 mt-1 px-2.5 py-2 rounded-md text-[12px] leading-relaxed whitespace-pre-wrap break-words font-sans"
                        style={{ background: "color-mix(in srgb, var(--md-sys-color-surface) 60%, transparent)", color: "var(--md-sys-color-text)" }}
                      >
                        {r.body}
                      </pre>
                    )}
                    {!isAccepted && (
                      <div className="flex items-center justify-end gap-1.5 mt-1.5">
                        <button
                          type="button"
                          className="flex items-center gap-1 rounded-md px-2 py-1 text-[11px] cursor-pointer"
                          style={{ color: "var(--md-sys-color-text-secondary)", border: "1px solid var(--md-sys-color-outline-variant)" }}
                          onClick={() => void dismiss(r)}
                        >
                          <X size={11} />
                          忽略
                        </button>
                        <button
                          type="button"
                          className="flex items-center gap-1 rounded-md px-2 py-1 text-[11px] cursor-pointer"
                          style={{ color: "var(--md-sys-color-on-primary)", background: "var(--md-sys-color-primary)" }}
                          onClick={() => void accept(r)}
                        >
                          <Check size={11} />
                          接受写入
                        </button>
                      </div>
                    )}
                    {isAccepted && (
                      <div className="text-right text-[11px] mt-1" style={{ color: "var(--md-sys-color-success)" }}>
                        已写入长期记忆
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
            <div
              className="px-3 py-2 text-[10.5px]"
              style={{
                color: "var(--md-sys-color-text-secondary)",
                borderTop: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 45%, transparent)",
              }}
            >
              接受=写入长期记忆（记忆面板可复核）；忽略=出队丢弃。完整管理见记忆面板「建议」区。
            </div>
          </div>
        </div>
      )}
    </>
  );
}
