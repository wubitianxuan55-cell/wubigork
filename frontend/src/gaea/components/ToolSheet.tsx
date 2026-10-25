// ToolSheet — 办公板块里的临时工具面板。
//
// 边界（docs/gaea-ai-workbench-ui-2026-10.md）：CAD 看图 / 进度计划不是工作台，
// 不占导航、不占右栏一级 Tab；它们是 AI 或用户从对话里调用的工具。打开时在
// 对话区上方浮出，关闭即回原对话；产物留在文件/产物里，重开不重跑解析。
import { useEffect, useRef } from "react";
import type { ReactNode } from "react";
import { Wrench, X } from "../icons";

export function ToolSheet({
  open,
  title,
  subtitle,
  onClose,
  children,
  testId = "tool-sheet",
}: {
  open: boolean;
  title: string;
  /** 头右侧说明（如「临时工具 · 关闭后结果留在产物」）。 */
  subtitle?: string;
  onClose: () => void;
  children: ReactNode;
  testId?: string;
}) {
  const panelRef = useRef<HTMLDivElement>(null);
  const returnFocusRef = useRef<HTMLElement | null>(null);
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  useEffect(() => {
    if (!open) return;
    returnFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const panel = panelRef.current;
    // 焦点进面板容器本身（不抢第一个按钮的 focus ring）；Tab 再进具体控件。
    panel?.focus();
    // 容器焦点只为键盘入口；焦点环留给内部控件（全局 :focus-visible 带 !important，内联也压）。
    panel?.style.setProperty("outline", "none", "important");

    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        onCloseRef.current();
        return;
      }
      if (event.key !== "Tab" || !panel) return;
      // 焦点陷阱：Tab 在面板内循环，不跑到背后的对话/输入框。
      const nodes = Array.from(
        panel.querySelectorAll<HTMLElement>(
          'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
        ),
      ).filter((el) => el.offsetParent !== null);
      if (nodes.length === 0) return;
      const firstNode = nodes[0];
      const lastNode = nodes[nodes.length - 1];
      if (event.shiftKey && document.activeElement === firstNode) {
        event.preventDefault();
        lastNode.focus();
      } else if (!event.shiftKey && document.activeElement === lastNode) {
        event.preventDefault();
        firstNode.focus();
      }
    };
    document.addEventListener("keydown", onKey, true);
    return () => {
      document.removeEventListener("keydown", onKey, true);
      returnFocusRef.current?.focus?.();
    };
  }, [open]);

  if (!open) return null;

  return (
    <div className="absolute inset-0 z-40" data-testid={testId}>
      <div
        className="absolute inset-0 bg-black/45"
        aria-hidden="true"
        data-testid={`${testId}-backdrop`}
        onClick={onClose}
      />
      <section
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        tabIndex={-1}
        className="tool-sheet-panel absolute inset-2 flex min-h-0 flex-col overflow-hidden rounded-xl border border-border bg-bg shadow-2xl"
        style={{ boxShadow: "var(--ds-shadow-dialog, 0 24px 80px rgba(0,0,0,.55))" }}
      >
        <header className="flex h-10 shrink-0 items-center gap-2 border-b border-border-soft px-3">
          <span
            className="inline-flex items-center gap-1 rounded border px-1.5 py-px font-mono text-[9px] tracking-wider text-accent"
            style={{ borderColor: "color-mix(in srgb, var(--gaea-glow) 35%, transparent)", background: "var(--accent-soft)" }}
          >
            <Wrench size={10} aria-hidden />
            工具
          </span>
          <b className="text-[12px] font-medium">{title}</b>
          {subtitle && <span className="ml-auto hidden text-[9px] text-fg-faint sm:inline">{subtitle}</span>}
          <button
            type="button"
            className="ml-2 inline-flex h-7 w-7 items-center justify-center rounded-md border-0 bg-transparent text-fg-dim cursor-pointer hover:bg-bg-soft hover:text-fg"
            onClick={onClose}
            aria-label={`关闭${title}`}
            data-testid={`${testId}-close`}
          >
            <X size={12} />
          </button>
        </header>
        <div className="min-h-0 flex-1 overflow-hidden">{children}</div>
      </section>
    </div>
  );
}


