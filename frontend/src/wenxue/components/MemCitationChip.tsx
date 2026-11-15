import { memo, useCallback, useEffect, useState } from "react";
import { Brain } from "../icons";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";
import type { MemoryFact } from "../lib/types";

// MemCitationChip — 记忆引用徽标（C2 记忆引用可追溯，蒸馏 codex memories/read
// citation 闭环）：渲染模型回答中的 [MEM:<name>] 引用键，点击弹层展示记忆
// 详情（标题/类型/正文摘要/最近使用/沉淀来源）——办公用户可验证「你引用的
// 资料是不是真的」，并暴露记忆的沉淀来源会话。
export const MemCitationChip = memo(function MemCitationChip({ name }: { name: string }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [fact, setFact] = useState<MemoryFact | null>(null);
  const [missing, setMissing] = useState(false);

  const load = useCallback(async () => {
    setMissing(false);
    try {
      const view = await app.Memory();
      const hit = view.facts.find((f) => f.name.toLowerCase() === name.toLowerCase()) ?? null;
      if (hit) setFact(hit);
      else setMissing(true);
    } catch {
      setMissing(true);
    }
  }, [name]);

  useEffect(() => {
    if (!open) return;
    void load();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, load]);

  return (
    <span className="relative inline-block align-middle">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        title={t("mem.citationChipTitle", { name })}
        aria-label={t("mem.citationChipTitle", { name })}
        className="inline-flex items-center gap-1 align-middle mx-0.5 px-1 py-px rounded bg-accent/10 text-accent text-[0.82em] font-mono cursor-pointer hover:bg-accent/20 transition-colors"
      >
        <Brain size={10} className="shrink-0" aria-hidden />
        <span className="max-w-[180px] truncate">{name}</span>
      </button>
      {open && (
        <>
          {/* 透明遮罩：点击弹层外任意处关闭（不挡视觉） */}
          <span className="fixed inset-0 z-40 cursor-default" onClick={() => setOpen(false)} aria-hidden />
          <span
            role="dialog"
            aria-label={t("mem.citationTitle")}
            className="absolute left-0 top-full z-50 mt-1 w-72 block rounded-md border border-border-soft bg-bg-elev-2 shadow-lg p-3 text-left"
          >
            {missing ? (
              <span className="block text-[12px] text-fg-dim">{t("mem.citationNotFound")}</span>
            ) : fact ? (
              <>
                <span className="flex items-center gap-1.5 mb-1">
                  <Brain size={12} className="text-accent shrink-0" aria-hidden />
                  <span className="text-[13px] font-semibold text-fg truncate">{fact.title || fact.name}</span>
                  <span className="ml-auto shrink-0 text-[9px] uppercase text-fg-faint/70 border border-border-soft/60 rounded px-1 py-px font-mono">
                    {fact.type}
                  </span>
                </span>
                {fact.description && (
                  <span className="block text-[12px] text-fg-dim leading-relaxed mb-1">{fact.description}</span>
                )}
                {fact.body && (
                  <span className="block text-[12px] text-fg-dim/80 leading-relaxed max-h-28 overflow-auto whitespace-pre-wrap">
                    {fact.body}
                  </span>
                )}
                {(fact.sourceSession || fact.lastUsedAt) && (
                  <span className="block mt-2 pt-2 border-t border-border-soft/60 text-[10.5px] text-fg-faint space-y-0.5">
                    {fact.sourceSession && (
                      <span className="block truncate" title={fact.sourceSession}>
                        {t("mem.citationSource")}: {fact.sourceSession}
                        {fact.sourceMessage ? ` · ${fact.sourceMessage}` : ""}
                      </span>
                    )}
                    {fact.lastUsedAt && (
                      <span className="block">{t("mem.citationLastUsed")}: {fact.lastUsedAt.slice(0, 10)}</span>
                    )}
                  </span>
                )}
              </>
            ) : (
              <span className="block text-[12px] text-fg-faint">{t("mem.citationLoading")}</span>
            )}
            <button
              type="button"
              onClick={() => setOpen(false)}
              className="absolute right-1.5 top-1.5 border-0 bg-transparent text-fg-faint/60 cursor-pointer hover:text-fg text-[11px] px-1"
              aria-label={t("common.close")}
            >
              ✕
            </button>
          </span>
        </>
      )}
    </span>
  );
});
