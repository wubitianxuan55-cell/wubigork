import { useCallback, useEffect, useState } from "react";
import { ChevronDown, ChevronRight, RefreshCw } from "../icons";
import type { AIFeedbackItem } from "../lib/types";
import { app } from "../lib/bridge";
import { useToast } from "./Toast";
import { saveExportBlob } from "../lib/saveFile";

/**
 * AIFeedbackSection — 记忆面板「AI 反馈」分区（2026-11-20 立项）：ai_feedback
 * 工具收集的模型侧反馈（<userDir>/ai-feedback.jsonl，全局跨板块），此处查看/
 * 删除/勾选导出 md（saveExportBlob 双门：壳内另存为对话框）。导出分组按
 * category，徽标显示原始枚举值（模型自报，技术词保留原文）。
 */
// wenxue 镜像（2026-11-20）：与 gaea/components/AIFeedbackSection 同源——板块纪律
// 走本板块 bridge/types/saveFile；反馈库全局同源（<userDir>/ai-feedback.jsonl）。
export function AIFeedbackSection(p: { active: boolean; onCount?: (n: number) => void }) {
  const toast = useToast();
  const [items, setItems] = useState<AIFeedbackItem[]>([]);
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [expanded, setExpanded] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setBusy(true);
    try {
      const list = (await app.AIFeedbackList()) ?? [];
      setItems(list);
      setSelected(new Set());
      p.onCount?.(list.length);
    } catch (e) {
      toast.show(String(e), "error");
    } finally {
      setBusy(false);
    }
  }, [toast]);

  useEffect(() => {
    if (p.active) void refresh();
  }, [p.active, refresh]);

  const toggle = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const allSelected = items.length > 0 && selected.size === items.length;
  const toggleAll = () => setSelected(allSelected ? new Set() : new Set(items.map((i) => i.id)));

  const remove = async (id: string) => {
    setDeleting(id);
    try {
      await app.AIFeedbackDelete(id);
      setItems((prev) => {
        const next = prev.filter((i) => i.id !== id);
        p.onCount?.(next.length);
        return next;
      });
      setSelected((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    } catch (e) {
      toast.show(String(e), "error");
    } finally {
      setDeleting(null);
    }
  };

  // 导出选中为 md：按 category 分组，组内保持时间序（新→旧）。
  const exportMd = async () => {
    const chosen = items.filter((i) => selected.has(i.id));
    if (chosen.length === 0) {
      toast.show("先勾选要导出的反馈", "info");
      return;
    }
    const now = new Date();
    const stamp = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;
    const lines: string[] = [
      `# AI 反馈导出（${stamp}）`,
      "",
      `共 ${chosen.length} 条，来自 ai_feedback 工具（模型在任务中遇到的困难与改进建议）。`,
      "",
    ];
    const catName: Record<string, string> = {
      "tool-friction": "工具不好用（tool-friction）",
      bug: "行为不对（bug）",
      "missing-capability": "缺能力（missing-capability）",
      workflow: "工作流摩擦（workflow）",
      other: "其他（other）",
    };
    const sevName: Record<string, string> = { high: "高", medium: "中", low: "低" };
    for (const cat of ["bug", "tool-friction", "missing-capability", "workflow", "other"]) {
      const group = chosen.filter((i) => (i.category || "other") === cat);
      if (group.length === 0) continue;
      lines.push(`## ${catName[cat] ?? cat}`, "");
      for (const it of group) {
        lines.push(`### ${it.title}`);
        const meta = [
          it.board ? `板块：${it.board}` : "",
          it.severity ? `严重度：${sevName[it.severity] ?? it.severity}` : "",
          `时间：${it.createdAt}`,
        ].filter(Boolean).join(" ｜ ");
        lines.push(meta, "");
        if (it.detail) {
          lines.push(it.detail, "");
        }
      }
    }
    const blob = new Blob([lines.join("\n")], { type: "text/markdown" });
    const saved = await saveExportBlob(blob, `ai-feedback-${stamp}.md`);
    if (saved) toast.show("已导出 md", "info");
  };

  if (busy && items.length === 0) {
    return <div className="text-fg-faint text-[11.5px] px-1 py-3">{"加载反馈…"}</div>;
  }
  if (items.length === 0) {
    return <div className="text-fg-faint text-[11.5px] px-1 py-3">{"暂无反馈——模型在任务中遇到困难、发现不足时会经 ai_feedback 工具提交到这里"}</div>;
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2 flex-wrap">
        <label className="flex items-center gap-1 text-[11px] text-fg-faint cursor-pointer select-none">
          <input type="checkbox" checked={allSelected} onChange={toggleAll} className="accent-accent" />
          {"全选"}
        </label>
        <button
          className="text-[11px] px-2 py-0.5 rounded-md border border-border-soft text-fg-faint hover:text-fg hover:border-accent transition-colors bg-transparent cursor-pointer disabled:opacity-50"
          disabled={selected.size === 0}
          onClick={() => void exportMd()}
        >
          {`导出选中 (${selected.size})`}
        </button>
        <button
          className="ml-auto flex items-center gap-1 text-[11px] text-fg-faint hover:text-fg bg-transparent border-0 cursor-pointer"
          onClick={() => void refresh()}
        >
          <RefreshCw size={11} />
        </button>
      </div>
      {items.map((it) => (
        <div
          key={it.id}
          className="border border-border-soft rounded-xl p-3 bg-bg-soft/60 hover:bg-bg-soft transition-colors"
        >
          <div className="flex items-start gap-2">
            <input
              type="checkbox"
              checked={selected.has(it.id)}
              onChange={() => toggle(it.id)}
              className="mt-0.5 accent-accent"
              aria-label={it.title}
            />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-1.5 mb-1 flex-wrap">
                {it.board && <span className="badge badge--muted">{it.board}</span>}
                {it.category && <span className="badge badge--muted">{it.category}</span>}
                {it.severity === "high" && (
                  <span className="text-rose-400 text-[10px] font-semibold">{"高"}</span>
                )}
                <span className="text-fg-faint/60 text-[10px] ml-auto">{it.createdAt}</span>
              </div>
              <button
                className="text-left w-full bg-transparent border-0 p-0 cursor-pointer text-fg text-[12.5px] font-medium flex items-start gap-1"
                onClick={() => setExpanded(expanded === it.id ? null : it.id)}
              >
                {expanded === it.id ? <ChevronDown size={12} className="mt-0.5 shrink-0" /> : <ChevronRight size={12} className="mt-0.5 shrink-0" />}
                <span className="min-w-0">{it.title}</span>
              </button>
              {expanded === it.id && it.detail && (
                <div className="mt-1.5 text-fg-faint text-[11px] leading-relaxed whitespace-pre-wrap border-l-2 border-fg-faint/15 pl-2">
                  {it.detail}
                </div>
              )}
            </div>
            <button
              className="shrink-0 text-[11px] text-fg-faint hover:text-rose-400 bg-transparent border-0 cursor-pointer disabled:opacity-50"
              disabled={deleting === it.id}
              onClick={() => void remove(it.id)}
            >
              {"删除"}
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}
