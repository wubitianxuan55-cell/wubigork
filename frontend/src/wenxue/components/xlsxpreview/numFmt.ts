/**
 * xlsxpreview/numFmt — 自 XlsxPreview 原样搬移的纯工具族（行为逐字节不变）：
 * 单元格引用解析 / 列字母、NumFmt 数字格式近似显示、AI 编辑预设常量。
 * 仅供 XlsxPreview 及其子面板（SheetGrid/FormulaBar）内部使用，不构成组件导出面。
 */
import type { XlsxCell } from "../../lib/types";

export function parseRef(ref: string): { col: number; row: number } {
  const m = /^([A-Z]+)(\d+)$/.exec(ref);
  if (!m) return { col: 0, row: 0 };
  let col = 0;
  for (const ch of m[1]) col = col * 26 + (ch.charCodeAt(0) - 64);
  return { col, row: parseInt(m[2], 10) };
}

export function colToLetter(n: number): string {
  let s = "";
  while (n > 0) {
    const rem = (n - 1) % 26;
    s = String.fromCharCode(65 + rem) + s;
    n = Math.floor((n - 1) / 26);
  }
  return s;
}

// —— 数字格式显示（NumFmt 近似实现）——
// 内置编号 → 格式串（只覆盖常用编号，其余原样显示）
const BUILTIN_NUM_FMTS: Record<string, string> = {
  "1": "0",
  "2": "0.00",
  "3": "#,##0",
  "4": "#,##0.00",
  "5": "$#,##0",
  "6": "$#,##0",
  "7": "$#,##0.00",
  "8": "$#,##0.00",
  "9": "0%",
  "10": "0.00%",
  "11": "0.00E+00",
  "44": "¥#,##0",
  "45": "¥#,##0",
  "46": "¥#,##0.00",
  "47": "¥#,##0.00",
};

const DATE_FMT_IDS = new Set(["14", "15", "16", "17", "18", "19", "20", "21", "22", "27", "28", "29", "30", "31", "32", "33", "34", "35", "36", "50", "51", "52", "53", "54", "55", "56", "57", "58"]);

function applyNumFmt(n: number, fmt: string): string {
  let pattern = fmt.trim();
  if (/^\d+$/.test(pattern) && DATE_FMT_IDS.has(pattern)) return String(n);
  if (/^\d+$/.test(pattern) && BUILTIN_NUM_FMTS[pattern]) pattern = BUILTIN_NUM_FMTS[pattern];
  if (!pattern || /general/i.test(pattern)) return String(n);
  // 只取分号前第一段，去掉 [DBNum*]/[$-xxx]/[Red] 等标记
  pattern = (pattern.split(";")[0] ?? pattern).replace(/\[[^\]]*\]/g, "").trim();
  if (!pattern) return String(n);
  const neg = n < 0;
  const abs = Math.abs(n);
  const isPct = pattern.includes("%");
  const v = isPct ? abs * 100 : abs;
  const m = /\.(0+)/.exec(pattern);
  const decimals = m ? m[1].length : pattern.includes(".") ? 2 : 0;
  const hasThousands = pattern.includes(",");
  let s = v.toFixed(decimals);
  if (hasThousands) {
    const [ip, fp] = s.split(".");
    s = ip.replace(/\B(?=(\d{3})+(?!\d))/g, ",") + (fp !== undefined ? "." + fp : "");
  }
  if (isPct) s += "%";
  if (pattern.includes("$")) s = "$" + s;
  if (pattern.includes("¥") || pattern.includes("￥")) s = "¥" + s;
  if (pattern.includes("€")) s = "€" + s;
  return (neg ? "-" : "") + s;
}

export function formatCellValue(cell: XlsxCell | undefined): string {
  const raw = cell?.value ?? "";
  if (raw === "") return "";
  if (cell?.type === "error") return raw;
  const fmt = cell?.style?.numFmt ?? "";
  if (!fmt || /general/i.test(fmt)) return raw;
  const n = Number(raw);
  if (!Number.isFinite(n)) return raw;
  return applyNumFmt(n, fmt);
}

/**
 * XlsxPreview 渲染后端提取的结构化单元格视图：sheet 切换、公式标识、
 * 样式近似还原、合并单元格与列宽。为 P0-③ 单元格编辑提供承载。
 */
export const EDIT_PRESETS = [
  { label: "求和", instruction: "对相关数据求和，把公式写入选中单元格" },
  { label: "平均值", instruction: "计算相关数据的平均值，把公式写入选中单元格" },
  { label: "拆分列", instruction: "把选中单元格所在列按分隔符拆分为多列，新列写入相邻列并加表头" },
  { label: "清洗", instruction: "清洗相关数据：去掉首尾空格、统一大小写" },
  { label: "加粗", instruction: "把选中单元格加粗（保留其他样式）" },
  { label: "高亮", instruction: "把选中单元格填充浅黄色 FFF2CC" },
  { label: "合并表头", instruction: "把选中区域合并为一个居中表头单元格" },
];