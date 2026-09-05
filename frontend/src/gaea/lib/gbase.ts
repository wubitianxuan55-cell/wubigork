// gbase.ts — 多维表视图层纯函数（B1，docs/gaea-office-mindmap-base-design-2026-09.md §4）。
// 架构口径：数据永远在 .xlsx（字段=列、首行=表头），`.gbase.json` sidecar 只存视图配置
// （分组/筛选/排序/着色）。本文件是单一权威：配置解析（字段级容错）、sheet→记录模型、
// 视图计算（filter/sort/groupBy/colorRules）、失配检测。全部纯函数，jsdom 可测。

import type { XlsxSheet } from "./types";

// ─── 配置 schema ────────────────────────────────────────────────

export type GbaseFilterOp =
  | "eq"
  | "ne"
  | "gt"
  | "gte"
  | "lt"
  | "lte"
  | "contains"
  | "empty"
  | "notEmpty";

const FILTER_OPS: ReadonlySet<string> = new Set<string>([
  "eq", "ne", "gt", "gte", "lt", "lte", "contains", "empty", "notEmpty",
]);

export interface GbaseFilterCondition {
  column: string;
  op: GbaseFilterOp;
  value?: string | number;
}

export interface GbaseView {
  id: string;
  name: string;
  /** v1 仅 grid（分组视图）；board/画廊列 B2。 */
  type: "grid";
  /** 绑定 sheet 名；缺省作用于当前激活 sheet。 */
  sheet?: string;
  /** 分组列名（表头文本）；缺省平铺。 */
  groupBy?: string;
  filter?: { op: "and" | "or"; conditions: GbaseFilterCondition[] };
  sort?: { column: string; dir: "asc" | "desc" }[];
  /** 行级条件着色（首条命中优先）。 */
  colorRules?: { column: string; op: GbaseFilterOp; value?: string | number; color: string }[];
}

export interface GbaseConfig {
  version: 1;
  views: GbaseView[];
}

export const GBASE_MAX_VIEWS = 24;
export const GBASE_MAX_CONDITIONS = 20;
export const GBASE_MAX_SORTS = 5;
export const GBASE_MAX_RULES = 10;

/** report.xlsx → report.gbase.json（同目录同名 sidecar）。 */
export function gbaseSidecarPath(relPath: string): string {
  const dot = relPath.lastIndexOf(".");
  const slash = relPath.lastIndexOf("/");
  const base = dot > slash && dot >= 0 ? relPath.slice(0, dot) : relPath;
  return `${base}.gbase.json`;
}

/** 解析视图配置：根形状坏 → 整体失败；字段级问题 → 丢弃该视图/条件并保留 errors 提示。 */
export function parseGbaseConfig(text: string): { config: GbaseConfig | null; error: string } {
  let raw: unknown;
  try {
    raw = JSON.parse(text);
  } catch (e) {
    return { config: null, error: `JSON 解析失败：${String(e)}` };
  }
  if (typeof raw !== "object" || raw === null || Array.isArray(raw)) {
    return { config: null, error: "根必须是对象" };
  }
  const obj = raw as Record<string, unknown>;
  if (obj.version !== 1) return { config: null, error: "version 必须为 1" };
  if (!Array.isArray(obj.views)) return { config: null, error: "views 必须是数组" };

  const views: GbaseView[] = [];
  const errors: string[] = [];
  for (let i = 0; i < obj.views.length; i++) {
    if (views.length >= GBASE_MAX_VIEWS) {
      errors.push(`视图数超过上限 ${GBASE_MAX_VIEWS}，已截断`);
      break;
    }
    const v: unknown = obj.views[i];
    if (typeof v !== "object" || v === null) {
      errors.push(`views[${i}] 非对象，已跳过`);
      continue;
    }
    const o = v as Record<string, unknown>;
    if (o.type !== undefined && o.type !== "grid") {
      errors.push(`views[${i}].type 仅支持 grid，已跳过`);
      continue;
    }
    const view: GbaseView = {
      id: typeof o.id === "string" && o.id ? o.id : `v${views.length + 1}`,
      name: typeof o.name === "string" && o.name.trim() ? o.name.trim().slice(0, 60) : `视图${views.length + 1}`,
      type: "grid",
    };
    if (typeof o.sheet === "string" && o.sheet.trim()) view.sheet = o.sheet.trim();
    if (typeof o.groupBy === "string" && o.groupBy.trim()) view.groupBy = o.groupBy.trim();

    if (typeof o.filter === "object" && o.filter !== null) {
      const f = o.filter as Record<string, unknown>;
      const op = f.op === "or" ? "or" : "and";
      const conditions: GbaseFilterCondition[] = [];
      if (Array.isArray(f.conditions)) {
        for (let j = 0; j < f.conditions.length && conditions.length < GBASE_MAX_CONDITIONS; j++) {
          const c: unknown = f.conditions[j];
          if (typeof c !== "object" || c === null) continue;
          const co = c as Record<string, unknown>;
          if (typeof co.column !== "string" || !co.column.trim()) continue;
          const cop = typeof co.op === "string" && FILTER_OPS.has(co.op) ? co.op : "eq";
          conditions.push({
            column: co.column.trim(),
            op: cop as GbaseFilterOp,
            ...(co.value === undefined ? {} : { value: co.value as string | number }),
          });
        }
        if (f.conditions.length > GBASE_MAX_CONDITIONS) errors.push(`views[${i}] 条件数超上限，已截断`);
      }
      if (conditions.length > 0) view.filter = { op, conditions };
    }

    if (Array.isArray(o.sort)) {
      const sorts: { column: string; dir: "asc" | "desc" }[] = [];
      for (let j = 0; j < o.sort.length && sorts.length < GBASE_MAX_SORTS; j++) {
        const s: unknown = o.sort[j];
        if (typeof s !== "object" || s === null) continue;
        const so = s as Record<string, unknown>;
        if (typeof so.column !== "string" || !so.column.trim()) continue;
        sorts.push({ column: so.column.trim(), dir: so.dir === "desc" ? "desc" : "asc" });
      }
      if (sorts.length > 0) view.sort = sorts;
    }

    if (Array.isArray(o.colorRules)) {
      const rules: NonNullable<GbaseView["colorRules"]> = [];
      for (let j = 0; j < o.colorRules.length && rules.length < GBASE_MAX_RULES; j++) {
        const r: unknown = o.colorRules[j];
        if (typeof r !== "object" || r === null) continue;
        const ro = r as Record<string, unknown>;
        if (typeof ro.column !== "string" || !ro.column.trim()) continue;
        if (typeof ro.color !== "string" || !/^#[0-9a-fA-F]{3,8}$/.test(ro.color)) continue;
        const rop = typeof ro.op === "string" && FILTER_OPS.has(ro.op) ? ro.op : "eq";
        rules.push({ column: ro.column.trim(), op: rop as GbaseFilterOp, color: ro.color, ...(ro.value === undefined ? {} : { value: ro.value as string | number }) });
      }
      if (rules.length > 0) view.colorRules = rules;
    }

    views.push(view);
  }
  if (views.length === 0 && errors.length === 0) return { config: null, error: "views 为空" };
  return { config: { version: 1, views }, error: errors.join("；") };
}

// ─── sheet → 记录模型 ───────────────────────────────────────────

function parseGRef(ref: string): { col: number; row: number } {
  const m = /^([A-Z]+)(\d+)$/.exec(ref);
  if (!m) return { col: 0, row: 0 };
  let col = 0;
  for (const ch of m[1]) col = col * 26 + (ch.charCodeAt(0) - 64);
  return { col, row: parseInt(m[2], 10) };
}

export interface GbaseRecord {
  /** 来源 xlsx 行号（1 起），供调试/未来联动选中。 */
  rowIndex: number;
  /** 列名（表头文本）→ 单元格文本值。 */
  cells: Record<string, string>;
}

export interface GbaseSheetModel {
  /** 表头字段名（按列序；无名表头列回落「列A」式命名）。 */
  fields: string[];
  /** 第 2 行起的数据记录（全空行跳过）。 */
  records: GbaseRecord[];
}

export function gbaseSheetModel(sheet: XlsxSheet): GbaseSheetModel {
  const fieldByCol = new Map<number, string>();
  let maxFieldCol = 0;
  for (const cell of sheet.rows[0] ?? []) {
    const p = parseGRef(cell.ref);
    if (p.row !== 1 || p.col === 0) continue;
    const name = String(cell.value ?? "").trim();
    fieldByCol.set(p.col, name || `列${String.fromCharCode(64 + p.col)}`);
    if (p.col > maxFieldCol) maxFieldCol = p.col;
  }
  const fields: string[] = [];
  for (let c = 1; c <= maxFieldCol; c++) fields.push(fieldByCol.get(c) ?? `列${String.fromCharCode(64 + c)}`);

  const records: GbaseRecord[] = [];
  for (const row of sheet.rows) {
    if (row.length === 0) continue;
    const first = parseGRef(row[0]!.ref);
    if (first.row <= 1) continue;
    const cells: Record<string, string> = {};
    let hasValue = false;
    for (const cell of row) {
      const p = parseGRef(cell.ref);
      const field = fieldByCol.get(p.col);
      if (!field) continue;
      const v = String(cell.value ?? "");
      cells[field] = v;
      if (v.trim() !== "") hasValue = true;
    }
    if (hasValue) records.push({ rowIndex: first.row, cells });
  }
  return { fields, records };
}

// ─── 视图计算 ───────────────────────────────────────────────────

/** 数值可比较数值，否则字典序（空串最小）。 */
export function gbaseCompare(a: string, b: string): number {
  const na = Number(a);
  const nb = Number(b);
  if (a !== "" && b !== "" && Number.isFinite(na) && Number.isFinite(nb)) return na - nb;
  return a < b ? -1 : a > b ? 1 : 0;
}

function gbaseEq(a: string, b: string): boolean {
  const na = Number(a);
  const nb = Number(b);
  if (a !== "" && b !== "" && Number.isFinite(na) && Number.isFinite(nb)) return na === nb;
  return a === b;
}

export function matchGbaseCondition(value: string, c: GbaseFilterCondition): boolean {
  const v = (value ?? "").trim();
  const target = c.value === undefined ? "" : String(c.value);
  switch (c.op) {
    case "empty":
      return v === "";
    case "notEmpty":
      return v !== "";
    case "eq":
      return gbaseEq(v, target);
    case "ne":
      return !gbaseEq(v, target);
    case "contains":
      return v.toLowerCase().includes(target.toLowerCase());
    case "gt":
      return gbaseCompare(v, target) > 0;
    case "gte":
      return gbaseCompare(v, target) >= 0;
    case "lt":
      return gbaseCompare(v, target) < 0;
    case "lte":
      return gbaseCompare(v, target) <= 0;
    default:
      return false;
  }
}

export interface GbaseGroup {
  /** 分组键（无 groupBy 时为 ""）；空值归入「（空）」。 */
  key: string;
  records: GbaseRecord[];
}

/** 视图主计算：filter → sort（稳定）→ groupBy（首现顺序；空值归「（空）」）。 */
export function applyGbaseView(
  model: GbaseSheetModel,
  view: GbaseView,
): { groups: GbaseGroup[]; filteredOut: number } {
  let rows = model.records;
  let filteredOut = 0;
  if (view.filter && view.filter.conditions.length > 0) {
    const before = rows.length;
    rows = rows.filter((r) => {
      const results = view.filter!.conditions.map((c) => matchGbaseCondition(r.cells[c.column] ?? "", c));
      return view.filter!.op === "or" ? results.some(Boolean) : results.every(Boolean);
    });
    filteredOut = before - rows.length;
  }
  if (view.sort && view.sort.length > 0) {
    // 多键稳定排序：先按来源行号还原稳定性，再逐键比较
    rows = rows
      .map((r, i) => ({ r, i }))
      .sort((x, y) => {
        for (const s of view.sort!) {
          const cmp = gbaseCompare((x.r.cells[s.column] ?? "").trim(), (y.r.cells[s.column] ?? "").trim());
          if (cmp !== 0) return s.dir === "desc" ? -cmp : cmp;
        }
        return x.i - y.i;
      })
      .map((x) => x.r);
  }
  if (view.groupBy && model.fields.includes(view.groupBy)) {
    const order: string[] = [];
    const map = new Map<string, GbaseRecord[]>();
    for (const r of rows) {
      const key = (r.cells[view.groupBy] ?? "").trim() || "（空）";
      if (!map.has(key)) {
        map.set(key, []);
        order.push(key);
      }
      (map.get(key) as GbaseRecord[]).push(r);
    }
    return { groups: order.map((key) => ({ key, records: map.get(key) as GbaseRecord[] })), filteredOut };
  }
  return { groups: rows.length > 0 ? [{ key: "", records: rows }] : [], filteredOut };
}

/** 视图引用的列中不存在于当前 sheet 表头的那些（去重保序）——非空则降级回表格视图。 */
export function gbaseMissingColumns(view: GbaseView, fields: string[]): string[] {
  const set = new Set(fields);
  const missing: string[] = [];
  const push = (col: string | undefined): void => {
    if (col && !set.has(col) && !missing.includes(col)) missing.push(col);
  };
  push(view.groupBy);
  for (const c of view.filter?.conditions ?? []) push(c.column);
  for (const s of view.sort ?? []) push(s.column);
  for (const r of view.colorRules ?? []) push(r.column);
  return missing;
}

/** 行级着色：首条命中的 color。 */
export function gbaseRowColor(record: GbaseRecord, view: GbaseView): string | null {
  for (const rule of view.colorRules ?? []) {
    if (matchGbaseCondition(record.cells[rule.column] ?? "", rule)) return rule.color;
  }
  return null;
}
