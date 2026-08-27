import { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { MessageSquare, X } from "../icons";
import { useComposerInsertStore } from "../lib/store";

/**
 * SelectionToComposer — C4 选区转对话（v3.1.1 蒸馏收尾·纯前端）。
 *
 * 办公板内选中任意正文（对话文本 / 文件预览 / 过程输出）→ 选区上方浮出
 * 「转为提问」按钮 → 点击把选中文本以引用块形式插入输入框（可继续编辑后发送）。
 * 忽略输入框/文本域/下拉/弹窗内的选区，避免干扰既有交互。
 */
export function SelectionToComposer() {
  const [pos, setPos] = useState<{ x: number; y: number } | null>(null);
  const [text, setText] = useState("");
  const btnRef = useRef<HTMLButtonElement>(null);

  const update = useCallback(() => {
    const sel = window.getSelection();
    const t = sel?.toString().trim() ?? "";
    if (!t || t.length < 2 || t.length > 4000) {
      setPos(null);
      return;
    }
    // 忽略可编辑/控件内与浮层（弹窗/下拉/命令面板）里的选区
    const node = sel?.anchorNode;
    const el = node && node.nodeType === Node.TEXT_NODE ? node.parentElement : (node as HTMLElement | null);
    if (!el || !sel || sel.rangeCount === 0) {
      setPos(null);
      return;
    }
    if (
      el.closest("input, textarea, select, [contenteditable], .ant-modal, .ant-modal-wrap, .ant-select, [data-no-quote]")
    ) {
      setPos(null);
      return;
    }
    const rect = sel.getRangeAt(0).getBoundingClientRect();
    if (rect.width === 0 && rect.height === 0) {
      setPos(null);
      return;
    }
    setText(t);
    setPos({ x: Math.min(rect.left + Math.min(rect.width / 2, 140), window.innerWidth - 130), y: rect.top });
  }, []);

  useEffect(() => {
    // mouseup 延后一拍，等选区稳定后再定位（点击浮动按钮时不被误清）
    const onUp = () => {
      window.setTimeout(update, 0);
    };
    document.addEventListener("mouseup", onUp);
    document.addEventListener("keyup", onUp);
    document.addEventListener("selectionchange", update);
    return () => {
      document.removeEventListener("mouseup", onUp);
      document.removeEventListener("keyup", onUp);
      document.removeEventListener("selectionchange", update);
    };
  }, [update]);

  const dismiss = useCallback(() => {
    setPos(null);
    setText("");
  }, []);

  const insert = useCallback(() => {
    const quoted = text
      .split("\n")
      .map((l) => `> ${l}`)
      .join("\n");
    useComposerInsertStore.getState().requestText(`${quoted}\n\n请基于以上内容继续处理。`);
    dismiss();
  }, [text, dismiss]);

  if (!pos) return null;

  return createPortal(
    <div
      className="fixed z-[1200] flex items-center gap-1 rounded-lg border border-border bg-bg-elev-2 shadow-[0_8px_24px_rgba(0,0,0,0.35)] px-1.5 py-1 anim-menu-in"
      style={{ left: pos.x, top: Math.max(8, pos.y - 38) }}
      role="toolbar"
      aria-label="选区操作"
      onMouseDown={(e) => e.preventDefault() /* 保持选区不被按钮点击清空 */}
    >
      <button
        ref={btnRef}
        type="button"
        className="inline-flex items-center gap-1 px-2 h-6 rounded-md bg-accent text-white text-[11px] hover:opacity-90 transition-opacity"
        onClick={insert}
        title="把选中文本以引用块插入输入框，可编辑后发送"
      >
        <MessageSquare size={11} /> 转为提问
      </button>
      <button
        type="button"
        className="inline-flex items-center justify-center w-6 h-6 rounded-md text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors"
        onClick={dismiss}
        title="取消"
        aria-label="取消"
      >
        <X size={11} />
      </button>
    </div>,
    document.body,
  );
}

export default SelectionToComposer;
