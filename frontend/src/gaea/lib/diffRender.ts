// diffRender — diff 渲染层纯函数（蒸馏规划 2c 统一 diff 渲染升级）。
//
// 三项升级，全部作用于「展示模型」而非 DiffRow 本身（lib/diff.ts 的
// DiffRow 语义不变，GitPanel/ChangesPanel 两个数据源零改动）：
//  1. 改蓝配对：相邻的 删除块+新增块 按行两两配对，标为「改动」对（蓝色
//     底，替代整段红/绿），未配对完的余量保持红/绿；
//  2. 行内字符高亮：配对行内部做字符级 LCS，未变字符正常显示、变化片段
//     加删除/新增强调；
//  3. 上下文折叠：连续超过阈值（keep*2）的上下文行收起中段，可展开。
// 语法着色不在本刀（依赖高亮器，属阶段三 CodeMirror/3a 范围）。
import type { DiffRow } from "./diff";

// ── 改蓝配对 ────────────────────────────────────────────────────

/** 配对后的展示行：row 携带配对方文本时渲染为改蓝对（old/new 两行成对）。 */
export type DiffPresentRow =
  | { kind: "row"; row: DiffRow; pairOld?: string; pairNew?: string }
  | { kind: "fold"; count: number; rows: DiffRow[] }; // rows=被收起的上下文行（展开时渲染）

// 把一段连续的非上下文行规范化为「先删后增」两段（LCS 输出可能交错，
// 配对前先归位；稳定重排不改行集合）。
function splitRun(run: DiffRow[]): { dels: DiffRow[]; adds: DiffRow[] } {
  const dels = run.filter((r) => r.type === "del");
  const adds = run.filter((r) => r.type === "add");
  return { dels, adds };
}

/**
 * 改蓝配对：扫描行序列，每个「连续删块+连续增块」区域内按序两两配对
 * （min(删数, 增数) 对），配对成功的行为 pairOld/pairNew 成对行；配不上的
 * 余量原样保留。纯函数，不修改输入。
 */
export function pairModifications(rows: DiffRow[]): DiffPresentRow[] {
  const out: DiffPresentRow[] = [];
  let i = 0;
  while (i < rows.length) {
    if (rows[i]!.type === "ctx") {
      out.push({ kind: "row", row: rows[i]! });
      i += 1;
      continue;
    }
    // 收集连续非 ctx 的 run
    let j = i;
    while (j < rows.length && rows[j]!.type !== "ctx") j += 1;
    const run = rows.slice(i, j);
    const { dels, adds } = splitRun(run);
    const pairN = Math.min(dels.length, adds.length);
    for (let k = 0; k < pairN; k += 1) {
      out.push({ kind: "row", row: { type: "del", text: dels[k]!.text }, pairOld: dels[k]!.text, pairNew: adds[k]!.text });
      out.push({ kind: "row", row: { type: "add", text: adds[k]!.text }, pairOld: dels[k]!.text, pairNew: adds[k]!.text });
    }
    for (let k = pairN; k < dels.length; k += 1) out.push({ kind: "row", row: dels[k]! });
    for (let k = pairN; k < adds.length; k += 1) out.push({ kind: "row", row: adds[k]! });
    i = j;
  }
  return out;
}

// ── 行内字符高亮 ────────────────────────────────────────────────

export interface CharSeg {
  text: string;
  /** true=该片段在配对行中发生了变化（old 侧被删 / new 侧被增）。 */
  changed: boolean;
}

// 字符级行内差异上限：超长行整行视为变化（LCS O(n*m) 防撑爆）。
const MAX_CHAR_DIFF = 240;

/**
 * 配对行的字符级分段：old/new 两侧各自输出 [未变,变化,未变…] 片段。
 * 任一侧超长或全等时退化为单段（changed=两侧不同）。
 */
export function charSegments(a: string, b: string): { oldSegs: CharSeg[]; newSegs: CharSeg[] } {
  if (a === b) {
    return {
      oldSegs: [{ text: a, changed: false }],
      newSegs: [{ text: b, changed: false }],
    };
  }
  if (a.length > MAX_CHAR_DIFF || b.length > MAX_CHAR_DIFF) {
    return {
      oldSegs: [{ text: a, changed: true }],
      newSegs: [{ text: b, changed: true }],
    };
  }
  // 字符级 LCS（行短，O(n*m) 可接受）
  const x = Array.from(a);
  const y = Array.from(b);
  const n = x.length;
  const m = y.length;
  const dp: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));
  for (let i = n - 1; i >= 0; i -= 1) {
    for (let j = m - 1; j >= 0; j -= 1) {
      dp[i][j] = x[i] === y[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1]);
    }
  }
  type Seg = { text: string; changed: boolean };
  const oldSegs: Seg[] = [];
  const newSegs: Seg[] = [];
  const push = (segs: Seg[], changed: boolean, ch: string) => {
    const last = segs[segs.length - 1];
    if (last && last.changed === changed) last.text += ch;
    else segs.push({ text: ch, changed });
  };
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (x[i] === y[j]) {
      push(oldSegs, false, x[i]!);
      push(newSegs, false, y[j]!);
      i += 1;
      j += 1;
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      push(oldSegs, true, x[i]!);
      i += 1;
    } else {
      push(newSegs, true, y[j]!);
      j += 1;
    }
  }
  while (i < n) {
    push(oldSegs, true, x[i]!);
    i += 1;
  }
  while (j < m) {
    push(newSegs, true, y[j]!);
    j += 1;
  }
  return { oldSegs, newSegs };
}

// ── 上下文折叠 ──────────────────────────────────────────────────

/** 折叠阈值：上下文连续 run 超过 keep*2 行时收起中段，两端各留 keep 行。 */
export const FOLD_KEEP = 3;

/**
 * 对配对后的展示行做上下文折叠：连续 ctx 行 run 长度 > keep*2 时，
 * 首尾各留 keep 行、中段收成一条 fold 行（count=收起行数）。
 */
export function foldContext(rows: DiffPresentRow[], keep: number = FOLD_KEEP): DiffPresentRow[] {
  const out: DiffPresentRow[] = [];
  let i = 0;
  while (i < rows.length) {
    const r = rows[i]!;
    if (r.kind !== "row" || r.row.type !== "ctx") {
      out.push(r);
      i += 1;
      continue;
    }
    let j = i;
    while (
      j < rows.length &&
      rows[j]!.kind === "row" &&
      (rows[j] as Extract<DiffPresentRow, { kind: "row" }>).row.type === "ctx"
    )
      j += 1;
    const run = rows.slice(i, j);
    if (run.length <= keep * 2) {
      out.push(...run);
    } else {
      out.push(...run.slice(0, keep));
      const middle = run.slice(keep, run.length - keep);
      out.push({ kind: "fold", count: middle.length, rows: middle.map((p) => (p as Extract<DiffPresentRow, { kind: "row" }>).row) });
      out.push(...run.slice(run.length - keep));
    }
    i = j;
  }
  return out;
}
