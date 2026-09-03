// versionCompare — 产物版本间对比数据层（A1 交付验收闭环，调研 A#4/O#2）。
//
// 数据源：证据链基线快照（JournalChangeRecord.baselinePath，工作区相对路径，
// 与预览/回滚同源）与当前文件本体。文本类文件经 ReadFile 读全文 → diffLines
// 行级差异；非文本（xlsx/docx/pdf/图片）无法以文本 diff 呈现 → kind:
// "unsupported"，UI 降级为「双版本并排预览」入口（A2 再做结构化对比）。
//
// 纯函数（buildTextDiff/diffStatOf/isTextComparable）可单测；async 包装只做
// 取数与编排。ReadFile 失败契约：返回空 FilePreview（markdown 空、size 0），
// 视为「内容不可用」，不抛错（与缺失态宁漏勿误同口径）。

import { app } from "./bridge";
import { diffLines, type DiffRow } from "./diff";

export interface VersionTextDiff {
  kind: "text";
  rows: DiffRow[];
  add: number;
  del: number;
  /** 基线或当前任一侧内容不可用（读取失败/空文件）时为 true，UI 需提示。 */
  contentMissing: boolean;
}

export type VersionCompareResult =
  | VersionTextDiff
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
export interface ClampedDiff {
  shown: DiffRow[];
  total: number;
  truncated: boolean;
}

/** UI 纯函数：长 diff 折叠——超过 max 行只保留前 max 行（LCS 全量行可能上万，UI 有界渲染）。 */
export function clampDiffRows(rows: DiffRow[], max: number): ClampedDiff {
  if (rows.length <= max) return { shown: rows, total: rows.length, truncated: false };
  return { shown: rows.slice(0, max), total: rows.length, truncated: true };
}

/** 取数编排：基线快照 vs 当前文件。非文本类型直接返回 unsupported。 */
export async function compareVersionWithCurrent(baselinePath: string, currentPath: string): Promise<VersionCompareResult> {
  if (!isTextComparable(currentPath)) return { kind: "unsupported", ext: extOfPath(currentPath) };
  const [base, cur] = await Promise.all([app.ReadFile(baselinePath), app.ReadFile(currentPath)]);
  const baseText = base?.markdown ?? "";
  const curText = cur?.markdown ?? "";
  const missing = (base?.size ?? 0) === 0 || (cur?.size ?? 0) === 0;
  return buildTextDiff(baseText, curText, missing);
}
