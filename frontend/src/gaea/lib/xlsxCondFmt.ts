// xlsxCondFmt.ts — xlsx 条件格式「CellIs 静态可判定子集」的前端渲染判定
// （办公 U4 渲染缺口 #2 前半；后端提取在 internal/office/xlsxpreview/condfmt.go，
// op 口径同源锚定）。语义边界：
//   - 只判 CellIs 比较符（大于/小于/等于/介于等），阈值必须是常量（数字或
//     带引号文本）；表达式/色阶/数据条等后端已跳过（condSkipped 计数）。
//   - 数字比较按数值；equal/notEqual 文本按 Excel 惯例大小写不敏感。
//   - 多规则命中时 priority 小者（Excel 高优先级）胜出，逐属性整体取自该条。

import type { XlsxCell, XlsxCondRule } from "./types";

/** "A1" → { col: 1, row: 1 }；非法返回 null */
export function parseCellRef(ref: string): { col: number; row: number } | null {
  const m = /^([A-Za-z]{1,3})(\d{1,7})$/.exec(ref.trim());
  if (!m) return null;
  let col = 0;
  for (const ch of m[1].toUpperCase()) col = col * 26 + (ch.charCodeAt(0) - 64);
  return { col, row: Number(m[2]) };
}

/** sqref 单段 "D2:D100" → [start, end]；单格 "D2" → [D2, D2] */
export function parseRange(range: string): { c1: number; r1: number; c2: number; r2: number } | null {
  const parts = range.split(":");
  if (parts.length === 0 || parts.length > 2) return null;
  const a = parseCellRef(parts[0]);
  const b = parts.length === 2 ? parseCellRef(parts[1]) : a;
  if (!a || !b) return null;
  return {
    c1: Math.min(a.col, b.col), r1: Math.min(a.row, b.row),
    c2: Math.max(a.col, b.col), r2: Math.max(a.row, b.row),
  };
}

/** 规则是否覆盖该单元格（range 包含判定） */
export function condRuleCovers(rule: XlsxCondRule, ref: string): boolean {
  const p = parseCellRef(ref);
  const rg = parseRange(rule.range);
  if (!p || !rg) return false;
  return p.col >= rg.c1 && p.col <= rg.c2 && p.row >= rg.r1 && p.row <= rg.r2;
}

/** 阈值原文 → { num } 或 { text }（带引号文本剥引号）；不可判定返回 null */
export function parseThreshold(raw: string): { num?: number; text?: string } | null {
  const t = raw.trim();
  if (t === "") return null;
  if (t.startsWith(`"`) && t.endsWith(`"`) && t.length >= 2) return { text: t.slice(1, -1) };
  const n = Number(t);
  if (Number.isFinite(n)) return { num: n };
  return null;
}

/** 单元格值是否命中该规则（op 语义与后端 condSupportedOps 同源） */
export function condRuleMatches(rule: XlsxCondRule, cell: XlsxCell | undefined): boolean {
  if (!cell) return false;
  const raw = String(cell.value ?? "").trim();
  if (raw === "") return false; // 空单元格不参与比较（Excel 同款语义）
  const thresholds = rule.formulas.map(parseThreshold);
  if (thresholds.some((t) => t === null)) return false;
  const n = Number(raw.replace(/,/g, "")); // 容忍千分位显示值
  const isNum = Number.isFinite(n) && thresholds.every((t) => t!.num !== undefined);
  if (isNum) {
    const v = n;
    const a = thresholds[0]!.num!;
    const b = thresholds.length > 1 ? thresholds[1]!.num! : 0;
    const lo = Math.min(a, b), hi = Math.max(a, b);
    switch (rule.op) {
      case "greaterThan": return v > a;
      case "lessThan": return v < a;
      case "greaterThanOrEqual": return v >= a;
      case "lessThanOrEqual": return v <= a;
      case "equal": return v === a;
      case "notEqual": return v !== a;
      case "between": return v >= lo && v <= hi;
      case "notBetween": return v < lo || v > hi;
      default: return false;
    }
  }
  // 文本路径：仅 equal/notEqual 有定义（大小写不敏感，Excel 惯例）
  const th = thresholds[0]!;
  if (th.num !== undefined || th.text === undefined) return false;
  const lv = raw.toLowerCase();
  const lt = th.text.toLowerCase();
  return rule.op === "equal" ? lv === lt : rule.op === "notEqual" ? lv !== lt : false;
}

export interface CondStyle {
  fill?: string;
  fontColor?: string;
  bold?: boolean;
}

/** 该单元格的条件格式样式（priority 小者=高优先级整体胜出；未命中返回 null） */
export function condStyleFor(
  rules: XlsxCondRule[] | undefined,
  ref: string,
  cell: XlsxCell | undefined,
): CondStyle | null {
  if (!rules || rules.length === 0 || !cell) return null;
  const sorted = [...rules].sort((a, b) => (a.priority ?? 0) - (b.priority ?? 0));
  for (const rule of sorted) {
    if (!condRuleCovers(rule, ref)) continue;
    if (!condRuleMatches(rule, cell)) continue;
    return { fill: rule.fill, fontColor: rule.fontColor, bold: rule.bold };
  }
  return null;
}
