// xlsxCellDiff.ts — xlsx 版本结构化对比（v4.28 A2「线 C」交付验收闭环）纯函数层。
//
// Why: 版本时间线的「与当前对比」此前对 .xlsx 只能给 kind:"unsupported" 降级
// （双版本并排预览）。本模块把两份 xlsx 的结构化单元格预览（后端 GaeaPreview
// xlsx 分支 → PreviewResult.body JSON，即 lib/types 的 XlsxSheet/XlsxCell）
// 对齐出 sheet 级 + 单元格级差异，收掉这笔记账。
//
// How: 纯函数、无 React 依赖、不触碰 bridge——两侧取数（app.Preview）与 async
// 编排在 versionCompare.ts，这里只做给定两组 XlsxSheet[] 的差异计算：
//   1) sheet 按名字对齐：只在当前侧存在 → 新增 sheet；只在基线侧 → 删除 sheet
//      （如实整表列出，不展开逐格——新表整张都是「新增」，逐格罗列全是噪音）；
//   2) 同名 sheet 内单元格按 ref（"A1"）对齐：仅当前侧有 → add，仅基线侧有 →
//      del，两侧都有但取值不同 → change（含旧值新值）；
//   3) 比较口径：value 直接字符串全等比较；formula 串也当文本如实比较（不为
//      公式做求值/重算语义——前端没有求值器，且版本对比关心的是「写进了什么」）；
//   4) 输出按行号→列号排序（A1、B2 的自然阅读序），变更列表超 maxPerSheet 截断
//      并标记 truncated（沿用 clampDiffRows 的「只展示前 N 条 + 计数不失真」风格：
//      add/del/change 计数永远按截断前全量统计）。

import type { XlsxCell, XlsxSheet } from "./types";

/** 单个 sheet 内一条单元格级变更（add/del/change，change 含旧值新值）。 */
export interface XlsxCellChangeRow {
  kind: "add" | "del" | "change";
  /** 单元格引用（"A1"），两侧同名 sheet 内直接可比。 */
  ref: string;
  /** 基线侧值（add 行为空串，渲染层显示占位）。 */
  old: string;
  /** 当前侧值（del 行为空串）。 */
  new: string;
  /** 当前侧为公式时给出公式串（不含前导"="，展示 fx 徽标用），与 XlsxCellChange 惯例一致。 */
  formula?: string;
}

/** sheet 级状态：changed = 同名但有单元格差异；add/del = 整表新增/删除。 */
export type XlsxSheetDiffState = "changed" | "add" | "del";

export interface XlsxSheetDiff {
  name: string;
  state: XlsxSheetDiffState;
  /** 单元格级变更列表（changed 时有值；add/del sheet 恒为空列表——整表级差异不再逐格罗列）。 */
  cells: XlsxCellChangeRow[];
  /** 新增单元格数（截断前全量）。 */
  add: number;
  /** 删除单元格数（截断前全量）。 */
  del: number;
  /** 修改单元格数（截断前全量）。 */
  change: number;
  /** 变更总条数（截断前 = add+del+change，冗余字段省得渲染层再加一遍）。 */
  total: number;
  /** 变更列表超上限被截断（cells 只保留前 maxPerSheet 条，计数不失真）。 */
  truncated: boolean;
}

/** 每 sheet 变更列表展示上限：与 VersionTimeline 的 MAX_DIFF_ROWS(200) 同一数量级口径。 */
export const MAX_XLSX_SHEET_CELLS = 200;

// ref → (行号, 列号)，供变更列表按自然阅读序输出；非法 ref（理论不出现）排最后。
function refOrder(ref: string): [number, number] {
  const m = /^([A-Z]+)(\d+)$/.exec(ref);
  if (!m) return [Number.MAX_SAFE_INTEGER, Number.MAX_SAFE_INTEGER];
  let col = 0;
  for (const ch of m[1]) col = col * 26 + (ch.charCodeAt(0) - 64);
  return [parseInt(m[2], 10), col];
}

// rows[][] 展平为 ref → cell 映射（同 ref 后写覆盖先写，理论不出现重复）。
function cellMapOf(sheet: XlsxSheet): Map<string, XlsxCell> {
  const map = new Map<string, XlsxCell>();
  for (const row of sheet.rows) {
    for (const cell of row) map.set(cell.ref, cell);
  }
  return map;
}

// 取值口径：value 直接当字符串；formula 缺省记空串（参与相等判定）。
function valueOf(cell: XlsxCell | undefined): string {
  return cell?.value ?? "";
}
function formulaOf(cell: XlsxCell | undefined): string {
  return cell?.formula ?? "";
}

/**
 * 纯函数：两份 xlsx 的 sheet 结构 → sheet 级 + 单元格级差异。
 * 只输出有差异的 sheet（同名且无单元格差异的不进结果——「无差异」由上层
 * add/del/change 全 0 判定，与文本 diff 的空行集合同口径）。任何一侧给出空
 * 数组即视为该侧整体为空，照常产出差异（宁漏勿误的对称面：内容缺失由
 * versionCompare 的 contentMissing 标注，这里不做隐式吞并）。
 */
export function diffXlsxSheets(
  base: XlsxSheet[],
  cur: XlsxSheet[],
  maxPerSheet: number = MAX_XLSX_SHEET_CELLS,
): XlsxSheetDiff[] {
  const baseByName = new Map<string, XlsxSheet>();
  for (const s of base) baseByName.set(s.name, s);
  const seenCur = new Set<string>();

  const out: XlsxSheetDiff[] = [];
  // 先按当前侧顺序走：同名 sheet 逐格对齐，新增 sheet 整表标记。
  for (const s of cur) {
    seenCur.add(s.name);
    const prev = baseByName.get(s.name);
    if (!prev) {
      out.push({
        name: s.name,
        state: "add",
        cells: [],
        add: 0,
        del: 0,
        change: 0,
        total: 0,
        truncated: false,
      });
      continue;
    }
    const diff = diffOneSheet(prev, s, maxPerSheet);
    if (diff) out.push(diff);
  }
  // 再补基线侧独有（被删除）的 sheet，保持基线内相对顺序。
  for (const s of base) {
    if (!seenCur.has(s.name)) {
      out.push({
        name: s.name,
        state: "del",
        cells: [],
        add: 0,
        del: 0,
        change: 0,
        total: 0,
        truncated: false,
      });
    }
  }
  return out;
}

// 同名 sheet 逐格对齐；无差异返回 null（调用方跳过该 sheet）。
function diffOneSheet(prev: XlsxSheet, cur: XlsxSheet, maxPerSheet: number): XlsxSheetDiff | null {
  const prevCells = cellMapOf(prev);
  const curCells = cellMapOf(cur);
  const rows: XlsxCellChangeRow[] = [];
  let add = 0;
  let del = 0;
  let change = 0;

  for (const [ref, curCell] of curCells) {
    const prevCell = prevCells.get(ref);
    if (!prevCell) {
      add++;
      rows.push({ kind: "add", ref, old: "", new: valueOf(curCell), formula: curCell.formula });
      continue;
    }
    // 值与公式串都当文本如实比较：任一不同即 change（不做求值语义）。
    if (valueOf(prevCell) !== valueOf(curCell) || formulaOf(prevCell) !== formulaOf(curCell)) {
      change++;
      rows.push({
        kind: "change",
        ref,
        old: valueOf(prevCell),
        new: valueOf(curCell),
        formula: curCell.formula,
      });
    }
  }
  for (const [ref, prevCell] of prevCells) {
    if (!curCells.has(ref)) {
      del++;
      rows.push({ kind: "del", ref, old: valueOf(prevCell), new: "" });
    }
  }

  if (rows.length === 0) return null;
  rows.sort((a, b) => {
    const [ra, ca] = refOrder(a.ref);
    const [rb, cb] = refOrder(b.ref);
    return ra !== rb ? ra - rb : ca - cb;
  });
  const total = rows.length;
  const truncated = total > maxPerSheet;
  return {
    name: cur.name,
    state: "changed",
    cells: truncated ? rows.slice(0, maxPerSheet) : rows,
    add,
    del,
    change,
    total,
    truncated,
  };
}
