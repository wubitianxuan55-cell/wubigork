/**
 * xlsxpreview/FormulaBar — 自 XlsxPreview 原样搬移的公式栏（选中单元格后的
 * 第一层操作：直接输入值或 =公式，回车写回）。受控展示组件，写回经 onCommit 上抛。
 */
import { useEffect, useMemo, useState } from "react";
import type { XlsxSheet } from "../../lib/types";

export function FormulaBar({
  sheet,
  selected,
  onCommit,
  disabled,
}: {
  sheet: XlsxSheet;
  selected: string | null;
  onCommit: (ref: string, value: string) => void;
  disabled?: boolean;
}) {
  const cell = useMemo(() => {
    if (!selected) return null;
    for (const row of sheet.rows) {
      const c = row.find((x) => x.ref === selected);
      if (c) return c;
    }
    return null;
  }, [sheet, selected]);

  const [draft, setDraft] = useState("");
  useEffect(() => {
    setDraft(cell ? (cell.formula ? `=${cell.formula}` : String(cell.value ?? "")) : "");
  }, [cell]);

  return (
    <div className="flex items-center gap-2 px-3 py-1 border-b border-border-soft bg-bg-soft/30 text-[11px] shrink-0">
      <span className="font-mono text-accent w-12 shrink-0">{selected ?? "—"}</span>
      <span className="text-fg-faint shrink-0">fx</span>
      <input
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            if (selected && !disabled) onCommit(selected, draft);
          } else if (e.key === "Escape") {
            e.preventDefault();
            setDraft(cell ? (cell.formula ? `=${cell.formula}` : String(cell.value ?? "")) : "");
          }
        }}
        disabled={disabled || !selected}
        placeholder="选中单元格后输入值，或 =公式 直接写公式"
        className="flex-1 min-w-0 px-2 py-1 rounded-md border border-border-soft bg-bg text-[12px] font-mono text-fg-dim outline-none focus:border-accent/50 disabled:opacity-40"
      />
    </div>
  );
}