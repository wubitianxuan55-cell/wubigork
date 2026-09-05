// marker 可选列（v4.87 统一 diff 查看器）：docx 段落序号 / xlsx 单元格 ref 等
// 「行定位」标注。ChangesDiff 在 +/- 符号列后渲染定宽右对齐列；缺省不渲染，
// 既有数据源（ChangesPanel LCS / GitPanel unified diff）零变化。
export type DiffRow = { type: "ctx" | "add" | "del"; text: string; marker?: string };

// diffLines is a classic LCS line diff. Used by the diff seam to render edit-tool
// before/after; a real editor (Monaco/CodeMirror merge) would replace the
// rendering, but this keeps the algorithm in one place.
export function diffLines(a: string, b: string): DiffRow[] {
  const x = a.split("\n");
  const y = b.split("\n");
  const n = x.length;
  const m = y.length;
  const dp: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      dp[i][j] = x[i] === y[j] ? dp[i + 1][j + 1] + 1 : Math.max(dp[i + 1][j], dp[i][j + 1]);
    }
  }
  const rows: DiffRow[] = [];
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (x[i] === y[j]) {
      rows.push({ type: "ctx", text: x[i] });
      i++;
      j++;
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      rows.push({ type: "del", text: x[i] });
      i++;
    } else {
      rows.push({ type: "add", text: y[j] });
      j++;
    }
  }
  while (i < n) rows.push({ type: "del", text: x[i++] });
  while (j < m) rows.push({ type: "add", text: y[j++] });
  return rows;
}
