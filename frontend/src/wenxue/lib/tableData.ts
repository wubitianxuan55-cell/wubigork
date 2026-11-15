// 粘贴表格即数据：从输入文本中识别 CSV/TSV 样式的表格块，并转成 Markdown
// 表格，让粘贴的表格成为结构化上下文（对标豆包/WorkBuddy 的"粘贴即数据"）。
// 纯逻辑模块，不依赖 React。

const DELIMS = ["\t", ",", "，", ";", "；"] as const;

export interface TableBlock {
  startLine: number;
  endLine: number;
  rows: number;
  cols: number;
  delim: string;
  lines: string[];
}

function splitCells(line: string, delim: string): string[] {
  return line.split(delim).map((c) => c.trim());
}

// 识别文本中第一段"多行、同分隔符、列数一致且 ≥2 列"的表格块。
// 返回 null 表示未识别到（单行、散文性文本不会误报）。
export function detectTableBlock(text: string): TableBlock | null {
  const lines = text.split("\n");
  for (let start = 0; start < lines.length; start++) {
    const first = lines[start].trim();
    if (!first) continue;
    for (const delim of DELIMS) {
      const firstCells = splitCells(first, delim);
      if (firstCells.length < 2) continue;
      const cols = firstCells.length;
      let end = start;
      while (end + 1 < lines.length) {
        const next = lines[end + 1].trim();
        if (!next) break;
        const cells = splitCells(next, delim);
        if (cells.length !== cols) break;
        end++;
      }
      if (end - start + 1 >= 2) {
        return {
          startLine: start,
          endLine: end,
          rows: end - start + 1,
          cols,
          delim,
          lines: lines.slice(start, end + 1),
        };
      }
    }
  }
  return null;
}

// 表格块 → GFM Markdown 表格（转义 |、统一按分隔符切分）。
export function tableBlockToMarkdown(block: TableBlock): string {
  const esc = (s: string) => s.replace(/\|/g, "\\|");
  const rows = block.lines.map((l) => splitCells(l, block.delim).map(esc));
  const sep = rows[0].map(() => "---");
  const fmt = (r: string[]) => `| ${r.join(" | ")} |`;
  return [fmt(rows[0]), fmt(sep), ...rows.slice(1).map(fmt)].join("\n");
}

// 整段文本应用表格转换：识别到的表格块替换为 Markdown 表格，其余不变。
export function applyTableConversion(text: string, enabled: boolean): string {
  if (!enabled) return text;
  const block = detectTableBlock(text);
  if (!block) return text;
  const lines = text.split("\n");
  const converted = tableBlockToMarkdown(block);
  return [...lines.slice(0, block.startLine), converted, ...lines.slice(block.endLine + 1)].join("\n");
}
