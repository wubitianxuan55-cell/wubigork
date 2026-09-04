// versionCompare — 产物版本间对比数据层（A1 交付验收闭环，调研 A#4/O#2）。
//
// 数据源：证据链基线快照（JournalChangeRecord.baselinePath，工作区相对路径，
// 与预览/回滚同源）与当前文件本体。
//   - 文本类文件经 ReadFile 读全文 → diffLines 行级差异；
//   - .docx 经 Preview 拿原始文件 dataUrl → JSZip+DOMParser 提取段落序列
//     （复用 docxText 套路）→ 段级 LCS diff（docxTextDiff）；
//   - .xlsx 经 Preview 拿结构化单元格 JSON（PreviewResult.body → XlsxSheet[]，
//     与 XlsxPreview 组件同一数据面）→ sheet 名对齐 + 单元格 ref 对齐差异
//     （xlsxCellDiff）；
//   - 其余类型（pdf/图片等）无法结构化对比 → kind: "unsupported"，UI 降级为
//     「双版本并排预览」入口。
//
// 降级口径（宁漏勿误，全程不抛错）：
//   - 读取/取数失败：文本侧 ReadFile 失败契约是返回空 FilePreview（markdown
//     空、size 0），视为「内容不可用」照常 diff 并标 contentMissing；
//   - docx/xlsx 侧：一侧 size 0（空文件/读取失败回空壳）→ 该侧按空内容计、
//     照常 diff 并标 contentMissing（与文本侧同口径）；但结构不可信——Preview
//     抛错 / kind 不符 / docx 无 dataUrl / xlsx body 非 JSON / 解包解析失败——
//     一律整体降级 unsupported（此时硬做 diff 会产出整篇误报，宁漏勿误；
//     UI 的 unsupported 分支仍保留并排预览入口兜底）。
//
// 纯函数（buildTextDiff/buildDocxDiff/buildXlsxDiff/diffStatOf/isTextComparable/
// clampDiffRows）可单测；async 包装只做取数与编排。

import { app } from "./bridge";
import { diffLines, type DiffRow } from "./diff";
import { extractDocxParagraphs } from "./docxText";
import { diffDocxParagraphs, docxDiffStat, type DocxRow } from "./docxTextDiff";
import { diffXlsxSheets, MAX_XLSX_SHEET_CELLS, type XlsxSheetDiff } from "./xlsxCellDiff";
import type { XlsxPreview, XlsxSheet } from "./types";

export interface VersionTextDiff {
  kind: "text";
  rows: DiffRow[];
  add: number;
  del: number;
  /** 基线或当前任一侧内容不可用（读取失败/空文件）时为 true，UI 需提示。 */
  contentMissing: boolean;
}

/** docx 段级结构化对比结果：段落序列的增删改（改 = 相邻 del+add 对）。 */
export interface VersionDocxDiff {
  kind: "docx";
  rows: DocxRow[];
  add: number;
  del: number;
  contentMissing: boolean;
}

/** xlsx 结构化对比结果：sheet 级 + 单元格级差异（只含有差异的 sheet）。 */
export interface VersionXlsxDiff {
  kind: "xlsx";
  sheets: XlsxSheetDiff[];
  /** 汇总新增量 = Σ(新增单元格 + 修改单元格 + 新增 sheet)；「修改」按 +1 计，与行级 diff 的 −1/+1 语义对齐，供 stat 徽标直接消费。 */
  add: number;
  /** 汇总删除量 = Σ(删除单元格 + 修改单元格 + 删除 sheet)。 */
  del: number;
  /** 单元格级「值/公式修改」合计（不计入 add/del，展示层细分用）。 */
  change: number;
  contentMissing: boolean;
}

export type VersionCompareResult =
  | VersionTextDiff
  | VersionDocxDiff
  | VersionXlsxDiff
  | { kind: "unsupported"; ext: string };

const TEXT_DIFF_EXTS = new Set([
  ".md", ".markdown", ".txt", ".csv", ".json", ".toml", ".yaml", ".yml",
  ".ini", ".xml", ".html", ".htm", ".css", ".js", ".mjs", ".cjs", ".ts",
  ".tsx", ".jsx", ".py", ".sh", ".sql", ".log",
]);

export function extOfPath(path: string): string {
  const m = /\.[^.\\/]+$/.exec(path);
  return m ? m[0].toLowerCase() : "";
}

export function isTextComparable(path: string): boolean {
  return TEXT_DIFF_EXTS.has(extOfPath(path));
}

/** 纯函数：两版全文 → 行级 diff + 增删计数。 */
export function buildTextDiff(baseText: string, currentText: string, contentMissing = false): VersionTextDiff {
  const rows = diffLines(baseText, currentText);
  let add = 0;
  let del = 0;
  for (const r of rows) {
    if (r.type === "add") add++;
    else if (r.type === "del") del++;
  }
  return { kind: "text", rows, add, del, contentMissing };
}

/** 纯函数：两侧段落序列 → docx 段级 diff 结果（含计数）。 */
export function buildDocxDiff(baseParas: string[], curParas: string[], contentMissing = false): VersionDocxDiff {
  const rows = diffDocxParagraphs(baseParas, curParas);
  const { add, del } = docxDiffStat(rows);
  return { kind: "docx", rows, add, del, contentMissing };
}

/** 纯函数：两侧 sheet 结构 → xlsx 差异结果（含汇总计数；截断口径在 xlsxCellDiff）。 */
export function buildXlsxDiff(baseSheets: XlsxSheet[], curSheets: XlsxSheet[], contentMissing = false): VersionXlsxDiff {
  const sheets = diffXlsxSheets(baseSheets, curSheets, MAX_XLSX_SHEET_CELLS);
  let add = 0;
  let del = 0;
  let change = 0;
  for (const s of sheets) {
    add += s.add + s.change + (s.state === "add" ? 1 : 0);
    del += s.del + s.change + (s.state === "del" ? 1 : 0);
    change += s.change;
  }
  return { kind: "xlsx", sheets, add, del, change, contentMissing };
}

export function diffStatOf(rows: DiffRow[]): { add: number; del: number } {
  let add = 0;
  let del = 0;
  for (const r of rows) {
    if (r.type === "add") add++;
    else if (r.type === "del") del++;
  }
  return { add, del };
}

/** clampDiffRows 的返回：shown 为可安全渲染的行，truncated 标记是否被折叠。 */
export interface ClampedDiff<T = DiffRow> {
  shown: T[];
  total: number;
  truncated: boolean;
}

/** UI 纯函数：长 diff 折叠——超过 max 行只保留前 max 行（LCS 全量行可能上万，UI 有界渲染）。
 *  泛型化以同时服务文本行（DiffRow）与 docx 段级行（DocxRow，含序号）；对
 *  DiffRow[] 的既有调用行为逐字节不变。 */
export function clampDiffRows<T extends DiffRow>(rows: T[], max: number): ClampedDiff<T> {
  if (rows.length <= max) return { shown: rows, total: rows.length, truncated: false };
  return { shown: rows.slice(0, max), total: rows.length, truncated: true };
}

/** 文本类：基线快照 vs 当前文件（ReadFile 失败契约 = 空 FilePreview，见头部注释）。 */
async function compareTextWithCurrent(baselinePath: string, currentPath: string): Promise<VersionTextDiff> {
  const [base, cur] = await Promise.all([app.ReadFile(baselinePath), app.ReadFile(currentPath)]);
  const baseText = base?.markdown ?? "";
  const curText = cur?.markdown ?? "";
  const missing = (base?.size ?? 0) === 0 || (cur?.size ?? 0) === 0;
  return buildTextDiff(baseText, curText, missing);
}

// docx 取数：一侧不可结构化解析 → null（上层整体降级 unsupported）；一侧内容
// 缺失（size 0）→ 空段落 + size 0（上层标 contentMissing，照常 diff）。
async function loadDocxSide(rel: string): Promise<{ paras: string[]; size: number } | null> {
  try {
    const r = await app.Preview(rel);
    if (!r || r.kind !== "docx") return null;
    if ((r.size ?? 0) === 0) return { paras: [], size: 0 };
    if (!r.dataUrl) return null; // 有体量却拿不到原始包，结构不可信
    return { paras: await extractDocxParagraphs(r.dataUrl), size: r.size };
  } catch {
    return null; // 读取/解包/解析失败：宁漏勿误，交由上层降级（不抛错）
  }
}

async function compareDocxWithCurrent(baselinePath: string, currentPath: string): Promise<VersionCompareResult> {
  const [base, cur] = await Promise.all([loadDocxSide(baselinePath), loadDocxSide(currentPath)]);
  if (!base || !cur) return { kind: "unsupported", ext: ".docx" };
  return buildDocxDiff(base.paras, cur.paras, base.size === 0 || cur.size === 0);
}

// xlsx 取数：body 为结构化单元格 JSON（GaeaPreview xlsx 分支契约），解析失败
// 视为结构不可信 → null（上层降级 unsupported）；口径同 loadDocxSide。
async function loadXlsxSide(rel: string): Promise<{ sheets: XlsxSheet[]; size: number } | null> {
  try {
    const r = await app.Preview(rel);
    if (!r || r.kind !== "xlsx") return null;
    if ((r.size ?? 0) === 0) return { sheets: [], size: 0 };
    if (!r.body) return null;
    const parsed = JSON.parse(r.body) as XlsxPreview;
    if (!parsed || !Array.isArray(parsed.sheets)) return null;
    return { sheets: parsed.sheets, size: r.size };
  } catch {
    return null; // 读取/JSON 解析失败：宁漏勿误（不抛错）
  }
}

async function compareXlsxWithCurrent(baselinePath: string, currentPath: string): Promise<VersionCompareResult> {
  const [base, cur] = await Promise.all([loadXlsxSide(baselinePath), loadXlsxSide(currentPath)]);
  if (!base || !cur) return { kind: "unsupported", ext: ".xlsx" };
  return buildXlsxDiff(base.sheets, cur.sheets, base.size === 0 || cur.size === 0);
}

/** 取数编排：基线快照 vs 当前文件，按扩展名分派 text/docx/xlsx，其余 unsupported
 *  （分派与降级口径见文件头注释）。 */
export async function compareVersionWithCurrent(baselinePath: string, currentPath: string): Promise<VersionCompareResult> {
  const ext = extOfPath(currentPath);
  if (isTextComparable(currentPath)) return compareTextWithCurrent(baselinePath, currentPath);
  if (ext === ".docx") return compareDocxWithCurrent(baselinePath, currentPath);
  if (ext === ".xlsx") return compareXlsxWithCurrent(baselinePath, currentPath);
  return { kind: "unsupported", ext };
}
