import { useEffect, useRef } from "react";
import { Brain, Copy } from "../icons";
import { useT } from "../lib/i18n";
import { useToast } from "./Toast";
import { parseMemoryPeek } from "../lib/memoryPeek";

// MemoryPeekModal — 提炼记忆点击查看弹窗（波3.5 五卡实弹反馈第三条）：
// 过程卡里的 remember（记忆沉淀）/fact_add（事实底座）工具行点击 → 弹窗展示
// 该条记忆的完整内容（类型标签 + 标题/钩子/正文），不再要求展开工具卡翻 JSON。
// 自绘遮罩弹窗（ApprovalModal 同视觉语言：office-decision 系 token）；Escape/
// 遮罩点击关闭；body 只读 + 复制按钮。

export interface MemoryPeekPayload {
  tool: string;
  args?: string;
}

export function MemoryPeekModal({ payload, onClose }: { payload: MemoryPeekPayload | null; onClose: () => void }) {
  const t = useT();
  const toast = useToast();
  const bodyRef = useRef<HTMLPreElement | null>(null);
  const parsed = payload ? parseMemoryPeek(payload.tool, payload.args) : null;
  useEffect(() => {
    if (!payload) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [payload, onClose]);
  if (!payload || !parsed) return null;
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      style={{ background: "color-mix(in srgb, var(--md-sys-color-scrim) 45%, transparent)" }}
      onClick={onClose}
      role="presentation"
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label={parsed.title}
        className="max-w-[560px] w-[calc(100%-48px)] max-h-[70vh] flex flex-col rounded-xl overflow-hidden"
        style={{
          background: "var(--md-sys-color-surface-container)",
          border: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 60%, transparent)",
          boxShadow: "0 12px 40px color-mix(in srgb, var(--md-sys-color-scrim) 30%, transparent)",
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2 px-4 py-3" style={{ borderBottom: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 45%, transparent)" }}>
          <Brain size={14} style={{ color: "var(--gaea-glow)" }} />
          <span className="text-[13px] font-medium truncate flex-1" style={{ color: "var(--md-sys-color-text)" }}>
            {parsed.title}
          </span>
          <span
            className="shrink-0 rounded-full px-1.5 py-px text-[9px] font-mono leading-none"
            style={{ color: "var(--md-sys-color-text-secondary)", border: "1px solid var(--md-sys-color-outline-variant)" }}
          >
            {parsed.typeLabel}
          </span>
        </div>
        {parsed.hook && (
          <div className="px-4 pt-2.5 text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            {parsed.hook}
          </div>
        )}
        <pre
          ref={bodyRef}
          className="flex-1 overflow-auto px-4 py-3 m-0 text-[12px] leading-relaxed whitespace-pre-wrap break-words font-sans"
          style={{ color: "var(--md-sys-color-text)" }}
        >
          {parsed.body}
        </pre>
        <div className="flex items-center justify-end gap-1.5 px-3 py-2" style={{ borderTop: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 45%, transparent)" }}>
          <button
            type="button"
            className="flex items-center gap-1 rounded-md px-2 py-1 text-[11px] cursor-pointer"
            style={{ color: "var(--md-sys-color-text-secondary)", border: "1px solid var(--md-sys-color-outline-variant)" }}
            onClick={() => {
              void navigator.clipboard
                .writeText(parsed.body)
                .then(() => toast.show(t("deliver.copyPathDone"), "info"))
                .catch(() => toast.show(t("deliver.copyFail"), "warn"));
            }}
          >
            <Copy size={11} />
            {t("deliver.copyPath")}
          </button>
          <button
            type="button"
            className="rounded-md px-2.5 py-1 text-[11px] cursor-pointer"
            style={{ color: "var(--md-sys-color-on-primary)", background: "var(--md-sys-color-primary)" }}
            onClick={onClose}
          >
            {t("common.close")}
          </button>
        </div>
      </div>
    </div>
  );
}
