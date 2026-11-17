export type TableAlignment = "---" | ":--" | "--:" | ":-:";

export interface TableSnapshot {
  rows: string[][];
  align: TableAlignment[];
}

export function markdownTextAlign(value: string | undefined): "left" | "center" | "right" | undefined {
  if (value === "left" || value === "center" || value === "right") return value;
  return undefined;
}

function cellText(cell: HTMLTableCellElement): string {
  return (cell.textContent ?? "").replace(/\r?\n/g, " ").replace(/\s+/g, " ").trim();
}

function alignmentOf(cell: HTMLTableCellElement | undefined): TableAlignment {
  if (!cell) return "---";
  const value = (cell.getAttribute("align") ?? cell.style.textAlign ?? "").trim().toLowerCase();
  if (value === "left" || value === "start") return ":--";
  if (value === "right" || value === "end") return "--:";
  if (value === "center") return ":-:";
  return "---";
}

/** Capture the logical table source before visual wrapping/truncation. */
export function snapshotTable(table: HTMLTableElement): TableSnapshot | null {
  const domRows = Array.from(table.rows);
  if (domRows.length === 0) return null;
  const rows = domRows.map((row) => Array.from(row.cells).map(cellText));
  const width = Math.max(0, ...rows.map((row) => row.length));
  if (width === 0) return null;
  const normalized = rows.map((row) => [...row, ...Array.from({ length: width - row.length }, () => "")]);
  const headerCells = Array.from(domRows[0].cells);
  const align = Array.from({ length: width }, (_, i) => alignmentOf(headerCells[i]));
  return { rows: normalized, align };
}

export function escapeMarkdownCell(value: string): string {
  return value
    .replace(/\r?\n/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .replace(/\\/g, "\\\\")
    .replace(/\|/g, "\\|");
}

export function tableToMarkdown(snapshot: TableSnapshot): string {
  if (snapshot.rows.length === 0) return "";
  const width = Math.max(snapshot.rows[0]?.length ?? 0, ...snapshot.rows.map((row) => row.length));
  if (width === 0) return "";
  const align = Array.from({ length: width }, (_, i) => snapshot.align[i] ?? "---");
  const formatRow = (row: string[]): string => {
    const cells = Array.from({ length: width }, (_, i) => escapeMarkdownCell(row[i] ?? ""));
    return `| ${cells.join(" | ")} |`;
  };
  const [header, ...body] = snapshot.rows;
  return [formatRow(header), `|${align.join("|")}|`, ...body.map(formatRow)].join("\n");
}
