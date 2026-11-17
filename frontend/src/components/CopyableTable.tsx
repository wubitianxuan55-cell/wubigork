import { useCallback, useEffect, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { copyToClipboard } from "../gaea/lib/clipboard";
import { sanitizeHtml } from "../gaea/lib/sanitize";
import { snapshotTable, tableToMarkdown } from "../gaea/lib/copyMarkdown";

type CopyState = "idle" | "rich" | "plain" | "error";

export function CopyableTable({
  children,
  containerClassName,
  containerStyle,
  tableStyle,
}: {
  children: ReactNode;
  containerClassName?: string;
  containerStyle?: CSSProperties;
  tableStyle?: CSSProperties;
}) {
  const tableRef = useRef<HTMLTableElement>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [state, setState] = useState<CopyState>("idle");

  useEffect(() => () => {
    if (timerRef.current) clearTimeout(timerRef.current);
  }, []);

  const copy = useCallback(async () => {
    const table = tableRef.current;
    const snapshot = table ? snapshotTable(table) : null;
    if (!table || !snapshot) {
      setState("error");
      return;
    }
    const result = await copyToClipboard(tableToMarkdown(snapshot), {
      html: sanitizeHtml(table.outerHTML),
    });
    setState(result === "rich" ? "rich" : result === "plain" ? "plain" : "error");
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => setState("idle"), 1800);
  }, []);

  const label = state === "rich"
    ? "已复制"
    : state === "plain"
      ? "已复制为纯文本"
      : state === "error"
        ? "复制失败"
        : "复制表格";

  return (
    <div className={containerClassName} style={{ position: "relative", ...containerStyle }}>
      <button
        type="button"
        data-testid="table-copy"
        onClick={() => void copy()}
        title="复制为 GFM 表格；支持时同时写入富文本表示"
        style={{
          position: "absolute", top: 6, right: 6, zIndex: 2,
          border: "1px solid var(--md-sys-color-outline-variant)",
          borderRadius: 6, padding: "2px 7px",
          background: "var(--md-sys-color-surface-container-high, #1e293b)",
          color: "var(--md-sys-color-on-surface-variant, #cbd5e1)",
          fontSize: 10.5, cursor: "pointer",
        }}
      >
        {label}
      </button>
      <table ref={tableRef} style={tableStyle}>{children}</table>
    </div>
  );
}
