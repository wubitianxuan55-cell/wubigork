// verifyDiff.ts — 证据链「声明↔实况」前端近似比对（纯函数，零绑定依赖）。
//
// v4.8 Verifier 产品化：把 xlsx_apply 证据卡的 opsJson（AI 原始操作集，对齐
// internal/office/xlsxedit.Op schema）与 GaeaPreview 当前实况逐格比对，产出
// VerifyDiffRow 供证据卡第 1 层渲染；describeOp/opImpact/opBatchCount 供第 2 层
// 操作回放时间线。仅做可解释的快速可视化——权威结论以 Verifier 双通道复核
// （VerifyRecord）为准，本模块不替代复核。
import type { VerifyDiffRow, XlsxOpView } from "./types";

// ── 声明提取 ────────────────────────────────────────────────

// 解析 opsJson（JSON 数组或 {ops:[...]}）→ 操作视图；失败/缺省返回 []。
export function parseOps(opsJson?: string): XlsxOpView[] {
  if (!opsJson) return [];
  let raw: unknown;
  try {
    raw = JSON.parse(opsJson) as unknown;
  } catch {
    return [];
  }
  const arr = Array.isArray(raw)
    ? raw
    : raw && typeof raw === "object" && Array.isArray((raw as { ops?: unknown }).ops)
      ? (raw as { ops: unknown[] }).ops
      : null;
  if (!arr) return [];
  return arr.filter(
    (o): o is XlsxOpView =>
      !!o && typeof o === "object" && typeof (o as XlsxOpView).type === "string",
  );
}

// 属于可逐格比对的声明类 op（set_value/set_formula 单格写入；replace 只能标注跳过）。
export function isClaimableOp(op: XlsxOpView): boolean {
  return op.type === "set_value" || op.type === "set_formula" || op.type === "replace";
}

// ── 实况索引（GaeaPreview xlsx 预览 JSON → sheet!cell → 值/公式）──

export interface CellActual {
  value?: string;
  formula?: string;
}

export function cellKey(sheet: string, ref: string): string {
  return `${sheet.trim()}!${ref.toUpperCase()}`;
}

export function buildCellIndex(body?: string): Record<string, CellActual> {
  const idx: Record<string, CellActual> = {};
  if (!body) return idx;
  let data: { sheets?: { name?: string; rows?: { ref?: string; value?: unknown; formula?: string }[][] }[] };
  try {
    data = JSON.parse(body) as typeof data;
  } catch {
    return idx;
  }
  for (const s of data.sheets ?? []) {
    const sheetName = String(s?.name ?? "").trim();
    if (!sheetName) continue;
    for (const row of s?.rows ?? []) {
      for (const cell of row ?? []) {
        if (!cell || typeof cell !== "object" || !cell.ref) continue;
        idx[cellKey(sheetName, cell.ref)] = {
          value: cell.value == null ? undefined : String(cell.value),
          formula: cell.formula ? String(cell.formula) : undefined,
        };
      }
    }
  }
  return idx;
}

// ── 归一化比对 ───────────────────────────────────────────────

// 公式归一化：去前导 "=" 与首尾空白（与 xlsxedit.applyOne TrimPrefix 同语义）。
export function normFormula(f: string): string {
  return f.trim().replace(/^=/, "").trim();
}

function toNum(s: string): number | null {
  const n = Number(s.trim());
  return Number.isFinite(n) ? n : null;
}

// 逐格比对：数值容差 1e-9 / 字符串去空白 / 公式去前导 = 与空白；
// 实况缺失（预览无该格/无公式文本）→ skip（前端无法核对）。
export function compareCell(
  claimed: string,
  actual?: string,
  actualFormula?: string,
  isFormula = false,
): VerifyDiffRow["ok"] {
  if (isFormula) {
    if (actualFormula == null) return "skip";
    return normFormula(claimed) === normFormula(actualFormula) ? "match" : "mismatch";
  }
  if (actual == null) return "skip";
  const a = toNum(claimed);
  const b = toNum(actual);
  if (a !== null && b !== null) return Math.abs(a - b) <= 1e-9 ? "match" : "mismatch";
  return claimed.trim() === actual.trim() ? "match" : "mismatch";
}

// ── 组装：ops + 实况索引 → diff 行 ───────────────────────────

export function buildVerifyDiff(
  ops: XlsxOpView[],
  index: Record<string, CellActual>,
): VerifyDiffRow[] {
  const rows: VerifyDiffRow[] = [];
  for (const op of ops) {
    const sheet = op.sheet ?? "";
    if (!sheet) continue;
    if (op.type === "set_value") {
      if (!op.target) continue;
      const claimed = String(op.value ?? "");
      const key = cellKey(sheet, op.target);
      rows.push({
        sheet,
        cell: op.target.toUpperCase(),
        claimed,
        actual: index[key]?.value ?? "（无）",
        ok: compareCell(claimed, index[key]?.value, index[key]?.formula, false),
      });
    } else if (op.type === "set_formula") {
      if (!op.target) continue;
      const raw = String(op.formula ?? "");
      const claimed = `fx =${normFormula(raw)}`;
      const actualFormula = index[cellKey(sheet, op.target)]?.formula;
      rows.push({
        sheet,
        cell: op.target.toUpperCase(),
        claimed,
        actual: actualFormula != null ? `fx =${normFormula(actualFormula)}` : (index[cellKey(sheet, op.target)]?.value ?? "（无）"),
        ok: compareCell(raw, index[cellKey(sheet, op.target)]?.value, actualFormula, true),
      });
    } else if (op.type === "replace" && op.range) {
      // 替换是范围批量 op：命中格数依赖原值，前端无法从声明推导 → 单行跳过标注。
      rows.push({
        sheet,
        cell: op.range.toUpperCase(),
        claimed: `${op.find ?? ""} → ${op.replace ?? ""}`,
        actual: "（批量，逐格核对以复核为准）",
        ok: "skip",
      });
    }
  }
  return rows;
}

// ── 第 2 层：操作回放描述（对齐 xlsxedit.applyOne desc 风格）──

export function describeOp(op: XlsxOpView): string {
  switch (op.type) {
    case "set_value": return `写入值 ${op.target}=${String(op.value ?? "")}`;
    case "set_formula": return `写入公式 ${op.target}=${op.formula ?? ""}`;
    case "fill_range": return `填充 ${op.range ?? ""} = ${String(op.value ?? "")}`;
    case "transform": return `逐行公式 ${op.range ?? ""}`;
    case "replace": return `替换 ${op.range ?? ""}：${op.find ?? ""} → ${op.replace ?? ""}`;
    case "split_column": return `拆分 ${op.col ?? ""} 列`;
    case "clean": return `清洗 ${op.range ?? ""}`;
    case "set_style": return `设置样式 ${op.target ?? op.range ?? ""}`;
    case "merge_cells": return `合并 ${op.range ?? ""}`;
    case "unmerge_cells": return `取消合并 ${op.range ?? ""}`;
    case "set_col_width": return `列宽 ${op.col ?? ""} = ${op.width ?? ""}`;
    default: return `未知操作 ${op.type}`;
  }
}

// 影响区域（sheet!target / sheet!range / sheet!col 列）。
export function opImpact(op: XlsxOpView): string {
  const sheet = op.sheet ?? "";
  const loc = op.target ?? op.range ?? (op.col ? `${op.col} 列` : "");
  return sheet && loc ? `${sheet}!${loc}` : loc || sheet;
}

// ── 批量 op 折叠计数徽标（fill_range/transform 等只展示单行+计数）──

const RANGE_RE = /^([A-Za-z]+)(\d+)(?::([A-Za-z]+)(\d+))?$/;

export function colToNum(col: string): number {
  let n = 0;
  for (const ch of col.toUpperCase()) n = n * 26 + (ch.charCodeAt(0) - 64);
  return n;
}

function rangeRows(range?: string): number | null {
  const m = RANGE_RE.exec((range ?? "").trim());
  if (!m) return null;
  const r1 = parseInt(m[2], 10);
  const r2 = m[4] ? parseInt(m[4], 10) : r1;
  return r1 < 1 || r2 < 1 ? null : r2 - r1 + 1;
}

function rangeCellsCount(range?: string): number | null {
  const m = RANGE_RE.exec((range ?? "").trim());
  if (!m) return null;
  const c1 = colToNum(m[1]);
  const r1 = parseInt(m[2], 10);
  const c2 = m[3] ? colToNum(m[3]) : c1;
  const r2 = m[4] ? parseInt(m[4], 10) : r1;
  if (!c1 || !c2 || r1 < 1 || r2 < 1 || c1 > c2 || r1 > r2) return null;
  return (r2 - r1 + 1) * (c2 - c1 + 1);
}

// 批量 op 的折叠计数（null = 无计数）；replace 命中格数依赖原值，不给计数。
export function opBatchCount(op: XlsxOpView): string | null {
  switch (op.type) {
    case "fill_range":
    case "clean": {
      const n = rangeCellsCount(op.range);
      return n != null ? `${n} 格` : null;
    }
    case "transform": {
      const n = rangeRows(op.range);
      return n != null ? `${n} 行` : null;
    }
    case "split_column": return op.newCols?.length ? `${op.newCols.length} 列` : null;
    default: return null;
  }
}
