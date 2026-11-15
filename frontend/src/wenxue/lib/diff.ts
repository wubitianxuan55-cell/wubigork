// marker 可选列（v4.87 统一 diff 查看器）：docx 段落序号 / xlsx 单元格 ref 等
// 「行定位」标注。ChangesDiff 在 +/- 符号列后渲染定宽右对齐列；缺省不渲染，
// 既有数据源（ChangesPanel LCS / GitPanel unified diff）零变化。
export type DiffRow = { type: "ctx" | "add" | "del"; text: string; marker?: string };

// lcsDiff 的输出操作流：ctx=两侧共有（ai/bi 均有效），del=仅 a 侧（bi=-1），
// add=仅 b 侧（ai=-1）。调用方按各自元素语义取值（行文本/段序号/页锚点/字符）。
export interface LcsOp {
  type: "ctx" | "add" | "del";
  /** a 侧下标（ctx/del 有效；add 时为 -1）。 */
  ai: number;
  /** b 侧下标（ctx/add 有效；del 时为 -1）。 */
  bi: number;
}

// lcsDiff 是经典全量矩阵 LCS DP 的唯一实现（FE4-01 收敛点）：此前行级
//（diffLines）、docx 段级（docxTextDiff）、pptx 页签名对齐（pptxTextDiff）、
// 字符级行内高亮（diffRender.charSegments）四份同款 DP 各写一遍，现收成这一份
// 泛型，调用方只保留各自的元素类型与取值适配。回溯取向统一为「先耗 a 侧」
//（dp[i+1][j] >= dp[i][j+1] 时先出 del）——与既有四份实现逐位一致，同分时的
// 输出顺序是调用方（改蓝配对、锚点配对）可见行为，不可更动。
export function lcsDiff<T>(a: T[], b: T[], eq: (x: T, y: T) => boolean): LcsOp[] {
  const n = a.length;
  const m = b.length;
  // dp[i][j] = a[i..] 与 b[j..] 的 LCS 长度（全量矩阵，回溯每一步都要用
  // dp[i+1][j]/dp[i][j+1]，故不能滚动数组化）。
  const dp: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      dp[i][j] = eq(a[i]!, b[j]!) ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1]);
    }
  }
  const ops: LcsOp[] = [];
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (eq(a[i]!, b[j]!)) {
      ops.push({ type: "ctx", ai: i, bi: j });
      i++;
      j++;
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      ops.push({ type: "del", ai: i, bi: -1 });
      i++;
    } else {
      ops.push({ type: "add", ai: -1, bi: j });
      j++;
    }
  }
  while (i < n) {
    ops.push({ type: "del", ai: i, bi: -1 });
    i++;
  }
  while (j < m) {
    ops.push({ type: "add", ai: -1, bi: j });
    j++;
  }
  return ops;
}

// diffLines is a classic LCS line diff. Used by the diff seam to render edit-tool
// before/after; a real editor (Monaco/CodeMirror merge) would replace the
// rendering, but this keeps the algorithm in one place (the DP itself now lives
// in lcsDiff, shared with docx/pptx/char-level diffs).
export function diffLines(a: string, b: string): DiffRow[] {
  const x = a.split("\n");
  const y = b.split("\n");
  return lcsDiff(x, y, (p, q) => p === q).map((op) =>
    op.type === "add"
      ? { type: "add" as const, text: y[op.bi]! }
      : { type: op.type, text: x[op.ai]! },
  );
}
