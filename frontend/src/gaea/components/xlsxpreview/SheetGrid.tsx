/**
 * xlsxpreview/SheetGrid — 自 XlsxPreview 原样搬移的表格网格（行虚拟滚动 / 冻结行
 * spacer / 合并单元格 / 条件格式 / 直接编辑）。虚拟滚动逻辑、行窗口计算、冻结行
 * spacer 语义与拆分前逐位等价（大表阈值 300 行、overscan 10、TH_H=26 / ROW_H=28 不变）。
 */
import { useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties } from "react";
import { Table } from "../../icons";
import type { XlsxCell, XlsxSheet } from "../../lib/types";
import { condStyleFor } from "../../lib/xlsxCondFmt";
import { colToLetter, formatCellValue, parseRef } from "./numFmt";

export function SheetGrid({
  sheet,
  selected,
  onSelect,
  selectedCol,
  onSelectCol,
  editingRef,
  draft,
  onDraftChange,
  onEditCell,
  onCommit,
  onCancelEdit,
  disabled,
}: {
  sheet: XlsxSheet;
  selected: string | null;
  onSelect: (ref: string) => void;
  selectedCol: string | null;
  onSelectCol: (letter: string) => void;
  editingRef: string | null;
  draft: string;
  onDraftChange: (v: string) => void;
  onEditCell: (ref: string, initial: string) => void;
  onCommit: (ref: string, value: string) => void;
  onCancelEdit: () => void;
  disabled?: boolean;
}) {
  const { maxRow, maxCol, cellMap, mergeTopLeft, mergeContinuations } = useMemo(() => {
    const map = new Map<string, XlsxCell>();
    let maxRow = 0;
    let maxCol = 0;
    for (const row of sheet.rows) {
      for (const cell of row) {
        const { col, row: r } = parseRef(cell.ref);
        map.set(cell.ref, cell);
        if (r > maxRow) maxRow = r;
        if (col > maxCol) maxCol = col;
      }
    }
    const topLeft = new Map<string, { colspan: number; rowspan: number }>();
    const continuations = new Set<string>();
    for (const m of sheet.merged ?? []) {
      const [a, b] = m.split(":");
      if (!a || !b) continue;
      const p1 = parseRef(a);
      const p2 = parseRef(b);
      const colspan = Math.max(1, p2.col - p1.col + 1);
      const rowspan = Math.max(1, p2.row - p1.row + 1);
      topLeft.set(a, { colspan, rowspan });
      for (let r = p1.row; r <= p2.row; r++) {
        for (let c = p1.col; c <= p2.col; c++) {
          const ref = colToLetter(c) + r;
          if (ref !== a) continuations.add(ref);
        }
      }
    }
    return { maxRow, maxCol, cellMap: map, mergeTopLeft: topLeft, mergeContinuations: continuations };
  }, [sheet]);

  // 冻结窗格：只冻结顶部行（最常见场景），固定行高保证 sticky 偏移对齐
  const freezeRow = Math.max(0, sheet.freeze?.row ?? 0);
  const TH_H = 26;
  const ROW_H = 28;

  // —— 大表格行虚拟滚动（观察项收账）——
  // 后端预览上限 2000 行 × 100 列：全量渲染最多 20 万 td，滚动/切换卡顿。
  // 行数超阈值时只渲染可见窗口 ± overscan（冻结行常驻），spacer 行保持滚动
  // 条总高与行号对齐；小表全量渲染，行为逐字节不变。wrap 单元格在虚拟化
  // 模式下按固定行高裁剪（溢出隐藏），与冻结行既有的 ROW_H 对齐假设一致。
  const VIRTUALIZE_MIN_ROWS = 300;
  const OVERSCAN_ROWS = 10;
  const containerRef = useRef<HTMLDivElement>(null);
  const [scrollTop, setScrollTop] = useState(0);
  const [viewportH, setViewportH] = useState(0);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const measure = () => setViewportH(el.clientHeight);
    measure();
    if (typeof ResizeObserver !== "undefined") {
      const ro = new ResizeObserver(measure);
      ro.observe(el);
      return () => ro.disconnect();
    }
    return undefined;
  }, []);

  // 切换 sheet 重置滚动位置（不同表总高不同，防残留偏移）
  useEffect(() => {
    setScrollTop(0);
    const el = containerRef.current;
    if (el) el.scrollTop = 0;
  }, [sheet]);

  const virtualize = maxRow > VIRTUALIZE_MIN_ROWS;
  const frozen = Math.min(freezeRow, maxRow);
  const renderedRows: number[] = [];
  let topPad = 0;
  let bottomPad = 0;
  if (virtualize) {
    for (let r = 1; r <= frozen; r++) renderedRows.push(r);
    const firstRow = Math.max(1, Math.floor((scrollTop - TH_H) / ROW_H) - OVERSCAN_ROWS);
    const lastRow = Math.min(maxRow, Math.ceil((scrollTop - TH_H + viewportH) / ROW_H) + OVERSCAN_ROWS);
    const start = Math.max(frozen + 1, firstRow);
    const end = Math.max(start - 1, lastRow);
    for (let r = start; r <= end; r++) renderedRows.push(r);
    topPad = Math.min(Math.max(0, start - frozen - 1), Math.max(0, maxRow - frozen));
    bottomPad = Math.max(0, maxRow - end);
  } else {
    for (let r = 1; r <= maxRow; r++) renderedRows.push(r);
  }

  if (maxRow === 0 || maxCol === 0) {
    return <div className="p-6 text-center text-[12px] text-fg-faint">（空工作表）</div>;
  }

  const headerCells: string[] = [];
  for (let c = 1; c <= maxCol; c++) headerCells.push(colToLetter(c));

  return (
    <div
      ref={containerRef}
      onScroll={(e) => setScrollTop(e.currentTarget.scrollTop)}
      className="p-3 h-full overflow-auto docx-preview-body"
    >
      <table
        className="border-collapse text-[12px] leading-tight"
        style={{ borderSpacing: 0 }}
      >
        <thead>
          <tr>
            <th
              className="sticky top-0 left-0 z-30 w-8 min-w-[32px] px-1 py-1 text-center text-[10px] text-fg-faint font-normal"
              style={{ background: "var(--bg-elevated, #181b21)", border: "1px solid rgba(128,128,140,0.16)", height: TH_H }}
            />
            {headerCells.map((l) => (
              <th
                key={l}
                onClick={() => onSelectCol(l)}
                className={`sticky top-0 z-20 px-2 py-1 text-center text-[10px] font-normal cursor-pointer transition-colors ${
                  selectedCol === l ? "text-accent" : "text-fg-faint hover:text-fg"
                }`}
                style={{
                  background: selectedCol === l ? "var(--accent-soft, rgba(99,102,241,0.12))" : "var(--bg-elevated, #181b21)",
                  border: "1px solid rgba(128,128,140,0.16)",
                  minWidth: 56,
                  height: TH_H,
                }}
                title="点击选中列（可插列/删列）"
              >
                {l}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {topPad > 0 && (
            <tr aria-hidden="true" style={{ height: topPad * ROW_H }}>
              <td colSpan={maxCol + 1} style={{ border: "none", padding: 0 }} />
            </tr>
          )}
          {renderedRows.map((r) => (
            <tr key={r} style={virtualize ? { height: ROW_H } : undefined}>
              <td
                className="sticky left-0 z-10 px-1 py-0.5 text-center text-[10px] text-fg-faint"
                style={{
                  background: "var(--bg-elevated, #181b21)",
                  border: "1px solid rgba(128,128,140,0.16)",
                  ...(freezeRow > 0 && r <= freezeRow
                    ? { top: TH_H + (r - 1) * ROW_H, zIndex: 25, height: ROW_H, overflow: "hidden" as const }
                    : {}),
                  ...(virtualize && r > freezeRow ? { overflow: "hidden" as const } : {}),
                }}
              >
                {r}
              </td>
              {headerCells.map((letter) => {
                const ref = letter + r;
                if (mergeContinuations.has(ref)) return null;
                const cell = cellMap.get(ref);
                const merge = mergeTopLeft.get(ref);
                const isEditing = editingRef === ref;
                const isFrozenRow = freezeRow > 0 && r <= freezeRow;
                const isColSelected = selectedCol === letter;
                // 条件格式命中：fill/fontColor/bold 整体覆盖基础样式（Excel 语义）
                const cond = condStyleFor(sheet.condRules, ref, cell);
                const displayBackground = cond?.fill
                  ? `#${cond.fill}`
                  : cell?.style?.fill
                    ? `#${cell.style.fill}`
                    : isFrozenRow
                      ? "var(--bg)"
                      : isColSelected
                        ? "var(--accent-soft, rgba(99,102,241,0.06))"
                        : undefined;
                return (
                  <td
                    key={ref}
                    colSpan={merge?.colspan}
                    rowSpan={merge?.rowspan}
                    onClick={() => { if (!isEditing) onSelect(ref); }}
                    onDoubleClick={(e) => {
                      e.stopPropagation();
                      if (!disabled && !isEditing) {
                        onEditCell(ref, cell?.formula ? `=${cell.formula}` : String(cell?.value ?? ""));
                      }
                    }}
                    className={`relative px-2 py-1 text-fg ${isEditing ? "" : "cursor-cell select-none"} ${
                      selected === ref ? "outline outline-2 outline-accent" : ""
                    } ${isFrozenRow ? "sticky" : ""}`}
                    style={{
                      border: "1px solid rgba(128,128,140,0.16)",
                      minWidth: sheet.colWidths?.[letter] ? sheet.colWidths[letter] * 7 : 56,
                      ...(isFrozenRow
                        ? { top: TH_H + (r - 1) * ROW_H, zIndex: 12, height: ROW_H, overflow: "hidden" as const }
                        : {}),
                      fontWeight: cond?.bold || cell?.style?.bold ? 600 : undefined,
                      fontStyle: cell?.style?.italic ? "italic" : undefined,
                      textDecoration: cell?.style?.underline
                        ? "underline"
                        : cell?.style?.strike
                          ? "line-through"
                          : undefined,
                      color: cond?.fontColor
                        ? `#${cond.fontColor}`
                        : cell?.style?.fontColor
                          ? `#${cell.style.fontColor}`
                          : undefined,
                      background: displayBackground,
                      textAlign: (cell?.style?.align as CSSProperties["textAlign"]) ?? undefined,
                      whiteSpace: isFrozenRow ? "nowrap" : cell?.style?.wrap ? "pre-wrap" : "nowrap",
                      fontVariantNumeric: "tabular-nums",
                      ...(virtualize && !isFrozenRow ? { overflow: "hidden" as const } : {}),
                    }}
                    title={cell?.formula ? `=${cell.formula}` : undefined}
                  >
                    {isEditing ? (
                      <input
                        autoFocus
                        value={draft}
                        onChange={(e) => onDraftChange(e.target.value)}
                        onKeyDown={(e) => {
                          e.stopPropagation();
                          if (e.key === "Enter") {
                            e.preventDefault();
                            onCommit(ref, draft);
                          } else if (e.key === "Escape") {
                            e.preventDefault();
                            onCancelEdit();
                          }
                        }}
                        onBlur={onCancelEdit}
                        onClick={(e) => e.stopPropagation()}
                        onDoubleClick={(e) => e.stopPropagation()}
                        className="w-full min-w-[72px] bg-bg text-fg font-mono text-[12px] outline-none ring-2 ring-accent rounded-sm px-1"
                      />
                    ) : (
                      <>
                        {cell?.formula && (
                          <span className="absolute top-0 right-0.5 text-[8px] font-semibold text-accent/80 pointer-events-none">
                            fx
                          </span>
                        )}
                        <span className={cell?.type === "error" ? "text-err" : ""}>{formatCellValue(cell)}</span>
                      </>
                    )}
                  </td>
                );
              })}
            </tr>
          ))}
          {bottomPad > 0 && (
            <tr aria-hidden="true" style={{ height: bottomPad * ROW_H }}>
              <td colSpan={maxCol + 1} style={{ border: "none", padding: 0 }} />
            </tr>
          )}
        </tbody>
      </table>
      <div className="flex items-center gap-1.5 px-1 py-2 text-[10px] text-fg-faint">
        <Table size={11} />
        双击单元格直接编辑；或在 fx 栏输入值（= 开头写公式），回车保存到文件
      </div>
    </div>
  );
}